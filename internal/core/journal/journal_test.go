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

var testKey = []byte("drydock-test-audit-key-0123456789")

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

// fakeVerifier returns a canned verification result (the journal delegates the
// cryptographic check to the audit package).
type fakeVerifier struct {
	result audit.VerifyResult
	err    error
}

func (v fakeVerifier) Verify(context.Context) (audit.VerifyResult, error) {
	return v.result, v.err
}

// chain builds a valid keyed hash-chained audit log from records.
func chain(t *testing.T, records ...audit.Record) []domain.AuditEntry {
	t.Helper()
	var entries []domain.AuditEntry
	prevMAC := ""
	at := time.Unix(0, 1_700_000_000_000_000_000).UTC()
	for i, r := range records {
		e := domain.AuditEntry{
			Seq: int64(i + 1), At: at.Add(time.Duration(i) * time.Second),
			Action: r.Action, HostRef: r.HostRef, Subject: r.Subject, Detail: r.Detail,
			PrevMAC: prevMAC,
		}
		mac, err := audit.ComputeMAC(testKey, prevMAC, e)
		require.NoError(t, err)
		e.MAC = mac
		prevMAC = mac
		entries = append(entries, e)
	}
	return entries
}

func TestDestructiveOnlyUsesDomainKinds(t *testing.T) {
	store := &fakeStore{}
	svc := journal.New(store, fakeVerifier{})

	_, err := svc.Operations(context.Background(), journal.Filter{DestructiveOnly: true})
	require.NoError(t, err)
	assert.ElementsMatch(t, domain.DestructiveKinds(), store.lastQuery.Kinds)
}

func TestKindFilterPassesSingleKind(t *testing.T) {
	store := &fakeStore{}
	svc := journal.New(store, fakeVerifier{})

	_, err := svc.Operations(context.Background(), journal.Filter{
		HostRef: "h1", Kind: string(domain.OpContainerStart), Limit: 50,
	})
	require.NoError(t, err)
	assert.Equal(t, "h1", store.lastQuery.HostRef)
	assert.Equal(t, []domain.OperationKind{domain.OpContainerStart}, store.lastQuery.Kinds)
	assert.Equal(t, 50, store.lastQuery.Limit)
}

func TestAuditTrailForwardsIntactState(t *testing.T) {
	store := &fakeStore{entries: chain(
		t,
		audit.Record{Action: domain.ActionHostConnect, Subject: "local"},
		audit.Record{Action: domain.ActionContainerStop, Subject: "c1"},
	)}
	svc := journal.New(store, fakeVerifier{result: audit.VerifyResult{State: audit.VerifyIntact, VerifiedCount: 2}})

	status, err := svc.AuditTrail(context.Background())
	require.NoError(t, err)
	assert.True(t, status.Verified)
	assert.Equal(t, "intact", status.State)
	assert.Equal(t, 2, status.VerifiedCount)
	assert.Len(t, status.Entries, 2)
	assert.Empty(t, status.Error)
}

func TestAuditTrailForwardsTamperedState(t *testing.T) {
	store := &fakeStore{entries: chain(t, audit.Record{Action: domain.ActionHostConnect, Subject: "local"})}
	svc := journal.New(store, fakeVerifier{
		result: audit.VerifyResult{State: audit.VerifyInPlaceTampered, VerifiedCount: 1},
		err:    audit.ErrChainBroken,
	})

	status, err := svc.AuditTrail(context.Background())
	require.NoError(t, err)
	assert.False(t, status.Verified)
	assert.Equal(t, "in_place_tampered", status.State)
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
	svc := journal.New(store, fakeVerifier{result: audit.VerifyResult{State: audit.VerifyIntact, VerifiedCount: 2}})

	exp, err := svc.Export(context.Background())
	require.NoError(t, err)
	assert.True(t, exp.AuditVerified)
	assert.Equal(t, "intact", exp.AuditState)
	assert.Equal(t, journal.ExportSchemaVersion, exp.SchemaVersion)
	assert.Equal(t, -1, store.lastQuery.Limit, "export uses an unbounded query")

	data, err := journal.MarshalExport(exp)
	require.NoError(t, err)

	got, err := journal.ParseExport(data)
	require.NoError(t, err)
	assert.Equal(t, exp, got, "export round-trips through JSON unchanged")

	// The exported chain is structurally consistent without the install key.
	n, err := journal.VerifyExportedChain(got)
	require.NoError(t, err)
	assert.Equal(t, 2, n)
}
