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

// seedOperations writes a small, varied operation history for query tests.
func seedOperations(t *testing.T, store *Store) {
	t.Helper()
	ctx := context.Background()
	base := time.Unix(0, 1_700_000_000_000_000_000).UTC()
	ops := []domain.Operation{
		{ID: "o1", HostRef: "local", Kind: domain.OpContainerStart, Target: "c1", Result: "ok", StartedAt: base},
		{ID: "o2", HostRef: "local", Kind: domain.OpContainerRemove, Target: "c1", Result: "ok", StartedAt: base.Add(1 * time.Minute)},
		{ID: "o3", HostRef: "local", Kind: domain.OpImagePrune, Target: "image.prune", Result: "ok", StartedAt: base.Add(2 * time.Minute)},
		{ID: "o4", HostRef: "remote", Kind: domain.OpContainerStop, Target: "c9", Result: "ok", StartedAt: base.Add(3 * time.Minute)},
	}
	for _, op := range ops {
		require.NoError(t, store.SaveOperation(ctx, op))
	}
}

func ids(ops []domain.Operation) []string {
	out := make([]string, len(ops))
	for i, op := range ops {
		out[i] = op.ID
	}
	return out
}

func TestOperationsOrderedMostRecentFirst(t *testing.T) {
	ctx := context.Background()
	store := tempStore(t)
	seedLocalHost(t, store)
	seedRemoteHost(t, store)
	seedOperations(t, store)

	ops, err := store.Operations(ctx, domain.OperationQuery{})
	require.NoError(t, err)
	assert.Equal(t, []string{"o4", "o3", "o2", "o1"}, ids(ops), "newest first")
}

func TestOperationsFilterByHost(t *testing.T) {
	ctx := context.Background()
	store := tempStore(t)
	seedLocalHost(t, store)
	seedRemoteHost(t, store)
	seedOperations(t, store)

	ops, err := store.Operations(ctx, domain.OperationQuery{HostRef: "remote"})
	require.NoError(t, err)
	assert.Equal(t, []string{"o4"}, ids(ops))
}

func TestOperationsFilterByKinds(t *testing.T) {
	ctx := context.Background()
	store := tempStore(t)
	seedLocalHost(t, store)
	seedRemoteHost(t, store)
	seedOperations(t, store)

	// Destructive-only, expressed as the domain's destructive kind set.
	ops, err := store.Operations(ctx, domain.OperationQuery{Kinds: domain.DestructiveKinds()})
	require.NoError(t, err)
	assert.Equal(t, []string{"o3", "o2"}, ids(ops), "only remove + image-prune are destructive")
}

func TestOperationsFilterByTimeWindow(t *testing.T) {
	ctx := context.Background()
	store := tempStore(t)
	seedLocalHost(t, store)
	seedRemoteHost(t, store)
	seedOperations(t, store)

	base := time.Unix(0, 1_700_000_000_000_000_000).UTC()
	ops, err := store.Operations(ctx, domain.OperationQuery{
		Since: base.Add(1 * time.Minute),
		Until: base.Add(3 * time.Minute), // exclusive
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"o3", "o2"}, ids(ops))
}

func TestOperationsLimitAndUnbounded(t *testing.T) {
	ctx := context.Background()
	store := tempStore(t)
	seedLocalHost(t, store)
	seedRemoteHost(t, store)
	seedOperations(t, store)

	limited, err := store.Operations(ctx, domain.OperationQuery{Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, []string{"o4", "o3"}, ids(limited))

	// A negative limit is unbounded (export path).
	all, err := store.Operations(ctx, domain.OperationQuery{Limit: -1})
	require.NoError(t, err)
	assert.Len(t, all, 4)
}

func TestOperationRoundTripDecodesFields(t *testing.T) {
	ctx := context.Background()
	store := tempStore(t)
	seedLocalHost(t, store)

	want := domain.Operation{
		ID: "rt", HostRef: "local", Kind: domain.OpContainerRemove, Target: "c1",
		OptionSet: map[string]any{"force": true}, Result: "ok", BytesReclaimed: 2048,
		StartedAt: time.Unix(0, 1_700_000_000_000_000_000).UTC(),
		EndedAt:   time.Unix(0, 1_700_000_005_000_000_000).UTC(),
	}
	require.NoError(t, store.SaveOperation(ctx, want))

	got, err := store.Operations(ctx, domain.OperationQuery{HostRef: "local"})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, want.Kind, got[0].Kind)
	assert.Equal(t, want.Target, got[0].Target)
	assert.Equal(t, int64(2048), got[0].BytesReclaimed)
	assert.Equal(t, true, got[0].OptionSet["force"])
	assert.True(t, want.EndedAt.Equal(got[0].EndedAt))
}

func seedRemoteHost(t *testing.T, store *Store) {
	t.Helper()
	require.NoError(t, store.SaveHost(context.Background(), domain.Host{
		ID: "remote", Name: "remote", Transport: domain.TransportSSH, Endpoint: "ssh://x",
	}))
}

func seedLocalHost(t *testing.T, store *Store) {
	t.Helper()
	require.NoError(t, store.SaveHost(context.Background(), domain.Host{
		ID: "local", Name: "local", Transport: domain.TransportLocal, Endpoint: "unix:///x",
	}))
}
