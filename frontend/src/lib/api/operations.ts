// Typed seam over the container-operation and streaming bindings.
import {
  KillContainer,
  RemoveContainer,
  RestartContainer,
  StartContainer,
  StopContainer,
  StopContainerLogs,
  StopContainerStats,
  StreamContainerLogs,
  StreamContainerStats,
} from '../../../wailsjs/go/app/App';
import { EventsOn } from '../../../wailsjs/runtime/runtime';

/** A live stats sample pushed on the "stats:<id>" event. */
export interface StatsSample {
  containerId: string;
  cpuPct: number;
  memBytes: number;
  netRx: number;
  netTx: number;
}

export function startContainer(hostId: string, id: string): Promise<void> {
  return StartContainer(hostId, id);
}
export function stopContainer(hostId: string, id: string): Promise<void> {
  return StopContainer(hostId, id);
}
export function restartContainer(hostId: string, id: string): Promise<void> {
  return RestartContainer(hostId, id);
}
export function killContainer(
  hostId: string,
  id: string,
  ack: boolean,
): Promise<void> {
  return KillContainer(hostId, id, ack);
}
export function removeContainer(
  hostId: string,
  id: string,
  force: boolean,
  volumes: boolean,
  ack: boolean,
): Promise<void> {
  return RemoveContainer(hostId, id, force, volumes, ack);
}

/** Starts the log stream for a container; lines arrive via onLogLine. */
export function streamLogs(hostId: string, id: string): Promise<void> {
  return StreamContainerLogs(hostId, id);
}
export function stopLogs(id: string): Promise<void> {
  return StopContainerLogs(id);
}
export function onLogLine(
  id: string,
  handler: (line: string) => void,
): () => void {
  return EventsOn(`logs:${id}`, (line: string) => handler(line));
}

/** Starts the stats stream for a container; samples arrive via onStatsSample. */
export function streamStats(hostId: string, id: string): Promise<void> {
  return StreamContainerStats(hostId, id);
}
export function stopStats(id: string): Promise<void> {
  return StopContainerStats(id);
}
export function onStatsSample(
  id: string,
  handler: (s: StatsSample) => void,
): () => void {
  return EventsOn(`stats:${id}`, (s: StatsSample) => handler(s));
}
