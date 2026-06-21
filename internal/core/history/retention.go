// Package history enforces the rolling-retention policy for live state
// (PROJECT-BOOK §7.6/§7.7/§7.12.4): resource samples and host-timeline entries
// are never persisted indefinitely; rows older than the retention window are
// pruned.
package history

import (
	"context"
	"fmt"
	"time"
)

// Pruner deletes rolling-history rows older than a cutoff (satisfied by the
// store) — resource samples and timeline entries share the retention window.
type Pruner interface {
	PruneResourceSamples(ctx context.Context, before time.Time) (int64, error)
	PruneTimelineEntries(ctx context.Context, before time.Time) (int64, error)
}

// Retention sweeps samples outside the configured window.
type Retention struct {
	pruner Pruner
	window time.Duration
}

// NewRetention returns a retention sweeper for the given window.
func NewRetention(pruner Pruner, window time.Duration) *Retention {
	return &Retention{pruner: pruner, window: window}
}

// Sweep deletes resource samples and timeline entries older than now-window,
// returning the total number removed.
func (r *Retention) Sweep(ctx context.Context, now time.Time) (int64, error) {
	cutoff := now.Add(-r.window)
	samples, err := r.pruner.PruneResourceSamples(ctx, cutoff)
	if err != nil {
		return 0, fmt.Errorf("sweeping resource history: %w", err)
	}
	timeline, err := r.pruner.PruneTimelineEntries(ctx, cutoff)
	if err != nil {
		return samples, fmt.Errorf("sweeping timeline: %w", err)
	}
	return samples + timeline, nil
}

// Run sweeps immediately and then every interval until ctx is cancelled. It owns
// its ticker and returns when ctx is done (no leaked goroutine).
func (r *Retention) Run(ctx context.Context, interval time.Duration, now func() time.Time) {
	if now == nil {
		now = time.Now
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	_, _ = r.Sweep(ctx, now())
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = r.Sweep(ctx, now())
		}
	}
}
