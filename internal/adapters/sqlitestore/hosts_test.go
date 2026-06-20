package sqlitestore

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/core/domain"
)

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
