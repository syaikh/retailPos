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
  needsReview: boolean | null;
  discrepancy: string;
  userId: number | null;
  limit: number;
  offset: number;
  sortBy: string;
  sortDir: string;
}

export interface CashMovement {
  id: number;
  shift_id: number;
  user_id: number;
  username?: string;
  type: 'cash_drop' | 'paid_in' | 'paid_out';
  amount: number;
  description?: string;
  created_at: string;
}

export interface CashMovementSummary {
  cash_drops: number;
  paid_ins: number;
  paid_outs: number;
  net_effect: number;
}

export interface PaymentMethodTotal {
  method: string;
  amount: number;
  count: number;
}

export interface ShiftReportData {
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
  opened_at: string;
  closed_at: string | null;
  duration_minutes: number;
  payment_breakdown: PaymentMethodTotal[];
  cash_movement_summary: CashMovementSummary;
}
