import { formatLocaleDateInJakarta } from '$shared/utils/jakartaTime';
import { formatCurrency } from '$shared/utils/currency';

export { formatCurrency };

export function formatDateTime(value?: string): string {
  if (!value) return '-';
  return formatLocaleDateInJakarta(value, { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' });
}

export function formatDate(value?: string): string {
  if (!value) return '-';
  return formatLocaleDateInJakarta(value, { day: 'numeric', month: 'short', year: 'numeric' });
}