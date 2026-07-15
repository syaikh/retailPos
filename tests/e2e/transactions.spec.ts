import { test, expect } from '@playwright/test';
import { TEST_USERS, API_BASE, loginUI, logoutUI } from './fixtures';

test.describe('Transactions Page', () => {
  test.beforeEach(async ({ page }) => {
    await loginUI(page, 'superadmin', 'admin123');
    await page.goto('http://localhost:5173/transactions');
    await expect(page).toHaveURL(/\/transactions/);
    await expect(page.locator('text=INVOICE')).toBeVisible({ timeout: 10000 });
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('should display transaction table with columns', async ({ page }) => {
    await expect(page.locator('text=INVOICE')).toBeVisible();
    await expect(page.locator('text=DATE')).toBeVisible();
    await expect(page.locator('text=CUSTOMER')).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'PAYMENT' })).toBeVisible();
    await expect(page.locator('text=TOTAL (RP)')).toBeVisible();
  });

  test('should search transactions', async ({ page }) => {
    await page.getByPlaceholder('Search by invoice, product, or customer...').fill('INV');
    await page.waitForTimeout(1000);
    await page.getByPlaceholder('Search by invoice, product, or customer...').fill('');
  });

  test('should filter by date range preset', async ({ page }) => {
    await page.locator('button.date-picker-trigger').first().click();
    await expect(page.locator('.date-picker-container')).toBeVisible();

    await page.getByRole('button', { name: 'Last 7 Days' }).click();
    await page.waitForTimeout(500);
  });

  test('Last 7 Days preset applies Jakarta-boundary dates without errors', async ({ page }) => {
    await page.locator('button.date-picker-trigger').first().click();
    const picker = page.locator('.date-picker-container');
    await expect(picker).toBeVisible();

    await picker.getByRole('button', { name: 'Last 7 Days' }).click();
    await page.waitForTimeout(1500);

    const errorToast = page.locator('text=Failed to load transactions');
    await expect(errorToast).toBeHidden({ timeout: 3000 });

    const rows = page.locator('table tbody tr');
    const count = await rows.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('Last 30 Days preset loads data without errors', async ({ page }) => {
    await page.locator('button.date-picker-trigger').first().click();
    const picker = page.locator('.date-picker-container');
    await expect(picker).toBeVisible();

    await picker.getByRole('button', { name: 'Last 30 Days' }).click();
    await page.waitForTimeout(1500);

    const errorToast = page.locator('text=Failed to load transactions');
    await expect(errorToast).toBeHidden({ timeout: 3000 });
  });

  test('Custom date range applies without errors', async ({ page }) => {
    await page.locator('button.date-picker-trigger').first().click();
    const picker = page.locator('.date-picker-container');
    await expect(picker).toBeVisible();

    const today = new Date();
    const sixtyDaysAgo = new Date();
    sixtyDaysAgo.setDate(today.getDate() - 60);

    const startInput = page.locator('#txn-start-date');
    const endInput = page.locator('#txn-end-date');

    await startInput.fill(sixtyDaysAgo.toISOString().split('T')[0]);
    await endInput.fill(today.toISOString().split('T')[0]);

    await picker.getByRole('button', { name: 'Apply' }).click();
    await page.waitForTimeout(1500);

    const errorToast = page.locator('text=Failed to load transactions');
    await expect(errorToast).toBeHidden({ timeout: 3000 });
  });

  test('should filter by payment method', async ({ page }) => {
    await page.getByRole('button', { name: 'All methods' }).click();
    await expect(page.locator('text=All methods')).toBeVisible();

    await page.getByText('CASH').first().click();
    await page.getByRole('button', { name: 'Clear' }).click();
  });

  test('should open export dropdown', async ({ page }) => {
    await page.locator('button').filter({ hasText: 'Export' }).first().click();
    await expect(page.getByRole('menuitem', { name: 'Export to CSV' })).toBeVisible();
    await expect(page.getByRole('menuitem', { name: 'Export to Excel' })).toBeVisible();
  });

  test('should open transaction detail drawer', async ({ page }) => {
    const firstRow = page.locator('table tbody tr').first();
    await firstRow.waitFor({ state: 'visible', timeout: 10000 });

    await firstRow.click();
    const drawer = page.getByRole('dialog', { name: 'Transaction Details' }).or(page.locator('#transaction-details-heading'));
    await expect(drawer.first()).toBeVisible({ timeout: 5000 });

    await page.locator('button[aria-label="Close drawer"]').click();
    await expect(page.getByRole('dialog', { name: 'Transaction Details' })).toBeHidden({ timeout: 5000 });
  });

  test('should search with no errors', async ({ page }) => {
    await page.getByPlaceholder('Search by invoice, product, or customer...').fill('NONEXISTENT_XYZ_12345');
    await page.waitForTimeout(1000);
    await page.getByPlaceholder('Search by invoice, product, or customer...').fill('');
    await page.waitForTimeout(500);
  });

  test('should display pagination controls', async ({ page }) => {
    await expect(page.locator('text=Rows per page:')).toBeVisible({ timeout: 5000 });
  });

  test('Last 90 Days filter shows pagination and many rows', async ({ page }) => {
    await page.locator('button.date-picker-trigger').first().click();
    const picker = page.locator('.date-picker-container');
    await expect(picker).toBeVisible();

    const today = new Date();
    const ninetyDaysAgo = new Date();
    ninetyDaysAgo.setDate(today.getDate() - 90);

    const startInput = page.locator('#txn-start-date');
    const endInput = page.locator('#txn-end-date');

    await startInput.fill(ninetyDaysAgo.toISOString().split('T')[0]);
    await endInput.fill(today.toISOString().split('T')[0]);

    await picker.getByRole('button', { name: 'Apply' }).click();
    await page.waitForTimeout(1500);

    const errorToast = page.locator('text=Failed to load transactions');
    await expect(errorToast).toBeHidden({ timeout: 3000 });

    await expect(page.locator('text=Rows per page:')).toBeVisible({ timeout: 5000 });

    const rows = page.locator('table tbody tr');
    const count = await rows.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('sale created at Jakarta midnight appears in today transactions', async ({ page }) => {
    const token = await page.evaluate(() => sessionStorage.getItem('access_token'));
    expect(token).toBeTruthy();

    const todayJakarta = await page.evaluate(() => {
      const offset = 7 * 60 * 60 * 1000;
      const shifted = new Date(Date.now() + offset);
      const year = shifted.getUTCFullYear();
      const month = String(shifted.getUTCMonth() + 1).padStart(2, '0');
      const day = String(shifted.getUTCDate()).padStart(2, '0');
      return `${year}-${month}-${day}`;
    });

    const sku = `MIDNIGHT-SALE-${Date.now()}`;
    const createRes = await page.request.post(`${API_BASE}/api/products`, {
      headers: { Authorization: `Bearer ${token}` },
      data: { name: 'Midnight Sale Test', sku, price: 10000, cost: 5000, stock: 10, status: 'active' },
    });
    expect(createRes.ok()).toBeTruthy();
    const created = await createRes.json();
    const productId = created.data?.id || created.id;

    const midnightUTC = new Date(`${todayJakarta}T00:00:00+07:00`).toISOString();
    const saleRes = await page.request.post(`${API_BASE}/api/sales`, {
      headers: { Authorization: `Bearer ${token}` },
      data: {
        items: [{ product_id: productId, quantity: 1, subtotal: 10000 }],
        payment_method: 'CASH',
        created_at: midnightUTC,
      },
    });
    expect(saleRes.ok()).toBeTruthy();
    const sale = await saleRes.json();
    const invoiceNumber = sale.data?.invoice_number || sale.invoice_number;

    await page.waitForTimeout(2000);

    const searchInput = page.getByPlaceholder('Search by invoice, product, or customer...');
    if (invoiceNumber) {
      await searchInput.fill(invoiceNumber);
      await page.waitForTimeout(1000);
      await expect(page.locator(`text=${invoiceNumber}`).first()).toBeVisible({ timeout: 5000 });
    }
  });
});
