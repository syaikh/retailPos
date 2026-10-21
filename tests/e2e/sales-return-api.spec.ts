import { test, expect } from './fixtures';
import { apiAs, ApiDriver, loginDriver } from './api-driver';
import { TestDataTracker, execSQL, querySQL } from './db-helper';
import {
  authAs,
  ensureOpenShift,
  startFreshCart,
  addCartItem,
  checkoutCart,
  cashPayments,
  closeActiveShift,
} from './pos-api';

/**
 * Sales Return (Retur Penjualan) data contract, driven at the API layer.
 *
 * This is the API half of Phase 4.5 of docs/design/sales-return-plan.md. The
 * five plan scenarios (replacement, refund, window expired, manager approval,
 * approval expired) are all expressed here as HTTP behaviour: the modal that
 * fronts them is thin and already covered by the salereturn component guards,
 * so clicking through it would test render timing rather than the contract.
 *
 * Determinism notes, because two of these scenarios are policy-driven:
 *
 *   - The replacement-vs-refund split is decided SERVER-SIDE by whether the
 *     product's global stock row is > 0 at process time (service.go
 *     proposeOutcomes). We therefore create our own product per test and drive
 *     its global stock with POST /inventory/adjust, which takes a signed delta
 *     — so "stock available" and "last unit" are exact, not hopeful.
 *   - The approval gate fires when total_refund_amount > return.approval_threshold,
 *     and loadPolicy accepts any threshold >= 0. Writing 0 parks EVERY refund
 *     for approval; writing a huge number lets every refund complete inline.
 *     Both values are written explicitly rather than relying on the seeded
 *     default (500000), and restored in afterAll — app_settings is global to the
 *     database, so leaking a 0 here would park every return in every later spec.
 *     Safe under the CI sharding scheme: each shard provisions its own DB,
 *     backend and frontend (.github/workflows/e2e.yml), and workers:1 +
 *     fullyParallel:false keep specs within a shard strictly sequential.
 */

const data = (b: any) => (b && b.data !== undefined ? b.data : b);

const SETTING_APPROVAL_THRESHOLD = 'return.approval_threshold';
const SETTING_WINDOW_DAYS = 'return.window_days';

/** Refund an amount that always exceeds the threshold, whatever it is set to. */
const HUGE_THRESHOLD = '999999999';

