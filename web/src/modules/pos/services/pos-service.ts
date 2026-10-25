import apiClient from "$shared/api/http-client";
import type { PaymentAllocation, CartSession } from "../types";

function unwrapCart(r: { data?: { data?: unknown } }): CartSession {
  return (r.data?.data || r.data) as CartSession;
}

export async function createCart(payload?: {
  store_id?: number;
  shift_id?: number;
  customer_id?: number;
}): Promise<CartSession> {
  const r = await apiClient.post("/pos/cart", payload || {});
  return unwrapCart(r);
}

export async function getOpenCart(): Promise<CartSession> {
  const r = await apiClient.get("/pos/cart");
  return unwrapCart(r);
}

export async function getHeldCarts(): Promise<CartSession[]> {
  const r = await apiClient.get("/pos/cart/held");
  const data = r.data?.data || r.data || [];
  return Array.isArray(data) ? data : [];
}

export async function getCart(id: number): Promise<CartSession> {
  const r = await apiClient.get(`/pos/cart/${id}`);
  return unwrapCart(r);
}

export async function addCartItem(
  cartId: number,
  item: {
    product_id: number;
    quantity: number;
    customer_group_id?: number;
    store_id?: number;
    shift_id?: number;
    customer_id?: number;
  },
): Promise<CartSession> {
  const r = await apiClient.post("/pos/cart/items", item);
  return unwrapCart(r);
}

export async function updateCartItemQuantity(
  cartId: number,
  itemId: number,
  quantity: number,
): Promise<CartSession> {
  const r = await apiClient.patch(`/pos/cart/items/${itemId}`, { quantity });
  return unwrapCart(r);
}

export async function removeCartItem(
  cartId: number,
  itemId: number,
): Promise<CartSession> {
  const r = await apiClient.delete(`/pos/cart/items/${itemId}`);
  return unwrapCart(r);
}

export async function updateCartCustomer(
  cartId: number,
  customerId: number | null,
): Promise<CartSession> {
  const r = await apiClient.patch(`/pos/cart/${cartId}/customer`, {
    customer_id: customerId,
  });
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

export async function cancelCart(cartId: number): Promise<CartSession> {
  const r = await apiClient.post(`/pos/cart/${cartId}/cancel`);
  return unwrapCart(r);
}

export async function checkoutCart(
  cartId: number,
  payments: PaymentAllocation[],
): Promise<unknown> {
  const r = await apiClient.post(`/pos/cart/${cartId}/checkout`, { payments });
  return r.data?.data || r.data;
}
