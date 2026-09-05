/**
 * Format a duration between two ISO timestamps as a human-readable string.
 * Returns '-' if closedAt is null or the duration is negative.
 */
export function formatDuration(openedAt: string, closedAt: string | null): string {
  if (!closedAt) return '-';
  const opened = new Date(openedAt);
  const closed = new Date(closedAt);
  const diffMs = closed.getTime() - opened.getTime();
  if (diffMs < 0) return '-';
  const totalMinutes = Math.floor(diffMs / 60000);
  const h = Math.floor(totalMinutes / 60);
  const m = totalMinutes % 60;
  if (h === 0) return `${m}m`;
  if (m === 0) return `${h}h`;
  return `${h}h ${m}m`;
}
