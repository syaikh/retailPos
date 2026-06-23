import { describe, it, expect } from 'vitest';

describe('reporting-utils', () => {
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
      expect(getPeriodLabel({ hour: 5, total: 100 })).toBe('05:00');
    });

    it('formats date item', async () => {
      const { getPeriodLabel } = await import('../reporting-utils');
      const result = getPeriodLabel({ date: '2026-06-15', total: 100 });
      expect(result).toContain('Jun');
      expect(result).toContain('15');
    });

    it('returns label fallback', async () => {
      const { getPeriodLabel } = await import('../reporting-utils');
      expect(getPeriodLabel({ label: 'Week 1', total: 100 })).toBe('Week 1');
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

  describe('getChartType', () => {
    it('returns hourly for realtime/yesterday/daily', async () => {
      const { getChartType } = await import('../reporting-utils');
      expect(getChartType('realtime')).toBe('hourly');
      expect(getChartType('yesterday')).toBe('hourly');
      expect(getChartType('daily')).toBe('hourly');
    });

    it('returns daily for 7days/30days/weekly/monthly', async () => {
      const { getChartType } = await import('../reporting-utils');
      expect(getChartType('7days')).toBe('daily');
      expect(getChartType('30days')).toBe('daily');
      expect(getChartType('weekly')).toBe('daily');
      expect(getChartType('monthly')).toBe('daily');
    });

    it('returns yearly for yearly', async () => {
      const { getChartType } = await import('../reporting-utils');
      expect(getChartType('yearly')).toBe('yearly');
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
});
