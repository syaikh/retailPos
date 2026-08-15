import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('$shared/utils/jakartaTime', () => ({
  getCurrentJakartaHour: vi.fn(() => 14),
  getTodayInJakarta: vi.fn(() => '2026-07-27'),
  getDateNDaysAgoInJakarta: vi.fn((n: number) => {
    const d = new Date('2026-07-27T00:00:00Z');
    d.setUTCDate(d.getUTCDate() - n);
    return d.toISOString().split('T')[0];
  }),
}));

vi.mock('$shared/i18n', () => ({
  labels: {
    day: 'Day',
    noData: 'No Data',
    currentPeriod: 'Current Period',
    prevPeriod: 'Prev Period (Rp)'
  }
}));

import { buildChartConfig } from '../chart-config';
import { getCurrentJakartaHour, getTodayInJakarta, getDateNDaysAgoInJakarta } from '$shared/utils/jakartaTime';

const baseParams = {
  chartData: [],
  prevChartData: [],
  endDate: '2026-07-27',
  selectedMonthlyRange: undefined,
  selectedYearlyRange: undefined,
  chartYear: 2026,
};

function makeHourlyData(hours: number[]) {
  return hours.map(h => ({ date: String(h), total: (h + 1) * 1000 }));
}

describe('buildChartConfig – hourly', () => {
  beforeEach(() => {
    vi.mocked(getCurrentJakartaHour).mockReturnValue(14);
  });

  it('shows 24 hours for yesterday period', () => {
    const config = buildChartConfig({ ...baseParams, chartType: 'hourly', chartData: makeHourlyData([0, 23]), activePeriodType: 'yesterday' });
    expect(config.data.labels).toHaveLength(24);
    expect(config.type).toBe('line');
  });

  it('shows 24 hours for daily period', () => {
    const config = buildChartConfig({ ...baseParams, chartType: 'hourly', chartData: makeHourlyData([0, 14]), activePeriodType: 'daily' });
    expect(config.data.labels).toHaveLength(24);
  });

  it('shows 0 through last completed hour for realtime period', () => {
    vi.mocked(getCurrentJakartaHour).mockReturnValue(14);
    const config = buildChartConfig({ ...baseParams, chartType: 'hourly', chartData: makeHourlyData([0, 14]), activePeriodType: 'realtime' });
    // Hour 14 is in-progress; only hours 0..13 (completed) are shown.
    expect(config.data.labels).toHaveLength(14);
    expect(config.data.labels[13]).toBe('13:00');
  });

  it('fills zero for missing hours', () => {
    const config = buildChartConfig({ ...baseParams, chartType: 'hourly', chartData: [{ date: '0', total: 5000 }, { date: '23', total: 3000 }], activePeriodType: 'yesterday' });
    expect(config.data.datasets[0].data[0]).toBe(5000);
    expect(config.data.datasets[0].data[1]).toBe(0);
    expect(config.data.datasets[0].data[23]).toBe(3000);
  });

  it('includes previous period data', () => {
    const config = buildChartConfig({ ...baseParams, chartType: 'hourly', chartData: [{ date: '10', total: 5000 }], prevChartData: [{ date: '10', total: 4000 }], activePeriodType: 'yesterday' });
    expect(config.data.datasets).toHaveLength(2);
    expect(config.data.datasets[1].data[10]).toBe(4000);
  });
});

