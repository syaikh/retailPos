export interface SaleItem {
  id: number;
  product_id: number;
  name?: string;
  quantity: number;
  unit_price: number;
  subtotal: number;
  dpp_amount?: number;
  tax_amount?: number;
  original_price?: number;
  pricing_rule_id?: number;
  pricing_rule_name?: string;
  pricing_rule_type?: string;
  pricing_type?: string;
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
  cash_received?: number;
  change_due?: number;
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
  cashierId?: number;
  dateRange?: string;
}

export interface FilterState {
  searchQuery: string;
  paymentMethods: string[];
  minTotal: number | null;
  maxTotal: number | null;
  dateRange: string;
  startDate: string;
  endDate: string;
  page: number;
  pageSize: number;
  sortBy: string;
  sortDir: string;
}
