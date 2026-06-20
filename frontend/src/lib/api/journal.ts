// Typed seam over the history/audit bindings (PROJECT-BOOK §7.11.8, §2.8).
import {
  AuditTrail,
  ExportJournal,
  OperationHistory,
} from '../../../wailsjs/go/app/App';
import type { domain, journal } from '../../../wailsjs/go/models';

export type Operation = domain.Operation;
export type AuditEntry = domain.AuditEntry;
export type AuditStatus = journal.AuditStatus;

/** Recorded operations for a host, optionally narrowed by kind or destructiveness. */
export function operationHistory(
  hostId: string,
  kind: string,
  destructiveOnly: boolean,
  limit: number,
): Promise<Operation[]> {
  return OperationHistory(hostId, kind, destructiveOnly, limit);
}

/** The audit log plus its chain-verification result. */
export function auditTrail(): Promise<AuditStatus> {
  return AuditTrail();
}

/**
 * Exports the full accountability record (every operation + the audit chain) and
 * triggers a download of the JSON file from the webview — no backend file APIs.
 */
export async function downloadJournalExport(): Promise<void> {
  const json = await ExportJournal();
  const blob = new Blob([json], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  try {
    const a = document.createElement('a');
    a.href = url;
    a.download = 'drydock-journal.json';
    document.body.appendChild(a);
    a.click();
    a.remove();
  } finally {
    URL.revokeObjectURL(url);
  }
}
