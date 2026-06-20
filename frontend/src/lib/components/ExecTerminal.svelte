<script lang="ts">
  // An in-app terminal into a container over the API exec endpoints
  // (PROJECT-BOOK §7.11.4) — the thing the CLI tools do unreliably over SSH.
  // xterm renders; keystrokes and output stream through the typed exec seam. The
  // session is torn down on destroy so no exec connection is left open.
  import { onMount, onDestroy } from 'svelte';
  import { Terminal } from '@xterm/xterm';
  import { FitAddon } from '@xterm/addon-fit';
  import '@xterm/xterm/css/xterm.css';
  import {
    onExecExit,
    onExecOutput,
    resizeExec,
    sendExecInput,
    startExec,
    stopExec,
  } from '../api/exec';

  export let hostId: string;
  export let containerId: string;
  /** Program to run (argv[0]); empty lets the backend pick a default shell. */
  export let shell = '';

  let host: HTMLDivElement;
  let term: Terminal | null = null;
  let fit: FitAddon | null = null;
  let sessionId: string | null = null;
  let observer: ResizeObserver | null = null;
  let unsubscribes: Array<() => void> = [];
  let error = '';
  let ended = false;

  // The terminal's colours track the app's dark tokens.
  const theme = {
    background: '#0e1116',
    foreground: '#d6dae1',
    cursor: '#d6dae1',
    selectionBackground: '#2a3340',
  };

  onMount(async () => {
    term = new Terminal({
      fontFamily:
        'var(--font-mono, ui-monospace, SFMono-Regular, Menlo, monospace)',
      fontSize: 13,
      cursorBlink: true,
      theme,
    });
    fit = new FitAddon();
    term.loadAddon(fit);
    term.open(host);
    safeFit();

    try {
      sessionId = await startExec(hostId, containerId, shell);
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
      return;
    }

    const id = sessionId;
    unsubscribes.push(
      onExecOutput(id, (bytes) => term?.write(bytes)),
      onExecExit(id, () => {
        ended = true;
        term?.write('\r\n\x1b[2m[session ended]\x1b[0m\r\n');
      }),
    );

    // Forward keystrokes and TTY size changes to the session.
    term.onData((data) => void sendExecInput(id, data));
    term.onResize(({ cols, rows }) => void resizeExec(id, cols, rows));

    // Keep the remote TTY matched to the pane size.
    observer = new ResizeObserver(() => safeFit());
    observer.observe(host);
    safeFit();
    term.focus();
  });

  onDestroy(() => {
    unsubscribes.forEach((u) => u());
    observer?.disconnect();
    if (sessionId) void stopExec(sessionId);
    term?.dispose();
  });

  function safeFit() {
    try {
      fit?.fit();
    } catch {
      // The pane can be momentarily zero-sized during layout; ignore.
    }
  }
</script>

<div class="exec">
  {#if error}
    <p class="error" role="alert">Could not start a shell: {error}</p>
  {/if}
  <div class="screen" class:ended bind:this={host}></div>
</div>

<style>
  .exec {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    background: #0e1116;
  }

  .error {
    margin: 0;
    padding: var(--space-2) var(--space-5);
    color: var(--color-danger);
    background: var(--color-danger-soft);
    font-size: var(--text-sm);
  }

  .screen {
    flex: 1;
    min-height: 0;
    padding: var(--space-2) var(--space-3);
  }
  .screen.ended {
    opacity: 0.7;
  }
</style>
