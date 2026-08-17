import apiClient from '$shared/api/http-client';
import type { PosProduct, PaymentAllocation, CartSession, CartItem } from '../types';
import { getTodayInJakarta, getDateNDaysAgoInJakarta } from '$shared/utils/jakartaTime';

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

export async function getSaleById(id: number): Promise<unknown> {
  const r = await apiClient.get(`/sales/${id}`);
  return r.data?.data || r.data;
}

export async function getLastSale(): Promise<unknown> {
  const endDate = getTodayInJakarta();
  const startDate = getDateNDaysAgoInJakarta(7);
  const r = await apiClient.get(`/sales?limit=1&offset=0&startDate=${startDate}&endDate=${endDate}`);
  const body = r.data;
  const data = body?.data || body;
  if (Array.isArray(data) && data.length > 0) return data[0];
  if (data && (data as Record<string, unknown>).id) return data;
  return null;
}

function unwrapCart(r: { data?: { data?: unknown } }): CartSession {
  return (r.data?.data || r.data) as CartSession;
}

export async function createCart(payload?: {
  store_id?: number;
  shift_id?: number;
  customer_id?: number;
}): Promise<CartSession> {
  const r = await apiClient.post('/pos/cart', payload || {});
  return unwrapCart(r);
}

export async function getOpenCart(): Promise<CartSession> {
  const r = await apiClient.get('/pos/cart');
  return unwrapCart(r);
}

export async function getHeldCarts(): Promise<CartSession[]> {
  const r = await apiClient.get('/pos/cart/held');
  const data = r.data?.data || r.data || [];
  return Array.isArray(data) ? data : [];
}

export async function getCart(id: number): Promise<CartSession> {
  const r = await apiClient.get(`/pos/cart/${id}`);
  return unwrapCart(r);
}

export async function addCartItem(cartId: number, item: {
  product_id: number;
  quantity: number;
  customer_group_id?: number;
  store_id?: number;
  shift_id?: number;
  customer_id?: number;
}): Promise<CartSession> {
  const r = await apiClient.post('/pos/cart/items', item);
  return unwrapCart(r);
}

export async function updateCartItemQuantity(cartId: number, itemId: number, quantity: number): Promise<CartSession> {
  const r = await apiClient.patch(`/pos/cart/items/${itemId}`, { quantity });
  return unwrapCart(r);
}

export async function removeCartItem(cartId: number, itemId: number): Promise<CartSession> {
  const r = await apiClient.delete(`/pos/cart/items/${itemId}`);
  return unwrapCart(r);
}

export async function updateCartCustomer(cartId: number, customerId: number | null): Promise<CartSession> {
  const r = await apiClient.patch(`/pos/cart/${cartId}/customer`, { customer_id: customerId });
  return unwrapCart(r);
}

export async function holdCart(cartId: number): Promise<CartSession> {
  const r = await apiClient.post(`/pos/cart/${cartId}/hold`);
  return unwrapCart(r);
}

export async function resumeCart(cartId: number): Promise<CartSession> {
  const r = await apiClient.post(`/pos/cart/${cartId}/resume`);
  return unwrapCart(r);
}

export async function checkoutCart(cartId: number, payments: PaymentAllocation[]): Promise<unknown> {
  const r = await apiClient.post(`/pos/cart/${cartId}/checkout`, { payments });
  return r.data?.data || r.data;
}
