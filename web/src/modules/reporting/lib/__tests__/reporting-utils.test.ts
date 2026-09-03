import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { setLocale } from '$shared/i18n';

describe('reporting-utils', () => {
  beforeAll(() => setLocale('id'));
  afterAll(() => setLocale('en'));

  describe('formatCurrencyShort', () => {
    it('formats billions', async () => {
      const { formatCurrencyShort } = await import('../reporting-utils');
      expect(formatCurrencyShort(1500000000)).toBe('Rp 1.5M');
    });

    it('formats millions', async () => {
      const { formatCurrencyShort } = await import('../reporting-utils');
      expect(formatCurrencyShort(2500000)).toBe('Rp 2.5jt');
    });

    it('formats thousands', async () => {
      const { formatCurrencyShort } = await import('../reporting-utils');
      expect(formatCurrencyShort(5000)).toBe('Rp 5k');
    });

    it('formats small values', async () => {
      const { formatCurrencyShort } = await import('../reporting-utils');
      expect(formatCurrencyShort(999)).toBe('Rp 999');
    });
  });

  describe('formatLargeNumber', () => {
    it('formats billions', async () => {
      const { formatLargeNumber } = await import('../reporting-utils');
      expect(formatLargeNumber(2000000000)).toBe('2M');
    });

    it('formats millions', async () => {
      const { formatLargeNumber } = await import('../reporting-utils');
      expect(formatLargeNumber(3500000)).toBe('3.5jt');
    });

    it('formats thousands', async () => {
      const { formatLargeNumber } = await import('../reporting-utils');
      expect(formatLargeNumber(7500)).toBe('8k');
    });
  });

  describe('formatDate', () => {
    it('formats date string', async () => {
      const { formatDate } = await import('../reporting-utils');
      const result = formatDate('2026-06-22');
      expect(result).toContain('22');
      expect(result).toContain('2026');
    });

    it('returns empty for undefined', async () => {
      const { formatDate } = await import('../reporting-utils');
      expect(formatDate()).toBe('');
    });
  });

  describe('getPeriodLabel', () => {
    it('formats hour item', async () => {
      const { getPeriodLabel } = await import('../reporting-utils');
      expect(getPeriodLabel({ hour: 5 })).toBe('05:00');
    });

    it('formats date item', async () => {
      const { getPeriodLabel } = await import('../reporting-utils');
      const result = getPeriodLabel({ date: '2026-06-15' });
      expect(result).toContain('Jun');
      expect(result).toContain('15');
    });

    it('returns label fallback', async () => {
      const { getPeriodLabel } = await import('../reporting-utils');
      expect(getPeriodLabel({ label: 'Week 1' })).toBe('Week 1');
    });

    it('returns empty for null', async () => {
      const { getPeriodLabel } = await import('../reporting-utils');
      expect(getPeriodLabel(null as unknown as { hour?: number; date?: string; month_start?: string; label?: string })).toBe('');
    });
  });

  describe('getPeriodDateRange', () => {
    it('returns today for realtime', async () => {
      const { getPeriodDateRange } = await import('../reporting-utils');
      const { getTodayInJakarta } = await import('$shared/utils/jakartaTime');
      const result = getPeriodDateRange('realtime');
      expect(result.start).toBe(getTodayInJakarta());
      expect(result.end).toBe(getTodayInJakarta());
    });

    it('returns yesterday range', async () => {
      const { getPeriodDateRange } = await import('../reporting-utils');
      const { getDateNDaysAgoInJakarta } = await import('$shared/utils/jakartaTime');
      const yesterday = getDateNDaysAgoInJakarta(1);
      const result = getPeriodDateRange('yesterday');
      expect(result.start).toBe(yesterday);
      expect(result.end).toBe(yesterday);
    });

    it('returns 7days range', async () => {
      const { getPeriodDateRange } = await import('../reporting-utils');
      const { getDateNDaysAgoInJakarta } = await import('$shared/utils/jakartaTime');
      const result = getPeriodDateRange('7days');
      expect(result.start).toBe(getDateNDaysAgoInJakarta(7));
      expect(result.end).toBe(getDateNDaysAgoInJakarta(1));
    });
  });

  describe('getBackendPeriodType', () => {
    it('maps period types correctly', async () => {
      const { getBackendPeriodType } = await import('../reporting-utils');
      expect(getBackendPeriodType('realtime')).toBe('daily');
      expect(getBackendPeriodType('weekly')).toBe('weekly');
      expect(getBackendPeriodType('monthly')).toBe('monthly');
      expect(getBackendPeriodType('yearly')).toBe('yearly');
    });
  });

  describe('getComparisonMode', () => {
    it('maps modes correctly', async () => {
      const { getComparisonMode } = await import('../reporting-utils');
      expect(getComparisonMode('realtime')).toBe('realtime');
      expect(getComparisonMode('daily')).toBe('completed');
      expect(getComparisonMode('yearly')).toBe('todate');
      expect(getComparisonMode('30days')).toBe('30days');
    });
  });

  describe('getShiftDays', () => {
    it('returns correct shift days', async () => {
      const { getShiftDays } = await import('../reporting-utils');
      expect(getShiftDays('realtime')).toBe(1);
      expect(getShiftDays('weekly')).toBe(7);
      expect(getShiftDays('30days')).toBe(30);
      expect(getShiftDays('yearly')).toBe(0);
    });
  });

  describe('getFirstOfMonthNAgoInJakarta', () => {
    it('returns YYYY-MM-01 format for current month', async () => {
      const { getFirstOfMonthNAgoInJakarta } = await import('../reporting-utils');
      const result = getFirstOfMonthNAgoInJakarta(0);
      expect(result).toMatch(/^\d{4}-\d{2}-01$/);
    });

    it('returns first of previous month', async () => {
      const { getFirstOfMonthNAgoInJakarta, formatDate } = await import('../reporting-utils');
      const result = getFirstOfMonthNAgoInJakarta(1);
      expect(result).toMatch(/^\d{4}-\d{2}-01$/);
      const currentFirst = getFirstOfMonthNAgoInJakarta(0);
      expect(result).not.toBe(currentFirst);
    });
  });

  describe('formatDate', () => {
    it('returns empty string for falsy input', async () => {
      const { formatDate } = await import('../reporting-utils');
      expect(formatDate('')).toBe('');
      expect(formatDate(undefined)).toBe('');
    });

    it('formats date string correctly', async () => {
      const { formatDate } = await import('../reporting-utils');
      const result = formatDate('2026-06-15');
      expect(result).toContain('Jun');
      expect(result).toContain('2026');
      expect(result).toContain('15');
    });
  });

  describe('getPeriodLabel', () => {
    it('returns empty string for null', async () => {
      const { getPeriodLabel } = await import('../reporting-utils');
      expect(getPeriodLabel(null as unknown as { hour?: number; date?: string; month_start?: string; label?: string })).toBe('');
    });

    it('formats hourly items', async () => {
      const { getPeriodLabel } = await import('../reporting-utils');
      expect(getPeriodLabel({ hour: 3 })).toBe('03:00');
      expect(getPeriodLabel({ hour: 14 })).toBe('14:00');
    });

    it('formats daily items', async () => {
      const { getPeriodLabel } = await import('../reporting-utils');
      const result = getPeriodLabel({ date: '2026-06-15' });
      expect(result).toContain('Jun');
      expect(result).toContain('15');
    });

    it('formats monthly items', async () => {
      const { getPeriodLabel } = await import('../reporting-utils');
      const result = getPeriodLabel({ month_start: '2026-06-01' });
      expect(result).toContain('Jun');
      expect(result).toContain('2026');
    });

    it('renders correct date regardless of browser timezone', async () => {
      const { getPeriodLabel } = await import('../reporting-utils');
      const result = getPeriodLabel({ date: '2026-01-01' });
      expect(result).toBe('1 Jan');
    });
  });

  describe('formatDayDate', () => {
    it('returns empty string for falsy input', async () => {
      const { formatDayDate } = await import('../reporting-utils');
      expect(formatDayDate('')).toBe('');
      expect(formatDayDate(undefined)).toBe('');
      expect(formatDayDate(null as unknown as string)).toBe('');
    });

    it('formats date as "Day, DD Mon"', async () => {
      const { formatDayDate } = await import('../reporting-utils');
      const result = formatDayDate('2026-07-14');
      expect(result).toBe('Sel, 14 Jul');
    });

    it('formats a Sunday correctly', async () => {
      const { formatDayDate } = await import('../reporting-utils');
      const result = formatDayDate('2026-07-13');
      expect(result).toBe('Sen, 13 Jul');
    });

    it('formats year boundaries', async () => {
      const { formatDayDate } = await import('../reporting-utils');
      const result = formatDayDate('2026-01-01');
      expect(result).toBe('Kam, 1 Jan');
    });

    it('formats month boundaries', async () => {
      const { formatDayDate } = await import('../reporting-utils');
      const result = formatDayDate('2026-06-01');
      expect(result).toBe('Sen, 1 Jun');
    });

    it('does not include year', async () => {
      const { formatDayDate } = await import('../reporting-utils');
      const result = formatDayDate('2026-12-25');
      expect(result).not.toContain('2026');
      expect(result).toBe('Jum, 25 Des');
    });
  });
});
