// Typed seam over the generated engine bindings. Components import data types
// and calls from here, never from generated paths directly (PROJECT-BOOK §2.8).
import {
  ListContainers,
  ListImages,
  ListNetworks,
  ListVolumes,
  LocalEngine,
} from '../../../wailsjs/go/app/App';
import type { app, domain } from '../../../wailsjs/go/models';

export type LocalEngineStatus = app.LocalEngineStatus;
export type Container = domain.Container;
export type Image = domain.Image;
export type Volume = domain.Volume;
export type Network = domain.Network;

/** Probes the local Docker engine; never rejects — availability is in the result. */
export function localEngine(): Promise<LocalEngineStatus> {
  // Guard the case where the Wails runtime is not injected (app opened outside
  // the desktop shell): report unavailable rather than throwing on mount.
  if (!('go' in globalThis)) {
    return Promise.resolve({
      available: false,
      hostId: 'local',
      engineVersion: '',
      apiVersion: '',
      os: '',
      arch: '',
      degraded: false,
    });
  }
  return LocalEngine();
}

export function listContainers(hostId: string): Promise<Container[]> {
  return ListContainers(hostId);
}

export function listImages(hostId: string): Promise<Image[]> {
  return ListImages(hostId);
}

export function listVolumes(hostId: string): Promise<Volume[]> {
  return ListVolumes(hostId);
}

export function listNetworks(hostId: string): Promise<Network[]> {
  return ListNetworks(hostId);
}
