// Package secret defines the secret-storage port and the application-layer
// sealing of sensitive values (PROJECT-BOOK §7.9, ADR-0009). Secrets live in the
// OS keyring; sensitive persisted fields are AEAD-sealed before they touch disk,
// since the pure-Go SQLite driver offers no transparent encryption (ADR-0007).
package secret

import (
	"context"
	"errors"
)

// ErrNotFound is returned by a SecretStore when no value exists for a key.
var ErrNotFound = errors.New("secret not found")

// ErrUnavailable signals that the secret backend itself could not be reached
// (e.g. the OS keyring is locked or absent). Maps to ERR_SECRET_UNAVAILABLE
// (PROJECT-BOOK §8.4).
var ErrUnavailable = errors.New("secret store unavailable")

// SecretStore is the port for the OS keyring. It is consumer-defined here
// (PROJECT-BOOK §2.1) and implemented by the keyring adapter. Values are opaque
// strings; callers encode binary secrets (e.g. base64) as needed.
type SecretStore interface {
	// Get returns the value for key, or ErrNotFound if absent.
	Get(ctx context.Context, key string) (string, error)
	// Set stores value under key, replacing any existing value.
	Set(ctx context.Context, key, value string) error
	// Delete removes key; removing an absent key is not an error.
	Delete(ctx context.Context, key string) error
}
