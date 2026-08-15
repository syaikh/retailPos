export interface Supplier {
  id: number;
  name: string;
  code: string;
  contact_name?: string;
  phone?: string;
  email?: string;
  address?: string;
  notes?: string;
  is_active: boolean;
  is_consignment?: boolean;
  store_id?: number;
  created_at?: string;
  updated_at?: string;
}

export interface CreateSupplierPayload {
  name: string;
  code: string;
  contact_name?: string;
  phone?: string;
  email?: string;
  address?: string;
  notes?: string;
  is_active: boolean;
  is_consignment?: boolean;
}

export interface UpdateSupplierPayload {
  name?: string;
  code?: string;
  contact_name?: string;
  phone?: string;
  email?: string;
  address?: string;
  notes?: string;
  is_active?: boolean;
  is_consignment?: boolean;
}

export interface ProductSupplier {
  id: number;
  product_id: number;
  supplier_id: number;
  supplier_sku?: string;
  unit_cost: number;
  lead_time_days: number;
  is_preferred: boolean;
  created_at?: string;
  supplier_name?: string;
  supplier_code?: string;
  product_name?: string;
  product_sku?: string;
}
