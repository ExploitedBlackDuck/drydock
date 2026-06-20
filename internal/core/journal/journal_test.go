package journal_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/core/audit"
	"github.com/drydock/drydock/internal/core/domain"
	"github.com/drydock/drydock/internal/core/journal"
)

// fakeStore records the query it was handed and returns canned data.
type fakeStore struct {
	lastQuery domain.OperationQuery
	ops       []domain.Operation
	entries   []domain.AuditEntry
}

func (f *fakeStore) Operations(_ context.Context, q domain.OperationQuery) ([]domain.Operation, error) {
	f.lastQuery = q
	return f.ops, nil
}

func (f *fakeStore) AuditEntries(context.Context) ([]domain.AuditEntry, error) {
	return f.entries, nil
}

// chain builds a valid hash-chained audit log from records.
func chain(t *testing.T, records ...audit.Record) []domain.AuditEntry {
	t.Helper()
	var entries []domain.AuditEntry
	prevHash := ""
	at := time.Unix(0, 1_700_000_000_000_000_000).UTC()
	for i, r := range records {
		e := domain.AuditEntry{
			Seq: int64(i + 1), At: at.Add(time.Duration(i) * time.Second),
			Action: r.Action, HostRef: r.HostRef, Subject: r.Subject, Detail: r.Detail,
			PrevHash: prevHash,
		}
		hash, err := audit.ComputeHash(prevHash, e)
		require.NoError(t, err)
		e.Hash = hash
		prevHash = hash
		entries = append(entries, e)
	}
	return entries
}

func TestDestructiveOnlyUsesDomainKinds(t *testing.T) {
	store := &fakeStore{}
	svc := journal.New(store)

	_, err := svc.Operations(context.Background(), journal.Filter{DestructiveOnly: true})
	require.NoError(t, err)
	assert.ElementsMatch(t, domain.DestructiveKinds(), store.lastQuery.Kinds)
}

func TestKindFilterPassesSingleKind(t *testing.T) {
	store := &fakeStore{}
	svc := journal.New(store)

	_, err := svc.Operations(context.Background(), journal.Filter{
		HostRef: "h1", Kind: string(domain.OpContainerStart), Limit: 50,
	})
	require.NoError(t, err)
	assert.Equal(t, "h1", store.lastQuery.HostRef)
	assert.Equal(t, []domain.OperationKind{domain.OpContainerStart}, store.lastQuery.Kinds)
	assert.Equal(t, 50, store.lastQuery.Limit)
}

func TestAuditTrailReportsIntactChain(t *testing.T) {
	store := &fakeStore{entries: chain(
		t,
		audit.Record{Action: domain.ActionHostConnect, Subject: "local"},
		audit.Record{Action: domain.ActionContainerStop, Subject: "c1"},
	)}
	svc := journal.New(store)

	status, err := svc.AuditTrail(context.Background())
	require.NoError(t, err)
	assert.True(t, status.Verified)
	assert.Equal(t, 2, status.VerifiedCount)
	assert.Empty(t, status.Error)
}

func TestAuditTrailDetectsTampering(t *testing.T) {
	entries := chain(
		t,
		audit.Record{Action: domain.ActionHostConnect, Subject: "local"},
		audit.Record{Action: domain.ActionContainerRemove, Subject: "c1"},
	)
	// Tamper with a recorded entry's content after it was hashed.
	entries[1].Subject = "c2"
	store := &fakeStore{entries: entries}
	svc := journal.New(store)

	status, err := svc.AuditTrail(context.Background())
	require.NoError(t, err)
	assert.False(t, status.Verified)
	assert.Equal(t, 1, status.VerifiedCount, "first entry verified before the break")
	assert.Contains(t, status.Error, "audit chain broken")
}

func TestExportRoundTripsAndChainVerifies(t *testing.T) {
	store := &fakeStore{
		ops: []domain.Operation{{
			ID: "op1", HostRef: "local", Kind: domain.OpContainerRemove, Target: "c1",
			OptionSet: map[string]any{"force": true, "ack": true}, Result: "ok",
			StartedAt: time.Unix(0, 1_700_000_000_000_000_000).UTC(),
			EndedAt:   time.Unix(0, 1_700_000_001_000_000_000).UTC(),
		}},
		entries: chain(
			t,
			audit.Record{Action: domain.ActionHostConnect, Subject: "local"},
			audit.Record{Action: domain.ActionContainerRemove, Subject: "c1", Detail: map[string]any{"ack": true}},
		),
	}
	svc := journal.New(store)

	exp, err := svc.Export(context.Background())
	require.NoError(t, err)
	assert.True(t, exp.AuditVerified)
	assert.Equal(t, journal.ExportSchemaVersion, exp.SchemaVersion)

	// Export uses an unbounded query so the whole record is captured.
	assert.Equal(t, -1, store.lastQuery.Limit)

	data, err := journal.MarshalExport(exp)
	require.NoError(t, err)

	got, err := journal.ParseExport(data)
	require.NoError(t, err)
	assert.Equal(t, exp, got, "export round-trips through JSON unchanged")

	// The exported chain verifies independently of any database.
	n, err := journal.VerifyExportedChain(got)
	require.NoError(t, err)
	assert.Equal(t, 2, n)
}
