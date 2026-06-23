export interface MasterCategory {
  id: number;
  name: string;
  slug: string;
  description?: string;
  is_active: boolean;
  product_count?: number;
  created_at?: string;
  updated_at?: string;
}

export interface CreateCategoryPayload {
  name: string;
  description?: string;
}

export interface UpdateCategoryPayload {
  name?: string;
  description?: string;
  is_active?: boolean;
}

export interface MasterBrand {
  id: number;
  name: string;
  slug: string;
  is_active: boolean;
  product_count?: number;
}

export interface MasterTaxClass {
  id: number;
  name: string;
  rate: number;
  is_active: boolean;
}

export interface MasterUnitOfMeasure {
  id: number;
  name: string;
  abbreviation: string;
}

export interface MasterPaymentMethod {
  id: number;
  name: string;
  code: string;
  is_active: boolean;
}
