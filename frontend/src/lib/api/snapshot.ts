// Typed seam over the volume-snapshot bindings (PROJECT-BOOK §7.12.6, §2.8).
import {
  PreviewVolumeSnapshot,
  SnapshotVolume,
} from '../../../wailsjs/go/app/App';
import type { domain } from '../../../wailsjs/go/models';

export type VolumeSnapshotPreview = domain.VolumeSnapshotPreview;

/** States the cost (destination, estimated size and duration) before running. */
export function previewVolumeSnapshot(
  hostId: string,
  volume: string,
): Promise<VolumeSnapshotPreview> {
  return PreviewVolumeSnapshot(hostId, volume);
}

/** Captures the volume to the previewed destination; returns the file written. */
export function snapshotVolume(
  hostId: string,
  volume: string,
): Promise<string> {
  return SnapshotVolume(hostId, volume);
}
