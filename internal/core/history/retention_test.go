package history_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/core/history"
)

type fakePruner struct {
	cutoff   time.Time
	called   bool
	removed  int64
	timeline int64
}

func (p *fakePruner) PruneResourceSamples(_ context.Context, before time.Time) (int64, error) {
	p.called = true
	p.cutoff = before
	return p.removed, nil
}

func (p *fakePruner) PruneTimelineEntries(_ context.Context, before time.Time) (int64, error) {
	p.cutoff = before
	return p.timeline, nil
}

func TestSweepPrunesByRetentionWindow(t *testing.T) {
	pruner := &fakePruner{removed: 7, timeline: 3}
	r := history.NewRetention(pruner, 24*time.Hour)
	now := time.Unix(1_700_000_000, 0).UTC()

	removed, err := r.Sweep(context.Background(), now)
	require.NoError(t, err)

	assert.True(t, pruner.called)
	assert.Equal(t, int64(10), removed, "resource samples + timeline entries")
	assert.Equal(t, now.Add(-24*time.Hour), pruner.cutoff, "cutoff is now minus the retention window")
}
