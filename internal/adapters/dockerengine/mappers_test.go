package dockerengine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/core/domain"
)

const hostRef = "host-1"

func loadFixture(t *testing.T, name string, into any) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "api", name))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, into))
}

func TestMapContainersFromFixture(t *testing.T) {
	var summaries []container.Summary
	loadFixture(t, "containers_list.json", &summaries)
	require.Len(t, summaries, 3)

	got := make([]domain.Container, 0, len(summaries))
	for _, s := range summaries {
		got = append(got, mapContainer(hostRef, s))
	}

	// Running, compose-labelled, port-mapped container.
	web := got[0]
	assert.Equal(t, "web", web.Name, "leading slash stripped from name")
	assert.Equal(t, hostRef, web.HostRef)
	assert.Equal(t, "running", web.State)
	assert.Equal(t, "shop", web.ComposeProject)
	assert.Equal(t, time.Unix(1718800000, 0).UTC(), web.Created)
	require.Len(t, web.Ports, 2)
	assert.Equal(t, domain.Port{IP: "0.0.0.0", PrivatePort: 80, PublicPort: 8080, Protocol: "tcp"}, web.Ports[0])
	assert.Equal(t, uint16(443), web.Ports[1].PrivatePort)

	// Stopped container.
	assert.Equal(t, "exited", got[1].State)

	// Container with no compose label.
	assert.Empty(t, got[2].ComposeProject)
	assert.Empty(t, got[2].Ports)
}

func TestMapImagesFromFixture(t *testing.T) {
	var summaries []image.Summary
	loadFixture(t, "images_list.json", &summaries)
	require.Len(t, summaries, 3)

	tagged := mapImage(hostRef, summaries[0])
	assert.Equal(t, "registry.example.com/team/web", tagged.Repo)
	assert.Equal(t, "1.4.2", tagged.Tag)
	assert.False(t, tagged.Dangling)
	assert.True(t, tagged.InUse, "Containers > 0 means in use")
	assert.Equal(t, int64(182452345), tagged.Size)

	dangling := mapImage(hostRef, summaries[1])
	assert.True(t, dangling.Dangling)
	assert.Equal(t, "<none>", dangling.Repo)
	assert.False(t, dangling.InUse, "Containers == -1 is not in use")

	// Registry with a port must not be split as repo:tag on the port colon.
	withPort := mapImage(hostRef, summaries[2])
	assert.Equal(t, "registry.example.com:5000/team/cache", withPort.Repo)
	assert.Equal(t, "edge", withPort.Tag)
	assert.False(t, withPort.InUse, "Containers == 0 is not in use")
}

func TestMapVolumesFromFixture(t *testing.T) {
	var resp volume.ListResponse
	loadFixture(t, "volumes_list.json", &resp)
	require.Len(t, resp.Volumes, 3)

	inUse := mapVolume(hostRef, resp.Volumes[0])
	assert.Equal(t, "shop_db-data", inUse.Name)
	assert.Equal(t, int64(104857600), inUse.Size)
	assert.True(t, inUse.InUse)

	idle := mapVolume(hostRef, resp.Volumes[1])
	assert.False(t, idle.InUse)
	assert.Equal(t, int64(0), idle.Size)

	// No usage data reported -> size unknown (-1), not in use.
	noUsage := mapVolume(hostRef, resp.Volumes[2])
	assert.Equal(t, int64(-1), noUsage.Size)
	assert.False(t, noUsage.InUse)
}

func TestMapNetworksFromFixture(t *testing.T) {
	var summaries []network.Summary
	loadFixture(t, "networks_list.json", &summaries)
	require.Len(t, summaries, 3)

	bridge := mapNetwork(hostRef, summaries[0])
	assert.Equal(t, "bridge", bridge.Name)
	assert.Equal(t, "bridge", bridge.Driver)
	assert.False(t, bridge.InUse)

	withContainers := mapNetwork(hostRef, summaries[2])
	assert.Equal(t, "shop_default", withContainers.Name)
	assert.True(t, withContainers.InUse, "a network with attached containers is in use")
}

func TestSplitRepoTag(t *testing.T) {
	tests := []struct {
		name      string
		repoTags  []string
		wantRepo  string
		wantTag   string
		wantDangl bool
	}{
		{"simple", []string{"example/app:1.0.0"}, "example/app", "1.0.0", false},
		{"no tag", []string{"example/app"}, "example/app", "latest", false},
		{"registry port", []string{"registry.example.com:5000/app:edge"}, "registry.example.com:5000/app", "edge", false},
		{"registry port no tag", []string{"registry.example.com:5000/app"}, "registry.example.com:5000/app", "latest", false},
		{"dangling", []string{"<none>:<none>"}, "<none>", "<none>", true},
		{"empty", nil, "<none>", "<none>", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, tag, dangling := splitRepoTag(tt.repoTags)
			assert.Equal(t, tt.wantRepo, repo)
			assert.Equal(t, tt.wantTag, tag)
			assert.Equal(t, tt.wantDangl, dangling)
		})
	}
}
