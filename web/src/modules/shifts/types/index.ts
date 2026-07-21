export interface Shift {
  id: number;
  user_id: number;
  username: string;
  store_id: number | null;
  store_name: string;
  status: 'open' | 'closed';
  opening_balance: number;
  closing_balance: number | null;
  cash_sales: number;
  non_cash_sales: number;
  total_sales: number;
  transaction_count: number;
  discrepancy: number | null;
  notes: string | null;
  needs_review: boolean;
  reviewed_by: number | null;
  reviewed_at: string | null;
  opened_at: string;
  closed_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface ShiftFilters {
  status: string;
  userId: number | null;
  limit: number;
  offset: number;
  sortBy: string;
  sortDir: string;
}
