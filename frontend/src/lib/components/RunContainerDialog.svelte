<script lang="ts">
  // The option-rich run/create builder (PROJECT-BOOK §7.5, ADR-0011): tune a
  // container from catalogued options (no free-text flags), see the resolved
  // operation, then run. The fields and their help/risk come from the catalog —
  // an option absent from the catalog is absent here.
  import { createEventDispatcher, onMount } from 'svelte';
  import {
    optionCatalog,
    runContainer,
    type CatalogOption,
    type RunSpec,
  } from '../api/run';

  export let hostId: string;

  const dispatch = createEventDispatcher<{ ran: void; cancel: void }>();

  let catalog: Record<string, CatalogOption> = {};
  let busy = false;
  let error = '';

  // Builder state.
  let image = '';
  let name = '';
  let envText = '';
  let publishText = '';
  let volumeText = '';
  let commandText = '';
  let restart = 'no';
  let networkHost = false;
  let user = '';
  let workdir = '';

  onMount(async () => {
    try {
      const list = await optionCatalog();
      catalog = Object.fromEntries(list.map((o) => [o.name, o]));
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    }
  });

  const has = (n: string) => n in catalog;
  function lines(text: string): string[] {
    return text
      .split('\n')
      .map((l) => l.trim())
      .filter(Boolean);
  }

  $: spec = {
    Image: image.trim(),
    Name: name.trim(),
    Command: lines(commandText),
    Env: lines(envText),
    Publish: lines(publishText),
    Volumes: lines(volumeText),
    Restart: restart,
    NetworkHost: networkHost,
    User: user.trim(),
    WorkingDir: workdir.trim(),
  } satisfies RunSpec;

  // Resolved-operation summary (env values are never shown — secret, ADR-0023).
  $: resolved = [
    spec.Image && `image ${spec.Image}`,
    spec.Name && `name ${spec.Name}`,
    spec.Env.length && `${spec.Env.length} env var(s) ‹redacted›`,
    spec.Publish.length && `publish ${spec.Publish.join(', ')}`,
    spec.Volumes.length && `volumes ${spec.Volumes.join(', ')}`,
    spec.Restart !== 'no' && `restart=${spec.Restart}`,
    spec.NetworkHost && 'network=host',
    spec.User && `user ${spec.User}`,
  ].filter(Boolean) as string[];

  function riskTone(name: string): 'neutral' | 'warn' | 'danger' {
    const o = catalog[name];
    if (!o) return 'neutral';
    if (o.risk === 'destructive') return 'danger';
    if (o.affectsData) return 'warn';
    return 'neutral';
  }

  async function onRun() {
    if (!spec.Image) {
      error = 'An image is required.';
      return;
    }
    busy = true;
    error = '';
    try {
      await runContainer(hostId, spec);
      dispatch('ran');
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      busy = false;
    }
  }
</script>

