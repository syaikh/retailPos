export interface PosProduct {
  id: number;
  name: string;
  sku: string;
  price: number;
  stock: number;
  barcode?: string;
  tax_rate?: number;
  unit?: string;
}

export interface CartItem {
  id: number;
  cart_session_id: number;
  product_id: number;
  product_name: string;
  quantity: number;
  unit_price: number;
  original_price: number;
  discount: number;
  pricing_rule_id?: number;
  pricing_rule_name?: string;
  pricing_rule_type?: string;
  pricing_type?: string;
  cost: number;
  tax_class_id?: number;
  tax_rate?: number;
  snapshot_created_at?: string;
  subtotal: number;
  dpp_amount: number;
  tax_amount: number;
}

export type CartStatus = 'open' | 'held' | 'checked_out' | 'cancelled' | 'expired';

export interface CartSession {
  id: number;
  cashier_id: number;
  store_id?: number;
  shift_id?: number;
  customer_id?: number;
  status: CartStatus;
  subtotal: number;
  discount: number;
  tax: number;
  total_amount: number;
  expired_at?: string;
  items?: CartItem[];
  created_at?: string;
  updated_at?: string;
}

export interface PaymentAllocation {
  payment_method_code: string;
  amount: number;
  reference_number?: string;
}

export interface PaymentOption {
  id: string;
  label: string;
  icon: string;
}
