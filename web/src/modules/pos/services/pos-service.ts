import apiClient from '$shared/api/http-client';
import type { PosProduct, CheckoutPayload } from '../types';
import { getTodayInJakarta } from '$shared/utils/jakartaTime';

export interface ProductsResponse {
  data: PosProduct[];
  total: number;
}

export async function getPosProducts(limit: number, offset: number, search: string): Promise<ProductsResponse> {
  const r = await apiClient.get(`/products?limit=${limit}&offset=${offset}&search=${search}&status=active`);
  return { data: r.data.data || [], total: r.data.total || 0 };
}

export async function getCustomers(limit = 200): Promise<{ id: number; name: string; phone?: string; email?: string }[]> {
  const r = await apiClient.get(`/customers?limit=${limit}`);
  return r.data.data || [];
}

export async function searchCustomers(
  query: string,
  limit = 10,
): Promise<{ id: number; name: string; phone?: string; email?: string }[]> {
  const r = await apiClient.get('/customers', { params: { search: query, limit } });
  return r.data.data || [];
}

export async function createSale(payload: CheckoutPayload): Promise<{ data?: unknown }> {
  const r = await apiClient.post('/sales', payload);
  return r;
}

export async function getSaleById(id: number): Promise<unknown> {
  const r = await apiClient.get(`/sales/${id}`);
  return r.data?.data || r.data;
}

export async function getLastSale(): Promise<unknown> {
  const endDate = getTodayInJakarta();
  const startDate = '2025-01-01';
  const r = await apiClient.get(`/sales?limit=1&offset=0&startDate=${startDate}&endDate=${endDate}`);
  const body = r.data;
  const data = body?.data || body;
  if (Array.isArray(data) && data.length > 0) return data[0];
  if (data && (data as Record<string, unknown>).id) return data;
  return null;
}
