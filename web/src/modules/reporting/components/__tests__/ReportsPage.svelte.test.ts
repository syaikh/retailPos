import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'ReportsPage.svelte'), 'utf-8');
}

describe('ReportsPage.svelte source-structure guards', () => {
  const src = getSource();

  it('imports apiFetch for HTTP calls', () => {
    expect(src).toContain("import { apiFetch } from '$shared/api/http-client'");
  });

  it('imports Jakarta time utilities', () => {
    expect(src).toContain("import { getTodayInJakarta, getDateNDaysAgoInJakarta, getCurrentJakartaHour, getJakartaDayOfWeek } from '$shared/utils/jakartaTime'");
  });

  it('imports calendar components', () => {
    expect(src).toContain("import { SelectableCalendar, MonthlyCalendar, YearCalendar }");
  });

  it('imports @internationalized/date for CalendarDate', () => {
    expect(src).toContain("import { CalendarDate } from '@internationalized/date'");
  });

  it('imports chart action', () => {
    expect(src).toContain("import { chart } from '$shared/actions/chart'");
  });

  it('uses $state for chart data, KPI data, period type', () => {
    expect(src).toContain('let chartData = $state');
    expect(src).toContain('let prevChartData = $state');
    expect(src).toContain('let kpiData = $state');
    expect(src).toContain('let selectedPeriodType = $state');
  });

  it('defines KPI data structure', () => {
    expect(src).toContain('totalRevenue: 0');
    expect(src).toContain('totalOrders: 0');
    expect(src).toContain('avgOrderValue: 0');
    expect(src).toContain('percentChange: 0');
    expect(src).toContain('comparisonType');
  });

  it('uses $derived for date constraints', () => {
    expect(src).toContain('let yesterdayDate = $derived');
  });

  it('has fetchSalesWithRange and fetchSales functions', () => {
    expect(src).toContain('async function fetchSalesWithRange');
    expect(src).toContain('async function fetchSales');
  });

  it('has fetchAvailableYears function', () => {
    expect(src).toContain('async function fetchAvailableYears');
  });
});
