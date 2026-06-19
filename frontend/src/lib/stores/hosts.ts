// Host registry store (PROJECT-BOOK §7.11.10): the set of known hosts, the
// active selection, and the load status. Runtime/view state is kept here, not
// scattered in components. The backend binding that populates the registry lands
// in P3 (SSH transport + add-host wizard); until then the store reports a ready,
// empty registry, which drives the "no hosts yet" empty state.

import { derived, writable } from 'svelte/store';
import type { Host } from '../types/domain';

/** Load status for an async-backed collection (drives loading/error/empty UI). */
export type LoadStatus = 'idle' | 'loading' | 'ready' | 'error';

interface HostsState {
  status: LoadStatus;
  hosts: Host[];
  activeId: string | null;
  error: string | null;
}

const initial: HostsState = {
  status: 'ready',
  hosts: [],
  activeId: null,
  error: null,
};

function createHostsStore() {
  const { subscribe, update, set } = writable<HostsState>(initial);

  return {
    subscribe,
    /** Replace the registry (called by the binding layer once wired). */
    setHosts(hosts: Host[]) {
      update((s) => ({
        ...s,
        status: 'ready',
        hosts,
        error: null,
        activeId: s.activeId ?? hosts[0]?.id ?? null,
      }));
    },
    /** Mark the registry as loading. */
    loading() {
      update((s) => ({ ...s, status: 'loading' }));
    },
    /** Record a load failure with a human-readable message from the error DTO. */
    failed(message: string) {
      update((s) => ({ ...s, status: 'error', error: message }));
    },
    /** Append a host to the registry and make it active. */
    add(host: Host) {
      update((s) => ({
        ...s,
        status: 'ready',
        hosts: [...s.hosts, host],
        activeId: host.id,
      }));
    },
    /** Select the active host. */
    select(id: string) {
      update((s) => ({ ...s, activeId: id }));
    },
    reset() {
      set(initial);
    },
  };
}

export const hosts = createHostsStore();

/** The currently selected host, or null when none is active. */
export const activeHost = derived(
  hosts,
  ($hosts) => $hosts.hosts.find((h) => h.id === $hosts.activeId) ?? null,
);
