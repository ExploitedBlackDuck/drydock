package dockerengine

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/pkg/stdcopy"
)

// snapshotMount is where the volume is mounted inside the helper container.
const snapshotMount = "/drydock-volume"

// SnapshotVolume streams a read-only tar of the volume to dest via a throwaway
// helper container (ADR-0020). The helper runs argv (no shell) and is always
// removed.
func (c *Client) SnapshotVolume(ctx context.Context, volume, helperImage, dest string) (int64, error) {
	if err := c.ensureHelperImage(ctx, helperImage); err != nil {
		return 0, err
	}

	created, err := c.cli.ContainerCreate(ctx, &container.Config{
		Image:        helperImage,
		Cmd:          []string{"tar", "-cf", "-", "-C", snapshotMount, "."},
		AttachStdout: true,
		AttachStderr: true,
	}, &container.HostConfig{
		AutoRemove: false,
		Mounts: []mount.Mount{{
			Type: mount.TypeVolume, Source: volume, Target: snapshotMount, ReadOnly: true,
		}},
	}, nil, nil, "")
	if err != nil {
		return 0, fmt.Errorf("creating snapshot helper for volume %q: %w", volume, err)
	}
	defer func() {
		_ = c.cli.ContainerRemove(context.WithoutCancel(ctx), created.ID, container.RemoveOptions{Force: true})
	}()

	attached, err := c.cli.ContainerAttach(ctx, created.ID, container.AttachOptions{
		Stream: true, Stdout: true, Stderr: true,
	})
	if err != nil {
		return 0, fmt.Errorf("attaching snapshot helper: %w", err)
	}
	defer attached.Close()

	if err := c.cli.ContainerStart(ctx, created.ID, container.StartOptions{}); err != nil {
		return 0, fmt.Errorf("starting snapshot helper: %w", err)
	}

	out, err := os.Create(dest) //nolint:gosec // G304: dest is the snapshot file the operator chose to write (ADR-0020)
	if err != nil {
		return 0, fmt.Errorf("creating snapshot file %q: %w", dest, err)
	}
	defer func() { _ = out.Close() }()

	counter := &countingWriter{w: out}
	// Demultiplex the attach stream: stdout is the tar, stderr is discarded.
	if _, err := stdcopy.StdCopy(counter, io.Discard, attached.Reader); err != nil {
		return counter.n, fmt.Errorf("streaming snapshot of volume %q: %w", volume, err)
	}
	return counter.n, nil
}

// RestoreVolume extracts a tar from src into the volume via a throwaway helper.
func (c *Client) RestoreVolume(ctx context.Context, volume, helperImage, src string) error {
	if err := c.ensureHelperImage(ctx, helperImage); err != nil {
		return err
	}

	created, err := c.cli.ContainerCreate(ctx, &container.Config{
		Image:       helperImage,
		Cmd:         []string{"tar", "-xf", "-", "-C", snapshotMount},
		AttachStdin: true,
		OpenStdin:   true,
		StdinOnce:   true,
	}, &container.HostConfig{
		Mounts: []mount.Mount{{
			Type: mount.TypeVolume, Source: volume, Target: snapshotMount, ReadOnly: false,
		}},
	}, nil, nil, "")
	if err != nil {
		return fmt.Errorf("creating restore helper for volume %q: %w", volume, err)
	}
	defer func() {
		_ = c.cli.ContainerRemove(context.WithoutCancel(ctx), created.ID, container.RemoveOptions{Force: true})
	}()

	attached, err := c.cli.ContainerAttach(ctx, created.ID, container.AttachOptions{Stream: true, Stdin: true})
	if err != nil {
		return fmt.Errorf("attaching restore helper: %w", err)
	}
	defer attached.Close()

	if err := c.cli.ContainerStart(ctx, created.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("starting restore helper: %w", err)
	}

	in, err := os.Open(src) //nolint:gosec // G304: src is the snapshot file the operator chose to restore (ADR-0020)
	if err != nil {
		return fmt.Errorf("opening snapshot file %q: %w", src, err)
	}
	defer func() { _ = in.Close() }()

	if _, err := io.Copy(attached.Conn, in); err != nil {
		return fmt.Errorf("streaming restore into volume %q: %w", volume, err)
	}
	_ = attached.CloseWrite()
	return nil
}

// ensureHelperImage uses the helper image if it is already present, otherwise
// pulls the digest-pinned image. A pull failure (e.g. an air-gapped host) fails
// closed with a clear error rather than silently doing nothing (ADR-0020).
func (c *Client) ensureHelperImage(ctx context.Context, helperImage string) error {
	images, err := c.cli.ImageList(ctx, image.ListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", helperImage)),
	})
	if err == nil && len(images) > 0 {
		return nil
	}
	pulled, err := c.cli.ImagePull(ctx, helperImage, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("pulling snapshot helper %q (host may be air-gapped): %w", helperImage, err)
	}
	defer func() { _ = pulled.Close() }()
	if _, err := io.Copy(io.Discard, pulled); err != nil {
		return fmt.Errorf("pulling snapshot helper %q: %w", helperImage, err)
	}
	return nil
}

// countingWriter counts bytes written through it.
type countingWriter struct {
	w io.Writer
	n int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}
