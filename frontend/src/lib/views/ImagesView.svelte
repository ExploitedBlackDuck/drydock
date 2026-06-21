<script lang="ts">
  import { onMount } from 'svelte';
  import ResourceView from './ResourceView.svelte';
  import { images } from '../stores/objects';
  import { formatBytes, shortId } from '../util/format';
  import {
    checkImageDrift,
    listImageProvenance,
    type ImageProvenance,
  } from '../api/provenance';

  export let hostId: string;

  // Provenance (age, :latest, drift) keyed by image ref. Listing is local — no
  // registry call — and the per-image "check" is the only thing that reaches a
  // registry, through the host's daemon (ADR-0019).
  let provenance: Record<string, ImageProvenance> = {};
  let checking = '';

  function refOf(repo: string, tag: string): string {
    return repo && repo !== '<none>' ? `${repo}:${tag}` : repo;
  }

  async function loadProvenance() {
    try {
      const list = await listImageProvenance(hostId);
      provenance = Object.fromEntries(list.map((p) => [p.ImageRef, p]));
    } catch {
      provenance = {};
    }
  }

  let loadedFor: string | null = null;
  $: if (hostId && hostId !== loadedFor) {
    loadedFor = hostId;
    void loadProvenance();
  }
  onMount(() => {
    if (loadedFor === null && hostId) {
      loadedFor = hostId;
      void loadProvenance();
    }
  });

  async function onCheck(ref: string) {
    checking = ref;
    try {
      const updated = await checkImageDrift(hostId, ref);
      provenance = { ...provenance, [ref]: updated };
    } catch {
      // A failure means the registry is unreachable from the host; leave the
      // local provenance as-is (the badge stays "uncertain").
    } finally {
      checking = '';
    }
  }

  type Badge = { tone: 'ok' | 'warn' | 'danger' | 'neutral'; label: string };
  function badge(p: ImageProvenance | undefined): Badge {
    if (!p) return { tone: 'neutral', label: '—' };
    if (p.Untagged) return { tone: 'warn', label: 'untagged' };
    if (p.Checked)
      return p.TagDrifted
        ? { tone: 'danger', label: 'drifted' }
        : { tone: 'ok', label: 'current' };
    if (p.Latest) return { tone: 'warn', label: ':latest' };
    return { tone: 'neutral', label: 'unchecked' };
  }
</script>

<ResourceView
  {hostId}
  store={images}
  icon="images"
  emptyTitle="No images"
  emptyMessage="This host has no images pulled or built yet."
  let:rows
>
  <table class="dd-table">
    <thead>
      <tr>
        <th>Repository</th>
        <th>Tag</th>
        <th class="num">Size</th>
        <th>Status</th>
        <th>Provenance</th>
        <th>ID</th>
      </tr>
    </thead>
    <tbody>
      {#each rows as img (img.ID)}
        {@const ref = refOf(img.Repo, img.Tag)}
        {@const b = badge(provenance[ref])}
        <tr>
          <td>{img.Repo}</td>
          <td class="mono">{img.Tag}</td>
          <td class="num">{formatBytes(img.Size)}</td>
          <td class="muted">
            {#if img.Dangling}dangling{:else if img.InUse}in use{:else}unused{/if}
          </td>
          <td class="prov">
            <span class="badge {b.tone}">{b.label}</span>
            {#if !img.Dangling}
              <button
                class="check"
                disabled={checking === ref}
                title="Check the registry for this tag's current digest (via the host)"
                on:click={() => onCheck(ref)}
                >{checking === ref ? '…' : 'check'}</button
              >
            {/if}
          </td>
          <td class="mono">{shortId(img.ID)}</td>
        </tr>
      {/each}
    </tbody>
  </table>
</ResourceView>

<style>
  .prov {
    display: flex;
    align-items: center;
    gap: 8px;
    white-space: nowrap;
  }
  .badge {
    display: inline-block;
    padding: 1px 7px;
    border-radius: 999px;
    font-size: var(--text-xs);
    border: 1px solid var(--color-border-strong);
    color: var(--color-text-muted);
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
  .check {
    padding: 1px 8px;
    border: 1px solid var(--color-border-strong);
    border-radius: var(--radius-sm);
    background: var(--color-surface-raised);
    color: var(--color-text-muted);
    font-size: var(--text-xs);
  }
  .check:hover:not(:disabled) {
    border-color: var(--color-accent);
    color: var(--color-text);
  }
  .check:disabled {
    opacity: 0.5;
  }
</style>
