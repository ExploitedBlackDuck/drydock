package sqlitestore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/adapters/auditmark"
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
	require.ErrorIs(t, err, ErrSchemaNewer, "refusal carries the dedicated schema-newer code")
}

// TestBackupProducesConsistentSnapshot verifies the SQLite backup path (ADR-0024):
// a VACUUM INTO snapshot opens as a valid database at the same schema version
// with the written rows present, even while another writer is busy.
func TestBackupProducesConsistentSnapshot(t *testing.T) {
	ctx := context.Background()
	store := tempStore(t)
	seedLocalHost(t, store)

	// A concurrent writer keeps appending audit rows during the backup.
	key := []byte("drydock-store-test-audit-key-0123")
	log := audit.New(store, nil, key, nil)
	done := make(chan struct{})
	go func() {
		for i := 0; i < 50; i++ {
			_, _ = log.Append(ctx, audit.Record{Action: domain.ActionContainerStart, Subject: "c"})
		}
		close(done)
	}()

	dest := filepath.Join(t.TempDir(), "snapshot.db")
	got, err := store.Backup(ctx, dest)
	require.NoError(t, err)
	assert.Equal(t, dest, got)
	<-done

	// The snapshot opens cleanly and carries the seeded host.
	backup, err := Open(ctx, dest)
	require.NoError(t, err)
	defer func() { _ = backup.Close() }()
	version, err := backup.SchemaVersion(ctx)
	require.NoError(t, err)
	assert.Positive(t, version)
	hosts, err := backup.Hosts(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, hosts, "the backup is a consistent, queryable database")

	// A second backup to the same path is refused rather than clobbering.
	_, err = store.Backup(ctx, dest)
	assert.Error(t, err)
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
		MAC:     "deadbeef",
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

	entry := domain.AuditEntry{Seq: 1, At: time.Now().UTC(), Action: domain.ActionHostConnect, MAC: "h1"}
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
	key := []byte("drydock-store-test-audit-key-0123")
	log := audit.New(store, nil, key, nil)

	for _, subject := range []string{"alpha", "beta", "gamma"} {
		_, err := log.Append(ctx, audit.Record{
			Action:  domain.ActionContainerRemove,
			HostRef: "host-1",
			Subject: subject,
			Detail:  map[string]any{"force": true},
		})
		require.NoError(t, err)
	}

	result, err := log.Verify(ctx)
	require.NoError(t, err)
	assert.Equal(t, audit.VerifyIntact, result.State)
	assert.Equal(t, 3, result.VerifiedCount)

	// Tamper with a persisted entry directly, bypassing the append path.
	_, err = store.db.ExecContext(ctx, `UPDATE audit_log SET subject = 'forged' WHERE seq = 2`)
	require.NoError(t, err)

	result, err = log.Verify(ctx)
	assert.ErrorIs(t, err, audit.ErrChainBroken)
	assert.Equal(t, audit.VerifyInPlaceTampered, result.State)
}

// TestAuditChainDetectsTruncationWithRealStore verifies that removing rows from
// the audit table's tail is caught via the external high-water mark, even though
// the remaining keyed chain is internally valid (ADR-0025, P1 gate).
func TestAuditChainDetectsTruncationWithRealStore(t *testing.T) {
	ctx := context.Background()
	store := tempStore(t)
	key := []byte("drydock-store-test-audit-key-0123")
	mark := auditmark.NewFile(filepath.Join(t.TempDir(), "audit.hwm"))
	log := audit.New(store, nil, key, mark)

	for _, subject := range []string{"alpha", "beta", "gamma", "delta"} {
		_, err := log.Append(ctx, audit.Record{Action: domain.ActionContainerStop, Subject: subject})
		require.NoError(t, err)
	}
	intact, err := log.Verify(ctx)
	require.NoError(t, err)
	require.Equal(t, audit.VerifyIntact, intact.State)

	// Delete the tail directly: the remaining chain still links cleanly, but the
	// high-water mark (seq 4) now exceeds the last row.
	_, err = store.db.ExecContext(ctx, `DELETE FROM audit_log WHERE seq > 2`)
	require.NoError(t, err)

	result, err := log.Verify(ctx)
	assert.ErrorIs(t, err, audit.ErrChainBroken)
	assert.Equal(t, audit.VerifyTruncated, result.State)
}
