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

func fixedClock() audit.Clock {
	t := time.Unix(0, 1_700_000_000_000_000_000).UTC()
	return func() time.Time { return t }
}

func TestAppendAssignsSequenceAndChains(t *testing.T) {
	ctx := context.Background()
	log := audit.New(&memStore{}, fixedClock())

	first, err := log.Append(ctx, audit.Record{Action: domain.ActionHostConnect, HostRef: "h1", Subject: "alpha"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), first.Seq)
	assert.Empty(t, first.PrevHash, "first entry has no predecessor")
	assert.NotEmpty(t, first.Hash)

	second, err := log.Append(ctx, audit.Record{Action: domain.ActionContainerStop, HostRef: "h1", Subject: "beta"})
	require.NoError(t, err)
	assert.Equal(t, int64(2), second.Seq)
	assert.Equal(t, first.Hash, second.PrevHash, "each entry links to the previous hash")
}

func TestVerifyAcceptsIntactChain(t *testing.T) {
	ctx := context.Background()
	store := &memStore{}
	log := audit.New(store, fixedClock())

	for i := 0; i < 5; i++ {
		_, err := log.Append(ctx, audit.Record{Action: domain.ActionContainerStart, Subject: "c"})
		require.NoError(t, err)
	}

	count, err := log.Verify(ctx)
	require.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestVerifyDetectsTamperedContent(t *testing.T) {
	ctx := context.Background()
	store := &memStore{}
	log := audit.New(store, fixedClock())

	for i := 0; i < 3; i++ {
		_, err := log.Append(ctx, audit.Record{Action: domain.ActionContainerStart, Subject: "c"})
		require.NoError(t, err)
	}

	// Mutate a recorded entry's content without recomputing its hash.
	store.entries[1].Subject = "tampered"

	_, err := log.Verify(ctx)
	assert.ErrorIs(t, err, audit.ErrChainBroken)
}

func TestVerifyDetectsBrokenBackLink(t *testing.T) {
	ctx := context.Background()
	store := &memStore{}
	log := audit.New(store, fixedClock())

	for i := 0; i < 3; i++ {
		_, err := log.Append(ctx, audit.Record{Action: domain.ActionContainerStart, Subject: "c"})
		require.NoError(t, err)
	}

	// Removing a middle entry breaks both sequence contiguity and the back-link.
	store.entries = append(store.entries[:1], store.entries[2:]...)

	_, err := log.Verify(ctx)
	assert.ErrorIs(t, err, audit.ErrChainBroken)
}

func TestComputeHashIsDeterministicAndContentSensitive(t *testing.T) {
	entry := domain.AuditEntry{
		Seq:     1,
		At:      time.Unix(0, 1).UTC(),
		Action:  domain.ActionSystemPrune,
		HostRef: "h1",
		Subject: "all",
		Detail:  map[string]any{"reclaimed": 1024, "volumes": false},
	}

	h1, err := audit.ComputeHash("", entry)
	require.NoError(t, err)
	h2, err := audit.ComputeHash("", entry)
	require.NoError(t, err)
	assert.Equal(t, h1, h2, "same input yields same hash")

	entry.Subject = "different"
	h3, err := audit.ComputeHash("", entry)
	require.NoError(t, err)
	assert.NotEqual(t, h1, h3, "changed content yields a different hash")
}
