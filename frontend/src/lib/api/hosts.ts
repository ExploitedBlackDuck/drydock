// Typed seam over the generated host bindings, mapping the backend HostDTO to
// the frontend Host model.
import {
  AddHost,
  ConnectHost,
  DisconnectHost,
  ListHosts,
  RemoveHost,
  SetObserveMode,
} from '../../../wailsjs/go/app/App';
import type { app } from '../../../wailsjs/go/models';
import { HostStatus, Transport, Trust, type Host } from '../types/domain';

export interface AddHostInput {
  name: string;
  transport: Transport;
  endpoint: string;
  observeMode: boolean;
}

function toHost(dto: app.HostDTO): Host {
  return {
    id: dto.id,
    name: dto.name,
    transport: dto.transport as Transport,
    endpoint: dto.endpoint,
    trust: dto.trust as Trust,
    observeMode: dto.observeMode,
    status: dto.connected ? HostStatus.Connected : HostStatus.Disconnected,
    engineVersion: dto.engineVersion || undefined,
    apiVersion: dto.apiVersion || undefined,
  };
}

export async function listHosts(): Promise<Host[]> {
  const dtos = await ListHosts();
  return (dtos ?? []).map(toHost);
}

export async function addHost(input: AddHostInput): Promise<Host> {
  return toHost(await AddHost(input));
}

export async function connectHost(id: string): Promise<Host> {
  return toHost(await ConnectHost(id));
}

export function disconnectHost(id: string): Promise<void> {
  return DisconnectHost(id);
}

export function removeHost(id: string): Promise<void> {
  return RemoveHost(id);
}

export function setObserveMode(id: string, observe: boolean): Promise<void> {
  return SetObserveMode(id, observe);
}
