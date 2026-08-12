import type { CartItem } from '../types';

export function formatCurrency(value?: number): string {
  if (value == null || isNaN(value)) return 'Rp 0';
  return value.toLocaleString('id-ID');
}

export function calculateTax(items: CartItem[]): number {
  return items.reduce((sum, item) => {
    const rate = item.tax_rate || 0;
    if (rate <= 0) return sum;
    const lineTotal = item.unit_price * item.quantity;
    const dpp = Math.round(lineTotal * 100 / (100 + rate));
    return sum + (lineTotal - dpp);
  }, 0);
}

export function calculateSubtotal(items: CartItem[]): number {
  return items.reduce((sum, item) => sum + item.unit_price * item.quantity, 0);
}

export function calculateTotalItems(items: CartItem[]): number {
  return items.reduce((sum, item) => sum + item.quantity, 0);
}

export const QUICK_CASH_PRESETS = [50000, 100000, 150000, 200000];

export const QUICK_SEARCH_KEY = 'F2';
export const QUICK_CHECKOUT_KEY = 'F4';
export const QUICK_FILL_TOTAL_KEY = 'F6';
