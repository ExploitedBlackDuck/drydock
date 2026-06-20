// Typed seam over the generated engine bindings. Components import data types
// and calls from here, never from generated paths directly (PROJECT-BOOK §2.8).
import {
  ListContainers,
  ListImages,
  ListNetworks,
  ListVolumes,
} from '../../../wailsjs/go/app/App';
import type { domain } from '../../../wailsjs/go/models';

export type Container = domain.Container;
export type Image = domain.Image;
export type Volume = domain.Volume;
export type Network = domain.Network;

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
