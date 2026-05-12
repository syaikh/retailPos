export interface User {
  id: number;
  username: string;
  email: string;
  role?: {
    id: number;
    name: string;
  } | string;
  store_id?: number;
  is_active?: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface Category {
  id: number;
  name: string;
  description?: string;
}

export interface Brand {
  id: number;
  name: string;
  description?: string;
  is_active?: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface TaxClass {
  id: number;
  name: string;
  rate_percent: number;
  description?: string;
  is_active?: boolean;
  created_at?: string;
}

export interface UnitOfMeasure {
  id: number;
  code: string;
  name: string;
  description?: string;
  is_active?: boolean;
  created_at?: string;
}

export interface Warehouse {
  id: number;
  name: string;
  code: string;
  address?: string;
  store_id?: number;
  is_active?: boolean;
  created_at?: string;
}

export interface Dimensions {
  length: number;
  width: number;
  height: number;
}

export interface Product {
  id: number;
  name: string;
  sku: string;
  price: number;
  stock: number;
  stock_min?: number;
  unit?: string;
  category?: Category;
  category_id?: number;
  description?: string;
  // Phase 1 extensions
  brand_id?: number;
  brand_name?: string;
  barcode?: string;
  cost?: number;
  tax_class_id?: number;
  tax_rate?: number;
  weight_grams?: number;
  dimensions_cm?: Dimensions;
  unit_of_measure_id?: number;
  unit_of_measure?: string;
  default_discount_percent?: number;
  is_track_expiry?: boolean;
  is_track_batch?: boolean;
  is_active_for_sale?: boolean;
  is_active_for_purchase?: boolean;
  status?: string; // draft, active, inactive, discontinued, archived
  store_id?: number;
  created_at?: string;
  updated_at?: string;
}

export interface SaleItem {
  id: number;
  product_id: number;
  quantity: number;
  unit_price: number;
  subtotal: number;
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
}

export interface Permission {
  id: number;
  name: string;
  code: string;
  description?: string;
}

export interface Role {
  id: number;
  name: string;
  description?: string;
  is_system?: boolean;
  permissions: string[];
}

export interface AuditLog {
  id: number;
  actor?: string;
  username?: string;
  action?: string;
  resource?: string;
  description?: string;
  ip_address?: string;
  created_at?: string;
  timestamp?: string;
}