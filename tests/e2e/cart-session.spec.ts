import { test, expect } from './fixtures';
import { apiAs, loginDriver } from './api-driver';
import { ApiDriver } from './api-driver';
import { TestDataTracker, backdateCartExpiry } from './db-helper';
import {
  createCashier,
  ensureOpenShift,
  getOrCreateOpenCart,
  startFreshCart,
  addCartItem,
  holdCart,
  findProductWithStock,
} from './pos-api';

// ============================================================================
// Areas A-C: cart session lifecycle, hold/resume/expiry, cross-cashier
// isolation. Traceability: docs/design/Test_Spec_Cashier_Scenario_Coverage.md
// All interactions run on the API driver (apiAs) — no browser.
// ============================================================================

test.describe('Cart Session Lifecycle (CS-A*, CS-B*, CS-C*)', () => {
  let tracker: TestDataTracker;
  let adminToken: string;
  let cashierToken: string;
  let product: { id: number; price: number };

  // Recreated per test from the stored tokens so each test uses its own request
  // fixture (a request captured in beforeAll must never be reused inside tests).
  let admin: ApiDriver;
  let cashier: ApiDriver;

  test.beforeAll(async ({ request }) => {
    tracker = new TestDataTracker();
    admin = await apiAs(request, 'superadmin');
    cashier = await apiAs(request, 'cashier');
    adminToken = admin.token;
    cashierToken = cashier.token;
    // Tracked so cleanup removes the shift (sales cascade) — otherwise the
    // shift survives with stale total_sales counters and poisons specs that
    // cross-check shift totals against the sales table.
    tracker.trackShift(await ensureOpenShift(request, cashier));
    product = await findProductWithStock(request, admin, 'Quality Model');
  });

  test.beforeEach(async ({ request }) => {
    admin = new ApiDriver(request, adminToken);
    cashier = new ApiDriver(request, cashierToken);
  });

  test.afterAll(async () => {
    tracker.cleanup();
  });

  test('CS-A1: first item add creates an open cart automatically', async ({ request }) => {
    const openRes = await cashier.get('/api/pos/cart');
    if (openRes.ok && openRes.body?.data?.id) {
      // Start clean: hold any leftover open cart so this test owns its state.
      await holdCart(request, cashier, openRes.body.data.id);
      tracker.trackCart(openRes.body.data.id);
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

    const res = await cashier.patch(`/api/pos/cart/items/${item.id}`, { quantity: 4 });
    expect(res.ok).toBeTruthy();
    const updated = res.body.data.items.find((i: any) => i.id === item.id);
    expect(updated.quantity).toBe(4);
    expect(updated.subtotal).toBe(4 * updated.unit_price);
  });

  test('CS-A4: DELETE item removes the line', async ({ request }) => {
    await addCartItem(request, cashier, product.id, 1);
    const cart = await getOrCreateOpenCart(request, cashier);
    tracker.trackCart(cart.id);
    const item = cart.items.find((i: any) => i.product_id === product.id);
    expect(item).toBeTruthy();

    const res = await cashier.del(`/api/pos/cart/items/${item.id}`);
    expect(res.ok).toBeTruthy();
    const after = await cashier.get('/api/pos/cart');
    expect(after.body.data.items.find((i: any) => i.id === item.id)).toBeFalsy();
  });

  test('CS-A5: hold parks the cart; resume reopens it', async ({ request }) => {
    const cart = await getOrCreateOpenCart(request, cashier);
    tracker.trackCart(cart.id);
    await addCartItem(request, cashier, product.id, 1);

    await holdCart(request, cashier, cart.id);

    // Open cart is gone while held...
    const openBody = (await cashier.get('/api/pos/cart')).body;
    expect(openBody.data?.id ?? null).not.toBe(cart.id);

    // ...it appears in the held list...
    const heldBody = (await cashier.get('/api/pos/cart/held')).body;
    expect(heldBody.data.some((c: any) => c.id === cart.id)).toBeTruthy();

    // ...and resume restores it with items intact.
    const resumed = (await cashier.post(`/api/pos/cart/${cart.id}/resume`, {})).body.data;
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

    const res = await cashier.post(`/api/pos/cart/${cart.id}/checkout`, {
      payments: [{ payment_method_code: 'CASH', amount: 1000 }],
    });
    expect(res.status).toBe(409);
    expect(res.body.error.message.toLowerCase()).toContain('expired');
  });

  test('CS-A8: checkout of an empty cart fails with 409', async ({ request }) => {
    const cart = await getOrCreateOpenCart(request, cashier);
    tracker.trackCart(cart.id);

    const res = await cashier.post(`/api/pos/cart/${cart.id}/checkout`, {
      payments: [{ payment_method_code: 'CASH', amount: 0 }],
    });
    expect(res.status).toBe(409);
  });

  test('CS-B1: cashier B cannot read cashier A’s cart by id', async ({ request }) => {
    const second = await createCashier(request, admin, `b1_${Date.now()}`);
    tracker.trackUser(second.id);
    await ensureOpenShift(request, second.ctx);
    const secondApi = new ApiDriver(request, second.ctx.token);

    const cart = await getOrCreateOpenCart(request, cashier);
    tracker.trackCart(cart.id);
    await addCartItem(request, cashier, product.id, 1);

    const res = await secondApi.get(`/api/pos/cart/${cart.id}`);
    // ErrCartNotOwned → 403; must not drift to 404 (existence leak) or 500.
    expect(res.status).toBe(403);
  });

  test('CS-B2: cashier B cannot checkout cashier A’s cart', async ({ request }) => {
    const second = await createCashier(request, admin, `b2_${Date.now()}`);
    tracker.trackUser(second.id);
    await ensureOpenShift(request, second.ctx);
    const secondApi = new ApiDriver(request, second.ctx.token);

    const cart = await getOrCreateOpenCart(request, cashier);
    tracker.trackCart(cart.id);
    await addCartItem(request, cashier, product.id, 1);
    const total = (await cashier.get('/api/pos/cart')).body.data.total_amount;

    const res = await secondApi.post(`/api/pos/cart/${cart.id}/checkout`, {
      payments: [{ payment_method_code: 'CASH', amount: total }],
    });
    // ErrCartNotOwned → 403; must not drift to 404 (existence leak) or 500.
    expect(res.status).toBe(403);

    // The cart is still open and intact for its owner.
    expect((await cashier.get('/api/pos/cart')).body.data.id).toBe(cart.id);
  });

  test('CS-B3: cashier B cannot resume cashier A’s held cart', async ({ request }) => {
    const second = await createCashier(request, admin, `b3_${Date.now()}`);
    tracker.trackUser(second.id);
    await ensureOpenShift(request, second.ctx);
    const secondApi = new ApiDriver(request, second.ctx.token);

    const cart = await getOrCreateOpenCart(request, cashier);
    tracker.trackCart(cart.id);
    await addCartItem(request, cashier, product.id, 1);
    await holdCart(request, cashier, cart.id);

    const res = await secondApi.post(`/api/pos/cart/${cart.id}/resume`, {});
    // ErrCartNotOwned → 403; must not drift to 404 (existence leak) or 500.
    expect(res.status).toBe(403);
  });

  test('CS-C1: customer set on cart survives hold/resume', async ({ request }) => {
    const suffix = Date.now();
    const custRes = await admin.post('/api/customers', {
      name: `E2E Cart Cust ${suffix}`,
      phone: `0812${suffix % 100000000}`,
      email: `cartcust${suffix}@e2e.test`,
    });
    expect(custRes.ok).toBeTruthy();
    const customer = custRes.body.data;
    tracker.trackCustomer(customer.id);

    const cart = await getOrCreateOpenCart(request, cashier);
    tracker.trackCart(cart.id);
    await addCartItem(request, cashier, product.id, 1);

    const setRes = await cashier.patch(`/api/pos/cart/${cart.id}/customer`, { customer_id: customer.id });
    expect(setRes.ok).toBeTruthy();

    await holdCart(request, cashier, cart.id);
    const resumed = (await cashier.post(`/api/pos/cart/${cart.id}/resume`, {})).body.data;
    expect(resumed.customer_id).toBe(customer.id);
  });

  test('CS-C2: held-cart list only shows the caller’s carts', async ({ request }) => {
    const second = await createCashier(request, admin, `c2_${Date.now()}`);
    tracker.trackUser(second.id);
    await ensureOpenShift(request, second.ctx);
    const secondApi = new ApiDriver(request, second.ctx.token);

    const aCart = await getOrCreateOpenCart(request, cashier);
    tracker.trackCart(aCart.id);
    await addCartItem(request, cashier, product.id, 1);
    await holdCart(request, cashier, aCart.id);

    const bBody = (await secondApi.get('/api/pos/cart/held')).body;
    expect(bBody.data.some((c: any) => c.id === aCart.id)).toBeFalsy();
  });
});
