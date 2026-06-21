package sqlitestore

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/core/audit"
	"github.com/drydock/drydock/internal/core/domain"
)

func TestTimelineRoundTripAndRetention(t *testing.T) {
	ctx := context.Background()
	store := tempStore(t)

	base := time.Unix(0, 1_700_000_000_000_000_000).UTC()
	code := 137
	for i := 0; i < 4; i++ {
		require.NoError(t, store.SaveTimelineEntry(ctx, domain.TimelineEntry{
			HostRef: "local", At: base.Add(time.Duration(i) * time.Minute),
			Source: domain.TimelineEngine, Kind: "die", Subject: "web", ExitCode: &code,
		}))
	}

	got, err := store.RecentTimelineEntries(ctx, "local", 10)
	require.NoError(t, err)
	require.Len(t, got, 4)
	assert.True(t, got[0].At.Before(got[3].At), "oldest first")
	require.NotNil(t, got[0].ExitCode)
	assert.Equal(t, 137, *got[0].ExitCode)

	pruned, err := store.PruneTimelineEntries(ctx, base.Add(2*time.Minute))
	require.NoError(t, err)
	assert.Equal(t, int64(2), pruned)
}

// TestTimelineNeverEntersAuditChain proves the separateness guarantee (ADR-0018):
// writing engine timeline entries does not touch the audit log, and the audit
// chain still verifies.
func TestTimelineNeverEntersAuditChain(t *testing.T) {
	ctx := context.Background()
	store := tempStore(t)
	key := []byte("drydock-store-test-audit-key-0123")
	log := audit.New(store, nil, key, nil)

	_, err := log.Append(ctx, audit.Record{Action: domain.ActionContainerStop, Subject: "c1"})
	require.NoError(t, err)

	// A flood of engine events into the timeline.
	for i := 0; i < 5; i++ {
		require.NoError(t, store.SaveTimelineEntry(ctx, domain.TimelineEntry{
			HostRef: "local", At: time.Now().UTC(), Source: domain.TimelineEngine, Kind: "die",
		}))
	}

	// The audit log is untouched: still exactly one entry, chain intact.
	entries, err := store.AuditEntries(ctx)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "engine events do not become audit entries")
	result, err := log.Verify(ctx)
	require.NoError(t, err)
	assert.Equal(t, audit.VerifyIntact, result.State)
}
