<script lang="ts">
  // Host connection status, shown as a shaped+coloured dot plus an accessible
  // label. Shape and text carry the meaning so it is never colour-only (§8.5).
  import { HostStatus, type HostStatus as Status } from '../types/domain';

  export let status: Status;
  export let withLabel = false;

  const label: Record<Status, string> = {
    [HostStatus.Connected]: 'Connected',
    [HostStatus.Connecting]: 'Connecting',
    [HostStatus.Disconnected]: 'Disconnected',
    [HostStatus.Degraded]: 'Reduced capability',
    [HostStatus.Error]: 'Error',
  };
</script>

<span class="wrap" title={label[status]}>
  <span class="dot {status}" aria-hidden="true"></span>
  <span class={withLabel ? 'label' : 'sr-only'}>{label[status]}</span>
</span>

<style>
  .wrap {
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }

  .dot {
    width: 9px;
    height: 9px;
    border-radius: 50%;
    flex: none;
    background: var(--color-text-faint);
  }

  .dot.connected {
    background: var(--color-ok);
  }
  .dot.connecting {
    background: var(--color-pending);
    animation: pulse 1.2s ease-in-out infinite;
  }
  /* Degraded: a hollow ring distinguishes it by shape, not colour alone. */
  .dot.degraded {
    background: transparent;
    border: 2px solid var(--color-warn);
  }
  .dot.error {
    background: var(--color-danger);
  }
  .dot.disconnected {
    background: transparent;
    border: 2px solid var(--color-text-faint);
  }

  .label {
    font-size: var(--text-xs);
    color: var(--color-text-muted);
  }

  .sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    margin: -1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
    border: 0;
  }

  @keyframes pulse {
    0%,
    100% {
      opacity: 1;
    }
    50% {
      opacity: 0.4;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .dot.connecting {
      animation: none;
    }
  }
</style>
