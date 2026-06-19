<script lang="ts">
  // The shared frame for the empty / error / degraded states every view must
  // define (PROJECT-BOOK §7.11.9). A centred icon, title, message, and optional
  // action. Loading uses a distinct streaming indicator (LoadingState).
  import Icon from '../Icon.svelte';

  export let icon = 'inbox';
  export let title: string;
  export let message = '';
  /** Visual tone: neutral for empty, danger for errors/untrusted. */
  export let tone: 'neutral' | 'danger' = 'neutral';
</script>

<div class="state" class:danger={tone === 'danger'}>
  <div class="icon" aria-hidden="true"><Icon name={icon} size={28} /></div>
  <h2>{title}</h2>
  {#if message}<p>{message}</p>{/if}
  {#if $$slots.action}
    <div class="action"><slot name="action" /></div>
  {/if}
</div>

<style>
  .state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-3);
    height: 100%;
    padding: var(--space-6);
    text-align: center;
    color: var(--color-text-muted);
  }

  .icon {
    display: grid;
    place-items: center;
    width: 56px;
    height: 56px;
    border-radius: var(--radius-lg);
    background: var(--color-surface-raised);
    border: 1px solid var(--color-border);
    color: var(--color-text-faint);
  }

  .danger .icon {
    background: var(--color-danger-soft);
    border-color: var(--color-danger);
    color: var(--color-danger);
  }

  h2 {
    margin: 0;
    font-size: var(--text-lg);
    font-weight: 600;
    color: var(--color-text);
  }

  p {
    margin: 0;
    max-width: 42ch;
    font-size: var(--text-sm);
    line-height: 1.5;
  }

  .action {
    margin-top: var(--space-2);
  }
</style>
