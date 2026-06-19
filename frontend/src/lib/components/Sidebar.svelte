<script lang="ts">
  // The left sidebar: brand, host switcher, and view navigation.
  import { createEventDispatcher } from 'svelte';
  import Icon from './Icon.svelte';
  import HostSwitcher from './HostSwitcher.svelte';
  import NavRail from './NavRail.svelte';
  import { activeHost } from '../stores/hosts';

  const dispatch = createEventDispatcher<{ addhost: void }>();
</script>

<aside>
  <div class="brand">
    <span class="mark" aria-hidden="true"><Icon name="server" size={18} /></span
    >
    <span class="name">Drydock</span>
  </div>

  <HostSwitcher on:addhost={() => dispatch('addhost')} />
  <NavRail enabled={$activeHost !== null} />
</aside>

<style>
  aside {
    display: flex;
    flex-direction: column;
    width: var(--sidebar-width);
    flex: none;
    height: 100vh;
    background: var(--color-surface);
    border-right: 1px solid var(--color-border);
  }

  .brand {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    height: var(--topbar-height);
    flex: none;
    padding: 0 var(--space-4);
    border-bottom: 1px solid var(--color-border);
  }

  .mark {
    display: grid;
    place-items: center;
    width: 28px;
    height: 28px;
    border-radius: var(--radius-sm);
    background: var(--color-accent-soft);
    color: var(--color-accent);
  }

  .name {
    font-size: var(--text-lg);
    font-weight: 600;
    letter-spacing: -0.01em;
  }
</style>
