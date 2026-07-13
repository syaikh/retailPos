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

  it('imports extracted child components', () => {
    expect(src).toContain("import PeriodSelector from './PeriodSelector.svelte'");
    expect(src).toContain("import KPICards from './KPICards.svelte'");
    expect(src).toContain("import ChartArea from './ChartArea.svelte'");
    expect(src).toContain("import BestWorstBadges from './BestWorstBadges.svelte'");
    expect(src).toContain("import RevenueDataTable from './RevenueDataTable.svelte'");
  });

  it('uses child components in template', () => {
    expect(src).toContain('<PeriodSelector');
    expect(src).toContain('<KPICards');
    expect(src).toContain('<ChartArea');
    expect(src).toContain('<BestWorstBadges');
    expect(src).toContain('<RevenueDataTable');
  });

  it('uses $state for chart data, KPI data', () => {
    expect(src).toContain('let chartData = $state');
    expect(src).toContain('let prevChartData = $state');
    expect(src).toContain('let kpiData = $state');
  });

  it('defines KPI data structure', () => {
    expect(src).toContain('totalRevenue: 0');
    expect(src).toContain('totalOrders: 0');
    expect(src).toContain('avgOrderValue: 0');
    expect(src).toContain('percentChange: 0');
    expect(src).toContain('comparisonType');
  });

  it('has fetchSalesWithRange function', () => {
    expect(src).toContain('async function fetchSalesWithRange');
  });

  it('has fetchAvailableYears function', () => {
    expect(src).toContain('async function fetchAvailableYears');
  });

  it('has export functions', () => {
    expect(src).toContain('async function exportToExcel');
    expect(src).toContain('async function exportToPDF');
  });

  it('has toggleSort function', () => {
    expect(src).toContain('function toggleSort');
  });

  it('has chartConfig derived', () => {
    expect(src).toContain('let chartConfig = $derived');
  });

  it('has best/worst period derivations', () => {
    expect(src).toContain('let bestPeriod = $derived');
    expect(src).toContain('let worstPeriod = $derived');
  });

  it('has tableRows and sortedRows derivations', () => {
    expect(src).toContain('let tableRows = $derived');
    expect(src).toContain('let sortedRows = $derived');
  });

  it('has chart tooltip UTC date pattern', () => {
    const match = src.match(/T00:00:00[^Z]/);
    if (match) {
      expect(match[0]).toContain('T00:00:00Z');
    }
  });

  it('does NOT import calendar components directly', () => {
    expect(src).not.toContain("SelectableCalendar");
    expect(src).not.toContain("MonthlyCalendar");
    expect(src).not.toContain("YearCalendar");
  });

  it('does NOT import svelte:window', () => {
    expect(src).not.toContain('<svelte:window');
  });
});
