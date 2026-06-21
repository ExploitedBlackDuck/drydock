package secret

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

// dataKeyName is the keyring entry holding the per-install data key.
const dataKeyName = "data-key"

// auditKeyName is the keyring entry holding the per-install audit HMAC key.
const auditKeyName = "audit-key"

// auditKeySize is the HMAC-SHA256 key length for the audit chain (ADR-0025).
const auditKeySize = 32

// LoadOrCreateAuditKey returns the per-install audit HMAC key, creating and
// persisting a fresh random key in the keyring on first run (ADR-0025). The key
// is separate from the data key so the audit chain's integrity does not depend
// on the value-sealing key. When the keyring is unavailable the caller runs the
// audit log in its degraded key-unavailable mode (it must not block startup).
func LoadOrCreateAuditKey(ctx context.Context, store SecretStore) ([]byte, error) {
	return loadOrCreateKey(ctx, store, auditKeyName, auditKeySize)
}

// loadOrCreateKey fetches a base64-encoded keyring secret of the expected size,
// creating a fresh random one if absent.
func loadOrCreateKey(ctx context.Context, store SecretStore, name string, size int) ([]byte, error) {
	encoded, err := store.Get(ctx, name)
	switch {
	case err == nil:
		key, decErr := base64.StdEncoding.DecodeString(encoded)
		if decErr != nil {
			return nil, fmt.Errorf("decoding %s: %w", name, decErr)
		}
		if len(key) != size {
			return nil, fmt.Errorf("%s has wrong length %d", name, len(key))
		}
		return key, nil
	case errors.Is(err, ErrNotFound):
		key := make([]byte, size)
		if _, randErr := rand.Read(key); randErr != nil {
			return nil, fmt.Errorf("generating %s: %w", name, randErr)
		}
		if setErr := store.Set(ctx, name, base64.StdEncoding.EncodeToString(key)); setErr != nil {
			return nil, fmt.Errorf("storing %s: %w", name, setErr)
		}
		return key, nil
	default:
		return nil, fmt.Errorf("loading %s: %w", name, err)
	}
}

// SealedValue is a value encrypted with the data key: a per-seal random nonce
// and the AEAD ciphertext. It maps directly to the sealed_values table columns
// (PROJECT-BOOK §7.7).
type SealedValue struct {
	Nonce      []byte
	Ciphertext []byte
}

// Sealer seals and opens sensitive values with the per-install data key using
// XChaCha20-Poly1305 (ADR-0009). The data key is held only in memory after being
// loaded from, or created in, the OS keyring.
type Sealer struct {
	aead interface {
		Seal(dst, nonce, plaintext, additionalData []byte) []byte
		Open(dst, nonce, ciphertext, additionalData []byte) ([]byte, error)
		NonceSize() int
	}
}

// NewSealer loads the per-install data key from the secret store, creating and
// persisting a fresh random key on first run.
func NewSealer(ctx context.Context, store SecretStore) (*Sealer, error) {
	key, err := loadOrCreateDataKey(ctx, store)
	if err != nil {
		return nil, err
	}
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("initializing AEAD: %w", err)
	}
	return &Sealer{aead: aead}, nil
}

func loadOrCreateDataKey(ctx context.Context, store SecretStore) ([]byte, error) {
	return loadOrCreateKey(ctx, store, dataKeyName, chacha20poly1305.KeySize)
}

// Seal encrypts plaintext, returning a fresh-nonce SealedValue.
func (s *Sealer) Seal(plaintext []byte) (SealedValue, error) {
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return SealedValue{}, fmt.Errorf("generating nonce: %w", err)
	}
	ciphertext := s.aead.Seal(nil, nonce, plaintext, nil)
	return SealedValue{Nonce: nonce, Ciphertext: ciphertext}, nil
}

// Open decrypts a SealedValue, returning the original plaintext. A wrong key or
// tampered ciphertext fails authentication and returns an error.
func (s *Sealer) Open(v SealedValue) ([]byte, error) {
	if len(v.Nonce) != s.aead.NonceSize() {
		return nil, fmt.Errorf("nonce has wrong length %d", len(v.Nonce))
	}
	plaintext, err := s.aead.Open(nil, v.Nonce, v.Ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("opening sealed value: %w", err)
	}
	return plaintext, nil
}
