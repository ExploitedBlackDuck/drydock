package dockerengine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/docker/api/types/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/core/domain"
)

func TestMapEventFromFixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "api", "events.json"))
	require.NoError(t, err)
	var msgs []events.Message
	require.NoError(t, json.Unmarshal(data, &msgs))
	require.Len(t, msgs, 2)

	die := mapEvent(msgs[0])
	assert.Equal(t, domain.EventTypeContainer, die.Type)
	assert.Equal(t, domain.EventActionDie, die.Action)
	assert.Equal(t, "3f1a9c2b7e4d", die.ContainerID)
	assert.Equal(t, "web", die.ContainerName)
	assert.Equal(t, time.Unix(0, 1718800000000000000).UTC(), die.At)

	// Compound action is reduced to its base.
	health := mapEvent(msgs[1])
	assert.Equal(t, domain.EventActionHealth, health.Action)
}
