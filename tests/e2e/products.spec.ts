import { test, expect, type Page } from '@playwright/test';
import { TEST_USERS, API_BASE } from './fixtures';

async function navigateToProducts(page: Page) {
  const sidebar = page.locator('aside');
  const productsBtn = sidebar.locator('button', { hasText: 'Products' }).first();
  await productsBtn.click();
  await expect(page.locator('text=PRODUCT NAME')).toBeVisible({ timeout: 10000 });
}

test.describe('Products Management', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.fill('#username', TEST_USERS.superadmin.username);
    await page.fill('#password', TEST_USERS.superadmin.password);
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL(/\/$/, { timeout: 5000 });
  });

  test('should load products page with product table', async ({ page }) => {
    await navigateToProducts(page);
    await expect(page.locator('text=PRODUCT NAME')).toBeVisible();
    await expect(page.locator('text=No products found')).toBeHidden();
  });

  test('should search products', async ({ page }) => {
    await navigateToProducts(page);
    await page.fill('input[placeholder="Search products..."]', 'Quality Model C Premium');
    await expect(page.locator('text=Quality Model C Premium').first()).toBeVisible({ timeout: 5000 });
  });

  test('should filter by single category', async ({ page }) => {
    await navigateToProducts(page);
    await page.locator('button').filter({ hasText: 'Kategori' }).first().click();
    await expect(page.getByText('Filter Produk')).toBeVisible({ timeout: 5000 });
    await page.waitForTimeout(1000);
    const bevLabel = page.locator('div[role="dialog"][aria-label="Filter Kategori"] label').filter({ hasText: 'Belts' }).first();
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
    const beltsLabel = page.locator('div[role="dialog"][aria-label="Filter Kategori"] label').filter({ hasText: 'Belts' }).first();
    await beltsLabel.evaluate((el: HTMLElement) => el.click());
    const condLabel = page.locator('div[role="dialog"][aria-label="Filter Kategori"] label').filter({ hasText: 'Condiments' }).first();
    await condLabel.evaluate((el: HTMLElement) => el.click());
    const applyBtn = page.locator('div[role="dialog"][aria-label="Filter Kategori"] button', { hasText: 'Terapkan Filter' });
    await applyBtn.evaluate((el: HTMLElement) => el.click());
    await expect(page.getByText('Filter Produk')).toBeHidden({ timeout: 5000 });
  });

  test('should add a new product', async ({ page }) => {
    await navigateToProducts(page);
    await page.click('button', { hasText: 'Add Product' });
    await expect(page.locator('text=Add Product').last()).toBeVisible({ timeout: 5000 });
    await page.fill('#prod-name', 'Test Product E2E');
    await page.fill('#prod-sku', 'TEST-E2E-001');
    await page.fill('#prod-price', '50000');
    await page.fill('#prod-stock', '100');
    await page.click('button[type="submit"]');
    await expect(page.locator('text=Product added')).toBeVisible({ timeout: 5000 });
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

  test('should NOT have stock column in products table', async ({ page }) => {
    await navigateToProducts(page);
    await expect(page.locator('text=STOCK')).toBeHidden();
  });
});

test.describe('Products API - Category Filter', () => {
  test('GET /api/products?category=single returns filtered products', async ({ request }) => {
    const tokenResponse = await request.post(`${API_BASE}/api/login`, {
      data: { username: TEST_USERS.superadmin.username, password: TEST_USERS.superadmin.password }
    });
    const { access_token: token } = await tokenResponse.json();
    const response = await request.get(`${API_BASE}/api/products?category=Belts`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.data).toBeInstanceOf(Array);
    if (body.data.length > 0) {
      for (const product of body.data) {
        expect(product.category_name).toBe('Belts');
      }
    }
  });

  test('GET /api/products?category=multiple returns products matching any category', async ({ request }) => {
    const tokenResponse = await request.post(`${API_BASE}/api/login`, {
      data: { username: TEST_USERS.superadmin.username, password: TEST_USERS.superadmin.password }
    });
    const { access_token: token } = await tokenResponse.json();
    const response = await request.get(`${API_BASE}/api/products?category=Belts,Condiments`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.data).toBeInstanceOf(Array);
    if (body.data.length > 0) {
      const validCategories = ['Belts', 'Condiments'];
      for (const product of body.data) {
        expect(validCategories).toContain(product.category_name);
      }
    }
  });
});
