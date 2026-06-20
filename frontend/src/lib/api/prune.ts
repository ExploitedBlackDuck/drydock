// Typed seam over the prune-preview and prune/remove bindings.
import {
  GetPruneImpact,
  PruneBuildCache,
  PruneContainers,
  PruneImages,
  RemoveVolume,
} from '../../../wailsjs/go/app/App';
import type { domain } from '../../../wailsjs/go/models';

export type PruneImpact = domain.PruneImpact;
export type PruneCategory = domain.PruneCategory;
export type VolumeRef = domain.VolumeRef;

export function getPruneImpact(hostId: string): Promise<PruneImpact> {
  return GetPruneImpact(hostId);
}

export function pruneImages(
  hostId: string,
  all: boolean,
  ack: boolean,
): Promise<number> {
  return PruneImages(hostId, all, ack);
}

export function pruneContainers(hostId: string, ack: boolean): Promise<number> {
  return PruneContainers(hostId, ack);
}

export function pruneBuildCache(hostId: string, ack: boolean): Promise<number> {
  return PruneBuildCache(hostId, ack);
}

export function removeVolume(
  hostId: string,
  name: string,
  ack: boolean,
): Promise<void> {
  return RemoveVolume(hostId, name, ack);
}
