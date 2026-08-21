import { test, expect } from './fixtures';
import { API_BASE, TEST_USERS } from './fixtures';
import { TestDataTracker, scalarSQL } from './db-helper';
import {
  authAs,
  createCashier,
  ensureOpenShift,
  closeShift,
  getOrCreateOpenCart,
  addCartItem,
  holdCart,
  checkoutCart,
  cashPayments,
  findProductWithStock,
  type AuthCtx,
} from './pos-api';

// ============================================================================
// Area E: shift rules — open/close lifecycle, totals accumulation, and the
// D2/D3/D4 security regressions (closed-shift rejection, shift_id ownership,
// owner-scoped sale reads).
// Traceability: docs/design/Test_Spec_Cashier_Scenario_Coverage.md
// ============================================================================

test.describe('Shift Rules and Ownership (CS-E*)', () => {
  let tracker: TestDataTracker;
  let admin: AuthCtx;
  let cashierA: AuthCtx;
  let product: { id: number; price: number };

  test.beforeAll(async ({ request }) => {
    tracker = new TestDataTracker();
    admin = await authAs(request, 'superadmin');
    cashierA = await authAs(request, 'cashier');
    product = await findProductWithStock(request, admin, 'Quality Model');
  });

  test.afterAll(async () => {
    tracker.cleanup();
  });

  test('CS-E1/E2: opening a shift makes it active; a second open is rejected', async ({ request }) => {
    const shiftId = await ensureOpenShift(request, cashierA);
    tracker.trackShift(shiftId);

    const active = await request.get(`${API_BASE}/api/shifts/active`, { headers: cashierA.headers });
    expect(active.ok()).toBeTruthy();
    expect((await active.json()).data.id).toBe(shiftId);

    const res = await request.post(`${API_BASE}/api/shifts/open`, {
      headers: cashierA.headers,
      data: { opening_balance: 500000 },
    });
    // Rejected as a duplicate open (current contract: 400 with a flat error string).
    expect(res.status()).toBe(400);
    expect((await res.text()).toLowerCase()).toContain('already');
  });

  test('CS-E4/E7: checkout accumulates onto the shift; summary matches sales', async ({ request }) => {
    const shiftId = await ensureOpenShift(request, cashierA);
    tracker.trackShift(shiftId);

    // Cart is created bound to the open shift — the POS UI always works this way.
    const cart = await getOrCreateOpenCart(request, cashierA, shiftId);
    tracker.trackCart(cart.id);
    const add = await addCartItem(request, cashierA, product.id, 2);
    expect(add.status).toBe(200);
    const total = add.body.data.total_amount;

    const saleRes = await checkoutCart(request, cashierA, cart.id, [
      { payment_method_code: 'CASH', amount: Math.floor(total / 2) },
      { payment_method_code: 'QRIS', amount: total - Math.floor(total / 2), reference_number: 'E2E-SPLIT' },
    ]);
    expect(saleRes.status).toBe(201);
    tracker.trackSale(saleRes.body.data.id);

    const active = await (await request.get(`${API_BASE}/api/shifts/active`, { headers: cashierA.headers })).json();
    expect(active.data.total_sales).toBeGreaterThanOrEqual(total);
    expect(active.data.transaction_count).toBeGreaterThanOrEqual(1);

    // Shift totals equal the sum of its completed sales (DB cross-check).
    const dbSum = scalarSQL(`SELECT COALESCE(SUM(total_amount),0) FROM sales WHERE shift_id = ${shiftId} AND status = 'completed'`);
    expect(Number(dbSum)).toBe(active.data.total_sales);
  });

  test('CS-E5/E8 [D2]: held cart survives close; resume OK but checkout fails cleanly', async ({ request }) => {
    const second = await createCashier(request, admin, `e5_${Date.now()}`);
    tracker.trackUser(second.id);
    const shiftId = await ensureOpenShift(request, second.ctx);
    tracker.trackShift(shiftId);

    const cart = await getOrCreateOpenCart(request, second.ctx, shiftId);
    tracker.trackCart(cart.id);
    const add = await addCartItem(request, second.ctx, product.id, 1);
    expect(add.status).toBe(200);
    const total = add.body.data.total_amount;

    await holdCart(request, second.ctx, cart.id);
    await closeShift(request, second.ctx, shiftId);

    // Resume still works after the shift closed...
    const resume = await request.post(`${API_BASE}/api/pos/cart/${cart.id}/resume`, { headers: second.ctx.headers });
    expect(resume.ok()).toBeTruthy();

    // ...but checkout is rejected with a friendly 409, not a raw 500.
    const co = await checkoutCart(request, second.ctx, cart.id, cashPayments(total));
    expect(co.status).toBe(409);
    expect(co.body.error.message.toLowerCase()).toContain('shift');
    expect(JSON.stringify(co.body)).not.toContain('Internal Server Error');
  });

  test('CS-E6 [D2]: direct sale referencing a closed shift fails with 409', async ({ request }) => {
    const second = await createCashier(request, admin, `e6_${Date.now()}`);
    tracker.trackUser(second.id);
    const shiftId = await ensureOpenShift(request, second.ctx);
    tracker.trackShift(shiftId);
    await closeShift(request, second.ctx, shiftId);

    const res = await request.post(`${API_BASE}/api/sales`, {
      headers: second.ctx.headers,
      data: {
        payment_method: 'CASH',
        shift_id: shiftId,
        items: [{ product_id: product.id, quantity: 1 }],
        payments: [{ payment_method_code: 'CASH', amount: product.price }],
      },
    });
    expect(res.status()).toBe(409);
    const body = await res.json();
    expect(body.error.message.toLowerCase()).toContain('shift');
  });

  test('CS-E9 [D3]: foreign shift_id never accumulates onto another user\u2019s shift', async ({ request }) => {
    // Cashier A owns an open shift.
    const shiftIdA = await ensureOpenShift(request, cashierA);
    tracker.trackShift(shiftIdA);
    const before = scalarSQL(`SELECT transaction_count FROM shifts WHERE id = ${shiftIdA}`);

    // Cashier B tries to attach their sale to A's open shift.
    const second = await createCashier(request, admin, `e9_${Date.now()}`);
    tracker.trackUser(second.id);
    await ensureOpenShift(request, second.ctx); // B has their own shift too

    const res = await request.post(`${API_BASE}/api/sales`, {
      headers: second.ctx.headers,
      data: {
        payment_method: 'CASH',
        shift_id: shiftIdA,
        items: [{ product_id: product.id, quantity: 1 }],
        payments: [{ payment_method_code: 'CASH', amount: product.price }],
      },
    });
    // Foreign/closed shift contribution maps to ErrShiftNotOpen → 409.
    expect(res.status()).toBe(409);

    // A's shift totals are untouched.
    const after = scalarSQL(`SELECT transaction_count FROM shifts WHERE id = ${shiftIdA}`);
    expect(after).toBe(before);
  });

  test('CS-E10 [D4]: sale reads are owner-scoped for cashiers, store-wide for managers', async ({ request }) => {
    const manager = await authAs(request, 'manager');

    // Cashier A completes a sale.
    const shiftIdA = await ensureOpenShift(request, cashierA);
    tracker.trackShift(shiftIdA);
    const cart = await getOrCreateOpenCart(request, cashierA, shiftIdA);
    tracker.trackCart(cart.id);
    const add = await addCartItem(request, cashierA, product.id, 1);
    const total = add.body.data.total_amount;
    const saleRes = await checkoutCart(request, cashierA, cart.id, cashPayments(total));
    expect(saleRes.status).toBe(201);
    const saleId = saleRes.body.data.id;
    tracker.trackSale(saleId);

    // Cashier B sees only their own history...
    const second = await createCashier(request, admin, `e10_${Date.now()}`);
    tracker.trackUser(second.id);
    const bShift = await ensureOpenShift(request, second.ctx);
    tracker.trackShift(bShift);
    const bList = await (await request.get(`${API_BASE}/api/sales?limit=100`, { headers: second.ctx.headers })).json();
    for (const s of bList.data ?? []) {
      expect(s.cashier_id).toBe(second.id);
    }

    // ...cannot widen via cashier_id filter...
    const widened = await (
      await request.get(`${API_BASE}/api/sales?limit=100&cashier_id=${TEST_USERS.cashier.id}`, { headers: second.ctx.headers })
    ).json();
    for (const s of widened.data ?? []) {
      expect(s.cashier_id).toBe(second.id);
    }

    // ...and gets 404 (not 403) on A's sale detail so existence is not leaked.
    const detail = await request.get(`${API_BASE}/api/sales/${saleId}`, { headers: second.ctx.headers });
    expect(detail.status()).toBe(404);

    // Manager reads store-wide and can filter by cashier.
    const mListRes = await request.get(`${API_BASE}/api/sales?limit=100&cashier_id=${TEST_USERS.cashier.id}`, { headers: manager.headers });
    expect(mListRes.ok(), `manager sales list failed: ${await mListRes.text()}`).toBeTruthy();
    const mList = await mListRes.json();
    expect(mList.data.length).toBeGreaterThan(0);
    const mDetail = await request.get(`${API_BASE}/api/sales/${saleId}`, { headers: manager.headers });
    expect(mDetail.status()).toBe(200);
  });
});
