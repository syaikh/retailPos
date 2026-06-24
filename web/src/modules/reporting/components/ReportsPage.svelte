<script>
  import { onMount } from 'svelte';
  import { apiFetch } from '$shared/api/http-client';
  import { toast } from '$shared/stores/toast.svelte';
  import { getTodayInJakarta, getDateNDaysAgoInJakarta, getCurrentJakartaHour } from '$shared/utils/jakartaTime';
  import { formatDate, getPeriodLabel } from '$modules/reporting/lib/reporting-utils';
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

  let chartConfig = $derived.by(() => {
    let labels = [];
    let values = [];
    let prevValues = [];
    let dateStrings = [];
    let prevDateStrings = [];
    let currentChartType = chartType;

    if (chartType === 'hourly') {
      currentChartType = 'line';
      labels = chartData.map(d => `${String(d.hour).padStart(2, '0')}:00`);
      values = chartData.map(d => d.total);
      prevValues = chartData.map(d => {
        const prev = prevChartData.find(p => p.hour === d.hour);
        return prev ? prev.total : null;
      });
    } else if (chartType === 'daily') {
      currentChartType = 'line';
      if (activePeriodType === 'monthly') {
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
          const yesterday = getDateNDaysAgoInJakarta(1);
          const dataMap = {};
          chartData.forEach(d => {
            if (d.date && d.date <= yesterday) {
              dataMap[d.date] = d.total;
            }
          });
          const prevSorted = [...prevChartData]
            .filter(d => d.date)
            .sort((a, b) => a.date.localeCompare(b.date));
          const prevValuesList = prevSorted.map(d => d.total || 0);
          let prevIdx = 0;
          labels = [];
          values = [];
          prevValues = [];
          dateStrings = [];
          prevDateStrings = [];
          for (let day = 1; day <= daysInMonth; day++) {
            const dateStr = `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
            labels.push(`Day ${day}`);
            dateStrings.push(dateStr);
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
        const prevByDate = {};
        prevChartData.forEach(pd => {
          if (pd.date) prevByDate[pd.date] = pd.total;
        });
        const dataMap = {};
        chartData.forEach(d => {
          if (d.date) dataMap[d.date] = d.total;
        });
        const sortedCurrent = chartData.filter(d => d.date && d.date <= endDate).sort((a, b) => a.date.localeCompare(b.date));
        const sortedPrev = [...prevChartData].filter(d => d.date && d.date <= endDate).sort((a, b) => a.date.localeCompare(b.date));
        let dayOffset = 0;
        if (sortedCurrent.length > 0 && sortedPrev.length > 0) {
          const diffMs = new Date(sortedCurrent[0].date).getTime() - new Date(sortedPrev[0].date).getTime();
          dayOffset = Math.round(diffMs / 86400000);
        }
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
        const prevByDate = {};
        prevChartData.forEach(pd => {
          if (pd.date) prevByDate[pd.date] = pd.total;
        });
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
                    const d = new Date(date + 'T00:00:00Z');
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
        const start = selectedMonthlyRange.start;
        if (start.year === todayJakarta[0] && start.month === todayJakarta[1]) {
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

      exportPeriod = backendPeriodType;
      exportMode = comparisonMode;
      exportDate = comparisonDate;
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

        const chartTotal = chartData.reduce((sum, item) => {
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

        if (activePeriodType === 'monthly' && !meta.is_partial) {
          prevChartData = [];
          kpiData.previousRevenue = 0;
          kpiData.previousOrders = 0;
          kpiData.previousAvgOrderValue = 0;
          kpiData.previousRevenuePerDay = 0;
          kpiData.previousPeakRevenue = null;
          kpiData.previousPeakRevenueMonth = null;
        }
      }
    } catch (error) {
      toast.error('Failed to load sales data');
    } finally {
      loading = false;
    }
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

      const rangeDateFormat = (ds) => {
        if (!ds) return '';
        const d = new Date(ds + 'T00:00:00Z');
        return `${String(d.getUTCDate()).padStart(2, '0')} ${d.toLocaleString('id-ID', { month: 'short', timeZone: 'UTC' })} ${d.getUTCFullYear()}`;
      };
      const pDesc = (() => {
        const s = rangeDateFormat(startDate);
        const e = rangeDateFormat(endDate);
        switch (selectedPeriodType) {
          case 'realtime': return `Real-time (00:00 - ${currentTimeHour})`;
          case 'yesterday': return `Yesterday · ${s}`;
          case '7days': return `7 Days · ${s} - ${e}`;
          case '30days': return `30 Days · ${s} - ${e}`;
          case 'daily': return `Daily · ${s}`;
          case 'weekly': return `Weekly · ${s} - ${e}`;
          case 'monthly': return `Monthly · ${s} - ${e}`;
          case 'yearly': return `Yearly · ${s} - ${e}`;
          default: return `${s} - ${e}`;
        }
      })();

      doc.setFontSize(16);
      doc.text('Revenue Report', margin, yPos);
      yPos += 8;
      doc.setFontSize(10);
      doc.text(`Period: ${pDesc}`, margin, yPos);
      yPos += 6;
      doc.text(`Granularity: ${chartType === 'hourly' ? 'Hourly' : chartType === 'daily' ? 'Daily' : 'Periodic'}`, margin, yPos);
      yPos += 6;

      if (comparisonDateRange) {
        doc.setFontSize(9);
        doc.text(`Comparison: ${statCardLabels.comparisonLabel} · ${comparisonDateRange}`, margin, yPos);
        yPos += 6;
      }

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

      if (bestPeriod) {
        doc.setFontSize(9);
        doc.text(`Best ${bestWorstHeading}: ${getPeriodLabel(bestPeriod)} — Rp ${(bestPeriod.total || 0).toLocaleString('id-ID')}`, margin, yPos);
        yPos += 5;
      }
      if (worstPeriod && worstPeriod.total !== bestPeriod?.total) {
        doc.setFontSize(9);
        doc.text(`Worst ${bestWorstHeading}: ${getPeriodLabel(worstPeriod)} — Rp ${(worstPeriod.total || 0).toLocaleString('id-ID')}`, margin, yPos);
        if (chartType === 'hourly') {
          doc.setFontSize(8);
          doc.setFont('Helvetica', 'italic');
          doc.text('(zero-revenue hours excluded)', margin, yPos + 3);
          doc.setFont('Helvetica', 'normal');
          yPos += 8;
        } else {
          yPos += 5;
        }
      }

      doc.addPage();
      yPos = 20;

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
