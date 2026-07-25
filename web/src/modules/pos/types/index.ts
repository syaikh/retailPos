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
  name: string;
  sku: string;
  price: number;
  original_price: number;
  stock: number;
  quantity: number;
  barcode?: string;
  tax_rate?: number;
  unit?: string;
  pricing_rule_id?: number;
  pricing_rule_name?: string;
  pricing_rule_type?: string;
  pricing_type?: string;
  discount?: number;
}

export interface PaymentAllocation {
  payment_method_code: string;
  amount: number;
  reference_number?: string;
}

export interface CheckoutPayload {
  items: {
    product_id: number;
    quantity: number;
    unit_price: number;
    subtotal: number;
    pricing_rule_id?: number;
    pricing_rule_name?: string;
    pricing_rule_type?: string;
    pricing_type?: string;
    original_price?: number;
  }[];
  cashier_id: number;
  store_id: number | null;
  shift_id: number | null;
  subtotal: number;
  discount: number;
  tax: number;
  total_amount: number;
  payment_method: string;
  payments?: PaymentAllocation[];
  customer_id: number | null;
  status: string;
}

export interface PaymentOption {
  id: string;
  label: string;
  icon: string;
}
