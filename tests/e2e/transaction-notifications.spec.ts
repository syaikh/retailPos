import { test, expect } from './fixtures';
import {
  TEST_USERS,
  API_BASE,
  loginUI,
  logoutUI,
  getToken,
  authHeader,
} from './fixtures';

const FT_TAB = 'Find Transaction';

/**
 * Create a product + stock + sale via the API and return its id/invoice.
 * `overrides` is merged into the sale payload (e.g. customer_id, created_at).
 */
async function createSale(
  request: any,
  token: string,
  overrides: Record<string, unknown> = {},
): Promise<{ id: number; invoice: string }> {
  const sku = `E2E-TXN-${Date.now()}-${Math.floor(Math.random() * 1000)}`;
  const createRes = await request.post(`${API_BASE}/api/products`, {
    headers: authHeader(token),
    data: { name: 'E2E Txn Sale', sku, price: 10000, cost: 5000, stock: 10, status: 'active' },
  });
  expect(createRes.ok(), `product create failed: ${createRes.status()}`).toBeTruthy();
  const created = await createRes.json();
  const productId = created.data?.id || created.id;

  await request.post(`${API_BASE}/api/inventory/adjust`, {
    headers: authHeader(token),
    data: { product_id: productId, quantity_change: 10, notes: 'E2E seed stock' },
  });

  const saleRes = await request.post(`${API_BASE}/api/sales`, {
    headers: authHeader(token),
    data: {
      items: [{ product_id: productId, quantity: 1 }],
      payment_method: 'CASH',
      ...overrides,
    },
  });
  expect(saleRes.ok(), `sale create failed: ${saleRes.status()}`).toBeTruthy();
  const sale = await saleRes.json();
  return {
    id: sale.data?.id || sale.id,
    invoice: sale.data?.invoice_number || sale.invoice_number,
  };
}

