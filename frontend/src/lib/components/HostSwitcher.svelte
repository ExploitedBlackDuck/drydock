<script lang="ts">
  // The always-visible host switcher (PROJECT-BOOK §7.11.1). Lists known hosts
  // with their status and lets the operator switch the active host or start the
  // add-host flow. Reads the registry store; never holds its own copy.
  import { createEventDispatcher } from 'svelte';
  import Icon from './Icon.svelte';
  import StatusDot from './StatusDot.svelte';
  import { hosts } from '../stores/hosts';

  const dispatch = createEventDispatcher<{ addhost: void }>();
</script>

<div class="switcher">
  <div class="heading">Hosts</div>

  {#if $hosts.hosts.length === 0}
    <p class="empty">No hosts connected.</p>
  {:else}
    <ul>
      {#each $hosts.hosts as host (host.id)}
        <li>
          <button
            class="host"
            class:active={host.id === $hosts.activeId}
            aria-current={host.id === $hosts.activeId ? 'true' : undefined}
            on:click={() => hosts.select(host.id)}
          >
            <StatusDot status={host.status} />
            <span class="name">{host.name}</span>
          </button>
        </li>
      {/each}
    </ul>
  {/if}

  <button class="add" on:click={() => dispatch('addhost')}>
    <Icon name="plus" size={15} />
    Add host
  </button>
</div>

<style>
  .switcher {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    padding: var(--space-3);
    border-bottom: 1px solid var(--color-border);
  }

  .heading {
    font-size: var(--text-xs);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--color-text-faint);
  }

  .empty {
    margin: 0;
    font-size: var(--text-sm);
    color: var(--color-text-faint);
  }

  ul {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .host {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    width: 100%;
    padding: 7px var(--space-2);
    border: none;
    border-radius: var(--radius-sm);
    background: transparent;
    color: var(--color-text);
    font-size: var(--text-sm);
    text-align: left;
  }

  .host:hover {
    background: var(--color-surface-hover);
  }

  .host.active {
    background: var(--color-accent-soft);
    color: var(--color-text);
  }

  .name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .add {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    padding: 7px;
    border: 1px solid var(--color-border-strong);
    border-radius: var(--radius-sm);
    background: var(--color-surface-raised);
    color: var(--color-text);
    font-size: var(--text-sm);
    font-weight: 500;
  }

  .add:hover {
    background: var(--color-surface-hover);
    border-color: var(--color-accent);
  }
</style>
