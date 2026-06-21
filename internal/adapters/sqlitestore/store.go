// Package sqlitestore is the SQLite implementation of Drydock's persistence
// ports (ADR-0007), using the pure-Go modernc.org/sqlite driver so cross-
// compilation stays CGO-free. All SQL lives in this package; the core depends
// only on narrow, consumer-defined interfaces that *Store satisfies.
package sqlitestore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

// Store is a SQLite-backed implementation of the audit and secret-persistence
// ports. Construct it with Open.
type Store struct {
	db   *sql.DB
	now  func() time.Time
	path string
}

// Option configures a Store.
type Option func(*Store)

// WithClock overrides the time source (used by tests for determinism).
func WithClock(now func() time.Time) Option {
	return func(s *Store) { s.now = now }
}

// Open opens (creating if needed) the SQLite database at path, applies all
// pending migrations, and verifies the schema version. It returns an error
// wrapping ErrMigration if the on-disk schema is newer than this build.
func Open(ctx context.Context, path string, opts ...Option) (*Store, error) {
	s := &Store{now: time.Now}
	for _, opt := range opts {
		opt(s)
	}

	dsn := "file:" + path + "?" + url.Values{
		"_pragma": {"busy_timeout(5000)", "journal_mode(WAL)", "foreign_keys(1)"},
	}.Encode()

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database %q: %w", path, err)
	}
	// One writer connection avoids "database is locked" under WAL for a desktop
	// single-process workload; reads still share it.
	db.SetMaxOpenConns(1)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connecting to database %q: %w", path, err)
	}

	if _, err := migrate(ctx, db, s.now); err != nil {
		_ = db.Close()
		return nil, err
	}

	s.db = db
	s.path = path
	return s, nil
}

// SchemaVersion returns the current applied schema version.
func (s *Store) SchemaVersion(ctx context.Context) (int, error) {
	return schemaVersion(ctx, s.db)
}

// Backup writes a consistent single-file snapshot of the database to dest using
// SQLite's VACUUM INTO (ADR-0022). Unlike a raw file copy, this is safe under
// WAL with concurrent reads and writes — it snapshots a consistent point. dest
// must not already exist. It returns dest on success.
func (s *Store) Backup(ctx context.Context, dest string) (string, error) {
	if _, err := os.Stat(dest); err == nil {
		return "", fmt.Errorf("backup destination already exists: %s", dest)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("checking backup destination %q: %w", dest, err)
	}
	// dest is a bound parameter, never interpolated into the SQL text.
	if _, err := s.db.ExecContext(ctx, `VACUUM INTO ?`, dest); err != nil {
		return "", fmt.Errorf("backing up database to %q: %w", dest, err)
	}
	return dest, nil
}

// DefaultBackupPath returns a timestamped backup path beside the live database,
// so a caller without its own destination still gets a stable, discoverable one.
func (s *Store) DefaultBackupPath() string {
	return fmt.Sprintf("%s.backup-%s", s.path, s.now().UTC().Format("20060102T150405Z"))
}

// Close releases the database connection.
func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}
