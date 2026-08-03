export interface StorageLocation {
  id: number;
  code: string;
  name: string;
  warehouse_id?: number | null;
  store_id?: number | null;
  notes?: string;
  is_active: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface StorageLocationFilters {
  limit: number;
  offset: number;
  search?: string;
  is_active?: boolean;
}
