package dockerengine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/drydock/drydock/internal/core/domain"
	"github.com/drydock/drydock/internal/core/engine"
)

// StartContainer starts a stopped container.
func (c *Client) StartContainer(ctx context.Context, id string) error {
	if err := c.cli.ContainerStart(ctx, id, container.StartOptions{}); err != nil {
		return fmt.Errorf("starting container %q: %w", id, err)
	}
	return nil
}

// StopContainer gracefully stops a running container.
func (c *Client) StopContainer(ctx context.Context, id string) error {
	if err := c.cli.ContainerStop(ctx, id, container.StopOptions{}); err != nil {
		return fmt.Errorf("stopping container %q: %w", id, err)
	}
	return nil
}

// RestartContainer restarts a container.
func (c *Client) RestartContainer(ctx context.Context, id string) error {
	if err := c.cli.ContainerRestart(ctx, id, container.StopOptions{}); err != nil {
		return fmt.Errorf("restarting container %q: %w", id, err)
	}
	return nil
}

// KillContainer sends SIGKILL to a container.
func (c *Client) KillContainer(ctx context.Context, id string) error {
	if err := c.cli.ContainerKill(ctx, id, "KILL"); err != nil {
		return fmt.Errorf("killing container %q: %w", id, err)
	}
	return nil
}

// RemoveContainer removes a container per opts.
func (c *Client) RemoveContainer(ctx context.Context, id string, opts engine.RemoveOptions) error {
	err := c.cli.ContainerRemove(ctx, id, container.RemoveOptions{
		Force:         opts.Force,
		RemoveVolumes: opts.Volumes,
	})
	if err != nil {
		return fmt.Errorf("removing container %q: %w", id, err)
	}
	return nil
}

// ContainerLogs returns a plain-text log stream. Multiplexed (non-TTY) output is
// demultiplexed into a single reader; closing it (or cancelling ctx) stops the
// underlying stream with no leaked goroutine.
func (c *Client) ContainerLogs(ctx context.Context, id string, opts engine.LogOptions) (io.ReadCloser, error) {
	tail := "all"
	if opts.Tail > 0 {
		tail = strconv.Itoa(opts.Tail)
	}

	tty := false
	if insp, err := c.cli.ContainerInspect(ctx, id); err == nil {
		tty = insp.Config.Tty
	}

	rc, err := c.cli.ContainerLogs(ctx, id, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     opts.Follow,
		Timestamps: opts.Timestamps,
		Tail:       tail,
	})
	if err != nil {
		return nil, fmt.Errorf("streaming logs for container %q: %w", id, err)
	}
	if tty {
		return rc, nil
	}

	pr, pw := io.Pipe()
	go func() {
		_, copyErr := stdcopy.StdCopy(pw, pw, rc)
		_ = pw.CloseWithError(copyErr)
	}()
	return &demuxReadCloser{pr: pr, src: rc}, nil
}

// demuxReadCloser exposes the demultiplexed log stream and tears down the source
// on Close.
type demuxReadCloser struct {
	pr  *io.PipeReader
	src io.Closer
}

func (d *demuxReadCloser) Read(p []byte) (int, error) { return d.pr.Read(p) }

func (d *demuxReadCloser) Close() error {
	err := d.src.Close()
	_ = d.pr.Close()
	return err
}

// StreamStats samples live stats into sink until ctx is cancelled.
func (c *Client) StreamStats(ctx context.Context, id string, sink func(domain.ResourceSample)) error {
	resp, err := c.cli.ContainerStats(ctx, id, true)
	if err != nil {
		return fmt.Errorf("streaming stats for container %q: %w", id, err)
	}
	defer func() { _ = resp.Body.Close() }()

	dec := json.NewDecoder(resp.Body)
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		var raw container.StatsResponse
		if err := dec.Decode(&raw); err != nil {
			if ctx.Err() != nil || err == io.EOF {
				return nil
			}
			return fmt.Errorf("decoding stats for container %q: %w", id, err)
		}
		sink(toSample(c.hostRef, id, raw))
	}
}

