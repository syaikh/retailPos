export { getPosProducts, getCustomers, searchCustomers, createSale, getSaleById, getLastSale } from './services/pos-service';
export { formatCurrency, calculateTax, calculateSubtotal, calculateTotalItems, buildCheckoutPayload, QUICK_CASH_PRESETS } from './lib/pos-utils';
export type { PosProduct, CartItem, CheckoutPayload, PaymentOption } from './types';