/** Log in as cashier and open an active shift (required to load /transactions). */
async function openCashierShift(page: any, request: any) {
  const token = await getToken(request, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
  await request.post(`${API_BASE}/api/shifts/close-all`, { headers: authHeader(token) });

  await loginUI(page, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
  await page.goto('/shifts');
  await expect(page).toHaveURL(/\/shifts/);

  await page.getByRole('button', { name: 'Open Shift' }).first().click();
  const dialog = page.getByRole('dialog', { name: 'Open Shift' });
  await expect(dialog).toBeVisible();
  await page.locator('#opening-balance').fill('100000');
  await dialog.getByRole('button', { name: 'Open Shift' }).click();
  await expect(page.locator('text=Active Shift')).toBeVisible({ timeout: 10000 });
}

test.describe('Find Transaction (cashier, search-only)', () => {
  test.beforeEach(async ({ page, request }) => {
    await openCashierShift(page, request);
    await page.goto('/transactions');
    await expect(page).toHaveURL(/\/transactions/);
    await page.getByRole('button', { name: FT_TAB }).click();
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('shows search hint and no default table before a query', async ({ page }) => {
    await expect(
      page.getByText('Enter an invoice number to look up a transaction.'),
    ).toBeVisible();
    // No table is rendered until the cashier performs an explicit search.
    expect(await page.locator('table').count()).toBe(0);
  });

  test('exact invoice search returns a redacted summary row, opening the drawer', async ({ page, request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const { invoice } = await createSale(request, token);

    await page.getByPlaceholder('Search by invoice number...').fill(invoice);
    await page.getByPlaceholder('Search by invoice number...').press('Enter');

    const row = page.locator('table tbody tr', { hasText: invoice }).first();
    await expect(row).toBeVisible({ timeout: 5000 });
    await row.click();

    const drawer = page.getByRole('dialog', { name: 'Transaction Details' });
    await expect(drawer).toBeVisible({ timeout: 5000 });
    await expect(drawer).toContainText(invoice);
  });

  test('no-match search shows the no-results message', async ({ page }) => {
    await page.getByPlaceholder('Search by invoice number...').fill('NONEXISTENT_XYZ_12345');
    await page.getByPlaceholder('Search by invoice number...').press('Enter');
    await expect(
      page.getByText('No results for "NONEXISTENT_XYZ_12345"'),
    ).toBeVisible({ timeout: 5000 });
  });

  test('sorting a column re-queries the lookup and reorders ascending', async ({ page, request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    await createSale(request, token);
    await createSale(request, token);

    await page.getByPlaceholder('Search by invoice number...').fill('INV-2026');
    await page.getByPlaceholder('Search by invoice number...').press('Enter');
    await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 5000 });

    const readInvoices = () =>
      page
        .locator('table tbody tr')
        .evaluateAll((trs) => trs.map((tr) => (tr.querySelector('td')?.textContent || '').trim()));

    const before = await readInvoices();
    expect(before.length).toBeGreaterThan(1);

    await page.getByRole('button', { name: 'INVOICE', exact: true }).click();
    await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 5000 });

    const after = await readInvoices();
    // The lookup must have re-queried: the returned page is now sorted
    // ascending by invoice number (first INVOICE click => asc), and that
    // differs from the default created_at-desc order.
    const sorted = [...after].sort((a, b) => a.localeCompare(b));
    expect(after).toEqual(sorted);
    expect(after).not.toEqual(before);
  });

  test('search lookup requests the full history window (epoch start) so old receipts are reachable', async ({ page, request }) => {
    // The sale-creation API ignores any caller-supplied `created_at` (the
    // repository INSERT hardcodes the column list and lets the DB default it),
    // so an E2E test cannot backdate a sale over HTTP. The date-independence
    // of the Find Transaction lookup is therefore enforced by the FRONTEND,
    // which widens `start_date` to 2000-01-01 whenever a search term is
    // present (bypassing the backend's 30-day default window). We assert that
    // outgoing request param directly — if the widening were removed the value
    // would be today's date and this test would fail.
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const { invoice } = await createSale(request, token);

    let capturedStart = '';
    await page.route('**/api/sales/lookup*', async (route) => {
      const url = new URL(route.request().url());
      capturedStart = url.searchParams.get('start_date') || '';
      await route.fulfill({ json: { data: [], total: 0 } });
    });

    await page.getByPlaceholder('Search by invoice number...').fill(invoice);
    await page.getByPlaceholder('Search by invoice number...').press('Enter');

    await expect
      .poll(() => capturedStart, { timeout: 5000 })
      .toBe('2000-01-01');
  });
});

test.describe('Transaction deep-link & refresh (superadmin/manager)', () => {
  test.beforeEach(async ({ page }) => {
    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    await page.goto('/transactions');
    await expect(page).toHaveURL(/\/transactions/);
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('deep link opens the owner transaction detail drawer', async ({ page, request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const { id, invoice } = await createSale(request, token);

    await page.goto(`/transactions?txn=${id}`);
    const drawer = page.getByRole('dialog', { name: 'Transaction Details' });
    await expect(drawer).toBeVisible({ timeout: 5000 });
    await expect(drawer).toContainText(invoice);
  });

  test('notification click deep-links to the transaction detail drawer', async ({ page, request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const { id, invoice } = await createSale(request, token);

    const bell = page.getByRole('button', { name: 'Notifications' });
    await bell.click();

    const notif = page.locator('button', { hasText: invoice }).first();
    await expect(notif).toBeVisible({ timeout: 10000 });
    await notif.click();

    await expect(page).toHaveURL(new RegExp(`/transactions\\?txn=${id}`), { timeout: 5000 });
    const drawer = page.getByRole('dialog', { name: 'Transaction Details' });
    await expect(drawer).toBeVisible({ timeout: 5000 });
    await expect(drawer).toContainText(invoice);
  });

  test('My Transactions shows an Updated WIB timestamp and refreshes without error', async ({ page }) => {
    const ts = page.getByText(/Updated \d{1,2}:\d{2} WIB/).first();
    await expect(ts).toBeVisible({ timeout: 5000 });

    await page.getByRole('button', { name: 'Refresh' }).click();
    await expect(ts).toBeVisible({ timeout: 5000 });
    await expect(page.getByText(/Failed to load/)).toBeHidden({ timeout: 5000 });
  });
});

test.describe('Cross-cashier redacted deep-link (cashier)', () => {
  test('foreign sale opens in redacted lookup mode (no customer PII)', async ({ page, request }) => {
    // Sale owned by superadmin, with a named customer (PII we must NOT see).
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const { id, invoice } = await createSale(request, token, { customer_id: 2 });

    await openCashierShift(page, request);
    await page.goto(`/transactions?txn=${id}`);

    const drawer = page.getByRole('dialog', { name: 'Transaction Details' });
    await expect(drawer).toBeVisible({ timeout: 5000 });
    await expect(drawer).toContainText(invoice);
    // The foreign (non-owner, non-report.view) cashier must see a redacted
    // lookup, never the customer's real name.
    await expect(drawer).not.toContainText('Hasan Hakim');
  });
});

test.describe('Manager new-transactions banner', () => {
  test.beforeEach(async ({ page }) => {
    await loginUI(page, TEST_USERS.manager.username, TEST_USERS.manager.password);
    await page.goto('/transactions');
    await expect(page).toHaveURL(/\/transactions/);
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('banner appears on sale_created and clears on View', async ({ page, request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    await createSale(request, token);

    await expect(page.getByText(/new transactions since/)).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: 'View' }).click();
    await expect(page.getByText(/new transactions since/)).toBeHidden({ timeout: 5000 });
  });

  test('banner clears on manual Refresh', async ({ page, request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    await createSale(request, token);

    await expect(page.getByText(/new transactions since/)).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: 'Refresh' }).click();
    await expect(page.getByText(/new transactions since/)).toBeHidden({ timeout: 5000 });
  });
});
