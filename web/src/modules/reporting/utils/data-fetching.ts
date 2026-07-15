import { apiFetch } from '$shared/api/http-client';
import { toast } from '$shared/stores/toast.svelte';
import { getTodayInJakarta, getDateNDaysAgoInJakarta, getCurrentJakartaHour } from '$shared/utils/jakartaTime';

interface FetchSalesWithRangeParams {
  start: string;
  end: string;
  chartType: string;
  activePeriodType: string;
  selectedMonthlyRange?: { start: { year: number; month: number; day: number }; end: { year: number; month: number; day: number } };
  selectedYearlyRange?: { start: { year: number }; end: { year: number } };
  peakChartValue: number;
}

interface KpiData {
  totalRevenue: number;
  previousRevenue: number;
  totalOrders: number;
  previousOrders: number;
  avgOrderValue: number;
  previousAvgOrderValue: number;
  revenuePerDay: number;
  previousRevenuePerDay: number;
  peakRevenueHour: number;
  previousPeakRevenue: number;
  peakRevenueMonth: number;
  previousPeakRevenueMonth: number;
  percentChange: number;
  comparisonType: string;
  isPartial: boolean;
  periodInfo: Record<string, unknown>;
}

interface FetchSalesResult {
  chartData: Array<{ date?: string; total: number }>;
  prevChartData: Array<{ date?: string; total: number }>;
  startDate: string;
  endDate: string;
  chartEndDate: string;
  prevStart: string;
  prevEnd: string;
  exportPeriod: string;
  exportMode: string;
  exportDate: string;
  kpiData: KpiData | null;
}

/**
 * Fetches sales data for a given date range and comparison period.
 * Returns an object with the state values to apply, or throws on failure.
 */
