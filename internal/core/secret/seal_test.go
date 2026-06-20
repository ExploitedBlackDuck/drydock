package secret_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/core/secret"
)

// memSecrets is an in-memory SecretStore for testing without an OS keyring.
type memSecrets struct {
	m map[string]string
}

func newMemSecrets() *memSecrets { return &memSecrets{m: map[string]string{}} }

func (s *memSecrets) Get(_ context.Context, key string) (string, error) {
	v, ok := s.m[key]
	if !ok {
		return "", secret.ErrNotFound
	}
	return v, nil
}

func (s *memSecrets) Set(_ context.Context, key, value string) error {
	s.m[key] = value
	return nil
}

func (s *memSecrets) Delete(_ context.Context, key string) error {
	delete(s.m, key)
	return nil
}

func TestSealOpenRoundTrip(t *testing.T) {
	ctx := context.Background()
	sealer, err := secret.NewSealer(ctx, newMemSecrets())
	require.NoError(t, err)

	plaintext := []byte("an SSH passphrase")
	sealed, err := sealer.Seal(plaintext)
	require.NoError(t, err)
	assert.NotEmpty(t, sealed.Nonce)
	assert.NotContains(t, string(sealed.Ciphertext), "passphrase", "ciphertext must not reveal plaintext")

	opened, err := sealer.Open(sealed)
	require.NoError(t, err)
	assert.Equal(t, plaintext, opened)
}

func TestSealUsesFreshNoncePerCall(t *testing.T) {
	sealer, err := secret.NewSealer(context.Background(), newMemSecrets())
	require.NoError(t, err)

	a, err := sealer.Seal([]byte("same"))
	require.NoError(t, err)
	b, err := sealer.Seal([]byte("same"))
	require.NoError(t, err)

	assert.NotEqual(t, a.Nonce, b.Nonce)
	assert.NotEqual(t, a.Ciphertext, b.Ciphertext, "same plaintext seals to different ciphertext")
}

func TestDataKeyPersistsAcrossSealers(t *testing.T) {
	ctx := context.Background()
	store := newMemSecrets()

	first, err := secret.NewSealer(ctx, store)
	require.NoError(t, err)
	sealed, err := first.Seal([]byte("durable"))
	require.NoError(t, err)

	// A new sealer over the same store reuses the persisted data key.
	second, err := secret.NewSealer(ctx, store)
	require.NoError(t, err)
	opened, err := second.Open(sealed)
	require.NoError(t, err)
	assert.Equal(t, []byte("durable"), opened)
}

func TestOpenFailsWithWrongKey(t *testing.T) {
	ctx := context.Background()

	sealer, err := secret.NewSealer(ctx, newMemSecrets())
	require.NoError(t, err)
	sealed, err := sealer.Seal([]byte("secret"))
	require.NoError(t, err)

	// A different install (fresh keyring) generates a different data key.
	other, err := secret.NewSealer(ctx, newMemSecrets())
	require.NoError(t, err)
	_, err = other.Open(sealed)
	assert.Error(t, err, "a value sealed under a different key must not open")
}

func TestOpenFailsOnTamperedCiphertext(t *testing.T) {
	sealer, err := secret.NewSealer(context.Background(), newMemSecrets())
	require.NoError(t, err)
	sealed, err := sealer.Seal([]byte("secret"))
	require.NoError(t, err)

	sealed.Ciphertext[0] ^= 0xFF // flip a bit

	_, err = sealer.Open(sealed)
	assert.Error(t, err, "AEAD authentication must reject tampering")
}
