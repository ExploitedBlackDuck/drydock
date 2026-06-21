package app

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/drydock/drydock/internal/core/domain"
)

// snapshotThroughputBytesPerSec is a rough estimate used to state expected
// duration before a snapshot runs (ADR-0020). It is intentionally conservative.
const snapshotThroughputBytesPerSec = 50 * 1024 * 1024

// PreviewVolumeSnapshot states the cost of snapshotting a volume before it runs
// (ADR-0020): the destination Drydock will write, an estimated size (from
// `system df`), and an expected duration. Nothing is started.
func (a *App) PreviewVolumeSnapshot(hostID, volume string) (domain.VolumeSnapshotPreview, error) {
	ctx, cancel := a.requestCtx()
	defer cancel()
	eng, err := a.registry.Engine(hostID)
	if err != nil {
		return domain.VolumeSnapshotPreview{}, err
	}

	var sizeBytes int64
	if usage, derr := eng.DiskUsage(ctx); derr == nil {
		for _, v := range usage.Volumes {
			if v.Name == volume {
				sizeBytes = v.Size
				break
			}
		}
	}

	seconds := int64(1)
	if sizeBytes > 0 {
		seconds = sizeBytes/snapshotThroughputBytesPerSec + 1
	}
	return domain.VolumeSnapshotPreview{
		Volume:          volume,
		Destination:     a.snapshotDestination(volume),
		EstimatedBytes:  sizeBytes,
		EstimatedSecond: seconds,
	}, nil
}

// SnapshotVolume captures a volume to the previewed destination and returns the
// file path written. It is an explicit, acknowledged, audited operation; it
// never blocks or precedes deletion.
func (a *App) SnapshotVolume(hostID, volume string) (string, error) {
	dest := a.snapshotDestination(volume)
	ctx, cancel := a.requestCtx()
	defer cancel()
	if err := a.ops.VolumeSnapshot(ctx, hostID, volume, dest, a.snapshotHelper, true); err != nil {
		return "", err
	}
	return dest, nil
}

// RestoreVolume extracts a snapshot file back into a volume — a separate,
// deliberately-confirmed, observe-mode-blocked operation (ADR-0020).
func (a *App) RestoreVolume(hostID, volume, source string) error {
	ctx, cancel := a.requestCtx()
	defer cancel()
	return a.ops.VolumeRestore(ctx, hostID, volume, source, a.snapshotHelper, true)
}

// snapshotDestination is where a volume's snapshot is written: a per-volume,
// timestamped tar under the operator's home directory (a plaintext archive
// outside Drydock's sealed store — the operator protects it).
func (a *App) snapshotDestination(volume string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	stamp := time.Now().UTC().Format("20060102T150405Z")
	return filepath.Join(home, fmt.Sprintf("drydock-%s-%s.tar", volume, stamp))
}
