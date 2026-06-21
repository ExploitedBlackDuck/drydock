<script lang="ts">
  // Exposure map (PROJECT-BOOK §7.12.3, ADR-0017): what each container publishes
  // and how far it reaches. All-interfaces bindings on a remotely-reached host
  // are surfaced first; host-network containers are listed explicitly (their
  // exposure is not derivable from bindings). Read-only insight — Drydock never
  // edits a firewall or rebinds a port; it reports the binding, not the host's
  // actual reachable addresses.
  import StateMessage from '../components/states/StateMessage.svelte';
  import LoadingState from '../components/states/LoadingState.svelte';
  import {
    hostExposure,
    type ExposureMap,
    type PortBinding,
  } from '../api/expose';

  export let hostId: string;

  let status: 'loading' | 'ready' | 'error' = 'loading';
  let map: ExposureMap | null = null;
  let error = '';

  async function load() {
    status = 'loading';
    error = '';
    try {
      map = await hostExposure(hostId);
      status = 'ready';
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
      status = 'error';
    }
  }

  let loadedFor: string | null = null;
  $: if (hostId && hostId !== loadedFor) {
    loadedFor = hostId;
    void load();
  }

  const reachTone: Record<string, 'ok' | 'warn' | 'danger'> = {
    loopback: 'ok',
    private: 'warn',
    all_interfaces: 'danger',
  };
  const reachLabel: Record<string, string> = {
    loopback: 'loopback',
    private: 'private/LAN',
    all_interfaces: 'all interfaces',
  };

  function source(b: PortBinding): string {
    const ip = b.HostIP || '0.0.0.0';
    return `${ip}:${b.HostPort}`;
  }

  $: bindings = map?.Bindings ?? [];
  $: hostNet = map?.HostNetwork ?? [];
  $: flagged = bindings.filter((b) => b.Flagged);
  $: empty = bindings.length === 0 && hostNet.length === 0;
</script>

<div class="exposure">
  {#if status === 'loading'}
    <LoadingState label="Computing exposure…" />
  {:else if status === 'error'}
    <StateMessage
      tone="danger"
      icon="alert"
      title="Could not compute exposure"
      message={error}
    />
  {:else if empty}
    <StateMessage
      icon="exposure"
      title="Nothing published"
      message="No container on this host publishes a port, and none use host networking."
    />
  {:else}
    {#if flagged.length > 0}
      <div class="callout" role="alert">
        <strong>{flagged.length}</strong> binding{flagged.length === 1
          ? ''
          : 's'} reach
        <strong>all interfaces</strong> on a remotely-reached host — plausibly reachable
        from outside the host.
      </div>
    {/if}

    <div class="body">
      {#if bindings.length > 0}
        <table class="dd-table">
          <thead>
            <tr>
              <th>Reach</th>
              <th>Published</th>
              <th>Container port</th>
              <th>Proto</th>
              <th>Container</th>
            </tr>
          </thead>
          <tbody>
            {#each bindings as b (b.ContainerID + b.HostIP + b.HostPort + b.Protocol)}
              <tr class:flagged={b.Flagged}>
                <td>
                  <span class="badge {reachTone[b.Reach] ?? 'warn'}"
                    >{reachLabel[b.Reach] ?? b.Reach}</span
                  >
                </td>
                <td class="mono">{source(b)}</td>
                <td class="mono">{b.ContainerPort}</td>
                <td class="mono">{b.Protocol}</td>
                <td>{b.ContainerName}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      {/if}

      {#if hostNet.length > 0}
        <div class="hostnet">
          <h3>Host networking</h3>
          <p class="note">
            These containers share the host's network namespace — their exposure
            is <em>not derivable from port bindings</em>.
          </p>
          <ul>
            {#each hostNet as h (h.ContainerID)}
              <li>{h.ContainerName}</li>
            {/each}
          </ul>
        </div>
      {/if}

      <p class="scope">
        Reach is classified at the daemon layer only — an upstream cloud
        security group or host firewall is invisible here. This reports the
        binding, not the host's actual reachable addresses.
      </p>
    </div>
  {/if}
</div>

<style>
  .exposure {
    display: flex;
    flex-direction: column;
    height: 100%;
  }
  .callout {
    flex: none;
    margin: var(--space-4) var(--space-5) 0;
    padding: var(--space-3) var(--space-4);
    border: 1px solid var(--color-danger);
    border-radius: var(--radius-md);
    background: var(--color-danger-soft);
    color: var(--color-danger);
    font-size: var(--text-sm);
  }
  .body {
    flex: 1;
    min-height: 0;
    overflow: auto;
    padding: var(--space-4) var(--space-5);
  }
  tr.flagged {
    background: var(--color-danger-soft);
  }
  .badge {
    display: inline-block;
    padding: 1px 8px;
    border-radius: 999px;
    font-size: var(--text-xs);
    border: 1px solid var(--color-border-strong);
  }
  .badge.ok {
    color: var(--color-ok);
    border-color: color-mix(in srgb, var(--color-ok) 45%, transparent);
  }
  .badge.warn {
    color: var(--color-warn);
    border-color: color-mix(in srgb, var(--color-warn) 45%, transparent);
  }
  .badge.danger {
    color: var(--color-danger);
    border-color: color-mix(in srgb, var(--color-danger) 45%, transparent);
  }
  .hostnet {
    margin-top: var(--space-5);
  }
  .hostnet h3 {
    margin: 0 0 var(--space-1);
    font-size: var(--text-md);
  }
  .hostnet ul {
    margin: var(--space-2) 0 0;
    padding-left: var(--space-5);
  }
  .note {
    margin: 0;
    color: var(--color-warn);
    font-size: var(--text-sm);
  }
  .scope {
    margin: var(--space-5) 0 0;
    color: var(--color-text-faint);
    font-size: var(--text-xs);
  }
</style>
