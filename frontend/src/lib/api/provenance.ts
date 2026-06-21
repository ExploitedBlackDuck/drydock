// Typed seam over the image-provenance bindings (PROJECT-BOOK §7.12.5, §2.8).
import {
  CheckImageDrift,
  ListImageProvenance,
} from '../../../wailsjs/go/app/App';
import type { domain } from '../../../wailsjs/go/models';

export type ImageProvenance = domain.ImageProvenance;

/** Local provenance for the host's images — no registry call (no phone-home). */
export function listImageProvenance(
  hostId: string,
): Promise<ImageProvenance[]> {
  return ListImageProvenance(hostId);
}

/** Explicit, operator-initiated registry digest check through the host's daemon. */
export function checkImageDrift(
  hostId: string,
  imageRef: string,
): Promise<ImageProvenance> {
  return CheckImageDrift(hostId, imageRef);
}