describe('buildChartConfig – daily', () => {
  beforeEach(() => {
    vi.mocked(getCurrentJakartaHour).mockReturnValue(14);
  });

  it('daily + monthly period builds day-by-day labels for the month', () => {
    const endDate = '2026-07-27';
    const chartData = [
      { date: '2026-07-01', total: 10000 },
      { date: '2026-07-15', total: 20000 },
      { date: '2026-07-27', total: 30000 },
    ];
    const config = buildChartConfig({ ...baseParams, chartType: 'daily', chartData, activePeriodType: 'monthly', endDate });
    expect(config.type).toBe('line');
    expect(config.data.labels.length).toBe(31); // July has 31 days
    expect(config.data.labels[0]).toBe('Day 1');
    expect(config.data.labels[30]).toBe('Day 31');
    expect(config.data.datasets[0].data.length).toBe(31);
    // Day 1 should have the 10000 value
    expect(config.data.datasets[0].data[0]).toBe(10000);
    // Day 28+ should be null (future dates > yesterday)
    expect(config.data.datasets[0].data[30]).toBeNull();
  });

  it('daily + monthly with selectedMonthlyRange', () => {
    const selectedMonthlyRange = { start: { year: 2026, month: 7, day: 1 }, end: { year: 2026, month: 7, day: 27 } };
    const chartData = [{ date: '2026-07-15', total: 15000 }];
    const config = buildChartConfig({ ...baseParams, chartType: 'daily', chartData, activePeriodType: 'monthly', selectedMonthlyRange });
    expect(config.data.labels.length).toBe(31);
  });

  it('daily + weekly period builds Mon-Sun labels', () => {
    const weekData = [
      { date: '2026-07-27', total: 50000 }, // Monday
    ];
    const prevWeekData = [
      { date: '2026-07-20', total: 40000 }, // Monday of prev week
    ];
    // July 27 2026 is a Monday
    const config = buildChartConfig({ ...baseParams, chartType: 'daily', chartData: weekData, prevChartData: prevWeekData, activePeriodType: 'weekly', endDate: '2026-07-27' });
    expect(config.data.labels.length).toBe(7);
    // The first label should have the current date and prev date
    expect(config.data.datasets[0].data[0]).toBe(50000);
    expect(config.data.datasets[1].data[0]).toBe(40000);
  });

  it('daily + 7days period', () => {
    const chartData = [
      { date: '2026-07-27', total: 30000 },
      { date: '2026-07-26', total: 20000 },
    ];
    const prevData = [
      { date: '2026-07-20', total: 15000 },
    ];
    const config = buildChartConfig({ ...baseParams, chartType: 'daily', chartData, prevChartData: prevData, activePeriodType: '7days', endDate: '2026-07-27' });
    expect(config.data.labels.length).toBeGreaterThan(0);
    // 7days period sorts data chronologically
    expect(config.data.datasets).toHaveLength(2);
  });

  it('daily + 30days period', () => {
    const chartData = [
      { date: '2026-07-27', total: 30000 },
      { date: '2026-06-28', total: 10000 },
    ];
    const config = buildChartConfig({ ...baseParams, chartType: 'daily', chartData, activePeriodType: '30days', endDate: '2026-07-27' });
    expect(config.data.labels.length).toBeGreaterThan(0);
  });

  it('daily + monthly period with prev data shows two datasets', () => {
    const endDate = '2026-07-27';
    const chartData = [{ date: '2026-07-01', total: 10000 }];
    const prevData = [{ date: '2026-06-01', total: 8000 }];
    const config = buildChartConfig({ ...baseParams, chartType: 'daily', chartData, prevChartData: prevData, activePeriodType: 'monthly', endDate });
    expect(config.data.datasets).toHaveLength(2);
    expect(config.data.datasets[1].data[0]).toBe(8000);
  });
});

describe('buildChartConfig – monthly/yearly', () => {
  it('monthly + yearly active period builds month-by-month', () => {
    const chartData = [
      { month_start: '2026-01-01', total: 50000 },
      { month_start: '2026-06-01', total: 75000 },
    ];
    const config = buildChartConfig({ ...baseParams, chartType: 'monthly', chartData, activePeriodType: 'yearly', chartYear: 2026 });
    expect(config.type).toBe('bar');
    // July 27 -> months 1-6 completed
    expect(config.data.labels.length).toBe(6);
    expect(config.data.labels[0]).toContain('Jan');
    expect(config.data.datasets[0].data[0]).toBe(50000);
  });

  it('monthly with non-yearly active period', () => {
    const chartData = [
      { month_start: '2026-06-01', total: 60000 },
      { month_start: '2026-07-01', total: 70000 },
    ];
    const config = buildChartConfig({ ...baseParams, chartType: 'monthly', chartData, activePeriodType: '30days' });
    expect(config.type).toBe('bar');
    expect(config.data.labels.length).toBe(2);
  });

  it('yearly chart uses bar type', () => {
    const chartData = [
      { month_start: '2026-01-01', total: 50000 },
    ];
    const config = buildChartConfig({ ...baseParams, chartType: 'yearly', chartData, activePeriodType: 'yearly', chartYear: 2026 });
    expect(config.type).toBe('bar');
    expect(config.data.labels.length).toBe(6);
  });
});

describe('buildChartConfig – weekly', () => {
  it('weekly chart builds week-based labels', () => {
    const chartData = [
      { week_start: '2026-07-20', week_end: '2026-07-26', total: 100000 },
      { week_start: '2026-07-13', week_end: '2026-07-19', total: 90000 },
    ];
    const prevData = [
      { week_start: '2026-07-13', week_end: '2026-07-19', total: 80000 },
    ];
    const config = buildChartConfig({ ...baseParams, chartType: 'weekly', chartData, prevChartData: prevData, activePeriodType: 'weekly' });
    expect(config.type).toBe('bar');
    expect(config.data.labels.length).toBe(2);
    expect(config.data.datasets[0].data[0]).toBe(100000);
    expect(config.data.datasets[1].data[0]).toBe(80000);
  });

  it('weekly chart falls back to label for items without week_start', () => {
    const chartData = [
      { total: 50000, label: 'Custom' },
    ];
    const config = buildChartConfig({ ...baseParams, chartType: 'weekly', chartData, activePeriodType: 'weekly' });
    expect(config.data.labels[0]).toBe('Custom');
  });
});

