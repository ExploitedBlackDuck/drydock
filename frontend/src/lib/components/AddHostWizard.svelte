<script lang="ts">
  // The add-host wizard (PROJECT-BOOK §7.11.2): choose a transport, name the
  // host, and provide its endpoint / SSH identity. It records a host profile in
  // the registry. Live connection testing and engine/API version reporting are
  // wired when the transport adapters land (P3); the "Test connection" action is
  // therefore present but disabled, with a note, rather than faking a result.
  import { createEventDispatcher } from 'svelte';
  import Icon from './Icon.svelte';
  import { hosts } from '../stores/hosts';
  import {
    Transport,
    Trust,
    HostStatus,
    type Transport as TransportT,
  } from '../types/domain';

  const dispatch = createEventDispatcher<{ close: void }>();

  let name = '';
  let transport: TransportT = Transport.SSH;
  let endpoint = '';
  let identity = '';
  let observeMode = false;

  const placeholders: Record<TransportT, string> = {
    [Transport.Local]: 'unix:///var/run/docker.sock',
    [Transport.SSH]: 'ssh://user@host',
    [Transport.TLS]: 'tcp://host:2376',
  };

  const transportOptions: { value: TransportT; label: string; icon: string }[] =
    [
      { value: Transport.Local, label: 'Local', icon: 'local' },
      { value: Transport.SSH, label: 'SSH', icon: 'ssh' },
      { value: Transport.TLS, label: 'mTLS', icon: 'tls' },
    ];

  $: trimmedEndpoint = endpoint.trim() || placeholders[transport];
  $: canAdd = name.trim().length > 0;

  function add() {
    if (!canAdd) return;
    hosts.add({
      id: crypto.randomUUID(),
      name: name.trim(),
      transport,
      endpoint: trimmedEndpoint,
      trust: Trust.Trusted,
      observeMode,
      status: HostStatus.Disconnected,
    });
    dispatch('close');
  }

  function onKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape') dispatch('close');
  }

  function onBackdropClick(event: MouseEvent) {
    // Close only when the backdrop itself is clicked, not its children — avoids
    // needing a stopPropagation handler on the dialog (and the a11y warning).
    if (event.target === event.currentTarget) dispatch('close');
  }
</script>

<svelte:window on:keydown={onKeydown} />

