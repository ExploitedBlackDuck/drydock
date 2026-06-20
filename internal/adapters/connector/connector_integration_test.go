//go:build integration

// SSH transport integration test (PROJECT-BOOK §7.10 P3 gate). It connects to a
// host over SSH, lists containers, disconnects, and asserts no goroutine (i.e.
// no tunnel) is leaked. It is skipped unless DRYDOCK_TEST_SSH_ENDPOINT names a
// reachable SSH host whose user can reach the Docker socket, e.g.:
//
//	DRYDOCK_TEST_SSH_ENDPOINT=ssh://deploy@host task test:integration

package connector_test

import (
	"context"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/adapters/connector"
	"github.com/drydock/drydock/internal/core/domain"
)

func endpointOrSkip(t *testing.T) string {
	t.Helper()
	ep := os.Getenv("DRYDOCK_TEST_SSH_ENDPOINT")
	if ep == "" {
		t.Skip("set DRYDOCK_TEST_SSH_ENDPOINT to run the SSH transport test")
	}
	return ep
}

func TestSSHConnectListDisconnect(t *testing.T) {
	ctx := context.Background()
	host := domain.Host{
		ID:        "test-ssh",
		Name:      "test-ssh",
		Transport: domain.TransportSSH,
		Endpoint:  endpointOrSkip(t),
	}

	eng, err := connector.New().Connect(ctx, host)
	require.NoError(t, err)

	info, err := eng.Info(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, info.EngineVersion)

	containers, err := eng.ListContainers(ctx)
	require.NoError(t, err)
	for _, c := range containers {
		assert.Equal(t, host.ID, c.HostRef)
	}

	require.NoError(t, eng.Close())
}

func TestSSHNoLeakedTunnel(t *testing.T) {
	ctx := context.Background()
	endpoint := endpointOrSkip(t)
	host := domain.Host{ID: "leak", Name: "leak", Transport: domain.TransportSSH, Endpoint: endpoint}

	// Warm up and establish a baseline.
	warm, err := connector.New().Connect(ctx, host)
	require.NoError(t, err)
	_, _ = warm.ListContainers(ctx)
	require.NoError(t, warm.Close())
	baseline := stableGoroutines()

	for i := 0; i < 5; i++ {
		eng, err := connector.New().Connect(ctx, host)
		require.NoError(t, err)
		_, err = eng.ListContainers(ctx)
		require.NoError(t, err)
		require.NoError(t, eng.Close())
	}

	after := stableGoroutines()
	assert.LessOrEqual(t, after, baseline+3,
		"goroutines grew from %d to %d — an SSH tunnel leaked after Close", baseline, after)
}

func stableGoroutines() int {
	prev := runtime.NumGoroutine()
	for i := 0; i < 20; i++ {
		time.Sleep(50 * time.Millisecond)
		runtime.GC()
		n := runtime.NumGoroutine()
		if n == prev {
			return n
		}
		prev = n
	}
	return prev
}
