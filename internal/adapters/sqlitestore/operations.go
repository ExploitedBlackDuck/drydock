package sqlitestore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/drydock/drydock/internal/core/domain"
)

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
