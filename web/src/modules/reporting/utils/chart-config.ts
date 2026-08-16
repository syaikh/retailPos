import { getCurrentJakartaHour, getTodayInJakarta, getDateNDaysAgoInJakarta } from '$shared/utils/jakartaTime';
import { labels, formatLocaleDate } from '$shared/i18n';

interface ChartDataPoint {
  date?: string;
  total: number;
  month_start?: string;
  week_start?: string;
  week_end?: string;
  label?: string;
}

interface ChartConfigParams {
  chartType: string;
  chartData: ChartDataPoint[];
  prevChartData: Array<{ date?: string; total: number; month_start?: string; week_start?: string; week_end?: string }>;
  activePeriodType: string;
  endDate?: string;
  selectedMonthlyRange?: { start: { year: number; month: number; day: number }; end: { year: number; month: number; day: number } };
  selectedYearlyRange?: { start: { year: number }; end: { year: number } };
  chartYear: number;
}

/**
 * Builds the Chart.js configuration object from report state.
 * All dependencies are passed as parameters to avoid coupling to Svelte reactivity.
 */
export function buildChartConfig({
  chartType,
  chartData,
  prevChartData,
  activePeriodType,
  endDate,
  selectedMonthlyRange,
  selectedYearlyRange,
  chartYear,
}: ChartConfigParams) {
  let chartLabels = [];
  let values = [];
  let prevValues = [];
  let dateStrings = [];
  let prevDateStrings = [];
  let currentChartType = chartType;

  if (chartType === 'hourly') {
    currentChartType = 'line';
    const isCompletedPeriod = activePeriodType === 'yesterday' || activePeriodType === 'daily';
    const currentHour = getCurrentJakartaHour();
    const hours = isCompletedPeriod
      ? Array.from({ length: 24 }, (_, i) => i)
      : Array.from({ length: currentHour }, (_, i) => i);
    const dataByHour: Record<number, number> = {};
    chartData.forEach(d => { if (d.date) dataByHour[parseInt(d.date)] = d.total; });
    const prevByHour: Record<number, number> = {};
    prevChartData.forEach(d => { if (d.date) prevByHour[parseInt(d.date)] = d.total; });
    chartLabels = hours.map(h => `${String(h).padStart(2, '0')}:00`);
    values = hours.map(h => dataByHour[h] || 0);
    prevValues = hours.map(h => prevByHour[h] !== undefined ? prevByHour[h] : 0);
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
        const dim = new Date(year, month, 0).getDate();
        const yesterday = getDateNDaysAgoInJakarta(1);
        const dataMap: Record<string, number> = {};
        chartData.forEach(d => {
          if (d.date && d.date <= yesterday) {
            dataMap[d.date] = d.total;
          }
        });
        const prevSorted = [...prevChartData]
          .filter(d => d.date)
          .sort((a, b) => a.date!.localeCompare(b.date!));
        const prevValuesList = prevSorted.map(d => d.total || 0);
        let prevIdx = 0;
        chartLabels = [];
        values = [];
        prevValues = [];
        dateStrings = [];
        prevDateStrings = [];
        for (let day = 1; day <= dim; day++) {
          const dateStr = `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
          chartLabels.push(`${labels.day} ${day}`);
          dateStrings.push(dateStr);
          if (dateStr <= yesterday) {
            const prevItem = prevSorted[prevIdx];
            const hasPrev = prevItem !== undefined;
            prevDateStrings.push(prevItem ? prevItem.date! : '');
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
      const prevByDate: Record<string, number> = {};
      prevChartData.forEach(pd => {
        if (pd.date) prevByDate[pd.date] = pd.total;
      });
      const dataMap: Record<string, number> = {};
      chartData.forEach(d => {
        if (d.date) dataMap[d.date] = d.total;
      });
      const sortedCurrent = chartData.filter(d => d.date && endDate && d.date <= endDate).sort((a, b) => a.date!.localeCompare(b.date!));
      const sortedPrev = [...prevChartData].filter(d => d.date && endDate && d.date <= endDate).sort((a, b) => a.date!.localeCompare(b.date!));
      let dayOffset = 0;
      if (sortedCurrent.length > 0 && sortedPrev.length > 0) {
        const diffMs = new Date(sortedCurrent[0].date!).getTime() - new Date(sortedPrev[0].date!).getTime();
        dayOffset = Math.round(diffMs / 86400000);
      }
      const endDateTime = new Date(endDate + 'T00:00:00Z');
      const endDayOfWeek = endDateTime.getUTCDay();
      const daysSinceMonday = endDayOfWeek === 0 ? 6 : endDayOfWeek - 1;
      const mondayDate = new Date(endDateTime.getTime() - daysSinceMonday * 86400000);
      const sundayDate = new Date(mondayDate);
      sundayDate.setDate(mondayDate.getDate() + 6);
      const dayMs = 86400000;
      chartLabels = [];
      values = [];
      prevValues = [];
      dateStrings = [];
      prevDateStrings = [];
      for (let d = new Date(mondayDate); d <= sundayDate; d = new Date(d.getTime() + dayMs)) {
        const dateStr = d.toISOString().split('T')[0];
        const currentLabel = formatLocaleDate(d, { month: 'short', day: 'numeric' });
        if (dateStr <= (endDate ?? '')) {
          const total = dataMap[dateStr];
          const expectedPrev = new Date(d.getTime() - dayOffset * dayMs);
          const expectedPrevStr = expectedPrev.toISOString().split('T')[0];
          const prevTotal = prevByDate[expectedPrevStr];
          const hasPrev = prevTotal !== undefined;
          const prevLabel = hasPrev
            ? formatLocaleDate(expectedPrev, { month: 'short', day: 'numeric' })
            : labels.noData;
          chartLabels.push(`${currentLabel}\n${prevLabel}`);
          dateStrings.push(dateStr);
          prevDateStrings.push(hasPrev ? expectedPrevStr : '');
          values.push(total !== undefined ? total : 0);
          prevValues.push(hasPrev ? prevTotal : null);
        } else {
          chartLabels.push(`${currentLabel}\n${labels.noData}`);
          dateStrings.push(dateStr);
          prevDateStrings.push('');
          values.push(null);
          prevValues.push(null);
        }
      }
    } else {
      const prevByDate: Record<string, number> = {};
      prevChartData.forEach(pd => {
        if (pd.date) prevByDate[pd.date] = pd.total;
      });
      const sortedCurrent = chartData.filter(d => d.date).sort((a, b) => a.date!.localeCompare(b.date!));
      const sortedPrev = [...prevChartData].filter(d => d.date).sort((a, b) => a.date!.localeCompare(b.date!));
      let dayOffset = 0;
      if (sortedCurrent.length > 0 && sortedPrev.length > 0) {
        const diffMs = new Date(sortedCurrent[0].date!).getTime() - new Date(sortedPrev[0].date!).getTime();
        dayOffset = Math.round(diffMs / 86400000);
        if (isNaN(dayOffset)) dayOffset = 0;
      }
      chartLabels = [];
      values = [];
      prevValues = [];
      dateStrings = [];
      prevDateStrings = [];
      chartData.forEach((d, i) => {
        if (!d.date) {
          chartLabels.push(String(i + 1));
          dateStrings.push('');
          prevDateStrings.push('');
          values.push(d.total);
          prevValues.push(null);
          return;
        }
        const currentDate = new Date(d.date);
        if (isNaN(currentDate.getTime())) {
          chartLabels.push(String(i + 1));
          dateStrings.push('');
          prevDateStrings.push('');
          values.push(d.total);
          prevValues.push(null);
          return;
        }
        const currentLabel = formatLocaleDate(currentDate, { month: 'short', day: 'numeric' });
        const expectedPrev = new Date(currentDate.getTime() - dayOffset * 86400000);
        const expectedPrevStr = expectedPrev.toISOString().split('T')[0];
        const prevTotal = prevByDate[expectedPrevStr];
        const hasPrev = prevTotal !== undefined;
        const prevLabel = hasPrev
          ? formatLocaleDate(expectedPrev, { month: 'short', day: 'numeric' })
          : labels.noData;
        chartLabels.push(`${currentLabel}\n${prevLabel}`);
        dateStrings.push(d.date);
        prevDateStrings.push(hasPrev ? expectedPrevStr : '');
        values.push(d.total);
        prevValues.push(hasPrev ? prevTotal : null);
      });
    }
  } else if (chartType === 'monthly' || chartType === 'yearly') {
    currentChartType = 'bar';
    if (activePeriodType === 'yearly') {
      const currentByMonth: Record<number, number> = {};
      chartData.forEach(d => {
        if (d.month_start) {
          const m = parseInt(d.month_start.split('-')[1]);
          if (!isNaN(m)) currentByMonth[m] = d.total;
        }
      });
      const prevByMonth: Record<number, number> = {};
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
      chartLabels = [];
      values = [];
      prevValues = [];
      dateStrings = [];
      prevDateStrings = [];
      for (let m = 1; m <= totalMonths; m++) {
        const currentDate = new Date(chartYear, m - 1, 1);
        const currentLabel = formatLocaleDate(currentDate, { month: 'short' }) + ' ' + chartYear;
        const prevDate = new Date(prevYear, m - 1, 1);
        const hasPrevData = prevByMonth[m] !== undefined;
        const prevLabel = hasPrevData
          ? formatLocaleDate(prevDate, { month: 'short' }) + ' ' + prevYear
          : labels.noData;
        chartLabels.push(`${currentLabel}\n${prevLabel}`);
        dateStrings.push(currentDate.toISOString().split('T')[0]);
        prevDateStrings.push(hasPrevData ? prevDate.toISOString().split('T')[0] : '');
        values.push(currentByMonth[m] || 0);
        prevValues.push(hasPrevData ? prevByMonth[m] : null);
      }
    } else {
      const prevSorted = [...prevChartData]
        .filter(d => d.month_start)
        .sort((a, b) => a.month_start!.localeCompare(b.month_start!));
      chartLabels = chartData.map((d, i) => {
        if (d.month_start) {
          const currentDate = new Date(d.month_start);
          const currentLabel = formatLocaleDate(currentDate, { month: 'short', year: '2-digit' });
          const prevItem = prevSorted[i];
          const prevLabel = prevItem
            ? formatLocaleDate(new Date(prevItem.month_start!), { month: 'short' })
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
      .sort((a, b) => a.week_start!.localeCompare(b.week_start!));
    chartLabels = chartData.map((d, i) => {
      if (d.week_start && d.week_end) {
        const start = new Date(d.week_start);
        const end = new Date(d.week_end);
        const startStr = formatLocaleDate(start, { month: 'short', day: 'numeric' });
        const endStr = formatLocaleDate(end, { month: 'short', day: 'numeric' });
        const currentLabel = `${startStr} - ${endStr}`;
        const prevItem = prevSorted[i];
        const prevLabel = prevItem
          ? formatLocaleDate(new Date(prevItem.week_start!), { month: 'short', day: 'numeric' }) + ' - ' +
            formatLocaleDate(new Date(prevItem.week_end!), { month: 'short', day: 'numeric' })
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
      labels: chartLabels,
      datasets: [
        {
          label: labels.currentPeriod,
          data: values,
          borderColor: '#0ea5e9',
          backgroundColor: currentChartType === 'bar' ? '#0ea5e9' : 'rgba(14, 165, 233, 0.15)',
          borderWidth: currentChartType === 'bar' ? 0 : 2,
          pointBackgroundColor: '#0ea5e9',
          pointBorderColor: '#0ea5e9',
          pointBorderWidth: 0,
          pointRadius: currentChartType === 'bar' ? 0 : 4,
          pointHoverRadius: currentChartType === 'bar' ? 0 : 6,
          tension: 0,
          spanGaps: true
        },
        ...(hasPrevData ? [{
          label: labels.prevPeriod,
          data: prevValues,
          borderColor: '#94a3b8',
          backgroundColor: currentChartType === 'bar' ? 'rgba(148, 163, 184, 0.5)' : 'rgba(148, 163, 184, 0.05)',
          borderWidth: currentChartType === 'bar' ? 0 : 2,
          pointBackgroundColor: '#94a3b8',
          pointBorderColor: '#94a3b8',
          pointBorderWidth: 0,
          pointRadius: currentChartType === 'bar' ? 0 : 4,
          pointHoverRadius: currentChartType === 'bar' ? 0 : 6,
          tension: 0,
          spanGaps: true
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
            title: function(items: { label?: string; dataIndex: number }[]) {
              if (!items.length) return '';
              if (activePeriodType === 'monthly' && chartType === 'daily') {
                const idx = items[0].dataIndex;
                return `${labels.day} ${idx + 1}`;
              }
              return items[0].label;
            },
            label: function(context: { parsed: { y: number | null }; dataset: { label?: string }; dataIndex: number; datasetIndex: number }) {
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
                  label += ` (${formatLocaleDate(d, { month: 'short', year: 'numeric' })})`;
                 } else if (dsIdx === 1) {
                   label += `: ${labels.noData}`;
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
            footer: function(items: { parsed: { y: number | null } }[]) {
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
            callback: function(this: { getLabelForValue: (val: number) => string }, val: number, _idx: number, _ticks: unknown[]) {
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
            callback: function(value: number) {
              if (value >= 1000000000) return 'Rp ' + (value / 1000000000).toFixed(1).replace(/\.0$/, '') + ' M';
              if (value > 1000000) return 'Rp ' + (value / 1000000).toFixed(1).replace(/\.0$/, '') + ' jt';
              if (value > 1000) return 'Rp ' + (value / 1000).toFixed(0) + ' Rb';
              if (value === 0) return 'Rp 0';
              return 'Rp ' + value;
            }
          },
          min: 0,
          suggestedMax: function(context: { chart: { data: { datasets: { data: number[] }[] } } }) {
            const allValues = context.chart.data.datasets.flatMap((ds: { data: number[] }) => ds.data);
            const positiveValues = allValues.filter((v: number) => v > 0);
            if (positiveValues.length === 0 && allValues.length > 0) return 1000;
            if (allValues.length === 0) return 1000;
            const maxValue = Math.max(...allValues);
            return maxValue + maxValue * 0.1;
          }
        }
      }
    }
  };
}
