//go:build integration

package dockerengine_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSnapshotAndRestoreVolume drives the volume snapshot/restore plumbing
// against a real daemon: it populates a volume, snapshots it to a tar via the
// read-only helper, and restores the tar into a fresh volume — exercising the
// helper container's create/attach/stream/cleanup path.
func TestSnapshotAndRestoreVolume(t *testing.T) {
	ctx := context.Background()
	eng := openOrSkip(t)
	defer func() { _ = eng.Close() }()

	raw, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	require.NoError(t, err)
	defer func() { _ = raw.Close() }()

	imageRef := firstImageRefOrSkip(t, eng)

	src, err := raw.VolumeCreate(ctx, volume.CreateOptions{})
	require.NoError(t, err)
	defer func() { _ = raw.VolumeRemove(ctx, src.Name, true) }()

	// Best-effort populate (works on a shell-bearing image; the snapshot plumbing
	// is validated either way — an empty volume still tars to a non-empty stream).
	writer, err := raw.ContainerCreate(ctx, &container.Config{
		Image: imageRef,
		Cmd:   []string{"sh", "-c", "echo drydock-marker > /data/marker"},
	}, &container.HostConfig{
		Mounts: []mount.Mount{{Type: mount.TypeVolume, Source: src.Name, Target: "/data"}},
	}, nil, nil, "")
	if err == nil {
		defer func() { _ = raw.ContainerRemove(ctx, writer.ID, container.RemoveOptions{Force: true}) }()
		if startErr := raw.ContainerStart(ctx, writer.ID, container.StartOptions{}); startErr == nil {
			waitC, errC := raw.ContainerWait(ctx, writer.ID, container.WaitConditionNotRunning)
			select {
			case <-waitC:
			case <-errC:
			case <-time.After(10 * time.Second):
			}
		}
	}

	dest := filepath.Join(t.TempDir(), "snapshot.tar")
	n, err := eng.SnapshotVolume(ctx, src.Name, imageRef, dest)
	require.NoError(t, err)
	assert.Positive(t, n, "snapshot streamed a non-empty tar")
	info, err := os.Stat(dest)
	require.NoError(t, err)
	assert.Positive(t, info.Size())

	// Restore into a fresh volume — the helper runs and the call returns cleanly.
	dst, err := raw.VolumeCreate(ctx, volume.CreateOptions{})
	require.NoError(t, err)
	defer func() { _ = raw.VolumeRemove(ctx, dst.Name, true) }()
	require.NoError(t, eng.RestoreVolume(ctx, dst.Name, imageRef, dest))
}
