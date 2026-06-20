<script lang="ts">
  // Top-level application layout: sidebar + top bar + the active view, with the
  // add-host wizard mounted on demand. Pure composition; data lives in stores.
  import Sidebar from './Sidebar.svelte';
  import TopBar from './TopBar.svelte';
  import AddHostWizard from './AddHostWizard.svelte';
  import RestartLoopBanner from './RestartLoopBanner.svelte';
  import PrimaryView from '../views/PrimaryView.svelte';
  import WelcomeScreen from '../views/WelcomeScreen.svelte';
  import { HostStatus } from '../types/domain';
  import { activeHost } from '../stores/hosts';

  export let version = '';

  let wizardOpen = false;
</script>

<div class="app">
  <Sidebar on:addhost={() => (wizardOpen = true)} />

  <div class="main">
    <TopBar {version} />
    {#if $activeHost && $activeHost.status === HostStatus.Connected}
      <RestartLoopBanner hostId={$activeHost.id} />
    {/if}
    <main>
      {#if $activeHost}
        <PrimaryView host={$activeHost} />
      {:else}
        <WelcomeScreen on:addhost={() => (wizardOpen = true)} />
      {/if}
    </main>
  </div>
</div>

{#if wizardOpen}
  <AddHostWizard on:close={() => (wizardOpen = false)} />
{/if}

<style>
  .app {
    display: flex;
    height: 100vh;
    overflow: hidden;
  }

  .main {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-width: 0;
  }

  main {
    flex: 1;
    min-height: 0;
    overflow: hidden;
    background: var(--color-bg);
  }
</style>
