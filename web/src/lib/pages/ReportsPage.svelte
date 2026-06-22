<script>
  import { onMount } from 'svelte';
  import { fly } from 'svelte/transition';
  import { apiFetch } from '$lib/api/client';
  import { toast } from '$lib/stores/toast';
import Button from '$lib/components/ui/Button.svelte';
import { chart } from '$lib/actions/chart';
  import { getTodayInJakarta, getDateNDaysAgoInJakarta, getCurrentJakartaHour, getJakartaDayOfWeek } from '$lib/utils/jakartaTime';
  import Skeleton from '$lib/components/ui/Skeleton.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';
  import { SelectableCalendar, MonthlyCalendar, YearCalendar } from '$lib/components/calendar';
  import { CalendarDate } from '@internationalized/date';
  import {
    BarChart3,
    CalendarDays, Download, FileSpreadsheet,
    ChevronDown,
    Clock, TrendingUp, TrendingDown, CircleDollarSign, Info
  } from 'lucide-svelte';

  let loading = $state(true);
  let chartData = $state([]);
  let prevChartData = $state([]);
  let availableYears = $state([]);

  // KPI data
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

// PRD Section 5: Unified period state
let selectedPeriodType = $state('realtime'); // realtime, yesterday, 7days, 30days, daily, weekly, monthly, yearly
let activePeriodType = $state('realtime'); // The period type that has active data (starts as realtime)
let dropdownOpen = $state(false);
let hoveredOption = $state(null);
let timezoneString = $state('GMT+07');

// Derived values for max constraints (Jakarta timezone)
let yesterdayDate = $derived(
  new CalendarDate(
    parseInt(getDateNDaysAgoInJakarta(1).split('-')[0]),
    parseInt(getDateNDaysAgoInJakarta(1).split('-')[1]),
    parseInt(getDateNDaysAgoInJakarta(1).split('-')[2])
  )
);

// Daily selector state - default to yesterday
let selectedDailyDate = $state(null);
// Track if user has made a selection to distinguish initial load from user click
let dailySelectionMade = $state(false);

// Weekly selector state
let selectedWeeklyRange = $state(null);
// Track if user has made a selection
let weeklySelectionMade = $state(false);

// Monthly selector state
let selectedMonthlyRange = $state(null);
// Track if user has made a selection
let monthlySelectionMade = $state(false);

// Yearly selector state
let selectedYearlyRange = $state(null);
// Track if user has made a selection
let yearlySelectionMade = $state(false);

// Selected year for display fallback (Jakarta timezone)
let selectedYear = $state(parseInt(getTodayInJakarta().split('-')[0]));

// Export dropdown
let showExportDropdown = $state(false);

// Data table state
let showDataTable = $state(false);
let sortColumn = $state('period');
let sortAsc = $state(true);

// Chart canvas ref for toDataURL export
let chartCanvas = $state();

// Live time state for realtime updates (Jakarta timezone)
let currentTimeHour = $state(`${String(getCurrentJakartaHour()).padStart(2, '0')}:00`);
// Reactive Jakarta hour derived from the live-updating currentTimeHour
let currentJakartaHour = $derived(parseInt(currentTimeHour.split(':')[0]));

// Period options for dropdown
const periodOptions = [
  { value: 'realtime', label: 'Real-time', icon: Clock, description: 'Hourly revenue from 00:00 until now' },
  { value: 'yesterday', label: 'Yesterday', icon: CalendarDays, description: 'Hourly revenue for the full previous day' },
  { value: '7days', label: '7 Days', icon: CalendarDays, description: 'Daily revenue for the last 7 days' },
  { value: '30days', label: '30 Days', icon: CalendarDays, description: 'Daily revenue for the last 30 days' },
  { type: 'separator', label: 'Daily' },
  { value: 'daily', label: 'Daily', icon: CalendarDays, description: 'Select a specific date for hourly revenue' },
  { type: 'separator', label: 'Extended' },
  { value: 'weekly', label: 'Weekly', icon: CalendarDays, description: 'Weekly revenue - select a week' },
  { value: 'monthly', label: 'Monthly', icon: CalendarDays, description: 'Monthly revenue - select a month' },
  { value: 'yearly', label: 'Yearly', icon: CalendarDays, description: 'Yearly revenue - select a year' },
];

// Derived: Chart type based on period selection (per PRD section 5)
// Uses activePeriodType so chart doesn't change until user selects a date
let chartType = $derived(
  ['realtime', 'yesterday', 'daily'].includes(activePeriodType) ? 'hourly' :
  ['7days', '30days', 'weekly', 'monthly'].includes(activePeriodType) ? 'daily' :
  ['yearly'].includes(activePeriodType) ? 'yearly' : 'monthly'
);

