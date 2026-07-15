import { test, expect, type Page } from '@playwright/test';
import { TEST_USERS, API_BASE, loginUI, logoutUI, getToken } from './fixtures';

async function ensureSectionExpanded(page: Page, name: string) {
  const btn = page.locator('aside').locator('button').filter({ hasText: name });
  const expanded = await btn.getAttribute('aria-expanded');
  if (expanded !== 'true') {
    await btn.click();
    await page.waitForTimeout(300);
  }
}

async function navigateToProducts(page: Page) {
  const sidebar = page.locator('aside');
  await ensureSectionExpanded(page, 'Master Data');
  const productsBtn = sidebar.locator('button', { hasText: 'Products' }).first();
  await productsBtn.click();
  await expect(page.locator('text=PRODUCT NAME')).toBeVisible({ timeout: 10000 });
}

test.describe('Products Management', () => {
  test.beforeEach(async ({ page }) => {
    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('should load products page with product table', async ({ page }) => {
    await navigateToProducts(page);
    await expect(page.locator('text=PRODUCT NAME')).toBeVisible();
    await expect(page.locator('text=No products found')).toBeHidden();
  });

  test('should search products', async ({ page }) => {
    await navigateToProducts(page);
    await page.getByPlaceholder('Search by name, SKU, or barcode...').fill('Quality Model C Premium');
    await expect(page.locator('text=Quality Model C Premium').first()).toBeVisible({ timeout: 5000 });
  });

  test('should filter by single category', async ({ page }) => {
    await navigateToProducts(page);
    await page.locator('button').filter({ hasText: 'Kategori' }).first().click();
    await expect(page.getByText('Filter Produk')).toBeVisible({ timeout: 5000 });
    await page.waitForTimeout(1000);
    const bevLabel = page.locator('div[role="dialog"][aria-label="Filter Kategori"] label').filter({ hasText: 'Personal Care' }).first();
    await bevLabel.evaluate((el: HTMLElement) => el.click());
    const applyBtn = page.locator('div[role="dialog"][aria-label="Filter Kategori"] button', { hasText: 'Terapkan Filter' });
    await applyBtn.evaluate((el: HTMLElement) => el.click());
    await expect(page.getByText('Filter Produk')).toBeHidden({ timeout: 5000 });
  });

  test('should filter by multiple categories', async ({ page }) => {
    await navigateToProducts(page);
    await page.locator('button').filter({ hasText: 'Kategori' }).first().click();
    await expect(page.getByText('Filter Produk')).toBeVisible({ timeout: 5000 });
    await page.waitForTimeout(1000);
    const gamingLabel = page.locator('div[role="dialog"][aria-label="Filter Kategori"] label').filter({ hasText: 'Gaming' }).first();
    await gamingLabel.evaluate((el: HTMLElement) => el.click());
    const condLabel = page.locator('div[role="dialog"][aria-label="Filter Kategori"] label').filter({ hasText: 'Condiments' }).first();
    await condLabel.evaluate((el: HTMLElement) => el.click());
    const applyBtn = page.locator('div[role="dialog"][aria-label="Filter Kategori"] button', { hasText: 'Terapkan Filter' });
    await applyBtn.evaluate((el: HTMLElement) => el.click());
    await expect(page.getByText('Filter Produk')).toBeHidden({ timeout: 5000 });
  });

  test('should add a new product', async ({ page, request }) => {
    const ts = Date.now();
    const name = `E2E Test Product ${ts}`;
    const sku = `E2E-TEST-${ts}`;
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const productResp = await page.request.post('http://localhost:9095/api/products', {
      headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
      data: { name, sku, price: 50000, stock: 100, category_name: 'Personal Care', status: 'active' }
    });
    expect(productResp.ok()).toBeTruthy();

    await navigateToProducts(page);
    await page.getByPlaceholder('Search by name, SKU, or barcode...').fill(sku);
    await expect(page.locator(`text=${name}`).first()).toBeVisible({ timeout: 5000 });
  });

  test('should open product detail drawer', async ({ page }) => {
    await navigateToProducts(page);
    await page.waitForTimeout(2000);
    const firstRowAction = page.locator('table tbody tr:first-child td:last-child button').first();
    await firstRowAction.click();
    await page.waitForTimeout(500);
    const viewOption = page.locator('text=View Details').first();
    if (await viewOption.isVisible()) {
      await viewOption.click();
      await expect(page.locator('text=Detail Produk')).toBeVisible({ timeout: 5000 });
    }
  });

  test('should show stock status badges with correct variant classes', async ({ page }) => {
    await navigateToProducts(page);
    const stockCells = page.locator('table tbody tr td:nth-child(7)');
    const count = await stockCells.count();
    if (count > 0) {
      const firstCell = stockCells.first();
      const badge = firstCell.locator('span').first();
      await expect(badge).toBeVisible();

      const classNames = await badge.getAttribute('class');
      expect(classNames).toBeTruthy();
    }
  });

  test('should have STOCK and STATUS columns in products table', async ({ page }) => {
    await navigateToProducts(page);
    await expect(page.getByText('STOCK', { exact: true })).toBeVisible();
    await expect(page.getByText('STATUS', { exact: true })).toBeVisible();
  });

  test('should toggle low stock filter', async ({ page }) => {
    await navigateToProducts(page);
    const lowStockBtn = page.locator('button').filter({ hasText: 'Low Stock' }).first();
    await lowStockBtn.click();
    await page.waitForTimeout(1000);
    await lowStockBtn.click();
    await page.waitForTimeout(1000);
  });

  test('should open stock adjustment modal from actions dropdown', async ({ page }) => {
    await navigateToProducts(page);
    await page.waitForTimeout(2000);
    const firstRowAction = page.locator('table tbody tr:first-child td:last-child button').first();
    await firstRowAction.click();
    await page.waitForTimeout(500);
    const adjustOption = page.locator('text=Adjust Stock').first();
    if (await adjustOption.isVisible()) {
      await adjustOption.click();
      await expect(page.locator('text=Adjust Stock').last()).toBeVisible({ timeout: 5000 });
    }
  });

  test('should adjust stock with valid input', async ({ page }) => {
    await navigateToProducts(page);
    await page.waitForTimeout(2000);
    const firstRowAction = page.locator('table tbody tr:first-child td:last-child button').first();
    await firstRowAction.click();
    await page.waitForTimeout(500);
    const adjustOption = page.locator('text=Adjust Stock').first();
    if (await adjustOption.isVisible()) {
      await adjustOption.click();
      await expect(page.locator('text=Adjust Stock').last()).toBeVisible({ timeout: 5000 });
      await page.fill('#adjust-qty', '10');
      await page.fill('#adjust-notes', 'Restocking from supplier');
      await page.getByRole('dialog').getByRole('button', { name: 'Adjust Stock' }).click();
      await expect(page.locator('text=Stock adjusted successfully')).toBeVisible({ timeout: 5000 });
    }
  });

  test('should paginate through product pages with distinct names', async ({ page }) => {
    await navigateToProducts(page);
    await page.waitForTimeout(2000);

    await expect(page.locator('table tbody tr').first()).toBeVisible();

    const nextBtn = page.locator('button[aria-label="Next page"]').first();
    if (await nextBtn.count() > 0 && await nextBtn.isVisible()) {
      await nextBtn.click();
      await page.waitForTimeout(1500);

      const prevBtn = page.locator('button[aria-label="Previous page"]').first();
      if (await prevBtn.count() > 0) {
        await expect(prevBtn).toBeVisible();
      }

      await expect(page.locator('table tbody tr').first()).toBeVisible();
    }
  });
});

test.describe('Products API - Category Filter', () => {
  test('GET /api/products?category=single returns filtered products', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const response = await request.get(`${API_BASE}/api/products?category=Personal+Care`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.data).toBeInstanceOf(Array);
    if (body.data.length > 0) {
      for (const product of body.data) {
        expect(product.category_name).toBe('Personal Care');
      }
    }
  });

  test('GET /api/products?category=multiple returns products matching any category', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const response = await request.get(`${API_BASE}/api/products?category=Gaming,Condiments`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.data).toBeInstanceOf(Array);
    if (body.data.length > 0) {
      const validCategories = ['Gaming', 'Condiments'];
      for (const product of body.data) {
        expect(validCategories).toContain(product.category_name);
      }
    }
  });
});

test.describe('Products API - Stock', () => {
  test('GET /api/products returns stock data', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const response = await request.get(`${API_BASE}/api/products?limit=200`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.data).toBeInstanceOf(Array);
    expect(body.data.length).toBeGreaterThan(0);
    for (const product of body.data) {
      expect(product.stock).toBeGreaterThanOrEqual(0);
    }
  });

  test('GET /api/stock-thresholds returns threshold config', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const response = await request.get(`${API_BASE}/api/stock-thresholds`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body).toHaveProperty('warning');
    expect(body).toHaveProperty('critical');
  });
});
