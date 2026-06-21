// Typed seam over the Compose bindings (PROJECT-BOOK §2.8, §7.11.6). Components
// import the stack types and calls from here, never the generated paths.
import {
  ApplyComposePlan,
  ComposeDown,
  ComposeUp,
  ComputeComposePlan,
  ListStacks,
} from '../../../wailsjs/go/app/App';
import type { domain } from '../../../wailsjs/go/models';

export type Stack = domain.Stack;
export type StackService = domain.StackService;
export type ComposePlan = domain.ComposePlan;
export type ServiceChange = domain.ServiceChange;
export type ResourceChange = domain.ResourceChange;

export function listStacks(hostId: string): Promise<Stack[]> {
  return ListStacks(hostId);
}

/** Previews what bringing a stack up would do (ADR-0016). */
export function computeComposePlan(
  hostId: string,
  project: string,
): Promise<ComposePlan> {
  return ComputeComposePlan(hostId, project);
}

/** Applies a stack's plan; a destructive plan requires ack=true. */
export function applyComposePlan(
  hostId: string,
  project: string,
  ack: boolean,
): Promise<void> {
  return ApplyComposePlan(hostId, project, ack);
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
