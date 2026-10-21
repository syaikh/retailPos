// Sales Return (Retur Penjualan) — PRD docs/prd/PRD-Sales-Return.md,
// Phase 2 frontend: the cashier's return modal, the manager's approval queue
// and review modal, plus the service layer over internal/salereturn's API.
export { default as ReturnModal } from "./components/ReturnModal.svelte";
export { default as ApprovalModal } from "./components/ApprovalModal.svelte";
export { default as PendingReturnsModal } from "./components/PendingReturnsModal.svelte";
export {
  ineligibleText,
  returnStatusLabel,
  returnStatusVariant,
} from "./lib/labels";
export { buildReturnReceipt } from "./lib/receipt";
export {
  getReturnableSale,
  processReturn,
  listReturnsBySale,
  getReturnById,
  listPendingApprovals,
  approveReturn,
  rejectReturn,
} from "./services/salereturn-service";
export type {
  IneligibleReason,
  ProcessReturnRequest,
  ReturnApproval,
  ReturnableItem,
  ReturnableSale,
  ReturnItemRequest,
  ReturnLineSelection,
  SaleReturn,
  SaleReturnItem,
  SaleReturnPayment,
} from "./types";
