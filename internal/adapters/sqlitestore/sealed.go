package sqlitestore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/drydock/drydock/internal/core/secret"
)

// ErrSealedValueNotFound is returned when no sealed value exists for an id.
var ErrSealedValueNotFound = errors.New("sealed value not found")

// PutSealedValue stores (or replaces) an AEAD-sealed value under id within a
// scope (e.g. "host.passphrase"). The plaintext is never written; only the
// nonce and ciphertext (ADR-0009).
func (s *Store) PutSealedValue(ctx context.Context, id, scope string, v secret.SealedValue) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sealed_values (id, scope, nonce, sealed_bytes)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET scope = excluded.scope,
		     nonce = excluded.nonce, sealed_bytes = excluded.sealed_bytes`,
		id, scope, v.Nonce, v.Ciphertext)
	if err != nil {
		return fmt.Errorf("storing sealed value %q: %w", id, err)
	}
	return nil
}

// GetSealedValue returns the sealed value for id, or ErrSealedValueNotFound.
func (s *Store) GetSealedValue(ctx context.Context, id string) (secret.SealedValue, error) {
	var v secret.SealedValue
	err := s.db.QueryRowContext(ctx,
		`SELECT nonce, sealed_bytes FROM sealed_values WHERE id = ?`, id).
		Scan(&v.Nonce, &v.Ciphertext)
	if errors.Is(err, sql.ErrNoRows) {
		return secret.SealedValue{}, ErrSealedValueNotFound
	}
	if err != nil {
		return secret.SealedValue{}, fmt.Errorf("reading sealed value %q: %w", id, err)
	}
	return v, nil
}
