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
 * Midnight Jakarta = 17:00 UTC of the previous UTC calendar day.
 *   e.g. 2026-06-16 00:00 WIB = 2026-06-15 17:00 UTC
 */
function midnightJakartaEpoch(year: number, month: number, day: number): number {
  // Subtract the 7-hour offset from 00:00 UTC of the target date to get
  // the UTC instant that corresponds to midnight Jakarta.
  return Date.UTC(year, month, day, 0, 0, 0, 0) - JAKARTA_OFFSET_MS;
}

/**
 * Given a UTC epoch millisecond value, return the Jakarta calendar date parts.
 *
 * Because midnight Jakarta is 7 hours behind UTC midnight of the same calendar
 * date, we add the offset back before reading UTC fields so that the calendar
 * date matches Jakarta's wall clock.
 */
function jakartaDateFromEpoch(epochMs: number): { year: number; month: number; day: number } {
  const shifted = new Date(epochMs + JAKARTA_OFFSET_MS);
  return {
    year: shifted.getUTCFullYear(),
    month: shifted.getUTCMonth(),
    day: shifted.getUTCDate(),
  };
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
  const { year: y, month: m, day: d } = jakartaDateFromEpoch(targetMs);
  return formatYYYYMMDD(y, m, d);
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
  const targetDate = new Date(Date.UTC(year, month - monthsAgo, 1));
  const targetMidnightJKT = midnightJakartaEpoch(
    targetDate.getUTCFullYear(),
    targetDate.getUTCMonth(),
    targetDate.getUTCDate()
  );
  const { year: y, month: m, day: d } = jakartaDateFromEpoch(targetMidnightJKT);
  return formatYYYYMMDD(y, m, d);
}

/**
 * Maximum selectable date for a `<input type="date">` — today in Jakarta.
 * Use as the `max` attribute so users cannot pick a future date that the
 * server (configured for Jakarta) would consider invalid.
 */
export function getMaxDateInJakarta(): string {
  return getTodayInJakarta();
}

/**
 * Gets the Jakarta hour from a UTC date string.
 * Jakarta is UTC+7, so we add 7 hours to get the Jakarta time.
 */
export function getJakartaHourFromUTC(dateString: string): number {
  const date = new Date(dateString);
  // Add 7 hours to convert UTC to Jakarta time
  const jakartaHour = (date.getUTCHours() + 7) % 24;
  return jakartaHour;
}

/**
 * Gets the current Jakarta hour.
 */
export function getCurrentJakartaHour(): number {
  const now = new Date();
  return (now.getUTCHours() + 7) % 24;
}

/**
 * Returns today's date in Jakarta timezone as an object for CalendarDate construction.
 * This should be used by calendar components to get the correct "today" reference.
 */
export function getTodayJakartaDate(): { year: number; month: number; day: number } {
  const { year, month, day } = getJakartaDateParts();
  return { year, month: month + 1, day }; // month is 0-indexed, CalendarDate uses 1-indexed
}

/**
 * Returns the Jakarta day of week (0=Sunday, 1=Monday, ..., 6=Saturday).
 * Used for the 3-day rule threshold calculation.
 * 
 * Per the requirement: Monday=1, Tuesday=2, Wednesday=3, Thursday=4 (threshold day)
 * Weeks starting on Mon-Wed should be disabled (only 1-2 days completed by Thursday)
 * Weeks starting on Thu or later are selectable (3+ days completed)
 */
export function getJakartaDayOfWeek(): number {
  const shifted = new Date(Date.now() + JAKARTA_OFFSET_MS);
  return shifted.getUTCDay(); // 0=Sunday, 1=Monday, ..., 6=Saturday
}

/**
 * Returns the number of completed days in the current week (Jakarta).
 * Counts from Monday to yesterday.
 * 
 * For example:
 * - Sunday (start of week): 0 days completed
 * - Monday: 0 days completed  
 * - Tuesday: 1 day completed (Monday)
 * - Wednesday: 2 days completed (Mon, Tue)
 * - Thursday: 3 days completed (Mon, Tue, Wed) - threshold met
 * - Friday: 4 days completed
 * - Saturday: 5 days completed (full week minus today)
 */
/**
 * Format a UTC ISO date string to Jakarta date display (dd Mon yyyy).
 * Example: "2026-06-16T07:30:00Z" → "16 Jun 2026"
 */
export function formatDateInJakarta(isoString: string): string {
  const date = new Date(isoString);
  if (isNaN(date.getTime())) return '—';
  const jakartaDate = new Date(date.getTime() + JAKARTA_OFFSET_MS);
  const day = jakartaDate.getUTCDate().toString().padStart(2, '0');
  const months = ['Jan','Feb','Mar','Apr','Mei','Jun','Jul','Agu','Sep','Okt','Nov','Des'];
  const month = months[jakartaDate.getUTCMonth()];
  return `${day} ${month} ${jakartaDate.getUTCFullYear()}`;
}

/**
 * Format a UTC ISO date string to Jakarta time display (HH:mm:ss).
 * Example: "2026-06-16T07:30:00Z" → "14:30:00"
 */
