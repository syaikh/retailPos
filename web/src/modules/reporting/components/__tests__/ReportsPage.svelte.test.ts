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

  it('imports formatDayDate for date comparison labels', () => {
    expect(src).toContain("formatDayDate");
  });

  it('has parseJakartaDate helper for timezone-safe date parsing', () => {
    expect(src).toContain('function parseJakartaDate');
  });

  it('has shiftDate helper function', () => {
    expect(src).toContain('function shiftDate');
  });

  it('uses parseJakartaDate for date parsing (no naive Z suffix)', () => {
    expect(src).toContain('parseJakartaDate(metaStart)');
    expect(src).toContain('parseJakartaDate(metaEnd)');
  });

  it('uses meta.current_start / current_end from periodInfo', () => {
    expect(src).toContain('kpiData.periodInfo?.current_start');
    expect(src).toContain('kpiData.periodInfo?.current_end');
  });

  it('shows 00:00 - 23:00 range for yesterday period', () => {
    const yesterdayBlock = src.indexOf("activePeriodType === 'yesterday'");
    const rangeBlock = src.indexOf("'00:00 - 23:00'", yesterdayBlock);
    expect(rangeBlock).toBeGreaterThan(yesterdayBlock);
  });

  it('shows 00:00 - 23:00 range for daily period', () => {
    const dailyBlock = src.indexOf("activePeriodType === 'daily'");
    const rangeBlock = src.indexOf("'00:00 - 23:00'", dailyBlock);
    expect(rangeBlock).toBeGreaterThan(dailyBlock);
  });

  it('uses formatDayDate with shiftDate for yesterday comparison label', () => {
    const yesterdayLabelMatch = src.match(/yesterday.*formatDayDate.*shiftDate|yesterday.*shiftDate.*formatDayDate/s);
    expect(yesterdayLabelMatch).toBeTruthy();
  });

  it('uses formatDayDate with shiftDate for daily comparison label', () => {
    const dailyLabelMatch = src.match(/daily.*formatDayDate.*shiftDate|daily.*shiftDate.*formatDayDate/s);
    expect(dailyLabelMatch).toBeTruthy();
  });

  it('uses shiftDate in comparisonDateRange derived', () => {
    const rangeIdx = src.indexOf('let comparisonDateRange = $derived.by');
    expect(rangeIdx).toBeGreaterThan(-1);
    const rangeBlock = src.substring(rangeIdx, rangeIdx + 1400);
    expect(rangeBlock).toContain('shiftDate(metaStart');
    expect(rangeBlock).toContain('shiftDate(metaEnd');
  });

  it('uses current_start/current_end in comparisonDateRange', () => {
    const rangeIdx = src.indexOf('let comparisonDateRange = $derived.by');
    expect(rangeIdx).toBeGreaterThan(-1);
    const rangeBlock = src.substring(rangeIdx, rangeIdx + 1400);
    expect(rangeBlock).toContain('current_start');
    expect(rangeBlock).toContain('current_end');
  });

  it('returns 1 Jan - 31 Dec format for yearly comparisonDateRange', () => {
    const rangeIdx = src.indexOf('let comparisonDateRange = $derived.by');
    const yearlyIdx = src.indexOf("'yearly'", rangeIdx);
    expect(yearlyIdx).toBeGreaterThan(rangeIdx);
    const yearBlock = src.substring(yearlyIdx, yearlyIdx + 300);
    expect(yearBlock).toContain('1 Jan');
    expect(yearBlock).toContain('31 Dec');
  });
});
