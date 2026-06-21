package sqlitestore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/drydock/drydock/internal/core/domain"
)

// AppendAuditEntry persists a fully-formed audit entry. It implements
// audit.Store. The audit_log table has no UPDATE/DELETE path — entries are
// immutable once written.
func (s *Store) AppendAuditEntry(ctx context.Context, e domain.AuditEntry) error {
	detail, err := json.Marshal(e.Detail)
	if err != nil {
		return fmt.Errorf("encoding audit detail: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO audit_log (seq, at, action, host_id, subject, detail, prev_mac, mac)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		e.Seq, e.At.UTC().UnixNano(), string(e.Action), nullable(e.HostRef),
		e.Subject, string(detail), e.PrevMAC, e.MAC)
	if err != nil {
		return fmt.Errorf("inserting audit entry: %w", err)
	}
	return nil
}

// LastAuditEntry returns the most recent entry, or false when the log is empty.
// It implements audit.Store.
func (s *Store) LastAuditEntry(ctx context.Context) (domain.AuditEntry, bool, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT seq, at, action, host_id, subject, detail, prev_mac, mac
		 FROM audit_log ORDER BY seq DESC LIMIT 1`)
	entry, err := scanAuditEntry(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.AuditEntry{}, false, nil
	}
	if err != nil {
		return domain.AuditEntry{}, false, fmt.Errorf("reading last audit entry: %w", err)
	}
	return entry, true, nil
}

// AuditEntries returns every entry ordered by ascending sequence. It implements
// audit.Store.
func (s *Store) AuditEntries(ctx context.Context) ([]domain.AuditEntry, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT seq, at, action, host_id, subject, detail, prev_mac, mac
		 FROM audit_log ORDER BY seq ASC`)
	if err != nil {
		return nil, fmt.Errorf("querying audit entries: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var entries []domain.AuditEntry
	for rows.Next() {
		entry, scanErr := scanAuditEntry(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scanning audit entry: %w", scanErr)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating audit entries: %w", err)
	}
	return entries, nil
}

// scanner abstracts *sql.Row and *sql.Rows for the shared scan helper.
type scanner interface {
	Scan(dest ...any) error
}

func scanAuditEntry(row scanner) (domain.AuditEntry, error) {
	var (
		e          domain.AuditEntry
		atNanos    int64
		action     string
		hostID     sql.NullString
		detailJSON string
	)
	if err := row.Scan(&e.Seq, &atNanos, &action, &hostID, &e.Subject, &detailJSON, &e.PrevMAC, &e.MAC); err != nil {
		return domain.AuditEntry{}, err
	}
	e.At = time.Unix(0, atNanos).UTC()
	e.Action = domain.Action(action)
	e.HostRef = hostID.String
	if err := json.Unmarshal([]byte(detailJSON), &e.Detail); err != nil {
		return domain.AuditEntry{}, fmt.Errorf("decoding audit detail: %w", err)
	}
	return e, nil
}

// nullable maps an empty string to SQL NULL so optional columns are not stored
// as empty strings.
func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
