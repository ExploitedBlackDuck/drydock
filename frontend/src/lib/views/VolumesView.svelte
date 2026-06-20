<script lang="ts">
  import ResourceView from './ResourceView.svelte';
  import { volumes } from '../stores/objects';
  import { formatBytes } from '../util/format';
  import { removeVolume } from '../api/prune';

  export let hostId: string;
  export let observeMode = false;

  let busy = '';
  let error: string | null = null;

  async function onRemove(name: string, inUse: boolean) {
    const extra = inUse
      ? '\n\nThis volume is IN USE — its data will be lost.'
      : '';
    if (
      !window.confirm(
        `Remove volume "${name}"?${extra}\n\nThis cannot be undone.`,
      )
    )
      return;
    busy = name;
    error = null;
    try {
      // Each volume is confirmed and removed individually — never in bulk (§7.4).
      await removeVolume(hostId, name, true);
      await volumes.load(hostId);
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      busy = '';
    }
  }
</script>

<ResourceView
  {hostId}
  store={volumes}
  icon="volumes"
  emptyTitle="No volumes"
  emptyMessage="This host has no volumes. Each volume is confirmed individually before removal."
  let:rows
>
  {#if error}<p class="err" role="alert">{error}</p>{/if}
  <table class="dd-table">
    <thead>
      <tr>
        <th>Name</th>
        <th>Driver</th>
        <th class="num">Size</th>
        <th>Status</th>
        <th>Mountpoint</th>
        <th class="actions-col">Action</th>
      </tr>
    </thead>
    <tbody>
      {#each rows as v (v.Name)}
        <tr>
          <td>{v.Name}</td>
          <td class="muted">{v.Driver}</td>
          <td class="num">{formatBytes(v.Size)}</td>
          <td class="muted">{v.InUse ? 'in use' : 'unused'}</td>
          <td class="mono">{v.Mountpoint}</td>
          <td>
            <button
              class="danger"
              disabled={observeMode || busy === v.Name}
              title={observeMode ? 'Host is observe-only' : ''}
              on:click={() => onRemove(v.Name, v.InUse)}
            >
              Remove
            </button>
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
</ResourceView>

<style>
  .err {
    margin: 0;
    padding: var(--space-2) var(--space-5);
    color: var(--color-danger);
    background: var(--color-danger-soft);
    font-size: var(--text-sm);
  }
  .actions-col {
    width: 1%;
  }
  .danger {
    padding: 3px 9px;
    border: 1px solid var(--color-border-strong);
    border-radius: var(--radius-sm);
    background: var(--color-surface-raised);
    color: var(--color-text);
    font-size: var(--text-xs);
  }
  .danger:hover:not(:disabled) {
    border-color: var(--color-danger);
    color: var(--color-danger);
  }
  .danger:disabled {
    opacity: 0.45;
    cursor: not-allowed;
  }
</style>
