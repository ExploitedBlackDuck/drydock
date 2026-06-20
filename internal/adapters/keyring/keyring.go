// Package keyring implements the secret.SecretStore port against the OS keyring
// (macOS Keychain / Linux Secret Service) via github.com/zalando/go-keyring
// (ADR-0009). It holds the per-install data key and any retained passphrases;
// SSH private keys are referenced elsewhere, never stored here.
package keyring

import (
	"context"
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"

	"github.com/drydock/drydock/internal/core/secret"
)

// service is the keyring service name under which all Drydock secrets are filed.
const service = "drydock"

// Store is the OS-keyring-backed SecretStore.
type Store struct {
	service string
}

// New returns a keyring-backed secret store.
func New() *Store {
	return &Store{service: service}
}

// Get returns the value for key, mapping an absent entry to secret.ErrNotFound
// and any backend failure to secret.ErrUnavailable.
func (s *Store) Get(ctx context.Context, key string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	value, err := keyring.Get(s.service, key)
	switch {
	case err == nil:
		return value, nil
	case errors.Is(err, keyring.ErrNotFound):
		return "", secret.ErrNotFound
	default:
		return "", fmt.Errorf("reading %q from keyring: %w", key, errors.Join(secret.ErrUnavailable, err))
	}
}

// Set stores value under key.
func (s *Store) Set(ctx context.Context, key, value string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := keyring.Set(s.service, key, value); err != nil {
		return fmt.Errorf("writing %q to keyring: %w", key, errors.Join(secret.ErrUnavailable, err))
	}
	return nil
}

// Delete removes key; an absent key is not an error.
func (s *Store) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	err := keyring.Delete(s.service, key)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("deleting %q from keyring: %w", key, errors.Join(secret.ErrUnavailable, err))
	}
	return nil
}
