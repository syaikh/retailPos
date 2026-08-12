export { getPosProducts, getCustomers, searchCustomers, getSaleById, getLastSale } from './services/pos-service';
export { formatCurrency, calculateTax, calculateSubtotal, calculateTotalItems, QUICK_CASH_PRESETS } from './lib/pos-utils';
export type { PosProduct, CartItem, PaymentOption } from './types';
