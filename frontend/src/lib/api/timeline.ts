// Typed seam over the host-timeline binding (PROJECT-BOOK §7.12.4, §2.8).
import { GetHostTimeline } from '../../../wailsjs/go/app/App';
import type { app, domain } from '../../../wailsjs/go/models';

export type HostTimeline = app.HostTimelineDTO;
export type TimelineEntry = domain.TimelineEntry;

/** The host's timeline: engine events interleaved with audit references. */
export function getHostTimeline(
  hostId: string,
  limit: number,
): Promise<HostTimeline> {
  return GetHostTimeline(hostId, limit);
}
