<script lang="ts">
  // Loads a resource for the active host and resolves its loading/error/empty
  // states (PROJECT-BOOK §7.11.9), exposing the rows to the default slot when
  // ready. Reloads whenever the host changes.
  import { onMount } from 'svelte';
  import StateMessage from '../components/states/StateMessage.svelte';
  import LoadingState from '../components/states/LoadingState.svelte';
  import type { ResourceStore } from '../stores/objects';

  type T = $$Generic;

  export let hostId: string;
  export let store: ResourceStore<T>;
  export let icon: string;
  export let emptyTitle: string;
  export let emptyMessage: string;

  // Reload when the host changes; onMount covers the initial render.
  let loadedFor: string | null = null;
  $: if (hostId && hostId !== loadedFor) {
    loadedFor = hostId;
    void store.load(hostId);
  }
  onMount(() => {
    if (loadedFor === null && hostId) {
      loadedFor = hostId;
      void store.load(hostId);
    }
  });

  $: state = $store;
</script>

{#if state.status === 'loading' && state.data.length === 0}
  <LoadingState label="Loading…" />
{:else if state.status === 'error'}
  <StateMessage
    tone="danger"
    icon="alert"
    title="Could not load"
    message={state.error ?? 'The engine request failed.'}
  />
{:else if state.data.length === 0}
  <StateMessage {icon} title={emptyTitle} message={emptyMessage} />
{:else}
  <slot rows={state.data} />
{/if}
