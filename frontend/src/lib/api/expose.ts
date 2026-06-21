// Typed seam over the exposure binding (PROJECT-BOOK §7.12.3, §2.8).
import { HostExposure } from '../../../wailsjs/go/app/App';
import type { domain } from '../../../wailsjs/go/models';

export type ExposureMap = domain.ExposureMap;
export type PortBinding = domain.PortBinding;
export type HostNetworkRef = domain.HostNetworkRef;

/** Computes the host's exposure map — read-only insight, never enforcement. */
export function hostExposure(hostId: string): Promise<ExposureMap> {
  return HostExposure(hostId);
}
