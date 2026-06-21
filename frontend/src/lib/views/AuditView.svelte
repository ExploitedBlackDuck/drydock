<script lang="ts">
  // Audit-log view (PROJECT-BOOK §7.11.8): the append-only, HMAC-keyed-chained
  // record of every consequential action, with a chain-verification indicator
  // distinguishing intact / in-place-tampered / truncated / key-unavailable
  // (ADR-0025) and export. The audit log spans all hosts, so it is not
  // host-scoped.
  import { onMount } from 'svelte';
  import StateMessage from '../components/states/StateMessage.svelte';
  import LoadingState from '../components/states/LoadingState.svelte';
  import {
    auditTrail,
    downloadJournalExport,
    type AuditStatus,
  } from '../api/journal';
  import { formatTimestamp } from '../util/format';

  let status: 'loading' | 'ready' | 'error' = 'loading';
  let trail: AuditStatus | null = null;
  let error = '';
  let exporting = false;

  async function load() {
    status = 'loading';
    error = '';
    try {
      trail = await auditTrail();
      status = 'ready';
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
      status = 'error';
    }
  }

  onMount(load);

  async function onExport() {
    exporting = true;
    try {
      await downloadJournalExport();
    } finally {
      exporting = false;
    }
  }

  $: entries = trail?.Entries ?? [];

  // The four chain-verification states (ADR-0025), each with its own cue. The
  // indicator states its guarantee honestly: tamper-evident, keyed, and
  // truncation-aware — not a claim of tamper-proofness.
  type ChainView = {
    tone: 'ok' | 'warn' | 'danger';
    role: string;
    label: string;
  };
  function chainView(t: AuditStatus): ChainView {
    switch (t.State) {
      case 'intact':
        return {
          tone: 'ok',
          role: 'status',
          label: `Chain intact · ${t.VerifiedCount} entries verified`,
        };
      case 'truncated':
        return {
          tone: 'danger',
          role: 'alert',
          label: `Truncated · ${t.VerifiedCount} entries before the tail was removed`,
        };
      case 'key_unavailable':
        return {
          tone: 'warn',
          role: 'status',
          label: `Key unavailable · structure checked (${t.VerifiedCount}), authenticity not confirmed`,
        };
      default: // in_place_tampered
        return {
          tone: 'danger',
          role: 'alert',
          label: `Tampering detected · verified ${t.VerifiedCount} before a break`,
        };
    }
  }
</script>

<div class="audit">
  <div class="toolbar">
    {#if trail}
      {@const cv = chainView(trail)}
      <span
        class="chip {cv.tone}"
        role={cv.role}
        title="Tamper-evident, keyed, truncation-aware — not a claim of tamper-proofness"
      >
        <span class="dot" aria-hidden="true"></span>
        {cv.label}
      </span>
    {/if}
    <span class="spacer"></span>
    <button
      class="export"
      disabled={exporting || status !== 'ready'}
      on:click={onExport}>{exporting ? 'Exporting…' : 'Export…'}</button
    >
  </div>

  {#if !trail?.Verified && trail?.Error}
    <p class="break" role="alert">{trail.Error}</p>
  {/if}

  <div class="body">
    {#if status === 'loading'}
      <LoadingState label="Loading audit log…" />
    {:else if status === 'error'}
      <StateMessage
        tone="danger"
        icon="alert"
        title="Could not load audit log"
        message={error}
      />
    {:else if entries.length === 0}
      <StateMessage
        icon="audit"
        title="No audit entries"
        message="Connecting a host and acting on it writes tamper-evident entries here."
      />
    {:else}
      <table class="dd-table">
        <thead>
          <tr>
            <th>#</th>
            <th>When</th>
            <th>Action</th>
            <th>Host</th>
            <th>Subject</th>
          </tr>
        </thead>
        <tbody>
          {#each entries as e (e.Seq)}
            <tr>
              <td class="mono seq">{e.Seq}</td>
              <td class="when">{formatTimestamp(e.At)}</td>
              <td class="mono">{e.Action}</td>
              <td class="mono">{e.HostRef || '—'}</td>
              <td class="mono">{e.Subject || '—'}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </div>
</div>

<style>
  .audit {
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
  }
  .spacer {
    flex: 1;
  }

  .chip {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    padding: 3px 10px;
    border-radius: 999px;
    font-size: var(--text-xs);
    font-weight: 500;
    border: 1px solid var(--color-border-strong);
  }
  .chip .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
  }
  .chip.ok {
    color: var(--color-ok);
    border-color: color-mix(in srgb, var(--color-ok) 45%, transparent);
  }
  .chip.ok .dot {
    background: var(--color-ok);
  }
  .chip.danger {
    color: var(--color-danger);
    border-color: color-mix(in srgb, var(--color-danger) 45%, transparent);
  }
  .chip.danger .dot {
    background: var(--color-danger);
  }
  .chip.warn {
    color: var(--color-warn);
    border-color: color-mix(in srgb, var(--color-warn) 45%, transparent);
  }
  .chip.warn .dot {
    background: var(--color-warn);
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

  .break {
    margin: 0;
    padding: var(--space-2) var(--space-5);
    color: var(--color-danger);
    background: var(--color-danger-soft);
    font-size: var(--text-sm);
  }

  .body {
    flex: 1;
    min-height: 0;
    overflow: auto;
  }
  .seq {
    color: var(--color-text-faint);
  }
  .when {
    white-space: nowrap;
    color: var(--color-text-muted);
  }
</style>
