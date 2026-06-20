// Package history enforces the rolling-retention policy for resource samples
// (PROJECT-BOOK §7.6/§7.7): live state is never persisted indefinitely; samples
// older than the retention window are pruned.
package history

import (
	"context"
	"fmt"
	"time"
)

// Pruner deletes resource samples older than a cutoff (satisfied by the store).
type Pruner interface {
	PruneResourceSamples(ctx context.Context, before time.Time) (int64, error)
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

// Sweep deletes samples older than now-window, returning the number removed.
func (r *Retention) Sweep(ctx context.Context, now time.Time) (int64, error) {
	n, err := r.pruner.PruneResourceSamples(ctx, now.Add(-r.window))
	if err != nil {
		return 0, fmt.Errorf("sweeping resource history: %w", err)
	}
	return n, nil
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
