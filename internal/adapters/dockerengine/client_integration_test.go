//go:build integration

// These tests need a real Docker (or API-compatible) daemon and are gated
// behind the integration build tag (PROJECT-BOOK §2.5). They skip cleanly when
// no daemon is reachable, so the default `go test ./...` stays green on a bare
// machine. Run with `task test:integration`.

package dockerengine_test

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/adapters/dockerengine"
)

// openOrSkip connects to the local daemon, skipping the test when unavailable.
func openOrSkip(t *testing.T) *dockerengine.Client {
	t.Helper()
	client, err := dockerengine.Open("local")
	require.NoError(t, err)
	if _, err := client.Info(context.Background()); err != nil {
		_ = client.Close()
		t.Skipf("no reachable Docker daemon: %v", err)
	}
	return client
}

func TestListAgainstRealDaemon(t *testing.T) {
	ctx := context.Background()
	client := openOrSkip(t)
	defer func() { _ = client.Close() }()

	info, err := client.Info(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, info.EngineVersion)
	assert.NotEmpty(t, info.APIVersion)

	// Each listing must succeed; emptiness is environment-dependent, so we only
	// assert no error and that host stamping happened where rows exist.
	containers, err := client.ListContainers(ctx)
	require.NoError(t, err)
	for _, c := range containers {
		assert.Equal(t, "local", c.HostRef)
	}

	_, err = client.ListImages(ctx)
	require.NoError(t, err)
	_, err = client.ListVolumes(ctx)
	require.NoError(t, err)
	_, err = client.ListNetworks(ctx)
	require.NoError(t, err)
}

// TestNoLeakedConnectionsAfterClose verifies the supervised-connection rule
// (PROJECT-BOOK §2.3/§7.2): after using and closing an engine, goroutines
// return to their baseline — no background readers or idle connections linger.
func TestNoLeakedConnectionsAfterClose(t *testing.T) {
	ctx := context.Background()

	// Establish a baseline after the runtime has warmed up.
	probe := openOrSkip(t)
	_, _ = probe.ListContainers(ctx)
	_ = probe.Close()
	baseline := stableGoroutines()

	for i := 0; i < 10; i++ {
		client, err := dockerengine.Open("local")
		require.NoError(t, err)
		_, err = client.ListContainers(ctx)
		require.NoError(t, err)
		_, err = client.ListNetworks(ctx)
		require.NoError(t, err)
		require.NoError(t, client.Close())
	}

	after := stableGoroutines()
	// Allow small scheduler jitter, but not growth proportional to the 10 opens.
	assert.LessOrEqual(t, after, baseline+3,
		"goroutines grew from %d to %d — a connection or reader leaked", baseline, after)
}

// stableGoroutines waits for the goroutine count to settle and returns it.
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
