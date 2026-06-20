package sqlitestore

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// ErrMigration indicates a schema migration failed or the on-disk schema version
// is ahead of what this binary understands. Maps to ERR_STORE_MIGRATION
// (PROJECT-BOOK §8.4).
var ErrMigration = errors.New("store migration failed")

type migration struct {
	version int
	name    string
	sql     string
}

// loadMigrations reads and orders the embedded migrations by version.
func loadMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return nil, fmt.Errorf("reading migrations: %w", err)
	}

	migrations := make([]migration, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".sql") {
			continue
		}
		prefix, _, ok := strings.Cut(name, "_")
		if !ok {
			return nil, fmt.Errorf("%w: migration %q lacks a version prefix", ErrMigration, name)
		}
		version, convErr := strconv.Atoi(prefix)
		if convErr != nil {
			return nil, fmt.Errorf("%w: migration %q has a non-numeric version", ErrMigration, name)
		}
		body, readErr := fs.ReadFile(migrationFiles, "migrations/"+name)
		if readErr != nil {
			return nil, fmt.Errorf("reading migration %q: %w", name, readErr)
		}
		migrations = append(migrations, migration{version: version, name: name, sql: string(body)})
	}

	sort.Slice(migrations, func(i, j int) bool { return migrations[i].version < migrations[j].version })
	for i, m := range migrations {
		if m.version != i+1 {
			return nil, fmt.Errorf("%w: migrations are not contiguous (expected %d, got %d)", ErrMigration, i+1, m.version)
		}
	}
	return migrations, nil
}

// migrate applies every migration not yet recorded, in a transaction per
// migration, and returns the resulting schema version. A database whose version
// exceeds the latest known migration is rejected rather than silently used.
func migrate(ctx context.Context, db *sql.DB, now func() time.Time) (int, error) {
	migrations, err := loadMigrations()
	if err != nil {
		return 0, err
	}

	if _, err := db.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS schema_migrations (
			version    INTEGER PRIMARY KEY,
			applied_at INTEGER NOT NULL
		)`); err != nil {
		return 0, fmt.Errorf("creating schema_migrations: %w", errors.Join(ErrMigration, err))
	}

	current, err := schemaVersion(ctx, db)
	if err != nil {
		return 0, err
	}

	latest := len(migrations)
	if current > latest {
		return 0, fmt.Errorf("%w: database schema version %d is newer than this build supports (%d)", ErrMigration, current, latest)
	}

	for _, m := range migrations {
		if m.version <= current {
			continue
		}
		if err := applyMigration(ctx, db, m, now); err != nil {
			return 0, err
		}
	}
	return latest, nil
}

func applyMigration(ctx context.Context, db *sql.DB, m migration, now func() time.Time) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %q: %w", m.name, errors.Join(ErrMigration, err))
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, m.sql); err != nil {
		return fmt.Errorf("applying migration %q: %w", m.name, errors.Join(ErrMigration, err))
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`,
		m.version, now().UTC().UnixNano()); err != nil {
		return fmt.Errorf("recording migration %q: %w", m.name, errors.Join(ErrMigration, err))
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing migration %q: %w", m.name, errors.Join(ErrMigration, err))
	}
	return nil
}

// schemaVersion returns the highest applied migration version, or 0 if none.
func schemaVersion(ctx context.Context, db *sql.DB) (int, error) {
	var version sql.NullInt64
	err := db.QueryRowContext(ctx, `SELECT MAX(version) FROM schema_migrations`).Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("reading schema version: %w", errors.Join(ErrMigration, err))
	}
	if !version.Valid {
		return 0, nil
	}
	return int(version.Int64), nil
}
