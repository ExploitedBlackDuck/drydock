<script lang="ts">
  // The disk dashboard (PROJECT-BOOK §7.6/§7.7): `system df` made legible as
  // per-category reclaimable space — build cache first-class — each routed into
  // a preview-and-confirm prune. Volumes are listed individually and removed one
  // at a time; never bulk (§7.4).
  import { onMount } from 'svelte';
  import Icon from '../components/Icon.svelte';
  import LoadingState from '../components/states/LoadingState.svelte';
  import StateMessage from '../components/states/StateMessage.svelte';
  import { formatBytes } from '../util/format';
  import {
    getPruneImpact,
    pruneBuildCache,
    pruneContainers,
    pruneImages,
    removeVolume,
    type PruneCategory,
    type PruneImpact,
  } from '../api/prune';

  export let hostId: string;
  export let observeMode = false;

  let status: 'loading' | 'ready' | 'error' = 'loading';
  let impact: PruneImpact | null = null;
  let error: string | null = null;
  let busy = '';
  let loadedFor: string | null = null;

  $: if (hostId && hostId !== loadedFor) {
    loadedFor = hostId;
    void load();
  }
  onMount(() => {
    if (loadedFor === null && hostId) {
      loadedFor = hostId;
      void load();
    }
  });

  async function load() {
    status = 'loading';
    error = null;
    try {
      impact = await getPruneImpact(hostId);
      status = 'ready';
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
      status = 'error';
    }
  }

  function pruneFor(cat: PruneCategory): () => Promise<number> {
    switch (cat.Kind) {
      case 'stopped-containers':
        return () => pruneContainers(hostId, true);
      case 'dangling-images':
        return () => pruneImages(hostId, false, true);
      case 'unused-images':
        return () => pruneImages(hostId, true, true);
      case 'build-cache':
        return () => pruneBuildCache(hostId, true);
      default:
        return () => Promise.resolve(0);
    }
  }

  async function runPrune(cat: PruneCategory) {
    if (
      !window.confirm(
        `Prune ${cat.Label.toLowerCase()} (${cat.ObjectCount} object(s), ${formatBytes(cat.ReclaimableBytes)})? This cannot be undone.`,
      )
    )
      return;
    busy = cat.Kind;
    error = null;
    try {
      await pruneFor(cat)();
      await load();
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      busy = '';
    }
  }

  async function onRemoveVolume(name: string, inUse: boolean) {
    const extra = inUse
      ? '\n\nThis volume is IN USE — its data will be lost.'
      : '';
    if (
      !window.confirm(
        `Remove volume "${name}"?${extra}\n\nThis cannot be undone.`,
      )
    )
      return;
    busy = 'vol:' + name;
    error = null;
    try {
      await removeVolume(hostId, name, true);
      await load();
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      busy = '';
    }
  }
</script>

