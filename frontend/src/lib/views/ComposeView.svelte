<script lang="ts">
  // Compose stacks viewed and controlled as a unit (PROJECT-BOOK §7.11.6).
  // Stacks are grouped in the core; this view renders per-service status, opens a
  // container's logs/stats in the shared detail drawer, and routes up/down
  // through the operations service. `down -v` is gated behind an explicit
  // acknowledgement because it deletes the stack's volumes.
  import ResourceView from './ResourceView.svelte';
  import StateBadge from '../components/StateBadge.svelte';
  import ContainerDetail from '../components/ContainerDetail.svelte';
  import { stacks } from '../stores/objects';
  import type { Stack } from '../api/compose';
  import { composeDown, composeUp } from '../api/compose';
  import type { Container } from '../api/engine';

  export let hostId: string;
  export let observeMode = false;

  let selected: Container | null = null;
  let busy = '';
  let actionError: string | null = null;

  // Map a stack's aggregate state to a tone; "partial" is the one to notice.
  const stackTone: Record<string, 'ok' | 'warn' | 'neutral'> = {
    running: 'ok',
    partial: 'warn',
    stopped: 'neutral',
  };

  async function act(project: string, fn: () => Promise<void>) {
    busy = project;
    actionError = null;
    try {
      await fn();
      await stacks.load(hostId);
    } catch (err) {
      actionError = err instanceof Error ? err.message : String(err);
    } finally {
      busy = '';
    }
  }

  function onUp(s: Stack) {
    void act(s.Project, () => composeUp(hostId, s.Project));
  }

  function onDown(s: Stack) {
    if (
      !window.confirm(
        `Take stack "${s.Project}" down? Its ${s.Total} container(s) will be stopped and removed.`,
      )
    )
      return;
    void act(s.Project, () => composeDown(hostId, s.Project, false, true));
  }

  function onDownVolumes(s: Stack) {
    if (
      !window.confirm(
        `Take stack "${s.Project}" down AND delete its volumes?\n\n` +
          'This removes the stack containers and its named volumes — persistent ' +
          'data is permanently deleted. This cannot be undone.',
      )
    )
      return;
    void act(s.Project, () => composeDown(hostId, s.Project, true, true));
  }
</script>

<div class="compose" class:split={selected !== null}>
  <div class="list">
    <ResourceView
      {hostId}
      store={stacks}
      icon="compose"
      emptyTitle="No Compose projects"
      emptyMessage="No containers on this host carry Compose project labels."
      let:rows
    >
      {#if actionError}
        <p class="action-error" role="alert">{actionError}</p>
      {/if}

      <div class="stacks">
        {#each rows as s (s.Project)}
          <section class="stack">
            <header>
              <div class="title">
                <h3>{s.Project}</h3>
                <span
                  class="pill {stackTone[s.State] ?? 'neutral'}"
                  title="Stack state: {s.State}"
                >
                  <span class="dot" aria-hidden="true"></span>{s.State}
                </span>
                <span class="count">{s.Running}/{s.Total} running</span>
              </div>

              <div class="actions">
                {#if observeMode}
                  <span class="observe-note" title="Host is observe-only"
                    >read-only</span
                  >
                {:else}
                  <button disabled={busy === s.Project} on:click={() => onUp(s)}
                    >Up</button
                  >
                  <button
                    disabled={busy === s.Project}
                    on:click={() => onDown(s)}>Down</button
                  >
                  <button
                    class="danger"
                    disabled={busy === s.Project}
                    title="Stop and remove the stack and delete its volumes"
                    on:click={() => onDownVolumes(s)}>Down -v</button
                  >
                {/if}
              </div>
            </header>

            <table class="dd-table">
              <thead>
                <tr>
                  <th>Service</th>
                  <th>Status</th>
                  <th>Containers</th>
                </tr>
              </thead>
              <tbody>
                {#each s.Services as svc (svc.Name)}
                  <tr>
                    <td class="svc">{svc.Name}</td>
                    <td class="mono">{svc.Running}/{svc.Total}</td>
                    <td class="containers-cell">
                      {#each svc.Containers as c (c.ID)}
                        <span class="container">
                          <button
                            class="link"
                            on:click={() => (selected = c)}
                            title="Open logs and stats">{c.Name}</button
                          >
                          <StateBadge state={c.State} />
                        </span>
                      {/each}
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </section>
        {/each}
      </div>
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
        <ContainerDetail {hostId} container={selected} />
      {/key}
    </div>
  {/if}
</div>

<style>
  .compose {
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

  .stacks {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
    padding: var(--space-4) var(--space-5);
  }

  .stack {
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    background: var(--color-surface-raised);
    overflow: hidden;
  }

  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    padding: var(--space-3) var(--space-4);
    border-bottom: 1px solid var(--color-border);
  }

  .title {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }

  h3 {
    margin: 0;
    font-size: var(--text-md);
    font-weight: 600;
  }

  .count {
    font-size: var(--text-xs);
    color: var(--color-text-muted);
  }

  .pill {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 2px 8px;
    border-radius: 999px;
    font-size: var(--text-xs);
    font-weight: 500;
    text-transform: capitalize;
    border: 1px solid var(--color-border-strong);
    color: var(--color-text-muted);
  }
  .pill .dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--color-text-faint);
  }
  .pill.ok {
    color: var(--color-ok);
    border-color: color-mix(in srgb, var(--color-ok) 45%, transparent);
  }
  .pill.ok .dot {
    background: var(--color-ok);
  }
  .pill.warn {
    color: var(--color-warn);
    border-color: color-mix(in srgb, var(--color-warn) 45%, transparent);
  }
  .pill.warn .dot {
    background: var(--color-warn);
  }
  .pill.neutral .dot {
    background: transparent;
    border: 1.5px solid var(--color-text-faint);
    width: 8px;
    height: 8px;
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
    background: var(--color-surface);
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

  .svc {
    font-weight: 500;
  }

  .containers-cell {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-3);
  }

  .container {
    display: inline-flex;
    align-items: center;
    gap: 6px;
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

  .action-error {
    margin: 0;
    padding: var(--space-2) var(--space-5);
    color: var(--color-danger);
    background: var(--color-danger-soft);
    font-size: var(--text-sm);
  }
</style>
