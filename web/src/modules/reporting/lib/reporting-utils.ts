import { getTodayInJakarta, getDateNDaysAgoInJakarta, getCurrentJakartaHour, getJakartaDayOfWeek } from '$shared/utils/jakartaTime';

export function formatCurrencyShort(value: number | null | undefined): string {
  if (value == null) return 'Rp 0';
  if (value >= 1000000000) return 'Rp ' + (value / 1000000000).toFixed(1).replace(/\.0$/, '') + 'M';
  if (value >= 1000000) return 'Rp ' + (value / 1000000).toFixed(1).replace(/\.0$/, '') + 'jt';
  if (value >= 1000) return 'Rp ' + (value / 1000).toFixed(0) + 'k';
  return 'Rp ' + value.toLocaleString('id-ID');
}

export function formatLargeNumber(value: number | null | undefined): string {
  if (value == null) return '0';
  if (value >= 1000000000) return (value / 1000000000).toFixed(1).replace(/\.0$/, '') + 'M';
  if (value >= 1000000) return (value / 1000000).toFixed(1).replace(/\.0$/, '') + 'jt';
  if (value >= 1000) return (value / 1000).toFixed(0) + 'k';
  return value.toLocaleString('id-ID');
}

export function formatDate(dateString?: string): string {
  if (!dateString) return '';
  const date = new Date(dateString + 'T00:00:00Z');
  const day = date.getUTCDate().toString().padStart(2, '0');
  const month = date.toLocaleString('id-ID', { month: 'short', timeZone: 'UTC' });
  const year = date.getUTCFullYear();
  return `${day} ${month} ${year}`;
}

export function getPeriodLabel(item: { hour?: number; date?: string; month_start?: string; label?: string }): string {
  if (!item) return '';
  if (item.hour !== undefined) return `${String(item.hour).padStart(2, '0')}:00`;
  if (item.date) {
    if (/^\d{1,2}$/.test(item.date)) {
      return `${item.date.padStart(2, '0')}:00`;
    }
    const d = new Date(item.date + 'T00:00:00Z');
    const day = d.getUTCDate();
    const month = d.toLocaleString('id-ID', { month: 'short', timeZone: 'UTC' });
    return `${day} ${month}`;
  }
  if (item.month_start) {
    const d = new Date(item.month_start + 'T00:00:00Z');
    const month = d.toLocaleString('id-ID', { month: 'short', timeZone: 'UTC' });
    const year = d.getUTCFullYear();
    return `${month} ${year}`;
  }
  return item.label || '';
}

export function formatDayDate(dateString?: string): string {
  if (!dateString) return '';
  const date = new Date(dateString + 'T00:00:00Z');
  const dayName = date.toLocaleString('id-ID', { weekday: 'short', timeZone: 'UTC' });
  const day = date.getUTCDate();
  const month = date.toLocaleString('id-ID', { month: 'short', timeZone: 'UTC' });
  return `${dayName}, ${day} ${month}`;
}

export function getFirstOfMonthNAgoInJakarta(n: number): string {
  const today = getTodayInJakarta().split('-').map(Number);
  const totalMonths = (today[0] * 12 + today[1] - 1) - n;
  const year = Math.floor(totalMonths / 12);
  const month = (totalMonths % 12) + 1;
  return `${year}-${String(month).padStart(2, '0')}-01`;
}

export function getPeriodDateRange(periodType: string): { start: string; end: string } {
  const today = getTodayInJakarta();
  const daysAgo = (n: number) => getDateNDaysAgoInJakarta(n);

  switch (periodType) {
    case 'realtime':
      return { start: today, end: today };
    case 'yesterday':
      return { start: daysAgo(1), end: daysAgo(1) };
    case '7days':
      return { start: daysAgo(7), end: daysAgo(1) };
    case '30days':
      return { start: daysAgo(30), end: daysAgo(1) };
    default: {
      const defaultEnd = daysAgo(1);
      return { start: daysAgo(8), end: defaultEnd };
    }
  }
}

export function getBackendPeriodType(activePeriodType: string): string {
  if (['realtime', 'yesterday', 'daily'].includes(activePeriodType)) return 'daily';
  if (activePeriodType === 'weekly') return 'weekly';
  if (activePeriodType === 'monthly') return 'monthly';
  if (activePeriodType === 'yearly') return 'yearly';
  return 'daily';
}

export function getComparisonMode(activePeriodType: string): string {
  if (activePeriodType === 'realtime') return 'realtime';
  if (['daily', 'yesterday'].includes(activePeriodType)) return 'completed';
  if (activePeriodType === 'yearly') return 'todate';
  if (activePeriodType === '30days') return '30days';
  return 'todate';
}

export function getShiftDays(activePeriodType: string): number {
  if (['realtime', 'daily', 'yesterday'].includes(activePeriodType)) return 1;
  if (['weekly', '7days'].includes(activePeriodType)) return 7;
  if (activePeriodType === '30days') return 30;
  return 0;
}

export function getPeriodDescription(periodType: string, currentTimeHour: string, formatDateFn: (d: string) => string): string {
  const range = getPeriodDateRange(periodType);
  const start = formatDateFn(range.start);
  const end = formatDateFn(range.end);

  switch (periodType) {
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
