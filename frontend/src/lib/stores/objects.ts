// Engine object stores (PROJECT-BOOK §7.11.10: the `objects` concern). Each
// resource tracks its own load status so views can render loading/error/empty
// distinctly. Data is fetched per host through the typed bindings.
import { writable } from 'svelte/store';
import {
  listContainers,
  listImages,
  listNetworks,
  listVolumes,
  type Container,
  type Image,
  type Network,
  type Volume,
} from '../api/engine';
import { listStacks, type Stack } from '../api/compose';
import type { LoadStatus } from './hosts';

export interface ResourceState<T> {
  status: LoadStatus;
  data: T[];
  error: string | null;
}

export interface ResourceStore<T> {
  subscribe: (run: (value: ResourceState<T>) => void) => () => void;
  load: (hostId: string) => Promise<void>;
}

function createResource<T>(
  loader: (hostId: string) => Promise<T[]>,
): ResourceStore<T> {
  const { subscribe, set } = writable<ResourceState<T>>({
    status: 'idle',
    data: [],
    error: null,
  });

  async function load(hostId: string): Promise<void> {
    set({ status: 'loading', data: [], error: null });
    try {
      const data = (await loader(hostId)) ?? [];
      set({ status: 'ready', data, error: null });
    } catch (err) {
      set({ status: 'error', data: [], error: messageOf(err) });
    }
  }

  return { subscribe, load };
}

function messageOf(err: unknown): string {
  if (err instanceof Error) return err.message;
  return typeof err === 'string' ? err : 'Unexpected error';
}

export const containers = createResource<Container>(listContainers);
export const images = createResource<Image>(listImages);
export const volumes = createResource<Volume>(listVolumes);
export const networks = createResource<Network>(listNetworks);
export const stacks = createResource<Stack>(listStacks);

/**
 * Refetches every object collection for a host — used on a resync (ADR-0021) so
 * the views show authoritative state after a live stream dropped, rather than
 * keeping whatever was last streamed.
 */
export function reloadObjects(hostId: string): Promise<void> {
  return Promise.all([
    containers.load(hostId),
    images.load(hostId),
    volumes.load(hostId),
    networks.load(hostId),
    stacks.load(hostId),
  ]).then(() => undefined);
}
