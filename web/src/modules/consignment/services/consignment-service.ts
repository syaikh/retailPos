import apiClient from "$shared/api/http-client";
import type {
  Arrangement,
  ConsignmentReturn,
  ConsignmentSupplierRef,
  CreateArrangementPayload,
  CreatePayoutPayload,
  CreateSettlementPayload,
  EditReceiptPayload,
  PaymentMethod,
  Payout,
  PendingReturn,
  PendingReturnPayload,
  Receipt,
  ReceiptPayload,
  ReturnItemPayload,
  ReturnPayload,
  Settlement,
  SetTermsPayload,
  StockRow,
  Term,
} from "../types";

export async function listConsignmentSuppliers(): Promise<
  ConsignmentSupplierRef[]
> {
  const res = await apiClient.get("/consignment/suppliers");
  return res.data.data || [];
}

export interface ArrangementListParams {
  limit?: number;
  offset?: number;
  search?: string;
  status?: string;
}

export async function listArrangements(
  params: ArrangementListParams = {},
): Promise<{ data: Arrangement[]; total: number }> {
  const res = await apiClient.get("/consignment/arrangements", { params });
  return { data: res.data.data || [], total: res.data.total || 0 };
}

export async function getArrangement(id: number): Promise<Arrangement> {
  const res = await apiClient.get(`/consignment/arrangements/${id}`);
  return res.data.data;
}

export async function createArrangement(
  payload: CreateArrangementPayload,
): Promise<Arrangement> {
  const res = await apiClient.post("/consignment/arrangements", payload);
  return res.data.data;
}

export async function endArrangement(id: number): Promise<Arrangement> {
  const res = await apiClient.post(`/consignment/arrangements/${id}/end`);
  return res.data.data;
}

export async function setTerms(
  arrangementId: number,
  terms: SetTermsPayload[],
): Promise<Term[]> {
  const res = await apiClient.put(
    `/consignment/arrangements/${arrangementId}/terms`,
    terms,
  );
  return res.data.data || [];
}

export async function addTerm(
  arrangementId: number,
  payload: SetTermsPayload,
): Promise<Term> {
  const res = await apiClient.post(
    `/consignment/arrangements/${arrangementId}/terms`,
    payload,
  );
  return res.data.data;
}

export async function removeTerm(
  arrangementId: number,
  productId: number,
): Promise<void> {
  await apiClient.delete(
    `/consignment/arrangements/${arrangementId}/terms/${productId}`,
  );
}

export interface AddTermProductOption {
  id: number;
  sku: string;
  name: string;
}

export interface SearchProductResult {
  products: AddTermProductOption[];
  exactMatch: boolean;
}

export async function listAddTermProductOptions(
  arrangementId: number,
): Promise<AddTermProductOption[]> {
  const res = await apiClient.get(
    `/consignment/arrangements/${arrangementId}/available-products`,
  );
  return res.data.data || [];
}

export async function searchAvailableProducts(
  arrangementId: number,
  search: string,
): Promise<SearchProductResult> {
  const res = await apiClient.get(
    `/consignment/arrangements/${arrangementId}/available-products`,
    { params: { search } },
  );
  return {
    products: res.data.data || [],
    exactMatch: res.data.exact_match ?? false,
  };
}

export async function listReceipts(
  supplierId: number,
  productId?: number,
): Promise<Receipt[]> {
  const params = new URLSearchParams({ supplier_id: String(supplierId) });
  if (productId) params.set("product_id", String(productId));
  const res = await apiClient.get(
    `/consignment/receipts?${params.toString()}`,
  );
  return res.data.data || [];
}

export async function getReceipt(id: number): Promise<Receipt> {
  const res = await apiClient.get(`/consignment/receipts/${id}`);
  return res.data.data;
}

export async function createReceipt(payload: ReceiptPayload): Promise<Receipt> {
  const res = await apiClient.post("/consignment/receipts", payload);
  return res.data.data;
}

export async function editReceipt(
  receiptId: number,
  payload: EditReceiptPayload,
): Promise<Receipt> {
  const res = await apiClient.put(
    `/consignment/receipts/${receiptId}`,
    payload,
  );
  return res.data.data;
}

export async function listStock(supplierId: number): Promise<StockRow[]> {
  const res = await apiClient.get(
    `/consignment/stock?supplier_id=${supplierId}`,
  );
  return res.data.data || [];
}

export async function listPendingReturns(
  supplierId: number,
): Promise<PendingReturn[]> {
  const res = await apiClient.get(
    `/consignment/pending-returns?supplier_id=${supplierId}`,
  );
  return res.data.data || [];
}

