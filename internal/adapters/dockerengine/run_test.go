package dockerengine

import (
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/core/domain"
)

func TestBuildRunConfigAssemblesTypedParams(t *testing.T) {
	config, hostConfig, err := buildRunConfig(domain.RunSpec{
		Image:       "nginx:1.27",
		Env:         []string{"A=1", "B=2"},
		Publish:     []string{"127.0.0.1:8080:80/tcp"},
		Volumes:     []string{"web-data:/usr/share/nginx/html:ro"},
		Restart:     "unless-stopped",
		NetworkHost: false,
		User:        "1000",
		WorkingDir:  "/app",
	})
	require.NoError(t, err)

	assert.Equal(t, "nginx:1.27", config.Image)
	assert.Equal(t, []string{"A=1", "B=2"}, config.Env)
	assert.Equal(t, "1000", config.User)
	assert.Equal(t, "/app", config.WorkingDir)
	assert.Equal(t, []string{"web-data:/usr/share/nginx/html:ro"}, hostConfig.Binds)
	assert.Equal(t, container.RestartPolicyMode("unless-stopped"), hostConfig.RestartPolicy.Name)

	require.Contains(t, hostConfig.PortBindings, nat.Port("80/tcp"))
	binding := hostConfig.PortBindings[nat.Port("80/tcp")]
	require.Len(t, binding, 1)
	assert.Equal(t, "127.0.0.1", binding[0].HostIP)
	assert.Equal(t, "8080", binding[0].HostPort)
}

func TestBuildRunConfigHostNetwork(t *testing.T) {
	_, hostConfig, err := buildRunConfig(domain.RunSpec{Image: "busybox", NetworkHost: true})
	require.NoError(t, err)
	assert.Equal(t, container.NetworkMode("host"), hostConfig.NetworkMode)
	assert.Empty(t, string(hostConfig.RestartPolicy.Name), "no restart policy unless requested")
}

func TestBuildRunConfigRejectsBadPort(t *testing.T) {
	_, _, err := buildRunConfig(domain.RunSpec{Image: "x", Publish: []string{"not-a-port"}})
	assert.Error(t, err)
}
