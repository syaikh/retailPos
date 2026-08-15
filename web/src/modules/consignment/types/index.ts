import type { Labels } from '$shared/i18n';

export interface ConsignmentSupplierRef {
  id: number;
  name: string;
  is_consignment: boolean;
}

export interface Term {
  id: number;
  arrangement_id: number;
  product_id: number;
  product_sku?: string;
  product_name?: string;
  price: number;
  store_share_type: string;
  store_share_value: number;
  effective_from?: string;
  created_by: number;
  created_at: string;
}

export interface Arrangement {
  id: number;
  supplier_id: number;
  supplier_name?: string;
  store_id: number;
  store_name?: string;
  status: string;
  last_visit_at?: string;
  ended_at?: string;
  created_by: number;
  created_at: string;
  updated_at: string;
  terms: Term[];
}

export interface ReceiptItem {
  id: number;
  consignment_receipt_id: number;
  product_id: number;
  product_sku?: string;
  product_name?: string;
  accepted_qty: number;
  price: number;
  store_share_type: string;
  store_share_value: number;
  notes?: string;
}

export interface Receipt {
  id: number;
  receipt_number: string;
  supplier_id: number;
  supplier_name?: string;
  store_id: number;
  arrangement_id: number;
  received_by: number;
  received_by_username?: string;
  received_at: string;
  notes?: string;
  created_at: string;
  items: ReceiptItem[];
}

export interface StockRow {
  product_id: number;
  product_sku?: string;
  product_name?: string;
  supplier_id: number;
  supplier_name?: string;
  arrangement_id: number;
  store_id: number;
  available_qty: number;
  pending_return_qty: number;
  updated_at?: string;
}

export interface PendingReturn {
  id: number;
  supplier_id: number;
  product_id: number;
  product_sku?: string;
  product_name?: string;
  arrangement_id: number;
  store_id: number;
  qty: number;
  reason: string;
  notes?: string;
  status: string;
  returned_at?: string;
  created_by: number;
  created_at: string;
}

export interface ReturnItem {
  id: number;
  consignment_return_id: number;
  product_id: number;
  product_sku?: string;
  product_name?: string;
  qty: number;
  reason: string;
  pending_return_id?: number;
  notes?: string;
}

export interface ConsignmentReturn {
  id: number;
  return_number: string;
  supplier_id: number;
  supplier_name?: string;
  store_id: number;
  arrangement_id: number;
  returned_by: number;
  returned_by_username?: string;
  returned_at: string;
  notes?: string;
  created_at: string;
  items: ReturnItem[];
}

export interface SettlementItem {
  id: number;
  consignment_settlement_id: number;
  consignment_sale_item_id: number;
  product_id: number;
  product_name?: string;
  quantity: number;
  unit_price: number;
  subtotal: number;
  store_share: number;
}

export interface Payout {
  id: number;
  payout_number: string;
  settlement_id: number;
  payment_method_id: number;
  payment_method_code?: string;
  payment_method_name?: string;
  amount: number;
  reference_number?: string;
  paid_by: number;
  paid_by_username?: string;
  paid_at: string;
  notes?: string;
  created_at: string;
}

export interface Settlement {
  id: number;
  settlement_number: string;
  supplier_id: number;
  supplier_name?: string;
  store_id: number;
  total_sale_value: number;
  total_store_share: number;
  total_payable: number;
  status: string;
  created_by: number;
  created_at: string;
  paid_at?: string;
  items: SettlementItem[];
  payouts: Payout[];
}

export interface PaymentMethod {
  id: number;
  code: string;
  name: string;
}

export interface SupplierSummary {
  supplier_id: number;
  supplier_name?: string;
  arrangement_id: number;
  store_id: number;
  available_qty: number;
  pending_return_qty: number;
  unsettled_value: number;
  unsettled_qty: number;
}

export interface CreateArrangementPayload {
  supplier_id: number;
  store_id?: number;
}

export interface SetTermsPayload {
  product_id: number;
  price: number;
  store_share_type: string;
  store_share_value: number;
}

export interface ReceiptItemPayload {
  product_id: number;
  accepted_qty: number;
  notes?: string;
}

export interface ReceiptPayload {
  arrangement_id: number;
  notes?: string;
  items: ReceiptItemPayload[];
}

export interface PendingReturnPayload {
  product_id: number;
  qty: number;
  reason: string;
  notes?: string;
}

export interface ReturnItemPayload {
  product_id: number;
  qty: number;
  reason: string;
  pending_return_id?: number;
  notes?: string;
}

export interface ReturnPayload {
  arrangement_id: number;
  notes?: string;
  items: ReturnItemPayload[];
}

export interface CreateSettlementPayload {
  supplier_id: number;
}

export interface CreatePayoutPayload {
  payment_method_id: number;
  amount: number;
  reference_number?: string;
  notes?: string;
}

export const ARRANGEMENT_STATUS_ACTIVE = 'active';
export const ARRANGEMENT_STATUS_ENDED = 'ended';

export const SHARE_TYPE_PERCENTAGE = 'percentage';
export const SHARE_TYPE_FIXED_AMOUNT = 'fixed_amount';

export const PENDING_RETURN_OPEN = 'open';
export const PENDING_RETURN_RETURNED = 'returned';

export const SETTLEMENT_PENDING_PAYMENT = 'pending_payment';
export const SETTLEMENT_PAID = 'paid';

export const RETURN_REASON_DAMAGED = 'damaged';
export const RETURN_REASON_EXPIRED = 'expired';
export const RETURN_REASON_CUSTOMER_RETURN = 'customer_return';
export const RETURN_REASON_OTHER = 'other';

export const ARRANGEMENT_STATUS_LABELS: Record<string, keyof Labels> = {
  [ARRANGEMENT_STATUS_ACTIVE]: 'arrangementStatusActive',
  [ARRANGEMENT_STATUS_ENDED]: 'arrangementStatusEnded',
};

export const SHARE_TYPE_LABELS: Record<string, keyof Labels> = {
  [SHARE_TYPE_PERCENTAGE]: 'shareTypePercentage',
  [SHARE_TYPE_FIXED_AMOUNT]: 'shareTypeFixedAmount',
};

export const PENDING_RETURN_STATUS_LABELS: Record<string, keyof Labels> = {
  [PENDING_RETURN_OPEN]: 'pendingReturnStatusOpen',
  [PENDING_RETURN_RETURNED]: 'pendingReturnStatusReturned',
};

export const SETTLEMENT_STATUS_LABELS: Record<string, keyof Labels> = {
  [SETTLEMENT_PENDING_PAYMENT]: 'settlementStatusPendingPayment',
  [SETTLEMENT_PAID]: 'settlementStatusPaid',
};

export const RETURN_REASON_LABELS: Record<string, keyof Labels> = {
  [RETURN_REASON_DAMAGED]: 'returnReasonDamaged',
  [RETURN_REASON_EXPIRED]: 'returnReasonExpired',
  [RETURN_REASON_CUSTOMER_RETURN]: 'returnReasonCustomerReturn',
  [RETURN_REASON_OTHER]: 'returnReasonOther',
};

export const RETURN_REASONS = [
  RETURN_REASON_DAMAGED,
  RETURN_REASON_EXPIRED,
  RETURN_REASON_CUSTOMER_RETURN,
  RETURN_REASON_OTHER,
];