<script lang="ts">
  // Host timeline (PROJECT-BOOK §7.12.4, ADR-0018): mapped engine events
  // interleaved with references to Drydock's audit log — best-effort and
  // labelled, distinct from the audit log's completeness guarantee. Stream gaps
  // and host-vs-desktop clock skew are surfaced, not hidden.
  import StateMessage from '../components/states/StateMessage.svelte';
  import LoadingState from '../components/states/LoadingState.svelte';
  import {
    getHostTimeline,
    type HostTimeline,
    type TimelineEntry,
  } from '../api/timeline';
  import { formatTimestamp } from '../util/format';

  export let hostId: string;

  let status: 'loading' | 'ready' | 'error' = 'loading';
  let data: HostTimeline | null = null;
  let error = '';

  async function load() {
    status = 'loading';
    error = '';
    try {
      data = await getHostTimeline(hostId, 300);
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

  // Show newest first.
  $: entries = [...(data?.entries ?? [])].reverse();

  function tone(e: TimelineEntry): 'ok' | 'warn' | 'danger' | 'neutral' {
    if (e.Source === 'audit') return 'neutral';
    if (e.Kind === 'die' && (e.ExitCode ?? 0) !== 0) return 'danger';
    if (e.Kind === 'oom' || e.HealthStatus === 'unhealthy') return 'danger';
    if (e.Kind === 'stream-gap') return 'warn';
    if (e.Kind === 'start' || e.HealthStatus === 'healthy') return 'ok';
    return 'neutral';
  }

  function detail(e: TimelineEntry): string {
    if (e.ExitCode !== undefined && e.ExitCode !== null)
      return `exit ${e.ExitCode}`;
    if (e.HealthStatus) return e.HealthStatus;
    return '';
  }
</script>

<div class="timeline">
  {#if data && data.skewSeconds > 0}
    <div class="skew" role="note">
      Clock skew of ~{Math.round(data.skewSeconds)}s between this host and the
      desktop — ordering of engine vs. audited events may be approximate.
    </div>
  {/if}

  <div class="body">
    {#if status === 'loading'}
      <LoadingState label="Loading timeline…" />
    {:else if status === 'error'}
      <StateMessage
        tone="danger"
        icon="alert"
        title="Could not load timeline"
        message={error}
      />
    {:else if entries.length === 0}
      <StateMessage
        icon="timeline"
        title="No timeline yet"
        message="Engine events and audited actions for this host appear here as they occur."
      />
    {:else}
      <ul class="rows">
        {#each entries as e, i (i)}
          <li class="row {tone(e)}">
            <span class="when">{formatTimestamp(e.At)}</span>
            <span class="src {e.Source}">{e.Source}</span>
            <span class="kind">{e.Kind}</span>
            <span class="subject">{e.Subject}</span>
            <span class="meta">{detail(e)}</span>
          </li>
        {/each}
      </ul>
    {/if}
  </div>
</div>

<style>
  .timeline {
    display: flex;
    flex-direction: column;
    height: 100%;
  }
  .skew {
    flex: none;
    margin: var(--space-4) var(--space-5) 0;
    padding: var(--space-2) var(--space-4);
    border: 1px solid var(--color-warn);
    border-radius: var(--radius-md);
    background: rgba(219, 154, 58, 0.12);
    color: var(--color-warn);
    font-size: var(--text-sm);
  }
  .body {
    flex: 1;
    min-height: 0;
    overflow: auto;
    padding: var(--space-3) var(--space-5);
  }
  .rows {
    list-style: none;
    margin: 0;
    padding: 0;
  }
  .row {
    display: grid;
    grid-template-columns: 13rem 5rem 7rem 1fr auto;
    gap: var(--space-3);
    align-items: baseline;
    padding: 5px 0;
    border-bottom: 1px solid var(--color-border);
    font-size: var(--text-sm);
  }
  .when {
    color: var(--color-text-muted);
    white-space: nowrap;
    font-variant-numeric: tabular-nums;
  }
  .src {
    font-size: var(--text-xs);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--color-text-faint);
  }
  .src.audit {
    color: var(--color-accent);
  }
  .kind {
    font-family: var(--font-mono);
    font-size: var(--text-xs);
  }
  .subject {
    color: var(--color-text);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .meta {
    font-size: var(--text-xs);
    color: var(--color-text-muted);
    white-space: nowrap;
  }
  .row.danger .kind {
    color: var(--color-danger);
  }
  .row.warn .kind {
    color: var(--color-warn);
  }
  .row.ok .kind {
    color: var(--color-ok);
  }
</style>
