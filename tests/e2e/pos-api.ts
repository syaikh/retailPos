import { API_BASE, getToken, authHeader, TEST_USERS } from './fixtures';

// ============================================================================
// Shared POS/cart/shift API helpers for the cashier scenario specs.
// All calls go through the public REST API — no UI required.
// ============================================================================

export interface AuthCtx {
  token: string;
  headers: Record<string, string>;
}

export async function authAs(request: any, username: string): Promise<AuthCtx> {
  const user = TEST_USERS[username as keyof typeof TEST_USERS];
  const token = await getToken(request, user.username, user.password);
  return { token, headers: authHeader(token) };
}

/**
 * Login as a fresh cashier created through the admin API. Returns the new
 * user's id (tracked by the caller for cleanup) plus an auth context.
 */
export async function createCashier(request: any, admin: AuthCtx, suffix: string): Promise<{ id: number; ctx: AuthCtx }> {
  const username = `e2ecashier${suffix}`.toLowerCase().replace(/[^a-z0-9]/g, '');
  const res = await request.post(`${API_BASE}/api/admin/users`, {
    headers: admin.headers,
    data: {
      username,
      email: `${username}@e2e.test`,
      password: 'admin123',
      role_id: 4,
      is_active: true,
    },
  });
  if (!res.ok()) {
    throw new Error(`createCashier failed (${res.status()}): ${await res.text()}`);
  }
  const body = await res.json();
  const created = body.data ?? body.user ?? body;
  const id = created.id;
  const token = await getToken(request, username, 'admin123');
  return { id, ctx: { token, headers: authHeader(token) } };
}

/** Returns the caller's active shift id, opening one if necessary. */
export async function ensureOpenShift(request: any, ctx: AuthCtx): Promise<number> {
  const activeRes = await request.get(`${API_BASE}/api/shifts/active`, { headers: ctx.headers });
  if (activeRes.ok()) {
    const body = await activeRes.json();
    if (body.data?.id) return body.data.id;
  }
  const openRes = await request.post(`${API_BASE}/api/shifts/open`, {
    headers: ctx.headers,
    data: { opening_balance: 500000 },
  });
  if (!openRes.ok() && openRes.status() !== 409) {
    throw new Error(`open shift failed (${openRes.status()}): ${await openRes.text()}`);
  }
  const after = await request.get(`${API_BASE}/api/shifts/active`, { headers: ctx.headers });
  const body = await after.json();
  if (!body.data?.id) throw new Error('no active shift after open');
  return body.data.id;
}

export async function closeShift(request: any, ctx: AuthCtx, shiftId: number, closingBalance = 0): Promise<void> {
  const res = await request.post(`${API_BASE}/api/shifts/${shiftId}/close`, {
    headers: ctx.headers,
    data: { closing_balance: closingBalance },
  });
  if (!res.ok()) {
    throw new Error(`close shift failed (${res.status()}): ${await res.text()}`);
  }
}

/** Returns the caller's open cart, creating one when none exists. */
export async function getOrCreateOpenCart(request: any, ctx: AuthCtx, shiftId?: number): Promise<any> {
  let res = await request.get(`${API_BASE}/api/pos/cart`, { headers: ctx.headers });
  if (res.ok()) {
    const body = await res.json();
    if (body.data?.id) return body.data;
  }
  res = await request.post(`${API_BASE}/api/pos/cart`, {
    headers: ctx.headers,
    data: shiftId ? { shift_id: shiftId } : {},
  });
  if (!res.ok()) {
    throw new Error(`create cart failed (${res.status()}): ${await res.text()}`);
  }
  return (await res.json()).data;
}

/**
 * Parks any open cart (held carts are tracked for cleanup by the caller) and
 * returns a brand-new empty open cart — deterministic state per test.
 */
export async function startFreshCart(request: any, ctx: AuthCtx, shiftId?: number): Promise<any> {
  const open = await request.get(`${API_BASE}/api/pos/cart`, { headers: ctx.headers });
  if (open.ok()) {
    const body = await open.json();
    if (body.data?.id && body.data.status === 'open') {
      const hold = await request.post(`${API_BASE}/api/pos/cart/${body.data.id}/hold`, { headers: ctx.headers });
      if (!hold.ok()) {
        throw new Error(`hold leftover cart failed (${hold.status()}): ${await hold.text()}`);
      }
    }
  }
  return getOrCreateOpenCart(request, ctx, shiftId);
}

export async function addCartItem(
  request: any,
  ctx: AuthCtx,
  productId: number,
  quantity = 1,
  extra: Record<string, unknown> = {}
): Promise<{ status: number; body: any }> {
  const res = await request.post(`${API_BASE}/api/pos/cart/items`, {
    headers: ctx.headers,
    data: { product_id: productId, quantity, ...extra },
  });
  return { status: res.status(), body: await res.json().catch(() => ({})) };
}

export async function holdCart(request: any, ctx: AuthCtx, cartId: number): Promise<void> {
  const res = await request.post(`${API_BASE}/api/pos/cart/${cartId}/hold`, { headers: ctx.headers });
  if (!res.ok()) throw new Error(`hold failed (${res.status()}): ${await res.text()}`);
}

export interface CheckoutResult {
  status: number;
  body: any;
}

export async function checkoutCart(
  request: any,
  ctx: AuthCtx,
  cartId: number,
  payments?: Array<{ payment_method_code: string; amount: number; reference_number?: string }>
): Promise<CheckoutResult> {
  const res = await request.post(`${API_BASE}/api/pos/cart/${cartId}/checkout`, {
    headers: ctx.headers,
    data: payments ? { payments } : {},
  });
  return { status: res.status(), body: await res.json().catch(() => ({})) };
}

/** Cash payment set covering exactly `total`. */
export function cashPayments(total: number): Array<{ payment_method_code: string; amount: number }> {
  return [{ payment_method_code: 'CASH', amount: total }];
}

/** Finds an active product matching `search` and boosts its stock. */
export async function findProductWithStock(request: any, admin: AuthCtx, search: string, stockBoost = 500): Promise<{ id: number; price: number }> {
  const res = await request.get(`${API_BASE}/api/products?search=${encodeURIComponent(search)}&status=active&limit=1`, {
    headers: admin.headers,
  });
  const body = await res.json();
  const product = body.data?.[0];
  if (!product) throw new Error(`no product found for search "${search}"`);
  await request.post(`${API_BASE}/api/inventory/adjust`, {
    headers: admin.headers,
    data: { product_id: product.id, quantity_change: stockBoost, notes: 'E2E cashier scenario stock boost' },
  });
  return { id: product.id, price: product.price };
}
