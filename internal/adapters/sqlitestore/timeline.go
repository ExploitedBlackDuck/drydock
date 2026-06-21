package sqlitestore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/drydock/drydock/internal/core/domain"
)

// SaveTimelineEntry appends one mapped engine event to the host timeline
// (ADR-0018). It writes only to timeline_entries — never to audit_log, which
// stays the separate, hash-chained record of Drydock-authored actions.
func (s *Store) SaveTimelineEntry(ctx context.Context, e domain.TimelineEntry) error {
	detail, err := json.Marshal(e.Detail)
	if err != nil {
		return fmt.Errorf("encoding timeline detail: %w", err)
	}
	var exitCode any
	if e.ExitCode != nil {
		exitCode = *e.ExitCode
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO timeline_entries (host_id, at, source, kind, subject, exit_code, detail)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		e.HostRef, e.At.UTC().UnixNano(), string(e.Source), e.Kind, e.Subject, exitCode, string(detail))
	if err != nil {
		return fmt.Errorf("saving timeline entry: %w", err)
	}
	return nil
}

// RecentTimelineEntries returns up to limit most-recent engine timeline entries
// for a host, oldest first (ready to interleave with audit references).
func (s *Store) RecentTimelineEntries(ctx context.Context, hostID string, limit int) ([]domain.TimelineEntry, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT host_id, at, source, kind, subject, exit_code, detail
		 FROM timeline_entries WHERE host_id = ? ORDER BY at DESC LIMIT ?`, hostID, limit)
	if err != nil {
		return nil, fmt.Errorf("querying timeline: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var entries []domain.TimelineEntry
	for rows.Next() {
		var (
			e          domain.TimelineEntry
			atNanos    int64
			source     string
			exitCode   sql.NullInt64
			detailJSON string
		)
		if err := rows.Scan(&e.HostRef, &atNanos, &source, &e.Kind, &e.Subject, &exitCode, &detailJSON); err != nil {
			return nil, fmt.Errorf("scanning timeline entry: %w", err)
		}
		e.At = time.Unix(0, atNanos).UTC()
		e.Source = domain.TimelineSource(source)
		if exitCode.Valid {
			n := int(exitCode.Int64)
			e.ExitCode = &n
		}
		if err := json.Unmarshal([]byte(detailJSON), &e.Detail); err != nil {
			return nil, fmt.Errorf("decoding timeline detail: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating timeline: %w", err)
	}
	// Reverse to oldest-first.
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	return entries, nil
}

// PruneTimelineEntries deletes timeline rows older than the cutoff (rolling
// retention, §7.7/§7.12.4).
func (s *Store) PruneTimelineEntries(ctx context.Context, before time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM timeline_entries WHERE at < ?`, before.UTC().UnixNano())
	if err != nil {
		return 0, fmt.Errorf("pruning timeline: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("counting pruned timeline rows: %w", err)
	}
	return n, nil
}
