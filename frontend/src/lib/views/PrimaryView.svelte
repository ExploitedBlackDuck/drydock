<script lang="ts">
  // Renders the active view for the active host, resolving the four states every
  // view must define (PROJECT-BOOK §7.11.9): loading, error, degraded, and the
  // connected empty/data state. Live object data arrives once the Engine port is
  // wired (P2); until then a connected host shows its empty state and a
  // not-yet-connected host shows the matching disconnected/error/degraded state.
  import ViewShell from './ViewShell.svelte';
  import StateMessage from '../components/states/StateMessage.svelte';
  import LoadingState from '../components/states/LoadingState.svelte';
  import ContainersView from './ContainersView.svelte';
  import ComposeView from './ComposeView.svelte';
  import ImagesView from './ImagesView.svelte';
  import VolumesView from './VolumesView.svelte';
  import NetworksView from './NetworksView.svelte';
  import DiskView from './DiskView.svelte';
  import HistoryView from './HistoryView.svelte';
  import AuditView from './AuditView.svelte';
  import { VIEW_CONTENT } from './content';
  import { ViewId, VIEWS } from '../types/view';
  import { activeView } from '../stores/navigation';
  import { HostStatus, type Host } from '../types/domain';

  export let host: Host;

  $: meta = VIEWS.find((v) => v.id === $activeView)!;
  $: content = VIEW_CONTENT[$activeView];
  $: connected =
    host.status === HostStatus.Connected || host.status === HostStatus.Degraded;
</script>

<ViewShell title={meta.label} description={content.description}>
  {#if host.status === HostStatus.Connecting}
    <LoadingState label={`Connecting to ${host.name}…`} />
  {:else if host.status === HostStatus.Error}
    <StateMessage
      tone="danger"
      icon="alert"
      title="Connection failed"
      message={`Drydock could not reach ${host.name}. Check the transport, credentials, and that the host is reachable.`}
    />
  {:else if host.status === HostStatus.Disconnected}
    <StateMessage
      icon={content.icon}
      title="Host disconnected"
      message={`${host.name} is not connected. Live ${meta.label.toLowerCase()} load once a transport to the host is established.`}
    />
  {:else}
    {#if host.status === HostStatus.Degraded}
      <div class="degraded" role="note">
        Reduced-capability mode: this host's engine is below Drydock's minimum
        supported API version, so some features are unavailable.
      </div>
    {/if}

    {#if connected && $activeView === ViewId.Containers}
      <ContainersView hostId={host.id} observeMode={host.observeMode} />
    {:else if connected && $activeView === ViewId.Compose}
      <ComposeView hostId={host.id} observeMode={host.observeMode} />
    {:else if connected && $activeView === ViewId.Images}
      <ImagesView hostId={host.id} />
    {:else if connected && $activeView === ViewId.Volumes}
      <VolumesView hostId={host.id} observeMode={host.observeMode} />
    {:else if connected && $activeView === ViewId.Networks}
      <NetworksView hostId={host.id} />
    {:else if connected && $activeView === ViewId.Disk}
      <DiskView hostId={host.id} observeMode={host.observeMode} />
    {:else if connected && $activeView === ViewId.History}
      <HistoryView hostId={host.id} />
    {:else if connected && $activeView === ViewId.Audit}
      <AuditView />
    {:else}
      <StateMessage
        icon={content.icon}
        title={content.emptyTitle}
        message={content.emptyMessage}
      />
    {/if}
  {/if}
</ViewShell>

<style>
  .degraded {
    margin: var(--space-4) var(--space-5) 0;
    padding: var(--space-3) var(--space-4);
    border: 1px solid var(--color-warn);
    border-radius: var(--radius-md);
    background: rgba(219, 154, 58, 0.12);
    color: var(--color-warn);
    font-size: var(--text-sm);
  }
</style>
