import apiClient from '$shared/api/http-client';
import type { PosProduct, CheckoutPayload, PaymentAllocation } from '../types';
import { getTodayInJakarta } from '$shared/utils/jakartaTime';

export interface ProductsResponse {
  data: PosProduct[];
  total: number;
}

export interface ParkedSale {
  id: number;
  invoice_number: string;
  status: string;
  total_amount: number;
  items?: ParkedSaleItem[];
  created_at: string;
}

export interface ParkedSaleItem {
  product_id: number;
  name?: string;
  quantity: number;
  unit_price: number;
  subtotal: number;
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

export async function createSale(payload: CheckoutPayload & { parked_sale_id?: number }): Promise<{ data?: unknown }> {
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

export async function parkSale(payload: {
  items: { product_id: number; quantity: number; subtotal: number }[];
  payment_method?: string;
  recalled_sale_id?: number | null;
}): Promise<ParkedSale> {
  const r = await apiClient.post('/sales/parked', payload);
  return r.data?.data || r.data;
}

export async function listParkedSales(): Promise<ParkedSale[]> {
  const r = await apiClient.get('/sales/parked');
  return r.data?.data || [];
}

export async function getParkedSaleById(id: number): Promise<ParkedSale> {
  const r = await apiClient.get(`/sales/parked/${id}`);
  return r.data?.data || r.data;
}

export async function recallParkedSale(id: number): Promise<ParkedSale> {
  const r = await apiClient.post(`/sales/parked/${id}/recall`);
  return r.data?.data || r.data;
}

export async function cancelParkedSale(id: number): Promise<void> {
  await apiClient.delete(`/sales/parked/${id}`);
}
