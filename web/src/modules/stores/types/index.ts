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

/** Catalog health as reported by the readiness endpoint (shared catalogue). */
export interface ReadinessCatalog {
  active_products: number;
  zero_stock_products: number;
}

/**
 * Computed onboarding state for a store — never stored server-side, always
 * derived from staff, profile, storage location and catalog health.
 */
export interface StoreReadiness {
  store_id: number;
  name: string;
  is_active: boolean;
  address_set: boolean;
  phone_set: boolean;
  /** Active accounts per role; roles with zero staff are absent. */
  staff: Record<string, number>;
  required_roles: string[];
  catalog: ReadinessCatalog;
  storage_locations: number;
  ready: boolean;
  /** Stable blocker codes, e.g. "staff.finance", "store.phone". */
  blockers: string[];
}
