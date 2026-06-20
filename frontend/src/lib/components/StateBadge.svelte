<script lang="ts">
  // Container state shown as a shaped, coloured pill plus its label, so meaning
  // is never colour-only (PROJECT-BOOK §8.5).
  export let state: string;

  // Map engine states to a tone. Unknown states fall back to neutral.
  const tone: Record<string, 'ok' | 'warn' | 'danger' | 'neutral'> = {
    running: 'ok',
    restarting: 'warn',
    paused: 'warn',
    created: 'neutral',
    exited: 'neutral',
    removing: 'danger',
    dead: 'danger',
  };

  $: t = tone[state] ?? 'neutral';
</script>

<span class="badge {t}" title="State: {state}">
  <span class="dot" aria-hidden="true"></span>
  {state}
</span>

<style>
  .badge {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 2px 8px;
    border-radius: 999px;
    font-size: var(--text-xs);
    font-weight: 500;
    text-transform: capitalize;
    border: 1px solid var(--color-border-strong);
    color: var(--color-text-muted);
  }
  .dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--color-text-faint);
  }
  .ok {
    color: var(--color-ok);
    border-color: color-mix(in srgb, var(--color-ok) 45%, transparent);
  }
  .ok .dot {
    background: var(--color-ok);
  }
  .warn {
    color: var(--color-warn);
    border-color: color-mix(in srgb, var(--color-warn) 45%, transparent);
  }
  .warn .dot {
    background: var(--color-warn);
  }
  .danger {
    color: var(--color-danger);
    border-color: color-mix(in srgb, var(--color-danger) 45%, transparent);
  }
  .danger .dot {
    background: var(--color-danger);
  }
  .neutral .dot {
    /* hollow ring distinguishes neutral by shape as well as colour */
    background: transparent;
    border: 1.5px solid var(--color-text-faint);
    width: 8px;
    height: 8px;
  }
</style>
