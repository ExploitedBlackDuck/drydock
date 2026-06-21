<script lang="ts">
  // Refetch-on-resync (PROJECT-BOOK §7.11.9, ADR-0021): when the live event
  // stream for the active host drops or is re-established, the backend emits a
  // resync signal. Rather than trust the now-stale streamed state, we refetch
  // authoritative data (host status + object lists) and show a brief indicator.
  // Mounted keyed by host id, so its subscription tears down on host change.
  import { onMount } from 'svelte';
  import { onResync } from '../api/dashboard';
  import { reloadObjects } from '../stores/objects';
  import { hosts } from '../stores/hosts';

  export let hostId: string;

  let resyncing = false;

  onMount(() =>
    onResync(hostId, async () => {
      resyncing = true;
      try {
        await hosts.refresh();
        await reloadObjects(hostId);
      } catch {
        // A failed refetch leaves the views in their own error/empty state; the
        // host switcher reflects the disconnect. Never resume stale data.
      } finally {
        resyncing = false;
      }
    }),
  );
</script>

{#if resyncing}
  <div class="resync" role="status">
    <span class="dot" aria-hidden="true"></span>
    Resyncing live state…
  </div>
{/if}

<style>
  .resync {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: var(--space-2) var(--space-5);
    background: var(--color-surface-raised);
    border-bottom: 1px solid var(--color-border);
    color: var(--color-text-muted);
    font-size: var(--text-sm);
  }
  .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--color-warn);
    animation: pulse 1s ease-in-out infinite;
  }
  @keyframes pulse {
    0%,
    100% {
      opacity: 0.4;
    }
    50% {
      opacity: 1;
    }
  }
</style>
