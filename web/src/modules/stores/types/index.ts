export interface Store {
  id: number;
  name: string;
  address?: string;
  phone?: string;
  is_active: boolean;
  created_at?: string;
}

export interface CreateStorePayload {
  name: string;
  address?: string;
  phone?: string;
}

export interface UpdateStorePayload {
  name?: string;
  address?: string;
  phone?: string;
  is_active?: boolean;
}

export interface StoreListParams {
  limit: number;
  offset: number;
  search?: string;
  is_active?: boolean;
}

export interface StoreListResponse {
  data: Store[];
  total: number;
}
