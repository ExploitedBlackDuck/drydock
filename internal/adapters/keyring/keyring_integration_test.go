//go:build integration

// These tests touch the real OS keyring and are gated behind the integration
// build tag (PROJECT-BOOK §2.5). They are skipped by the default `go test ./...`
// so a bare machine stays green; run them with `task test:integration`.

package keyring_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/adapters/keyring"
	"github.com/drydock/drydock/internal/core/secret"
)

func TestKeyringRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := keyring.New()
	const key = "integration-test-key"

	// Clean up regardless of outcome.
	t.Cleanup(func() { _ = store.Delete(ctx, key) })

	require.NoError(t, store.Set(ctx, key, "s3cr3t-value"))

	got, err := store.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, "s3cr3t-value", got)

	require.NoError(t, store.Delete(ctx, key))

	_, err = store.Get(ctx, key)
	assert.ErrorIs(t, err, secret.ErrNotFound)
}

func TestKeyringGetMissingReturnsNotFound(t *testing.T) {
	_, err := keyring.New().Get(context.Background(), "definitely-not-set-xyz")
	assert.True(t, errors.Is(err, secret.ErrNotFound))
}
