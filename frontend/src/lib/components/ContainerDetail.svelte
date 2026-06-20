<script lang="ts">
  // A bottom drawer for a container: live stats and a streamed log tail
  // (PROJECT-BOOK §7.11.3/4). Streams start on mount and are torn down on
  // destroy, so no backend stream is left running.
  import { onMount, onDestroy, tick } from 'svelte';
  import Icon from './Icon.svelte';
  import StateBadge from './StateBadge.svelte';
  import Sparkline from './Sparkline.svelte';
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
  import { getResourceHistory } from '../api/dashboard';
  import ExecTerminal from './ExecTerminal.svelte';
  import type { Container } from '../api/engine';

  export let hostId: string;
  export let container: Container;
  export let observeMode = false;

  // The detail body switches between the streamed logs and an interactive shell
  // (PROJECT-BOOK §7.11.4). Exec is a mutation, so it is unavailable on
  // observe-only hosts.
  type Tab = 'logs' | 'shell';
  let tab: Tab = 'logs';
  $: running = container.State === 'running';
  $: canShell = running && !observeMode;

  const MAX_LINES = 2000;
  const MAX_HISTORY = 120;
  let lines: string[] = [];
  let stats: StatsSample | null = null;
  let cpuSeries: number[] = [];
  let logEl: HTMLDivElement;
  let unsubscribes: Array<() => void> = [];

  onMount(() => {
    // Seed the chart with persisted rolling history, then extend it live.
    getResourceHistory(hostId, container.ID, MAX_HISTORY)
      .then((samples) => (cpuSeries = (samples ?? []).map((s) => s.CPUPct)))
      .catch(() => {});

    unsubscribes.push(
      onLogLine(container.ID, (line) => {
        lines = [...lines.slice(-(MAX_LINES - 1)), line];
        autoscroll();
      }),
    );
    unsubscribes.push(
      onStatsSample(container.ID, (s) => {
        stats = s;
        cpuSeries = [...cpuSeries.slice(-(MAX_HISTORY - 1)), s.cpuPct];
      }),
    );
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
        <span class="spark" title="CPU history"
          ><Sparkline values={cpuSeries} /></span
        >
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

  <div class="tabs" role="tablist">
    <button
      class="tab"
      class:active={tab === 'logs'}
      role="tab"
      aria-selected={tab === 'logs'}
      on:click={() => (tab = 'logs')}>Logs</button
    >
    <button
      class="tab"
      class:active={tab === 'shell'}
      role="tab"
      aria-selected={tab === 'shell'}
      disabled={!canShell}
      title={canShell
        ? 'Open an interactive shell'
        : observeMode
          ? 'Host is observe-only'
          : 'Container is not running'}
      on:click={() => (tab = 'shell')}>Shell</button
    >
  </div>

  {#if tab === 'logs'}
    <div class="logs" bind:this={logEl}>
      {#if lines.length === 0}
        <p class="empty">
          <Icon name="history" size={16} /> Waiting for log output…
        </p>
      {:else}
        {#each lines as line, i (i)}<div class="line">{line}</div>{/each}
      {/if}
    </div>
  {:else if canShell}
    {#key container.ID}
      <ExecTerminal {hostId} containerId={container.ID} />
    {/key}
  {/if}
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

  .spark {
    display: inline-flex;
    align-items: center;
    opacity: 0.9;
  }

  .tabs {
    flex: none;
    display: flex;
    gap: 2px;
    padding: 0 var(--space-5);
    border-bottom: 1px solid var(--color-border);
  }
  .tab {
    padding: 6px 12px;
    border: none;
    border-bottom: 2px solid transparent;
    background: transparent;
    color: var(--color-text-muted);
    font-size: var(--text-sm);
  }
  .tab:hover:not(:disabled) {
    color: var(--color-text);
  }
  .tab.active {
    color: var(--color-text);
    border-bottom-color: var(--color-accent);
  }
  .tab:disabled {
    opacity: 0.4;
    cursor: not-allowed;
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
