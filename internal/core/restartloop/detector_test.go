package restartloop_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/core/domain"
	"github.com/drydock/drydock/internal/core/restartloop"
)

func loadEvents(t *testing.T) []domain.EngineEvent {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "crashloop_events.json"))
	require.NoError(t, err)
	var events []domain.EngineEvent
	require.NoError(t, json.Unmarshal(data, &events))
	return events
}

func TestDetectsCrashLoopFromEventFixture(t *testing.T) {
	detector := restartloop.New(3, 60*time.Second)

	alerts := map[string]restartloop.Alert{}
	for _, e := range loadEvents(t) {
		if alert, looping := detector.Observe(e); looping {
			alerts[alert.ContainerID] = alert
		}
	}

	// web died 3x within the window -> flagged; db died once -> not flagged.
	require.Contains(t, alerts, "c-web")
	assert.Equal(t, "web", alerts["c-web"].ContainerName)
	assert.GreaterOrEqual(t, alerts["c-web"].Deaths, 3)
	assert.NotContains(t, alerts, "c-db")
	assert.NotContains(t, alerts, "c-worker")
}

func TestDeathsOutsideWindowDoNotLoop(t *testing.T) {
	detector := restartloop.New(3, 30*time.Second)
	base := time.Unix(1_700_000_000, 0).UTC()

	var fired bool
	for i := 0; i < 4; i++ {
		// One death per minute — never 3 within the 30s window.
		_, looping := detector.Observe(domain.EngineEvent{
			Type: domain.EventTypeContainer, Action: domain.EventActionDie,
			ContainerID: "c", At: base.Add(time.Duration(i) * time.Minute),
		})
		fired = fired || looping
	}
	assert.False(t, fired, "spaced-out deaths are not a restart loop")
}

func TestDestroyClearsHistory(t *testing.T) {
	detector := restartloop.New(2, time.Minute)
	base := time.Unix(1_700_000_000, 0).UTC()

	_, _ = detector.Observe(domain.EngineEvent{Type: domain.EventTypeContainer, Action: domain.EventActionDie, ContainerID: "c", At: base})
	_, _ = detector.Observe(domain.EngineEvent{Type: domain.EventTypeContainer, Action: domain.EventActionDestroy, ContainerID: "c", At: base.Add(time.Second)})

	_, looping := detector.Observe(domain.EngineEvent{Type: domain.EventTypeContainer, Action: domain.EventActionDie, ContainerID: "c", At: base.Add(2 * time.Second)})
	assert.False(t, looping, "history reset on destroy; a single later death is not a loop")
}

func TestNonContainerEventsIgnored(t *testing.T) {
	detector := restartloop.New(1, time.Minute)
	_, looping := detector.Observe(domain.EngineEvent{Type: "image", Action: "delete", ContainerID: ""})
	assert.False(t, looping)
}
