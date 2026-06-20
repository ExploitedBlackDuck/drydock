// Typed seam over the interactive-exec bindings (PROJECT-BOOK §7.11.4, §2.8).
// Terminal bytes cross the bridge base64-encoded so raw control sequences survive
// intact; the helpers here convert between xterm's strings and that wire form.
import {
  ResizeExec,
  SendExecInput,
  StartExec,
  StopExec,
} from '../../../wailsjs/go/app/App';
import { EventsOn } from '../../../wailsjs/runtime/runtime';

/** Opens a shell into a container and returns the session id. */
export function startExec(
  hostId: string,
  containerId: string,
  shell: string,
): Promise<string> {
  return StartExec(hostId, containerId, shell);
}

/** Sends operator keystrokes (a UTF-8 string) to the session. */
export function sendExecInput(sessionId: string, data: string): Promise<void> {
  return SendExecInput(sessionId, encodeInput(data));
}

/** Resizes the session's remote TTY. */
export function resizeExec(
  sessionId: string,
  cols: number,
  rows: number,
): Promise<void> {
  return ResizeExec(sessionId, cols, rows);
}

/** Ends the session and releases its connection. */
export function stopExec(sessionId: string): Promise<void> {
  return StopExec(sessionId);
}

/** Subscribes to container output, delivered as decoded bytes for term.write. */
export function onExecOutput(
  sessionId: string,
  handler: (bytes: Uint8Array) => void,
): () => void {
  return EventsOn(`exec:${sessionId}`, (chunk: string) =>
    handler(decodeOutput(chunk)),
  );
}

/** Subscribes to the session-ended signal. */
export function onExecExit(sessionId: string, handler: () => void): () => void {
  return EventsOn(`exec-exit:${sessionId}`, () => handler());
}

function encodeInput(s: string): string {
  const bytes = new TextEncoder().encode(s);
  let binary = '';
  for (const b of bytes) binary += String.fromCharCode(b);
  return btoa(binary);
}

function decodeOutput(b64: string): Uint8Array {
  const binary = atob(b64);
  const out = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) out[i] = binary.charCodeAt(i);
  return out;
}
