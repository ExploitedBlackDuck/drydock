// Host registry store (PROJECT-BOOK §7.11.10), backed by the Go registry. It
// loads host profiles and their connection state from the backend and exposes
// actions (add, connect, observe-mode) that round-trip through it.
import { derived, writable } from 'svelte/store';
import type { Host } from '../types/domain';
import {
  addHost as addHostApi,
  connectHost as connectHostApi,
  listHosts,
  removeHost as removeHostApi,
  setObserveMode as setObserveModeApi,
  type AddHostInput,
} from '../api/hosts';

export type LoadStatus = 'idle' | 'loading' | 'ready' | 'error';

interface HostsState {
  status: LoadStatus;
  hosts: Host[];
  activeId: string | null;
  error: string | null;
}

const initial: HostsState = {
  status: 'idle',
  hosts: [],
  activeId: null,
  error: null,
};

function messageOf(err: unknown): string {
  if (err instanceof Error) return err.message;
  return typeof err === 'string' ? err : 'Unexpected error';
}

function createHostsStore() {
  const { subscribe, update } = writable<HostsState>(initial);

  async function refresh(): Promise<void> {
    update((s) => ({ ...s, status: s.hosts.length ? s.status : 'loading' }));
    try {
      const hosts = await listHosts();
      update((s) => ({
        status: 'ready',
        hosts,
        error: null,
        activeId:
          s.activeId && hosts.some((h) => h.id === s.activeId)
            ? s.activeId
            : (hosts[0]?.id ?? null),
      }));
    } catch (err) {
      update((s) => ({ ...s, status: 'error', error: messageOf(err) }));
    }
  }

  return {
    subscribe,
    refresh,
    select(id: string) {
      update((s) => ({ ...s, activeId: id }));
    },
    async add(input: AddHostInput): Promise<void> {
      const host = await addHostApi(input);
      await refresh();
      update((s) => ({ ...s, activeId: host.id }));
    },
    async connect(id: string): Promise<void> {
      await connectHostApi(id);
      await refresh();
    },
    async remove(id: string): Promise<void> {
      await removeHostApi(id);
      await refresh();
    },
    async setObserveMode(id: string, observe: boolean): Promise<void> {
      await setObserveModeApi(id, observe);
      await refresh();
    },
  };
}

export const hosts = createHostsStore();

/** The currently selected host, or null when none is active. */
export const activeHost = derived(
  hosts,
  ($hosts) => $hosts.hosts.find((h) => h.id === $hosts.activeId) ?? null,
);
