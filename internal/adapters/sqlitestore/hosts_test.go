package sqlitestore

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/core/audit"
	"github.com/drydock/drydock/internal/core/domain"
)

// TestDeleteHostCascadesButPreservesAudit verifies that removing a host with
// recorded history succeeds — its operations, samples, and timeline go — while
// the independent, hash-chained audit log is left intact (ADR-0010).
func TestDeleteHostCascadesButPreservesAudit(t *testing.T) {
	ctx := context.Background()
	store := tempStore(t)
	seedLocalHost(t, store)

	require.NoError(t, store.SaveOperation(ctx, domain.Operation{
		ID: "op1", HostRef: "local", Kind: domain.OpContainerRemove, Target: "c1",
		Result: "ok", StartedAt: time.Now().UTC(),
	}))
	require.NoError(t, store.SaveResourceSample(ctx, domain.ResourceSample{
		HostRef: "local", ContainerID: "c1", At: time.Now().UTC(),
	}))
	require.NoError(t, store.SaveTimelineEntry(ctx, domain.TimelineEntry{
		HostRef: "local", At: time.Now().UTC(), Source: domain.TimelineEngine, Kind: "die",
	}))
	log := audit.New(store, nil, []byte("drydock-store-test-audit-key-0123"), nil)
	_, err := log.Append(ctx, audit.Record{Action: domain.ActionHostDisconnect, HostRef: "local", Subject: "local"})
	require.NoError(t, err)

	// Removal succeeds despite the foreign-keyed history.
	require.NoError(t, store.DeleteHost(ctx, "local"))

	ops, err := store.Operations(ctx, domain.OperationQuery{HostRef: "local"})
	require.NoError(t, err)
	assert.Empty(t, ops, "the host's operations are removed")
	timeline, err := store.RecentTimelineEntries(ctx, "local", 10)
	require.NoError(t, err)
	assert.Empty(t, timeline, "the host's timeline is removed")

	// The audit log is preserved — its actions survive the host's removal.
	entries, err := store.AuditEntries(ctx)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "the audit log is independent of the host")
}

func TestHostCRUD(t *testing.T) {
	ctx := context.Background()
	store := tempStore(t)

	host := domain.Host{
		ID:            "h1",
		Name:          "prod-1",
		Transport:     domain.TransportSSH,
		Endpoint:      "ssh://user@example-host",
		ObserveMode:   true,
		EngineVersion: "28.5.2",
		APIVersion:    "1.51",
	}
	require.NoError(t, store.SaveHost(ctx, host))

	got, err := store.Host(ctx, "h1")
	require.NoError(t, err)
	assert.Equal(t, "prod-1", got.Name)
	assert.True(t, got.ObserveMode)
	assert.Equal(t, domain.TrustTrusted, got.Trust, "trust derived on load")

	// Update via upsert.
	host.Name = "prod-renamed"
	require.NoError(t, store.SaveHost(ctx, host))
	got, err = store.Host(ctx, "h1")
	require.NoError(t, err)
	assert.Equal(t, "prod-renamed", got.Name)

	all, err := store.Hosts(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 1)

	require.NoError(t, store.DeleteHost(ctx, "h1"))
	_, err = store.Host(ctx, "h1")
	assert.ErrorIs(t, err, ErrHostNotFound)
}

func TestSetHostObserveMode(t *testing.T) {
	ctx := context.Background()
	store := tempStore(t)

	require.NoError(t, store.SaveHost(ctx, domain.Host{
		ID: "h1", Name: "n", Transport: domain.TransportLocal, Endpoint: "unix:///x",
	}))

	require.NoError(t, store.SetHostObserveMode(ctx, "h1", true))
	got, err := store.Host(ctx, "h1")
	require.NoError(t, err)
	assert.True(t, got.ObserveMode)

	assert.ErrorIs(t, store.SetHostObserveMode(ctx, "missing", true), ErrHostNotFound)
}

func TestUntrustedTransportDerivedOnLoad(t *testing.T) {
	ctx := context.Background()
	store := tempStore(t)

	require.NoError(t, store.SaveHost(ctx, domain.Host{
		ID: "h2", Name: "insecure", Transport: domain.TransportSSH, Endpoint: "tcp://example-host:2375",
	}))
	got, err := store.Host(ctx, "h2")
	require.NoError(t, err)
	assert.Equal(t, domain.TrustUntrusted, got.Trust)
}