describe('buildChartConfig – no previous data', () => {
  it('includes prev dataset with zeros when prevChartData is empty', () => {
    const config = buildChartConfig({ ...baseParams, chartType: 'hourly', chartData: makeHourlyData([0, 1]), activePeriodType: 'yesterday', prevChartData: [] });
    expect(config.data.datasets).toHaveLength(2);
    expect(config.data.datasets[1].data.every((v: number | null) => v === 0)).toBe(true);
  });
});

describe('buildChartConfig – daily with no date', () => {
  it('handles items without date in daily chart', () => {
    const chartData = [
      { total: 1000 },
      { date: '2026-07-27', total: 2000 },
    ];
    const config = buildChartConfig({ ...baseParams, chartType: 'daily', chartData, activePeriodType: '7days' });
    expect(config.data.labels[0]).toBe('1');
    expect(config.data.datasets[0].data[0]).toBe(1000);
  });

  it('handles invalid date in daily chart', () => {
    const chartData = [
      { date: 'not-a-date', total: 1000 },
    ];
    const config = buildChartConfig({ ...baseParams, chartType: 'daily', chartData, activePeriodType: '7days' });
    expect(config.data.labels[0]).toBe('1');
    expect(config.data.datasets[0].data[0]).toBe(1000);
  });

  it('handles empty sortedPrev for dayOffset', () => {
    const config = buildChartConfig({ ...baseParams, chartType: 'daily', chartData: [{ date: '2026-07-27', total: 1000 }], activePeriodType: '7days', prevChartData: [] });
    expect(config.data.datasets).toHaveLength(1);
  });
});

describe('buildChartConfig – monthly without month_start', () => {
  it('falls back to label for items without month_start', () => {
    const chartData = [
      { total: 5000, label: 'Custom Label' },
    ];
    const config = buildChartConfig({ ...baseParams, chartType: 'monthly', chartData, activePeriodType: '7days' });
    expect(config.data.labels[0]).toBe('Custom Label');
  });
});

