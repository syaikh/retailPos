export interface CustomerGroup {
  id: number;
  name: string;
  description?: string;
  is_active: boolean;
  customer_count?: number;
  color?: string;
  created_at?: string;
  updated_at?: string;
}

export interface CustomerGroupFilters {
  limit: number;
  offset: number;
  search?: string;
  is_active?: boolean;
  has_customers?: boolean;
}
