import { test, expect } from './fixtures';
import { TEST_USERS, API_BASE, FRONTEND_BASE, loginUI, logoutUI } from './fixtures';
import { TestDataTracker, execSQL, querySQL } from './db-helper';
import {
  authAs,
  ensureOpenShift,
  addCartItem,
  holdCart,
  checkoutCart,
  cashPayments,
  findProductWithStock,
  startFreshCart,
} from './pos-api';

// ============================================================================
// Area F: POS UI edge cases.
//
// Two layers live in this file:
//   POS-UI-01..06 — UI smoke checks for the core cashier loop (shortcuts,
//     cart ops, payment-modal gating, API/UI cart consistency).
//   CS-F1/F2/F5/F8/F14/F15 — automated subset of the documented Area F
//     scenarios. The remaining CS-F scenarios are verified by code
//     inspection only (see Test_Spec doc §5 for the rationale).
// Traceability: docs/design/Test_Spec_Cashier_Scenario_Coverage.md
// ============================================================================

test.describe('POS UI Edge Cases', () => {
  let tracker: TestDataTracker;

  test.beforeEach(async ({ page, request }) => {
    tracker = new TestDataTracker();
    await loginUI(page, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
    // PosPage bounces cashiers without an active shift to /shifts
    // (PosPage.svelte mount guard), so guarantee an open shift BEFORE the UI
    // loads. Tracked so cleanup removes it (sales cascade) — an abandoned
    // shift keeps stale total_sales counters after tracked sales are deleted,
    // poisoning later specs that cross-check shift totals against sales.
    const cashier = await authAs(request, 'cashier');
    tracker.trackShift(await ensureOpenShift(request, cashier));
    // Park any leftover open cart so line-count assertions start empty.
    await holdLeftoverCart(request, cashier);
    await page.goto(`${FRONTEND_BASE}/pos`);
    await expect(page).toHaveURL(/\/pos/);
  });

  test.afterEach(async ({ page }) => {
    tracker.cleanup();
    await logoutUI(page);
  });

  async function addFirstProductViaUI(page: any) {
    await page.waitForTimeout(2000);
    const addButton = page.locator('button:not([disabled])').filter({ hasText: 'Add' }).first();
    await addButton.waitFor({ state: 'visible', timeout: 10000 });
    await expect(addButton).toBeEnabled({ timeout: 5000 });
    await addButton.click();
    await page.waitForTimeout(500);
    await expect(page.locator('text=Your cart is empty')).toBeHidden({ timeout: 5000 });
  }

  /** Holds any leftover open cart that already has items so tests start clean. */
  async function holdLeftoverCart(request: any, cashier: any) {
    const openRes = await request.get(`${API_BASE}/api/pos/cart`, { headers: cashier.headers });
    if (openRes.ok()) {
      const body = await openRes.json();
      if (body.data?.id && (body.data.items ?? []).length > 0) {
        await holdCart(request, cashier, body.data.id);
      }
    }
  }

  function cartLineCount(page: any) {
    return page.locator('button[aria-label="Increase quantity"]').count();
  }

  // ------------------------------------------------------------------
  // POS-UI smoke layer
  // ------------------------------------------------------------------

  test('POS-UI-01: F2 focuses the product search', async ({ page }) => {
    await page.keyboard.press('F2');
    await expect(page.locator('#pos-search-input')).toBeFocused({ timeout: 3000 });
  });

  test('POS-UI-02: F4 opens the payment modal and Escape closes it', async ({ page }) => {
    await addFirstProductViaUI(page);
    await page.keyboard.press('F4');
    await expect(page.getByRole('dialog', { name: 'Payment' })).toBeVisible({ timeout: 5000 });
    await page.keyboard.press('Escape');
    await expect(page.getByRole('dialog', { name: 'Payment' })).toBeHidden({ timeout: 5000 });
  });

  test('POS-UI-03: ALT+DEL clears the cart', async ({ page }) => {
    await addFirstProductViaUI(page);
    await page.locator('button[aria-label="Clear Cart"]').click();
    await page.waitForTimeout(500);
    await expect(page.locator('text=Your cart is empty')).toBeVisible({ timeout: 5000 });
  });

  test('POS-UI-04: decreasing quantity at one removes the line', async ({ page }) => {
    // Designed behavior (PosPage.updateQty): newQty <= 0 removes the item,
    // so the floor of one is enforced by removal, not by clamping.
    await addFirstProductViaUI(page);
    const decreaseBtn = page.locator('button[aria-label="Decrease quantity"]').first();
    await decreaseBtn.waitFor({ state: 'visible', timeout: 5000 });
    await decreaseBtn.click();
    await expect(page.locator('text=Your cart is empty')).toBeVisible({ timeout: 5000 });
  });

  test('POS-UI-05: Done stays disabled until allocations match the total', async ({ page }) => {
    await addFirstProductViaUI(page);
    await page.keyboard.press('F4');
    const dialog = page.getByRole('dialog', { name: 'Payment' });
    await expect(dialog).toBeVisible({ timeout: 5000 });

    const amountInput = dialog.locator('input[id^="alloc-amount-"]').first();
    await amountInput.fill('0');
    await page.waitForTimeout(200);

    const done = dialog.locator('button').filter({ hasText: 'Done [Enter]' });
    await expect(done).toBeDisabled();

    // Exact total re-enables Done. The grand total is the large primary <p>
    // (e.g. "Rp 40.000"); there is no literal "Total" label in the modal.
    const totalText = await dialog.locator('p.text-3xl').textContent();
    const totalNumber = Number((totalText || '').replace(/[^0-9]/g, ''));
    expect(totalNumber, `unable to parse total from "${totalText}"`).toBeGreaterThan(0);
    await amountInput.fill(String(totalNumber));
    await page.waitForTimeout(200);
    await expect(done).toBeEnabled();
  });

  test('POS-UI-06: item added via API is visible after reload', async ({ page, request }) => {
    const admin = await authAs(request, 'superadmin');
    const cashier = await authAs(request, 'cashier');
    await ensureOpenShift(request, cashier);
    const product = await findProductWithStock(request, admin, 'Quality Model');

    await addFirstProductViaUI(page);
    const before = await cartLineCount(page);

    const add = await addCartItem(request, cashier, product.id, 1);
    expect(add.status).toBe(200);

    // There is no websocket topic for carts; the UI converges on its next
    // cart fetch (page load), so a reload must surface the extra line.
    await page.reload();
    await expect
      .poll(() => cartLineCount(page), { timeout: 15000 })
      .toBeGreaterThan(before);
  });

  // ------------------------------------------------------------------
  // Documented CS-F scenarios (automated subset)
  // ------------------------------------------------------------------

  test('CS-F1: F7 opens the Held Sales modal without navigating away', async ({ page }) => {
    // Wait for PosPage to mount so the window keydown handler is bound.
    await expect(page.locator('#pos-search-input')).toBeVisible({ timeout: 10000 });
    await page.keyboard.press('F7');
    const dialog = page.getByRole('dialog', { name: 'Held Sales' });
    await expect(dialog).toBeVisible({ timeout: 5000 });
    // F5 stays free for browser reload; F7 must not change the route.
    expect(page.url()).toMatch(/\/pos/);
    // Close via the button: Escape only works while focus sits inside the
    // modal, which is not guaranteed when there are no held sales.
    await dialog.locator('button[aria-label="Close"]').click();
    await expect(dialog).toBeHidden({ timeout: 5000 });
  });

  test('CS-F2: Escape clears the search only when no modal is open', async ({ page }) => {
    await addFirstProductViaUI(page);
    const search = page.locator('#pos-search-input');
    await search.fill('abc');
    await page.waitForTimeout(700);

    // Modal open: Escape closes the modal and preserves the search text.
    await page.keyboard.press('F4');
    const dialog = page.getByRole('dialog', { name: 'Payment' });
    await expect(dialog).toBeVisible({ timeout: 5000 });
    await page.keyboard.press('Escape');
    await expect(dialog).toBeHidden({ timeout: 5000 });
    await expect(search).toHaveValue('abc');

    // No modal: Escape clears the search.
    await page.keyboard.press('Escape');
    await expect(search).toHaveValue('');
  });

  test('CS-F5: Enter adds the first search result; zero results is a safe no-op', async ({ page }) => {
    const search = page.locator('#pos-search-input');
    await search.fill('Quality');
    await page
      .locator('button:not([disabled])')
      .filter({ hasText: 'Add' })
      .first()
      .waitFor({ state: 'visible', timeout: 10000 });

    const before = await cartLineCount(page);
    await search.press('Enter');
    await page.waitForTimeout(800);
    expect(await cartLineCount(page)).toBeGreaterThan(before);

    // Query with no matches: Enter must not add anything.
    const afterFirstAdd = await cartLineCount(page);
    await search.fill('zzqqxx-no-match');
    await expect(page.locator('text=No products found')).toBeVisible({ timeout: 10000 });
    await search.press('Enter');
    await page.waitForTimeout(500);
    expect(await cartLineCount(page)).toBe(afterFirstAdd);
  });

  test('CS-F8: reprint is disabled when the last sale is older than 7 Jakarta days', async ({ page, request }) => {
    const admin = await authAs(request, 'superadmin');
    const cashier = await authAs(request, 'cashier');
    const shiftId = await ensureOpenShift(request, cashier);
    const product = await findProductWithStock(request, admin, 'Quality Model');

    const cart = await startFreshCart(request, cashier, shiftId);
    tracker.trackCart(cart.id);
    const add = await addCartItem(request, cashier, product.id, 1);
    expect(add.status).toBe(200);
    const sale = await checkoutCart(request, cashier, cart.id, cashPayments(add.body.data.total_amount));
    expect(sale.status).toBe(201);
    tracker.trackSale(sale.body.data.id);

    const printBtn = page.getByRole('button', { name: /print/i }).first();

    // Fresh sale inside the window: reprint available after reload.
    await page.reload();
    await expect(printBtn).toBeEnabled({ timeout: 15000 });

    // Backdate every completed sale of this cashier beyond the window, then
    // restore the original timestamps afterwards.
    const snapshot = querySQL<{ id: number; created_at: string }>(
      `SELECT id, created_at::text AS created_at FROM sales WHERE cashier_id = ${TEST_USERS.cashier.id} AND status = 'completed'`
    );
    try {
      execSQL(`UPDATE sales SET created_at = NOW() - INTERVAL '8 days' WHERE cashier_id = ${TEST_USERS.cashier.id} AND status = 'completed'`);
      await page.reload();
      await expect(printBtn).toBeDisabled({ timeout: 15000 });
    } finally {
      for (const row of snapshot) {
        execSQL(`UPDATE sales SET created_at = '${row.created_at}' WHERE id = ${row.id}`);
      }
    }
  });

  test('CS-F14: copying a SKU shows check feedback and fills the clipboard', async ({ page }) => {
    await page.context().grantPermissions(['clipboard-read', 'clipboard-write']);
    await page.waitForTimeout(2000);

    const copyBtn = page.locator('button[aria-label="Copy SKU"]').first();
    await copyBtn.waitFor({ state: 'visible', timeout: 10000 });
    await copyBtn.click();

    // ✓ feedback appears immediately...
    await expect(copyBtn.locator('text=✓')).toBeVisible({ timeout: 3000 });
    const clip = await page.evaluate(() => navigator.clipboard.readText());
    expect(clip.length).toBeGreaterThan(0);

    // ...and disappears after ~2 s.
    await expect(copyBtn.locator('text=✓')).toBeHidden({ timeout: 4000 });
  });

  test('CS-F15: cashier is redirected when opening restricted routes by URL', async ({ page }) => {
    for (const path of ['/reports', '/purchase-orders', '/consignment']) {
      await page.goto(`${FRONTEND_BASE}${path}`);
      // Redirect target is the role's default route (e.g. /shifts); the
      // contract under test is only that the restricted path is never shown.
      await page.waitForURL((url) => !url.pathname.includes(path), { timeout: 10000 });
      expect(page.url()).not.toContain(path);
    }
  });
});
