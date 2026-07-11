<script>
  import { onMount } from 'svelte';
  import { apiFetch } from '$shared/api/http-client';
  import { toast } from '$shared/stores/toast.svelte';
  import { getTodayInJakarta, getDateNDaysAgoInJakarta, getCurrentJakartaHour } from '$shared/utils/jakartaTime';
  import { formatDate, getPeriodLabel } from '$modules/reporting/lib/reporting-utils';
  import { buildChartConfig } from '$modules/reporting/utils/chart-config';
  import { fetchSalesWithRange as fetchSales } from '$modules/reporting/utils/data-fetching';
  import { exportToExcel as doExportToExcel, exportToPDF as doExportToPDF } from '$modules/reporting/utils/export-utils';
  import PeriodSelector from './PeriodSelector.svelte';
  import KPICards from './KPICards.svelte';
  import ChartArea from './ChartArea.svelte';
  import BestWorstBadges from './BestWorstBadges.svelte';
  import DataTable from './DataTable.svelte';

  let loading = $state(true);
  let chartData = $state([]);
  let prevChartData = $state([]);
  let availableYears = $state([]);

  let kpiData = $state({
    totalRevenue: 0,
    previousRevenue: 0,
    totalOrders: 0,
    previousOrders: 0,
    avgOrderValue: 0,
    previousAvgOrderValue: 0,
    revenuePerDay: 0,
    previousRevenuePerDay: 0,
    peakRevenueHour: null,
    previousPeakRevenue: null,
    peakRevenueMonth: null,
    previousPeakRevenueMonth: null,
    percentChange: 0,
    comparisonType: 'zero',
    isPartial: false,
    periodInfo: null
  });

  let selectedPeriodType = $state('realtime');
  let activePeriodType = $state('realtime');
  let selectedYearlyRange = $state(null);
  let selectedMonthlyRange = $state(null);

  let timezoneString = $state('GMT+07');

  let showDataTable = $state(false);
  let sortColumn = $state('period');
  let sortAsc = $state(true);

  let chartCanvas = $state();

  let currentTimeHour = $state(`${String(getCurrentJakartaHour()).padStart(2, '0')}:00`);
  let currentJakartaHour = $derived(parseInt(currentTimeHour.split(':')[0]));

  let chartType = $derived(
    ['realtime', 'yesterday', 'daily'].includes(activePeriodType) ? 'hourly' :
    ['7days', '30days', 'weekly', 'monthly'].includes(activePeriodType) ? 'daily' :
    ['yearly'].includes(activePeriodType) ? 'yearly' : 'monthly'
  );

  let statCardLabels = $derived.by(() => {
    const getWeeklyDayRangeLabel = () => {
      const currentStart = kpiData.periodInfo?.current_period?.start;
      const currentEnd = kpiData.periodInfo?.current_period?.end;
      if (!currentStart || !currentEnd) return 'vs SAME WEEK LAST YEAR';
      const dayNames = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
      const currentStartDate = new Date(currentStart);
      const currentEndDate = new Date(currentEnd);
      const startDayName = dayNames[currentStartDate.getUTCDay()];
      const endDayName = dayNames[currentEndDate.getUTCDay()];
      return `vs Same Days Last Week (${startDayName}-${endDayName})`;
    };
    const getMonthlyDateRangeLabel = () => {
      const prevStart = kpiData.periodInfo?.previous_period?.start;
      const prevEnd = kpiData.periodInfo?.previous_period?.end;
      if (!prevStart || !prevEnd) return 'vs Previous Month';
      const prevStartDate = new Date(prevStart);
      const prevEndDate = new Date(prevEnd);
      const startDay = prevStartDate.getUTCDate();
      const endDay = prevEndDate.getUTCDate();
      if (startDay === endDay) {
        const monthNames = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
        return `vs ${startDay} ${monthNames[prevStartDate.getUTCMonth()]}`;
      }
      const monthNames = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
      const startStr = `${startDay} ${monthNames[prevStartDate.getUTCMonth()]}`;
      const endStr = `${endDay} ${monthNames[prevEndDate.getUTCMonth()]}`;
      return `vs ${startStr} - ${endStr}`;
    };
    return {
      card4: 
        chartType === 'hourly' ? 'Peak Revenue Hour' :
        activePeriodType === 'yearly' ? 'Peak Revenue Month' :
        activePeriodType === 'monthly' ? 'Avg. Revenue / Day' : 'Avg. Revenue / Day',
      card5:
        activePeriodType === 'realtime' ? 'vs YESTERDAY' :
        activePeriodType === 'yesterday' ? 'vs SAME DAY LAST WEEK' :
        activePeriodType === 'daily' ? 'vs SAME DAY LAST WEEK' :
        activePeriodType === '7days' ? 'vs PREVIOUS 7 DAYS' :
        activePeriodType === '30days' ? 'vs PREVIOUS 30 DAYS' :
        activePeriodType === 'weekly' ? (kpiData.isPartial && kpiData.periodInfo?.current_period ? 
          getWeeklyDayRangeLabel() : 'vs PREVIOUS WEEK') :
        activePeriodType === 'monthly' ? (kpiData.isPartial ? getMonthlyDateRangeLabel() : 'vs PREVIOUS MONTH') :
        activePeriodType === 'yearly' ? 'vs PREVIOUS YEAR' : 'vs PREVIOUS PERIOD',
      comparisonLabel:
        activePeriodType === 'realtime' ? 'vs Yesterday' :
        activePeriodType === 'yesterday' ? 'vs Same Day Last Week' :
        activePeriodType === 'daily' ? 'vs Same Day Last Week' :
        activePeriodType === '7days' ? 'vs Previous 7 Days' :
        activePeriodType === '30days' ? 'vs Previous 30 Days' :
        activePeriodType === 'weekly' ? (kpiData.isPartial && kpiData.periodInfo?.current_period ? 
          getWeeklyDayRangeLabel() : 'vs Previous Week') :
        activePeriodType === 'monthly' ? (kpiData.isPartial ? getMonthlyDateRangeLabel() : 'vs Previous Month') :
        activePeriodType === 'yearly' ? 'vs Previous Year' : 'vs Previous Period'
    };
  });

  let comparisonDateRange = $derived.by(() => {
    if (!kpiData.periodInfo?.previous_period) return '';
    const prev = kpiData.periodInfo.previous_period;
    if (activePeriodType === 'yearly' && kpiData.periodInfo?.current_period) {
      const currentYear = kpiData.periodInfo.current_period.start?.split('-')[0];
      if (currentYear) {
        const prevYear = parseInt(currentYear) - 1;
        return `1 Jan ${prevYear} - 31 Dec ${prevYear}`;
      }
    }
    if (activePeriodType === 'realtime') {
      return `00:00 - ${String(currentJakartaHour).padStart(2, '0')}:00`;
    }
    if (activePeriodType === 'yesterday') {
      if (prev.start) return formatDate(prev.start);
      return '';
    }
    if (activePeriodType === 'daily') {
      if (prev.start) return formatDate(prev.start);
      return '';
    }
    if (activePeriodType === 'weekly') {
      if (prev.start && prev.end) {
        if (kpiData.isPartial && kpiData.periodInfo?.current_period) {
          const curr = kpiData.periodInfo.current_period;
          const currStart = curr.start;
          const currEnd = curr.end;
          return `${formatDate(prev.start)} - ${formatDate(prev.end)} (${formatDate(currStart)} - ${formatDate(currEnd)})`;
        }
        return `${formatDate(prev.start)} - ${formatDate(prev.end)}`;
      }
      return '';
    }
    if (activePeriodType === 'monthly' && prev.start && prev.end) {
      if (kpiData.isPartial && kpiData.periodInfo?.current_period) {
        return `${formatDate(prev.start)} - ${formatDate(prev.end)}`;
      }
      return `${formatDate(prev.start)} - ${formatDate(prev.end)}`;
    }
    return prev.start && prev.end ? `${formatDate(prev.start)} - ${formatDate(prev.end)}` : '';
  });

  let peakChartValue = $derived.by(() => {
    if (chartData.length === 0) return null;
    return chartData.reduce((max, item) => item.total > max.total ? item : max, chartData[0]).total;
  });

  let chartTotalRevenue = $derived.by(() => {
    if (chartData.length === 0) return null;
    if (activePeriodType === 'realtime') {
      return kpiData.totalRevenue;
    }
    return chartData.reduce((sum, item) => sum + (item.total || 0), 0);
  });

  let chartYear = $derived.by(() => {
    if (activePeriodType === 'yearly' && selectedYearlyRange) {
      return selectedYearlyRange.start.year;
    }
    if (activePeriodType === 'monthly' && selectedMonthlyRange) {
      return selectedMonthlyRange.start.year;
    }
    if (endDate) {
      return parseInt(endDate.split('-')[0]);
    }
    return parseInt(getTodayInJakarta().split('-')[0]);
  });
  let daysInMonth = $derived.by(() => {
    const parts = getTodayInJakarta().split('-').map(Number);
    return new Date(parts[0], parts[1], 0).getDate();
  });

  let projectedRevenue = $derived.by(() => {
    if (activePeriodType === 'monthly' && kpiData.isPartial) {
      const currentDay = parseInt(getTodayInJakarta().split('-')[2]);
      return (kpiData.totalRevenue / currentDay) * daysInMonth;
    }
    return null;
  });
  let aovTrend = $derived.by(() => {
    if (!kpiData.previousAvgOrderValue || kpiData.previousAvgOrderValue === 0) return null;
    return kpiData.avgOrderValue > kpiData.previousAvgOrderValue ? 'up' : 'down';
  });

  let tableRows = $derived.by(() => {
    if (chartData.length === 0) return [];

    function computeDayOffset() {
      const sortedCurrent = chartData.filter(d => d.date).sort((a, b) => a.date.localeCompare(b.date));
      const sortedPrev = [...prevChartData].filter(d => d.date).sort((a, b) => a.date.localeCompare(b.date));
      if (sortedCurrent.length === 0 || sortedPrev.length === 0) return 0;
      const diffMs = new Date(sortedCurrent[0].date).getTime() - new Date(sortedPrev[0].date).getTime();
      return Math.round(diffMs / 86400000);
    }

    if (chartType === 'hourly') {
      return chartData.map(d => {
        const prev = prevChartData.find(p => p.date === d.date);
        const prevRev = prev ? prev.total : null;
        return {
          period: getPeriodLabel(d),
          dateStr: d.date,
          revenue: d.total,
          prevRevenue: prevRev,
          orderCount: null
        };
      });
    }

    if (activePeriodType === 'monthly') {
      const prevSorted = [...prevChartData].filter(d => d.date).sort((a, b) => a.date.localeCompare(b.date));
      return chartData.map((d, i) => {
        const prev = prevSorted[i];
        const prevRev = prev ? prev.total : null;
        const dateStr = d.date || '';
        const date = dateStr ? new Date(dateStr + 'T00:00:00Z') : null;
        return {
          period: date ? date.toLocaleString('en-US', { month: 'short', day: 'numeric' }) : d.label || `Day ${i + 1}`,
          dateStr,
          revenue: d.total || 0,
          prevRevenue: prevRev,
          orderCount: null
        };
      });
    }

    if (chartType === 'daily') {
      const prevByDate = {};
      prevChartData.forEach(p => { if (p.date) prevByDate[p.date] = p.total; });
      const dayOffset = computeDayOffset();
      const safeOffset = isNaN(dayOffset) ? 0 : dayOffset;
      return chartData.map(d => {
        if (!d.date) return null;
        const currentDate = new Date(d.date + 'T00:00:00Z');
        if (isNaN(currentDate.getTime())) return null;
        const expectedPrev = new Date(currentDate.getTime() - safeOffset * 86400000);
        const expectedPrevStr = expectedPrev.toISOString().split('T')[0];
        const prevTotal = prevByDate[expectedPrevStr];
        const hasPrev = prevTotal !== undefined;
        return {
          period: currentDate.toLocaleString('en-US', { month: 'short', day: 'numeric' }),
          dateStr: d.date,
          revenue: d.total || 0,
          prevRevenue: hasPrev ? prevTotal : null,
          orderCount: null
        };
      }).filter(Boolean);
    }

    const prevSorted = [...prevChartData].sort((a, b) => (a.month_start || a.date || '').localeCompare(b.month_start || b.date || ''));
    return chartData.map((d, i) => {
      const prev = prevSorted[i];
      const prevRev = prev ? prev.total : null;
      const dateStr = d.month_start || d.date || '';
      const date = dateStr ? new Date(dateStr + 'T00:00:00Z') : null;
      return {
        period: date ? date.toLocaleString('en-US', { month: 'short', year: 'numeric' }) : d.label || '',
        dateStr,
        revenue: d.total || 0,
        prevRevenue: prevRev,
        orderCount: d.order_count || null
      };
    });
  });

  let sortedRows = $derived.by(() => {
    const rows = [...tableRows];
    const dir = sortAsc ? 1 : -1;
    rows.sort((a, b) => {
      let cmp = 0;
      if (sortColumn === 'period') cmp = (a.dateStr || a.period).localeCompare(b.dateStr || b.period);
      else if (sortColumn === 'revenue') cmp = a.revenue - b.revenue;
      else if (sortColumn === 'prev') cmp = (a.prevRevenue ?? -1) - (b.prevRevenue ?? -1);
      else if (sortColumn === 'change') {
        const cA = a.prevRevenue > 0 ? ((a.revenue - a.prevRevenue) / a.prevRevenue) * 100 : 0;
        const cB = b.prevRevenue > 0 ? ((b.revenue - b.prevRevenue) / b.prevRevenue) * 100 : 0;
        cmp = cA - cB;
      }
      return cmp * dir;
    });
    return rows;
  });

  let bestPeriod = $derived.by(() => {
    if (chartData.length === 0) return null;
    return chartData.reduce((max, item) => {
      const val = item.total || 0;
      return val > (max.total || 0) ? item : max;
    }, chartData[0]);
  });

  let worstPeriod = $derived.by(() => {
    if (chartData.length === 0) return null;
    const nonZero = chartData.filter(item => (item.total || 0) > 0);
    if (nonZero.length <= 1) return null;
    return nonZero.reduce((min, item) => {
      const val = item.total || 0;
      return val < (min.total || 0) ? item : min;
    }, nonZero[0]);
  });

  let bestWorstHeading = $derived(
    chartType === 'hourly' ? 'Hour' :
    chartType === 'daily' ? 'Date' :
    chartType === 'yearly' ? 'Month' : 'Period'
  );

  let tablePeriodHeading = $derived(
    chartType === 'hourly' ? 'Hour' :
    chartType === 'daily' ? 'Date' :
    chartType === 'yearly' ? 'Month' : 'Period'
  );

  function toggleSort(col) {
    if (sortColumn === col) sortAsc = !sortAsc;
    else { sortColumn = col; sortAsc = true; }
  }

  $effect(() => {
    if (selectedPeriodType !== 'realtime') return;
    function updateTime() {
      currentTimeHour = `${String(getCurrentJakartaHour()).padStart(2, '0')}:00`;
    }
    updateTime();
    const interval = setInterval(updateTime, 60000);
    return () => clearInterval(interval);
  });

  let chartConfig = $derived(buildChartConfig({
    chartType,
    chartData,
    prevChartData,
    activePeriodType,
    endDate,
    selectedMonthlyRange,
    selectedYearlyRange,
    chartYear,
  }));

  let startDate = $state('');
  let endDate = $state('');
  let chartEndDate = $state('');
  let prevStart = $state('');
  let prevEnd = $state('');
  let exportPeriod = $state('');
  let exportMode = $state('');
  let exportDate = $state('');

  async function fetchSalesWithRange(start, end) {
    try {
      loading = true;
      chartData = [];
      prevChartData = [];

      const result = await fetchSales({
        start,
        end,
        chartType,
        activePeriodType,
        selectedMonthlyRange,
        selectedYearlyRange,
        peakChartValue,
      });

      chartData = result.chartData;
      prevChartData = result.prevChartData;
      startDate = result.startDate;
      endDate = result.endDate;
      chartEndDate = result.chartEndDate;
      prevStart = result.prevStart;
      prevEnd = result.prevEnd;
      exportPeriod = result.exportPeriod;
      exportMode = result.exportMode;
      exportDate = result.exportDate;
      if (result.kpiData) kpiData = result.kpiData;
    } catch (error) {
      toast.error('Failed to load sales data');
    } finally {
      loading = false;
    }
  }

  async function exportToExcel() {
    await doExportToExcel({
      exportPeriod,
      exportMode,
      exportDate,
      chartCanvas,
    });
  }

  async function exportToPDF() {
    await doExportToPDF({
      startDate,
      endDate,
      selectedPeriodType,
      currentTimeHour,
      chartType,
      comparisonDateRange,
      statCardLabels,
      kpiData,
      chartCanvas,
      chartData,
      bestPeriod,
      worstPeriod,
      bestWorstHeading,
      sortedRows,
    });
  }

  async function fetchAvailableYears() {
    try {
      const res = await apiFetch('/api/dashboard/years');
      if (res.ok) {
        const data = await res.json();
        availableYears = data.data || [];
      }
    } catch (e) {
      // Silently fail - will use defaults
    }
  }

  onMount(() => {
    fetchAvailableYears();
  });
