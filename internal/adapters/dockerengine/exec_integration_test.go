//go:build integration

package dockerengine_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/drydock/drydock/internal/core/engine"
)

// TestExecInteractiveReadWriteResize drives the interactive exec path end to end
// against a real daemon (P4's deferred terminal, §7.11.4): it allocates a TTY,
// writes to the session, reads the echoed bytes back, resizes the TTY, and
// closes — leaving no leaked goroutine. The command is argv (`cat`), never a
// shell string.
func TestExecInteractiveReadWriteResize(t *testing.T) {
	ctx := context.Background()
	eng := openOrSkip(t)
	defer func() { _ = eng.Close() }()

	raw, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	require.NoError(t, err)
	defer func() { _ = raw.Close() }()

	imageRef := firstImageRefOrSkip(t, eng)
	created, err := raw.ContainerCreate(ctx, &container.Config{
		Image: imageRef,
		Cmd:   []string{"sleep", "3600"},
	}, nil, nil, nil, "")
	require.NoError(t, err)
	id := created.ID
	defer func() { _ = raw.ContainerRemove(ctx, id, container.RemoveOptions{Force: true}) }()
	require.NoError(t, raw.ContainerStart(ctx, id, container.StartOptions{}))

	baseline := stableGoroutines()

	stream, err := eng.Exec(ctx, id, engine.ExecSpec{Cmd: []string{"cat"}, Tty: true})
	require.NoError(t, err)

	// `cat` over a TTY echoes what we send back to us.
	_, err = stream.Write([]byte("drydock-ping\n"))
	require.NoError(t, err)
	assert.Contains(t, readUntil(stream, "drydock-ping", 5*time.Second), "drydock-ping")

	// Resizing the live TTY succeeds.
	require.NoError(t, stream.Resize(ctx, 100, 40))

	// Half-closing stdin sends EOF to the command without erroring the call.
	require.NoError(t, stream.CloseStdin())

	require.NoError(t, stream.Close())

	after := stableGoroutines()
	assert.LessOrEqual(t, after, baseline+3,
		"goroutines grew from %d to %d — an exec stream leaked", baseline, after)
}

// readUntil reads from stream until the accumulated output contains want or the
// timeout elapses, returning whatever was read. The read goroutine ends when the
// stream is closed by the caller.
func readUntil(stream engine.ExecStream, want string, timeout time.Duration) string {
	type chunk struct {
		data string
		err  error
	}
	ch := make(chan chunk, 1)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stream.Read(buf)
			if n > 0 {
				ch <- chunk{data: string(buf[:n])}
			}
			if err != nil {
				ch <- chunk{err: err}
				return
			}
		}
	}()

	var got strings.Builder
	deadline := time.After(timeout)
	for {
		select {
		case c := <-ch:
			if c.err != nil {
				return got.String()
			}
			got.WriteString(c.data)
			if strings.Contains(got.String(), want) {
				return got.String()
			}
		case <-deadline:
			return got.String()
		}
	}
}
