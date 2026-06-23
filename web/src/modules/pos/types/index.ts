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
  stock: number;
  quantity: number;
  barcode?: string;
  tax_rate?: number;
  unit?: string;
}

export interface CheckoutPayload {
  items: {
    product_id: number;
    quantity: number;
    unit_price: number;
    subtotal: number;
  }[];
  cashier_id: number;
  store_id: number | null;
  subtotal: number;
  discount: number;
  tax: number;
  total_amount: number;
  payment_method: string;
  customer_id: number | null;
  status: string;
}

export interface PaymentOption {
  id: string;
  label: string;
  icon: string;
}
