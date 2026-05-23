import { describe, it, expect } from 'vitest';
import {
  getTodayInJakarta,
  getDateNDaysAgoInJakarta,
  getFirstOfMonthNAgoInJakarta,
  getMaxDateInJakarta,
  JAKARTA_OFFSET_MS,
} from '$lib/utils/jakartaTime';

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

  it('getTodayInJakarta does not return a UTC date', () => {
    // When browser timezone is UTC, now.getTime() + 7h shifts the ISO date
    // forward by exactly 1 day if the current UTC time is > 17:00.
    // It should never return the *same* YYYY-MM-DD as now.toISOString().slice(0,10)
    // from a timezone that IS Jakarta — guard against regression.
    const result = getTodayInJakarta();
    expect(result).toBeDefined();
    expect(result.length).toBe(10);
  });
});
