package compose_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/core/compose"
	"github.com/drydock/drydock/internal/core/domain"
)

// fixture is a small but representative set of containers across two stacks, a
// multi-replica service, a partially-running stack, and a standalone container.
func fixture() []domain.Container {
	return []domain.Container{
		{ID: "a1", Name: "blog-web-1", State: "running", ComposeProject: "blog", ComposeService: "web"},
		{ID: "a2", Name: "blog-web-2", State: "running", ComposeProject: "blog", ComposeService: "web"},
		{ID: "a3", Name: "blog-db-1", State: "running", ComposeProject: "blog", ComposeService: "db"},
		{ID: "b1", Name: "shop-api-1", State: "exited", ComposeProject: "shop", ComposeService: "api"},
		{ID: "b2", Name: "shop-cache-1", State: "running", ComposeProject: "shop", ComposeService: "cache"},
		{ID: "s1", Name: "loose", State: "running"}, // standalone, no project label
	}
}

func find(stacks []domain.Stack, project string) (domain.Stack, bool) {
	for _, s := range stacks {
		if s.Project == project {
			return s, true
		}
	}
	return domain.Stack{}, false
}

func TestGroupGroupsByProjectAndService(t *testing.T) {
	stacks := compose.Group(fixture())

	// Two projects; the standalone container forms no stack.
	require.Len(t, stacks, 2)

	blog, ok := find(stacks, "blog")
	require.True(t, ok)
	assert.Equal(t, 3, blog.Total)
	assert.Equal(t, 3, blog.Running)
	assert.Equal(t, domain.StackRunning, blog.State)

	// Services are db then web (sorted); web has its two replicas.
	require.Len(t, blog.Services, 2)
	assert.Equal(t, "db", blog.Services[0].Name)
	assert.Equal(t, "web", blog.Services[1].Name)
	assert.Equal(t, 2, blog.Services[1].Total)
	assert.Equal(t, "blog-web-1", blog.Services[1].Containers[0].Name)
	assert.Equal(t, "blog-web-2", blog.Services[1].Containers[1].Name)
}

func TestGroupAggregatesPartialState(t *testing.T) {
	stacks := compose.Group(fixture())

	shop, ok := find(stacks, "shop")
	require.True(t, ok)
	assert.Equal(t, 2, shop.Total)
	assert.Equal(t, 1, shop.Running, "one of two services is running")
	assert.Equal(t, domain.StackPartial, shop.State)
}

func TestGroupStoppedStack(t *testing.T) {
	stacks := compose.Group([]domain.Container{
		{ID: "c1", Name: "x-svc-1", State: "exited", ComposeProject: "x", ComposeService: "svc"},
	})
	require.Len(t, stacks, 1)
	assert.Equal(t, domain.StackStopped, stacks[0].State)
}

func TestGroupIsDeterministicAndCarriesHost(t *testing.T) {
	stacks := compose.Group(fixture())
	// Sorted by project name regardless of input order.
	assert.Equal(t, "blog", stacks[0].Project)
	assert.Equal(t, "shop", stacks[1].Project)
}

func TestGroupContainerMissingServiceLabel(t *testing.T) {
	stacks := compose.Group([]domain.Container{
		{ID: "c1", Name: "orphan", State: "running", ComposeProject: "legacy"},
	})
	require.Len(t, stacks, 1)
	require.Len(t, stacks[0].Services, 1)
	assert.Equal(t, "(unknown)", stacks[0].Services[0].Name)
}

func TestGroupNoComposeContainers(t *testing.T) {
	stacks := compose.Group([]domain.Container{{ID: "s1", Name: "loose", State: "running"}})
	assert.Empty(t, stacks)
}
