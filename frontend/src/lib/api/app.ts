// Typed access to the Go binding layer. The frontend talks to the backend only
// through generated Wails bindings (PROJECT-BOOK §2.8); this module is the thin,
// typed seam over them so components never reach into generated paths directly.
//
// The Wails runtime is injected by the desktop shell at load time. The guards
// below keep the app from crashing when it is not present (e.g. a browser opened
// outside the shell): bindings degrade to a no-op / rejected promise rather than
// throwing during mount.
import { Version } from '../../../wailsjs/go/app/App';
import { EventsOn } from '../../../wailsjs/runtime/runtime';
import { AppReady } from './events';

function runtimeReady(): boolean {
  return typeof window !== 'undefined' && 'runtime' in window;
}

/** Returns the running Drydock version from the backend. */
export function getVersion(): Promise<string> {
  if (!('go' in globalThis)) {
    return Promise.reject(new Error('backend runtime unavailable'));
  }
  return Version();
}

/**
 * Subscribes to the app-ready event, which carries the backend version once the
 * runtime is up. Returns an unsubscribe function.
 */
export function onAppReady(handler: (version: string) => void): () => void {
  if (!runtimeReady()) {
    return () => {};
  }
  return EventsOn(AppReady, (version: string) => handler(version));
}
