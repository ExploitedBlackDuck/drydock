//go:build integration

package dockerengine_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/adapters/dockerengine"
	"github.com/drydock/drydock/internal/core/domain"
	"github.com/drydock/drydock/internal/core/engine"
)

// runningContainerOrSkip returns the id of a running container, or skips.
func runningContainerOrSkip(t *testing.T, client *dockerengine.Client) string {
	t.Helper()
	containers, err := client.ListContainers(context.Background())
	require.NoError(t, err)
	for _, c := range containers {
		if c.State == "running" {
			return c.ID
		}
	}
	t.Skip("no running container to stream from")
	return ""
}

// TestLogStreamCancelNoLeak verifies cancellation propagates to the log stream
// and leaves no goroutine behind (PROJECT-BOOK §2.3 / P4 gate).
func TestLogStreamCancelNoLeak(t *testing.T) {
	client := openOrSkip(t)
	defer func() { _ = client.Close() }()
	id := runningContainerOrSkip(t, client)

	// Warm up, then take a baseline.
	{
		ctx, cancel := context.WithCancel(context.Background())
		rc, err := client.ContainerLogs(ctx, id, engine.LogOptions{Follow: true, Tail: 5})
		require.NoError(t, err)
		buf := make([]byte, 256)
		_, _ = rc.Read(buf)
		cancel()
		_ = rc.Close()
	}
	baseline := stableGoroutines()

	for i := 0; i < 5; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		rc, err := client.ContainerLogs(ctx, id, engine.LogOptions{Follow: true, Tail: 5})
		require.NoError(t, err)

		// Cancelling the context must make the stream end (Read returns).
		cancel()
		done := make(chan struct{})
		go func() {
			buf := make([]byte, 4096)
			for {
				if _, err := rc.Read(buf); err != nil {
					break
				}
			}
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("log stream did not end after context cancellation")
		}
		_ = rc.Close()
	}

	after := stableGoroutines()
	assert.LessOrEqual(t, after, baseline+3,
		"goroutines grew from %d to %d — a log stream leaked", baseline, after)
}

// TestStatsStreamCancelReturns verifies StreamStats returns promptly when its
// context is cancelled (no leaked goroutine).
func TestStatsStreamCancelReturns(t *testing.T) {
	client := openOrSkip(t)
	defer func() { _ = client.Close() }()
	id := runningContainerOrSkip(t, client)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- client.StreamStats(ctx, id, func(domain.ResourceSample) {})
	}()

	time.Sleep(500 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		assert.True(t, err == nil || err == context.Canceled || err == io.EOF,
			"clean return on cancel, got %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("StreamStats did not return after cancellation")
	}
}
