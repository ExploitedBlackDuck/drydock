// Typed seam over the dashboard bindings: engine-event subscription with
// restart-loop alerts, and the rolling resource history.
import {
  GetResourceHistory,
  SubscribeEvents,
  UnsubscribeEvents,
} from '../../../wailsjs/go/app/App';
import { EventsOn } from '../../../wailsjs/runtime/runtime';
import type { domain } from '../../../wailsjs/go/models';

export interface RestartLoopAlert {
  hostId: string;
  containerId: string;
  containerName: string;
  deaths: number;
}

export type ResourceSample = domain.ResourceSample;

export function subscribeEvents(hostId: string): Promise<void> {
  return SubscribeEvents(hostId);
}

export function unsubscribeEvents(hostId: string): Promise<void> {
  return UnsubscribeEvents(hostId);
}

export function onRestartLoop(
  hostId: string,
  handler: (alert: RestartLoopAlert) => void,
): () => void {
  return EventsOn(`restart-loop:${hostId}`, (alert: RestartLoopAlert) =>
    handler(alert),
  );
}

export function getResourceHistory(
  hostId: string,
  containerId: string,
  limit: number,
): Promise<ResourceSample[]> {
  return GetResourceHistory(hostId, containerId, limit);
}

/** A resync signal: the live stream dropped or was re-established (ADR-0021). */
export interface ResyncEvent {
  hostId: string;
  reason: string;
}

/**
 * Subscribes to resync signals for a host. On a resync the UI refetches
 * authoritative state rather than trusting the now-stale live data.
 */
export function onResync(
  hostId: string,
  handler: (event: ResyncEvent) => void,
): () => void {
  return EventsOn(`resync:${hostId}`, (event: ResyncEvent) => handler(event));
}
