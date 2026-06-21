<script lang="ts">
  // The compose plan preview (PROJECT-BOOK §7.12.2, ADR-0016): the operator
  // confirms the classified plan, not a black-box `up`. Destructive elements
  // (recreations that interrupt a running container or drop an anonymous volume)
  // and degraded/source-unavailable plans are surfaced before anything runs.
  import { createEventDispatcher } from 'svelte';
  import type { ComposePlan } from '../api/compose';

  export let plan: ComposePlan;
  export let busy = false;

  const dispatch = createEventDispatcher<{ confirm: void; cancel: void }>();

  const tone: Record<string, 'ok' | 'warn' | 'neutral'> = {
    create: 'ok',
    start: 'ok',
    noop: 'neutral',
    recreate: 'warn',
  };

  function onKey(e: KeyboardEvent) {
    if (e.key === 'Escape' && !busy) dispatch('cancel');
  }

  $: resources = [...(plan.Networks ?? []), ...(plan.Volumes ?? [])];
</script>

<svelte:window on:keydown={onKey} />

<div class="backdrop">
  <div class="panel" role="dialog" aria-label="Compose plan" aria-modal="true">
    <header>
      <h3>Apply plan for “{plan.Project}”</h3>
      <div class="badges">
        {#if plan.Degraded}
          <span
            class="badge warn"
            title="Convergence could not be determined with confidence"
            >degraded</span
          >
        {/if}
        {#if plan.Destructive}
          <span class="badge danger">destructive</span>
        {/if}
      </div>
    </header>

    {#each plan.Notes ?? [] as note}
      <p class="note" role="note">{note}</p>
    {/each}

    <div class="body">
      {#if (plan.Services ?? []).length === 0}
        <p class="empty">No service changes.</p>
      {:else}
        <table class="dd-table">
          <thead>
            <tr><th>Service</th><th>Action</th><th>Why</th></tr>
          </thead>
          <tbody>
            {#each plan.Services as s (s.Service)}
              <tr>
                <td class="svc">{s.Service}</td>
                <td>
                  <span class="badge {tone[s.Action] ?? 'neutral'}"
                    >{s.Action}</span
                  >
                  {#if s.InterruptsRunning}<span class="flag"
                      >interrupts running</span
                    >{/if}
                  {#if s.DropsAnonymousVolumes}<span class="flag danger"
                      >drops anon volume</span
                    >{/if}
                </td>
                <td class="why">{(s.Reasons ?? []).join('; ')}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      {/if}

      {#if resources.length > 0}
        <ul class="resources">
          {#each resources as r (r.Kind + r.Name)}
            <li>
              <span class="badge ok">{r.Action}</span>
              {r.Kind} <b>{r.Name}</b>
            </li>
          {/each}
        </ul>
      {/if}
    </div>

    <footer>
      {#if plan.Destructive}
        <p class="warn-line">
          This interrupts running containers or deletes data. Confirm to
          proceed.
        </p>
      {/if}
      <div class="actions">
        <button disabled={busy} on:click={() => dispatch('cancel')}
          >Cancel</button
        >
        <button
          class="primary"
          disabled={busy}
          on:click={() => dispatch('confirm')}
        >
          {busy
            ? 'Applying…'
            : plan.Destructive
              ? 'Apply (confirmed)'
              : 'Apply'}
        </button>
      </div>
    </footer>
  </div>
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 50;
  }
  .panel {
    width: min(640px, 92vw);
    max-height: 84vh;
    display: flex;
    flex-direction: column;
    background: var(--color-surface);
    border: 1px solid var(--color-border-strong);
    border-radius: var(--radius-md);
    overflow: hidden;
  }
  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    padding: var(--space-4) var(--space-5);
    border-bottom: 1px solid var(--color-border);
  }
  h3 {
    margin: 0;
    font-size: var(--text-md);
    font-weight: 600;
  }
  .badges {
    display: flex;
    gap: 6px;
  }
  .note {
    margin: 0;
    padding: var(--space-2) var(--space-5);
    background: var(--color-surface-raised);
    color: var(--color-warn);
    font-size: var(--text-sm);
  }
  .body {
    flex: 1;
    min-height: 0;
    overflow: auto;
    padding: var(--space-3) var(--space-5);
  }
  .svc {
    font-weight: 500;
  }
  .why {
    color: var(--color-text-muted);
    font-size: var(--text-xs);
  }
  .resources {
    list-style: none;
    margin: var(--space-3) 0 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
    font-size: var(--text-sm);
  }
  .badge {
    display: inline-block;
    padding: 1px 7px;
    border-radius: 999px;
    font-size: var(--text-xs);
    border: 1px solid var(--color-border-strong);
    text-transform: capitalize;
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
  .badge.neutral {
    color: var(--color-text-muted);
  }
  .flag {
    margin-left: 6px;
    font-size: var(--text-xs);
    color: var(--color-warn);
  }
  .flag.danger {
    color: var(--color-danger);
  }
  .empty {
    color: var(--color-text-muted);
    font-size: var(--text-sm);
  }
  footer {
    border-top: 1px solid var(--color-border);
    padding: var(--space-3) var(--space-5);
  }
  .warn-line {
    margin: 0 0 var(--space-2);
    color: var(--color-danger);
    font-size: var(--text-sm);
  }
  .actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--space-3);
  }
  .actions button {
    padding: 5px 14px;
    border: 1px solid var(--color-border-strong);
    border-radius: var(--radius-sm);
    background: var(--color-surface-raised);
    color: var(--color-text);
    font-size: var(--text-sm);
  }
  .actions button.primary {
    border-color: var(--color-accent);
    background: var(--color-accent-soft);
  }
  .actions button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
