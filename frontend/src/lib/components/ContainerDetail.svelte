<script lang="ts">
  // A bottom drawer for a container: live stats and a streamed log tail
  // (PROJECT-BOOK §7.11.3/4). Streams start on mount and are torn down on
  // destroy, so no backend stream is left running.
  import { onMount, onDestroy, tick } from 'svelte';
  import Icon from './Icon.svelte';
  import StateBadge from './StateBadge.svelte';
  import { formatBytes } from '../util/format';
  import {
    onLogLine,
    onStatsSample,
    stopLogs,
    stopStats,
    streamLogs,
    streamStats,
    type StatsSample,
  } from '../api/operations';
  import type { Container } from '../api/engine';

  export let hostId: string;
  export let container: Container;

  const MAX_LINES = 2000;
  let lines: string[] = [];
  let stats: StatsSample | null = null;
  let logEl: HTMLDivElement;
  let unsubscribes: Array<() => void> = [];

  onMount(() => {
    unsubscribes.push(
      onLogLine(container.ID, (line) => {
        lines = [...lines.slice(-(MAX_LINES - 1)), line];
        autoscroll();
      }),
    );
    unsubscribes.push(onStatsSample(container.ID, (s) => (stats = s)));
    void streamLogs(hostId, container.ID);
    void streamStats(hostId, container.ID);
  });

  onDestroy(() => {
    unsubscribes.forEach((u) => u());
    void stopLogs(container.ID);
    void stopStats(container.ID);
  });

  async function autoscroll() {
    await tick();
    if (logEl) logEl.scrollTop = logEl.scrollHeight;
  }
</script>

<section class="detail">
  <header>
    <div class="title">
      <span class="name">{container.Name}</span>
      <StateBadge state={container.State} />
      <span class="image">{container.Image}</span>
    </div>
    <div class="metrics">
      {#if stats}
        <span class="metric">CPU <b>{stats.cpuPct.toFixed(1)}%</b></span>
        <span class="metric">MEM <b>{formatBytes(stats.memBytes)}</b></span>
        <span class="metric"
          >NET <b>↓{formatBytes(stats.netRx)} ↑{formatBytes(stats.netTx)}</b
          ></span
        >
      {:else}
        <span class="metric muted">awaiting stats…</span>
      {/if}
    </div>
  </header>

  <div class="logs" bind:this={logEl}>
    {#if lines.length === 0}
      <p class="empty">
        <Icon name="history" size={16} /> Waiting for log output…
      </p>
    {:else}
      {#each lines as line, i (i)}<div class="line">{line}</div>{/each}
    {/if}
  </div>
</section>

<style>
  .detail {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: var(--color-surface);
    border-top: 1px solid var(--color-border-strong);
  }

  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-4);
    padding: var(--space-3) var(--space-5);
    border-bottom: 1px solid var(--color-border);
    flex: none;
  }

  .title {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    min-width: 0;
  }

  .name {
    font-weight: 600;
  }

  .image {
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    color: var(--color-text-faint);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .metrics {
    display: flex;
    gap: var(--space-4);
    flex: none;
  }

  .metric {
    font-size: var(--text-xs);
    color: var(--color-text-muted);
    font-variant-numeric: tabular-nums;
  }
  .metric b {
    color: var(--color-text);
    font-weight: 600;
  }
  .metric.muted {
    color: var(--color-text-faint);
  }

  .logs {
    flex: 1;
    min-height: 0;
    overflow: auto;
    padding: var(--space-3) var(--space-5);
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    line-height: 1.55;
    background: var(--color-bg);
  }

  .line {
    white-space: pre-wrap;
    word-break: break-word;
    color: var(--color-text);
  }

  .empty {
    display: flex;
    align-items: center;
    gap: 8px;
    color: var(--color-text-faint);
  }
</style>