export async function createPendingReturn(
  payload: PendingReturnPayload,
): Promise<PendingReturn> {
  const res = await apiClient.post("/consignment/pending-returns", payload);
  return res.data.data;
}

export async function listReturns(
  supplierId: number,
): Promise<ConsignmentReturn[]> {
  const res = await apiClient.get(
    `/consignment/returns?supplier_id=${supplierId}`,
  );
  return res.data.data || [];
}

export async function getReturn(id: number): Promise<ConsignmentReturn> {
  const res = await apiClient.get(`/consignment/returns/${id}`);
  return res.data.data;
}

export async function createReturn(
  payload: ReturnPayload,
): Promise<ConsignmentReturn> {
  const res = await apiClient.post("/consignment/returns", payload);
  return res.data.data;
}

export async function bulkReturnAllStock(
  arrangementId: number,
  supplierId: number,
): Promise<ConsignmentReturn> {
  const [stock, pendingReturns] = await Promise.all([
    listStock(supplierId),
    listPendingReturns(supplierId),
  ]);

  const openPending = pendingReturns.filter(
    (pr) => pr.status === "open" && pr.arrangement_id === arrangementId,
  );

  const items: ReturnItemPayload[] = [];

  for (const row of stock) {
    if (row.arrangement_id !== arrangementId) continue;
    if (row.available_qty <= 0 && row.pending_return_qty <= 0) continue;

    const productPending = openPending.filter(
      (pr) => pr.product_id === row.product_id,
    );

    if (productPending.length > 0) {
      for (const pr of productPending) {
        items.push({
          product_id: row.product_id,
          qty: pr.qty,
          reason: "termination",
          pending_return_id: pr.id,
        });
      }
    }

    if (row.available_qty > 0) {
      items.push({
        product_id: row.product_id,
        qty: row.available_qty,
        reason: "termination",
      });
    }
  }

  if (items.length === 0) {
    throw new Error("No stock to return");
  }

  return createReturn({
    arrangement_id: arrangementId,
    notes: "Bulk return — ending arrangement",
    items,
  });
}

export async function getSettlementPreview(
  supplierId: number,
): Promise<Settlement> {
  const res = await apiClient.get(
    `/consignment/settlements/preview?supplier_id=${supplierId}`,
  );
  return res.data.data;
}

export async function createSettlement(
  payload: CreateSettlementPayload,
): Promise<Settlement> {
  const res = await apiClient.post("/consignment/settlements", payload);
  return res.data.data;
}

export async function listSettlements(
  supplierId?: number,
  status?: string,
): Promise<Settlement[]> {
  const params = new URLSearchParams();
  if (supplierId) params.set("supplier_id", String(supplierId));
  if (status) params.set("status", status);
  const qs = params.toString();
  const res = await apiClient.get(
    `/consignment/settlements${qs ? `?${qs}` : ""}`,
  );
  return res.data.data || [];
}

export async function getSettlement(id: number): Promise<Settlement> {
  const res = await apiClient.get(`/consignment/settlements/${id}`);
  return res.data.data;
}

export async function listPaymentMethods(): Promise<PaymentMethod[]> {
  const res = await apiClient.get("/consignment/payment-methods");
  return res.data.data || [];
}

export async function createPayout(
  settlementId: number,
  payload: CreatePayoutPayload,
): Promise<Payout> {
  const res = await apiClient.post(
    `/consignment/settlements/${settlementId}/payouts`,
    payload,
  );
  return res.data.data;
}

export async function getSupplierSummary(supplierId: number): Promise<{
  available_qty: number;
  pending_return_qty: number;
  unsettled_value: number;
  unsettled_qty: number;
}> {
  const stock = await listStock(supplierId);
  let unsettledValue = 0;
  let unsettledQty = 0;
  try {
    const preview = await getSettlementPreview(supplierId);
    unsettledValue = preview.total_payable;
    unsettledQty = (preview.items || []).reduce(
      (sum, i) => sum + i.quantity,
      0,
    );
  } catch {
    // No unsettled sales yet — preview 422s with an empty settlement.
  }
  const available = stock.reduce((sum, r) => sum + r.available_qty, 0);
  const pendingReturn = stock.reduce((sum, r) => sum + r.pending_return_qty, 0);
  return {
    available_qty: available,
    pending_return_qty: pendingReturn,
    unsettled_value: unsettledValue,
    unsettled_qty: unsettledQty,
  };
}
