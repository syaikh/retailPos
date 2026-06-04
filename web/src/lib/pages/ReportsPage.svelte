<script>
  import { onMount } from 'svelte';
  import { fly } from 'svelte/transition';
  import { apiFetch } from '$lib/api/client';
  import { toast } from '$lib/stores/toast';
import { chart } from '$lib/actions/chart';
   import { getTodayInJakarta, getDateNDaysAgoInJakarta, getCurrentJakartaHour, getJakartaHourFromUTC } from '$lib/utils/jakartaTime';
  import Badge from '$lib/components/ui/Badge.svelte';
  import Skeleton from '$lib/components/ui/Skeleton.svelte';
  import Pagination from '$lib/components/ui/Pagination.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';
  import { SelectableCalendar, MonthlyCalendar, YearCalendar } from '$lib/components/calendar';
  import { CalendarDate } from '@internationalized/date';
  import {
    Receipt, BarChart3,
    CalendarDays, Download, FileSpreadsheet,
    ChevronDown, Eye, Search, X,
    Clock, TrendingUp, TrendingDown, Info,
    CircleDollarSign
  } from 'lucide-svelte';

  let loading = $state(true);
  let salesData = $state([]);
  let chartData = $state([]);
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);
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

// Selected year for display fallback
let selectedYear = $state(new Date().getFullYear());

// Export dropdown
let showExportDropdown = $state(false);

// Global transaction search (invoice number OR product name)
let searchQuery = $state('');

// Live time state for realtime updates
let currentTimeHour = $state(`${String(new Date().getHours()).padStart(2, '0')}:00`);

// Transaction details modal
let showTransactionModal = $state(false);
let selectedTransaction = $state(null);