export function formatTimeInJakarta(isoString: string): string {
  const date = new Date(isoString);
  if (isNaN(date.getTime())) return '—';
  const jakartaDate = new Date(date.getTime() + JAKARTA_OFFSET_MS);
  const hours = jakartaDate.getUTCHours().toString().padStart(2, '0');
  const minutes = jakartaDate.getUTCMinutes().toString().padStart(2, '0');
  const seconds = jakartaDate.getUTCSeconds().toString().padStart(2, '0');
  return `${hours}:${minutes}:${seconds}`;
}

/**
 * Format a UTC ISO date string to Jakarta datetime display (dd Mon yyyy HH:mm:ss).
 */
export function formatDateTimeInJakarta(isoString: string): string {
  const date = new Date(isoString);
  if (isNaN(date.getTime())) return '—';
  const jakartaDate = new Date(date.getTime() + JAKARTA_OFFSET_MS);
  const day = jakartaDate.getUTCDate().toString().padStart(2, '0');
  const months = ['Jan','Feb','Mar','Apr','Mei','Jun','Jul','Agu','Sep','Okt','Nov','Des'];
  const month = months[jakartaDate.getUTCMonth()];
  const hours = jakartaDate.getUTCHours().toString().padStart(2, '0');
  const minutes = jakartaDate.getUTCMinutes().toString().padStart(2, '0');
  const seconds = jakartaDate.getUTCSeconds().toString().padStart(2, '0');
  return `${day} ${month} ${jakartaDate.getUTCFullYear()} ${hours}:${minutes}:${seconds}`;
}

/**
 * Returns current Jakarta wall-clock time components.
 */
export function getCurrentJakartaClock(): { hours: string; minutes: string; seconds: string } {
  const shifted = new Date(Date.now() + JAKARTA_OFFSET_MS);
  return {
    hours: String(shifted.getUTCHours()).padStart(2, '0'),
    minutes: String(shifted.getUTCMinutes()).padStart(2, '0'),
    seconds: String(shifted.getUTCSeconds()).padStart(2, '0'),
  };
}

/**
 * Returns current Jakarta date as a formatted display object.
 */
export function getCurrentJakartaDateDisplay(): { day: number; month: string; year: number; weekday: string } {
  const shifted = new Date(Date.now() + JAKARTA_OFFSET_MS);
  const weekdays = ['Minggu','Senin','Selasa','Rabu','Kamis','Jumat','Sabtu'];
  const months = ['Januari','Februari','Maret','April','Mei','Juni','Juli','Agustus','September','Oktober','November','Desember'];
  return {
    day: shifted.getUTCDate(),
    month: months[shifted.getUTCMonth()],
    year: shifted.getUTCFullYear(),
    weekday: weekdays[shifted.getUTCDay()],
  };
}

/**
 * Format a YYYY-MM-DD Jakarta date string (e.g. "2026-01-25") to DD Mon YYYY
 * (e.g. "25 Jan 2026").
 */
export function formatJakartaDateStr(dateStr: string): string {
  if (!dateStr) return '';
  const parts = dateStr.split('-');
  if (parts.length !== 3) return dateStr;
  const y = Number(parts[0]);
  const m = Number(parts[1]);
  const d = Number(parts[2]);
  if (!y || !m || !d || m < 1 || m > 12) return dateStr;
  const months = ['Jan','Feb','Mar','Apr','Mei','Jun','Jul','Agu','Sep','Okt','Nov','Des'];
  return `${d} ${months[m - 1]} ${y}`;
}

export function getCompletedDaysInCurrentWeek(): number {
  const dayOfWeek = getJakartaDayOfWeek();
  // If today is Monday (1), no completed days yet (Mon to yesterday = none)
  // If today is Sunday (0), 6 completed days (Mon-Sat)
  // Formula: completed days = (dayOfWeek - 1 + 7) % 7, but at least 0
  // Actually: completed days = days from Monday to yesterday
  // Monday=1 -> yesterday was Sunday(0) of prev week -> need to handle edge case
  
  // Simpler: completed days in current week = days from Monday to yesterday
  // If dayOfWeek === 1 (Monday), yesterday was Sunday of previous week -> 0 days
  // If dayOfWeek === 2 (Tuesday), yesterday was Monday -> 1 day
  // If dayOfWeek === 4 (Thursday), yesterday was Wednesday -> 3 days (threshold met!)
  // If dayOfWeek === 0 (Sunday), yesterday was Saturday -> 6 days
  
  // Completed days = days from Monday to yesterday:
  // Monday=0, Tuesday=1, ..., Saturday=5, Sunday=6
  return (dayOfWeek + 6) % 7;
}

// ---------------------------------------------------------------------------
// Formatting helper
// ---------------------------------------------------------------------------

function formatYYYYMMDD(year: number, month0Indexed: number, day: number): string {
  const mm = String(month0Indexed + 1).padStart(2, '0');
  const dd = String(day).padStart(2, '0');
  return `${year}-${mm}-${dd}`;
}
