import { test, expect } from './fixtures';
import { API_BASE } from './fixtures';
import { TestDataTracker, backdateCartExpiry } from './db-helper';
import {
  authAs,
  createCashier,
  ensureOpenShift,
  getOrCreateOpenCart,
  startFreshCart,
  addCartItem,
  holdCart,
  findProductWithStock,
  type AuthCtx,
} from './pos-api';

// ============================================================================
// Areas A-C: cart session lifecycle, hold/resume/expiry, cross-cashier
// isolation. Traceability: docs/design/Test_Spec_Cashier_Scenario_Coverage.md
// ============================================================================

test.describe('Cart Session Lifecycle (CS-A*, CS-B*, CS-C*)', () => {
  let tracker: TestDataTracker;
  let admin: AuthCtx;
  let cashier: AuthCtx;
  let product: { id: number; price: number };

  test.beforeAll(async ({ request }) => {
    tracker = new TestDataTracker();
    admin = await authAs(request, 'superadmin');
    cashier = await authAs(request, 'cashier');
    // Tracked so cleanup removes the shift (sales cascade) — otherwise the
    // shift survives with stale total_sales counters and poisons specs that
    // cross-check shift totals against the sales table.
    tracker.trackShift(await ensureOpenShift(request, cashier));
    product = await findProductWithStock(request, admin, 'Quality Model');
  });

  test.afterAll(async () => {
    tracker.cleanup();
  });

  test('CS-A1: first item add creates an open cart automatically', async ({ request }) => {
    const res = await request.get(`${API_BASE}/api/pos/cart`, { headers: cashier.headers });
    if (res.ok()) {
      const body = await res.json();
      if (body.data?.id) {
        // Start clean: hold any leftover open cart so this test owns its state.
        await holdCart(request, cashier, body.data.id);
        tracker.trackCart(body.data.id);
      }
    }

    const add = await addCartItem(request, cashier, product.id, 1);
    expect(add.status).toBe(200);
    expect(add.body.data.status).toBe('open');
    expect(add.body.data.items).toHaveLength(1);
    tracker.trackCart(add.body.data.id);
  });

  test('CS-A2: repeated adds of the same product accumulate quantity', async ({ request }) => {
    const cart = await startFreshCart(request, cashier);
    tracker.trackCart(cart.id);

    await addCartItem(request, cashier, product.id, 1);
    const add2 = await addCartItem(request, cashier, product.id, 2);
    expect(add2.status).toBe(200);

    // The server appends one line per add; quantity accumulates across lines
    // and duplicate lines are merged later at checkout into the sale.
    const lines = add2.body.data.items.filter((i: any) => i.product_id === product.id);
    const totalQty = lines.reduce((sum: number, l: any) => sum + l.quantity, 0);
    expect(totalQty).toBe(3);
    expect(lines.length).toBeGreaterThanOrEqual(1);
    expect(add2.body.data.total_amount).toBe(3 * lines[0].unit_price);
  });

  test('CS-A3: PATCH item quantity updates line and totals', async ({ request }) => {
    await addCartItem(request, cashier, product.id, 1);
    const cart = await getOrCreateOpenCart(request, cashier);
    tracker.trackCart(cart.id);

    const item = cart.items.find((i: any) => i.product_id === product.id);
    expect(item).toBeTruthy();

    const res = await request.patch(`${API_BASE}/api/pos/cart/items/${item.id}`, {
      headers: cashier.headers,
      data: { quantity: 4 },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    const updated = body.data.items.find((i: any) => i.id === item.id);
    expect(updated.quantity).toBe(4);
    expect(updated.subtotal).toBe(4 * updated.unit_price);
  });

  test('CS-A4: DELETE item removes the line', async ({ request }) => {
    await addCartItem(request, cashier, product.id, 1);
    const cart = await getOrCreateOpenCart(request, cashier);
    tracker.trackCart(cart.id);
    const item = cart.items.find((i: any) => i.product_id === product.id);
    expect(item).toBeTruthy();

    const res = await request.delete(`${API_BASE}/api/pos/cart/items/${item.id}`, { headers: cashier.headers });
    expect(res.ok()).toBeTruthy();
    const after = await request.get(`${API_BASE}/api/pos/cart`, { headers: cashier.headers });
    const body = await after.json();
    expect(body.data.items.find((i: any) => i.id === item.id)).toBeFalsy();
  });

  test('CS-A5: hold parks the cart; resume reopens it', async ({ request }) => {
    const cart = await getOrCreateOpenCart(request, cashier);
    tracker.trackCart(cart.id);
    await addCartItem(request, cashier, product.id, 1);

    await holdCart(request, cashier, cart.id);

    // Open cart is gone while held...
    const openRes = await request.get(`${API_BASE}/api/pos/cart`, { headers: cashier.headers });
    const openBody = await openRes.json();
    expect(openBody.data?.id ?? null).not.toBe(cart.id);

    // ...it appears in the held list...
    const heldRes = await request.get(`${API_BASE}/api/pos/cart/held`, { headers: cashier.headers });
    expect(heldRes.ok()).toBeTruthy();
    const heldBody = await heldRes.json();
    expect(heldBody.data.some((c: any) => c.id === cart.id)).toBeTruthy();

    // ...and resume restores it with items intact.
    const resumeRes = await request.post(`${API_BASE}/api/pos/cart/${cart.id}/resume`, { headers: cashier.headers });
    expect(resumeRes.ok()).toBeTruthy();
    const resumed = (await resumeRes.json()).data;
    expect(resumed.status).toBe('open');
    expect(resumed.items.length).toBeGreaterThan(0);
  });

  test('CS-A6: expired cart is not returned as the open cart', async ({ request }) => {
    const cart = await getOrCreateOpenCart(request, cashier);
    tracker.trackCart(cart.id);
    await addCartItem(request, cashier, product.id, 1);

    backdateCartExpiry(cart.id);

    // A fresh add must not reuse the expired cart.
    const add = await addCartItem(request, cashier, product.id, 1);
    expect(add.status).toBe(200);
    const newCartId = add.body.data.id;
    tracker.trackCart(newCartId);
    expect(newCartId).not.toBe(cart.id);
  });

  test('CS-A7: checkout of an expired cart fails with 409', async ({ request }) => {
    const cart = await getOrCreateOpenCart(request, cashier);
    tracker.trackCart(cart.id);
    await addCartItem(request, cashier, product.id, 1);
    backdateCartExpiry(cart.id);

    const res = await request.post(`${API_BASE}/api/pos/cart/${cart.id}/checkout`, {
      headers: cashier.headers,
      data: { payments: [{ payment_method_code: 'CASH', amount: 1000 }] },
    });
    expect(res.status()).toBe(409);
    const body = await res.json();
    expect(body.error.message.toLowerCase()).toContain('expired');
  });

  test('CS-A8: checkout of an empty cart fails with 409', async ({ request }) => {
    const cart = await getOrCreateOpenCart(request, cashier);
    tracker.trackCart(cart.id);

    const res = await request.post(`${API_BASE}/api/pos/cart/${cart.id}/checkout`, {
      headers: cashier.headers,
      data: { payments: [{ payment_method_code: 'CASH', amount: 0 }] },
    });
    expect(res.status()).toBe(409);
  });

  test('CS-B1: cashier B cannot read cashier A\u2019s cart by id', async ({ request }) => {
    const second = await createCashier(request, admin, `b1_${Date.now()}`);
    tracker.trackUser(second.id);
    await ensureOpenShift(request, second.ctx);

    const cart = await getOrCreateOpenCart(request, cashier);
    tracker.trackCart(cart.id);
    await addCartItem(request, cashier, product.id, 1);

    const res = await request.get(`${API_BASE}/api/pos/cart/${cart.id}`, { headers: second.ctx.headers });
    // ErrCartNotOwned → 403; must not drift to 404 (existence leak) or 500.
    expect(res.status()).toBe(403);
  });

  test('CS-B2: cashier B cannot checkout cashier A\u2019s cart', async ({ request }) => {
    const second = await createCashier(request, admin, `b2_${Date.now()}`);
    tracker.trackUser(second.id);
    await ensureOpenShift(request, second.ctx);

    const cart = await getOrCreateOpenCart(request, cashier);
    tracker.trackCart(cart.id);
    await addCartItem(request, cashier, product.id, 1);
    const total = (await (await request.get(`${API_BASE}/api/pos/cart`, { headers: cashier.headers })).json()).data.total_amount;

    const res = await request.post(`${API_BASE}/api/pos/cart/${cart.id}/checkout`, {
      headers: second.ctx.headers,
      data: { payments: [{ payment_method_code: 'CASH', amount: total }] },
    });
    // ErrCartNotOwned → 403; must not drift to 404 (existence leak) or 500.
    expect(res.status()).toBe(403);

    // The cart is still open and intact for its owner.
    const own = await request.get(`${API_BASE}/api/pos/cart`, { headers: cashier.headers });
    expect((await own.json()).data.id).toBe(cart.id);
  });

  test('CS-B3: cashier B cannot resume cashier A\u2019s held cart', async ({ request }) => {
    const second = await createCashier(request, admin, `b3_${Date.now()}`);
    tracker.trackUser(second.id);
    await ensureOpenShift(request, second.ctx);

    const cart = await getOrCreateOpenCart(request, cashier);
    tracker.trackCart(cart.id);
    await addCartItem(request, cashier, product.id, 1);
    await holdCart(request, cashier, cart.id);

    const res = await request.post(`${API_BASE}/api/pos/cart/${cart.id}/resume`, { headers: second.ctx.headers });
    // ErrCartNotOwned → 403; must not drift to 404 (existence leak) or 500.
    expect(res.status()).toBe(403);
  });

  test('CS-C1: customer set on cart survives hold/resume', async ({ request }) => {
    const suffix = Date.now();
    const custRes = await request.post(`${API_BASE}/api/customers`, {
      headers: admin.headers,
      data: { name: `E2E Cart Cust ${suffix}`, phone: `0812${suffix % 100000000}`, email: `cartcust${suffix}@e2e.test` },
    });
    expect(custRes.ok()).toBeTruthy();
    const customer = (await custRes.json()).data;
    tracker.trackCustomer(customer.id);

    const cart = await getOrCreateOpenCart(request, cashier);
    tracker.trackCart(cart.id);
    await addCartItem(request, cashier, product.id, 1);

    const setRes = await request.patch(`${API_BASE}/api/pos/cart/${cart.id}/customer`, {
      headers: cashier.headers,
      data: { customer_id: customer.id },
    });
    expect(setRes.ok()).toBeTruthy();

    await holdCart(request, cashier, cart.id);
    const resumeRes = await request.post(`${API_BASE}/api/pos/cart/${cart.id}/resume`, { headers: cashier.headers });
    expect(resumeRes.ok()).toBeTruthy();
    expect((await resumeRes.json()).data.customer_id).toBe(customer.id);
  });

  test('CS-C2: held-cart list only shows the caller\u2019s carts', async ({ request }) => {
    const second = await createCashier(request, admin, `c2_${Date.now()}`);
    tracker.trackUser(second.id);
    await ensureOpenShift(request, second.ctx);

    const aCart = await getOrCreateOpenCart(request, cashier);
    tracker.trackCart(aCart.id);
    await addCartItem(request, cashier, product.id, 1);
    await holdCart(request, cashier, aCart.id);

    const bHeld = await request.get(`${API_BASE}/api/pos/cart/held`, { headers: second.ctx.headers });
    const bBody = await bHeld.json();
    expect(bBody.data.some((c: any) => c.id === aCart.id)).toBeFalsy();
  });
});