// Debounce helper — waits 300 ms after last keystroke before resolving
function debounce(fn, delay = 300) {
  let timer;
  return (...args) => {
    clearTimeout(timer);
    timer = setTimeout(() => fn(...args), delay);
  };
}

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
      activePeriodType === 'realtime' ? 'vs YESTERDAY (00:00 - ' + String(new Date().getHours()).padStart(2, '0') + ':00)' :
      activePeriodType === 'yesterday' ? 'vs SAME DAY LAST WEEK' :
      activePeriodType === 'daily' ? 'vs SAME DAY LAST WEEK' :
      activePeriodType === '7days' ? 'vs PREVIOUS 7 DAYS' :
      activePeriodType === '30days' ? 'vs PREVIOUS 30 DAYS' :
      activePeriodType === 'weekly' ? (kpiData.isPartial && kpiData.periodInfo?.current_period ? 
        getWeeklyDayRangeLabel() : 'vs SAME WEEK LAST YEAR') :
      activePeriodType === 'monthly' ? (kpiData.isPartial ? getMonthlyDateRangeLabel() : 'vs PREVIOUS MONTH') :
      activePeriodType === 'yearly' ? 'vs PREVIOUS YEAR' : 'vs PREVIOUS PERIOD',
    comparisonLabel:
      activePeriodType === 'realtime' ? 'vs Yesterday (' + String(new Date().getHours()).padStart(2, '0') + 'hrs)' :
      activePeriodType === 'yesterday' ? 'vs Same Day Last Week' :
      activePeriodType === 'daily' ? 'vs Same Day Last Week' :
      activePeriodType === '7days' ? 'vs Previous 7 Days' :
      activePeriodType === '30days' ? 'vs Previous 30 Days' :
      activePeriodType === 'weekly' ? (kpiData.isPartial && kpiData.periodInfo?.current_period ? 
        getWeeklyDayRangeLabel() : 'vs Same Week Last Year') :
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

  // Real-time: Show hour range "vs Kemarin pada 00:00 - HH:00"
  if (activePeriodType === 'realtime') {
    const now = new Date();
    const lastFullHour = now.getHours();
    return `00:00 - ${String(lastFullHour).padStart(2, '0')}:00`;
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

// Calculate cancellation/return rate from sales data
let cancellationRate = $derived.by(() => {
  if (!salesData || salesData.length === 0) return 0;
  const returned = salesData.filter(s => s.status === 'refunded').length;
  return (returned / salesData.length) * 100;
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
  return new Date().getFullYear();
});
let daysInMonth = $derived.by(() => {
  const today = new Date();
  return new Date(today.getFullYear(), today.getMonth() + 1, 0).getDate();
});

// Projected revenue for partial monthly views
let projectedRevenue = $derived.by(() => {
  if (activePeriodType === 'monthly' && kpiData.isPartial) {
    return (kpiData.totalRevenue / new Date().getDate()) * daysInMonth;
  }
  return null;
});
let aovTrend = $derived.by(() => {
  if (!kpiData.previousAvgOrderValue || kpiData.previousAvgOrderValue === 0) return null;
  return kpiData.avgOrderValue > kpiData.previousAvgOrderValue ? 'up' : 'down';
});

// Effect to update time every minute when in realtime mode
$effect(() => {
  if (selectedPeriodType !== 'realtime') return;

  function updateTime() {
    const now = new Date();
    currentTimeHour = `${String(now.getHours()).padStart(2, '0')}:00`;
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

// Format date and time: dd mmm yyyy hh:mm:ss
const formatDateTime = (date) => {
  const day = date.getDate().toString().padStart(2, '0');
  const month = date.toLocaleString('id-ID', { month: 'short' });
  const year = date.getFullYear();
  const hours = date.getHours().toString().padStart(2, '0');
  const minutes = date.getMinutes().toString().padStart(2, '0');
  const seconds = date.getSeconds().toString().padStart(2, '0');

  return `${day} ${month} ${year} ${hours}:${minutes}:${seconds}`;
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
      return { start: daysAgo(31), end: daysAgo(1) };
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
      return { start: today, end: today };
    case 'monthly':
      if (monthlySelectionMade && selectedMonthlyRange) {
        const start = selectedMonthlyRange.start;
        const end = selectedMonthlyRange.end;
        // For chart data: query full month range to get daily aggregation
        // Frontend will filter out dates after yesterday
        let endStr = `${end.year}-${String(end.month).padStart(2, '0')}-${String(end.day).padStart(2, '0')}`;
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
          const lastDayOfPrevMonth = new Date(currentYear, currentMonth, 0).getUTCDate();
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
  if (periodType === 'daily' || periodType === 'weekly' || periodType === 'monthly' || periodType === 'yearly') {
    // Don't change selectedPeriodType - button text stays as-is
    // Don't change activePeriodType - chart stays as-is
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
      if (selectedWeeklyRange) {
        return `Weekly · ${start} - ${end}`;
      }
      return 'Weekly · Select a week';
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
  let currentChartType = chartType;

  if (chartType === 'hourly') {
    currentChartType = 'line';
    // For realtime: show all hours up to and including the last full hour
    // Example: if current time is 05:39, show hours 0-5 (last full hour is 05:00)
    labels = chartData.map(d => `${String(d.hour).padStart(2, '0')}:00`);
    values = chartData.map(d => d.total);
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
        const daysInMonth = new Date(year, month, 0).getUTCDate();
        
        // Yesterday in Jakarta timezone - max date to show actual data for
        const yesterday = getDateNDaysAgoInJakarta(1);
        
        // Create a map of actual data, only including dates up to yesterday
        const dataMap = {};
        chartData.forEach(d => {
          if (d.date && d.date <= yesterday) {
            dataMap[d.date] = d.total;
          }
        });
        
        // Month names for X-axis labels
        const monthNames = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
        
        // Generate labels for all days in month with full date format
        labels = [];
        values = [];
        for (let day = 1; day <= daysInMonth; day++) {
          const dateStr = `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
          // Format: "1 May", "2 May", etc.
          labels.push(`${day} ${monthNames[month - 1]}`);
          // Show data for dates up to yesterday, null for future dates
          // Chart.js will skip null values
          if (dateStr <= yesterday) {
            values.push(dataMap[dateStr] || 0);
          } else {
            values.push(null);
          }
        }
      }
    } else {
      // For weekly/daily view, show data as-is with month + day
      labels = chartData.map((d, i) => {
        if (!d.date) return String(i + 1);
        const date = new Date(d.date);
        return date.toLocaleString('id-ID', { month: 'short', day: 'numeric' });
      });
      values = chartData.map(d => d.total);
    }
  } else if (chartType === 'monthly' || chartType === 'yearly') {
    currentChartType = 'bar';
    labels = chartData.map(d => {
      if (d.month_start) {
        const date = new Date(d.month_start);
        // Append year for yearly view, otherwise just month name
        if (activePeriodType === 'yearly') {
          return date.toLocaleString('id-ID', { month: 'short' }) + ' ' + chartYear;
        }
        return date.toLocaleString('id-ID', { month: 'short', year: '2-digit' });
      }
      return d.label || '';
    });
    values = chartData.map(d => d.total);
  } else {
    currentChartType = 'bar';
    labels = chartData.map(d => {
      if (d.week_start && d.week_end) {
        const start = new Date(d.week_start);
        const end = new Date(d.week_end);
        const startStr = start.toLocaleString('id-ID', { month: 'short', day: 'numeric' });
        const endStr = end.toLocaleString('id-ID', { month: 'short', day: 'numeric' });
        return `${startStr} - ${endStr}`;
      }
      return d.label || '';
    });
    values = chartData.map(d => d.total);
  }

  return {
    type: currentChartType,
    data: {
      labels,
      datasets: [{
        label: 'Revenue',
        data: values,
        borderColor: '#7c3aed',
        backgroundColor: currentChartType === 'bar' ? '#7c3aed' : 'rgba(124, 58, 237, 0.1)',
        borderWidth: currentChartType === 'bar' ? 0 : 2,
        pointBackgroundColor: '#fff',
        pointBorderColor: '#7c3aed',
        pointBorderWidth: 2,
        pointRadius: currentChartType === 'bar' ? 0 : 4,
        pointHoverRadius: currentChartType === 'bar' ? 0 : 6,
        fill: currentChartType === 'bar' ? true : true,
        tension: currentChartType === 'bar' ? 0 : 0.4
      }]
    },
options: {
       responsive: true,
       maintainAspectRatio: false,
       plugins: {
         legend: { display: false },
         tooltip: {
           callbacks: {
 label: function(context) {
                let label = context.dataset.label || '';
                if (label) label += ': ';
                if (context.parsed.y !== null) {
                  const val = context.parsed.y;
                  if (val >= 1000000000) label += 'Rp ' + (val / 1000000000).toFixed(1).replace(/\.0$/, '') + ' M';
                  else if (val > 1000000) label += 'Rp ' + (val / 1000000).toFixed(1).replace(/\.0$/, '') + ' jt';
                  else if (val > 1000) label += 'Rp ' + (val / 1000).toFixed(0) + ' Rb';
                  else label += 'Rp ' + val.toLocaleString('id-ID');
                }
                return label;
              }
           }
         }
       },
scales: {
          x: {
            grid: { display: false },
            ticks: { 
              color: '#9ca3af', 
              font: { family: 'inherit' },
              // For monthly view with many labels, rotate and limit ticks
              maxRotation: activePeriodType === 'monthly' ? 45 : 0,
              minRotation: activePeriodType === 'monthly' ? 45 : 0,
              autoSkip: true,
              maxTicksLimit: activePeriodType === 'monthly' ? 15 : 20
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
             const values = context.chart.data.datasets[0].data;
             const positiveValues = values.filter(v => v > 0);
             if (positiveValues.length === 0 && values.length > 0) return 1000;
             if (values.length === 0) return 1000;
             const maxValue = Math.max(...values);
             return maxValue + maxValue * 0.1;
           }
         }
       }
     }
  };
});

let startDate = $state('');
let endDate = $state('');

async function fetchSalesWithRange(start, end) {
  try {
    loading = true;
    startDate = start;
    endDate = end;

    const params = new URLSearchParams({
      startDate: start,
      endDate: end,
      limit: limit.toString(),
      offset: offset.toString(),
      search: searchQuery.trim(),
    });

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
  // yearly: use "completed" for current year (incomplete), "todate" for completed years
  const comparisonMode = activePeriodType === 'realtime' ? 'realtime' :
    activePeriodType === 'daily' ? 'completed' :
    activePeriodType === 'yesterday' ? 'completed' :
    activePeriodType === 'yearly' && selectedYearlyRange
      ? (selectedYearlyRange.start.year === parseInt(getTodayInJakarta().split('-')[0]) ? 'completed' : 'todate')
      : activePeriodType === '30days' ? '30days' : 'todate';

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
    comparisonDate = `${year}-12-31`;
  }

  // Chart endpoint uses full month range for monthly view to get daily aggregation
  // We need to use the original end date from the calendar for chart data
  let chartEndDate = end;
  if (activePeriodType === 'monthly' && selectedMonthlyRange) {
    // For monthly view, use the full month end from calendar selection
    const calendarEnd = selectedMonthlyRange.end;
    chartEndDate = `${calendarEnd.year}-${String(calendarEnd.month).padStart(2, '0')}-${String(calendarEnd.day).padStart(2, '0')}`;
  }
  if (activePeriodType === 'yearly' && selectedYearlyRange) {
    // For yearly view, use full year (Dec 31) for monthly aggregation
    const year = selectedYearlyRange.start.year;
    chartEndDate = `${year}-12-31`;
  }

  const [salesRes, chartRes, comparisonRes] = await Promise.all([
      apiFetch(`/api/sales?${params.toString()}`),
      apiFetch(`${chartEndpoint}?startDate=${start}&endDate=${chartEndDate}`),
      apiFetch(`/api/dashboard/comparison?period=${backendPeriodType}&mode=${comparisonMode}&date=${comparisonDate}`)
    ]);

    if (salesRes.ok) {
      const data = await salesRes.json();
      // For real-time, show all transactions from today (no hour filtering)
      // The realtime view shows live data as it happens
      if (activePeriodType === 'realtime') {
        salesData = data.data || [];
        total = data.total || 0;
      } else {
        salesData = data.data || [];
        total = data.total || 0;
      }
      salesData.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
    }

    if (chartRes.ok) {
      const cData = await chartRes.json();
      const rawData = cData.data || [];
      
      // For real-time, filter to only show full hours (include last full hour, exclude current partial hour)
      // If current time is 05:39, lastFullHour is 05, show hours 0-5 (inclusive)
      if (activePeriodType === 'realtime') {
        const lastFullHour = getCurrentJakartaHour();
        chartData = rawData.filter(item => {
          const hour = item.hour ?? parseInt(item.label?.split(':')[0] ?? '-1');
          return hour <= lastFullHour; // Include last full hour (0-5 when current is 05:xx)
        });
      } else if (chartType === 'daily' && activePeriodType === 'monthly') {
        // For monthly view: keep all dates for chart generation
        // The chart config will handle the full month label generation
        chartData = rawData;
      } else {
        chartData = rawData;
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

      // For real-time, totalRevenue uses comparison.current_revenue for consistency
      // For monthly/weekly/daily views, use chart total (sum of daily points)
      const totalRevenue = activePeriodType === 'realtime'
        ? comparison.current_revenue
        : (chartType === 'daily' && activePeriodType !== '7days' && activePeriodType !== '30days')
          ? chartTotal  // Use chart total for consistency with displayed chart data
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
      const { utils, writeFile } = await import('xlsx');

      const summaryData = [
        ['Revenue Report Summary', ''],
        ['Period', `${startDate} to ${endDate}`],
        ['Granularity', chartType],
        ['', ''],
        ['Total Revenue', `Rp ${kpiData.totalRevenue.toLocaleString('id-ID')}`],
        ['Total Orders', kpiData.totalOrders],
        ['Average Order Value', `Rp ${kpiData.avgOrderValue.toLocaleString('id-ID', { maximumFractionDigits: 0 })}`],
        ['Change vs Previous Period', `${kpiData.percentChange >= 0 ? '+' : ''}${kpiData.percentChange.toFixed(1)}%`]
      ];

      const headers = chartType === 'hourly' ? ['Hour', 'Revenue'] :
                     chartType === 'daily' ? ['Date', 'Revenue'] :
                     ['Period', 'Revenue', 'Orders'];

      const dataRows = chartData.map(item => {
        if (chartType === 'hourly') return [item.hour || item.label, item.total];
        if (chartType === 'daily') return [item.date, item.total];
        return [item.label, item.total, item.order_count];
      });

      const dataData = [headers, ...dataRows];

      const workbook = utils.book_new();
      utils.book_append_sheet(workbook, utils.aoa_to_sheet(summaryData), 'Summary');
      utils.book_append_sheet(workbook, utils.aoa_to_sheet(dataData), 'Data');

      const fileName = `revenue-report-${selectedPeriodType}-${startDate}-to-${endDate}.xlsx`;
      writeFile(workbook, fileName);

      toast.success('Excel export completed');
    } catch (error) {
      toast.error('Failed to export to Excel');
    }
  }

  async function exportToPDF() {
    try {
      const { jsPDF } = await import('jspdf');
      const { default: autoTable } = await import('jspdf-autotable');

      const doc = new jsPDF();

      doc.setFontSize(16);
      doc.text(`Revenue Report - ${selectedPeriodType}`, 20, 20);

      doc.setFontSize(12);
      doc.text(`Period: ${startDate} to ${endDate}`, 20, 30);

      const summaryBody = [
        ['Total Revenue', `Rp ${kpiData.totalRevenue.toLocaleString('id-ID')}`],
        ['Total Orders', kpiData.totalOrders.toString()],
        ['Average Order Value', `Rp ${kpiData.avgOrderValue.toLocaleString('id-ID', { maximumFractionDigits: 0 })}`],
        ['Change vs Previous Period', `${kpiData.percentChange >= 0 ? '+' : ''}${kpiData.percentChange.toFixed(1)}%`]
      ];

      autoTable(doc, {
        startY: 40,
        head: [['Metric', 'Value']],
        body: summaryBody,
        theme: 'grid'
      });

      let dataHeaders, dataBody;
      if (chartType === 'hourly') {
        dataHeaders = ['Hour', 'Revenue'];
        dataBody = chartData.map(item => [item.hour || item.label, `Rp ${item.total.toLocaleString('id-ID')}`]);
      } else if (chartType === 'daily') {
        dataHeaders = ['Date', 'Revenue'];
        dataBody = chartData.map(item => [item.date, `Rp ${item.total.toLocaleString('id-ID')}`]);
      } else {
        dataHeaders = ['Period', 'Revenue', 'Orders'];
        dataBody = chartData.map(item => [
          item.label,
          `Rp ${item.total.toLocaleString('id-ID')}`,
          item.order_count
        ]);
      }

      autoTable(doc, {
        startY: doc.lastAutoTable.finalY + 10,
        head: [dataHeaders],
        body: dataBody,
        theme: 'grid'
      });

      const fileName = `revenue-report-${selectedPeriodType}-${startDate}-to-${endDate}.pdf`;
      doc.save(fileName);

      toast.success('PDF export completed');
    } catch (error) {
      toast.error('Failed to export to PDF');
    }
  }

  const statusVariant = (s) =>
    s === 'completed' ? 'success' : s === 'refunded' ? 'danger' : 'warning';

  function getPaymentMethodVariant(method = '') {
    const m = method.toLowerCase();
    if (m === 'cash') return 'success';
    if (m === 'qris' || m.includes('ewallet') || m.includes('dana') || m.includes('ovo') || m.includes('gopay') || m.includes('linkaja')) return 'default';
    if (m.includes('credit') || m.includes('debit') || m === 'card') return 'primary';
    return 'muted';
  }

  function openTransactionDetails(transaction) {
    selectedTransaction = transaction;
    showTransactionModal = true;
  }

  function handlePageChange(newOffset, newLimit) {
    offset = newOffset;
    limit = newLimit;
    // Only fetch sales data for pagination, not chart/stats
    fetchSalesTableOnly();
  }

  // Debounced version of fetchSales used by the search input
  // Only updates the sales table, NOT the chart or stats cards
  const doSearch = debounce(() => {
    offset = 0;
    // Only fetch sales data - don't re-fetch chart/comparison
    fetchSalesTableOnly();
  }, 300);

  // Fetch ONLY sales table data (for search/pagination) without re-rendering chart or stats
  async function fetchSalesTableOnly() {
    try {
      const params = new URLSearchParams({
        startDate,
        endDate,
        limit: limit.toString(),
        offset: offset.toString(),
        search: searchQuery.trim(),
      });

      const salesRes = await apiFetch(`/api/sales?${params.toString()}`);
      if (salesRes.ok) {
        const data = await salesRes.json();
        salesData = data.data || [];
        total = data.total || 0;
        salesData.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
      }
    } catch (error) {
      toast.error('Failed to load sales data');
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
      <button
        class="btn btn-secondary flex items-center gap-2"
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
      </button>

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
                           // new Date(year, month, 0) returns last day of previous month
                           // For May (month=5), gives April 30
                           endMonth = currentMonth - 1;
                           const lastDayOfPrevMonth = new Date(year, currentMonth, 0).getUTCDate();
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
      <button
        class="btn btn-primary flex items-center gap-2 transition-all duration-300"
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
      </button>
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
            Export Chart to Excel
          </button>
          <button
            class="flex items-center gap-3 px-3 py-2 text-sm font-medium text-text-secondary hover:text-text-primary hover:bg-surface-hover rounded-xl transition-all duration-200 active:scale-[0.98] w-full text-left"
            role="menuitem"
            onclick={() => { showExportDropdown = false; exportToPDF(); }}
          >
            <Download size={16} class="text-danger-light" />
            Export Chart to PDF
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
         {:else if activePeriodType === 'realtime'}
           <span class="text-xs text-text-muted font-normal ml-2">(Data as of {String(new Date().getHours()).padStart(2, '0')}:00)</span>
         {/if}
       </h3>
     </div>

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
          <div class="text-xs font-medium text-text-secondary uppercase tracking-wide">Total Revenue
            {#if activePeriodType === 'realtime'}
              <span class="text-xs text-text-muted font-normal ml-1">(up to {String(new Date().getHours()).padStart(2, '0')}:00)</span>
            {/if}
          </div>
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
          <div class="text-xs font-medium text-text-secondary uppercase tracking-wide">Total Orders
            {#if activePeriodType === 'realtime'}
              <span class="text-xs text-text-muted font-normal ml-1" title="Data reflects performance up to the last completed hour">(up to {String(new Date().getHours()).padStart(2, '0')}:00)</span>
            {/if}
          </div>
          <div class="text-lg font-bold text-text-primary mt-1">
            {formatLargeNumber(kpiData.totalOrders)}
          </div>
          {#if cancellationRate > 0}
            <div class="text-xs text-danger mt-1 cursor-help" title="Cancellation/Return Rate">
              {cancellationRate.toFixed(1)}% returned
            </div>
          {/if}
          {#if kpiData.previousOrders > 0}
            <div class="text-xs text-text-secondary mt-1 font-medium">
              vs {formatLargeNumber(kpiData.previousOrders)}
            </div>
          {/if}
        </div>

        <div class="bg-surface rounded-lg p-4 border border-border/50">
          <div class="text-xs font-medium text-text-secondary uppercase tracking-wide">Avg Order Value
            {#if activePeriodType === 'realtime'}
              <span class="text-xs text-text-muted font-normal ml-1" title="Data reflects performance up to the last completed hour">(up to {String(new Date().getHours()).padStart(2, '0')}:00)</span>
            {/if}
          </div>
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
            {#if activePeriodType === 'realtime'}
              <span class="text-xs text-text-muted font-normal ml-1" title="Data reflects performance up to the last completed hour">(up to {String(new Date().getHours()).padStart(2, '0')}:00)</span>
            {/if}
          </div>
          <div class="flex items-baseline gap-1 mt-1">
            <span class="text-lg font-bold text-text-primary">
              {formatCurrencyShort(kpiData.peakRevenueHour !== null ? kpiData.peakRevenueHour : kpiData.revenuePerDay)}
            </span>
          </div>
          {#if kpiData.previousPeakRevenue !== null && kpiData.previousPeakRevenue > 0}
            <div class="text-xs text-text-secondary mt-1 font-medium">
              vs {formatCurrencyShort(kpiData.previousPeakRevenue)}
            </div>
          {:else if kpiData.previousRevenuePerDay > 0 && chartType !== 'hourly'}
            <div class="text-xs text-text-secondary mt-1 font-medium">
              vs {formatCurrencyShort(kpiData.previousRevenuePerDay)}
            </div>
          {/if}
        </div>

        <div class="bg-surface rounded-lg p-4 border border-border/50">
          <div class="text-xs font-medium text-text-secondary uppercase tracking-wide flex items-center gap-1">
            {statCardLabels.comparisonLabel}
            {#if activePeriodType === 'realtime'}
              <Info 
                size={12} 
                class="text-text-muted cursor-help" 
                title="Data reflects performance up to the last completed hour"
              />
            {/if}
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
          {#if comparisonDateRange}
            <div class="text-xs text-text-muted mt-1">
              vs {comparisonDateRange}
            </div>
          {/if}
        </div>
      {/if}
    </div>

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
        <canvas use:chart={chartConfig}></canvas>
      {/if}
    </div>
  </div>

  <!-- Sales table -->
  <div class="card p-0 overflow-hidden">
    <div class="px-4 py-3 border-b border-border flex flex-wrap items-center justify-between gap-3">
      <p class="text-sm font-semibold text-text-primary">Transaction History</p>
      <div class="flex items-center gap-3">
        {#if !loading}
          <span class="badge badge-muted">{total} records</span>
        {/if}
        <div class="relative">
          <Search
            size={15}
            class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none"
          />
          <input
            type="text"
            placeholder="Cari invoice atau produk..."
            bind:value={searchQuery}
            oninput={doSearch}
            class="pl-9 pr-4 py-1.5 text-sm bg-slate-900/50 border border-slate-800
                   text-white rounded-full outline-none transition-colors
                   focus:border-purple-500 placeholder:text-text-muted"
          />
          {#if searchQuery}
            <button
              type="button"
              onclick={() => { searchQuery = ''; doSearch(); }}
              class="absolute right-2.5 top-1/2 -translate-y-1/2 p-0.5
                     rounded-full hover:bg-slate-700 active:scale-95
                     text-text-muted hover:text-white transition-colors"
              aria-label="Clear search"
            >
              <X size={13} />
            </button>
          {/if}
        </div>
      </div>
    </div>

    {#if loading}
      <div class="divide-y divide-border">
        {#each { length: 5 } as _}
          <div class="flex items-center gap-4 px-4 py-3.5">
            <Skeleton width="w-32" height="h-4" />
            <Skeleton width="w-24" height="h-4" />
            <Skeleton width="w-20" height="h-6" rounded="rounded-full" class="ml-auto" />
            <Skeleton width="w-28" height="h-4" />
          </div>
        {/each}
      </div>
{:else if salesData.length === 0}
       <div class="px-4 py-12 text-center">
         <div class="empty-state-icon bg-surface w-20 h-20 mx-auto">
           <Banknote size={32} class="text-text-muted" />
         </div>
         <p class="text-text-primary font-semibold mt-4">No transactions found</p>
         <p class="text-text-muted text-sm mt-1">Try adjusting the date range</p>
       </div>
     {:else}
      <div class="overflow-x-auto">
        <table>
          <thead class="sticky top-0 bg-bg-secondary z-10 shadow-sm">
            <tr>
              <th>Invoice</th>
              <th>Date</th>
              <th>Items</th>
              <th>Payment</th>
              <th class="text-right">Total (Rp)</th>
            </tr>
          </thead>
          <tbody>
            {#each salesData as sale (sale.id)}
              <tr class="border-t border-border hover:bg-surface-hover/50 transition-colors">
                <td>
                  <button
                    class="font-mono text-sm font-medium text-white hover:text-primary-light transition-colors flex items-center gap-1.5 group underline decoration-border-strong underline-offset-4 hover:decoration-primary-light cursor-pointer"
                    onclick={() => openTransactionDetails(sale)}
                  >
                    <Eye size={14} class="opacity-70 group-hover:opacity-100 transition-opacity" />
                    {sale.invoice_number}
                  </button>
                </td>
                <td class="text-sm text-text-secondary">
                  {formatDateTime(new Date(sale.created_at))}
                </td>
                <td class="text-sm text-text-secondary">
                  {sale.items?.length || 0} items
                </td>
                <td>
                  <Badge variant={getPaymentMethodVariant(sale.payment_method)} class="text-sm px-3 py-1">
                    {sale.payment_method || '—'}
                  </Badge>
                </td>
                <td class="text-right text-sm font-semibold text-text-primary">
                  {(sale.total_amount || 0).toLocaleString('id-ID')}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>

      <div class="p-4 bg-surface-subtle/30">
        <Pagination 
          {total} 
          {limit} 
          {offset} 
          onPageChange={handlePageChange} 
        />
      </div>
    {/if}
  </div>

  <!-- Transaction Details Modal -->
  <Modal bind:open={showTransactionModal} title="Transaction Details" size="md">
    {#if selectedTransaction}
      <div class="space-y-4">
        <div>
          <p class="text-sm font-medium text-text-secondary">Invoice Number</p>
          <p class="text-text-primary">{selectedTransaction.invoice_number}</p>
        </div>
        <div>
          <p class="text-sm font-medium text-text-secondary">Date & Time</p>
          <p class="text-text-primary">{formatDateTime(new Date(selectedTransaction.created_at))}</p>
        </div>
        <div>
          <p class="text-sm font-medium text-text-secondary">Payment Method</p>
          <Badge variant={getPaymentMethodVariant(selectedTransaction.payment_method)} class="mt-1 text-sm px-3 py-1">
            {selectedTransaction.payment_method || '—'}
          </Badge>
        </div>
        <div>
          <p class="text-sm font-medium text-text-secondary">Status</p>
          <Badge variant={statusVariant(selectedTransaction.status)} class="mt-1">
            {selectedTransaction.status || 'completed'}
          </Badge>
        </div>
        {#if selectedTransaction.items && selectedTransaction.items.length > 0}
          <div>
            <p class="text-sm font-medium text-text-secondary mb-2 block">Items</p>
            <div class="space-y-2">
              {#each selectedTransaction.items as item}
                <div class="flex justify-between items-center py-2 px-3 bg-surface rounded-md border border-border">
                  <div>
                    <p class="text-sm font-medium text-text-primary">{item.name}</p>
                    <p class="text-xs text-text-secondary">Qty: {item.quantity}</p>
                  </div>
                  <p class="text-sm text-text-primary">{(item.unit_price * item.quantity).toLocaleString('id-ID')}</p>
                </div>
              {/each}
            </div>
          </div>
        {/if}
        <div class="border-t border-border pt-4">
          <div class="flex justify-between items-center">
            <span class="text-sm font-medium text-text-secondary">Total Amount</span>
            <span class="text-lg font-semibold text-text-primary">Rp {(selectedTransaction.total_amount || 0).toLocaleString('id-ID')}</span>
          </div>
        </div>
      </div>
    {/if}
  </Modal>
</div>