import apiClient from '$shared/api/http-client';
import type { Shift, ShiftFilters, CashMovement, ShiftReportData } from '../types';

export async function openShift(storeId: number | null, openingBalance: number): Promise<Shift> {
  const res = await apiClient.post('/shifts/open', {
    store_id: storeId,
    opening_balance: openingBalance,
  });
  return res.data.data;
}

export async function closeShift(
  shiftId: number,
  closingBalance: number,
  notes: string | null
): Promise<Shift> {
  const res = await apiClient.post(`/shifts/${shiftId}/close`, {
    closing_balance: closingBalance,
    notes,
  });
  return res.data.data;
}

export async function getActiveShift(): Promise<Shift | null> {
  try {
    const res = await apiClient.get('/shifts/active');
    return res.data.data;
  } catch {
    return null;
  }
}

export async function listShifts(filters: ShiftFilters, signal?: AbortSignal): Promise<{ data: Shift[]; total: number }> {
  const params = new URLSearchParams({
    limit: filters.limit.toString(),
    offset: filters.offset.toString(),
    sort_by: filters.sortBy,
    sort_dir: filters.sortDir,
  });
  if (filters.status) params.set('status', filters.status);
  if (filters.userId) params.set('user_id', filters.userId.toString());
  if (filters.needsReview != null) params.set('needs_review', filters.needsReview.toString());
  if (filters.discrepancy) params.set('discrepancy', filters.discrepancy);

  const res = await apiClient.get(`/shifts?${params.toString()}`, { signal });
  return { data: res.data.data || [], total: res.data.total || 0 };
}

export async function exportShifts(filters: ShiftFilters, format: 'csv' | 'xlsx'): Promise<Blob> {
  const params = new URLSearchParams({ format });
  if (filters.status) params.set('status', filters.status);
  if (filters.userId) params.set('user_id', filters.userId.toString());
  if (filters.needsReview != null) params.set('needs_review', filters.needsReview.toString());
  if (filters.discrepancy) params.set('discrepancy', filters.discrepancy);

  const res = await apiClient.get(`/shifts/export?${params.toString()}`, {
    responseType: 'blob',
  });
  return res.data;
}

export async function getShiftById(id: number): Promise<Shift | null> {
  try {
    const res = await apiClient.get(`/shifts/${id}`);
    return res.data.data;
  } catch {
    return null;
  }
}

export async function reviewShift(shiftId: number): Promise<Shift> {
  const res = await apiClient.post(`/shifts/${shiftId}/review`);
  return res.data.data;
}

export async function auditShift(shiftId: number, actualBalance: number): Promise<{
  shift: Shift;
  expected_cash: number;
  actual_balance: number;
  off_by: number;
  flagged_for_review: boolean;
}> {
  const res = await apiClient.post(`/shifts/${shiftId}/audit`, { actual_balance: actualBalance });
  return res.data.data;
}

export async function recordCashMovement(
  shiftId: number,
  type: 'cash_drop' | 'paid_in' | 'paid_out',
  amount: number,
  description: string | null
): Promise<CashMovement> {
  const res = await apiClient.post(`/shifts/${shiftId}/cash-movements`, {
    type,
    amount,
    description,
  });
  return res.data.data;
}

export async function listCashMovements(shiftId: number): Promise<CashMovement[]> {
  const res = await apiClient.get(`/shifts/${shiftId}/cash-movements`);
  return res.data.data || [];
}

export async function getShiftReport(shiftId: number): Promise<ShiftReportData> {
  const res = await apiClient.get(`/shifts/${shiftId}/report`);
  return res.data.data;
}
