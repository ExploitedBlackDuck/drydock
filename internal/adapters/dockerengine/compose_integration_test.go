//go:build integration

package dockerengine_test

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/adapters/dockerengine"
	"github.com/drydock/drydock/internal/core/compose"
	"github.com/drydock/drydock/internal/core/domain"
)

// TestComposeDownRemovesTheStack synthesizes a two-service Compose stack (the
// project labels Compose stamps), confirms Drydock discovers and groups it, then
// takes it down as a unit and confirms the containers are gone — proving the
// label-scoped lifecycle against a real daemon (P7 gate, §7.11.6). It never
// starts the containers, so it needs no specific image behaviour.
func TestComposeDownRemovesTheStack(t *testing.T) {
	ctx := context.Background()
	eng := openOrSkip(t)
	defer func() { _ = eng.Close() }()

	raw, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	require.NoError(t, err)
	defer func() { _ = raw.Close() }()

	imageRef := firstImageRefOrSkip(t, eng)
	const project = "drydock-it-compose"

	created := []string{}
	defer func() {
		for _, id := range created {
			_ = raw.ContainerRemove(ctx, id, container.RemoveOptions{Force: true})
		}
	}()

	for _, svc := range []string{"web", "db"} {
		resp, cerr := raw.ContainerCreate(ctx, &container.Config{
			Image: imageRef,
			Cmd:   []string{"sleep", "3600"},
			Labels: map[string]string{
				"com.docker.compose.project": project,
				"com.docker.compose.service": svc,
			},
		}, nil, nil, nil, "")
		require.NoError(t, cerr)
		created = append(created, resp.ID)
	}

	// Drydock discovers and groups the stack.
	containers, err := eng.ListContainers(ctx)
	require.NoError(t, err)
	stacks := compose.Group(containers)
	stack, ok := findStack(stacks, project)
	require.True(t, ok, "synthesized stack not discovered")
	assert.Equal(t, 2, stack.Total)
	assert.Len(t, stack.Services, 2)

	// Take it down as a unit; the containers are removed.
	require.NoError(t, eng.ComposeDown(ctx, project, false))

	containers, err = eng.ListContainers(ctx)
	require.NoError(t, err)
	_, stillThere := findStack(compose.Group(containers), project)
	assert.False(t, stillThere, "ComposeDown left the stack's containers behind")
}

func findStack(stacks []domain.Stack, project string) (domain.Stack, bool) {
	for _, s := range stacks {
		if s.Project == project {
			return s, true
		}
	}
	return domain.Stack{}, false
}

// firstImageRefOrSkip returns an image reference present on the daemon to base
// the throwaway containers on, skipping when the daemon holds no usable image.
func firstImageRefOrSkip(t *testing.T, eng *dockerengine.Client) string {
	t.Helper()
	images, err := eng.ListImages(context.Background())
	require.NoError(t, err)
	for _, img := range images {
		if img.Repo != "" && img.Repo != "<none>" && img.Tag != "" && img.Tag != "<none>" {
			return img.Repo + ":" + img.Tag
		}
	}
	t.Skip("no tagged image on the daemon to synthesize a stack from")
	return ""
}
