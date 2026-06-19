// Typed access to the Go binding layer. The frontend talks to the backend only
// through generated Wails bindings (PROJECT-BOOK §2.8); this module is the thin,
// typed seam over them so components never reach into generated paths directly.
import { Version } from '../../../wailsjs/go/app/App';
import { EventsOn } from '../../../wailsjs/runtime/runtime';
import { AppReady } from './events';

/** Returns the running Drydock version from the backend. */
export function getVersion(): Promise<string> {
  return Version();
}

/**
 * Subscribes to the app-ready event, which carries the backend version once the
 * runtime is up. Returns an unsubscribe function.
 */
export function onAppReady(handler: (version: string) => void): () => void {
  return EventsOn(AppReady, (version: string) => handler(version));
}
