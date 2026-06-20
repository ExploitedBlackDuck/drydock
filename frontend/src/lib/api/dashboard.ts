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