</script>

<div class="space-y-5">
  <PeriodSelector
    bind:selectedPeriodType
    bind:activePeriodType
    bind:selectedYearlyRange
    bind:selectedMonthlyRange
    {availableYears}
    {currentTimeHour}
    {timezoneString}
    onfetchsaleswithrange={fetchSalesWithRange}
    onexportexcel={exportToExcel}
    onexportpdf={exportToPDF}
  />

  <div class="card p-5">
    <div class="flex items-center justify-between mb-4">
      <h2 class="text-sm font-semibold text-text-primary">
        Revenue Overview - {chartType === 'hourly' ? 'Hourly' : chartType === 'daily' ? 'Daily' : 'Period'}
        {#if kpiData.isPartial && activePeriodType === 'weekly'}
          <span class="text-xs text-warning-light bg-warning/20 px-2 py-0.5 rounded-full font-normal ml-2">
            Partial Data - In Progress
          </span>
        {:else if kpiData.isPartial && activePeriodType === 'monthly'}
          <span class="text-xs text-warning-light bg-warning/20 px-2 py-0.5 rounded-full font-normal ml-2">
            Ongoing Month
          </span>
        {/if}
      </h2>
    </div>

    {#if !loading && (kpiData.previousRevenue > 0 || kpiData.previousOrders > 0 || kpiData.comparisonType !== 'zero')}
      <div class="flex flex-wrap items-center gap-x-2 gap-y-1 mb-3 px-1 text-xs text-text-muted">
        <span class="text-text-secondary font-medium">Comparison:</span>
        <span>{statCardLabels.comparisonLabel}</span>
        {#if comparisonDateRange}
          <span>· {comparisonDateRange}</span>
        {/if}
      </div>
    {/if}

    <KPICards
      {loading}
      {kpiData}
      {chartType}
      {activePeriodType}
      {peakChartValue}
      {projectedRevenue}
      {aovTrend}
      {statCardLabels}
      {comparisonDateRange}
    />

    <BestWorstBadges
      {bestPeriod}
      {worstPeriod}
      {bestWorstHeading}
      {tableRows}
      bind:showDataTable
      isHourly={chartType === 'hourly'}
    />

    <ChartArea bind:chartCanvas {chartConfig} {loading} {chartData} />

    <DataTable
      {showDataTable}
      {sortedRows}
      bind:sortColumn
      bind:sortAsc
      {tablePeriodHeading}
      ontogglesort={toggleSort}
    />
  </div>
</div>
