export interface SaleItem {
  id: number;
  product_id: number;
  name?: string;
  quantity: number;
  unit_price: number;
  subtotal: number;
  dpp_amount?: number;
  tax_amount?: number;
}

export interface Sale {
  id: number;
  invoice_number: string;
  cashier_id: number;
  store_id?: number;
  subtotal: number;
  discount: number;
  tax: number;
  total_amount: number;
  payment_method: string;
  status: string;
  items: SaleItem[];
  created_at: string;
  customer_name?: string;
}

export interface SaleFilters {
  startDate: string;
  endDate: string;
  limit: number;
  offset: number;
  search?: string;
  sortBy?: string;
  sortDir?: string;
  paymentMethods?: string[];
  minTotal?: number;
  maxTotal?: number;
}
