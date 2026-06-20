<script lang="ts">
  // Surfaces restart-loop (crash-loop) alerts for the active host (PROJECT-BOOK
  // §7.6/§7.11.7). It subscribes to the host's event stream and shows a callout
  // for any container detected restarting repeatedly.
  import { onDestroy } from 'svelte';
  import Icon from './Icon.svelte';
  import {
    onRestartLoop,
    subscribeEvents,
    unsubscribeEvents,
    type RestartLoopAlert,
  } from '../api/dashboard';

  export let hostId: string;

  let alerts: Record<string, RestartLoopAlert> = {};
  let dismissed = new Set<string>();
  let current: string | null = null;
  let unsub: (() => void) | null = null;

  // (Re)subscribe whenever the active host changes.
  $: if (hostId !== current) {
    teardown();
    current = hostId;
    alerts = {};
    dismissed = new Set();
    if (hostId) {
      void subscribeEvents(hostId);
      unsub = onRestartLoop(hostId, (a) => {
        alerts = { ...alerts, [a.containerId]: a };
      });
    }
  }

  $: visible = Object.values(alerts).filter(
    (a) => !dismissed.has(a.containerId),
  );

  function dismiss(id: string) {
    dismissed = new Set(dismissed).add(id);
  }

  function teardown() {
    if (unsub) {
      unsub();
      unsub = null;
    }
    if (current) void unsubscribeEvents(current);
  }

  onDestroy(teardown);
</script>

{#if visible.length > 0}
  <div class="banner" role="alert">
    <Icon name="alert" size={16} />
    <div class="msg">
      {#each visible as a (a.containerId)}
        <span class="item">
          <b>{a.containerName || a.containerId.slice(0, 12)}</b> is restarting
          repeatedly ({a.deaths} times)
          <button
            class="x"
            aria-label="Dismiss"
            on:click={() => dismiss(a.containerId)}>✕</button
          >
        </span>
      {/each}
    </div>
  </div>
{/if}

<style>
  .banner {
    display: flex;
    align-items: flex-start;
    gap: var(--space-2);
    padding: var(--space-2) var(--space-5);
    background: var(--color-danger-soft);
    color: var(--color-danger);
    border-bottom: 1px solid var(--color-danger);
    font-size: var(--text-sm);
  }
  .msg {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .item {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
  }
  .x {
    border: none;
    background: transparent;
    color: inherit;
    opacity: 0.7;
    font-size: 11px;
  }
  .x:hover {
    opacity: 1;
  }
</style>