// Exec starts an argv command inside a container (never a shell — ADR-0004) and
// returns its bidirectional stream.
func (c *Client) Exec(ctx context.Context, id string, spec engine.ExecSpec) (engine.ExecStream, error) {
	created, err := c.cli.ContainerExecCreate(ctx, id, buildExecOptions(spec))
	if err != nil {
		return nil, fmt.Errorf("creating exec in container %q: %w", id, err)
	}
	attached, err := c.cli.ContainerExecAttach(ctx, created.ID, container.ExecAttachOptions{Tty: spec.Tty})
	if err != nil {
		return nil, fmt.Errorf("attaching exec in container %q: %w", id, err)
	}
	return &execStream{resp: attached, cli: c.cli, execID: created.ID}, nil
}

// execStream adapts the SDK's hijacked connection to engine.ExecStream: read
// container output from the reader, write stdin to the connection, resize the
// remote TTY by exec id, and tear the connection down on Close.
type execStream struct {
	resp   types.HijackedResponse
	cli    *client.Client
	execID string
}

func (e *execStream) Read(p []byte) (int, error)  { return e.resp.Reader.Read(p) }
func (e *execStream) Write(p []byte) (int, error) { return e.resp.Conn.Write(p) }
func (e *execStream) Close() error                { e.resp.Close(); return nil }

// CloseStdin half-closes the input side of the hijacked connection, sending EOF
// to the command while output keeps streaming back (ADR-0022).
func (e *execStream) CloseStdin() error {
	if err := e.resp.CloseWrite(); err != nil {
		return fmt.Errorf("half-closing exec %q stdin: %w", e.execID, err)
	}
	return nil
}

// Resize adjusts the remote pseudo-TTY so full-screen programs (vim, top) render
// to the terminal pane's real dimensions.
func (e *execStream) Resize(ctx context.Context, cols, rows uint16) error {
	if err := e.cli.ContainerExecResize(ctx, e.execID, container.ResizeOptions{
		Height: uint(rows),
		Width:  uint(cols),
	}); err != nil {
		return fmt.Errorf("resizing exec %q: %w", e.execID, err)
	}
	return nil
}

// buildExecOptions maps an ExecSpec to SDK exec options. The command is passed
// as argv; nothing is interpolated into a shell.
func buildExecOptions(spec engine.ExecSpec) container.ExecOptions {
	return container.ExecOptions{
		Cmd:          spec.Cmd,
		User:         spec.User,
		WorkingDir:   spec.WorkingDir,
		Env:          spec.Env,
		Tty:          spec.Tty,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
	}
}

// toSample computes a ResourceSample from one stats frame, using the embedded
// previous-CPU snapshot for the CPU-percent delta. It is pure and table-tested.
func toSample(hostRef, containerID string, s container.StatsResponse) domain.ResourceSample {
	sample := domain.ResourceSample{
		HostRef:     hostRef,
		ContainerID: containerID,
		At:          s.Read,
		MemBytes:    int64(s.MemoryStats.Usage), //nolint:gosec // usage fits int64
	}
	if sample.At.IsZero() {
		sample.At = time.Now().UTC()
	}

	cpuDelta := float64(s.CPUStats.CPUUsage.TotalUsage) - float64(s.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(s.CPUStats.SystemUsage) - float64(s.PreCPUStats.SystemUsage)
	if cpuDelta > 0 && systemDelta > 0 {
		cpus := float64(s.CPUStats.OnlineCPUs)
		if cpus == 0 {
			cpus = float64(len(s.CPUStats.CPUUsage.PercpuUsage))
		}
		if cpus == 0 {
			cpus = 1
		}
		sample.CPUPct = (cpuDelta / systemDelta) * cpus * 100
	}

	for _, n := range s.Networks {
		sample.NetRx += int64(n.RxBytes) //nolint:gosec // counters fit int64
		sample.NetTx += int64(n.TxBytes) //nolint:gosec // counters fit int64
	}
	for _, b := range s.BlkioStats.IoServiceBytesRecursive {
		switch b.Op {
		case "Read", "read":
			sample.BlkRead += int64(b.Value) //nolint:gosec // counters fit int64
		case "Write", "write":
			sample.BlkWrite += int64(b.Value) //nolint:gosec // counters fit int64
		}
	}
	return sample
}
