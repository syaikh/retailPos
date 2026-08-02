export interface StockOpnameSession {
  id: number;
  session_number: string;
  scope_type: 'store' | 'warehouse' | 'category' | 'product';
  scope_id: number;
  warehouse_id: number | null;
  blind_count: boolean;
  status: 'draft' | 'counting' | 'pending_approval' | 'needs_recount' | 'approved' | 'cancelled';
  created_by: number;
  approved_by: number | null;
  approved_at: string | null;
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

export interface CreateStockOpnamePayload {
  scope_type: string;
  scope_id: number;
  warehouse_id?: number | null;
  blind_count: boolean;
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

export interface ApprovePayload {
  comment: string;
}

export interface RejectPayload {
  comment: string;
}

export interface RecountPayload {
  comment: string;
}

export const STOCK_OPNAME_STATUS_LABELS: Record<string, string> = {
  draft: 'Draft',
  counting: 'Counting',
  pending_approval: 'Pending Approval',
  needs_recount: 'Needs Recount',
  approved: 'Approved',
  cancelled: 'Cancelled',
};
