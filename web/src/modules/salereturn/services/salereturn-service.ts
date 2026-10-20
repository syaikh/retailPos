import apiClient from "$shared/api/http-client";
import type {
  ProcessReturnRequest,
  ReturnableSale,
  SaleReturn,
} from "../types";

// Every list endpoint is paginated (shared.NewPaginatedResponse) with a
// server cap of 100 rows per page; the approval queue and a sale's returns
// are small, so one max-size page is enough for the Phase 1 UI.
const LIST_LIMIT = 100;

/** GET /sales/:id/returnable — pre-flight snapshot for the return modal. */
export async function getReturnableSale(
  saleId: number,
): Promise<ReturnableSale | null> {
  const res = await apiClient.get(`/sales/${saleId}/returnable`);
  return res.data?.data ?? null;
}

/** POST /sales/:id/return — create/process a return (sale.return). */
export async function processReturn(
  saleId: number,
  request: ProcessReturnRequest,
): Promise<SaleReturn> {
  const res = await apiClient.post(`/sales/${saleId}/return`, request);
  return res.data?.data;
}

/** GET /sales/:id/returns — the returns already recorded against a sale. */
export async function listReturnsBySale(saleId: number): Promise<SaleReturn[]> {
  const res = await apiClient.get(`/sales/${saleId}/returns`, {
    params: { limit: LIST_LIMIT },
  });
  return res.data?.data || [];
}

/** GET /sales/returns/:returnId — one return's full detail. */
export async function getReturnById(
  returnId: number,
): Promise<SaleReturn | null> {
  const res = await apiClient.get(`/sales/returns/${returnId}`);
  return res.data?.data ?? null;
}

/** GET /sales/returns/pending — approval queue, oldest first
 * (sale.return.approve). */
export async function listPendingApprovals(): Promise<SaleReturn[]> {
  const res = await apiClient.get("/sales/returns/pending", {
    params: { limit: LIST_LIMIT },
  });
  return res.data?.data || [];
}

/** POST /sales/returns/:returnId/approve — complete a pending return. */
export async function approveReturn(
  returnId: number,
  note?: string,
): Promise<SaleReturn> {
  const res = await apiClient.post(`/sales/returns/${returnId}/approve`, {
    note: note || null,
  });
  return res.data?.data;
}

/** POST /sales/returns/:returnId/reject — cancel a pending return; the
 * rejection reason is mandatory server-side. */
export async function rejectReturn(
  returnId: number,
  reason: string,
): Promise<SaleReturn> {
  const res = await apiClient.post(`/sales/returns/${returnId}/reject`, {
    reason,
  });
  return res.data?.data;
}
