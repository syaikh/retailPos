// Wire types mirroring internal/salereturn's JSON contracts: the pre-flight
// snapshot (returnable.go) and the return payloads (presenter.go).

export type IneligibleReason =
  "not_completed" | "window_expired" | "shift_not_open" | "fully_returned";

export interface ReturnableItem {
  sale_item_id: number;
  product_id: number;
  product_name: string;
  quantity: number;
  returned_quantity: number;
  returnable_quantity: number;
  unit_price: number;
  subtotal: number;
  tax_rate: number;
  stock_available: number;
  consignment: boolean;
}

export interface ReturnableSale {
  sale_id: number;
  invoice_number: string;
  status: string;
  created_at: string;
  total_amount: number;
  eligible: boolean;
  ineligible_reason?: IneligibleReason;
  window_days: number;
  approval_threshold: number;
  items: ReturnableItem[];
}

export interface SaleReturnReplacement {
  id: number;
  product_id: number;
  quantity: number;
}

export interface SaleReturnItem {
  id: number;
  sale_item_id: number;
  product_id: number;
  product_name: string;
  quantity: number;
  unit_price: number;
  subtotal: number;
  dpp_amount: number;
  tax_amount: number;
  tax_rate: number;
  condition: string;
  outcome: string;
  notes: string | null;
  replacements: SaleReturnReplacement[];
}

export interface SaleReturnPayment {
  id: number;
  payment_method_id: number;
  payment_method_code: string;
  amount: number;
}

export interface ReturnApproval {
  id: number;
  requested_by: number;
  approved_by: number | null;
  status: "pending" | "approved" | "rejected" | "expired";
  reason: string | null;
  rejected_reason: string | null;
  expires_at: string | null;
}

export interface SaleReturn {
  id: number;
  return_number: string;
  original_sale_id: number;
  cashier_id: number;
  store_id: number;
  customer_id: number | null;
  shift_id: number;
  return_type: string;
  reason: string;
  condition: string;
  notes: string | null;
  subtotal: number;
  discount: number;
  restocking_fee: number;
  tax_reversal: number;
  total_refund_amount: number;
  // A rejected return flips to 'cancelled'; pending_approval lazily displays
  // as 'expired' past its approval window (see presentReturn).
  status: string;
  change_due: number;
  created_at: string;
  updated_at: string;
  expires_at: string | null;
  items: SaleReturnItem[];
  payments: SaleReturnPayment[];
  approval: ReturnApproval | null;
}

export interface ReturnItemRequest {
  sale_item_id: number;
  quantity: number;
  notes?: string;
}

/** One line's selection state inside the return modal, keyed by
 * sale_item_id in the modal's `selected` record. */
export interface ReturnLineSelection {
  checked: boolean;
  quantity: number;
}

export interface ProcessReturnRequest {
  reason: string;
  // PRD §10: Phase 1 locks the condition to 'damaged' server-side.
  condition: "damaged";
  notes?: string;
  items: ReturnItemRequest[];
}
