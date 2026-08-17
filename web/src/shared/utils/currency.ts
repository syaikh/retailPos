import { labels } from '$shared/i18n';

export function formatCurrency(value?: number, nullFallback?: string): string {
  if (value == null || isNaN(value)) return nullFallback ?? `${labels.currencySymbol} 0`;
  return `${labels.currencySymbol} ${value.toLocaleString('id-ID')}`;
}
