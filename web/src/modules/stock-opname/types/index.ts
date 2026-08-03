export type StockOpnameScopeType = 'store' | 'warehouse' | 'category' | 'brand' | 'supplier' | 'product' | 'location' | 'manual';

export type StockOpnameStatus =
  | 'draft'
  | 'open'
  | 'counting'
  | 'verification'
  | 'needs_recount'
  | 'approved'
  | 'posted'
  | 'closed'
  | 'cancelled';

export interface StockOpnameScope {
  id: number;
  stock_opname_id: number;
  scope_type: StockOpnameScopeType;
  scope_id: number;
  scope_name: string;
}

export interface StockOpnameSession {
  id: number;
  session_number: string;
  title: string;
  scope_type: StockOpnameScopeType;
  scope_id: number;
  scope_name: string;
  scopes: StockOpnameScope[];
  warehouse_id: number | null;
  store_id: number | null;
  location_id: number | null;
  blind_count: boolean;
  notes: string;
  status: StockOpnameStatus;
  created_by: number;
  opened_by: number | null;
  opened_at: string | null;
  verified_by: number | null;
  verified_at: string | null;
  approved_by: number | null;
  approved_at: string | null;
  posted_by: number | null;
  posted_at: string | null;
  closed_by: number | null;
  closed_at: string | null;
  total_difference: number;
  total_adjustment: number;
  cancelled_at: string | null;
  created_at: string;
  updated_at: string;
  items?: StockOpnameItem[];
  assignments?: StockOpnameAssignment[];
  summary?: SessionSummary;
}

export interface StockOpnameItem {
  id: number;
  stock_opname_id: number;
  product_id: number;
  product_name: string;
  sku: string;
  barcode: string;
  uom_name: string;
  opening_qty: number;
  expected_qty: number;
  physical_qty: number;
  difference_qty: number;
  adjustment_qty: number;
  status: 'pending' | 'counted';
  reason: string;
  count_sequence: number;
  last_counted_by: number | null;
  last_counted_at: string | null;
}

export interface StockOpnameAssignment {
  id: number;
  stock_opname_id: number;
  user_id: number;
  username: string;
  role: 'counter' | 'supervisor';
  assigned_at: string;
}

export interface AssignableUser {
  id: number;
  username: string;
  email: string;
  role_id: number;
  role_name: string;
}

export interface SessionSummary {
  total_items: number;
  counted_items: number;
  pending_items: number;
  total_difference: number;
  total_adjustment: number;
}

export interface CountRecord {
  id: number;
  stock_opname_item_id: number;
  count_sequence: number;
  physical_qty: number;
  counted_by: number;
  counted_by_user: string;
  counted_at: string;
  remarks: string;
}

export interface StockOpnameFilters {
  status: string;
  search: string;
  limit: number;
  offset: number;
}

export interface CreateScopePayload {
  scope_type: StockOpnameScopeType;
  scope_id: number;
  scope_name?: string;
}

export interface CreateStockOpnamePayload {
  title?: string;
  scopes?: CreateScopePayload[];
  scope_type?: StockOpnameScopeType;
  scope_id?: number;
  warehouse_id?: number | null;
  store_id?: number | null;
  blind_count: boolean;
  notes?: string;
}

export interface AssignPayload {
  user_id: number;
  role: 'counter' | 'supervisor';
}

export interface ReassignPayload {
  role: 'counter' | 'supervisor';
}

export interface SaveCountPayload {
  physical_qty: number;
  remarks?: string;
}

export interface OpenPayload {
  comment: string;
}

export interface VerifyPayload {
  comment: string;
}

export interface RejectPayload {
  comment: string;
}

export interface RecountPayload {
  comment: string;
}

export interface PostAdjustmentPayload {
  comment?: string;
  notes?: string;
}

export interface AdjustmentItem {
  id: number;
  adjustment_id: number;
  product_id: number;
  product_name: string;
  sku: string;
  warehouse_id: number | null;
  store_id: number | null;
  expected_qty: number;
  physical_qty: number;
  difference_qty: number;
  adjustment_qty: number;
  unit_cost: number;
  line_total: number;
  reason: string;
}

export interface Adjustment {
  id: number;
  adjustment_number: string;
  session_id: number;
  session_number: string;
  status: string;
  notes: string;
  created_by: number;
  created_by_name: string;
  created_at: string;
  items?: AdjustmentItem[];
  total_difference: number;
  total_adjustment: number;
}

export const STOCK_OPNAME_STATUS_LABELS: Record<string, string> = {
  draft: 'Draft',
  open: 'Open',
  counting: 'Counting',
  verification: 'Verification',
  needs_recount: 'Needs Recount',
  approved: 'Approved',
  posted: 'Posted',
  closed: 'Closed',
  cancelled: 'Cancelled',
};

export const STOCK_OPNAME_SCOPE_LABELS: Record<StockOpnameScopeType, string> = {
  store: 'Store',
  warehouse: 'Warehouse',
  category: 'Category',
  brand: 'Brand',
  supplier: 'Supplier',
  product: 'Product',
  location: 'Storage Location (Rack)',
  manual: 'Manual (all active products)',
};

export const STOCK_OPNAME_SCOPE_TYPES: StockOpnameScopeType[] = [
  'store',
  'warehouse',
  'category',
  'brand',
  'supplier',
  'product',
  'location',
  'manual',
];
