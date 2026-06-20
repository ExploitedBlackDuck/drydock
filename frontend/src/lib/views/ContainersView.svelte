<script lang="ts">
  import ResourceView from './ResourceView.svelte';
  import StateBadge from '../components/StateBadge.svelte';
  import ContainerDetail from '../components/ContainerDetail.svelte';
  import { containers } from '../stores/objects';
  import type { Container } from '../api/engine';
  import { shortId } from '../util/format';
  import {
    removeContainer,
    restartContainer,
    startContainer,
    stopContainer,
  } from '../api/operations';

  export let hostId: string;
  export let observeMode = false;

  let selected: Container | null = null;
  let busy = '';
  let actionError: string | null = null;

  $: running = (c: Container) => c.State === 'running';

  function ports(c: Container): string {
    if (!c.Ports || c.Ports.length === 0) return '';
    return c.Ports.filter((p) => p.PublicPort)
      .map((p) => `${p.PublicPort}→${p.PrivatePort}/${p.Protocol}`)
      .join(', ');
  }

  async function act(c: Container, fn: () => Promise<void>) {
    busy = c.ID;
    actionError = null;
    try {
      await fn();
      await containers.load(hostId);
    } catch (err) {
      actionError = err instanceof Error ? err.message : String(err);
    } finally {
      busy = '';
    }
  }

  function onRemove(c: Container) {
    if (!window.confirm(`Remove container "${c.Name}"? This cannot be undone.`))
      return;
    // force=false: a running container must be stopped first; ack=true (confirmed).
    void act(c, () => removeContainer(hostId, c.ID, false, false, true));
  }
</script>

<div class="containers" class:split={selected !== null}>
  <div class="list">
    <ResourceView
      {hostId}
      store={containers}
      icon="containers"
      emptyTitle="No containers"
      emptyMessage="This host has no containers. They appear here as soon as one is created."
      let:rows
    >
      {#if actionError}
        <p class="action-error" role="alert">{actionError}</p>
      {/if}
      <table class="dd-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>State</th>
            <th>Image</th>
            <th>Ports</th>
            <th>ID</th>
            <th class="actions-col">Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each rows as c (c.ID)}
            <tr class:selected={selected?.ID === c.ID}>
              <td
                ><button class="link" on:click={() => (selected = c)}
                  >{c.Name}</button
                ></td
              >
              <td><StateBadge state={c.State} /></td>
              <td class="mono">{c.Image}</td>
              <td class="mono">{ports(c)}</td>
              <td class="mono">{shortId(c.ID)}</td>
              <td class="actions">
                {#if observeMode}
                  <span class="observe-note" title="Host is observe-only"
                    >read-only</span
                  >
                {:else}
                  {#if running(c)}
                    <button
                      disabled={busy === c.ID}
                      on:click={() => act(c, () => stopContainer(hostId, c.ID))}
                      >Stop</button
                    >
                    <button
                      disabled={busy === c.ID}
                      on:click={() =>
                        act(c, () => restartContainer(hostId, c.ID))}
                      >Restart</button
                    >
                  {:else}
                    <button
                      disabled={busy === c.ID}
                      on:click={() =>
                        act(c, () => startContainer(hostId, c.ID))}
                      >Start</button
                    >
                  {/if}
                  <button
                    class="danger"
                    disabled={busy === c.ID}
                    on:click={() => onRemove(c)}>Remove</button
                  >
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </ResourceView>
  </div>

  {#if selected}
    <div class="drawer">
      <button
        class="drawer-close"
        on:click={() => (selected = null)}
        aria-label="Close detail">✕</button
      >
      {#key selected.ID}
        <ContainerDetail {hostId} container={selected} {observeMode} />
      {/key}
    </div>
  {/if}
</div>

<style>
  .containers {
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  .list {
    flex: 1;
    min-height: 0;
    overflow: auto;
  }

  .split .list {
    flex: 1 1 55%;
  }

  .drawer {
    position: relative;
    flex: 1 1 45%;
    min-height: 0;
  }

  .drawer-close {
    position: absolute;
    top: 8px;
    right: 12px;
    z-index: 2;
    border: none;
    background: transparent;
    color: var(--color-text-muted);
    font-size: 14px;
  }
  .drawer-close:hover {
    color: var(--color-text);
  }

  .link {
    border: none;
    background: none;
    padding: 0;
    color: var(--color-text);
    font: inherit;
    text-align: left;
  }
  .link:hover {
    color: var(--color-accent);
    text-decoration: underline;
  }

  tr.selected {
    background: var(--color-accent-soft);
  }

  .actions-col {
    width: 1%;
    white-space: nowrap;
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
    background: var(--color-surface-hover);
    border-color: var(--color-accent);
  }
  .actions button.danger:hover:not(:disabled) {
    border-color: var(--color-danger);
    color: var(--color-danger);
  }
  .actions button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .observe-note {
    font-size: var(--text-xs);
    color: var(--color-warn);
  }

  .action-error {
    margin: 0;
    padding: var(--space-2) var(--space-5);
    color: var(--color-danger);
    background: var(--color-danger-soft);
    font-size: var(--text-sm);
  }
</style>
