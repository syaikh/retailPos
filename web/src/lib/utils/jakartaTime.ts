/**
 * Timezone utilities for the application.
 *
 * All date-picker defaults, API date-strings, and range calculations must be
 * expressed in Asia/Jakarta (UTC+7) so they are consistent with the backend
 * session timezone (set via the DSN `timezone=Asia/Jakarta` parameter).
 *
 * **Core principle:** all arithmetic happens in UTC. We derive Jakarta
 * calendar values from UTC fields, never from the browser's local timezone.
 *
 * Mapping:  midnight Jakarta  = 07:00 UTC  (because 00:00 + 07:00 = 07:00 UTC)
 */

/** 7 hours expressed in milliseconds */
export const JAKARTA_OFFSET_MS = 7 * 60 * 60 * 1000;

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

/**
 * Returns the current Jakarta *calendar* date as { year, month, day } in UTC.
 *
 * Algorithm: shift browser now by +7 h → read the UTC fields of the shifted
 * instant.  Those UTC fields *are* the Jakarta calendar values because:
 *   JKT midnight  → UTC 07:00  (getUTCDate = Jakarta day, getUTCMonth = Jakarta month)
 */
function getJakartaDateParts(): { year: number; month: number; day: number } {
  const shifted = new Date(Date.now() + JAKARTA_OFFSET_MS);
  return {
    year: shifted.getUTCFullYear(),
    month: shifted.getUTCMonth(),  // 0-indexed
    day: shifted.getUTCDate(),
  };
}

/**
 * Returns the UTC epoch milliseconds of midnight (00:00) on a given Jakarta
 * calendar date.
 *
 * Midnight Jakarta = 07:00 UTC of that same calendar date.
 */
function midnightJakartaEpoch(year: number, month: number, day: number): number {
  // month is already 0-indexed; Date.UTC uses 0-indexed months.
  // 07:00 UTC = 00:00 WIB (Jakarta)
  return Date.UTC(year, month, day, 7, 0, 0, 0);
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

/**
 * Returns the current wall-clock date in Asia/Jakarta as YYYY-MM-DD.
 * Correct even when the browser's timezone is not Jakarta.
 *
 * Example: When it is 2025-06-01 00:00 JKT (= 2025-05-31 17:00 UTC),
 * this returns "2025-06-01", never "2025-05-31".
 */
export function getTodayInJakarta(): string {
  const { year, month, day } = getJakartaDateParts();
  return formatYYYYMMDD(year, month, day);
}

/**
 * Returns the date that was `daysAgo` full days *before today in Jakarta*,
 * formatted as YYYY-MM-DD.
 *
 * daysAgo = 0  → today   in Jakarta
 * daysAgo = 6  → 6 days ago in Jakarta (start of a 7-day window)
 * daysAgo = 84 → 12 weeks ago in Jakarta
 *
 * All arithmetic is done in UTC epoch milliseconds — no local-timezone
 * ambiguity.
 */
export function getDateNDaysAgoInJakarta(daysAgo: number): string {
  const { year, month, day } = getJakartaDateParts();
  const msPerDay = 86_400_000;
  const todayMidnightJKT = midnightJakartaEpoch(year, month, day);
  const targetMs = todayMidnightJKT - daysAgo * msPerDay; // safe: daysAgo ≤ ~365, result < 2^53
  const utc = new Date(targetMs);
  return formatYYYYMMDD(utc.getUTCFullYear(), utc.getUTCMonth(), utc.getUTCDate());
}

/**
 * Returns the 1st day of the Jakarta month that is `monthsAgo` months
 * *before* the current Jakarta month, formatted as YYYY-MM-DD.
 *
 * monthsAgo = 0  → 1st of the current Jakarta month
 * monthsAgo = 1  → 1st of last month  in Jakarta
 * monthsAgo = 11 → 1st of the month 11 months ago in Jakarta
 *
 * All arithmetic is done in UTC epoch milliseconds — no local-timezone
 * ambiguity.
 */
export function getFirstOfMonthNAgoInJakarta(monthsAgo: number): string {
  const { year, month } = getJakartaDateParts();
  // Reconstruct a Date in UTC with month shifted back by monthsAgo, day = 1
  const firstOfTargetMonth = new Date(Date.UTC(year, month - monthsAgo, 1, 7, 0, 0, 0));
  return formatYYYYMMDD(
    firstOfTargetMonth.getUTCFullYear(),
    firstOfTargetMonth.getUTCMonth(),
    firstOfTargetMonth.getUTCDate(),
  );
}

/**
 * Maximum selectable date for a `<input type="date">` — today in Jakarta.
 * Use as the `max` attribute so users cannot pick a future date that the
 * server (configured for Jakarta) would consider invalid.
 */
export function getMaxDateInJakarta(): string {
  return getTodayInJakarta();
}

// ---------------------------------------------------------------------------
// Formatting helper
// ---------------------------------------------------------------------------

function formatYYYYMMDD(year: number, month0Indexed: number, day: number): string {
  const mm = String(month0Indexed + 1).padStart(2, '0');
  const dd = String(day).padStart(2, '0');
  return `${year}-${mm}-${dd}`;
}
