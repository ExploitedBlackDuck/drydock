<script lang="ts">
  // The transport / trust / observe indicators that must always be visible for
  // the active host (PROJECT-BOOK §7.11.1, §8.5): the operator can never lose
  // track of which machine, reached how, they are about to act on. Each badge
  // pairs an icon and a label, so meaning is never colour-only.
  import Icon from './Icon.svelte';
  import { Transport, Trust, type Host } from '../types/domain';

  export let host: Host;

  const transportIcon = {
    [Transport.Local]: 'local',
    [Transport.SSH]: 'ssh',
    [Transport.TLS]: 'tls',
  };
  const transportLabel = {
    [Transport.Local]: 'Local',
    [Transport.SSH]: 'SSH',
    [Transport.TLS]: 'TLS',
  };
</script>

<div class="badges">
  <span class="badge" title="Transport: {transportLabel[host.transport]}">
    <Icon name={transportIcon[host.transport]} size={13} />
    {transportLabel[host.transport]}
  </span>

  <span
    class="badge"
    class:danger={host.trust === Trust.Untrusted}
    title={host.trust === Trust.Untrusted
      ? 'Untrusted transport — verify how access is travelling'
      : 'Trusted transport'}
  >
    <Icon
      name={host.trust === Trust.Untrusted ? 'alert' : 'shield'}
      size={13}
    />
    {host.trust === Trust.Untrusted ? 'Untrusted' : 'Trusted'}
  </span>

  {#if host.observeMode}
    <span
      class="badge observe"
      title="Observe-only: mutations are rejected in the core"
    >
      <Icon name="eye" size={13} />
      Observe
    </span>
  {/if}
</div>

<style>
  .badges {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2);
  }

  .badge {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding: 2px 7px;
    border-radius: 999px;
    border: 1px solid var(--color-border-strong);
    background: var(--color-surface-raised);
    color: var(--color-text-muted);
    font-size: var(--text-xs);
    font-weight: 500;
    white-space: nowrap;
  }

  .badge.danger {
    border-color: var(--color-danger);
    background: var(--color-danger-soft);
    color: var(--color-danger);
  }

  .badge.observe {
    border-color: var(--color-warn);
    color: var(--color-warn);
  }
</style>