<div class="backdrop">
  <div
    class="panel"
    role="dialog"
    aria-label="Run a container"
    aria-modal="true"
  >
    <header><h3>Run a container</h3></header>

    <div class="body">
      <label class="field">
        <span class="lbl">Image <em>required</em></span>
        <input bind:value={image} placeholder="e.g. nginx:1.27" />
      </label>
      <label class="field">
        <span class="lbl">Name</span>
        <input bind:value={name} placeholder="optional" />
      </label>

      {#if has('env')}
        <label class="field">
          <span class="lbl"
            >{catalog['env'].summary}
            <span class="badge {riskTone('env')}">secret</span></span
          >
          <textarea
            bind:value={envText}
            rows="2"
            placeholder="NAME=value (one per line)"
          ></textarea>
        </label>
      {/if}
      {#if has('publish')}
        <label class="field">
          <span class="lbl">{catalog['publish'].summary}</span>
          <textarea
            bind:value={publishText}
            rows="2"
            placeholder="127.0.0.1:8080:80 (one per line — prefer 127.0.0.1)"
          ></textarea>
        </label>
      {/if}
      {#if has('volume')}
        <label class="field">
          <span class="lbl"
            >{catalog['volume'].summary}
            <span class="badge {riskTone('volume')}">data</span></span
          >
          <textarea
            bind:value={volumeText}
            rows="2"
            placeholder="source:/target[:ro] (one per line)"
          ></textarea>
        </label>
      {/if}

      <label class="field">
        <span class="lbl">Command override</span>
        <textarea
          bind:value={commandText}
          rows="2"
          placeholder="one argument per line (optional)"
        ></textarea>
      </label>

      <div class="row">
        {#if has('restart')}
          <label class="field small">
            <span class="lbl">{catalog['restart'].summary}</span>
            <select bind:value={restart}>
              <option value="no">no</option>
              <option value="on-failure">on-failure</option>
              <option value="always">always</option>
              <option value="unless-stopped">unless-stopped</option>
            </select>
          </label>
        {/if}
        {#if has('user')}
          <label class="field small">
            <span class="lbl">{catalog['user'].summary}</span>
            <input bind:value={user} placeholder="optional" />
          </label>
        {/if}
        {#if has('network-host')}
          <label class="check">
            <input type="checkbox" bind:checked={networkHost} />
            {catalog['network-host'].summary}
          </label>
        {/if}
      </div>

      <div class="resolved">
        <span class="rlbl">Resolved</span>
        {#if resolved.length === 0}
          <span class="muted">choose an image and options</span>
        {:else}
          <code>{resolved.join('  ·  ')}</code>
        {/if}
      </div>
    </div>

    {#if error}<p class="error" role="alert">{error}</p>{/if}

    <footer>
      <button disabled={busy} on:click={() => dispatch('cancel')}>Cancel</button
      >
      <button class="primary" disabled={busy || !spec.Image} on:click={onRun}>
        {busy ? 'Running…' : 'Run'}
      </button>
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
    width: min(620px, 94vw);
    max-height: 88vh;
    display: flex;
    flex-direction: column;
    background: var(--color-surface);
    border: 1px solid var(--color-border-strong);
    border-radius: var(--radius-md);
    overflow: hidden;
  }
  header {
    padding: var(--space-4) var(--space-5);
    border-bottom: 1px solid var(--color-border);
  }
  header h3 {
    margin: 0;
    font-size: var(--text-md);
  }
  .body {
    flex: 1;
    min-height: 0;
    overflow: auto;
    padding: var(--space-4) var(--space-5);
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }
  .field {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .lbl {
    font-size: var(--text-sm);
    color: var(--color-text-muted);
  }
  .lbl em {
    color: var(--color-text-faint);
    font-style: normal;
    font-size: var(--text-xs);
  }
  input,
  textarea,
  select {
    background: var(--color-surface-raised);
    color: var(--color-text);
    border: 1px solid var(--color-border-strong);
    border-radius: var(--radius-sm);
    padding: 6px 8px;
    font-size: var(--text-sm);
    font-family: inherit;
  }
  textarea {
    resize: vertical;
    font-family: var(--font-mono);
    font-size: var(--text-xs);
  }
  .row {
    display: flex;
    gap: var(--space-4);
    align-items: flex-end;
    flex-wrap: wrap;
  }
  .field.small {
    flex: 0 0 auto;
  }
  .check {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: var(--text-sm);
    color: var(--color-text-muted);
  }
  .badge {
    display: inline-block;
    padding: 0 6px;
    border-radius: 999px;
    font-size: var(--text-xs);
    border: 1px solid var(--color-border-strong);
    color: var(--color-text-faint);
  }
  .badge.warn {
    color: var(--color-warn);
  }
  .badge.danger {
    color: var(--color-danger);
  }
  .resolved {
    margin-top: var(--space-2);
    padding: var(--space-2) var(--space-3);
    background: var(--color-bg);
    border-radius: var(--radius-sm);
    font-size: var(--text-xs);
  }
  .rlbl {
    color: var(--color-text-faint);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin-right: var(--space-2);
  }
  .resolved code {
    color: var(--color-text);
    word-break: break-word;
  }
  .muted {
    color: var(--color-text-faint);
  }
  .error {
    margin: 0;
    padding: var(--space-2) var(--space-5);
    color: var(--color-danger);
    background: var(--color-danger-soft);
    font-size: var(--text-sm);
  }
  footer {
    display: flex;
    justify-content: flex-end;
    gap: var(--space-3);
    padding: var(--space-3) var(--space-5);
    border-top: 1px solid var(--color-border);
  }
  footer button {
    padding: 5px 14px;
    border: 1px solid var(--color-border-strong);
    border-radius: var(--radius-sm);
    background: var(--color-surface-raised);
    color: var(--color-text);
    font-size: var(--text-sm);
  }
  footer button.primary {
    border-color: var(--color-accent);
    background: var(--color-accent-soft);
  }
  footer button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