{#if status === 'loading'}
  <LoadingState label="Reading disk usage…" />
{:else if status === 'error'}
  <StateMessage
    tone="danger"
    icon="alert"
    title="Could not read disk usage"
    message={error ?? ''}
  />
{:else if impact}
  <div class="disk">
    {#if error}<p class="err" role="alert">{error}</p>{/if}

    <div class="summary">
      <span class="total-label">Reclaimable</span>
      <span class="total">{formatBytes(impact.TotalReclaimable)}</span>
      <span class="hint"
        >across categories below — volumes are confirmed individually</span
      >
    </div>

    <div class="cards">
      {#each impact.Categories as cat (cat.Kind)}
        <div class="card" class:feature={cat.Kind === 'build-cache'}>
          <div class="card-head">
            <span class="card-label">{cat.Label}</span>
            {#if cat.Kind === 'build-cache'}<span class="tag"
                >often largest</span
              >{/if}
          </div>
          <div class="card-bytes">{formatBytes(cat.ReclaimableBytes)}</div>
          <div class="card-count">
            {cat.ObjectCount} object{cat.ObjectCount === 1 ? '' : 's'}
          </div>
          <button
            class="prune"
            disabled={observeMode || cat.ObjectCount === 0 || busy === cat.Kind}
            title={observeMode ? 'Host is observe-only' : ''}
            on:click={() => runPrune(cat)}
          >
            {busy === cat.Kind ? 'Pruning…' : 'Prune'}
          </button>
        </div>
      {/each}
    </div>

    <div class="volumes">
      <h2>
        Volumes <span class="vh">— removed one at a time, never in bulk</span>
      </h2>
      {#if impact.Volumes.length === 0}
        <p class="muted">No volumes on this host.</p>
      {:else}
        <table class="dd-table">
          <thead>
            <tr
              ><th>Name</th><th class="num">Size</th><th>Status</th><th
                class="actions-col">Action</th
              ></tr
            >
          </thead>
          <tbody>
            {#each impact.Volumes as v (v.Name)}
              <tr>
                <td>{v.Name}</td>
                <td class="num">{formatBytes(v.Size)}</td>
                <td class="muted">
                  {#if v.InUse}<span class="inuse"
                      ><Icon name="alert" size={12} /> in use</span
                    >{:else}unused{/if}
                </td>
                <td>
                  <button
                    class="danger"
                    disabled={observeMode || busy === 'vol:' + v.Name}
                    title={observeMode ? 'Host is observe-only' : ''}
                    on:click={() => onRemoveVolume(v.Name, v.InUse)}
                  >
                    Remove
                  </button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      {/if}
    </div>
  </div>
{/if}

<style>
  .disk {
    padding: var(--space-5);
    overflow: auto;
    height: 100%;
  }

  .err {
    margin: 0 0 var(--space-4);
    padding: var(--space-2) var(--space-3);
    color: var(--color-danger);
    background: var(--color-danger-soft);
    border-radius: var(--radius-sm);
    font-size: var(--text-sm);
  }

  .summary {
    display: flex;
    align-items: baseline;
    gap: var(--space-3);
    margin-bottom: var(--space-5);
  }
  .total-label {
    font-size: var(--text-sm);
    color: var(--color-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
  .total {
    font-size: 1.8rem;
    font-weight: 650;
    font-variant-numeric: tabular-nums;
  }
  .hint {
    font-size: var(--text-xs);
    color: var(--color-text-faint);
  }

  .cards {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
    gap: var(--space-3);
    margin-bottom: var(--space-6);
  }
  .card {
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    padding: var(--space-4);
    background: var(--color-surface);
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .card.feature {
    border-color: var(--color-accent);
    background: var(--color-accent-soft);
  }
  .card-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-2);
  }
  .card-label {
    font-size: var(--text-sm);
    color: var(--color-text-muted);
  }
  .tag {
    font-size: 0.62rem;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--color-accent);
    border: 1px solid var(--color-accent);
    border-radius: 999px;
    padding: 1px 6px;
  }
  .card-bytes {
    font-size: var(--text-xl);
    font-weight: 650;
    font-variant-numeric: tabular-nums;
  }
  .card-count {
    font-size: var(--text-xs);
    color: var(--color-text-faint);
  }
  .prune {
    margin-top: var(--space-2);
    padding: 5px 10px;
    border: 1px solid var(--color-border-strong);
    border-radius: var(--radius-sm);
    background: var(--color-surface-raised);
    color: var(--color-text);
    font-size: var(--text-sm);
    align-self: flex-start;
  }
  .prune:hover:not(:disabled) {
    border-color: var(--color-danger);
    color: var(--color-danger);
  }
  .prune:disabled {
    opacity: 0.45;
    cursor: not-allowed;
  }

  .volumes h2 {
    font-size: var(--text-lg);
    font-weight: 600;
    margin: 0 0 var(--space-3);
  }
  .vh {
    font-size: var(--text-sm);
    font-weight: 400;
    color: var(--color-text-faint);
  }
  .muted {
    color: var(--color-text-muted);
  }
  .inuse {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    color: var(--color-warn);
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
