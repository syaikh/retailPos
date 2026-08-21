import { test, expect } from './fixtures';
import { API_BASE } from './fixtures';
import { TestDataTracker } from './db-helper';
import {
  authAs,
  ensureOpenShift,
  startFreshCart,
  addCartItem,
  findProductWithStock,
  type AuthCtx,
} from './pos-api';

// ============================================================================
// Area D: payment validation on the cart checkout endpoint. The same rules
// are enforced on POST /api/sales and are already covered there by
// pos-flow.spec.ts; these cases exercise the cart path end-to-end.
// Traceability: docs/design/Test_Spec_Cashier_Scenario_Coverage.md (CS-D*)
// ============================================================================

test.describe('Payment Validation via Cart Checkout (CS-D*)', () => {
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

  /** Builds a fresh open cart holding exactly one unit of the product. */
  async function newCartWithOneItem(request: any): Promise<{ cartId: number; total: number }> {
    const cart = await startFreshCart(request, cashier);
    tracker.trackCart(cart.id);
    const add = await addCartItem(request, cashier, product.id, 1);
    expect(add.status).toBe(200);
    return { cartId: add.body.data.id, total: add.body.data.total_amount };
  }

  test('CS-D1: allocations below total are rejected with 400', async ({ request }) => {
    const { cartId, total } = await newCartWithOneItem(request);
    const res = await request.post(`${API_BASE}/api/pos/cart/${cartId}/checkout`, {
      headers: cashier.headers,
      data: { payments: [{ payment_method_code: 'CASH', amount: total - 100 }] },
    });
    expect(res.status()).toBe(400);
    expect((await res.json()).error.message).toContain('total payments do not match');
  });

  test('CS-D1b: allocations above total are rejected with 400', async ({ request }) => {
    const { cartId, total } = await newCartWithOneItem(request);
    const res = await request.post(`${API_BASE}/api/pos/cart/${cartId}/checkout`, {
      headers: cashier.headers,
      data: { payments: [{ payment_method_code: 'CASH', amount: total + 100 }] },
    });
    expect(res.status()).toBe(400);
  });

  test('CS-D2: more than 10 payment rows is rejected with 400', async ({ request }) => {
    const { cartId, total } = await newCartWithOneItem(request);
    // 11 QRIS rows of 0 would trip zero-amount first, so use a split that sums
    // correctly across 11 rows but still exceeds the row cap.
    const per = Math.floor(total / 11);
    const rows = Array.from({ length: 11 }, (_, i) => ({
      payment_method_code: 'QRIS',
      amount: i === 10 ? total - per * 10 : per,
      reference_number: `REF-${i}`,
    }));
    const res = await request.post(`${API_BASE}/api/pos/cart/${cartId}/checkout`, {
      headers: cashier.headers,
      data: { payments: rows },
    });
    expect(res.status()).toBe(400);
    expect((await res.json()).error.message.toLowerCase()).toContain('payment');
  });

  test('CS-D3: two CASH rows are rejected with 400', async ({ request }) => {
    const { cartId, total } = await newCartWithOneItem(request);
    const res = await request.post(`${API_BASE}/api/pos/cart/${cartId}/checkout`, {
      headers: cashier.headers,
      data: {
        payments: [
          { payment_method_code: 'CASH', amount: Math.floor(total / 2) },
          { payment_method_code: 'CASH', amount: total - Math.floor(total / 2) },
        ],
      },
    });
    expect(res.status()).toBe(400);
    expect((await res.json()).error.message).toContain('only one cash payment');
  });

  test('CS-D4: duplicate non-cash method is rejected with 400', async ({ request }) => {
    const { cartId, total } = await newCartWithOneItem(request);
    const res = await request.post(`${API_BASE}/api/pos/cart/${cartId}/checkout`, {
      headers: cashier.headers,
      data: {
        payments: [
          { payment_method_code: 'QRIS', amount: Math.floor(total / 2), reference_number: 'R1' },
          { payment_method_code: 'QRIS', amount: total - Math.floor(total / 2), reference_number: 'R2' },
        ],
      },
    });
    expect(res.status()).toBe(400);
    expect((await res.json()).error.message.toLowerCase()).toContain('duplicate');
  });

  test('CS-D5: reference-required method without reference is rejected', async ({ request }) => {
    const { cartId, total } = await newCartWithOneItem(request);
    const res = await request.post(`${API_BASE}/api/pos/cart/${cartId}/checkout`, {
      headers: cashier.headers,
      data: { payments: [{ payment_method_code: 'CARD', amount: total }] },
    });
    expect(res.status()).toBe(400);
    expect((await res.json()).error.message).toContain('reference number is required');
  });

  test('CS-D6: unknown payment method code is rejected', async ({ request }) => {
    const { cartId, total } = await newCartWithOneItem(request);
    const res = await request.post(`${API_BASE}/api/pos/cart/${cartId}/checkout`, {
      headers: cashier.headers,
      data: { payments: [{ payment_method_code: 'BITCOIN', amount: total }] },
    });
    expect(res.status()).toBe(400);
    expect((await res.json()).error.message).toContain('invalid payment method');
  });

  test('CS-D7: zero-amount allocation is rejected', async ({ request }) => {
    const { cartId } = await newCartWithOneItem(request);
    const res = await request.post(`${API_BASE}/api/pos/cart/${cartId}/checkout`, {
      headers: cashier.headers,
      data: { payments: [{ payment_method_code: 'CASH', amount: 0 }] },
    });
    expect(res.status()).toBe(400);
  });

  test('CS-D8: negative-amount allocation is rejected', async ({ request }) => {
    const { cartId, total } = await newCartWithOneItem(request);
    const res = await request.post(`${API_BASE}/api/pos/cart/${cartId}/checkout`, {
      headers: cashier.headers,
      data: { payments: [{ payment_method_code: 'CASH', amount: -total }] },
    });
    expect(res.status()).toBe(400);
  });

  test('CS-D9: valid mixed tender succeeds and records both payments', async ({ request }) => {
    const { cartId, total } = await newCartWithOneItem(request);
    const cash = Math.floor(total / 3);
    const res = await request.post(`${API_BASE}/api/pos/cart/${cartId}/checkout`, {
      headers: cashier.headers,
      data: {
        payments: [
          { payment_method_code: 'CASH', amount: cash },
          { payment_method_code: 'QRIS', amount: total - cash, reference_number: 'E2E-QR' },
        ],
      },
    });
    expect(res.status()).toBe(201);
    const sale = (await res.json()).data;
    tracker.trackSale(sale.id);
    expect(sale.payments).toHaveLength(2);
    expect(sale.total_amount).toBe(total);
  });

  test('CS-D10: empty payments array is rejected', async ({ request }) => {
    const { cartId } = await newCartWithOneItem(request);
    const res = await request.post(`${API_BASE}/api/pos/cart/${cartId}/checkout`, {
      headers: cashier.headers,
      data: { payments: [] },
    });
    // Empty payments short-circuits to ErrZeroPaymentAmount → 400.
    expect(res.status()).toBe(400);
  });
});
