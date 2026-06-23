export interface Category {
  id: number;
  name: string;
  slug?: string;
  description?: string;
  is_active?: boolean;
  product_count?: number;
  created_at?: string;
  updated_at?: string;
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

export interface Product {
  id: number;
  name: string;
  sku: string;
  price: number;
  stock: number;
  unit?: string;
  category?: Category;
  category_id?: number;
  category_name?: string;
  description?: string;
  brand_id?: number;
  brand_name?: string;
  barcode?: string;
  cost?: number;
  tax_class_id?: number;
  tax_rate?: number;
  weight_grams?: number;
  unit_of_measure_id?: number;
  unit_of_measure?: string;
  default_discount_percent?: number;
  status?: string;
  store_id?: number;
  store_name?: string;
  created_at?: string;
  updated_at?: string;
}

export interface ProductFormData {
  name: string;
  sku: string;
  barcode: string;
  category: string;
  brand_id: number | null;
  price: number;
  cost: number;
  stock: number;
  unit_of_measure_id: number | null;
  tax_class_id: number | null;
  weight_grams: number | null;
  description: string;
  status: string;
}

export interface ProductFilters {
  limit: number;
  offset: number;
  search?: string;
  category?: string[];
  status?: string;
  maxStock?: number;
  sortBy?: string;
  sortDir?: string;
}

export interface StockThreshold {
  warning: number;
  critical: number;
}


