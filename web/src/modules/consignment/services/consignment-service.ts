import apiClient from '$shared/api/http-client';
import type {
  Arrangement,
  ConsignmentReturn,
  ConsignmentSupplierRef,
  CreateArrangementPayload,
  CreatePayoutPayload,
  CreateSettlementPayload,
  PaymentMethod,
  Payout,
  PendingReturn,
  PendingReturnPayload,
  Receipt,
  ReceiptPayload,
  ReturnPayload,
  Settlement,
  SetTermsPayload,
  StockRow,
  Term,
} from '../types';

export async function listConsignmentSuppliers(): Promise<ConsignmentSupplierRef[]> {
  const res = await apiClient.get('/consignment/suppliers');
  return res.data.data || [];
}

export interface ArrangementListParams {
  limit?: number;
  offset?: number;
  search?: string;
  status?: string;
}

export async function listArrangements(params: ArrangementListParams = {}): Promise<{ data: Arrangement[]; total: number }> {
  const res = await apiClient.get('/consignment/arrangements', { params });
  return { data: res.data.data || [], total: res.data.total || 0 };
}

export async function getArrangement(id: number): Promise<Arrangement> {
  const res = await apiClient.get(`/consignment/arrangements/${id}`);
  return res.data.data;
}

export async function createArrangement(payload: CreateArrangementPayload): Promise<Arrangement> {
  const res = await apiClient.post('/consignment/arrangements', payload);
  return res.data.data;
}

export async function setTerms(arrangementId: number, terms: SetTermsPayload[]): Promise<Term[]> {
  const res = await apiClient.put(`/consignment/arrangements/${arrangementId}/terms`, terms);
  return res.data.data || [];
}

export async function listReceipts(supplierId: number): Promise<Receipt[]> {
  const res = await apiClient.get(`/consignment/receipts?supplier_id=${supplierId}`);
  return res.data.data || [];
}

export async function getReceipt(id: number): Promise<Receipt> {
  const res = await apiClient.get(`/consignment/receipts/${id}`);
  return res.data.data;
}

export async function createReceipt(payload: ReceiptPayload): Promise<Receipt> {
  const res = await apiClient.post('/consignment/receipts', payload);
  return res.data.data;
}

export async function listStock(supplierId: number): Promise<StockRow[]> {
  const res = await apiClient.get(`/consignment/stock?supplier_id=${supplierId}`);
  return res.data.data || [];
}

export async function listPendingReturns(supplierId: number): Promise<PendingReturn[]> {
  const res = await apiClient.get(`/consignment/pending-returns?supplier_id=${supplierId}`);
  return res.data.data || [];
}

export async function createPendingReturn(payload: PendingReturnPayload): Promise<PendingReturn> {
  const res = await apiClient.post('/consignment/pending-returns', payload);
  return res.data.data;
}

export async function listReturns(supplierId: number): Promise<ConsignmentReturn[]> {
  const res = await apiClient.get(`/consignment/returns?supplier_id=${supplierId}`);
  return res.data.data || [];
}

export async function getReturn(id: number): Promise<ConsignmentReturn> {
  const res = await apiClient.get(`/consignment/returns/${id}`);
  return res.data.data;
}

export async function createReturn(payload: ReturnPayload): Promise<ConsignmentReturn> {
  const res = await apiClient.post('/consignment/returns', payload);
  return res.data.data;
}

export async function getSettlementPreview(supplierId: number): Promise<Settlement> {
  const res = await apiClient.get(`/consignment/settlements/preview?supplier_id=${supplierId}`);
  return res.data.data;
}

export async function createSettlement(payload: CreateSettlementPayload): Promise<Settlement> {
  const res = await apiClient.post('/consignment/settlements', payload);
  return res.data.data;
}

export async function listSettlements(supplierId: number): Promise<Settlement[]> {
  const res = await apiClient.get(`/consignment/settlements?supplier_id=${supplierId}`);
  return res.data.data || [];
}

export async function getSettlement(id: number): Promise<Settlement> {
  const res = await apiClient.get(`/consignment/settlements/${id}`);
  return res.data.data;
}

export async function listPaymentMethods(): Promise<PaymentMethod[]> {
  const res = await apiClient.get('/consignment/payment-methods');
  return res.data.data || [];
}

export async function createPayout(settlementId: number, payload: CreatePayoutPayload): Promise<Payout> {
  const res = await apiClient.post(`/consignment/settlements/${settlementId}/payouts`, payload);
  return res.data.data;
}

export async function getSupplierSummary(supplierId: number): Promise<{ available_qty: number; pending_return_qty: number; unsettled_value: number; unsettled_qty: number }> {
  const stock = await listStock(supplierId);
  let unsettledValue = 0;
  let unsettledQty = 0;
  try {
    const preview = await getSettlementPreview(supplierId);
    unsettledValue = preview.total_payable;
    unsettledQty = (preview.items || []).reduce((sum, i) => sum + i.quantity, 0);
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