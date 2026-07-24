import type { CartItem, CheckoutPayload } from '../types';

export function formatCurrency(value?: number): string {
  if (value == null || isNaN(value)) return 'Rp 0';
  return value.toLocaleString('id-ID');
}

export function calculateTax(items: CartItem[]): number {
  return items.reduce((sum, item) => {
    const rate = item.tax_rate || 0;
    if (rate <= 0) return sum;
    const lineTotal = item.price * item.quantity;
    const dpp = Math.round(lineTotal * 100 / (100 + rate));
    return sum + (lineTotal - dpp);
  }, 0);
}

export function calculateSubtotal(items: CartItem[]): number {
  return items.reduce((sum, item) => sum + item.price * item.quantity, 0);
}

export function calculateTotalItems(items: CartItem[]): number {
  return items.reduce((sum, item) => sum + item.quantity, 0);
}

export function buildCheckoutPayload(
  cart: CartItem[],
  paymentMethod: string,
  customerId: number | null,
  cashierId: number,
  storeId: number | null,
): CheckoutPayload {
  const subtotal = calculateSubtotal(cart);
  const tax = calculateTax(cart);
  return {
    items: cart.map((item) => ({
      product_id: item.id,
      quantity: item.quantity,
      unit_price: item.price,
      subtotal: item.price * item.quantity,
      ...(item.pricing_rule_id ? {
        pricing_rule_id: item.pricing_rule_id,
        pricing_rule_name: item.pricing_rule_name,
        pricing_rule_type: item.pricing_rule_type,
        pricing_type: item.pricing_type,
        original_price: item.original_price,
      } : {}),
    })),
    cashier_id: cashierId,
    store_id: storeId,
    shift_id: null,
    subtotal,
    discount: 0,
    tax,
    total_amount: subtotal,
    payment_method: paymentMethod,
    customer_id: customerId,
    status: 'completed',
  };
}

export const QUICK_CASH_PRESETS = [50000, 100000, 150000, 200000];

export const QUICK_SEARCH_KEY = 'F2';
export const QUICK_CHECKOUT_KEY = 'F4';
export const QUICK_FILL_TOTAL_KEY = 'F6';
