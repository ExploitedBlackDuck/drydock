// Small presentation helpers shared across views.

/** Formats a byte count as a human-readable size. A negative size is unknown. */
export function formatBytes(bytes: number): string {
  if (bytes < 0) return '—';
  if (bytes === 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const exp = Math.min(
    Math.floor(Math.log(bytes) / Math.log(1024)),
    units.length - 1,
  );
  const value = bytes / Math.pow(1024, exp);
  return `${value.toFixed(exp === 0 ? 0 : 1)} ${units[exp]}`;
}

/** Shortens a 64-hex id to its first 12 characters (Docker's short id). */
export function shortId(id: string): string {
  const hex = id.startsWith('sha256:') ? id.slice('sha256:'.length) : id;
  return hex.slice(0, 12);
}

/**
 * Formats a backend timestamp (an RFC3339 string, as Wails delivers time.Time)
 * as a compact local date-time. Returns an em dash for a missing/zero time.
 */
export function formatTimestamp(value: unknown): string {
  if (!value || typeof value !== 'string') return '—';
  const d = new Date(value);
  if (Number.isNaN(d.getTime()) || d.getUTCFullYear() <= 1) return '—';
  return d.toLocaleString(undefined, {
    year: 'numeric',
    month: 'short',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
}
