<script lang="ts">
  // Operation-history browser (PROJECT-BOOK §7.11.8): past operations for the
  // active host with target, options, result, and — for prune — what was
  // reclaimed. Filterable by kind and to destructive operations only; the whole
  // record is exportable.
  import StateMessage from '../components/states/StateMessage.svelte';
  import LoadingState from '../components/states/LoadingState.svelte';
  import {
    operationHistory,
    downloadJournalExport,
    type Operation,
  } from '../api/journal';
  import { formatBytes, formatTimestamp } from '../util/format';

  export let hostId: string;

  // Filter options; each value matches a Go OperationKind so the store filters
  // server-side from the same vocabulary.
  const KINDS: ReadonlyArray<{ value: string; label: string }> = [
    { value: '', label: 'All kinds' },
    { value: 'container.start', label: 'Start' },
    { value: 'container.stop', label: 'Stop' },
    { value: 'container.restart', label: 'Restart' },
    { value: 'container.kill', label: 'Kill' },
    { value: 'container.remove', label: 'Remove' },
    { value: 'container.exec', label: 'Exec' },
    { value: 'image.prune', label: 'Image prune' },
    { value: 'container.prune', label: 'Container prune' },
    { value: 'buildcache.prune', label: 'Build-cache prune' },
    { value: 'volume.remove', label: 'Volume remove' },
    { value: 'compose.up', label: 'Compose up' },
    { value: 'compose.down', label: 'Compose down' },
  ];

  let kind = '';
  let destructiveOnly = false;
  let status: 'loading' | 'ready' | 'error' = 'loading';
  let ops: Operation[] = [];
  let error = '';
  let exporting = false;

  async function load() {
    status = 'loading';
    error = '';
    try {
      ops = (await operationHistory(hostId, kind, destructiveOnly, 200)) ?? [];
      status = 'ready';
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
      status = 'error';
    }
  }

  // Reload whenever the host or a filter changes.
  let lastKey = '';
  $: {
    const next = `${hostId}|${kind}|${destructiveOnly}`;
    if (next !== lastKey) {
      lastKey = next;
      void load();
    }
  }

  async function onExport() {
    exporting = true;
    try {
      await downloadJournalExport();
    } finally {
      exporting = false;
    }
  }

  function reclaimed(op: Operation): string {
    return op.BytesReclaimed > 0 ? formatBytes(op.BytesReclaimed) : '';
  }
  function options(op: Operation): string {
    const set = op.OptionSet ?? {};
    const keys = Object.keys(set);
    if (keys.length === 0) return '';
    return keys.map((k) => `${k}=${set[k]}`).join(' ');
  }
</script>

<div class="history">
  <div class="toolbar">
    <label
      >Kind
      <select bind:value={kind}>
        {#each KINDS as k (k.value)}
          <option value={k.value}>{k.label}</option>
        {/each}
      </select>
    </label>
    <label class="check">
      <input type="checkbox" bind:checked={destructiveOnly} />
      Destructive only
    </label>
    <span class="spacer"></span>
    <button class="export" disabled={exporting} on:click={onExport}
      >{exporting ? 'Exporting…' : 'Export…'}</button
    >
  </div>

  <div class="body">
    {#if status === 'loading'}
      <LoadingState label="Loading history…" />
    {:else if status === 'error'}
      <StateMessage
        tone="danger"
        icon="alert"
        title="Could not load history"
        message={error}
      />
    {:else if ops.length === 0}
      <StateMessage
        icon="history"
        title="No operations yet"
        message="Operations performed through Drydock on this host are recorded here."
      />
    {:else}
      <table class="dd-table">
        <thead>
          <tr>
            <th>When</th>
            <th>Kind</th>
            <th>Target</th>
            <th>Options</th>
            <th>Result</th>
            <th>Reclaimed</th>
          </tr>
        </thead>
        <tbody>
          {#each ops as op (op.ID)}
            <tr>
              <td class="when">{formatTimestamp(op.StartedAt)}</td>
              <td class="mono">{op.Kind}</td>
              <td class="mono">{op.Target}</td>
              <td class="mono opts">{options(op)}</td>
              <td>
                <span class="result" class:ok={op.Result === 'ok'}
                  >{op.Result}</span
                >
              </td>
              <td class="mono">{reclaimed(op)}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </div>
</div>

<style>
  .history {
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  .toolbar {
    flex: none;
    display: flex;
    align-items: center;
    gap: var(--space-4);
    padding: var(--space-3) var(--space-5);
    border-bottom: 1px solid var(--color-border);
    font-size: var(--text-sm);
    color: var(--color-text-muted);
  }
  .toolbar label {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
  }
  .toolbar .check {
    cursor: pointer;
  }
  select {
    background: var(--color-surface-raised);
    color: var(--color-text);
    border: 1px solid var(--color-border-strong);
    border-radius: var(--radius-sm);
    padding: 3px 6px;
    font-size: var(--text-sm);
  }
  .spacer {
    flex: 1;
  }
  .export {
    padding: 4px 12px;
    border: 1px solid var(--color-border-strong);
    border-radius: var(--radius-sm);
    background: var(--color-surface-raised);
    color: var(--color-text);
    font-size: var(--text-xs);
  }
  .export:hover:not(:disabled) {
    border-color: var(--color-accent);
    background: var(--color-surface-hover);
  }
  .export:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .body {
    flex: 1;
    min-height: 0;
    overflow: auto;
  }

  .when {
    white-space: nowrap;
    color: var(--color-text-muted);
  }
  .opts {
    color: var(--color-text-muted);
    max-width: 22ch;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .result {
    color: var(--color-warn);
  }
  .result.ok {
    color: var(--color-ok);
  }
</style>
