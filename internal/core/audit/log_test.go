package audit_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/core/audit"
	"github.com/drydock/drydock/internal/core/domain"
)

var testKey = []byte("drydock-test-audit-key-0123456789")

// memStore is an in-memory audit.Store for testing the chain logic without a DB.
type memStore struct {
	entries []domain.AuditEntry
}

func (m *memStore) LastAuditEntry(_ context.Context) (domain.AuditEntry, bool, error) {
	if len(m.entries) == 0 {
		return domain.AuditEntry{}, false, nil
	}
	return m.entries[len(m.entries)-1], true, nil
}

func (m *memStore) AppendAuditEntry(_ context.Context, e domain.AuditEntry) error {
	m.entries = append(m.entries, e)
	return nil
}

func (m *memStore) AuditEntries(_ context.Context) ([]domain.AuditEntry, error) {
	return m.entries, nil
}

// memMark is an in-memory audit.MarkStore.
type memMark struct {
	m  audit.HighWaterMark
	ok bool
}

func (mm *memMark) Get(context.Context) (audit.HighWaterMark, bool, error) {
	return mm.m, mm.ok, nil
}

func (mm *memMark) Set(_ context.Context, m audit.HighWaterMark) error {
	mm.m, mm.ok = m, true
	return nil
}

func fixedClock() audit.Clock {
	t := time.Unix(0, 1_700_000_000_000_000_000).UTC()
	return func() time.Time { return t }
}

func appendN(t *testing.T, log *audit.Log, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		_, err := log.Append(context.Background(), audit.Record{Action: domain.ActionContainerStart, Subject: "c"})
		require.NoError(t, err)
	}
}

func TestAppendAssignsSequenceAndChains(t *testing.T) {
	ctx := context.Background()
	log := audit.New(&memStore{}, fixedClock(), testKey, &memMark{})

	first, err := log.Append(ctx, audit.Record{Action: domain.ActionHostConnect, HostRef: "h1", Subject: "alpha"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), first.Seq)
	assert.Empty(t, first.PrevMAC, "first entry has no predecessor")
	assert.NotEmpty(t, first.MAC)

	second, err := log.Append(ctx, audit.Record{Action: domain.ActionContainerStop, HostRef: "h1", Subject: "beta"})
	require.NoError(t, err)
	assert.Equal(t, int64(2), second.Seq)
	assert.Equal(t, first.MAC, second.PrevMAC, "each entry links to the previous MAC")
}

func TestVerifyAcceptsIntactChain(t *testing.T) {
	ctx := context.Background()
	log := audit.New(&memStore{}, fixedClock(), testKey, &memMark{})
	appendN(t, log, 5)

	result, err := log.Verify(ctx)
	require.NoError(t, err)
	assert.Equal(t, audit.VerifyIntact, result.State)
	assert.Equal(t, 5, result.VerifiedCount)
}

func TestVerifyDetectsInPlaceTampering(t *testing.T) {
	ctx := context.Background()
	store := &memStore{}
	log := audit.New(store, fixedClock(), testKey, &memMark{})
	appendN(t, log, 3)

	// Mutate a recorded entry's content without recomputing its MAC.
	store.entries[1].Subject = "tampered"

	result, err := log.Verify(ctx)
	assert.ErrorIs(t, err, audit.ErrChainBroken)
	assert.Equal(t, audit.VerifyInPlaceTampered, result.State)
	assert.Equal(t, 1, result.VerifiedCount, "the first entry verified before the break")
}

func TestVerifyDetectsTruncation(t *testing.T) {
	ctx := context.Background()
	store := &memStore{}
	mark := &memMark{}
	log := audit.New(store, fixedClock(), testKey, mark)
	appendN(t, log, 4)

	// The mark now records seq 4. Truncate the table's tail; the remaining chain
	// is internally valid but shorter than the mark.
	store.entries = store.entries[:2]

	result, err := log.Verify(ctx)
	assert.ErrorIs(t, err, audit.ErrChainBroken)
	assert.Equal(t, audit.VerifyTruncated, result.State)
}

func TestVerifyReportsKeyUnavailable(t *testing.T) {
	ctx := context.Background()
	store := &memStore{}
	// Write a keyed chain, then verify with no key (keyring locked at verify time).
	keyed := audit.New(store, fixedClock(), testKey, &memMark{})
	appendN(t, keyed, 3)

	unkeyed := audit.New(store, fixedClock(), nil, &memMark{})
	result, err := unkeyed.Verify(ctx)
	require.NoError(t, err, "key-unavailable is a state, not an error")
	assert.Equal(t, audit.VerifyKeyUnavailable, result.State)
	assert.Equal(t, 3, result.VerifiedCount, "structure still checks out")
}

func TestKeyedMACDiffersFromUnkeyed(t *testing.T) {
	entry := domain.AuditEntry{Seq: 1, At: time.Unix(0, 1).UTC(), Action: domain.ActionSystemPrune, Subject: "all"}
	keyed, err := audit.ComputeMAC(testKey, "", entry)
	require.NoError(t, err)
	unkeyed, err := audit.ComputeMAC(nil, "", entry)
	require.NoError(t, err)
	assert.NotEqual(t, keyed, unkeyed, "the key changes the MAC")

	// Deterministic and content-sensitive.
	again, err := audit.ComputeMAC(testKey, "", entry)
	require.NoError(t, err)
	assert.Equal(t, keyed, again)
	entry.Subject = "different"
	changed, err := audit.ComputeMAC(testKey, "", entry)
	require.NoError(t, err)
	assert.NotEqual(t, keyed, changed)
}
