package sqlitestore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/drydock/drydock/internal/core/domain"
)

// defaultOperationLimit caps an unfiltered history query so a long-lived install
// never streams its entire operations table into the UI by accident.
const defaultOperationLimit = 500

// SaveOperation persists a completed operation record. Records are immutable.
func (s *Store) SaveOperation(ctx context.Context, op domain.Operation) error {
	optionSet, err := json.Marshal(op.OptionSet)
	if err != nil {
		return fmt.Errorf("encoding operation option set: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO operations (id, host_id, kind, target, option_set, result,
		     bytes_reclaimed, started_at, ended_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		op.ID, op.HostRef, string(op.Kind), op.Target, string(optionSet), op.Result,
		op.BytesReclaimed, op.StartedAt.UTC().UnixNano(), op.EndedAt.UTC().UnixNano())
	if err != nil {
		return fmt.Errorf("saving operation %q: %w", op.ID, err)
	}
	return nil
}

// Operations returns recorded operations matching the query, most recent first
// (PROJECT-BOOK §7.11.8). All SQL composition lives here; the WHERE clause is
// built from bound parameters only — no value is ever interpolated into the
// statement (ADR-0004/§2.8).
func (s *Store) Operations(ctx context.Context, q domain.OperationQuery) ([]domain.Operation, error) {
	var (
		where []string
		args  []any
	)
	if q.HostRef != "" {
		where = append(where, "host_id = ?")
		args = append(args, q.HostRef)
	}
	if len(q.Kinds) > 0 {
		placeholders := make([]string, len(q.Kinds))
		for i, k := range q.Kinds {
			placeholders[i] = "?"
			args = append(args, string(k))
		}
		where = append(where, "kind IN ("+strings.Join(placeholders, ", ")+")")
	}
	if !q.Since.IsZero() {
		where = append(where, "started_at >= ?")
		args = append(args, q.Since.UTC().UnixNano())
	}
	if !q.Until.IsZero() {
		where = append(where, "started_at < ?")
		args = append(args, q.Until.UTC().UnixNano())
	}

	query := `SELECT id, host_id, kind, target, option_set, result, bytes_reclaimed,
	                 started_at, ended_at
	          FROM operations`
	if len(where) > 0 {
		// The joined fragments are constant column predicates with bound `?`
		// placeholders; every value travels via args, never the string (ADR-0004).
		query += " WHERE " + strings.Join(where, " AND ") //nolint:gosec // G202: predicates are constants; values are bound parameters
	}
	query += " ORDER BY started_at DESC"
	// Limit: >0 caps to that many; 0 applies the default cap; <0 is unbounded
	// (used by export, which is the whole record, not a page).
	if q.Limit >= 0 {
		limit := q.Limit
		if limit == 0 {
			limit = defaultOperationLimit
		}
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying operations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var ops []domain.Operation
	for rows.Next() {
		op, scanErr := scanOperation(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scanning operation: %w", scanErr)
		}
		ops = append(ops, op)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating operations: %w", err)
	}
	return ops, nil
}

// scanOperation reads one operations row into the domain type.
func scanOperation(row scanner) (domain.Operation, error) {
	var (
		op           domain.Operation
		kind         string
		optionSet    string
		startedNanos int64
		endedNanos   sql.NullInt64
	)
	if err := row.Scan(&op.ID, &op.HostRef, &kind, &op.Target, &optionSet, &op.Result,
		&op.BytesReclaimed, &startedNanos, &endedNanos); err != nil {
		return domain.Operation{}, err
	}
	op.Kind = domain.OperationKind(kind)
	op.StartedAt = time.Unix(0, startedNanos).UTC()
	if endedNanos.Valid {
		op.EndedAt = time.Unix(0, endedNanos.Int64).UTC()
	}
	if err := json.Unmarshal([]byte(optionSet), &op.OptionSet); err != nil {
		return domain.Operation{}, fmt.Errorf("decoding operation option set: %w", err)
	}
	return op, nil
}

// SavePruneImpact persists the per-category impact that was confirmed for an
// operation (PROJECT-BOOK §7.7). The operation must already exist (FK).
func (s *Store) SavePruneImpact(ctx context.Context, operationID string, impact domain.PruneImpact) error {
	for _, cat := range impact.Categories {
		_, err := s.db.ExecContext(ctx,
			`INSERT INTO prune_impacts (operation_id, category, object_count, reclaimable_bytes)
			 VALUES (?, ?, ?, ?)`,
			operationID, string(cat.Kind), cat.ObjectCount, cat.ReclaimableBytes)
		if err != nil {
			return fmt.Errorf("saving prune impact for operation %q: %w", operationID, err)
		}
	}
	return nil
}

// SaveResourceSample appends one rolling-history resource sample.
func (s *Store) SaveResourceSample(ctx context.Context, sample domain.ResourceSample) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO resource_samples (host_id, container_id, at, cpu_pct, mem_bytes,
		     net_rx, net_tx, blk_read, blk_write)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sample.HostRef, sample.ContainerID, sample.At.UTC().UnixNano(), sample.CPUPct,
		sample.MemBytes, sample.NetRx, sample.NetTx, sample.BlkRead, sample.BlkWrite)
	if err != nil {
		return fmt.Errorf("saving resource sample: %w", err)
	}
	return nil
}

// RecentResourceSamples returns up to limit most-recent samples for a container,
// oldest first (ready to plot).
func (s *Store) RecentResourceSamples(ctx context.Context, hostID, containerID string, limit int) ([]domain.ResourceSample, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT host_id, container_id, at, cpu_pct, mem_bytes, net_rx, net_tx, blk_read, blk_write
		 FROM resource_samples
		 WHERE host_id = ? AND container_id = ?
		 ORDER BY at DESC LIMIT ?`, hostID, containerID, limit)
	if err != nil {
		return nil, fmt.Errorf("querying resource samples: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var samples []domain.ResourceSample
	for rows.Next() {
		var (
			sample  domain.ResourceSample
			atNanos int64
		)
		if err := rows.Scan(&sample.HostRef, &sample.ContainerID, &atNanos, &sample.CPUPct,
			&sample.MemBytes, &sample.NetRx, &sample.NetTx, &sample.BlkRead, &sample.BlkWrite); err != nil {
			return nil, fmt.Errorf("scanning resource sample: %w", err)
		}
		sample.At = time.Unix(0, atNanos).UTC()
		samples = append(samples, sample)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating resource samples: %w", err)
	}

	// Reverse to oldest-first.
	for i, j := 0, len(samples)-1; i < j; i, j = i+1, j-1 {
		samples[i], samples[j] = samples[j], samples[i]
	}
	return samples, nil
}

// PruneResourceSamples deletes samples older than the cutoff (rolling retention).
func (s *Store) PruneResourceSamples(ctx context.Context, before time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM resource_samples WHERE at < ?`, before.UTC().UnixNano())
	if err != nil {
		return 0, fmt.Errorf("pruning resource samples: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("counting pruned samples: %w", err)
	}
	return n, nil
}
