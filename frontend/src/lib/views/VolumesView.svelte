<script lang="ts">
  import ResourceView from './ResourceView.svelte';
  import { volumes } from '../stores/objects';
  import { formatBytes } from '../util/format';
  import { removeVolume } from '../api/prune';
  import {
    previewVolumeSnapshot,
    snapshotVolume,
    type VolumeSnapshotPreview,
  } from '../api/snapshot';

  export let hostId: string;
  export let observeMode = false;

  let busy = '';
  let error: string | null = null;

  // The offered snapshot safeguard (ADR-0020): preview the cost, then capture.
  // It never blocks deletion and is a mutation, so it is observe-mode-aware.
  let preview: VolumeSnapshotPreview | null = null;
  let snapBusy = false;
  let snapResult = '';

  async function onSnapshot(name: string) {
    error = null;
    snapResult = '';
    try {
      preview = await previewVolumeSnapshot(hostId, name);
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    }
  }

  async function confirmSnapshot() {
    if (!preview) return;
    snapBusy = true;
    try {
      snapResult = await snapshotVolume(hostId, preview.Volume);
      preview = null;
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
      preview = null;
    } finally {
      snapBusy = false;
    }
  }

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
  {#if snapResult}
    <p class="ok" role="status">
      Snapshot written to {snapResult} — a plaintext archive; keep it somewhere safe.
    </p>
  {/if}
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
          <td class="actions">
            <button
              disabled={observeMode || busy === v.Name}
              title={observeMode
                ? 'Host is observe-only'
                : 'Capture this volume to a tar file (offered, never required)'}
              on:click={() => onSnapshot(v.Name)}
            >
              Snapshot…
            </button>
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

{#if preview}
  <div class="backdrop">
    <div class="dialog" role="dialog" aria-label="Volume snapshot">
      <h3>Snapshot “{preview.Volume}”</h3>
      <dl>
        <dt>Destination</dt>
        <dd class="mono">{preview.Destination}</dd>
        <dt>Estimated size</dt>
        <dd>{formatBytes(preview.EstimatedBytes)}</dd>
        <dt>Estimated time</dt>
        <dd>~{preview.EstimatedSecond}s</dd>
      </dl>
      <p class="warn">
        The archive is plaintext, outside Drydock's encrypted store — protect
        the destination. This does not delete the volume.
      </p>
      <div class="dialog-actions">
        <button disabled={snapBusy} on:click={() => (preview = null)}
          >Cancel</button
        >
        <button class="primary" disabled={snapBusy} on:click={confirmSnapshot}>
          {snapBusy ? 'Capturing…' : 'Snapshot'}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .err {
    margin: 0;
    padding: var(--space-2) var(--space-5);
    color: var(--color-danger);
    background: var(--color-danger-soft);
    font-size: var(--text-sm);
  }
  .ok {
    margin: 0;
    padding: var(--space-2) var(--space-5);
    color: var(--color-text-muted);
    background: var(--color-surface-raised);
    font-size: var(--text-sm);
    font-family: var(--font-mono);
    word-break: break-all;
  }
  .actions-col {
    width: 1%;
  }
  .actions {
    display: flex;
    gap: 6px;
    white-space: nowrap;
  }
  .actions button {
    padding: 3px 9px;
    border: 1px solid var(--color-border-strong);
    border-radius: var(--radius-sm);
    background: var(--color-surface-raised);
    color: var(--color-text);
    font-size: var(--text-xs);
  }
  .actions button:hover:not(:disabled) {
    border-color: var(--color-accent);
  }
  .backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 50;
  }
  .dialog {
    width: min(520px, 92vw);
    background: var(--color-surface);
    border: 1px solid var(--color-border-strong);
    border-radius: var(--radius-md);
    padding: var(--space-5);
  }
  .dialog h3 {
    margin: 0 0 var(--space-3);
    font-size: var(--text-md);
  }
  .dialog dl {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 4px var(--space-4);
    margin: 0 0 var(--space-3);
    font-size: var(--text-sm);
  }
  .dialog dt {
    color: var(--color-text-muted);
  }
  .dialog dd {
    margin: 0;
    word-break: break-all;
  }
  .dialog .warn {
    margin: 0 0 var(--space-4);
    color: var(--color-warn);
    font-size: var(--text-sm);
  }
  .dialog-actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--space-3);
  }
  .dialog-actions button {
    padding: 5px 14px;
    border: 1px solid var(--color-border-strong);
    border-radius: var(--radius-sm);
    background: var(--color-surface-raised);
    color: var(--color-text);
    font-size: var(--text-sm);
  }
  .dialog-actions button.primary {
    border-color: var(--color-accent);
    background: var(--color-accent-soft);
  }
  .dialog-actions button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
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
