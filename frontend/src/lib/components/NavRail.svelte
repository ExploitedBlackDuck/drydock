<script lang="ts">
  // The primary-view navigation (PROJECT-BOOK §7.11.1), generated from the typed
  // VIEWS catalog. Disabled until a host is active, since every view is per-host.
  import Icon from './Icon.svelte';
  import { VIEWS } from '../types/view';
  import { activeView } from '../stores/navigation';

  export let enabled = true;
</script>

<nav aria-label="Views">
  <ul>
    {#each VIEWS as view (view.id)}
      <li>
        <button
          class="item"
          class:active={$activeView === view.id}
          aria-current={$activeView === view.id ? 'page' : undefined}
          disabled={!enabled}
          on:click={() => activeView.set(view.id)}
        >
          <Icon name={view.icon} size={17} />
          <span>{view.label}</span>
        </button>
      </li>
    {/each}
  </ul>
</nav>

<style>
  nav {
    flex: 1;
    overflow-y: auto;
    padding: var(--space-3);
  }

  ul {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .item {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    width: 100%;
    padding: 8px var(--space-3);
    border: none;
    border-radius: var(--radius-sm);
    background: transparent;
    color: var(--color-text-muted);
    font-size: var(--text-md);
    text-align: left;
  }

  .item:hover:not(:disabled) {
    background: var(--color-surface-hover);
    color: var(--color-text);
  }

  .item.active {
    background: var(--color-accent-soft);
    color: var(--color-text);
    font-weight: 500;
  }

  .item:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }
</style>
