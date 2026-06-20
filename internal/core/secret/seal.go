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
	encoded, err := store.Get(ctx, dataKeyName)
	switch {
	case err == nil:
		key, decErr := base64.StdEncoding.DecodeString(encoded)
		if decErr != nil {
			return nil, fmt.Errorf("decoding data key: %w", decErr)
		}
		if len(key) != chacha20poly1305.KeySize {
			return nil, fmt.Errorf("data key has wrong length %d", len(key))
		}
		return key, nil
	case errors.Is(err, ErrNotFound):
		key := make([]byte, chacha20poly1305.KeySize)
		if _, randErr := rand.Read(key); randErr != nil {
			return nil, fmt.Errorf("generating data key: %w", randErr)
		}
		if setErr := store.Set(ctx, dataKeyName, base64.StdEncoding.EncodeToString(key)); setErr != nil {
			return nil, fmt.Errorf("storing data key: %w", setErr)
		}
		return key, nil
	default:
		return nil, fmt.Errorf("loading data key: %w", err)
	}
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
