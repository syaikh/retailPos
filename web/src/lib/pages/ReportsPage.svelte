<script>
  import { onMount } from 'svelte';
  import { fly } from 'svelte/transition';
  import { apiFetch } from '$lib/api/client';
  import { toast } from '$lib/stores/toast';
  import { chart } from '$lib/actions/chart';
  import { getTodayInJakarta, getDateNDaysAgoInJakarta } from '$lib/utils/jakartaTime';
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
    Clock,
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
  ['7days', '30days', 'weekly', 'monthly'].includes(activePeriodType) ? 'daily' : 'monthly'
);

// Derived: Stat card labels based on period selection (per PRD section 5)
let statCardLabels = $derived({
  card4: 
    chartType === 'hourly' ? 'Peak Revenue Hour' :
    activePeriodType === 'yearly' ? 'Avg. Revenue / Month' :
    activePeriodType === 'monthly' ? 'Avg. Revenue / Week' : 'Avg. Revenue / Day',
  card5: 
    activePeriodType === 'realtime' ? 'vs Same Hours Yesterday' :
    activePeriodType === 'yesterday' ? 'vs Same Day Last Week' : 'vs Previous Period'
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

// Format date: dd mmm yyyy (id-ID format per PRD)
const formatDate = (dateString) => {
  if (!dateString) return '';
  const date = new Date(dateString);
  const day = date.getDate().toString().padStart(2, '0');
  const month = date.toLocaleString('id-ID', { month: 'short' });
  const year = date.getFullYear();
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
      return { start: daysAgo(8), end: daysAgo(1) };
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
        let endStr = `${end.year}-${String(end.month).padStart(2, '0')}-${String(end.day).padStart(2, '0')}`;
        // If current month selected, constrain end to yesterday
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
      return { start: today, end: today };
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
    labels = chartData.map((d, i) => `${String(i).padStart(2, '0')}:00`);
    values = chartData.map(d => d.total);
  } else if (chartType === 'daily') {
    currentChartType = 'line';
    labels = chartData.map(d => {
      const date = new Date(d.date);
      return date.toLocaleString('id-ID', { month: 'short', day: 'numeric' });
    });
    values = chartData.map(d => d.total);
  } else if (chartType === 'monthly') {
    currentChartType = 'bar';
    labels = chartData.map(d => {
      if (d.month_start) {
        const date = new Date(d.month_start);
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
                label += 'Rp ' + context.parsed.y.toLocaleString('id-ID');
              }
              return label;
            }
          }
        }
      },
      scales: {
        x: {
          grid: { display: false },
          ticks: { color: '#9ca3af', font: { family: 'inherit' } }
        },
        y: {
          border: { display: false },
          grid: { color: 'rgba(255, 255, 255, 0.05)' },
          ticks: {
            color: '#9ca3af',
            font: { family: 'inherit' },
            callback: function(value) {
              if (value > 1000000) return 'Rp ' + (value / 1000000).toFixed(1) + ' Jt';
              if (value > 1000) return 'Rp ' + (value / 1000).toFixed(0) + ' ribu';
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
    const chartEndpoint = chartType === 'monthly'
      ? '/api/dashboard/chart/monthly'
      : chartType === 'high-unit'
      ? '/api/dashboard/chart/yearly'
      : '/api/dashboard/chart';

    const [salesRes, chartRes, comparisonRes] = await Promise.all([
      apiFetch(`/api/sales?${params.toString()}`),
      apiFetch(`${chartEndpoint}?startDate=${start}&endDate=${end}`),
      apiFetch(`/api/dashboard/comparison?period=${selectedPeriodType}&mode=todate&date=${end}`)
    ]);

    if (salesRes.ok) {
      const data = await salesRes.json();
      salesData = data.data || [];
      total = data.total || 0;
      salesData.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
    }

    if (chartRes.ok) {
      const cData = await chartRes.json();
      chartData = cData.data || [];
    }

    if (comparisonRes.ok) {
      const compData = await comparisonRes.json();
      const comparison = compData.data;
      const meta = compData.meta;

      let percentChange = 0;
      let comparisonType = 'zero';

      if (comparison.previous_revenue === 0 && comparison.current_revenue > 0) {
        comparisonType = 'new';
        percentChange = Infinity;
      } else if (comparison.previous_revenue === 0 && comparison.current_revenue === 0) {
        comparisonType = 'zero';
        percentChange = 0;
      } else if (comparison.previous_revenue > 0) {
        comparisonType = 'normal';
        percentChange = ((comparison.current_revenue - comparison.previous_revenue) / comparison.previous_revenue) * 100;
      }

      kpiData = {
        totalRevenue: comparison.current_revenue,
        previousRevenue: comparison.previous_revenue,
        totalOrders: comparison.current_orders,
        previousOrders: comparison.previous_orders,
        avgOrderValue: comparison.current_aov,
        previousAvgOrderValue: comparison.previous_aov,
        revenuePerDay: comparison.revenue_per_day,
        previousRevenuePerDay: comparison.previous_revenue_per_day,
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
    const range = getPeriodDateRange(selectedPeriodType);
    fetchSalesWithRange(range.start, range.end);
  }

  // Debounced version of fetchSales used by the search input
  const doSearch = debounce(() => {
    offset = 0;
    const range = getPeriodDateRange(selectedPeriodType);
    fetchSalesWithRange(range.start, range.end);
  }, 300);

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
                         let endStr = `${end.year}-${String(end.month).padStart(2, '0')}-${String(end.day).padStart(2, '0')}`;
                         // If current month selected, constrain to yesterday
                         const todayJakarta = getTodayInJakarta().split('-').map(Number);
                         if (start.year === todayJakarta[0] && start.month === todayJakarta[1]) {
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
      </h3>
    </div>

    <!-- KPI Cards -->
    <div class="grid grid-cols-2 lg:grid-cols-5 gap-4 mb-6">
      {#if loading}
        {#each { length: 5 } as _}
          <div class="bg-surface rounded-lg p-4 border border-border/50">
            <Skeleton width="w-20" height="h-3" class="mb-2" />
            <Skeleton width="w-16" height="h-6" />
          </div>
        {/each}
      {:else}
         <div class="bg-surface rounded-lg p-4 border border-border/50">
           <div class="text-xs font-medium text-text-secondary uppercase tracking-wide">Total Revenue</div>
           <div class="text-lg font-bold text-text-primary mt-1">
             Rp {kpiData.totalRevenue.toLocaleString('id-ID')}
           </div>
           {#if kpiData.previousRevenue > 0}
             <div class="text-xs text-text-muted mt-1">
               vs Rp {kpiData.previousRevenue.toLocaleString('id-ID')}
             </div>
           {/if}
         </div>

         <div class="bg-surface rounded-lg p-4 border border-border/50">
           <div class="text-xs font-medium text-text-secondary uppercase tracking-wide">Total Orders</div>
           <div class="text-lg font-bold text-text-primary mt-1">
             {kpiData.totalOrders.toLocaleString()}
           </div>
           {#if kpiData.previousOrders > 0}
             <div class="text-xs text-text-muted mt-1">
               vs {kpiData.previousOrders.toLocaleString()}
             </div>
           {/if}
         </div>

         <div class="bg-surface rounded-lg p-4 border border-border/50">
           <div class="text-xs font-medium text-text-secondary uppercase tracking-wide">Avg Order Value</div>
           <div class="text-lg font-bold text-text-primary mt-1">
             Rp {kpiData.avgOrderValue.toLocaleString('id-ID', { maximumFractionDigits: 0 })}
           </div>
           {#if kpiData.previousAvgOrderValue > 0}
             <div class="text-xs text-text-muted mt-1">
               vs Rp {kpiData.previousAvgOrderValue.toLocaleString('id-ID', { maximumFractionDigits: 0 })}
             </div>
           {/if}
         </div>

         <div class="bg-surface rounded-lg p-4 border border-border/50">
           <div class="text-xs font-medium text-text-secondary uppercase tracking-wide">{statCardLabels.card4}</div>
           <div class="text-lg font-bold text-text-primary mt-1">
             Rp {kpiData.revenuePerDay.toLocaleString('id-ID')}
           </div>
           {#if kpiData.previousRevenuePerDay > 0}
             <div class="text-xs text-text-muted mt-1">
               vs Rp {kpiData.previousRevenuePerDay.toLocaleString('id-ID')}
             </div>
           {/if}
         </div>

         <div class="bg-surface rounded-lg p-4 border border-border/50">
           <div class="text-xs font-medium text-text-secondary uppercase tracking-wide">
             {statCardLabels.card5}
             {#if kpiData.isPartial}
               <span class="ml-1 text-[10px] bg-warning/20 text-warning px-1.5 py-0.5 rounded">
                 Partial
               </span>
             {/if}
           </div>
           <div class="flex items-center mt-1">
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
             <span class={`ml-2 ${
               kpiData.comparisonType === 'new' ? 'text-success' :
               kpiData.comparisonType === 'zero' ? 'text-text-secondary' :
               kpiData.percentChange > 0 ? 'text-success' : 'text-danger'
             }`}>
               {kpiData.comparisonType === 'new' ? '🚀' :
                kpiData.comparisonType === 'zero' ? '—' :
                kpiData.percentChange > 0 ? '↗' : '↘'}
             </span>
           </div>
           {#if kpiData.periodInfo}
             <div class="text-xs text-text-muted mt-1">
               {kpiData.periodInfo.current_period.start} to {kpiData.periodInfo.current_period.end}
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
          <Receipt size={32} class="text-text-muted" />
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