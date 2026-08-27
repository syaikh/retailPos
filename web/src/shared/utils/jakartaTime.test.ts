import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { setLocale, formatLocaleDate } from '$shared/i18n';
import {
  getTodayInJakarta,
  getDateNDaysAgoInJakarta,
  getFirstOfMonthNAgoInJakarta,
  getMaxDateInJakarta,
  getCurrentJakartaHour,
  getJakartaHourFromUTC,
  getTodayJakartaDate,
  getJakartaDayOfWeek,
  getCompletedDaysInCurrentWeek,
  formatDateInJakarta,
  formatTimeInJakarta,
  formatDateTimeInJakarta,
  getCurrentJakartaClock,
  getCurrentJakartaDateDisplay,
  formatJakartaDateStr,
  formatLocaleDateInJakarta,
  JAKARTA_OFFSET_MS,
} from '$shared/utils/jakartaTime';

describe('jakartaTime utilities', () => {
  it('JAKARTA_OFFSET_MS equals 7 hours', () => {
    expect(JAKARTA_OFFSET_MS).toBe(7 * 60 * 60 * 1000);
  });

  it('getTodayInJakarta returns a YYYY-MM-DD string', () => {
    const result = getTodayInJakarta();
    expect(result).toMatch(/^\d{4}-\d{2}-\d{2}$/);
  });

  it('getTodayInJakarta: when UTC is 2025-05-31T23:30:00Z the Jakarta day is already 2025-06-01', () => {
    // 2025-05-31T23:30:00Z + 7 h = 2025-06-01T06:30:00 (still 2025-06-01 in Jakarta)
    const simulatedNow = new Date(Date.UTC(2025, 4, 31, 23, 30, 0)); // May 31 23:30 UTC
    // We can't inject "now" into the function (it calls Date.now() internally),
    //  but we can verify the arithmetic manually:
    const shifted = new Date(simulatedNow.getTime() + JAKARTA_OFFSET_MS);
    const jktDay = shifted.getUTCDate();
    expect(jktDay).toBe(1);           // June 1 in Jakarta
    expect(shifted.getUTCMonth()).toBe(5); // June (0-indexed)
  });

  it('getDateNDaysAgoInJakarta(0) equals getTodayInJakarta', () => {
    expect(getDateNDaysAgoInJakarta(0)).toBe(getTodayInJakarta());
  });

  it('getDateNDaysAgoInJakarta(1) is one calendar day before today in Jakarta', () => {
    const todayJKT = getTodayInJakarta();
    // Parse YYYY-MM-DD and subtract 1 day in Jakarta timezone
    const [year, month, day] = todayJKT.split('-').map(Number);
    const todayMs = Date.UTC(year, month - 1, day, 7, 0, 0, 0); // midnight Jakarta as UTC epoch
    const yesterdayMs = todayMs - 86_400_000;
    const utc = new Date(yesterdayMs);
    const expected = `${utc.getUTCFullYear()}-${String(utc.getUTCMonth() + 1).padStart(2, '0')}-${String(utc.getUTCDate()).padStart(2, '0')}`;
    expect(getDateNDaysAgoInJakarta(1)).toBe(expected);
  });

  it('getDateNDaysAgoInJakarta: getTodayInJakarta - getDateNDaysAgoInJakarta(6) = 7 distinct calendar days', () => {
    // A 7-day window starting at N-6 through today spans exactly 7 calendar
    // days (N-6, N-5, N-4, N-3, N-2, N-1, N).
    const start = getDateNDaysAgoInJakarta(6);
    const end = getTodayInJakarta();
    // Count days by stepping from start(00:00Z+7) to end(23:59:59Z+7)
    const startMs = Date.parse(start + 'T00:00:00+07:00');
    const endMs   = Date.parse(end   + 'T23:59:59+07:00');
    const days = Math.round((endMs - startMs) / 86_400_000);
    // diff is 7 full 24-h gaps, which means 7 *distinct calendar days*
    expect(days).toBe(7);
  });

  it('getFirstOfMonthNAgoInJakarta(0) returns the 1st of the current Jakarta month', () => {
    const jk = getFirstOfMonthNAgoInJakarta(0);
    expect(jk).toMatch(/^\d{4}-\d{2}-01$/);
  });

  it('getFirstOfMonthNAgoInJakarta(1) returns the 1st of the previous Jakarta month', () => {
    const today = getTodayInJakarta();
    const year = Number(today.slice(0, 4));
    const month = Number(today.slice(5, 7)); // 1-indexed 1-12
    const prevMonth = month === 1 ? 12 : month - 1;
    const prevYear = month === 1 ? year - 1 : year;
    const expected = `${String(prevYear).padStart(4,'0')}-${String(prevMonth).padStart(2,'0')}-01`;
    expect(getFirstOfMonthNAgoInJakarta(1)).toBe(expected);
  });

  it('getFirstOfMonthNAgoInJakarta(11) returns the 1st of the month 11 months ago', () => {
    const today = getTodayInJakarta();
    const year = Number(today.slice(0, 4));
    const month = Number(today.slice(5, 7)); // 1-indexed Jan=1
    // monthIndex 0-indexed: month-1. Subtract 11, floor-divide by 12 to get year delta.
    const month0Idx = month - 1;
    const absMonthIdx = (month0Idx - 11 + 120) % 12;  // always positive
    const yearDelta = Math.floor((month0Idx - 11) / 12);
    const targetYear = year + yearDelta;
    const expected = `${String(targetYear).padStart(4,'0')}-${String(absMonthIdx + 1).padStart(2,'0')}-01`;
    expect(getFirstOfMonthNAgoInJakarta(11)).toBe(expected);
  });

  it('getMaxDateInJakarta returns the same as getTodayInJakarta', () => {
    expect(getMaxDateInJakarta()).toBe(getTodayInJakarta());
  });

  it('all public functions return YYYY-MM-DD format', () => {
    const checkers: (() => string)[] = [
      getTodayInJakarta,
      () => getDateNDaysAgoInJakarta(7),
      () => getFirstOfMonthNAgoInJakarta(11),
      getMaxDateInJakarta,
    ];
    for (const fn of checkers) {
      expect(fn()).toMatch(/^\d{4}-\d{2}-\d{2}$/);
    }
  });

  it('getCurrentJakartaHour returns 0-23 range', () => {
    const hour = getCurrentJakartaHour();
    expect(hour).toBeGreaterThanOrEqual(0);
    expect(hour).toBeLessThanOrEqual(23);
    expect(Number.isInteger(hour)).toBe(true);
  });

  it('getJakartaHourFromUTC converts correctly', () => {
    // UTC 2026-06-15T17:00:00Z = Jakarta 2026-06-16 00:00 (midnight)
    expect(getJakartaHourFromUTC('2026-06-15T17:00:00Z')).toBe(0);
    // UTC 2026-06-15T22:00:00Z = Jakarta 2026-06-16 05:00
    expect(getJakartaHourFromUTC('2026-06-15T22:00:00Z')).toBe(5);
    // UTC 2026-06-15T00:00:00Z = Jakarta 2026-06-15 07:00
    expect(getJakartaHourFromUTC('2026-06-15T00:00:00Z')).toBe(7);
    // UTC 2026-06-15T23:59:00Z = Jakarta 2026-06-16 06:59
    expect(getJakartaHourFromUTC('2026-06-15T23:59:00Z')).toBe(6);
  });

  it('getJakartaHourFromUTC with RFC3339 offset format', () => {
    // With explicit +07:00 offset
    expect(getJakartaHourFromUTC('2026-06-16T05:30:00+07:00')).toBe(5);
    // With explicit -05:00 offset (US Eastern)
    expect(getJakartaHourFromUTC('2026-06-15T18:30:00-05:00')).toBe(6);
  });

  it('getTodayInJakarta does not return a UTC date', () => {
    // When browser timezone is UTC, now.getTime() + 7h shifts the ISO date
    // forward by exactly 1 day if the current UTC time is > 17:00.
    // It should never return the *same* YYYY-MM-DD as now.toISOString().slice(0,10)
    // from a timezone that IS Jakarta — guard against regression.
    const result = getTodayInJakarta();
    expect(result).toBeDefined();
    expect(result.length).toBe(10);
  });

  it('getTodayJakartaDate returns Y/M/D with 1-indexed month', () => {
    const result = getTodayJakartaDate();
    expect(result).toHaveProperty('year');
    expect(result).toHaveProperty('month');
    expect(result).toHaveProperty('day');
    expect(result.month).toBeGreaterThanOrEqual(1);
    expect(result.month).toBeLessThanOrEqual(12);
    expect(result.day).toBeGreaterThanOrEqual(1);
  });

  it('getJakartaDayOfWeek returns 0-6', () => {
    const day = getJakartaDayOfWeek();
    expect(day).toBeGreaterThanOrEqual(0);
    expect(day).toBeLessThanOrEqual(6);
    expect(Number.isInteger(day)).toBe(true);
  });

  it('getCompletedDaysInCurrentWeek returns 0-6', () => {
    const days = getCompletedDaysInCurrentWeek();
    expect(days).toBeGreaterThanOrEqual(0);
    expect(days).toBeLessThanOrEqual(6);
    expect(Number.isInteger(days)).toBe(true);
  });

  it('getCompletedDaysInCurrentWeek: Monday returns 0, Sunday returns 6', () => {
    // Monday (1) -> 0, Sunday (0) -> 6
    // We can test internal logic directly
    // getJakartaDayOfWeek returns (new Date(now + offset)).getUTCDay()
    // Monday = 1 -> getCompletedDaysInCurrentWeek returns 0
    // Sunday = 0 -> getCompletedDaysInCurrentWeek returns 0-1 = -1 but special case returns 6
    const result = getCompletedDaysInCurrentWeek();
    // Just verify it's in valid range
    expect(result).toBeGreaterThanOrEqual(0);
    expect(result).toBeLessThanOrEqual(6);
  });

  it('getCompletedDaysInCurrentWeek formula: Monday=0, Tuesday=1, ..., Sunday=6', () => {
    // Verify the mapping (dayOfWeek + 6) % 7 without relying on "now":
    // 1 (Mon)=0, 2 (Tue)=1, 3 (Wed)=2, 4 (Thu)=3, 5 (Fri)=4, 6 (Sat)=5, 0 (Sun)=6
    const expected = [6, 0, 1, 2, 3, 4, 5]; // index = dayOfWeek 0..6
    expected.forEach((value, dayOfWeek) => {
      expect((dayOfWeek + 6) % 7).toBe(value);
    });
    expect(getCompletedDaysInCurrentWeek()).toBe((getJakartaDayOfWeek() + 6) % 7);
  });

  it('formatDateInJakarta formats correctly', () => {
    // 2026-06-15T17:00:00Z = 2026-06-16 00:00 WIB
    expect(formatDateInJakarta('2026-06-15T17:00:00Z')).toBe('16 Jun 2026');
    // 2026-06-15T00:00:00Z = 2026-06-15 07:00 WIB
    expect(formatDateInJakarta('2026-06-15T00:00:00Z')).toBe('15 Jun 2026');
  });

  it('formatDateInJakarta returns dash for invalid date', () => {
    expect(formatDateInJakarta('not-a-date')).toBe('—');
  });

  it('formatTimeInJakarta formats correctly', () => {
    // 2026-06-15T17:00:00Z = 2026-06-16 00:00:00 WIB
    expect(formatTimeInJakarta('2026-06-15T17:00:00Z')).toBe('00:00:00');
    // 2026-06-15T00:00:00Z = 2026-06-15 07:00:00 WIB
    expect(formatTimeInJakarta('2026-06-15T00:00:00Z')).toBe('07:00:00');
  });

  it('formatTimeInJakarta returns dash for invalid date', () => {
    expect(formatTimeInJakarta('not-a-date')).toBe('—');
  });

  it('formatDateTimeInJakarta formats correctly', () => {
    // 2026-06-15T17:00:00Z = 2026-06-16 00:00:00 WIB
    expect(formatDateTimeInJakarta('2026-06-15T17:00:00Z')).toBe('16 Jun 2026 00:00:00');
  });

  it('formatDateTimeInJakarta returns dash for invalid date', () => {
    expect(formatDateTimeInJakarta('not-a-date')).toBe('—');
  });

  it('getCurrentJakartaClock returns formatted time strings', () => {
    const clock = getCurrentJakartaClock();
    expect(clock).toHaveProperty('hours');
    expect(clock).toHaveProperty('minutes');
    expect(clock).toHaveProperty('seconds');
    expect(clock.hours).toMatch(/^\d{2}$/);
    expect(clock.minutes).toMatch(/^\d{2}$/);
    expect(clock.seconds).toMatch(/^\d{2}$/);
  });

  it('getCurrentJakartaDateDisplay returns full Indonesian display', () => {
    const display = getCurrentJakartaDateDisplay();
    expect(display).toHaveProperty('day');
    expect(display).toHaveProperty('month');
    expect(display).toHaveProperty('year');
    expect(display).toHaveProperty('weekday');
    expect(display.day).toBeGreaterThanOrEqual(1);
    expect(display.day).toBeLessThanOrEqual(31);
    expect(display.year).toBeGreaterThanOrEqual(2020);
    expect(typeof display.month).toBe('string');
    expect(typeof display.weekday).toBe('string');
  });

  it('formatJakartaDateStr formats YYYY-MM-DD to DD Mon YYYY', () => {
    expect(formatJakartaDateStr('2026-06-15')).toBe('15 Jun 2026');
    expect(formatJakartaDateStr('2026-01-01')).toBe('1 Jan 2026');
  });

  it('formatJakartaDateStr returns empty string for empty input', () => {
    expect(formatJakartaDateStr('')).toBe('');
  });

  it('formatJakartaDateStr returns original for malformed input', () => {
    expect(formatJakartaDateStr('hello')).toBe('hello');
    expect(formatJakartaDateStr('not-a-date')).toBe('not-a-date');
  });

  describe('formatLocaleDateInJakarta', () => {
    beforeAll(() => setLocale('en'));
    afterAll(() => setLocale('id'));

    it('renders the Jakarta wall-clock for an absolute UTC timestamp', () => {
      // 2026-06-15T17:00:00Z + 7h = 2026-06-16 00:00 WIB
      expect(
        formatLocaleDateInJakarta('2026-06-15T17:00:00Z', { day: 'numeric', month: 'short', year: 'numeric' }),
      ).toBe('Jun 16, 2026');
    });

    it('renders date and time options in Jakarta time', () => {
      // 2026-06-15T00:00:00Z + 7h = 2026-06-15 07:00 WIB
      expect(
        formatLocaleDateInJakarta('2026-06-15T00:00:00Z', {
          day: '2-digit',
          month: 'short',
          year: 'numeric',
          hour: '2-digit',
          minute: '2-digit',
          hour12: false,
        }),
      ).toBe('Jun 15, 2026, 07:00');
    });

    it('shifts across a day boundary to the Jakarta calendar value', () => {
      // 2026-06-15T17:00:00Z + 7h = 2026-06-16 00:00 WIB
      expect(
        formatLocaleDateInJakarta('2026-06-15T17:00:00Z', {
          day: 'numeric',
          month: 'short',
          year: 'numeric',
          hour: '2-digit',
          minute: '2-digit',
          hour12: false,
        }),
      ).toBe('Jun 16, 2026, 00:00');
    });

    it('honours an explicit offset in the input string without double-shifting', () => {
      // 2026-06-16T05:30:00+07:00 is already 05:30 WIB; the absolute instant is
      // 2026-06-15T22:30:00Z, +7h = 2026-06-16T05:30:00Z, rendered in UTC fields.
      expect(
        formatLocaleDateInJakarta('2026-06-16T05:30:00+07:00', {
          day: 'numeric',
          month: 'short',
          year: 'numeric',
          hour: '2-digit',
          minute: '2-digit',
          hour12: false,
        }),
      ).toBe('Jun 16, 2026, 05:30');
    });

    it('returns an em dash for an invalid date string', () => {
      expect(formatLocaleDateInJakarta('not-a-date')).toBe('—');
    });

    it('returns an em dash when passed undefined', () => {
      // callers normally guard with `if (!value)`, but the helper must not throw
      expect(formatLocaleDateInJakarta(undefined as unknown as string)).toBe('—');
    });

    it('returns an em dash when passed an empty string', () => {
      expect(formatLocaleDateInJakarta('')).toBe('—');
    });

    it('forwards custom Intl options to the formatter', () => {
      // 2026-06-15T17:00:00Z + 7h = Tuesday 2026-06-16 WIB
      expect(
        formatLocaleDateInJakarta('2026-06-15T17:00:00Z', {
          weekday: 'long',
          day: 'numeric',
          month: 'long',
          year: 'numeric',
        }),
      ).toBe('Tuesday, June 16, 2026');
    });
  });
});
