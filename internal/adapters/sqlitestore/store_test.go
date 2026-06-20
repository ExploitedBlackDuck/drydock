package sqlitestore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/core/audit"
	"github.com/drydock/drydock/internal/core/domain"
	"github.com/drydock/drydock/internal/core/secret"
)

func tempStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	store, err := Open(context.Background(), path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestOpenAppliesMigrationsAndReportsSchemaVersion(t *testing.T) {
	store := tempStore(t)

	version, err := store.SchemaVersion(context.Background())
	require.NoError(t, err)

	latest, err := loadMigrations()
	require.NoError(t, err)
	assert.Equal(t, len(latest), version)
	assert.Positive(t, version)
}

func TestOpenIsIdempotent(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")

	first, err := Open(ctx, path)
	require.NoError(t, err)
	v1, err := first.SchemaVersion(ctx)
	require.NoError(t, err)
	require.NoError(t, first.Close())

	// Re-opening an already-migrated database must not re-run migrations or fail.
	second, err := Open(ctx, path)
	require.NoError(t, err)
	defer func() { _ = second.Close() }()
	v2, err := second.SchemaVersion(ctx)
	require.NoError(t, err)

	assert.Equal(t, v1, v2)
}

func TestOpenRejectsNewerSchema(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")

	store, err := Open(ctx, path)
	require.NoError(t, err)
	// Simulate a database written by a newer build.
	_, err = store.db.ExecContext(ctx,
		`INSERT INTO schema_migrations (version, applied_at) VALUES (9999, 0)`)
	require.NoError(t, err)
	require.NoError(t, store.Close())

	_, err = Open(ctx, path)
	require.ErrorIs(t, err, ErrMigration)
}

func TestSealedValueRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := tempStore(t)

	want := secret.SealedValue{Nonce: []byte("nonce-bytes-24-aaaaaaaaaa"), Ciphertext: []byte("sealed-ciphertext")}
	require.NoError(t, store.PutSealedValue(ctx, "host-1", "host.passphrase", want))

	got, err := store.GetSealedValue(ctx, "host-1")
	require.NoError(t, err)
	assert.Equal(t, want, got)

	// Replacing the same id updates in place.
	replacement := secret.SealedValue{Nonce: []byte("different-nonce-24-bbbbbb"), Ciphertext: []byte("new")}
	require.NoError(t, store.PutSealedValue(ctx, "host-1", "host.passphrase", replacement))
	got, err = store.GetSealedValue(ctx, "host-1")
	require.NoError(t, err)
	assert.Equal(t, replacement, got)
}

func TestGetSealedValueMissing(t *testing.T) {
	_, err := tempStore(t).GetSealedValue(context.Background(), "absent")
	assert.ErrorIs(t, err, ErrSealedValueNotFound)
}

func TestAuditEntryRoundTripThroughStore(t *testing.T) {
	ctx := context.Background()
	store := tempStore(t)

	_, ok, err := store.LastAuditEntry(ctx)
	require.NoError(t, err)
	assert.False(t, ok, "fresh log is empty")

	entry := domain.AuditEntry{
		Seq:     1,
		At:      time.Unix(0, 1_700_000_000_000_000_000).UTC(),
		Action:  domain.ActionHostConnect,
		HostRef: "host-1",
		Subject: "example-host",
		Detail:  map[string]any{"transport": "ssh"},
		Hash:    "deadbeef",
	}
	require.NoError(t, store.AppendAuditEntry(ctx, entry))

	got, ok, err := store.LastAuditEntry(ctx)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, entry, got)
}

func TestAppendAuditEntryRejectsDuplicateSeq(t *testing.T) {
	ctx := context.Background()
	store := tempStore(t)

	entry := domain.AuditEntry{Seq: 1, At: time.Now().UTC(), Action: domain.ActionHostConnect, Hash: "h1"}
	require.NoError(t, store.AppendAuditEntry(ctx, entry))

	// The primary key on seq enforces append-only positions.
	err := store.AppendAuditEntry(ctx, entry)
	assert.Error(t, err)
}

// TestAuditChainVerifyAndTamperWithRealStore exercises the audit log end to end
// over a real SQLite database: an intact chain verifies, and a row mutated
// behind the log's back is detected (PROJECT-BOOK §7.10 P1 gate).
func TestAuditChainVerifyAndTamperWithRealStore(t *testing.T) {
	ctx := context.Background()
	store := tempStore(t)
	log := audit.New(store, nil)

	for _, subject := range []string{"alpha", "beta", "gamma"} {
		_, err := log.Append(ctx, audit.Record{
			Action:  domain.ActionContainerRemove,
			HostRef: "host-1",
			Subject: subject,
			Detail:  map[string]any{"force": true},
		})
		require.NoError(t, err)
	}

	count, err := log.Verify(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	// Tamper with a persisted entry directly, bypassing the append path.
	_, err = store.db.ExecContext(ctx, `UPDATE audit_log SET subject = 'forged' WHERE seq = 2`)
	require.NoError(t, err)

	_, err = log.Verify(ctx)
	assert.ErrorIs(t, err, audit.ErrChainBroken)
}
