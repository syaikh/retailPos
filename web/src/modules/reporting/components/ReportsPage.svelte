<script>
  import { onMount } from 'svelte';
  import { apiFetch } from '$shared/api/http-client';
  import { toast } from '$shared/stores/toast.svelte';
  import { getTodayInJakarta, getDateNDaysAgoInJakarta, getCurrentJakartaHour } from '$shared/utils/jakartaTime';
  import { formatDate, formatDayDate, getPeriodLabel } from '$modules/reporting/lib/reporting-utils';
  import { labels } from '$shared/i18n';
  import { buildChartConfig } from '$modules/reporting/utils/chart-config';
  import { fetchSalesWithRange as fetchSales } from '$modules/reporting/utils/data-fetching';
  import { exportToExcel as doExportToExcel, exportToPDF as doExportToPDF } from '$modules/reporting/utils/export-utils';
  import PeriodSelector from './PeriodSelector.svelte';
  import KPICards from './KPICards.svelte';
  import ChartArea from './ChartArea.svelte';
  import BestWorstBadges from './BestWorstBadges.svelte';
  import RevenueDataTable from './RevenueDataTable.svelte';

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

  let pricingBreakdown = $state([]);
  let loadingPricing = $state(false);

  let currentTimeHour = $state(`${String((getCurrentJakartaHour() - 1 + 24) % 24).padStart(2, '0')}:00`);
  let currentJakartaHour = $derived(parseInt(currentTimeHour.split(':')[0]));

  let chartType = $derived(
    ['realtime', 'yesterday', 'daily'].includes(activePeriodType) ? 'hourly' :
    ['7days', '30days', 'weekly', 'monthly'].includes(activePeriodType) ? 'daily' :
    ['yearly'].includes(activePeriodType) ? 'yearly' : 'monthly'
  );

  function parseJakartaDate(dateStr) {
    if (!dateStr) return null;
    const datePart = dateStr.split(' ')[0];
    const parts = datePart.split('-');
    if (parts.length !== 3) return null;
    const y = parseInt(parts[0]);
    const m = parseInt(parts[1]) - 1;
    const d = parseInt(parts[2]);
    if (isNaN(y) || isNaN(m) || isNaN(d)) return null;
    return new Date(Date.UTC(y, m, d));
  }

  function shiftDate(dateStr, days) {
    const dt = parseJakartaDate(dateStr);
    if (!dt) return '';
    dt.setUTCDate(dt.getUTCDate() - days);
    return dt.toISOString().split('T')[0];
  }

  let statCardLabels = $derived.by(() => {
    const metaStart = kpiData.periodInfo?.current_start;
    const metaEnd = kpiData.periodInfo?.current_end;
    const shiftDays = activePeriodType === 'realtime' ? 1
      : ['yesterday', 'daily', 'weekly', '7days'].includes(activePeriodType) ? 7
      : activePeriodType === '30days' ? 30 : 0;
    const prevStartStr = metaStart && shiftDays > 0 ? shiftDate(metaStart, shiftDays) : undefined;

    const getWeeklyDayRangeLabel = () => {
      if (!metaStart || !metaEnd) return labels.vsSameWeekLastYear;
      const dayNames = [labels.daySunShort, labels.dayMonShort, labels.dayTueShort, labels.dayWedShort, labels.dayThuShort, labels.dayFriShort, labels.daySatShort];
      const currentStartDate = parseJakartaDate(metaStart);
      const currentEndDate = parseJakartaDate(metaEnd);
      if (!currentStartDate || !currentEndDate) return labels.vsSameWeekLastYear;
      const startDayName = dayNames[currentStartDate.getUTCDay()];
      const endDayName = dayNames[currentEndDate.getUTCDay()];
      return labels.vsSameDaysLastWeek.replace('{start}', startDayName).replace('{end}', endDayName);
    };
    const getMonthlyDateRangeLabel = () => {
      if (!metaStart || !metaEnd) return labels.vsPrevMonth;
      const startDate = parseJakartaDate(metaStart);
      const endDate = parseJakartaDate(metaEnd);
      if (!startDate || !endDate) return labels.vsPrevMonth;
      const prevStartDate = new Date(startDate);
      prevStartDate.setUTCMonth(prevStartDate.getUTCMonth() - 1);
      const prevEndDate = new Date(endDate);
      prevEndDate.setUTCMonth(prevEndDate.getUTCMonth() - 1);
      const lastDayOfPrevMonth = new Date(prevStartDate.getUTCFullYear(), prevStartDate.getUTCMonth() + 1, 0).getUTCDate();
      const actualPrevEndDay = Math.min(prevEndDate.getUTCDate(), lastDayOfPrevMonth);
      prevEndDate.setUTCDate(actualPrevEndDay);
      const monthNames = [labels.monthJan, labels.monthFeb, labels.monthMar, labels.monthApr, labels.monthMay, labels.monthJun, labels.monthJul, labels.monthAug, labels.monthSep, labels.monthOct, labels.monthNov, labels.monthDec];
      const startDay = prevStartDate.getUTCDate();
      const endDay = prevEndDate.getUTCDate();
      if (startDay === endDay) {
        return labels.vsMonthDay.replace('{day}', String(startDay)).replace('{month}', monthNames[prevStartDate.getUTCMonth()]);
      }
      const startStr = `${startDay} ${monthNames[prevStartDate.getUTCMonth()]}`;
      const endStr = `${endDay} ${monthNames[prevEndDate.getUTCMonth()]}`;
      return labels.vsDateRange.replace('{start}', startStr).replace('{end}', endStr);
    };
    return {
      card4:
        chartType === 'hourly' ? labels.peakRevenueHour :
        activePeriodType === 'yearly' ? labels.peakRevenueMonth :
        labels.avgRevenuePerDay,
      card5:
        activePeriodType === 'realtime' ? labels.vsYesterday.replace('{date}', formatDayDate(getDateNDaysAgoInJakarta(1))) :
        activePeriodType === 'yesterday' ? (prevStartStr ? `${labels.vsLabel} ${formatDayDate(prevStartStr)}` : labels.vsSameDayLastWeek) :
        activePeriodType === 'daily' ? (prevStartStr ? `${labels.vsLabel} ${formatDayDate(prevStartStr)}` : labels.vsSameDayLastWeek) :
        activePeriodType === '7days' ? labels.vsPrev7DaysUpper :
        activePeriodType === '30days' ? labels.vsPrev30DaysUpper :
        activePeriodType === 'weekly' ? (kpiData.isPartial && metaStart ?
          getWeeklyDayRangeLabel() : labels.vsPrevWeekUpper) :
        activePeriodType === 'monthly' ? (kpiData.isPartial ? getMonthlyDateRangeLabel() : labels.vsPrevMonthUpper) :
        activePeriodType === 'yearly' ? labels.vsPrevYearUpper : labels.vsPrevPeriodUpper,
      comparisonLabel:
        activePeriodType === 'realtime' ? labels.vsYesterday.replace('{date}', formatDayDate(getDateNDaysAgoInJakarta(1))) :
        activePeriodType === 'yesterday' ? (prevStartStr ? `${labels.vsLabel} ${formatDayDate(prevStartStr)}` : labels.vsSameDayLastWeek) :
        activePeriodType === 'daily' ? (prevStartStr ? `${labels.vsLabel} ${formatDayDate(prevStartStr)}` : labels.vsSameDayLastWeek) :
        activePeriodType === '7days' ? labels.vsPrev7Days :
        activePeriodType === '30days' ? labels.vsPrev30Days :
        activePeriodType === 'weekly' ? (kpiData.isPartial && metaStart ?
          getWeeklyDayRangeLabel() : labels.vsPrevWeek) :
        activePeriodType === 'monthly' ? (kpiData.isPartial ? getMonthlyDateRangeLabel() : labels.vsPrevMonth) :
        activePeriodType === 'yearly' ? labels.vsPrevYear : labels.vsPrevPeriod
    };
  });

  let comparisonDateRange = $derived.by(() => {
    const metaStart = kpiData.periodInfo?.current_start;
    const metaEnd = kpiData.periodInfo?.current_end;
    if (!metaStart) return '';
    const shiftDays = activePeriodType === 'realtime' ? 1
      : ['yesterday', 'daily', 'weekly', '7days'].includes(activePeriodType) ? 7
      : activePeriodType === '30days' ? 30 : 0;
    const prevStartStr = shiftDays > 0 ? shiftDate(metaStart, shiftDays) : undefined;
    if (activePeriodType === 'yearly') {
      const currentYear = metaStart.split('-')[0];
      if (currentYear) {
        const prevYear = parseInt(currentYear) - 1;
        return `1 ${labels.monthJan} ${prevYear} - 31 ${labels.monthDec} ${prevYear}`;
      }
    }
    if (activePeriodType === 'realtime') {
      return `00:00 - ${String(currentJakartaHour).padStart(2, '0')}:00`;
    }
    if (activePeriodType === 'yesterday') {
      return '00:00 - 23:00';
    }
    if (activePeriodType === 'daily') {
      return '00:00 - 23:00';
    }
    if (activePeriodType === 'weekly') {
      if (metaStart && metaEnd) {
        const currStart = metaStart.split(' ')[0];
        const currEnd = metaEnd.split(' ')[0];
        if (kpiData.isPartial && prevStartStr) {
          const prevEndStr = shiftDate(metaEnd, shiftDays);
          return `${formatDate(prevStartStr)} - ${formatDate(prevEndStr)} (${formatDate(currStart)} - ${formatDate(currEnd)})`;
        }
        const prevEndStr = metaEnd ? shiftDate(metaEnd, shiftDays) : '';
        return `${formatDate(prevStartStr)} - ${formatDate(prevEndStr)}`;
      }
      return '';
    }
    if (activePeriodType === 'monthly' && metaStart && metaEnd) {
      const currStart = metaStart.split(' ')[0];
      const currEnd = metaEnd.split(' ')[0];
      if (kpiData.isPartial) {
        const prevEndStr = shiftDate(metaEnd, shiftDays);
        return `${formatDate(prevStartStr)} - ${formatDate(prevEndStr)}`;
      }
      const prevEndStr = shiftDate(metaEnd, shiftDays);
      return `${formatDate(prevStartStr)} - ${formatDate(prevEndStr)}`;
    }
    if (prevStartStr && metaEnd) {
      const prevEndStr = shiftDate(metaEnd, shiftDays);
      return `${formatDate(prevStartStr)} - ${formatDate(prevEndStr)}`;
    }
    return '';
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
    if (chartData.length === 0 && chartType !== 'hourly') return [];

    function computeDayOffset() {
      const sortedCurrent = chartData.filter(d => d.date).sort((a, b) => a.date.localeCompare(b.date));
      const sortedPrev = [...prevChartData].filter(d => d.date).sort((a, b) => a.date.localeCompare(b.date));
      if (sortedCurrent.length === 0 || sortedPrev.length === 0) return 0;
      const diffMs = new Date(sortedCurrent[0].date).getTime() - new Date(sortedPrev[0].date).getTime();
      return Math.round(diffMs / 86400000);
    }

    if (chartType === 'hourly') {
      const maxHour = activePeriodType === 'realtime' ? currentJakartaHour : 23;
      const currentByHour = new Map(chartData.map(d => [d.date, d.total]));
      const prevByHour = new Map(prevChartData.map(d => [d.date, d.total]));
      const hasPrevData = prevChartData.length > 0;

      const completeData = [];
      for (let h = 0; h <= maxHour; h++) {
        const hourStr = String(h).padStart(2, '0');
        completeData.push({
          date: hourStr,
          total: currentByHour.get(hourStr) ?? 0,
        });
      }

      return completeData.map(d => {
        const prevRev = hasPrevData ? (prevByHour.get(d.date) ?? 0) : null;
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
          period: date ? date.toLocaleString('id-ID', { month: 'short', day: 'numeric' }) : d.label || `${labels.day} ${i + 1}`,
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
          period: currentDate.toLocaleString('id-ID', { month: 'short', day: 'numeric' }),
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
          period: date ? date.toLocaleString('id-ID', { month: 'short', year: 'numeric' }) : d.label || '',
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
    chartType === 'hourly' ? labels.hour :
    chartType === 'daily' ? labels.date :
    chartType === 'yearly' ? labels.month : labels.period
  );

  let tablePeriodHeading = $derived(
    chartType === 'hourly' ? labels.hour :
    chartType === 'daily' ? labels.date :
    chartType === 'yearly' ? labels.month : labels.period
  );

  function toggleSort(col) {
    if (sortColumn === col) sortAsc = !sortAsc;
    else { sortColumn = col; sortAsc = true; }
  }

  $effect(() => {
    if (selectedPeriodType !== 'realtime') return;
    function updateTime() {
      currentTimeHour = `${String((getCurrentJakartaHour() - 1 + 24) % 24).padStart(2, '0')}:00`;
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
      fetchPricingBreakdown(startDate, endDate);
    } catch (error) {
      toast.error(labels.toastFailedLoadSalesData);
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
      startDate,
      endDate,
      selectedPeriodType,
      currentTimeHour,
      chartType,
      comparisonDateRange,
      statCardLabels,
      kpiData,
      chartData,
      bestPeriod,
      worstPeriod,
      bestWorstHeading,
      sortedRows,
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

  async function fetchPricingBreakdown(start, end) {
    try {
      loadingPricing = true;
      const params = new URLSearchParams();
      if (start) params.append('start', start);
      if (end) params.append('end', end);
      const res = await apiFetch(`/api/dashboard/pricing-breakdown?${params.toString()}`);
      if (res.ok) {
        const data = await res.json();
        pricingBreakdown = data.data || [];
      }
    } catch (e) {
      pricingBreakdown = [];
    } finally {
      loadingPricing = false;
    }
  }

  let totalPricingRevenue = $derived(
    pricingBreakdown.reduce((sum, item) => sum + (item.revenue || 0), 0)
  );

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
        {labels.revenueOverview} - {chartType === 'hourly' ? labels.hourly : chartType === 'daily' ? labels.daily : labels.period}
        {#if kpiData.isPartial && activePeriodType === 'weekly'}
          <span class="text-xs text-warning-light bg-warning/20 px-2 py-0.5 rounded-full font-normal ml-2">
            {labels.partialDataInProgress}
          </span>
        {:else if kpiData.isPartial && activePeriodType === 'monthly'}
          <span class="text-xs text-warning-light bg-warning/20 px-2 py-0.5 rounded-full font-normal ml-2">
            {labels.ongoingMonth}
          </span>
        {/if}
      </h2>
    </div>

    {#if !loading && (kpiData.previousRevenue > 0 || kpiData.previousOrders > 0 || kpiData.comparisonType !== 'zero')}
      <div class="flex flex-wrap items-center gap-x-2 gap-y-1 mb-3 px-1 text-xs text-text-muted">
        <span class="text-text-secondary font-medium">{labels.comparison}:</span>
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

    <RevenueDataTable
      {showDataTable}
      {sortedRows}
      bind:sortColumn
      bind:sortAsc
      {tablePeriodHeading}
      ontogglesort={toggleSort}
    />
  </div>

  {#if pricingBreakdown.length > 0}
    <div class="card p-5">
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-sm font-semibold text-text-primary">{labels.revenueByPricingType}</h2>
        {#if loadingPricing}
          <span class="text-xs text-text-muted">{labels.loading}</span>
        {/if}
      </div>
      <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
        {#each pricingBreakdown as item}
          <div class="rounded-xl border border-border p-3 bg-surface-default">
            <div class="flex items-center gap-2 mb-2">
              <span class="inline-flex items-center px-2 py-0.5 rounded text-[10px] font-semibold
                {item.pricing_type === 'discount' ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' :
                 item.pricing_type === 'wholesale' ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400' :
                 item.pricing_type === 'promotion' ? 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400' :
                 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400'}">
                {item.pricing_type}
              </span>
            </div>
            <p class="text-lg font-bold text-text-primary">Rp {(item.revenue || 0).toLocaleString('id-ID')}</p>
            <div class="flex items-center gap-2 mt-1">
              <span class="text-[10px] text-text-muted">{labels.ordersCount.replace('{count}', String(item.order_count || 0))}</span>
              <span class="text-[10px] text-text-muted">·</span>
              <span class="text-[10px] text-text-muted">{labels.itemsCount.replace('{count}', String(item.item_count || 0))}</span>
            </div>
            {#if totalPricingRevenue > 0}
              <div class="mt-2 h-1.5 rounded-full bg-surface overflow-hidden">
                <div
                  class="h-full rounded-full
                    {item.pricing_type === 'discount' ? 'bg-green-500' :
                     item.pricing_type === 'wholesale' ? 'bg-blue-500' :
                     item.pricing_type === 'promotion' ? 'bg-purple-500' :
                     'bg-gray-500'}"
                  style="width: {Math.min(100, ((item.revenue || 0) / totalPricingRevenue) * 100)}%"
                ></div>
              </div>
              <span class="text-[10px] text-text-muted text-right">
                {(((item.revenue || 0) / totalPricingRevenue) * 100).toFixed(1)}%
              </span>
            {/if}
          </div>
        {/each}
      </div>
    </div>
  {/if}
</div>