describe('buildChartConfig – tooltip callbacks', () => {
  it('tooltip title for monthly+daily returns Day prefix', () => {
    const config = buildChartConfig({ ...baseParams, chartType: 'daily', chartData: [{ date: '2026-07-01', total: 1000 }], activePeriodType: 'monthly', endDate: '2026-07-27' });
    const titleFn = config.options.plugins.tooltip.callbacks.title;
    const result = titleFn([{ dataIndex: 0, label: 'Day 1' }]);
    expect(result).toContain('Day ');
  });

  it('tooltip title for non-monthly daily returns item label', () => {
    const config = buildChartConfig({ ...baseParams, chartType: 'daily', chartData: [{ date: '2026-07-27', total: 1000 }], activePeriodType: '7days' });
    const titleFn = config.options.plugins.tooltip.callbacks.title;
    const result = titleFn([{ label: '27 Jul 2026', dataIndex: 0 }]);
    expect(result).toBe('27 Jul 2026');
  });

  it('tooltip title returns empty string for empty items', () => {
    const config = buildChartConfig({ ...baseParams, chartType: 'hourly', chartData: [], activePeriodType: 'yesterday' });
    const titleFn = config.options.plugins.tooltip.callbacks.title;
    expect(titleFn([])).toBe('');
  });

  it('tooltip label handles null y', () => {
    const config = buildChartConfig({ ...baseParams, chartType: 'daily', chartData: [{ date: '2026-07-01', total: 1000 }], activePeriodType: 'monthly', endDate: '2026-07-27' });
    const labelFn = config.options.plugins.tooltip.callbacks.label;
    const result = labelFn({ parsed: { y: null }, dataset: { label: 'Test' }, dataIndex: 0, datasetIndex: 0 } as any);
    expect(result).toBeNull();
  });

  it('tooltip label handles prev dataset without date', () => {
    const config = buildChartConfig({ ...baseParams, chartType: 'daily', chartData: [{ date: '2026-07-01', total: 1000 }], prevChartData: [{ date: '2026-06-01', total: 800 }], activePeriodType: 'monthly', endDate: '2026-07-27' });
    const labelFn = config.options.plugins.tooltip.callbacks.label;
    // Dataset index 1 = previous period
    const result = labelFn({ parsed: { y: 0 }, dataset: { label: 'Previous Period' }, dataIndex: 0, datasetIndex: 1 } as any);
    expect(typeof result).toBe('string');
  });

  it('tooltip label formats large numbers with M/jt/Rb suffixes', () => {
    const config = buildChartConfig({ ...baseParams, chartType: 'hourly', chartData: [{ date: '0', total: 1000 }], activePeriodType: 'yesterday' });
    const labelFn = config.options.plugins.tooltip.callbacks.label;

    let result = labelFn({ parsed: { y: 1500000000 }, dataset: { label: 'Test' }, dataIndex: 0, datasetIndex: 0 } as any);
    expect(result).toContain('M');

    result = labelFn({ parsed: { y: 2000000 }, dataset: { label: 'Test' }, dataIndex: 0, datasetIndex: 0 } as any);
    expect(result).toContain('jt');

    result = labelFn({ parsed: { y: 5000 }, dataset: { label: 'Test' }, dataIndex: 0, datasetIndex: 0 } as any);
    expect(result).toContain('Rb');

    result = labelFn({ parsed: { y: 999 }, dataset: { label: 'Test' }, dataIndex: 0, datasetIndex: 0 } as any);
    // No suffix for < 1000
    expect(result).toContain('Rp');
  });

  it('tooltip footer calculates difference between two periods', () => {
    const config = buildChartConfig({ ...baseParams, chartType: 'hourly', chartData: [{ date: '0', total: 1000 }], prevChartData: [{ date: '0', total: 500 }], activePeriodType: 'yesterday' });
    const footerFn = config.options.plugins.tooltip.callbacks.footer;
    const result = footerFn([
      { parsed: { y: 1000 } },
      { parsed: { y: 500 } },
    ] as any);
    expect(result).toContain('+');
    expect(result).toContain('100.0%');
  });

  it('tooltip footer handles null values', () => {
    const config = buildChartConfig({ ...baseParams, chartType: 'hourly', chartData: [{ date: '0', total: 1000 }], activePeriodType: 'yesterday' });
    const footerFn = config.options.plugins.tooltip.callbacks.footer;
    expect(footerFn([{ parsed: { y: null } }, { parsed: { y: 500 } }] as any)).toBe('Difference: N/A');
  });

  it('tooltip footer handles prev <= 0', () => {
    const config = buildChartConfig({ ...baseParams, chartType: 'hourly', chartData: [{ date: '0', total: 1000 }], prevChartData: [{ date: '0', total: 0 }], activePeriodType: 'yesterday' });
    const footerFn = config.options.plugins.tooltip.callbacks.footer;
    expect(footerFn([{ parsed: { y: 1000 } }, { parsed: { y: 0 } }] as any)).toBe('');
  });

  it('tooltip footer returns empty string for single item', () => {
    const config = buildChartConfig({ ...baseParams, chartType: 'hourly', chartData: [{ date: '0', total: 1000 }], activePeriodType: 'yesterday' });
    const footerFn = config.options.plugins.tooltip.callbacks.footer;
    expect(footerFn([{ parsed: { y: 1000 } }] as any)).toBe('');
  });

  it('tooltip footer formats negative and large differences', () => {
    const config = buildChartConfig({ ...baseParams, chartType: 'hourly', chartData: [{ date: '0', total: 500 }], prevChartData: [{ date: '0', total: 2000000 }], activePeriodType: 'yesterday' });
    const footerFn = config.options.plugins.tooltip.callbacks.footer;
    const result = footerFn([{ parsed: { y: 500 } }, { parsed: { y: 2000000 } }] as any);
    expect(result).toContain('-');
  });

  it('tooltip footer uses jt suffix for millions difference', () => {
    const config = buildChartConfig({ ...baseParams, chartType: 'hourly', chartData: [{ date: '0', total: 5000000 }], prevChartData: [{ date: '0', total: 1000000 }], activePeriodType: 'yesterday' });
    const footerFn = config.options.plugins.tooltip.callbacks.footer;
    const result = footerFn([{ parsed: { y: 5000000 } }, { parsed: { y: 1000000 } }] as any);
    expect(result).toContain('jt');
  });
});

