import { API_BASE, getToken, authHeader, TEST_USERS } from './fixtures';
import { ApiDriver } from './api-driver';

// ============================================================================
// Shared POS/cart/shift API helpers for the cashier scenario specs.
// Internally these run on the API driver (same layer as `apiAs`), so behaviour
// is tested headlessly. Each helper accepts either an AuthCtx (legacy) or an
// ApiDriver, and always uses the request that was passed in — never a cached
// request captured outside the calling test.
// ============================================================================

export interface AuthCtx {
  token: string;
  headers: Record<string, string>;
}

/** Normalise an AuthCtx or ApiDriver into an ApiDriver bound to `request`. */
function asApi(request: any, ctxOrApi: AuthCtx | ApiDriver): ApiDriver {
  return ctxOrApi instanceof ApiDriver ? ctxOrApi : new ApiDriver(request, ctxOrApi.token);
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
export async function createCashier(request: any, admin: AuthCtx | ApiDriver, suffix: string): Promise<{ id: number; ctx: AuthCtx }> {
  const api = asApi(request, admin);
  const username = `e2ecashier${suffix}`.toLowerCase().replace(/[^a-z0-9]/g, '');
  const res = await api.post('/api/admin/users', {
    username,
    email: `${username}@e2e.test`,
    password: 'admin123',
    role_id: 4,
    is_active: true,
  });
  if (!res.ok) {
    throw new Error(`createCashier failed (${res.status}): ${JSON.stringify(res.body)}`);
  }
  const created = (res.body?.data ?? res.body?.user ?? res.body);
  const id = created.id;
  const token = await getToken(request, username, 'admin123');
  return { id, ctx: { token, headers: authHeader(token) } };
}

/** Returns the caller's active shift id, opening one if necessary. */
export async function ensureOpenShift(request: any, ctx: AuthCtx | ApiDriver): Promise<number> {
  const api = asApi(request, ctx);
  const activeRes = await api.get('/api/shifts/active');
  if (activeRes.ok && activeRes.body?.data?.id) return activeRes.body.data.id;
  const openRes = await api.post('/api/shifts/open', { opening_balance: 500000 });
  if (!openRes.ok && openRes.status !== 409) {
    throw new Error(`open shift failed (${openRes.status}): ${JSON.stringify(openRes.body)}`);
  }
  const after = await api.get('/api/shifts/active');
  const id = after.body?.data?.id;
  if (!id) throw new Error('no active shift after open');
  return id;
}

export async function closeShift(request: any, ctx: AuthCtx | ApiDriver, shiftId: number, closingBalance = 0): Promise<void> {
  const api = asApi(request, ctx);
  const res = await api.post(`/api/shifts/${shiftId}/close`, { closing_balance: closingBalance });
  if (!res.ok) {
    throw new Error(`close shift failed (${res.status}): ${JSON.stringify(res.body)}`);
  }
}

/** Returns the caller's open cart, creating one when none exists. */
export async function getOrCreateOpenCart(request: any, ctx: AuthCtx | ApiDriver, shiftId?: number): Promise<any> {
  const api = asApi(request, ctx);
  let res = await api.get('/api/pos/cart');
  if (res.ok && res.body?.data?.id) return res.body.data;
  res = await api.post('/api/pos/cart', shiftId ? { shift_id: shiftId } : {});
  if (!res.ok) {
    throw new Error(`create cart failed (${res.status}): ${JSON.stringify(res.body)}`);
  }
  return res.body.data;
}

/**
 * Parks any open cart (held carts are tracked for cleanup by the caller) and
 * returns a brand-new empty open cart — deterministic state per test.
 */
export async function startFreshCart(request: any, ctx: AuthCtx | ApiDriver, shiftId?: number): Promise<any> {
  const api = asApi(request, ctx);
  const open = await api.get('/api/pos/cart');
  if (open.ok && open.body?.data?.id && open.body?.data?.status === 'open') {
    const hold = await api.post(`/api/pos/cart/${open.body.data.id}/hold`, {});
    if (!hold.ok) {
      throw new Error(`hold leftover cart failed (${hold.status}): ${JSON.stringify(hold.body)}`);
    }
  }
  return getOrCreateOpenCart(request, ctx, shiftId);
}

export async function addCartItem(
  request: any,
  ctx: AuthCtx | ApiDriver,
  productId: number,
  quantity = 1,
  extra: Record<string, unknown> = {}
): Promise<{ status: number; body: any }> {
  const api = asApi(request, ctx);
  const res = await api.post('/api/pos/cart/items', { product_id: productId, quantity, ...extra });
  return { status: res.status, body: res.body };
}

export async function holdCart(request: any, ctx: AuthCtx | ApiDriver, cartId: number): Promise<void> {
  const api = asApi(request, ctx);
  const res = await api.post(`/api/pos/cart/${cartId}/hold`, {});
  if (!res.ok) throw new Error(`hold failed (${res.status}): ${JSON.stringify(res.body)}`);
}

export interface CheckoutResult {
  status: number;
  body: any;
}

export async function checkoutCart(
  request: any,
  ctx: AuthCtx | ApiDriver,
  cartId: number,
  payments?: Array<{ payment_method_code: string; amount: number; reference_number?: string }>
): Promise<CheckoutResult> {
  const api = asApi(request, ctx);
  const res = await api.post(`/api/pos/cart/${cartId}/checkout`, payments ? { payments } : {});
  return { status: res.status, body: res.body };
}

/** Cash payment set covering exactly `total`. */
export function cashPayments(total: number): Array<{ payment_method_code: string; amount: number }> {
  return [{ payment_method_code: 'CASH', amount: total }];
}

/** Finds an active product matching `search` and boosts its stock. Deterministic: creates the product if search misses. */
export async function findProductWithStock(request: any, admin: AuthCtx | ApiDriver, search: string, stockBoost = 500): Promise<{ id: number; price: number }> {
  const api = asApi(request, admin);
  const res = await api.get(`/api/products?search=${encodeURIComponent(search)}&status=active&limit=1`);
  let product = res.body?.data?.[0];
  if (!product) {
    const cr = await api.post('/api/products', {
      name: `${search} E2E ${Date.now()}`,
      sku: `E2E-QM-${Date.now()}`,
      price: 10000,
      cost: 5000,
      stock: 10,
      status: 'active',
      category_id: 1,
    });
    if (!cr.ok) throw new Error(`create fallback product for "${search}" failed (${cr.status}): ${JSON.stringify(cr.body)}`);
    product = cr.body?.data ?? cr.body;
    // New product already has stock 10, no need to re-fetch
  }
  await api.post('/api/inventory/adjust', {
    product_id: product.id,
    quantity_change: stockBoost,
    notes: 'E2E cashier scenario stock boost',
  });
  return { id: product.id, price: product.price };
}
