import { apiFetch } from '$shared/api/http-client';
import { getAuthToken } from '$modules/auth';
import type { Sale, SaleFilters } from '../types';

const SLIDER_MAX_BOUND = 50000000;

export async function getSalesHistory(filters: SaleFilters, signal?: AbortSignal): Promise<{ data: Sale[]; total: number }> {
  const params = new URLSearchParams({
    startDate: filters.startDate,
    endDate: filters.endDate,
    limit: filters.limit.toString(),
    offset: filters.offset.toString(),
    search: filters.search || '',
    sortBy: filters.sortBy || 'created_at',
    sortDir: filters.sortDir || 'DESC',
  });
  if (filters.paymentMethods && filters.paymentMethods.length > 0) {
    params.set('paymentMethods', filters.paymentMethods.join(','));
  }
  if (filters.minTotal !== undefined && filters.minTotal > 0) {
    params.set('minTotal', filters.minTotal.toString());
  }
  if (filters.maxTotal !== undefined && filters.maxTotal < SLIDER_MAX_BOUND) {
    params.set('maxTotal', filters.maxTotal.toString());
  }
  const res = await apiFetch(`/api/sales?${params.toString()}`, { signal });
  if (res.ok) {
    const data = await res.json();
    return { data: data.data || [], total: data.total || 0 };
  }
  return { data: [], total: 0 };
}

export async function getPaymentMethods(signal?: AbortSignal): Promise<{ code: string; name: string }[]> {
  try {
    const res = await apiFetch('/api/payment-methods', { signal });
    if (res.ok) {
      const data = await res.json();
      return (data.data || data || []).filter((m: any) => m.is_active !== false);
    }
    return [];
  } catch {
    return [];
  }
}

export async function getSaleById(id: number): Promise<Sale | null> {
  try {
    const res = await apiFetch(`/api/sales/${id}`);
    if (res.ok) {
      return await res.json();
    }
    return null;
  } catch {
    return null;
  }
}

export async function exportSales(format: 'csv' | 'xlsx', filters: SaleFilters): Promise<Blob | null> {
  const token = getAuthToken();
  if (!token) return null;

  const params = new URLSearchParams({
    format,
    startDate: filters.startDate,
    endDate: filters.endDate,
    search: filters.search || '',
    sortBy: filters.sortBy || 'created_at',
    sortDir: filters.sortDir || 'DESC',
  });
  if (filters.paymentMethods && filters.paymentMethods.length > 0) {
    params.set('paymentMethods', filters.paymentMethods.join(','));
  }
  if (filters.minTotal !== undefined && filters.minTotal > 0) {
    params.set('minTotal', filters.minTotal.toString());
  }
  if (filters.maxTotal !== undefined && filters.maxTotal < SLIDER_MAX_BOUND) {
    params.set('maxTotal', filters.maxTotal.toString());
  }

  const res = await fetch(`/api/sales/export?${params.toString()}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) return null;
  return res.blob();
}
