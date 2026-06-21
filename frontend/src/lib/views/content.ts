// Per-view copy, keyed by the typed ViewId. Centralising the headings and
// empty-state messages keeps PrimaryView declarative and the wording consistent
// with the specification (PROJECT-BOOK §7.11.3–§7.11.8).

import { ViewId, type IconName } from '../types/view';

export interface ViewContent {
  description: string;
  icon: IconName;
  /** Title shown when the host is connected but the collection is empty. */
  emptyTitle: string;
  emptyMessage: string;
}

export const VIEW_CONTENT: Record<ViewId, ViewContent> = {
  [ViewId.Containers]: {
    description:
      'Start, stop, inspect, and follow logs and stats per container.',
    icon: 'containers',
    emptyTitle: 'No containers',
    emptyMessage:
      'This host has no containers. They appear here as soon as one is created.',
  },
  [ViewId.Compose]: {
    description:
      'Compose stacks grouped by project, viewed and controlled as a unit.',
    icon: 'compose',
    emptyTitle: 'No Compose projects',
    emptyMessage: 'No containers on this host carry Compose project labels.',
  },
  [ViewId.Images]: {
    description:
      'Inspect and remove images; preview reclaimable space before pruning.',
    icon: 'images',
    emptyTitle: 'No images',
    emptyMessage: 'This host has no images pulled or built yet.',
  },
  [ViewId.Volumes]: {
    description:
      'Named volumes hold persistent data and are never bulk-deleted.',
    icon: 'volumes',
    emptyTitle: 'No volumes',
    emptyMessage:
      'This host has no volumes. Each volume is confirmed individually before removal.',
  },
  [ViewId.Networks]: {
    description: 'Inspect networks, their drivers, and their attachments.',
    icon: 'networks',
    emptyTitle: 'No networks',
    emptyMessage: 'Only the default networks exist on this host.',
  },
  [ViewId.Exposure]: {
    description:
      'What each container publishes and how far it reaches — loopback vs all-interfaces.',
    icon: 'exposure',
    emptyTitle: 'Nothing published',
    emptyMessage:
      'No container on this host publishes a port, and none use host networking.',
  },
  [ViewId.Disk]: {
    description:
      'system df made legible — reclaimable space by category, build cache first-class.',
    icon: 'disk',
    emptyTitle: 'No disk usage',
    emptyMessage: 'Nothing is reclaimable on this host yet.',
  },
  [ViewId.History]: {
    description:
      'Past operations per host with their target, options, and result.',
    icon: 'history',
    emptyTitle: 'No operations yet',
    emptyMessage: 'Operations performed through Drydock are recorded here.',
  },
  [ViewId.Audit]: {
    description:
      'Append-only, hash-chained record of every consequential action.',
    icon: 'audit',
    emptyTitle: 'No audit entries',
    emptyMessage:
      'Connecting a host and acting on it writes tamper-evident entries here.',
  },
};