<!-- svelte-ignore a11y-click-events-have-key-events a11y-no-static-element-interactions -->
<div class="backdrop" on:click={onBackdropClick}>
  <div
    class="modal"
    role="dialog"
    aria-modal="true"
    aria-labelledby="wizard-title"
  >
    <header>
      <h2 id="wizard-title">Add a host</h2>
      <button
        class="close"
        aria-label="Close"
        on:click={() => dispatch('close')}
      >
        <Icon name="plus" size={18} />
      </button>
    </header>

    <div class="fields">
      <label class="field">
        <span>Name</span>
        <!-- svelte-ignore a11y-autofocus -->
        <input bind:value={name} placeholder="prod-1" autofocus />
      </label>

      <div class="field">
        <span>Transport</span>
        <div class="segmented" role="radiogroup" aria-label="Transport">
          {#each transportOptions as opt (opt.value)}
            <button
              type="button"
              role="radio"
              aria-checked={transport === opt.value}
              class:selected={transport === opt.value}
              on:click={() => (transport = opt.value)}
            >
              <Icon name={opt.icon} size={15} />
              {opt.label}
            </button>
          {/each}
        </div>
      </div>

      <label class="field">
        <span>Endpoint</span>
        <input
          bind:value={endpoint}
          placeholder={placeholders[transport]}
          spellcheck="false"
        />
      </label>

      {#if transport === Transport.SSH}
        <label class="field">
          <span>SSH identity <em>(optional)</em></span>
          <input
            bind:value={identity}
            placeholder="agent, or ~/.ssh/id_ed25519"
            spellcheck="false"
          />
          <small
            >Drydock references your existing keys/agent — it never copies
            private keys.</small
          >
        </label>
      {/if}

      <label class="checkbox">
        <input type="checkbox" bind:checked={observeMode} />
        <span>Observe-only — reject all mutations on this host</span>
      </label>
    </div>

    <p class="note">
      <Icon name="alert" size={14} />
      Connection testing and engine/API version detection arrive with the transport
      layer; this saves the host profile.
    </p>

    <footer>
      <button class="btn ghost" disabled>Test connection</button>
      <div class="spacer"></div>
      <button class="btn ghost" on:click={() => dispatch('close')}
        >Cancel</button
      >
      <button class="btn primary" disabled={!canAdd} on:click={add}
        >Add host</button
      >
    </footer>
  </div>
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.55);
    display: grid;
    place-items: center;
    z-index: 50;
  }

  .modal {
    width: min(460px, calc(100vw - 32px));
    background: var(--color-surface);
    border: 1px solid var(--color-border-strong);
    border-radius: var(--radius-lg);
    box-shadow: 0 24px 60px rgba(0, 0, 0, 0.5);
  }

  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--space-4) var(--space-5);
    border-bottom: 1px solid var(--color-border);
  }

  h2 {
    margin: 0;
    font-size: var(--text-lg);
    font-weight: 650;
  }

  .close {
    display: grid;
    place-items: center;
    width: 28px;
    height: 28px;
    border: none;
    border-radius: var(--radius-sm);
    background: transparent;
    color: var(--color-text-muted);
    transform: rotate(45deg);
  }
  .close:hover {
    background: var(--color-surface-hover);
    color: var(--color-text);
  }

  .fields {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
    padding: var(--space-5);
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .field > span {
    font-size: var(--text-sm);
    font-weight: 500;
    color: var(--color-text-muted);
  }
  .field em {
    font-style: normal;
    color: var(--color-text-faint);
    font-weight: 400;
  }

  .field input {
    padding: 8px 10px;
    border: 1px solid var(--color-border-strong);
    border-radius: var(--radius-sm);
    background: var(--color-bg);
    color: var(--color-text);
    font-size: var(--text-sm);
    font-family: var(--font-mono);
  }
  .field input:focus {
    border-color: var(--color-accent);
    outline: none;
  }

  small {
    font-size: var(--text-xs);
    color: var(--color-text-faint);
  }

  .segmented {
    display: flex;
    gap: 4px;
    padding: 3px;
    background: var(--color-bg);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
  }
  .segmented button {
    flex: 1;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    padding: 7px;
    border: none;
    border-radius: var(--radius-sm);
    background: transparent;
    color: var(--color-text-muted);
    font-size: var(--text-sm);
  }
  .segmented button.selected {
    background: var(--color-accent-soft);
    color: var(--color-text);
  }

  .checkbox {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    font-size: var(--text-sm);
    color: var(--color-text);
  }
  .checkbox input {
    width: 15px;
    height: 15px;
    accent-color: var(--color-accent);
  }

  .note {
    display: flex;
    align-items: flex-start;
    gap: 8px;
    margin: 0;
    padding: var(--space-3) var(--space-5);
    color: var(--color-text-faint);
    font-size: var(--text-xs);
    line-height: 1.5;
    border-top: 1px solid var(--color-border);
  }

  footer {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-4) var(--space-5);
    border-top: 1px solid var(--color-border);
  }
  .spacer {
    flex: 1;
  }

  .btn {
    padding: 8px 14px;
    border-radius: var(--radius-sm);
    font-size: var(--text-sm);
    font-weight: 500;
    border: 1px solid transparent;
  }
  .btn.ghost {
    background: transparent;
    border-color: var(--color-border-strong);
    color: var(--color-text);
  }
  .btn.ghost:hover:not(:disabled) {
    background: var(--color-surface-hover);
  }
  .btn.primary {
    background: var(--color-accent);
    color: #fff;
  }
  .btn.primary:hover:not(:disabled) {
    filter: brightness(1.08);
  }
  .btn:disabled {
    opacity: 0.45;
    cursor: not-allowed;
  }
</style>
