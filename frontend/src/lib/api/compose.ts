// Typed seam over the Compose bindings (PROJECT-BOOK §2.8, §7.11.6). Components
// import the stack types and calls from here, never the generated paths.
import {
  ComposeDown,
  ComposeUp,
  ListStacks,
} from '../../../wailsjs/go/app/App';
import type { domain } from '../../../wailsjs/go/models';

export type Stack = domain.Stack;
export type StackService = domain.StackService;

export function listStacks(hostId: string): Promise<Stack[]> {
  return ListStacks(hostId);
}

/** Brings a stack up by starting its containers (no acknowledgement needed). */
export function composeUp(hostId: string, project: string): Promise<void> {
  return ComposeUp(hostId, project);
}

/**
 * Takes a stack down. Destructive, so ack must be true; volumes adds `down -v`,
 * which deletes the stack's named volumes.
 */
export function composeDown(
  hostId: string,
  project: string,
  volumes: boolean,
  ack: boolean,
): Promise<void> {
  return ComposeDown(hostId, project, volumes, ack);
}
