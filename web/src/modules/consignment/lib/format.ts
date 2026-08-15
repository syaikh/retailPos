export function formatCurrency(value?: number): string {
  return 'Rp ' + (value ?? 0).toLocaleString('id-ID');
}

export function formatDateTime(value?: string): string {
  if (!value) return '-';
  const d = new Date(value);
  return isNaN(d.getTime()) ? '-' : d.toLocaleString('id-ID');
}

export function formatDate(value?: string): string {
  if (!value) return '-';
  const d = new Date(value);
  return isNaN(d.getTime()) ? '-' : d.toLocaleDateString('id-ID');
}