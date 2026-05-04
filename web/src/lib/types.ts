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

export interface Product {
  id: number;
  name: string;
  sku: string;
  price: number;
  stock: number;
  stock_min?: number;
  category?: Category;
  category_id?: number;
  description?: string;
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