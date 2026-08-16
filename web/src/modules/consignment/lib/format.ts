import { formatLocaleDate } from '$shared/i18n';

export function formatCurrency(value?: number): string {
  return 'Rp ' + (value ?? 0).toLocaleString('id-ID');
}

export function formatDateTime(value?: string): string {
  if (!value) return '-';
  const d = new Date(value);
  return isNaN(d.getTime()) ? '-' : formatLocaleDate(d, { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' });
}

export function formatDate(value?: string): string {
  if (!value) return '-';
  const d = new Date(value);
  return isNaN(d.getTime()) ? '-' : formatLocaleDate(d, { day: 'numeric', month: 'short', year: 'numeric' });
}