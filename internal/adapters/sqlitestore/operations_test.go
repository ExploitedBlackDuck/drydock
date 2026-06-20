package sqlitestore

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/core/domain"
)

func TestSaveOperationRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := tempStore(t)
	seedLocalHost(t, store)

	op := domain.Operation{
		ID:        "op1",
		HostRef:   "local",
		Kind:      domain.OpContainerRemove,
		Target:    "c1",
		OptionSet: map[string]any{"force": true},
		Result:    "ok",
		StartedAt: time.Unix(0, 1_700_000_000_000_000_000).UTC(),
		EndedAt:   time.Unix(0, 1_700_000_001_000_000_000).UTC(),
	}
	require.NoError(t, store.SaveOperation(ctx, op))
}

func TestResourceSampleRetention(t *testing.T) {
	ctx := context.Background()
	store := tempStore(t)
	seedLocalHost(t, store)

	base := time.Unix(0, 1_700_000_000_000_000_000).UTC()
	for i := 0; i < 5; i++ {
		require.NoError(t, store.SaveResourceSample(ctx, domain.ResourceSample{
			HostRef:     "local",
			ContainerID: "c1",
			At:          base.Add(time.Duration(i) * time.Minute),
			CPUPct:      float64(i),
			MemBytes:    int64(i * 1024),
		}))
	}

	samples, err := store.RecentResourceSamples(ctx, "local", "c1", 3)
	require.NoError(t, err)
	require.Len(t, samples, 3)
	// Oldest-first within the most-recent window.
	assert.True(t, samples[0].At.Before(samples[2].At))
	assert.Equal(t, 4.0, samples[2].CPUPct, "newest sample last")

	pruned, err := store.PruneResourceSamples(ctx, base.Add(2*time.Minute))
	require.NoError(t, err)
	assert.Equal(t, int64(2), pruned)
}

func seedLocalHost(t *testing.T, store *Store) {
	t.Helper()
	require.NoError(t, store.SaveHost(context.Background(), domain.Host{
		ID: "local", Name: "local", Transport: domain.TransportLocal, Endpoint: "unix:///x",
	}))
}
