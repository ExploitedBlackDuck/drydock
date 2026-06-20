<script lang="ts">
  import { onMount } from 'svelte';
  import AppShell from './lib/components/AppShell.svelte';
  import { getVersion, onAppReady } from './lib/api/app';
  import { localEngine } from './lib/api/engine';
  import { hosts } from './lib/stores/hosts';

  let version = '';

  onMount(() => {
    // The version is available via a binding immediately; the ready event
    // confirms the backend runtime started end to end.
    getVersion()
      .then((v) => (version = v))
      .catch(() => (version = 'unknown'));

    // Reflect the local Docker engine in the host switcher when reachable.
    localEngine()
      .then((status) => hosts.connectLocal(status))
      .catch(() => {
        /* no local engine; the welcome state guides adding a host */
      });

    return onAppReady((v) => (version = v));
  });
</script>

<AppShell {version} />
