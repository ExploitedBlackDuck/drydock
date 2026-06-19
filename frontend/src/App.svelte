<script lang="ts">
  import { onMount } from 'svelte';
  import { getVersion, onAppReady } from './lib/api/app';

  let version = '…';
  let ready = false;

  onMount(() => {
    // The version is available immediately via a binding call; the ready event
    // confirms the backend runtime has started end to end.
    getVersion()
      .then((v) => (version = v))
      .catch(() => (version = 'unknown'));

    const unsubscribe = onAppReady((v) => {
      version = v;
      ready = true;
    });
    return unsubscribe;
  });
</script>

<main>
  <h1>Drydock</h1>
  <p class="tagline">Docker hosts, made legible.</p>
  <p class="status" class:ready>
    <span class="dot" aria-hidden="true"></span>
    {ready ? `Backend ready · v${version}` : `Starting · v${version}`}
  </p>
</main>

<style>
  main {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100vh;
    gap: 0.5rem;
    text-align: center;
  }

  h1 {
    margin: 0;
    font-size: 2.5rem;
    letter-spacing: -0.02em;
  }

  .tagline {
    margin: 0;
    color: var(--color-text-muted);
  }

  .status {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    margin-top: 1rem;
    font-variant-numeric: tabular-nums;
    color: var(--color-text-muted);
  }

  .dot {
    width: 0.6rem;
    height: 0.6rem;
    border-radius: 50%;
    background: var(--color-pending);
  }

  .status.ready .dot {
    background: var(--color-ok);
  }
</style>