// Derived: Stat card labels based on period selection
let statCardLabels = $derived.by(() => {
  // Helper to get day range label for weekly partial periods
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
  
  // Helper to get monthly date range label
  const getMonthlyDateRangeLabel = () => {
    const prevStart = kpiData.periodInfo?.previous_period?.start;
    const prevEnd = kpiData.periodInfo?.previous_period?.end;
    
    if (!prevStart || !prevEnd) return 'vs Previous Month';
    
    // For MTD comparison, show: "vs dd Mon - dd Mon" (previous month's date range)
    // e.g., "vs 1 Mei - 2 Mei" for June 1-2 MTD comparison vs May 1-2
    const prevStartDate = new Date(prevStart);
    const prevEndDate = new Date(prevEnd);
    
    // Check if this is a single date or a range
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

// Format currency with abbreviations for large values (Rp 120.5jt, Rp 2.3M)
function formatCurrencyShort(value) {
  if (value >= 1000000000) return 'Rp ' + (value / 1000000000).toFixed(1).replace(/\.0$/, '') + 'M';
  if (value >= 1000000) return 'Rp ' + (value / 1000000).toFixed(1).replace(/\.0$/, '') + 'jt';
  if (value >= 1000) return 'Rp ' + (value / 1000).toFixed(0) + 'k';
  return 'Rp ' + value.toLocaleString('id-ID');
}

// Format large numbers for display (using jt for juta/millions, M for milyar/billions)
function formatLargeNumber(value) {
  if (value >= 1000000000) return (value / 1000000000).toFixed(1).replace(/\.0$/, '') + 'M';
  if (value >= 1000000) return (value / 1000000).toFixed(1).replace(/\.0$/, '') + 'jt';
  if (value >= 1000) return (value / 1000).toFixed(0) + 'k';
  return value.toLocaleString('id-ID');
}

// Get comparison date range label for card5
let comparisonDateRange = $derived.by(() => {
  if (!kpiData.periodInfo?.previous_period) return '';
  const prev = kpiData.periodInfo.previous_period;

  // Yearly: Show full previous year range (Jan 1 - Dec 31)
  if (activePeriodType === 'yearly' && kpiData.periodInfo?.current_period) {
    const currentYear = kpiData.periodInfo.current_period.start?.split('-')[0];
    if (currentYear) {
      const prevYear = parseInt(currentYear) - 1;
      return `1 Jan ${prevYear} - 31 Dec ${prevYear}`;
    }
  }

  // Real-time: Show hour range "00:00 - HH:00" (Jakarta timezone)
  if (activePeriodType === 'realtime') {
    return `00:00 - ${String(currentJakartaHour).padStart(2, '0')}:00`;
  }

  // Yesterday: Show just the single date (same day last week)
  if (activePeriodType === 'yesterday') {
    if (prev.start) return formatDate(prev.start);
    return '';
  }

  // Daily: Show just the single date (same day last week - H-7)
  if (activePeriodType === 'daily') {
    if (prev.start) return formatDate(prev.start);
    return '';
  }

  // For weekly: show partial week range if applicable
  if (activePeriodType === 'weekly') {
    if (prev.start && prev.end) {
      // For partial week, show the same day range
      if (kpiData.isPartial && kpiData.periodInfo?.current_period) {
        const curr = kpiData.periodInfo.current_period;
        const currStart = curr.start;
        const currEnd = curr.end;
        // Show previous period range matching current period length
        return `${formatDate(prev.start)} - ${formatDate(prev.end)} (${formatDate(currStart)} - ${formatDate(currEnd)})`;
      }
      return `${formatDate(prev.start)} - ${formatDate(prev.end)}`;
    }
    return '';
  }

  // For monthly: show like-for-like date range if partial
  if (activePeriodType === 'monthly' && prev.start && prev.end) {
    if (kpiData.isPartial && kpiData.periodInfo?.current_period) {
      // For MTD comparison, show the previous period date range
      // e.g., "vs 1-2 Mei 2026" for June 1-2 MTD comparison
      return `${formatDate(prev.start)} - ${formatDate(prev.end)}`;
    }
    return `${formatDate(prev.start)} - ${formatDate(prev.end)}`;
  }
  
  // For other periods, show the range
  return prev.start && prev.end ? `${formatDate(prev.start)} - ${formatDate(prev.end)}` : '';
});

// Get peak value from chart data for card4
let peakChartValue = $derived.by(() => {
  if (chartData.length === 0) return null;
  return chartData.reduce((max, item) => item.total > max.total ? item : max, chartData[0]).total;
});

// Calculate chart total (sum of all values) for chart-based periods
let chartTotalRevenue = $derived.by(() => {
  if (chartData.length === 0) return null;
  
  // For real-time, we use the comparison data which already filters to full hours
  // The chart shows full day but we only want to show full hours for consistency
  if (activePeriodType === 'realtime') {
    return kpiData.totalRevenue; // Use the filtered total from comparison
  }
  
  return chartData.reduce((sum, item) => sum + (item.total || 0), 0);
});

// Get year from selected yearly range or selectedMonthlyRange for X-axis labeling
let chartYear = $derived.by(() => {
  if (activePeriodType === 'yearly' && selectedYearlyRange) {
    return selectedYearlyRange.start.year;
  }
  if (activePeriodType === 'monthly' && selectedMonthlyRange) {
    return selectedMonthlyRange.start.year;
  }
  // For realtime/7days/30days, get year from endDate
  if (endDate) {
    return parseInt(endDate.split('-')[0]);
  }
  return parseInt(getTodayInJakarta().split('-')[0]);
});
let daysInMonth = $derived.by(() => {
  const parts = getTodayInJakarta().split('-').map(Number);
  return new Date(parts[0], parts[1], 0).getDate();
});

// Projected revenue for partial monthly views
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

function getPeriodLabel(item) {
  if (!item) return '';
  if (item.hour !== undefined) return `${String(item.hour).padStart(2, '0')}:00`;
  if (item.date) {
    const d = new Date(item.date + 'T00:00:00Z');
    return d.toLocaleString('en-US', { month: 'short', day: 'numeric' });
  }
  if (item.month_start) {
    const d = new Date(item.month_start + 'T00:00:00Z');
    return d.toLocaleString('en-US', { month: 'short', year: 'numeric' });
  }
  return item.label || '';
}

// Table data: aligned rows from chart + prev chart
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
      const prev = prevChartData.find(p => p.hour === d.hour);
      const prevRev = prev ? prev.total : null;
      return {
        period: getPeriodLabel(d),
        dateStr: '',
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
      const prevRev = prev && prev.total > 0 ? prev.total : null;
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
    return chartData.map(d => {
      if (!d.date) return null;
      const currentDate = new Date(d.date + 'T00:00:00Z');
      const expectedPrev = new Date(currentDate.getTime() - dayOffset * 86400000);
      const expectedPrevStr = expectedPrev.toISOString().split('T')[0];
      const prevTotal = prevByDate[expectedPrevStr];
      const hasPrev = prevTotal !== undefined && prevTotal > 0;
      return {
        period: currentDate.toLocaleString('en-US', { month: 'short', day: 'numeric' }),
        dateStr: d.date,
        revenue: d.total || 0,
        prevRevenue: hasPrev ? prevTotal : null,
        orderCount: null
      };
    }).filter(Boolean);
  }

  // yearly / monthly aggregation
  const prevSorted = [...prevChartData].sort((a, b) => (a.month_start || a.date || '').localeCompare(b.month_start || b.date || ''));
  return chartData.map((d, i) => {
    const prev = prevSorted[i];
    const prevRev = prev && prev.total > 0 ? prev.total : null;
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

// Best / worst performing period
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

// Effect to update time every minute when in realtime mode
$effect(() => {
  if (selectedPeriodType !== 'realtime') return;

  function updateTime() {
    currentTimeHour = `${String(getCurrentJakartaHour()).padStart(2, '0')}:00`;
  }

  updateTime();
  const interval = setInterval(updateTime, 60000);
  return () => clearInterval(interval);
});

// Format date: dd mmm yyyy (id-ID format per PRD) - timezone aware
const formatDate = (dateString) => {
  if (!dateString) return '';
  // Parse as UTC to avoid timezone shifts
  const date = new Date(dateString + 'T00:00:00Z');
  const day = date.getUTCDate().toString().padStart(2, '0');
  const month = date.toLocaleString('id-ID', { month: 'short', timeZone: 'UTC' });
  const year = date.getUTCFullYear();
  return `${day} ${month} ${year}`;
};

// Get date range based on period type
function getPeriodDateRange(periodType) {
  const today = getTodayInJakarta();
  const daysAgo = getDateNDaysAgoInJakarta;

  switch (periodType) {
    case 'realtime':
      return { start: today, end: today };
    case 'yesterday':
      return { start: daysAgo(1), end: daysAgo(1) };
    case '7days':
      return { start: daysAgo(7), end: daysAgo(1) };
    case '30days':
      return { start: daysAgo(30), end: daysAgo(1) };
case 'daily': {
        // Only return a date if user has made a selection
        if (!dailySelectionMade || !selectedDailyDate) {
          return { start: today, end: today };
        }
        const d = selectedDailyDate.start ?? selectedDailyDate;
        const y = d.year;
        const m = String(d.month).padStart(2, '0');
        const day = String(d.day).padStart(2, '0');
        const dateStr = `${y}-${m}-${day}`;
        return { start: dateStr, end: dateStr };
      }
    case 'weekly':
      if (weeklySelectionMade && selectedWeeklyRange) {
        const start = selectedWeeklyRange.start;
        const end = selectedWeeklyRange.end;
        let endStr = `${end.year}-${String(end.month).padStart(2, '0')}-${String(end.day).padStart(2, '0')}`;
        // Constrain end to yesterday if week includes future dates
        const yesterday = getDateNDaysAgoInJakarta(1).split('-').map(Number);
        const yesterdayDate = new CalendarDate(yesterday[0], yesterday[1], yesterday[2]);
        if (end.compare(yesterdayDate) > 0 && start.compare(yesterdayDate) <= 0) {
          endStr = `${yesterday[0]}-${String(yesterday[1]).padStart(2, '0')}-${String(yesterday[2]).padStart(2, '0')}`;
        }
        return {
          start: `${start.year}-${String(start.month).padStart(2, '0')}-${String(start.day).padStart(2, '0')}`,
          end: endStr
        };
      }
      // Default: current week (Monday to yesterday)
      {
        const dayOfWeek = getJakartaDayOfWeek();
        const mondayOffset = dayOfWeek === 0 ? 6 : dayOfWeek - 1;
        const monday = getDateNDaysAgoInJakarta(mondayOffset);
        const yesterday = getDateNDaysAgoInJakarta(1);
        return { start: monday, end: yesterday };
      }
    case 'monthly':
      if (monthlySelectionMade && selectedMonthlyRange) {
        const start = selectedMonthlyRange.start;
        const end = selectedMonthlyRange.end;
        // For chart data: query full month range to get daily aggregation
        // Frontend will filter out future dates in chartData
        let endStr = `${end.year}-${String(end.month).padStart(2, '0')}-${String(end.day).padStart(2, '0')}`;
        // For current month, use yesterday for dropdown period display
        const todayJakarta = getTodayInJakarta().split('-').map(Number);
        if (start.year === todayJakarta[0] && start.month === todayJakarta[1]) {
          const yesterday = getDateNDaysAgoInJakarta(1).split('-');
          endStr = `${yesterday[0]}-${yesterday[1]}-${yesterday[2]}`;
        }
        return {
          start: `${start.year}-${String(start.month).padStart(2, '0')}-${String(start.day).padStart(2, '0')}`,
          end: endStr
        };
      }
      // Default to last completed month (May 2026 when current is June 2026)
      const todayForMonthly = getTodayInJakarta().split('-').map(Number);
      const lastMonthStart = getFirstOfMonthNAgoInJakarta(1);
      const lastMonthEnd = getDateNDaysAgoInJakarta(1);
      return { start: lastMonthStart, end: lastMonthEnd };
    case 'yearly':
      if (yearlySelectionMade && selectedYearlyRange) {
        const start = selectedYearlyRange.start;
        let endMonth = 12;
        let endDay = 31;
        // If current year selected, constrain to last month
        const todayJakarta = getTodayInJakarta().split('-').map(Number);
        const currentYear = todayJakarta[0];
        const currentMonth = todayJakarta[1]; // 1-indexed month
        if (start.year === currentYear) {
          if (currentMonth === 1) {
            // January - no previous month in current year
            return { start: `${start.year}-01-01`, end: `${start.year}-01-01` };
          }
          // End at last day of previous month
          endMonth = currentMonth - 1;
          const lastDayOfPrevMonth = new Date(currentYear, currentMonth - 1, 0).getDate();
          endDay = lastDayOfPrevMonth;
        }
        return {
          start: `${start.year}-01-01`,
          end: `${start.year}-${String(endMonth).padStart(2, '0')}-${String(endDay).padStart(2, '0')}`
        };
      }
      return { start: `${selectedYear}-01-01`, end: `${selectedYear}-12-31` };
    default:
      return { start: daysAgo(8), end: daysAgo(1) };
  }
}

// Set period and fetch data
function setPeriod(periodType) {
  // For calendar-based periods, keep dropdown open to allow date selection
  // Don't fetch - user must select a date from the calendar
  if (periodType === 'daily' || periodType === 'monthly' || periodType === 'yearly') {
    // Don't change selectedPeriodType - button text stays as-is
    // Don't change activePeriodType - chart stays as-is
    return;
  }
  
  // Weekly auto-loads the current week (Monday to yesterday)
  if (periodType === 'weekly') {
    selectedPeriodType = 'weekly';
    activePeriodType = 'weekly';
    dropdownOpen = false;
    // Reset selection to ensure default (current week) is computed
    weeklySelectionMade = false;
    const range = getPeriodDateRange('weekly');
    // Set selection for calendar display
    const [sy, sm, sd] = range.start.split('-').map(Number);
    const [ey, em, ed] = range.end.split('-').map(Number);
    selectedWeeklyRange = {
      start: new CalendarDate(sy, sm, sd),
      end: new CalendarDate(ey, em, ed)
    };
    weeklySelectionMade = true;
    fetchSalesWithRange(range.start, range.end);
    return;
  }
  
  // For non-calendar periods (realtime, yesterday, 7days, 30days)
  selectedPeriodType = periodType;
  activePeriodType = periodType;
  dropdownOpen = false;
  const range = getPeriodDateRange(periodType);
  fetchSalesWithRange(range.start, range.end);
}

// Get period description for display
function getPeriodDescription() {
  const range = getPeriodDateRange(selectedPeriodType);
  const start = formatDate(range.start);
  const end = formatDate(range.end);

  switch (selectedPeriodType) {
    case 'realtime':
      return `Real-time (00:00 - ${currentTimeHour})`;
    case 'yesterday':
      return `Yesterday · ${start}`;
    case '7days':
      return `7 Days · ${start} - ${end}`;
    case '30days':
      return `30 Days · ${start} - ${end}`;
    case 'daily':
      return `Daily · ${start}`;
    case 'weekly':
      return `Weekly · ${start} - ${end}`;
    case 'monthly':
      return `Monthly · ${start} - ${end}`;
    case 'yearly':
      return `Yearly · ${start} - ${end}`;
    default:
      return `${start} - ${end}`;
  }
}

// Chart configuration
const chartConfig = $derived.by(() => {
  let labels = [];
  let values = [];
  let prevValues = [];
  let dateStrings = [];
  let prevDateStrings = [];
  let currentChartType = chartType;

  if (chartType === 'hourly') {
    currentChartType = 'line';
    // For realtime: show all hours up to and including the last full hour
    // Example: if current time is 05:39, show hours 0-5 (last full hour is 05:00)
    labels = chartData.map(d => `${String(d.hour).padStart(2, '0')}:00`);
    values = chartData.map(d => d.total);
    // Align prev data by hour index, null for missing hours
    prevValues = chartData.map(d => {
      const prev = prevChartData.find(p => p.hour === d.hour);
      return prev ? prev.total : null;
    });
  } else if (chartType === 'daily') {
    currentChartType = 'line';
    
    // For monthly view, generate labels for all days in the month
    // Show data only up to yesterday, null for future dates
    if (activePeriodType === 'monthly') {
      // Use selectedMonthlyRange end for full month range, fallback to endDate state
      // This ensures we show the full month on X-axis (e.g., June shows 1-30)
      let periodEnd = endDate;
      if (selectedMonthlyRange) {
        const end = selectedMonthlyRange.end;
        periodEnd = `${end.year}-${String(end.month).padStart(2, '0')}-${String(end.day).padStart(2, '0')}`;
      }
      if (periodEnd) {
        const endDateParts = periodEnd.split('-');
        const year = parseInt(endDateParts[0]);
        const month = parseInt(endDateParts[1]);
        const daysInMonth = new Date(year, month, 0).getDate();
        
        // Yesterday in Jakarta timezone - max date to show actual data for
        const yesterday = getDateNDaysAgoInJakarta(1);
        
        // Create a map of actual data, only including dates up to yesterday
        const dataMap = {};
        chartData.forEach(d => {
          if (d.date && d.date <= yesterday) {
            dataMap[d.date] = d.total;
          }
        });
        
        // Build prev values ordered by date (align by index — like-for-like)
        const prevSorted = [...prevChartData]
          .filter(d => d.date)
          .sort((a, b) => a.date.localeCompare(b.date));
        const prevValuesList = prevSorted.map(d => d.total || 0);
        let prevIdx = 0;
        
        // Generate Day N labels with parallel date strings for tooltip
        labels = [];
        values = [];
        prevValues = [];
        dateStrings = [];
        prevDateStrings = [];
        for (let day = 1; day <= daysInMonth; day++) {
          const dateStr = `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
          labels.push(`Day ${day}`);
          dateStrings.push(dateStr);
          // Show data for dates up to yesterday, null for future dates
          if (dateStr <= yesterday) {
            const prevItem = prevSorted[prevIdx];
            const hasPrev = prevItem && prevItem.total > 0;
            prevDateStrings.push(prevItem ? prevItem.date : '');
            values.push(dataMap[dateStr] || 0);
            prevValues.push(hasPrev ? prevItem.total : null);
            prevIdx++;
          } else {
            prevDateStrings.push('');
            values.push(null);
            prevValues.push(null);
          }
        }
      }
    } else if (activePeriodType === 'weekly') {
      // Build prev data map by date
      const prevByDate = {};
      prevChartData.forEach(pd => {
        if (pd.date) prevByDate[pd.date] = pd.total;
      });
      // Build current data map
      const dataMap = {};
      chartData.forEach(d => {
        if (d.date) dataMap[d.date] = d.total;
      });
      // Compute day offset between current and previous week
      const sortedCurrent = chartData.filter(d => d.date && d.date <= endDate).sort((a, b) => a.date.localeCompare(b.date));
      const sortedPrev = [...prevChartData].filter(d => d.date && d.date <= endDate).sort((a, b) => a.date.localeCompare(b.date));
      let dayOffset = 0;
      if (sortedCurrent.length > 0 && sortedPrev.length > 0) {
        const diffMs = new Date(sortedCurrent[0].date).getTime() - new Date(sortedPrev[0].date).getTime();
        dayOffset = Math.round(diffMs / 86400000);
      }
      // Extend X-axis to full week (Monday-Sunday) even for partial weeks
      const endDateTime = new Date(endDate + 'T00:00:00Z');
      const endDayOfWeek = endDateTime.getUTCDay();
      const daysSinceMonday = endDayOfWeek === 0 ? 6 : endDayOfWeek - 1;
      const mondayDate = new Date(endDateTime.getTime() - daysSinceMonday * 86400000);
      const sundayDate = new Date(mondayDate);
      sundayDate.setDate(mondayDate.getDate() + 6);
      const dayMs = 86400000;
      labels = [];
      values = [];
      prevValues = [];
      dateStrings = [];
      prevDateStrings = [];
      for (let d = new Date(mondayDate); d <= sundayDate; d = new Date(d.getTime() + dayMs)) {
        const dateStr = d.toISOString().split('T')[0];
        const currentLabel = d.toLocaleString('en-US', { month: 'short', day: 'numeric' });
        if (dateStr <= endDate) {
          const total = dataMap[dateStr];
          const expectedPrev = new Date(d.getTime() - dayOffset * 86400000);
          const expectedPrevStr = expectedPrev.toISOString().split('T')[0];
          const prevTotal = prevByDate[expectedPrevStr];
          const hasPrev = prevTotal !== undefined && prevTotal > 0;
          const prevLabel = hasPrev
            ? expectedPrev.toLocaleString('en-US', { month: 'short', day: 'numeric' })
            : 'No Data';
          labels.push(`${currentLabel}\n${prevLabel}`);
          dateStrings.push(dateStr);
          prevDateStrings.push(hasPrev ? expectedPrevStr : '');
          values.push(total !== undefined ? total : 0);
          prevValues.push(hasPrev ? prevTotal : null);
        } else {
          labels.push(`${currentLabel}\nNo Data`);
          dateStrings.push(dateStr);
          prevDateStrings.push('');
          values.push(null);
          prevValues.push(null);
        }
      }
    } else {
      // For 7days/30days view, use date-offset mapping instead of index alignment
      // Build prev data map by date
      const prevByDate = {};
      prevChartData.forEach(pd => {
        if (pd.date) prevByDate[pd.date] = pd.total;
      });
      // Compute day offset between current and previous period from first items
      const sortedCurrent = chartData.filter(d => d.date).sort((a, b) => a.date.localeCompare(b.date));
      const sortedPrev = [...prevChartData].filter(d => d.date).sort((a, b) => a.date.localeCompare(b.date));
      let dayOffset = 0;
      if (sortedCurrent.length > 0 && sortedPrev.length > 0) {
        const diffMs = new Date(sortedCurrent[0].date).getTime() - new Date(sortedPrev[0].date).getTime();
        dayOffset = Math.round(diffMs / 86400000);
      }
      labels = [];
      values = [];
      prevValues = [];
      dateStrings = [];
      prevDateStrings = [];
      chartData.forEach((d, i) => {
        if (!d.date) {
          labels.push(String(i + 1));
          dateStrings.push('');
          prevDateStrings.push('');
          values.push(d.total);
          prevValues.push(null);
          return;
        }
        const currentDate = new Date(d.date);
        const currentLabel = currentDate.toLocaleString('en-US', { month: 'short', day: 'numeric' });
        // Compute expected previous date using dayOffset
        const expectedPrev = new Date(currentDate.getTime() - dayOffset * 86400000);
        const expectedPrevStr = expectedPrev.toISOString().split('T')[0];
        const prevTotal = prevByDate[expectedPrevStr];
        const hasPrev = prevTotal !== undefined && prevTotal > 0;
        const prevLabel = hasPrev
          ? expectedPrev.toLocaleString('en-US', { month: 'short', day: 'numeric' })
          : 'No Data';
        labels.push(`${currentLabel}\n${prevLabel}`);
        dateStrings.push(d.date);
        prevDateStrings.push(hasPrev ? expectedPrevStr : '');
        values.push(d.total);
        prevValues.push(hasPrev ? prevTotal : null);
      });
    }
  } else if (chartType === 'monthly' || chartType === 'yearly') {
    currentChartType = 'bar';

    if (activePeriodType === 'yearly') {
      // Month-number mapping (1-12) for like-for-like alignment regardless of data availability
      const currentByMonth = {};
      chartData.forEach(d => {
        if (d.month_start) {
          const m = parseInt(d.month_start.split('-')[1]);
          if (!isNaN(m)) currentByMonth[m] = d.total;
        }
      });
      const prevByMonth = {};
      prevChartData.forEach(d => {
        if (d.month_start) {
          const m = parseInt(d.month_start.split('-')[1]);
          if (!isNaN(m)) prevByMonth[m] = d.total;
        }
      });

      // Determine how many months to display
      const todayJakarta = getTodayInJakarta().split('-').map(Number);
      const isCurrentYear = chartYear === todayJakarta[0];
      const totalMonths = isCurrentYear ? Math.max(0, todayJakarta[1] - 1) : 12;
      const prevYear = chartYear - 1;

      labels = [];
      values = [];
      prevValues = [];
      dateStrings = [];
      prevDateStrings = [];

      for (let m = 1; m <= totalMonths; m++) {
        const currentDate = new Date(chartYear, m - 1, 1);
        const currentLabel = currentDate.toLocaleString('en-US', { month: 'short' }) + ' ' + chartYear;
        const prevDate = new Date(prevYear, m - 1, 1);
        const hasPrevData = prevByMonth[m] !== undefined && prevByMonth[m] > 0;
        const prevLabel = hasPrevData
          ? prevDate.toLocaleString('en-US', { month: 'short' }) + ' ' + prevYear
          : 'No Data';

        labels.push(`${currentLabel}\n${prevLabel}`);
        dateStrings.push(currentDate.toISOString().split('T')[0]);
        prevDateStrings.push(hasPrevData ? prevDate.toISOString().split('T')[0] : '');
        values.push(currentByMonth[m] || 0);
        prevValues.push(hasPrevData ? prevByMonth[m] : null);
      }
    } else {
      // Fallback for unexpected chartType=monthly
      const prevSorted = [...prevChartData]
        .filter(d => d.month_start)
        .sort((a, b) => a.month_start.localeCompare(b.month_start));
      labels = chartData.map((d, i) => {
        if (d.month_start) {
          const currentDate = new Date(d.month_start);
          const currentLabel = currentDate.toLocaleString('en-US', { month: 'short', year: '2-digit' });
          const prevItem = prevSorted[i];
          const prevLabel = prevItem
            ? new Date(prevItem.month_start).toLocaleString('en-US', { month: 'short' })
            : '';
          return `${currentLabel}\n${prevLabel}`;
        }
        return d.label || '';
      });
      values = chartData.map(d => d.total);
      prevValues = prevSorted.map(d => d.total || 0);
    }
  } else {
    currentChartType = 'bar';
    const prevSorted = [...prevChartData]
      .filter(d => d.week_start)
      .sort((a, b) => a.week_start.localeCompare(b.week_start));
    labels = chartData.map((d, i) => {
      if (d.week_start && d.week_end) {
        const start = new Date(d.week_start);
        const end = new Date(d.week_end);
        const startStr = start.toLocaleString('en-US', { month: 'short', day: 'numeric' });
        const endStr = end.toLocaleString('en-US', { month: 'short', day: 'numeric' });
        const currentLabel = `${startStr} - ${endStr}`;
        const prevItem = prevSorted[i];
        const prevLabel = prevItem
          ? new Date(prevItem.week_start).toLocaleString('en-US', { month: 'short', day: 'numeric' }) + ' - ' +
            new Date(prevItem.week_end).toLocaleString('en-US', { month: 'short', day: 'numeric' })
          : '';
        return `${currentLabel}\n${prevLabel}`;
      }
      return d.label || '';
    });
    values = chartData.map(d => d.total);
    prevValues = prevSorted.map(d => d.total || 0);
  }

  const hasPrevData = prevValues.some(v => v !== null);

  return {
    type: currentChartType,
    data: {
      labels,
      datasets: [
        {
          label: 'Current Period',
          data: values,
          borderColor: '#0ea5e9',
          backgroundColor: currentChartType === 'bar' ? '#0ea5e9' : 'rgba(14, 165, 233, 0.15)',
          borderWidth: currentChartType === 'bar' ? 0 : 2,
          pointBackgroundColor: '#0ea5e9',
          pointBorderColor: '#0ea5e9',
          pointBorderWidth: 0,
          pointRadius: currentChartType === 'bar' ? 0 : 4,
          pointHoverRadius: currentChartType === 'bar' ? 0 : 6,
          tension: 0
        },
        ...(hasPrevData ? [{
          label: 'Previous Period',
          data: prevValues,
          borderColor: '#94a3b8',
          backgroundColor: currentChartType === 'bar' ? 'rgba(148, 163, 184, 0.5)' : 'rgba(148, 163, 184, 0.05)',
          borderWidth: currentChartType === 'bar' ? 0 : 2,
          pointBackgroundColor: '#94a3b8',
          pointBorderColor: '#94a3b8',
          pointBorderWidth: 0,
          pointRadius: currentChartType === 'bar' ? 0 : 4,
          pointHoverRadius: currentChartType === 'bar' ? 0 : 6,
          tension: 0
        }] : [])
      ]
    },
options: {
        responsive: true,
        maintainAspectRatio: false,
        layout: {
          padding: {
            bottom: chartType !== 'hourly' ? 20 : 0
          }
        },
         plugins: {
          legend: {
            display: true,
            position: 'bottom',
            labels: { color: '#cbd5e1', font: { family: 'inherit' }, usePointStyle: true, pointStyle: 'circle' }
          },
          tooltip: {
            mode: 'index',
            intersect: false,
            callbacks: {
              title: function(items) {
                if (!items.length) return '';
                if (activePeriodType === 'monthly' && chartType === 'daily') {
                  const idx = items[0].dataIndex;
                  return `Day ${idx + 1}`;
                }
                return items[0].label;
              },
              label: function(context) {
                if (context.parsed.y === null) return null;
                let label = context.dataset.label || '';
                const showDate = (chartType === 'daily' && ['monthly', 'weekly', '7days', '30days'].includes(activePeriodType)) ||
                                 (activePeriodType === 'yearly' && chartType === 'yearly');
                if (showDate) {
                  const idx = context.dataIndex;
                  const dsIdx = context.datasetIndex;
                  const date = dsIdx === 0 ? dateStrings[idx] : prevDateStrings[idx];
                  if (date) {
                    const d = new Date(date + 'T00:00:00');
                    label += ` (${d.toLocaleDateString('en-US', { month: 'short', year: 'numeric' })})`;
                  } else if (dsIdx === 1) {
                    label += ': No Data';
                    return label;
                  }
                } else {
                  label += ': ';
                }
                const val = context.parsed.y;
                if (val >= 1000000000) label += 'Rp ' + (val / 1000000000).toFixed(1).replace(/\.0$/, '') + ' M';
                else if (val > 1000000) label += 'Rp ' + (val / 1000000).toFixed(1).replace(/\.0$/, '') + ' jt';
                else if (val > 1000) label += 'Rp ' + (val / 1000).toFixed(0) + ' Rb';
                else label += 'Rp ' + val.toLocaleString('id-ID');
                return label;
              },
              footer: function(items) {
                if (items.length < 2) return '';
                const curr = items[0].parsed.y;
                const prev = items[1].parsed.y;
                if (curr === null || prev === null) return 'Difference: N/A';
                if (prev <= 0) return '';
                const diffVal = curr - prev;
                const diffPct = (diffVal / prev) * 100;
                const prefix = diffVal >= 0 ? '+' : '';
                let diffStr = prefix;
                if (diffVal >= 1000000000) diffStr += 'Rp ' + (diffVal / 1000000000).toFixed(1).replace(/\.0$/, '') + ' M';
                else if (diffVal > 1000000) diffStr += 'Rp ' + (diffVal / 1000000).toFixed(1).replace(/\.0$/, '') + ' jt';
                else if (diffVal > 1000) diffStr += 'Rp ' + (diffVal / 1000).toFixed(0) + ' Rb';
                else diffStr += 'Rp ' + diffVal.toLocaleString('id-ID');
                diffStr += ` (${prefix}${diffPct.toFixed(1)}%)`;
                return diffStr;
              }
            }
          }
       },
scales: {
          x: {
            grid: { display: false },
            ticks: {
              color: '#9ca3af',
              font: { family: 'inherit', size: 10 },
              maxRotation: 0,
              minRotation: 0,
              autoSkip: activePeriodType === 'monthly' || activePeriodType === '30days',
              maxTicksLimit: activePeriodType === 'monthly' || activePeriodType === '30days' ? 12 : undefined,
              callback: function(val, idx, ticks) {
                const label = this.getLabelForValue(val);
                if (label && label.includes('\n')) return label.split('\n');
                return label;
              }
            }
          },
          y: {
           border: { display: false },
           grid: { color: 'rgba(255, 255, 255, 0.05)' },
           ticks: {
             color: '#cbd5e1',
             font: { family: 'inherit' },
   callback: function(value) {
                if (value >= 1000000000) return 'Rp ' + (value / 1000000000).toFixed(1).replace(/\.0$/, '') + ' M';
                if (value > 1000000) return 'Rp ' + (value / 1000000).toFixed(1).replace(/\.0$/, '') + ' jt';
                if (value > 1000) return 'Rp ' + (value / 1000).toFixed(0) + ' Rb';
                if (value === 0) return 'Rp 0';
                return 'Rp ' + value;
              }
           },
           min: 0,
            suggestedMax: function(context) {
              const allValues = context.chart.data.datasets.flatMap(ds => ds.data);
              const positiveValues = allValues.filter(v => v > 0);
              if (positiveValues.length === 0 && allValues.length > 0) return 1000;
              if (allValues.length === 0) return 1000;
              const maxValue = Math.max(...allValues);
              return maxValue + maxValue * 0.1;
            }
         }
       }
     }
  };
});

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
    startDate = start;
    endDate = end;

// Select chart endpoint based on chartType
  const chartEndpoint = chartType === 'yearly'
    ? '/api/dashboard/chart/monthly' // yearly view uses monthly aggregation for whole year
    : '/api/dashboard/chart'; // daily view (for monthly/weekly/daily periods) uses daily aggregation

  // Map frontend period types to backend period types for comparison
  const backendPeriodType = activePeriodType === 'realtime' || activePeriodType === 'yesterday' || activePeriodType === 'daily'
    ? 'daily'
    : activePeriodType === 'weekly' ? 'weekly'
    : activePeriodType === 'monthly' ? 'monthly'
    : activePeriodType === 'yearly' ? 'yearly'
    : 'daily';

// Real-time uses "realtime" mode for today-vs-yesterday comparison
  // 30days uses "30days" mode for 30-day comparison
  // daily, yesterday use "completed" mode for same day/week comparison
  // yearly always uses todate mode (year-to-date vs same period last year)
  const comparisonMode = activePeriodType === 'realtime' ? 'realtime' :
    activePeriodType === 'daily' ? 'completed' :
    activePeriodType === 'yesterday' ? 'completed' :
    activePeriodType === 'yearly' ? 'todate' :
    activePeriodType === '30days' ? '30days' : 'todate';

  // Chart endpoint uses full month range for monthly view to get daily aggregation
  // We need to use the original end date from the calendar for chart data
  let _chartEndDate = end;
  if (activePeriodType === 'monthly' && selectedMonthlyRange) {
    const calendarEnd = selectedMonthlyRange.end;
    _chartEndDate = `${calendarEnd.year}-${String(calendarEnd.month).padStart(2, '0')}-${String(calendarEnd.day).padStart(2, '0')}`;
    // Cap current month to yesterday to avoid showing future dates
    const todayJakarta = getTodayInJakarta().split('-').map(Number);
    const start = selectedMonthlyRange.start;
    if (start.year === todayJakarta[0] && start.month === todayJakarta[1]) {
      _chartEndDate = getDateNDaysAgoInJakarta(1);
    }
  }
  if (activePeriodType === 'yearly' && selectedYearlyRange) {
    const year = selectedYearlyRange.start.year;
    const currentYear = parseInt(getTodayInJakarta().split('-')[0]);
    if (year === currentYear) {
      // Current year: only show data through last completed month
      _chartEndDate = end;
    } else {
      // Past years: full year
      _chartEndDate = `${year}-12-31`;
    }
  }
  // Use yesterday as comparison date for current month (MTD comparison)
  // Use Dec 31 of selected year for yearly comparison (so backend uses correct year)
  let comparisonDate = end;
  if (activePeriodType === 'monthly' && selectedMonthlyRange) {
    const todayJakarta = getTodayInJakarta().split('-').map(Number);
    const start = selectedMonthlyRange.start;
    if (start.year === todayJakarta[0] && start.month === todayJakarta[1]) {
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
    const prevEndDay = _chartEndParts[2];
    const prevEndMonth = prevMonth;
    const prevEndYear = prevYear;
    prevStart = `${prevYear}-${String(prevMonth).padStart(2, '0')}-01`;
    prevEnd = `${prevEndYear}-${String(prevEndMonth).padStart(2, '0')}-${String(prevEndDay).padStart(2, '0')}`;
  } else if (activePeriodType === 'yearly' && selectedYearlyRange) {
    const year = selectedYearlyRange.start.year;
    prevStart = `${year - 1}-01-01`;
    prevEnd = `${year - 1}-12-31`;
  }

  // Store export params for Excel export (backend uses same calculation as comparison API)
  exportPeriod = backendPeriodType;
  exportMode = comparisonMode;
  exportDate = comparisonDate;
  // Store the final chart end date for export use
  chartEndDate = _chartEndDate;
  const chartUrl = `${chartEndpoint}?startDate=${start}&endDate=${_chartEndDate}${prevStart ? `&prevStart=${prevStart}&prevEnd=${prevEnd}` : ''}`;

  const [dualRes, comparisonRes] = await Promise.all([
      apiFetch(chartUrl),
      apiFetch(`/api/dashboard/comparison?period=${backendPeriodType}&mode=${comparisonMode}&date=${comparisonDate}`)
    ]);

    if (dualRes.ok) {
      const dualData = await dualRes.json();
      const rawCurrent = dualData.current || dualData.data || [];
      const rawPrevious = dualData.previous || [];

      // For real-time, show data from 00:00 through the current hour (includes partial hour)
      if (activePeriodType === 'realtime') {
        const currentHour = getCurrentJakartaHour();
        chartData = rawCurrent.filter(item => {
          const hour = item.hour ?? parseInt(item.label?.split(':')[0] ?? '-1');
          return hour <= currentHour;
        });
        prevChartData = rawPrevious.filter(item => {
          const hour = item.hour ?? parseInt(item.label?.split(':')[0] ?? '-1');
          return hour <= currentHour;
        });
      } else {
        chartData = rawCurrent;
        prevChartData = rawPrevious;
      }
    }

    if (comparisonRes.ok) {
      const compData = await comparisonRes.json();
      const comparison = compData.data;
      const meta = compData.meta;

      let percentChange = 0;
      let comparisonType = 'zero';

      // Calculate chart total for use in multiple places (sum of actual data points, excluding nulls)
      const chartTotal = chartData.reduce((sum, item) => {
        const val = item.total || 0;
        return sum + (val > 0 ? val : 0);
      }, 0);

      // For real-time, totalRevenue uses chartTotal for consistency with the displayed value
      // For monthly/weekly/daily views, use chart total (sum of daily points)
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
        // For hourly/monthly charts, use chart total; otherwise use comparison value
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
  } catch (error) {
    toast.error('Failed to load sales data');
  } finally {
    loading = false;
  }
}

async function fetchSales() {
  const range = getPeriodDateRange(selectedPeriodType);
  await fetchSalesWithRange(range.start, range.end);
}

  async function exportToExcel() {
    try {
      const formData = new FormData();
      if (exportPeriod) formData.set('period', exportPeriod);
      if (exportMode) formData.set('mode', exportMode);
      if (exportDate) formData.set('date', exportDate);
      if (chartCanvas) {
        const temp = document.createElement('canvas');
        temp.width = chartCanvas.width;
        temp.height = chartCanvas.height;
        const tCtx = temp.getContext('2d');
        tCtx.fillStyle = '#111827';
        tCtx.fillRect(0, 0, temp.width, temp.height);
        tCtx.drawImage(chartCanvas, 0, 0);
        formData.set('chartData', temp.toDataURL('image/png'));
      }

      const res = await apiFetch('/api/dashboard/export', {
        method: 'POST',
        body: formData,
      });
      if (!res.ok) {
        const err = await res.text();
        throw new Error(err || 'Export failed');
      }

      const blob = await res.blob();
      const fileName = `dashboard-${getTodayInJakarta()}.xlsx`;
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = fileName;
      a.click();
      URL.revokeObjectURL(url);

      toast.success('Excel export completed');
    } catch (error) {
      console.error('Excel export error:', error);
      toast.error('Export Excel gagal: ' + (error.message || 'unknown error'));
    }
  }

  async function exportToPDF() {
    try {
      const { jsPDF } = await import('jspdf');
      const { default: autoTable } = await import('jspdf-autotable');

      const doc = new jsPDF('l', 'mm', 'a4');
      const pageWidth = doc.internal.pageSize.getWidth();
      const margin = 15;
      let yPos = 20;

      // Title
      doc.setFontSize(16);
      doc.text('Revenue Report', margin, yPos);
      yPos += 8;
      doc.setFontSize(10);
      doc.text(`Period: ${getPeriodDescription()}`, margin, yPos);
      yPos += 6;
      doc.text(`Granularity: ${chartType === 'hourly' ? 'Hourly' : chartType === 'daily' ? 'Daily' : 'Periodic'}`, margin, yPos);
      yPos += 6;

      if (comparisonDateRange) {
        doc.setFontSize(9);
        doc.text(`Comparison: ${statCardLabels.comparisonLabel} · ${comparisonDateRange}`, margin, yPos);
        yPos += 6;
      }

      // Summary table (matching Excel format)
      const fmt = (v) => `Rp ${v.toLocaleString('id-ID')}`;
      const fmtChg = (cur, prev) => {
        if (prev === 0) return cur > 0 ? '+100%' : '0%';
        const chg = (cur - prev) / prev;
        return `${chg >= 0 ? '+' : ''}${(chg * 100).toFixed(1)}%`;
      };
      let summaryBody;
      if (chartType === 'hourly') {
        summaryBody = [
          ['Revenue (RP)', fmt(kpiData.totalRevenue), fmt(kpiData.previousRevenue), fmtChg(kpiData.totalRevenue, kpiData.previousRevenue)],
          ['Orders', kpiData.totalOrders.toLocaleString('id-ID'), kpiData.previousOrders.toLocaleString('id-ID'), fmtChg(kpiData.totalOrders, kpiData.previousOrders)],
          ['Avg Order Value (RP)', fmt(kpiData.avgOrderValue), fmt(kpiData.previousAvgOrderValue), fmtChg(kpiData.avgOrderValue, kpiData.previousAvgOrderValue)],
          ['Peak Revenue Hour (RP)', fmt(kpiData.peakRevenueHour), fmt(kpiData.previousPeakRevenue), fmtChg(kpiData.peakRevenueHour, kpiData.previousPeakRevenue)],
        ];
      } else if (chartType === 'yearly') {
        summaryBody = [
          ['Revenue (RP)', fmt(kpiData.totalRevenue), fmt(kpiData.previousRevenue), fmtChg(kpiData.totalRevenue, kpiData.previousRevenue)],
          ['Orders', kpiData.totalOrders.toLocaleString('id-ID'), kpiData.previousOrders.toLocaleString('id-ID'), fmtChg(kpiData.totalOrders, kpiData.previousOrders)],
          ['Avg Order Value (RP)', fmt(kpiData.avgOrderValue), fmt(kpiData.previousAvgOrderValue), fmtChg(kpiData.avgOrderValue, kpiData.previousAvgOrderValue)],
          ['Peak Revenue Month (RP)', fmt(kpiData.peakRevenueMonth), fmt(kpiData.previousPeakRevenueMonth), fmtChg(kpiData.peakRevenueMonth, kpiData.previousPeakRevenueMonth)],
          ['Avg. Revenue / Month (RP)', fmt(kpiData.revenuePerDay * 30), fmt(kpiData.previousRevenuePerDay * 30), fmtChg(kpiData.revenuePerDay * 30, kpiData.previousRevenuePerDay * 30)],
        ];
      } else {
        summaryBody = [
          ['Revenue (RP)', fmt(kpiData.totalRevenue), fmt(kpiData.previousRevenue), fmtChg(kpiData.totalRevenue, kpiData.previousRevenue)],
          ['Orders', kpiData.totalOrders.toLocaleString('id-ID'), kpiData.previousOrders.toLocaleString('id-ID'), fmtChg(kpiData.totalOrders, kpiData.previousOrders)],
          ['Avg Order Value (RP)', fmt(kpiData.avgOrderValue), fmt(kpiData.previousAvgOrderValue), fmtChg(kpiData.avgOrderValue, kpiData.previousAvgOrderValue)],
          ['Revenue per Day (RP)', fmt(kpiData.revenuePerDay), fmt(kpiData.previousRevenuePerDay), fmtChg(kpiData.revenuePerDay, kpiData.previousRevenuePerDay)],
        ];
      }

      autoTable(doc, {
        startY: yPos + 2,
        head: [['Metric', 'Current Period', 'Previous Period', 'Change']],
        body: summaryBody,
        theme: 'grid',
        styles: { fontSize: 9 },
      });
      yPos = doc.lastAutoTable.finalY + 8;

      // Chart image via canvas.toDataURL (dark background)
      if (chartCanvas && chartData.length > 0) {
        const temp = document.createElement('canvas');
        temp.width = chartCanvas.width;
        temp.height = chartCanvas.height;
        const tCtx = temp.getContext('2d');
        tCtx.fillStyle = '#111827';
        tCtx.fillRect(0, 0, temp.width, temp.height);
        tCtx.drawImage(chartCanvas, 0, 0);
        const imgData = temp.toDataURL('image/png');
        const imgWidth = pageWidth - margin * 2;
        const imgHeight = (chartCanvas.height / chartCanvas.width) * imgWidth;
        const maxImgHeight = 90;
        const finalImgHeight = Math.min(imgHeight, maxImgHeight);
        doc.addImage(imgData, 'PNG', margin, yPos, imgWidth, finalImgHeight);
        yPos += finalImgHeight + 8;
      }

      // Best / Worst
      if (bestPeriod) {
        doc.setFontSize(9);
        doc.text(`Best ${bestWorstHeading}: ${getPeriodLabel(bestPeriod)} — Rp ${(bestPeriod.total || 0).toLocaleString('id-ID')}`, margin, yPos);
        yPos += 5;
      }
      if (worstPeriod && worstPeriod.total !== bestPeriod?.total) {
        doc.setFontSize(9);
        doc.text(`Worst ${bestWorstHeading}: ${getPeriodLabel(worstPeriod)} — Rp ${(worstPeriod.total || 0).toLocaleString('id-ID')}`, margin, yPos);
        yPos += 5;
      }

      doc.addPage();
      yPos = 20;

      // Data table from sortedRows
      if (sortedRows.length > 0) {
        const hasOrders = sortedRows.some(r => r.orderCount !== null);
        const headers = ['Period', 'Revenue (Rp)', 'Prev Period (Rp)', 'Change %'];
        if (hasOrders) headers.push('Orders');

        const body = sortedRows.map(row => {
          const change = row.prevRevenue > 0 ? (((row.revenue - row.prevRevenue) / row.prevRevenue) * 100) : null;
          const rowData = [
            row.period,
            row.revenue.toLocaleString('id-ID'),
            row.prevRevenue !== null ? row.prevRevenue.toLocaleString('id-ID') : '—',
            change !== null ? `${change >= 0 ? '+' : ''}${change.toFixed(1)}%` : '—',
          ];
          if (hasOrders) rowData.push(row.orderCount !== null ? row.orderCount.toString() : '—');
          return rowData;
        });

        // Total row
        const tRev = sortedRows.reduce((s, r) => s + (r.revenue || 0), 0);
        const tPrev = sortedRows.reduce((s, r) => s + (r.prevRevenue || 0), 0);
        const tChg = tPrev > 0 ? ((tRev - tPrev) / tPrev * 100) : null;
        const totalRow = [
          'TOTAL',
          tRev.toLocaleString('id-ID'),
          tPrev > 0 ? tPrev.toLocaleString('id-ID') : '—',
          tChg !== null ? `${tChg >= 0 ? '+' : ''}${tChg.toFixed(1)}%` : '—',
        ];
        if (hasOrders) {
          totalRow.push(sortedRows.reduce((s, r) => s + (r.orderCount || 0), 0).toString());
        }
        body.push(totalRow);

        autoTable(doc, {
          startY: yPos + 2,
          head: [headers],
          body,
          theme: 'grid',
          styles: { fontSize: 7 },
          headStyles: { fillColor: [124, 58, 237] },
          footStyles: { fillColor: [30, 41, 59] },
        });
      }

      const fileName = `revenue-report-${selectedPeriodType}-${startDate || 'N/A'}-${endDate || 'N/A'}.pdf`;
      doc.save(fileName);

      toast.success('PDF export completed');
    } catch (error) {
      console.error('PDF export error:', error);
      toast.error('Failed to export to PDF');
    }
  }

  // Fetch available years from backend
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
    fetchSales();
  });
</script>

<svelte:window 
  onclick={(e) => { 
    // Check if click was inside the dropdown menu using composedPath
    const path = e.composedPath?.() || [];
    const inDropdown = path.some(el => {
      const classList = el?.classList;
      return classList && (classList.contains('card-glass') || 
                          el.closest?.('.card-glass') !== null);
    });
    if(!inDropdown) {
      if(showExportDropdown) showExportDropdown = false; 
      if(dropdownOpen) dropdownOpen = false;
    }
  }} 
  onkeydown={(e) => { 
    if(e.key === 'Escape' && showExportDropdown) showExportDropdown = false; 
    if(e.key === 'Escape' && dropdownOpen) dropdownOpen = false;
  }} 
/>

<div class="space-y-5">

  <!-- Period Selector (Unified Dropdown) -->
  <div class="card p-4 flex flex-wrap items-center gap-4">
    <div class="flex items-center gap-2 text-sm font-medium text-text-secondary">
      <BarChart3 size={16} class="text-white" />
      Period
    </div>
    
    <!-- Unified Period Dropdown Trigger -->
    <div class="relative">
      <Button
        variant="secondary"
        class="flex items-center gap-2"
        onclick={(e) => { e.stopPropagation(); dropdownOpen = !dropdownOpen; }}
        aria-haspopup="menu"
        aria-expanded={dropdownOpen}
      >
        <CalendarDays size={15} />
        {getPeriodDescription()}
        <ChevronDown
          size={14}
          class="transition-transform duration-300 {dropdownOpen ? 'rotate-180' : ''}"
        />
      </Button>

      {#if dropdownOpen}
        <div
          class="absolute left-0 top-full mt-2 card-glass p-2 z-50 min-w-[28rem] flex gap-2"
          role="menu"
          tabindex="-1"
          transition:fly={{ y: -8, duration: 200 }}
        >
          <!-- Left Column: Period Options -->
          <div class="flex flex-col gap-1 min-w-32">
            {#each periodOptions as option}
{#if option.type === 'separator'}
                <div class="px-3 py-1 text-xs font-semibold text-text-muted uppercase tracking-wide">
                  {option.label}
                </div>
              {:else}
                {@const isCalendarOption = ['daily', 'weekly', 'monthly', 'yearly'].includes(option.value)}
                <button
                  type="button"
                  class="flex items-center gap-2 px-3 py-2 text-sm rounded-lg transition-colors {selectedPeriodType === option.value ? 'bg-primary/20 text-primary-light' : 'text-text-secondary hover:bg-surface-hover'}"
                  onclick={() => { if (!isCalendarOption) setPeriod(option.value); }}
                  onmouseenter={() => hoveredOption = option}
                >
                  <option.icon size={14} />
                  {option.label}
                </button>
              {/if}
            {/each}
          </div>

<!-- Right Column: Adaptive Details & Selectors -->
            <div class="flex-1 min-w-80 border-l border-border/50 pl-2">
              <div class="text-xs text-text-secondary mb-2">Details</div>
              
              {#if hoveredOption?.value === 'realtime'}
                <div class="text-sm text-text-primary mb-2">
                  Real-time Revenue
                </div>
                <div class="text-xs text-text-muted">
                  Shows hourly revenue from 00:00 until now
                </div>
              {:else if hoveredOption?.value === 'yesterday'}
                <div class="text-sm text-text-primary mb-2">
                  Yesterday Revenue
                </div>
                <div class="text-xs text-text-muted">
                  Shows hourly revenue for the full previous day
                </div>
              {:else if hoveredOption?.value === '7days'}
                <div class="text-sm text-text-primary mb-2">
                  7 Days Revenue
                </div>
                <div class="text-xs text-text-muted">
                  Shows daily revenue for the last 7 days until yesterday
                </div>
              {:else if hoveredOption?.value === '30days'}
                <div class="text-sm text-text-primary mb-2">
                  30 Days Revenue
                </div>
                <div class="text-xs text-text-muted">
                  Shows daily revenue for the last 30 days until yesterday
                </div>
              {:else if hoveredOption?.value === 'daily'}
                <div class="text-sm text-text-primary mb-2">
                  <span class="block text-xs text-text-muted mb-2">Select Date</span>
                  <SelectableCalendar
                    mode="day"
                    bind:value={selectedDailyDate}
                    minValue={new CalendarDate(2023, 6, 16)}
                    maxValue={yesterdayDate}
                    theme={{
                      bg: 'transparent',
                      text: '#e2e8f0',
                      muted: '#64748b',
                      border: '#334155',
                      hover: '#334155',
                      selected: '#7c3aed',
                      selectedText: '#ffffff',
                      todayBorder: '#0ea5e9',
                      radius: '8px'
                    }}
onValueChange={(val) => {
                        if (val) {
                          selectedDailyDate = val;
                          const d = val.start || val;
                          const y = d.year;
                          const m = String(d.month).padStart(2, '0');
                          const day = String(d.day).padStart(2, '0');
                          const dateStr = `${y}-${m}-${day}`;
                          dailySelectionMade = true;
                          activePeriodType = 'daily';
                          selectedPeriodType = 'daily';
                          dropdownOpen = false;
                          fetchSalesWithRange(dateStr, dateStr);
                        }
                      }}
                  />
                </div>
                <div class="text-xs text-text-muted">
                  Shows hourly revenue for the selected date
                </div>
              {:else if hoveredOption?.value === 'weekly'}
                <div class="text-sm text-text-primary mb-2">
                  <span class="block text-xs text-text-muted mb-2">Select Week</span>
                  <SelectableCalendar
                    mode="week"
                    bind:value={selectedWeeklyRange}
                    minValue={new CalendarDate(2023, 6, 16)}
                    maxValue={yesterdayDate}
                    theme={{
                      bg: 'transparent',
                      text: '#e2e8f0',
                      muted: '#64748b',
                      border: '#334155',
                      hover: '#334155',
                      selected: '#7c3aed',
                      selectedText: '#ffffff',
                      radius: '8px'
                    }}
onValueChange={(val) => {
                       if (val) {
                         selectedWeeklyRange = val;
                         weeklySelectionMade = true;
                         const start = val.start;
                         const end = val.end;
                         let endStr = `${end.year}-${String(end.month).padStart(2, '0')}-${String(end.day).padStart(2, '0')}`;
                         // If week overlaps with today, constrain to yesterday
                         const yesterday = getDateNDaysAgoInJakarta(1).split('-').map(Number);
                         const yesterdayDate = new CalendarDate(yesterday[0], yesterday[1], yesterday[2]);
                         if (end.compare(yesterdayDate) > 0 && start.compare(yesterdayDate) <= 0) {
                           endStr = `${yesterday[0]}-${yesterday[1]}-${yesterday[2]}`;
                         }
                         activePeriodType = 'weekly';
                         selectedPeriodType = 'weekly';
                         dropdownOpen = false;
                         fetchSalesWithRange(
                           `${start.year}-${String(start.month).padStart(2, '0')}-${String(start.day).padStart(2, '0')}`,
                           endStr
                         );
                       }
                     }}
                  />
                </div>
                <div class="text-xs text-text-muted">
                  Shows daily revenue for the selected week
                </div>
{:else if hoveredOption?.value === 'monthly'}
                <div class="text-sm text-text-primary mb-2">
                  <span class="block text-xs text-text-muted mb-2">Select Month</span>
                  <MonthlyCalendar
                    bind:value={selectedMonthlyRange}
                    minValue={new CalendarDate(2023, 6, 1)}
                    maxValue={yesterdayDate}
                    theme={{
                      bg: 'transparent',
                      text: '#e2e8f0',
                      muted: '#64748b',
                      border: '#334155',
                      hover: '#334155',
                      selected: '#7c3aed',
                      selectedText: '#ffffff',
                      radius: '8px'
                    }}
onValueChange={(val) => {
                      if (val) {
                        selectedMonthlyRange = val;
                        monthlySelectionMade = true;
                        const start = val.start;
                        const end = val.end;
                        const startStr = `${start.year}-${String(start.month).padStart(2, '0')}-${String(start.day).padStart(2, '0')}`;
                        // Keep the full month end for chart (daily aggregation)
                        // The frontend will filter out future dates in chartData
                        let endStr = `${end.year}-${String(end.month).padStart(2, '0')}-${String(end.day).padStart(2, '0')}`;
                        // For current month, use yesterday for sales table display
                        const todayJakarta = getTodayInJakarta().split('-').map(Number);
                        const isCurrentMonth = start.year === todayJakarta[0] && start.month === todayJakarta[1];
                        if (isCurrentMonth) {
                          const yesterday = getDateNDaysAgoInJakarta(1).split('-');
                          endStr = `${yesterday[0]}-${yesterday[1]}-${yesterday[2]}`;
                        }
                        activePeriodType = 'monthly';
                        selectedPeriodType = 'monthly';
                        dropdownOpen = false;
                        fetchSalesWithRange(startStr, endStr);
                      }
                    }}
                  />
                </div>
                <div class="text-xs text-text-muted">
                  Shows daily revenue for the selected month
                </div>
              {:else if hoveredOption?.value === 'yearly'}
                <div class="text-sm text-text-primary mb-2">
                  <span class="block text-xs text-text-muted mb-2">Select Year</span>
                  <YearCalendar
                    bind:value={selectedYearlyRange}
                    minValue={availableYears.length > 0 ? new CalendarDate(Math.min(...availableYears), 1, 1) : new CalendarDate(2023, 6, 16)}
                    maxValue={yesterdayDate}
                    {availableYears}
                    theme={{
                      bg: 'transparent',
                      text: '#e2e8f0',
                      muted: '#64748b',
                      border: '#334155',
                      hover: '#334155',
                      selected: '#7c3aed',
                      selectedText: '#ffffff',
                      radius: '8px'
                    }}
onValueChange={(val) => {
                       if (val) {
                         selectedYearlyRange = val;
                         yearlySelectionMade = true;
                         const year = val.start.year;
                         // For current year, constrain to last month (April for May)
                         const todayJakarta = getTodayInJakarta();
                         const todayParts = todayJakarta.split('-').map(Number);
                         const currentYear = todayParts[0];
                         const currentMonth = todayParts[1]; // 1-indexed month
                         let endMonth = 12;
                         let endDay = 31;
                         if (year === currentYear) {
                           // Current year: show only completed months
                           if (currentMonth === 1) {
                             // January - no previous month data in current year
                             // Fetch will show "No data available"
                             activePeriodType = 'yearly';
                             selectedPeriodType = 'yearly';
                             dropdownOpen = false;
                             fetchSalesWithRange(`${year}-01-01`, `${year}-01-01`);
                             return;
                           }
                            // End at last day of previous month
                            endMonth = currentMonth - 1;
                             const lastDayOfPrevMonth = new Date(year, currentMonth - 1, 0).getDate();
                           endDay = lastDayOfPrevMonth;
                         }
                         activePeriodType = 'yearly';
                         selectedPeriodType = 'yearly';
                         dropdownOpen = false;
                         fetchSalesWithRange(`${year}-01-01`, `${year}-${String(endMonth).padStart(2, '0')}-${String(endDay).padStart(2, '0')}`);
                       }
                     }}
                  />
                </div>
                <div class="text-xs text-text-muted">
                  Shows monthly revenue for the selected year
                </div>
              {/if}
             
             <div class="text-xs text-text-muted mt-2">
               Timezone: {timezoneString}
             </div>
           </div>
        </div>
      {/if}
    </div>
    
    <!-- Export Dropdown -->
    <div class="ml-auto relative">
      <Button
        variant="primary"
        class="flex items-center gap-2 transition-all duration-300"
        onclick={(e) => { e.stopPropagation(); showExportDropdown = !showExportDropdown; }}
        aria-haspopup="menu"
        aria-expanded={showExportDropdown}
      >
        <Download size={15} />
        Export
        <ChevronDown 
          size={14} 
          class="transition-transform duration-300 {showExportDropdown ? 'rotate-180' : ''}" 
        />
      </Button>
      {#if showExportDropdown}
        <div 
          class="absolute right-0 top-full mt-2 card-glass p-1.5 z-50 min-w-44 flex flex-col gap-0.5" 
          onclick={(e) => e.stopPropagation()} 
          onkeydown={(e) => e.stopPropagation()} 
          role="menu"
          tabindex="-1"
          transition:fly={{ y: -8, duration: 200 }}
        >
          <button
            class="flex items-center gap-3 px-3 py-2 text-sm font-medium text-text-secondary hover:text-text-primary hover:bg-surface-hover rounded-xl transition-all duration-200 active:scale-[0.98] w-full text-left"
            role="menuitem"
            onclick={() => { showExportDropdown = false; exportToExcel(); }}
          >
            <FileSpreadsheet size={16} class="text-success-light" />
            Export to Excel
          </button>
          <button
            class="flex items-center gap-3 px-3 py-2 text-sm font-medium text-text-secondary hover:text-text-primary hover:bg-surface-hover rounded-xl transition-all duration-200 active:scale-[0.98] w-full text-left"
            role="menuitem"
            onclick={() => { showExportDropdown = false; exportToPDF(); }}
          >
            <Download size={16} class="text-danger-light" />
            Export to PDF
          </button>
        </div>
      {/if}
    </div>
  </div>

<!-- Chart -->
   <div class="card p-5">
     <div class="flex items-center justify-between mb-4">
       <h3 class="text-sm font-semibold text-text-primary">
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
       </h3>
     </div>

    <!-- Comparison Period Info Bar (applies to all KPI cards below) -->
    {#if !loading && (kpiData.previousRevenue > 0 || kpiData.previousOrders > 0 || kpiData.comparisonType !== 'zero')}
      <div class="flex flex-wrap items-center gap-x-2 gap-y-1 mb-3 px-1 text-xs text-text-muted">
        <span class="text-text-secondary font-medium">Comparison:</span>
        <span>{statCardLabels.comparisonLabel}</span>
        {#if comparisonDateRange}
          <span>· {comparisonDateRange}</span>
        {/if}
      </div>
    {/if}

    <!-- KPI Cards -->
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-3 mb-6">
      {#if loading}
        {#each { length: 5 } as _}
          <div class="bg-surface rounded-lg p-3 border border-border/50 min-w-0">
            <Skeleton width="w-16" height="h-2.5" class="mb-1.5" />
            <Skeleton width="w-12" height="h-5" />
          </div>
        {/each}
      {:else}
        <div class="bg-surface rounded-lg p-4 border border-border/50">
          <div class="text-xs font-medium text-text-secondary uppercase tracking-wide">Total Revenue</div>
          <div class="text-lg font-bold text-text-primary mt-1">
            {formatCurrencyShort(kpiData.totalRevenue)}
          </div>
          {#if chartType === 'hourly' && peakChartValue !== null}
            <div class="text-xs text-text-muted mt-1">
              Peak: {formatCurrencyShort(peakChartValue)}
            </div>
          {/if}
          {#if projectedRevenue !== null}
            <div class="text-xs text-success mt-1 font-medium">
              Projected: {formatCurrencyShort(projectedRevenue)}
            </div>
          {/if}
          {#if kpiData.previousRevenue > 0}
            <div class="text-xs text-text-secondary mt-1 font-medium">
              vs {formatCurrencyShort(kpiData.previousRevenue)}
            </div>
          {/if}
        </div>

        <div class="bg-surface rounded-lg p-4 border border-border/50 relative">
          <div class="text-xs font-medium text-text-secondary uppercase tracking-wide">Total Orders</div>
          <div class="text-lg font-bold text-text-primary mt-1">
            {formatLargeNumber(kpiData.totalOrders)}
          </div>
          {#if kpiData.previousOrders > 0}
            <div class="text-xs text-text-secondary mt-1 font-medium">
              vs {formatLargeNumber(kpiData.previousOrders)}
            </div>
          {/if}
        </div>

        <div class="bg-surface rounded-lg p-4 border border-border/50">
          <div class="text-xs font-medium text-text-secondary uppercase tracking-wide">Avg Order Value</div>
          <div class="flex items-center gap-1 mt-1">
            <span class="text-lg font-bold text-text-primary">
              {formatCurrencyShort(kpiData.avgOrderValue)}
            </span>
            {#if aovTrend === 'up'}
              <TrendingUp size={14} class="text-success" />
            {:else if aovTrend === 'down'}
              <TrendingDown size={14} class="text-danger-light" />
            {/if}
          </div>
          {#if kpiData.previousAvgOrderValue > 0}
            <div class="text-xs text-text-secondary mt-1 font-medium">
              vs {formatCurrencyShort(kpiData.previousAvgOrderValue)}
            </div>
          {/if}
        </div>

        <div class="bg-surface rounded-lg p-4 border border-border/50">
          <div class="text-xs font-medium text-text-secondary uppercase tracking-wide">
            {statCardLabels.card4}
          </div>
          <div class="flex items-baseline gap-1 mt-1">
            <span class="text-lg font-bold text-text-primary">
              {formatCurrencyShort(
                chartType === 'hourly' ? kpiData.peakRevenueHour :
                chartType === 'yearly' ? kpiData.peakRevenueMonth :
                kpiData.revenuePerDay
              )}
            </span>
          </div>
          {#if chartType === 'yearly' && kpiData.revenuePerDay > 0}
            <div class="text-xs text-text-muted mt-1">
              Avg. / Month: {formatCurrencyShort(kpiData.revenuePerDay * 30)}
            </div>
          {/if}
          {#if chartType === 'hourly' && kpiData.previousPeakRevenue !== null && kpiData.previousPeakRevenue > 0}
            <div class="text-xs text-text-secondary mt-1 font-medium">
              vs {formatCurrencyShort(kpiData.previousPeakRevenue)}
            </div>
          {:else if chartType === 'yearly' && kpiData.previousPeakRevenueMonth > 0}
            <div class="text-xs text-text-secondary mt-1 font-medium">
              vs {formatCurrencyShort(kpiData.previousPeakRevenueMonth)}
            </div>
          {:else if kpiData.previousRevenuePerDay > 0}
            <div class="text-xs text-text-secondary mt-1 font-medium">
              vs {formatCurrencyShort(kpiData.previousRevenuePerDay)}
            </div>
          {/if}
        </div>

        <div class="bg-surface rounded-lg p-4 border border-border/50">
          <div class="text-xs font-medium text-text-secondary uppercase tracking-wide flex items-center gap-1">
            {statCardLabels.comparisonLabel}
          </div>
          <div class="flex items-baseline gap-1 mt-1">
            {#if kpiData.percentChange !== null}
              <span class={`text-lg font-bold ${
                kpiData.comparisonType === 'new' ? 'text-success' :
                kpiData.comparisonType === 'zero' ? 'text-text-secondary' :
                kpiData.percentChange > 0 ? 'text-success' : 'text-danger'
              }`}>
                {kpiData.comparisonType === 'new' ? 'NEW' :
                 kpiData.comparisonType === 'zero' ? '±0%' :
                 kpiData.percentChange >= 0 ? '+' + kpiData.percentChange.toFixed(1) + '%' :
                 kpiData.percentChange.toFixed(1) + '%'}
              </span>
              {#if kpiData.comparisonType !== 'new' && kpiData.comparisonType !== 'zero'}
                {#if kpiData.percentChange > 0}
                  <TrendingUp size={14} class="text-success" />
                {:else}
                  <TrendingDown size={14} class="text-danger-light" />
                {/if}
              {/if}
            {/if}
          </div>
        </div>
      {/if}
    </div>

    {#if !loading && (bestPeriod || worstPeriod)}
      <div class="flex flex-wrap items-center gap-3 mb-4 px-1">
        {#if bestPeriod}
          <div class="flex items-center gap-1.5 text-xs bg-success/10 text-success-light px-2.5 py-1.5 rounded-full border border-success/20">
            <TrendingUp size={12} />
            <span class="font-medium">Best {bestWorstHeading}:</span>
            <span>{getPeriodLabel(bestPeriod)}</span>
            <span class="font-semibold">{formatCurrencyShort(bestPeriod.total || 0)}</span>
          </div>
        {/if}
        {#if worstPeriod && worstPeriod.total !== bestPeriod?.total}
          <div class="flex items-center gap-1.5 text-xs bg-danger/10 text-danger-light px-2.5 py-1.5 rounded-full border border-danger/20">
            <TrendingDown size={12} />
            <span class="font-medium">Worst {bestWorstHeading}:</span>
            <span>{getPeriodLabel(worstPeriod)}</span>
            <span class="font-semibold">{formatCurrencyShort(worstPeriod.total || 0)}</span>
          </div>
        {/if}
        {#if tableRows.length > 0}
          <button
            class="text-xs text-text-muted hover:text-text-secondary transition-colors ml-auto flex items-center gap-1"
            onclick={() => showDataTable = !showDataTable}
          >
            {showDataTable ? 'Hide' : 'Show'} Data Table
            <ChevronDown size={12} class="transition-transform duration-200 {showDataTable ? 'rotate-180' : ''}" />
          </button>
        {/if}
      </div>
    {/if}

    <div class="h-64 relative">
      {#if loading}
        <div class="absolute inset-0 flex items-center justify-center rounded-xl border border-dashed border-primary/30 bg-primary-subtle/10 shadow-glow-primary-sm overflow-hidden">
          <div class="absolute inset-0 bg-linear-to-r from-transparent via-primary-subtle/20 to-transparent animate-shimmer" style="background-size: 200% 100%;"></div>
        </div>
      {:else if chartData.length === 0}
        <div class="absolute inset-0 flex items-center justify-center text-text-muted">
          No data available for this period
        </div>
      {:else}
        <canvas bind:this={chartCanvas} use:chart={chartConfig}></canvas>
      {/if}
    </div>

    <!-- Data Table -->
    {#if !loading && showDataTable && sortedRows.length > 0}
      <div class="mt-5 overflow-x-auto" transition:fly={{ y: -8, duration: 200 }}>
        <table class="w-full text-xs text-left border-collapse">
          <thead>
            <tr class="border-b border-border/50">
              <th
                class="py-2 px-3 font-medium text-text-secondary cursor-pointer hover:text-text-primary select-none whitespace-nowrap"
                onclick={() => toggleSort('period')}
              >
                {tablePeriodHeading}
                {#if sortColumn === 'period'}
                  <span class="ml-1">{sortAsc ? '▲' : '▼'}</span>
                {/if}
              </th>
              <th
                class="py-2 px-3 font-medium text-text-secondary !text-right cursor-pointer hover:text-text-primary select-none whitespace-nowrap"
                onclick={() => toggleSort('revenue')}
              >
                Revenue (Rp)
                {#if sortColumn === 'revenue'}
                  <span class="ml-1">{sortAsc ? '▲' : '▼'}</span>
                {/if}
              </th>
              <th
                class="py-2 px-3 font-medium text-text-secondary !text-right cursor-pointer hover:text-text-primary select-none whitespace-nowrap"
                onclick={() => toggleSort('prev')}
              >
                Prev Period (Rp)
                {#if sortColumn === 'prev'}
                  <span class="ml-1">{sortAsc ? '▲' : '▼'}</span>
                {/if}
              </th>
              <th
                class="py-2 px-3 font-medium text-text-secondary !text-right cursor-pointer hover:text-text-primary select-none whitespace-nowrap"
                onclick={() => toggleSort('change')}
              >
                Change
                {#if sortColumn === 'change'}
                  <span class="ml-1">{sortAsc ? '▲' : '▼'}</span>
                {/if}
              </th>
              {#if sortedRows.some(r => r.orderCount !== null)}
                <th class="py-2 px-3 font-medium text-text-secondary text-right whitespace-nowrap">
                  Orders
                </th>
              {/if}
            </tr>
          </thead>
          <tbody>
            {#each sortedRows as row (row.dateStr || row.period)}
              {@const change = row.prevRevenue > 0 ? ((row.revenue - row.prevRevenue) / row.prevRevenue) * 100 : null}
              <tr class="border-b border-border/30 hover:bg-surface-hover/50 transition-colors">
                <td class="py-2 px-3 text-text-primary whitespace-nowrap">{row.period}</td>
                <td class="py-2 px-3 text-text-primary text-right font-medium whitespace-nowrap">{row.revenue.toLocaleString('id-ID')}</td>
                <td class="py-2 px-3 text-text-secondary text-right whitespace-nowrap">
                  {#if row.prevRevenue !== null}
                    {row.prevRevenue.toLocaleString('id-ID')}
                  {:else}
                    <span class="text-text-muted">—</span>
                  {/if}
                </td>
                <td class="py-2 px-3 text-right whitespace-nowrap">
                  {#if change !== null}
                    <span class:font-medium={true} class:text-success={change > 0} class:text-danger={change < 0} class:text-text-muted={change === 0}>
                      {change > 0 ? '+' : ''}{change.toFixed(1)}%
                    </span>
                    {#if change > 0}
                      <TrendingUp size={10} class="inline text-success ml-0.5" />
                    {:else if change < 0}
                      <TrendingDown size={10} class="inline text-danger ml-0.5" />
                    {/if}
                  {:else}
                    <span class="text-text-muted">—</span>
                  {/if}
                </td>
                {#if sortedRows.some(r => r.orderCount !== null)}
                  <td class="py-2 px-3 text-text-secondary text-right whitespace-nowrap">{row.orderCount ?? '—'}</td>
                {/if}
              </tr>
            {/each}
          </tbody>
          <tfoot>
            {#if true}
              {@const totalRevenue = sortedRows.reduce((s, r) => s + (r.revenue || 0), 0)}
              {@const totalPrev = sortedRows.reduce((s, r) => s + (r.prevRevenue || 0), 0)}
              {@const hasPrevOverall = sortedRows.some(r => r.prevRevenue !== null)}
              <tr class="border-t-2 border-border/60 font-semibold">
                <td class="py-2.5 px-3 text-text-primary text-sm">Total</td>
                <td class="py-2.5 px-3 text-text-primary text-right text-sm">{totalRevenue.toLocaleString('id-ID')}</td>
                <td class="py-2.5 px-3 text-text-secondary text-right text-sm">
                  {#if hasPrevOverall}
                    {totalPrev.toLocaleString('id-ID')}
                  {:else}
                    <span class="text-text-muted">—</span>
                  {/if}
                </td>
                <td class="py-2.5 px-3 text-right text-sm">
                  {#if hasPrevOverall && totalPrev > 0}
                    {@const totalChange = ((totalRevenue - totalPrev) / totalPrev) * 100}
                    <span class:font-bold={true} class:text-success={totalChange > 0} class:text-danger={totalChange < 0}>
                      {totalChange > 0 ? '+' : ''}{totalChange.toFixed(1)}%
                    </span>
                  {:else}
                    <span class="text-text-muted">—</span>
                  {/if}
                </td>
                {#if sortedRows.some(r => r.orderCount !== null)}
                  {@const totalOrders = sortedRows.reduce((s, r) => s + (r.orderCount || 0), 0)}
                  <td class="py-2.5 px-3 text-text-secondary text-right text-sm">{totalOrders}</td>
                {/if}
              </tr>
            {/if}
          </tfoot>
        </table>
      </div>
    {/if}
  </div>

</div>