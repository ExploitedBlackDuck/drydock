// Frontend domain types, mirroring the Go domain model (PROJECT-BOOK §7.1).
// These are the shapes the typed Wails bindings will deliver; defining them here
// keeps components off `any` and gives the stores a contract to validate at the
// boundary (§2.8). Backend services that populate them land in later phases.

/** How root-equivalent access to a host is travelling (ADR-0005). */
export const Transport = {
  Local: 'local',
  SSH: 'ssh',
  TLS: 'tls',
} as const;
export type Transport = (typeof Transport)[keyof typeof Transport];

/** Whether the transport is trusted; an unauthenticated TCP socket is not. */
export const Trust = {
  Trusted: 'trusted',
  Untrusted: 'untrusted',
} as const;
export type Trust = (typeof Trust)[keyof typeof Trust];

/** Connection lifecycle state for a host (PROJECT-BOOK §7.11.9). */
export const HostStatus = {
  Connected: 'connected',
  Connecting: 'connecting',
  Disconnected: 'disconnected',
  /** Below the minimum supported API version — reduced capability (ADR-0008). */
  Degraded: 'degraded',
  Error: 'error',
} as const;
export type HostStatus = (typeof HostStatus)[keyof typeof HostStatus];

/** A connected or saved Docker host. */
export interface Host {
  id: string;
  name: string;
  transport: Transport;
  endpoint: string;
  trust: Trust;
  observeMode: boolean;
  status: HostStatus;
  engineVersion?: string;
  apiVersion?: string;
}
