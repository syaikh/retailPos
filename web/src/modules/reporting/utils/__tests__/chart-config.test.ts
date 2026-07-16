import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('$shared/utils/jakartaTime', () => ({
  getCurrentJakartaHour: vi.fn(() => 14),
  getTodayInJakarta: vi.fn(() => '2026-07-16'),
  getDateNDaysAgoInJakarta: vi.fn((n: number) => {
    const d = new Date('2026-07-16T00:00:00Z');
    d.setUTCDate(d.getUTCDate() - n);
    return d.toISOString().split('T')[0];
  }),
}));

import { buildChartConfig } from '../chart-config';
import { getCurrentJakartaHour } from '$shared/utils/jakartaTime';

function makeHourlyData(hours: number[]) {
  return hours.map(h => ({ date: String(h), total: (h + 1) * 1000 }));
}

describe('buildChartConfig – hourly chart hour range', () => {
  const baseParams = {
    chartData: [],
    prevChartData: [],
    endDate: '2026-07-16',
    selectedMonthlyRange: undefined,
    selectedYearlyRange: undefined,
    chartYear: 2026,
  };

  beforeEach(() => {
    vi.mocked(getCurrentJakartaHour).mockReturnValue(14);
  });

  it('shows 24 hours (0–23) for yesterday period', () => {
    const chartData = makeHourlyData([0, 1, 5, 10, 23]);
    const config = buildChartConfig({
      ...baseParams,
      chartType: 'hourly',
      chartData,
      activePeriodType: 'yesterday',
    });

    expect(config.data.labels).toHaveLength(24);
    expect(config.data.labels[0]).toBe('00:00');
    expect(config.data.labels[23]).toBe('23:00');
  });

  it('shows 24 hours (0–23) for daily period', () => {
    const chartData = makeHourlyData([0, 1, 5, 14]);
    const config = buildChartConfig({
      ...baseParams,
      chartType: 'hourly',
      chartData,
      activePeriodType: 'daily',
    });

    expect(config.data.labels).toHaveLength(24);
    expect(config.data.labels[0]).toBe('00:00');
    expect(config.data.labels[23]).toBe('23:00');
  });

  it('shows only 0–currentHour for realtime period', () => {
    vi.mocked(getCurrentJakartaHour).mockReturnValue(14);
    const chartData = makeHourlyData([0, 5, 10, 14]);
    const config = buildChartConfig({
      ...baseParams,
      chartType: 'hourly',
      chartData,
      activePeriodType: 'realtime',
    });

    expect(config.data.labels).toHaveLength(15);
    expect(config.data.labels[0]).toBe('00:00');
    expect(config.data.labels[14]).toBe('14:00');
  });

  it('fills zero for missing hours in yesterday data', () => {
    const chartData = [{ date: '0', total: 5000 }, { date: '23', total: 3000 }];
    const config = buildChartConfig({
      ...baseParams,
      chartType: 'hourly',
      chartData,
      activePeriodType: 'yesterday',
    });

    const values = config.data.datasets[0].data;
    expect(values).toHaveLength(24);
    expect(values[0]).toBe(5000);
    expect(values[1]).toBe(0);
    expect(values[22]).toBe(0);
    expect(values[23]).toBe(3000);
  });

  it('fills zero for missing hours in daily data', () => {
    const chartData = [{ date: '8', total: 7000 }];
    const config = buildChartConfig({
      ...baseParams,
      chartType: 'hourly',
      chartData,
      activePeriodType: 'daily',
    });

    const values = config.data.datasets[0].data;
    expect(values).toHaveLength(24);
    expect(values[8]).toBe(7000);
    expect(values[0]).toBe(0);
    expect(values[23]).toBe(0);
  });

  it('shows only 0–currentHour for non-completed periods like 7days', () => {
    vi.mocked(getCurrentJakartaHour).mockReturnValue(14);
    const chartData = makeHourlyData([0, 5, 14, 23]);
    const config = buildChartConfig({
      ...baseParams,
      chartType: 'hourly',
      chartData,
      activePeriodType: '7days',
    });

    expect(config.data.labels).toHaveLength(15);
    expect(config.data.labels[0]).toBe('00:00');
    expect(config.data.labels[14]).toBe('14:00');
    expect(config.data.labels).not.toContain('23:00');
  });

  it('uses line chart type for hourly', () => {
    const config = buildChartConfig({
      ...baseParams,
      chartType: 'hourly',
      chartData: makeHourlyData([0, 1, 2]),
      activePeriodType: 'yesterday',
    });

    expect(config.type).toBe('line');
  });
});

describe('buildChartConfig – previous period data for hourly', () => {
  const baseParams = {
    chartData: [],
    endDate: '2026-07-16',
    selectedMonthlyRange: undefined,
    selectedYearlyRange: undefined,
    chartYear: 2026,
  };

  beforeEach(() => {
    vi.mocked(getCurrentJakartaHour).mockReturnValue(14);
  });

  it('includes previous data for 24-hour range in yesterday', () => {
    const chartData = [{ date: '10', total: 5000 }];
    const prevChartData = [{ date: '10', total: 4000 }, { date: '14', total: 3000 }];
    const config = buildChartConfig({
      ...baseParams,
      chartType: 'hourly',
      chartData,
      prevChartData,
      activePeriodType: 'yesterday',
    });

    expect(config.data.datasets).toHaveLength(2);
    const prevValues = config.data.datasets[1].data;
    expect(prevValues).toHaveLength(24);
    expect(prevValues[10]).toBe(4000);
    expect(prevValues[14]).toBe(3000);
  });
});
