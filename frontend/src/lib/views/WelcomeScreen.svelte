<script lang="ts">
  // The application's empty state when no host is configured
  // (PROJECT-BOOK §7.11.9: "No hosts yet — add a host to begin").
  import { createEventDispatcher } from 'svelte';
  import Icon from '../components/Icon.svelte';

  const dispatch = createEventDispatcher<{ addhost: void }>();
</script>

<div class="welcome">
  <div class="mark" aria-hidden="true"><Icon name="server" size={34} /></div>
  <h1>No hosts yet</h1>
  <p>
    Add a Docker host to begin. Drydock connects to local and remote engines
    SSH-first; the daemon is treated as root-equivalent and is never exposed.
  </p>
  <button class="cta" on:click={() => dispatch('addhost')}>
    <Icon name="plus" size={16} />
    Add a host
  </button>
</div>

<style>
  .welcome {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-3);
    height: 100%;
    padding: var(--space-6);
    text-align: center;
  }

  .mark {
    display: grid;
    place-items: center;
    width: 72px;
    height: 72px;
    border-radius: var(--radius-lg);
    background: var(--color-accent-soft);
    color: var(--color-accent);
    margin-bottom: var(--space-2);
  }

  h1 {
    margin: 0;
    font-size: var(--text-xl);
    font-weight: 650;
  }

  p {
    margin: 0;
    max-width: 46ch;
    color: var(--color-text-muted);
    font-size: var(--text-md);
    line-height: 1.55;
  }

  .cta {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    margin-top: var(--space-3);
    padding: 9px var(--space-4);
    border: none;
    border-radius: var(--radius-md);
    background: var(--color-accent);
    color: #fff;
    font-size: var(--text-md);
    font-weight: 600;
  }

  .cta:hover {
    filter: brightness(1.08);
  }
</style>
