<script lang="ts">
  import { onMount } from 'svelte';
  import AppShell from './lib/components/AppShell.svelte';
  import { getVersion, onAppReady } from './lib/api/app';

  let version = '';

  onMount(() => {
    // The version is available via a binding immediately; the ready event
    // confirms the backend runtime started end to end.
    getVersion()
      .then((v) => (version = v))
      .catch(() => (version = 'unknown'));

    return onAppReady((v) => (version = v));
  });
</script>

<AppShell {version} />
