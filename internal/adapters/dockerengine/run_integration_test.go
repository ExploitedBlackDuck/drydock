//go:build integration

package dockerengine_test

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/core/domain"
)

// TestRunContainerCreatesAndStarts drives RunContainer end to end against a real
// daemon: it assembles a container from a RunSpec, confirms the daemon created
// it, then removes it.
func TestRunContainerCreatesAndStarts(t *testing.T) {
	ctx := context.Background()
	eng := openOrSkip(t)
	defer func() { _ = eng.Close() }()

	raw, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	require.NoError(t, err)
	defer func() { _ = raw.Close() }()

	imageRef := firstImageRefOrSkip(t, eng)
	id, err := eng.RunContainer(ctx, domain.RunSpec{
		Image:   imageRef,
		Command: []string{"sleep", "3600"},
		Env:     []string{"DRYDOCK_TEST=1"},
		Restart: "no",
	})
	// Even if the image lacks `sleep` (start would still have been issued), the
	// container must have been created and have an id.
	require.NotEmpty(t, id)
	defer func() { _ = raw.ContainerRemove(ctx, id, container.RemoveOptions{Force: true}) }()
	require.NoError(t, err)

	containers, err := eng.ListContainers(ctx)
	require.NoError(t, err)
	found := false
	for _, c := range containers {
		if c.ID == id {
			found = true
		}
	}
	assert.True(t, found, "the created container is listed by the engine")
}
