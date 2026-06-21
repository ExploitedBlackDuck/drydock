package timeline_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/core/domain"
	"github.com/drydock/drydock/internal/core/timeline"
)

func intp(n int) *int { return &n }

func TestFromEventKeepsLocalDropsSwarm(t *testing.T) {
	local := domain.EngineEvent{
		Action: domain.EventActionDie, ContainerName: "web", Scope: "local",
		ExitCode: intp(137), At: time.Unix(100, 0),
	}
	entry, ok := timeline.FromEvent("h", local)
	require.True(t, ok)
	assert.Equal(t, domain.TimelineEngine, entry.Source)
	assert.Equal(t, "die", entry.Kind)
	assert.Equal(t, "web", entry.Subject)
	require.NotNil(t, entry.ExitCode)
	assert.Equal(t, 137, *entry.ExitCode)

	_, ok = timeline.FromEvent("h", domain.EngineEvent{Action: "update", Scope: "swarm"})
	assert.False(t, ok, "swarm-scope events are filtered out")
}

func TestMergeInterleavesByTimeAndKeepsClocks(t *testing.T) {
	engine := []domain.TimelineEntry{
		{HostRef: "h", At: time.Unix(10, 0), Source: domain.TimelineEngine, Kind: "die"},
		{HostRef: "h", At: time.Unix(30, 0), Source: domain.TimelineEngine, Kind: "start"},
	}
	audit := []domain.AuditEntry{
		{HostRef: "h", At: time.Unix(20, 0), Action: domain.ActionContainerStop, Subject: "c1"},
	}

	merged := timeline.Merge(engine, audit)
	require.Len(t, merged, 3)
	// Interleaved by timestamp: engine(10), audit(20), engine(30).
	assert.Equal(t, domain.TimelineEngine, merged[0].Source)
	assert.Equal(t, domain.TimelineAudit, merged[1].Source)
	assert.Equal(t, "container.stop", merged[1].Kind)
	assert.Equal(t, domain.TimelineEngine, merged[2].Source)
	// Audit timestamp is preserved exactly (its desktop clock), not adjusted.
	assert.True(t, merged[1].At.Equal(time.Unix(20, 0)))
}

func TestSkewIsSurfacedNotHidden(t *testing.T) {
	host := time.Unix(1000, 0)
	desktop := time.Unix(1042, 0) // desktop is 42s ahead of the host clock
	assert.Equal(t, 42*time.Second, timeline.Skew(host, desktop))

	// Merge does not rewrite engine times to the desktop clock — the skew remains
	// visible in the data rather than being silently corrected away.
	engine := []domain.TimelineEntry{{At: host, Source: domain.TimelineEngine}}
	merged := timeline.Merge(engine, nil)
	assert.True(t, merged[0].At.Equal(host))
}
