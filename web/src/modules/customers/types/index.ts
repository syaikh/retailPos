export interface Customer {
  id: number;
  name: string;
  phone?: string;
  email?: string;
  address?: string;
  tax_id?: string;
  customer_group_id?: number;
  customer_group_name?: string;
  loyalty_points?: number;
  is_walk_in?: boolean;
  is_active?: boolean;
  note?: string;
  created_at?: string;
}

export interface CustomerFilters {
  search?: string;
  isActive?: string;
  customer_group_id?: number;
  limit?: number;
  offset?: number;
}