test.describe('Sales Return API Flow', () => {
  let admin: ApiDriver;
  // The cashier's own driver. GET /sales/:id/returnable evaluates the OPEN SHIFT
  // OF THE REQUESTING ACTOR (returnable.go: `s.activeShift.ActiveShiftID(ctx,
  // actor.ID)`), so a superadmin — who holds no shift — always reads
  // 'shift_not_open'. Every snapshot in this file must therefore be taken by the
  // cashier who actually holds the shift and will process the return.
  let cashierApi: ApiDriver;
  let tracker: TestDataTracker;
  let cashierCtx: Awaited<ReturnType<typeof authAs>>;
  let shiftId: number;
  // Captured in beforeEach so afterEach can close the shift (closeActiveShift
  // needs a Playwright request context, which afterEach does not receive).
  let req: any;

  // Policy values as seeded by migration 067 / 000_baseline.sql. Restored in
  // afterAll because app_settings is global to the database: leaking a 0
  // threshold here would park every return in every later spec.
  //
  // These are restored to the known seed defaults rather than to a snapshot of
  // the live values: GET /api/settings returns a fixed branding/receipt struct
  // and does NOT include the return.* policy keys, so there is nothing to
  // snapshot them from without reading the table directly.
  const SEEDED_POLICY: Record<string, string> = {
    [SETTING_APPROVAL_THRESHOLD]: '500000',
    [SETTING_WINDOW_DAYS]: '1',
  };

  test.afterAll(async ({ request }) => {
    // Restore the policy so later specs see the seeded defaults.
    const api = await apiAs(request, 'superadmin');
    await api.put('/api/settings', { settings: SEEDED_POLICY });
  });

  test.beforeEach(async ({ request }) => {
    req = request;
    admin = await apiAs(request, 'superadmin');
    cashierApi = await apiAs(request, 'cashier');
    tracker = new TestDataTracker();
    cashierCtx = await authAs(request, 'cashier');
    shiftId = await ensureOpenShift(request, cashierCtx);

    // Default policy for a scenario unless the test overrides it: refunds
    // complete inline (threshold out of reach) and the window is wide.
    await setPolicy({ [SETTING_APPROVAL_THRESHOLD]: HUGE_THRESHOLD, [SETTING_WINDOW_DAYS]: '1' });
  });

  test.afterEach(async () => {
    // sale_returns.original_sale_id is ON DELETE RESTRICT, so the tracked sales
    // cannot be removed while their returns exist. The child tables
    // (sale_return_items/replacements/payments, return_approvals) cascade from
    // the header, so deleting the header rows is enough. This must run BEFORE
    // tracker.cleanup(), which deletes the sales themselves.
    purgeReturnsForTrackedSales();
    tracker.cleanup();
    await closeActiveShift(req, cashierCtx).catch(() => {});
  });

  // ────────────────────────────── helpers ──────────────────────────────

  /** Upsert policy settings; the endpoint merges, so unrelated keys survive. */
  async function setPolicy(settings: Record<string, string>): Promise<void> {
    const res = await admin.put('/api/settings', { settings });
    expect(res.ok, `PUT /api/settings failed (${res.status}): ${JSON.stringify(res.body)}`).toBeTruthy();
  }

  /**
   * Remove the returns recorded against this spec's tracked sales so the
   * tracker's own `DELETE FROM sales` is not blocked by the RESTRICT foreign
   * key. Cascades cover the item/payment/approval children.
   */
  function purgeReturnsForTrackedSales(): void {
    if (tracker.saleIds.length === 0) return;
    execSQL(`DELETE FROM sale_returns WHERE original_sale_id IN (${tracker.saleIds.join(',')})`);
  }

  /**
   * Create a dedicated product so its stock is ours alone to drive, then give
   * it exactly `stock` global units. Returns the product id.
   */
  async function createProductWithStock(stock: number, price = 10000): Promise<number> {
    const cr = await admin.post('/api/products', {
      name: `E2E Return ${Date.now()}-${Math.floor(Math.random() * 1e6)}`,
      sku: `E2E-RET-${Date.now()}-${Math.floor(Math.random() * 1e6)}`,
      price,
      cost: 5000,
      stock,
      status: 'active',
      category_id: 1,
    });
    expect(cr.ok, `create product failed (${cr.status}): ${JSON.stringify(cr.body)}`).toBeTruthy();
    const product = data(cr.body) ?? cr.body;
    tracker.trackProduct(product.id);
    return product.id;
  }

  /**
   * Complete a sale through the POS API. Returns the sale id and the
   * server-authoritative returnable snapshot of that sale.
   */
  async function sellAndSnapshot(
    request: any,
    productId: number,
    quantity = 1
  ): Promise<{ saleId: number; snapshot: any }> {
    const cart = await startFreshCart(request, cashierCtx, shiftId);
    tracker.trackCart(cart.id);
    const add = await addCartItem(request, cashierCtx, productId, quantity);
    expect(add.status, `add to cart failed: ${JSON.stringify(add.body)}`).toBe(200);
    const sale = await checkoutCart(request, cashierCtx, cart.id, cashPayments(add.body.data.total_amount));
    expect(sale.status, `checkout failed: ${JSON.stringify(sale.body)}`).toBe(201);
    const saleId = data(sale.body).id;
    tracker.trackSale(saleId);

    const snap = await cashierApi.get(`/api/sales/${saleId}/returnable`);
    expect(snap.ok, `returnable failed (${snap.status})`).toBeTruthy();
    return { saleId, snapshot: data(snap.body) };
  }

  /** Drive a product's global stock to exactly `target` (signed adjust). */
  async function setStockTo(productId: number, current: number, target: number): Promise<void> {
    const delta = target - current;
    if (delta === 0) return;
    const res = await admin.post('/api/inventory/adjust', {
      product_id: productId,
      quantity_change: delta,
      notes: 'E2E sales-return stock control',
    });
    expect(res.ok, `stock adjust failed (${res.status}): ${JSON.stringify(res.body)}`).toBeTruthy();
  }

  /** Current global sellable stock for a product. */
  async function globalStock(productId: number): Promise<number> {
    const rows = querySQL<{ quantity: number }>(
      `SELECT COALESCE(quantity, 0) AS quantity FROM product_stock
       WHERE product_id = ${productId} AND warehouse_id IS NULL AND store_id IS NULL`
    );
    return rows.length ? Number(rows[0].quantity) : 0;
  }

  /**
   * Scenario preamble shared by the three approval-gated tests: a threshold of
   * 0 parks every refund for approval, and a product drained to zero stock
   * forces the refund outcome (with stock, proposeOutcomes would offer a
   * replacement instead, whose 0 refund never reaches the gate).
   */
  async function parkRefund(request: any): Promise<{ productId: number; saleId: number; pending: any }> {
    await setPolicy({ [SETTING_APPROVAL_THRESHOLD]: '0' });

    const productId = await createProductWithStock(1);
    const { saleId, snapshot } = await sellAndSnapshot(request, productId, 1);
    await setStockTo(productId, await globalStock(productId), 0);

    const created = await cashierApi.post(`/api/sales/${saleId}/return`, {
      reason: 'Barang rusak',
      condition: 'damaged',
      items: [{ sale_item_id: snapshot.items[0].sale_item_id, quantity: 1 }],
    });
    expect(created.status, `return failed: ${JSON.stringify(created.body)}`).toBe(201);
    const pending = data(created.body);
    expect(pending.status).toBe('pending_approval');
    expect(pending.expires_at).toBeTruthy();
    return { productId, saleId, pending };
  }

  // ─────────────────────── scenario 1: replacement ───────────────────────

  test('replacement: stock available → goods exchanged, no money refunded', async ({ request }) => {
    const productId = await createProductWithStock(50);
    const { saleId, snapshot } = await sellAndSnapshot(request, productId, 1);

    expect(snapshot.eligible, `sale not returnable: ${JSON.stringify(snapshot)}`).toBe(true);
    const line = snapshot.items[0];
    expect(line.returnable_quantity).toBe(1);
    expect(line.stock_available).toBeGreaterThan(0);

    const created = await cashierApi.post(`/api/sales/${saleId}/return`, {
      reason: 'Barang rusak',
      condition: 'damaged',
      items: [{ sale_item_id: line.sale_item_id, quantity: 1 }],
    });

    expect(created.status, `return failed: ${JSON.stringify(created.body)}`).toBe(201);
    const ret = data(created.body);

    // A replacement moves goods: the unit is written off and the same product
    // is handed over, so nothing is refunded.
    expect(ret.return_type).toBe('replacement');
    expect(ret.total_refund_amount).toBe(0);
    expect(ret.status).toBe('completed');
    expect(ret.original_sale_id).toBe(saleId);

    const item = ret.items[0];
    expect(item.outcome).toBe('written_off');
    expect(item.condition).toBe('damaged');
    expect(item.replacements).toHaveLength(1);
    expect(item.replacements[0].quantity).toBe(1);

    // The replacement unit came out of stock. Stock accounting across the
    // whole test: 50 created − 1 taken by the sale (49) − 1 handed over as the
    // replacement (48). The returned unit is written off, never restocked.
    expect(await globalStock(productId)).toBe(48);
  });

  // ───────────────────────── scenario 2: refund ─────────────────────────

  test('refund: last unit in stock → refund paid out, unit written off', async ({ request }) => {
    // Stock 0 at sale time is impossible to sell, so sell the last unit then
    // drain the global row to zero before processing the return.
    const productId = await createProductWithStock(1);
    const { saleId, snapshot } = await sellAndSnapshot(request, productId, 1);

    await setStockTo(productId, await globalStock(productId), 0);

    const line = snapshot.items[0];

    const freshSnap = await cashierApi.get(`/api/sales/${saleId}/returnable`);
    const freshLine = data(freshSnap.body).items[0];
    expect(freshLine.stock_available).toBe(0);

    const created = await cashierApi.post(`/api/sales/${saleId}/return`, {
      reason: 'Barang rusak',
      condition: 'damaged',
      items: [{ sale_item_id: freshLine.sale_item_id, quantity: 1 }],
    });

    expect(created.status, `return failed: ${JSON.stringify(created.body)}`).toBe(201);
    const ret = data(created.body);

    expect(ret.return_type).toBe('refund');
    expect(ret.total_refund_amount).toBeGreaterThan(0);
    expect(ret.status).toBe('completed');
    expect(ret.items[0].replacements).toHaveLength(0);
    expect(ret.items[0].outcome).toBe('written_off');

    // A refund leaves the drawer through the original payment method.
    expect(ret.payments.length).toBeGreaterThan(0);
    const paid = ret.payments.reduce((sum: number, p: any) => sum + p.amount, 0);
    expect(paid).toBe(ret.total_refund_amount);

    // Never restocked (Phase 1 decision): the written-off unit stays gone, so
    // the row must not have been credited back.
    expect(await globalStock(productId)).toBe(0);

    // The line is consumed: nothing left to return on this sale.
    const after = await cashierApi.get(`/api/sales/${saleId}/returnable`);
    const afterData = data(after.body);
    expect(afterData.eligible).toBe(false);
    expect(afterData.ineligible_reason).toBe('fully_returned');
    expect(afterData.items[0].returnable_quantity).toBe(0);
    expect(line.sale_item_id).toBe(freshLine.sale_item_id);
  });

  // ──────────────────── scenario 3: return window expired ────────────────

  test('window expired: a sale older than the window is refused', async ({ request }) => {
    const productId = await createProductWithStock(50);
    const { saleId } = await sellAndSnapshot(request, productId, 1);

    // Age the sale beyond the 1-day window, snapshotting so the timestamp is
    // restored even if an assertion fails mid-test.
    const before = querySQL<{ created_at: string }>(
      `SELECT created_at::text AS created_at FROM sales WHERE id = ${saleId}`
    );
    try {
      execSQL(`UPDATE sales SET created_at = NOW() - INTERVAL '8 days' WHERE id = ${saleId}`);

      // The pre-flight snapshot reports the expiry as data, not an error, so
      // the UI can disable the button with a reason.
      const snap = await cashierApi.get(`/api/sales/${saleId}/returnable`);
      expect(snap.ok).toBeTruthy();
      const snapshot = data(snap.body);
      expect(snapshot.eligible).toBe(false);
      expect(snapshot.ineligible_reason).toBe('window_expired');

      // And the write path refuses independently of the snapshot.
      const created = await cashierApi.post(`/api/sales/${saleId}/return`, {
        reason: 'Barang rusak',
        condition: 'damaged',
        items: [{ sale_item_id: snapshot.items[0].sale_item_id, quantity: 1 }],
      });
      expect(created.status).toBe(400);
      expect(String(created.body?.error ?? '')).toContain('return window expired');
    } finally {
      if (before.length) {
        execSQL(`UPDATE sales SET created_at = '${before[0].created_at}' WHERE id = ${saleId}`);
      }
    }
  });

  // ────────────── scenario 4: high-value return needs approval ────────────

  test('manager approval: a refund above the threshold parks, then a manager completes it', async ({ request }) => {
    const { pending } = await parkRefund(request);
    const pendingId = pending.id;

    // The approval object is NOT populated on the create response: ProcessReturn
    // calls InsertApproval but never assigns the in-memory ret.Approval, and
    // presentReturn emits null when it is nil. The approval state must be read
    // back. (Known gap, recorded for the phase review — the frontend branches on
    // `status`, not on `approval`, so nothing user-visible depends on it.)
    expect(pending.approval).toBeNull();

    const detail = await admin.get(`/api/sales/returns/${pendingId}`);
    expect(detail.ok, `detail failed (${detail.status})`).toBeTruthy();
    const detailBody = data(detail.body);
    expect(detailBody.approval).toBeTruthy();
    expect(detailBody.approval.status).toBe('pending');
    const approvalId = detailBody.approval.id;

    // A parked return has not moved goods or money yet: the figures are a
    // proposal, and the item row carries the written_off outcome the approval
    // transaction would commit (returnItemsOf sets it unconditionally).
    expect(detailBody.total_refund_amount).toBeGreaterThan(0);
    expect(detailBody.items[0].outcome).toBe('written_off');

    // The cashier cannot see the approval queue (no sale.return.approve).
    const cashierQueue = await cashierApi.get('/api/sales/returns/pending');
    expect(cashierQueue.status).toBe(403);

    // The manager sees it, oldest first.
    const manager = await apiAs(request, 'manager');
    const queue = await manager.get('/api/sales/returns/pending?limit=50');
    expect(queue.ok, `pending queue failed (${queue.status})`).toBeTruthy();
    const ids = (data(queue.body) ?? []).map((r: any) => r.id);
    expect(ids).toContain(pendingId);

    // Approval completes it and pays the refund out.
    const approved = await manager.post(`/api/sales/returns/${pendingId}/approve`, { note: 'Disetujui' });
    expect(approved.status, `approve failed: ${JSON.stringify(approved.body)}`).toBe(200);
    const done = data(approved.body);
    expect(done.status).toBe('completed');
    expect(done.return_type).toBe('refund');
    expect(done.total_refund_amount).toBeGreaterThan(0);

    // It has left the actionable queue.
    const afterQueue = await manager.get('/api/sales/returns/pending?limit=50');
    const afterIds = (data(afterQueue.body) ?? []).map((r: any) => r.id);
    expect(afterIds).not.toContain(pendingId);

    // The committed approval state is read back (the approve response carries
    // the same unrefreshed approval object as the create response).
    const approvedDetail = await admin.get(`/api/sales/returns/${pendingId}`);
    expect(approvedDetail.ok).toBeTruthy();
    const doneDetail = data(approvedDetail.body);
    expect(doneDetail.total_refund_amount).toBe(done.total_refund_amount);
    expect(doneDetail.approval).toBeTruthy();
    expect(doneDetail.approval.id).toBe(approvalId);
    expect(doneDetail.approval.status).toBe('approved');
    expect(doneDetail.approval.approved_by).toBeTruthy();

    // The refund was paid out to the original payment method (the sale was cash).
    expect(doneDetail.payments.length).toBeGreaterThan(0);
    const paid = doneDetail.payments.reduce((sum: number, p: any) => sum + p.amount, 0);
    expect(paid).toBe(doneDetail.total_refund_amount);
  });

  test('manager rejection: a parked refund can be cancelled and releases the line', async ({ request }) => {
    const { productId, saleId, pending } = await parkRefund(request);

    const manager = await apiAs(request, 'manager');
    const rejected = await manager.post(`/api/sales/returns/${pending.id}/reject`, {
      reason: 'Tidak sesuai kebijakan',
    });
    expect(rejected.status, `reject failed: ${JSON.stringify(rejected.body)}`).toBe(200);
    const cancelled = data(rejected.body);
    expect(cancelled.status).toBe('cancelled');

    // Read the committed approval state: RejectReturn sets ret.Status but does
    // not refresh the in-memory approval row, so the mutation response still
    // reports approval.status 'pending' alongside status 'cancelled'.
    const detail = await admin.get(`/api/sales/returns/${pending.id}`);
    expect(detail.ok).toBeTruthy();
    const rejectedDetail = data(detail.body);
    expect(rejectedDetail.status).toBe('cancelled');
    expect(rejectedDetail.approval.status).toBe('rejected');
    expect(rejectedDetail.approval.rejected_reason).toBe('Tidak sesuai kebijakan');
    // Nothing was paid out or restocked.
    expect(rejectedDetail.payments).toHaveLength(0);
    expect(await globalStock(productId)).toBe(0);

    // A cancelled return stops consuming the sale's returnable quantity, so
    // the same unit can be returned again.
    const snapAfter = await cashierApi.get(`/api/sales/${saleId}/returnable`);
    const afterData = data(snapAfter.body);
    expect(afterData.eligible).toBe(true);
    expect(afterData.items[0].returnable_quantity).toBe(1);
  });

  // ─────────────── scenario 5: approval window expires ───────────────────

  test('approval expired: approving an elapsed request reports the expiry and flips the row', async ({ request }) => {
    const { productId, saleId, pending } = await parkRefund(request);

    // The approval id is only on the read path (see scenario 4).
    const initial = await admin.get(`/api/sales/returns/${pending.id}`);
    const approvalId = data(initial.body).approval.id;

    // Expiry is LAZY: nothing sweeps the row, the elapsed window is only
    // observed when someone acts. Backdate the deadline to simulate the passage
    // of time without waiting the policy's 5 minutes. BOTH columns matter:
    // presentReturn derives the effective status from sale_returns.expires_at
    // (ret.ExpiresAt) while the approval's own window lives on
    // return_approvals.expires_at (used by presentReturn's approval block and
    // by ListPending).
    execSQL(`UPDATE return_approvals SET expires_at = NOW() - INTERVAL '1 minute' WHERE id = ${approvalId}`);
    execSQL(`UPDATE sale_returns SET expires_at = NOW() - INTERVAL '1 minute' WHERE id = ${pending.id}`);

    // Read paths report the EFFECTIVE status without flipping the row.
    const detail = await admin.get(`/api/sales/returns/${pending.id}`);
    expect(data(detail.body).status).toBe('expired');
    expect(data(detail.body).approval.status).toBe('expired');

    // The queue must no longer offer it as actionable.
    const manager = await apiAs(request, 'manager');
    const queue = await manager.get('/api/sales/returns/pending?limit=50');
    const ids = (data(queue.body) ?? []).map((r: any) => r.id);
    expect(ids).not.toContain(pending.id);

    // Approving is refused with a conflict, and the attempt self-heals the
    // row: the lazy flip happens inside the approval transaction.
    const approved = await manager.post(`/api/sales/returns/${pending.id}/approve`, {});
    expect(approved.status).toBe(409);
    expect(String(approved.body?.error ?? '')).toContain('approval window has expired');

    const rows = querySQL<{ status: string }>(
      `SELECT status FROM return_approvals WHERE id = ${approvalId}`
    );
    expect(rows[0]?.status).toBe('expired');

    // No goods or money moved.
    expect(await globalStock(productId)).toBe(0);
    const snapAfter = await cashierApi.get(`/api/sales/${saleId}/returnable`);
    // An expired return does not consume returnable quantity, so the line is
    // offered again.
    expect(data(snapAfter.body).items[0].returnable_quantity).toBe(1);
  });

  // ─────────────────────── guard: RBAC on the surface ─────────────────────

  test('RBAC: the approval queue and decisions are manager-only', async ({ request }) => {
    const manager = await apiAs(request, 'manager');

    expect((await cashierApi.get('/api/sales/returns/pending')).status).toBe(403);
    expect((await cashierApi.post('/api/sales/returns/1/approve', {})).status).toBe(403);
    expect((await cashierApi.post('/api/sales/returns/1/reject', { reason: 'x' })).status).toBe(403);

    // Managers hold sale.return.approve; the queue itself is readable.
    expect((await manager.get('/api/sales/returns/pending')).ok).toBeTruthy();

    // Approving a non-existent return is a 404, not a leak of existence.
    expect((await manager.post('/api/sales/returns/999999999/approve', {})).status).toBe(404);
  });

  test('RBAC: a cashier without sale.return cannot read the returnable snapshot', async ({ request }) => {
    // inventory_staff holds neither sale.return nor sale.view.
    const staff = await loginDriver(request, 'inventory_staff', 'admin123');
    const res = await staff.get('/api/sales/1/returnable');
    expect(res.status).toBe(403);
  });
});
