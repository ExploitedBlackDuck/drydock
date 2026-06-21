// The primary per-host views (PROJECT-BOOK §7.11.1) as a typed catalog. The
// navigation rail and router are generated from this list so a view id is never
// a loose string.

export const ViewId = {
  Containers: 'containers',
  Compose: 'compose',
  Images: 'images',
  Volumes: 'volumes',
  Networks: 'networks',
  Exposure: 'exposure',
  Disk: 'disk',
  History: 'history',
  Audit: 'audit',
} as const;
export type ViewId = (typeof ViewId)[keyof typeof ViewId];

/** Icon names resolved by the Icon component. */
export type IconName =
  | 'containers'
  | 'compose'
  | 'images'
  | 'volumes'
  | 'networks'
  | 'exposure'
  | 'disk'
  | 'history'
  | 'audit';

export interface ViewMeta {
  id: ViewId;
  label: string;
  icon: IconName;
}

/** Ordered navigation catalog. */
export const VIEWS: readonly ViewMeta[] = [
  { id: ViewId.Containers, label: 'Containers', icon: 'containers' },
  { id: ViewId.Compose, label: 'Compose', icon: 'compose' },
  { id: ViewId.Images, label: 'Images', icon: 'images' },
  { id: ViewId.Volumes, label: 'Volumes', icon: 'volumes' },
  { id: ViewId.Networks, label: 'Networks', icon: 'networks' },
  { id: ViewId.Exposure, label: 'Exposure', icon: 'exposure' },
  { id: ViewId.Disk, label: 'Disk', icon: 'disk' },
  { id: ViewId.History, label: 'History', icon: 'history' },
  { id: ViewId.Audit, label: 'Audit', icon: 'audit' },
];