export async function fetchSalesWithRange({
  start,
  end,
  chartType,
  activePeriodType,
  selectedMonthlyRange,
  selectedYearlyRange,
  peakChartValue,
}: FetchSalesWithRangeParams): Promise<FetchSalesResult> {
  const chartEndpoint = chartType === 'yearly'
    ? '/api/dashboard/chart/monthly'
    : '/api/dashboard/chart';

  const backendPeriodType = activePeriodType === 'realtime' || activePeriodType === 'yesterday' || activePeriodType === 'daily'
    ? 'daily'
    : activePeriodType === '7days' ? '7days'
    : activePeriodType === 'weekly' ? 'weekly'
    : activePeriodType === 'monthly' ? 'monthly'
    : activePeriodType === 'yearly' ? 'yearly'
    : 'daily';

  const comparisonMode = activePeriodType === 'realtime' ? 'realtime' :
    activePeriodType === 'daily' ? 'completed' :
    activePeriodType === 'yesterday' ? 'completed' :
    activePeriodType === 'yearly' ? 'todate' :
    activePeriodType === '30days' ? '30days' : 'todate';

  let _chartEndDate = end;
  if (activePeriodType === 'monthly' && selectedMonthlyRange) {
    const calendarEnd = selectedMonthlyRange.end;
    _chartEndDate = `${calendarEnd.year}-${String(calendarEnd.month).padStart(2, '0')}-${String(calendarEnd.day).padStart(2, '0')}`;
    const todayJakarta = getTodayInJakarta().split('-').map(Number);
    const selStart = selectedMonthlyRange.start;
    if (selStart.year === todayJakarta[0] && selStart.month === todayJakarta[1]) {
      _chartEndDate = getDateNDaysAgoInJakarta(1);
    }
  }
  if (activePeriodType === 'yearly' && selectedYearlyRange) {
    const year = selectedYearlyRange.start.year;
    const currentYear = parseInt(getTodayInJakarta().split('-')[0]);
    if (year === currentYear) {
      _chartEndDate = end;
    } else {
      _chartEndDate = `${year}-12-31`;
    }
  }
  let comparisonDate = end;
  if (activePeriodType === 'monthly' && selectedMonthlyRange) {
    const todayJakarta = getTodayInJakarta().split('-').map(Number);
    const selStart = selectedMonthlyRange.start;
    if (selStart.year === todayJakarta[0] && selStart.month === todayJakarta[1]) {
      comparisonDate = getDateNDaysAgoInJakarta(1);
    }
  }
  if (activePeriodType === 'yearly' && selectedYearlyRange) {
    const year = selectedYearlyRange.start.year;
    const currentYear = parseInt(getTodayInJakarta().split('-')[0]);
    if (year === currentYear) {
      comparisonDate = end;
    } else {
      comparisonDate = `${year}-12-31`;
    }
  }

  const shiftDays = activePeriodType === 'realtime' || activePeriodType === 'daily' || activePeriodType === 'yesterday' ? 1 :
    activePeriodType === 'weekly' || activePeriodType === '7days' ? 7 :
    activePeriodType === '30days' ? 30 : 0;

  let prevStart = '';
  let prevEnd = '';

  if (shiftDays > 0) {
    const startParts = start.split('-').map(Number);
    const endParts = _chartEndDate.split('-').map(Number);
    const startDateObj = new Date(Date.UTC(startParts[0], startParts[1] - 1, startParts[2]));
    const endDateObj = new Date(Date.UTC(endParts[0], endParts[1] - 1, endParts[2]));
    const prevStartObj = new Date(startDateObj.getTime() - shiftDays * 86400000);
    const prevEndObj = new Date(endDateObj.getTime() - shiftDays * 86400000);
    prevStart = `${prevStartObj.getUTCFullYear()}-${String(prevStartObj.getUTCMonth() + 1).padStart(2, '0')}-${String(prevStartObj.getUTCDate()).padStart(2, '0')}`;
    prevEnd = `${prevEndObj.getUTCFullYear()}-${String(prevEndObj.getUTCMonth() + 1).padStart(2, '0')}-${String(prevEndObj.getUTCDate()).padStart(2, '0')}`;
  } else if (activePeriodType === 'monthly' && selectedMonthlyRange) {
    const startM = selectedMonthlyRange.start;
    const prevMonth = startM.month === 1 ? 12 : startM.month - 1;
    const prevYear = startM.month === 1 ? startM.year - 1 : startM.year;
    const _chartEndParts = _chartEndDate.split('-').map(Number);
    const lastDayOfPrevMonth = new Date(prevYear, prevMonth, 0).getDate();
    const prevEndDay = Math.min(_chartEndParts[2], lastDayOfPrevMonth);
    const prevEndMonth = prevMonth;
    const prevEndYear = prevYear;
    prevStart = `${prevYear}-${String(prevMonth).padStart(2, '0')}-01`;
    prevEnd = `${prevEndYear}-${String(prevEndMonth).padStart(2, '0')}-${String(prevEndDay).padStart(2, '0')}`;
  } else if (activePeriodType === 'yearly' && selectedYearlyRange) {
    const year = selectedYearlyRange.start.year;
    prevStart = `${year - 1}-01-01`;
    prevEnd = `${year - 1}-12-31`;
  }

  const chartUrl = `${chartEndpoint}?startDate=${start}&endDate=${_chartEndDate}${prevStart ? `&prevStart=${prevStart}&prevEnd=${prevEnd}` : ''}`;

  const [dualRes, comparisonRes] = await Promise.all([
    apiFetch(chartUrl),
    apiFetch(`/api/dashboard/comparison?period=${backendPeriodType}&mode=${comparisonMode}&date=${comparisonDate}`)
  ]);

  let chartData = [];
  let prevChartData = [];

  if (dualRes.ok) {
    const dualData = await dualRes.json();
    const rawCurrent = dualData.data?.current || dualData.current || dualData.data || [];
    const rawPrevious = dualData.data?.previous || dualData.previous || [];

    if (activePeriodType === 'realtime') {
      const currentHour = getCurrentJakartaHour();
      chartData = rawCurrent.filter((item: { date?: string; total: number }) => {
        const hour = parseInt(item.date || '');
        return !isNaN(hour) && hour <= currentHour;
      });
      prevChartData = rawPrevious.filter((item: { date?: string; total: number }) => {
        const hour = parseInt(item.date || '');
        return !isNaN(hour) && hour <= currentHour;
      });
    } else {
      chartData = rawCurrent;
      prevChartData = rawPrevious;
    }
  }

  let kpiData = null;

  if (comparisonRes.ok) {
    const compData = await comparisonRes.json();
    const comparison = compData.data;
    const meta = compData.meta;

    let percentChange = 0;
    let comparisonType = 'zero';

    const chartTotal = chartData.reduce((sum: number, item: { date?: string; total: number }) => {
      const val = item.total || 0;
      return sum + (val > 0 ? val : 0);
    }, 0);

    const totalRevenue = (chartType === 'hourly' || (chartType === 'daily' && activePeriodType !== '7days' && activePeriodType !== '30days'))
      ? chartTotal
      : comparison.current_revenue;

    const previousRevenue = comparison.previous_revenue;

    if (previousRevenue === 0 && totalRevenue > 0) {
      comparisonType = 'new';
      percentChange = Infinity;
    } else if (previousRevenue === 0 && totalRevenue === 0) {
      comparisonType = 'zero';
      percentChange = 0;
    } else if (previousRevenue > 0) {
      comparisonType = 'normal';
      percentChange = ((totalRevenue - previousRevenue) / previousRevenue) * 100;
    }

    kpiData = {
      totalRevenue: (chartType === 'hourly' || (chartType === 'daily' && (activePeriodType === 'monthly' || activePeriodType === 'weekly' || activePeriodType === 'daily')) && chartData.length > 0)
        ? chartTotal
        : comparison.current_revenue,
      previousRevenue,
      totalOrders: comparison.current_orders,
      previousOrders: comparison.previous_orders,
      avgOrderValue: comparison.current_aov,
      previousAvgOrderValue: comparison.previous_aov,
      revenuePerDay: comparison.revenue_per_day,
      previousRevenuePerDay: comparison.previous_revenue_per_day,
      peakRevenueHour: comparison.peak_revenue_hour || peakChartValue,
      previousPeakRevenue: comparison.previous_peak_revenue,
      peakRevenueMonth: comparison.peak_revenue_month,
      previousPeakRevenueMonth: comparison.previous_peak_revenue_month,
      percentChange,
      comparisonType,
      isPartial: meta.is_partial,
      periodInfo: meta
    };
  }

  return {
    chartData,
    prevChartData,
    startDate: start,
    endDate: end,
    chartEndDate: _chartEndDate,
    prevStart,
    prevEnd,
    exportPeriod: backendPeriodType,
    exportMode: comparisonMode,
    exportDate: comparisonDate,
    kpiData,
  };
}