describe('buildChartConfig – scale callbacks', () => {
  it('x-axis tick returns split labels', () => {
    const config = buildChartConfig({ ...baseParams, chartType: 'hourly', chartData: [{ date: '0', total: 1000 }], activePeriodType: 'yesterday' });
    const tickFn = config.options.scales.x.ticks.callback;
    const mockAxis = { getLabelForValue: (v: number) => '00:00\n12:00' };
    const result = tickFn.call(mockAxis, 0, 0, []);
    expect(Array.isArray(result)).toBe(true);
    expect(result).toHaveLength(2);
  });

  it('x-axis tick returns single label', () => {
    const config = buildChartConfig({ ...baseParams, chartType: 'hourly', chartData: [{ date: '0', total: 1000 }], activePeriodType: 'yesterday' });
    const tickFn = config.options.scales.x.ticks.callback;
    const mockAxis = { getLabelForValue: (v: number) => '00:00' };
    const result = tickFn.call(mockAxis, 0, 0, []);
    expect(result).toBe('00:00');
  });

  it('y-axis tick formats large values', () => {
    const config = buildChartConfig({ ...baseParams, chartType: 'hourly', chartData: [{ date: '0', total: 1000 }], activePeriodType: 'yesterday' });
    const tickFn = config.options.scales.y.ticks.callback;

    expect(tickFn(1500000000)).toContain('M');
    expect(tickFn(2000000)).toContain('jt');
    expect(tickFn(5000)).toContain('Rb');
    expect(tickFn(0)).toBe('Rp 0');
    expect(tickFn(500)).toBe('Rp 500');
  });

  it('suggestedMax returns 1000 when all values are zero or empty', () => {
    const config = buildChartConfig({ ...baseParams, chartType: 'hourly', chartData: [{ date: '0', total: 0 }], activePeriodType: 'yesterday' });
    const maxFn = config.options.scales.y.suggestedMax;
    const mockChart = {
      chart: {
        data: {
          datasets: [{ data: [0] }],
        },
      },
    };
    const result = maxFn(mockChart as any);
    expect(result).toBe(1000);
  });

  it('suggestedMax returns 1000 when datasets are empty', () => {
    const config = buildChartConfig({ ...baseParams, chartType: 'hourly', chartData: [], activePeriodType: 'yesterday' });
    const maxFn = config.options.scales.y.suggestedMax;
    const mockChart = {
      chart: {
        data: {
          datasets: [],
        },
      },
    };
    const result = maxFn(mockChart as any);
    expect(result).toBe(1000);
  });

  it('suggestedMax returns max * 1.1 for positive values', () => {
    const config = buildChartConfig({ ...baseParams, chartType: 'hourly', chartData: [{ date: '0', total: 1000 }], activePeriodType: 'yesterday' });
    const maxFn = config.options.scales.y.suggestedMax;
    const mockChart = {
      chart: {
        data: {
          datasets: [{ data: [1000, 500] }],
        },
      },
    };
    const result = maxFn(mockChart as any);
    expect(result).toBe(1100);
  });
});

describe('buildChartConfig – yearly chart with prev data', () => {
  it('includes previous year data with No Data labels', () => {
    const chartData = [{ month_start: '2026-01-01', total: 50000 }];
    const prevData = [{ month_start: '2025-01-01', total: 40000 }];
    const config = buildChartConfig({ ...baseParams, chartType: 'yearly', chartData, prevChartData: prevData, activePeriodType: 'yearly', chartYear: 2026 });
    expect(config.data.datasets).toHaveLength(2);
    expect(config.data.datasets[1].data[0]).toBe(40000);
  });
});

describe('buildChartConfig – monthly chart with prevData sorted', () => {
  it('maps prev sorted data to labels with prev label', () => {
    const chartData = [
      { month_start: '2026-06-01', total: 60000 },
      { month_start: '2026-07-01', total: 70000 },
    ];
    const prevData = [
      { month_start: '2026-05-01', total: 50000 },
      { month_start: '2026-06-01', total: 55000 },
    ];
    const config = buildChartConfig({ ...baseParams, chartType: 'monthly', chartData, prevChartData: prevData, activePeriodType: '7days' });
    expect(config.data.datasets[1].data[0]).toBe(50000); // May sorted first
    expect(config.data.datasets[1].data[1]).toBe(55000);
  });
});

describe('buildChartConfig – daily chart with incomplete weekly data', () => {
  it('handles dates beyond endDate in weekly period', () => {
    const chartData = [
      { date: '2026-07-27', total: 1000 },
      { date: '2026-07-28', total: 2000 }, // beyond endDate
      { date: '2026-08-01', total: 3000 }, // beyond endDate
    ];
    const config = buildChartConfig({ ...baseParams, chartType: 'daily', chartData, activePeriodType: 'weekly', endDate: '2026-07-27' });
    expect(config.type).toBe('line');
    expect(config.data.datasets[0].data.length).toBe(7);
    // Data beyond endDate should not have values (they'll be filtered by dateStr <= endDate)
  });
});
