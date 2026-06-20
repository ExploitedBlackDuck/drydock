<script lang="ts">
  // The top bar reinforces the always-visible active-host context
  // (PROJECT-BOOK §7.11.1): which host, reached how, and its trust/observe
  // state, alongside its engine/API version when connected.
  import StatusDot from './StatusDot.svelte';
  import HostBadges from './HostBadges.svelte';
  import Icon from './Icon.svelte';
  import { activeHost, hosts } from '../stores/hosts';

  export let version = '';

  async function toggleObserve() {
    const host = $activeHost;
    if (!host) return;
    const next = !host.observeMode;
    const verb = next ? 'enable' : 'disable';
    // Explicit confirmation; the change is audited in the core (ADR-0013).
    if (
      !window.confirm(
        `${verb === 'enable' ? 'Enable' : 'Disable'} observe-only mode for "${host.name}"?`,
      )
    ) {
      return;
    }
    await hosts.setObserveMode(host.id, next);
  }
</script>

<header>
  <div class="host">
    {#if $activeHost}
      <StatusDot status={$activeHost.status} />
      <span class="name">{$activeHost.name}</span>
      <span class="endpoint">{$activeHost.endpoint}</span>
      <HostBadges host={$activeHost} />
      {#if $activeHost.engineVersion}
        <span class="version">
          Engine {$activeHost.engineVersion} · API {$activeHost.apiVersion}
        </span>
      {/if}
    {:else}
      <span class="none">No host selected</span>
    {/if}
  </div>

  <div class="actions">
    {#if $activeHost}
      <button
        class="observe"
        class:on={$activeHost.observeMode}
        on:click={toggleObserve}
        title="Observe-only rejects all mutations in the core"
      >
        <Icon name="eye" size={14} />
        {$activeHost.observeMode ? 'Observe on' : 'Observe off'}
      </button>
    {/if}
    {#if version}<span class="app-version">v{version}</span>{/if}
  </div>
</header>

<style>
  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-4);
    height: var(--topbar-height);
    flex: none;
    padding: 0 var(--space-5);
    background: var(--color-surface);
    border-bottom: 1px solid var(--color-border);
  }

  .host {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    min-width: 0;
  }

  .name {
    font-weight: 600;
    font-size: var(--text-md);
  }

  .endpoint {
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    color: var(--color-text-faint);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 28ch;
  }

  .version {
    font-size: var(--text-xs);
    color: var(--color-text-muted);
  }

  .none {
    color: var(--color-text-faint);
    font-size: var(--text-md);
  }

  .actions {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    flex: none;
  }

  .observe {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 4px 10px;
    border-radius: 999px;
    border: 1px solid var(--color-border-strong);
    background: transparent;
    color: var(--color-text-muted);
    font-size: var(--text-xs);
    font-weight: 500;
  }

  .observe:hover {
    background: var(--color-surface-hover);
  }

  .observe.on {
    border-color: var(--color-warn);
    color: var(--color-warn);
  }

  .app-version {
    font-size: var(--text-xs);
    color: var(--color-text-faint);
    font-variant-numeric: tabular-nums;
  }
</style>
