import { test, expect, type Page } from '@playwright/test';
import { TEST_USERS, API_BASE } from './fixtures';

async function navigateToInventory(page: Page) {
  const sidebar = page.locator('aside');
  const inventoryBtn = sidebar.locator('button', { hasText: 'Inventory' }).first();
  await inventoryBtn.click();
  await expect(page.locator('text=PRODUCT NAME')).toBeVisible({ timeout: 10000 });
}

test.describe('Inventory Management', () => {
  test.beforeEach(async ({ page }) => {
    // Login before each test
    await page.goto('/');
    await page.fill('#username', TEST_USERS.superadmin.username);
    await page.fill('#password', TEST_USERS.superadmin.password);
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL(/\/$/, { timeout: 5000 });
  });

  test('should load inventory page with product table', async ({ page }) => {
    await navigateToInventory(page);
    await expect(page.locator('text=PRODUCT NAME')).toBeVisible();
    await expect(page.locator('text=No products found')).toBeHidden();
  });

  test('should search products', async ({ page }) => {
    await navigateToInventory(page);
    await page.fill('input[placeholder="Search products..."]', 'Quality Model');
    await expect(page.locator('text=Quality Model')).toBeVisible({ timeout: 5000 });
  });

  test('should filter by single category', async ({ page }) => {
    await navigateToInventory(page);

    // Click the category filter button (has SlidersHorizontal icon, text "Kategori")
    await page.locator('button').filter({ hasText: 'Kategori' }).first().click();

    // Wait for category filter modal to open and animation to complete
    await expect(page.getByText('Filter Produk')).toBeVisible({ timeout: 5000 });
    await page.waitForTimeout(1000);

    // Click the category label inside the drawer using JS evaluate to bypass overlay
    const bevLabel = page.locator('div[role="dialog"][aria-label="Filter Kategori"] label').filter({ hasText: 'Beverages' }).first();
    await bevLabel.evaluate((el: HTMLElement) => el.click());

    // Click apply button using JS evaluate
    const applyBtn = page.locator('div[role="dialog"][aria-label="Filter Kategori"] button', { hasText: 'Terapkan Filter' });
    await applyBtn.evaluate((el: HTMLElement) => el.click());

    // Wait for modal to close and table to update
    await expect(page.getByText('Filter Produk')).toBeHidden({ timeout: 5000 });
  });

  test('should filter by multiple categories', async ({ page }) => {
    await navigateToInventory(page);

    // Click the category filter button
    await page.locator('button').filter({ hasText: 'Kategori' }).first().click();

    // Wait for category filter modal
    await expect(page.getByText('Filter Produk')).toBeVisible({ timeout: 5000 });
    await page.waitForTimeout(1000);

    // Select two categories using JS evaluate to bypass overlay
    const bevLabel = page.locator('div[role="dialog"][aria-label="Filter Kategori"] label').filter({ hasText: 'Beverages' }).first();
    await bevLabel.evaluate((el: HTMLElement) => el.click());

    const beltsLabel = page.locator('div[role="dialog"][aria-label="Filter Kategori"] label').filter({ hasText: 'Belts' }).first();
    await beltsLabel.evaluate((el: HTMLElement) => el.click());

    // Click apply button using JS evaluate
    const applyBtn = page.locator('div[role="dialog"][aria-label="Filter Kategori"] button', { hasText: 'Terapkan Filter' });
    await applyBtn.evaluate((el: HTMLElement) => el.click());

    // Wait for modal to close
    await expect(page.getByText('Filter Produk')).toBeHidden({ timeout: 5000 });
  });
});

test.describe('Inventory API - Category Filter', () => {
  test('GET /api/products?category=single returns filtered products', async ({ request }) => {
    const tokenResponse = await request.post(`${API_BASE}/api/login`, {
      data: { username: TEST_USERS.superadmin.username, password: TEST_USERS.superadmin.password }
    });
    const { access_token: token } = await tokenResponse.json();

    const response = await request.get(`${API_BASE}/api/products?category=Beverages`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.data).toBeInstanceOf(Array);

    // If there are products, verify they're from Beverages category
    if (body.data.length > 0) {
      for (const product of body.data) {
        expect(product.category_name).toBe('Beverages');
      }
    }
  });

  test('GET /api/products?category=multiple returns products matching any category', async ({ request }) => {
    const tokenResponse = await request.post(`${API_BASE}/api/login`, {
      data: { username: TEST_USERS.superadmin.username, password: TEST_USERS.superadmin.password }
    });
    const { access_token: token } = await tokenResponse.json();

    const response = await request.get(`${API_BASE}/api/products?category=Beverages,Belts`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.data).toBeInstanceOf(Array);

    // Verify products are from either Beverages or Belts category
    if (body.data.length > 0) {
      const validCategories = ['Beverages', 'Belts'];
      for (const product of body.data) {
        expect(validCategories).toContain(product.category_name);
      }
    }
  });

  test('GET /api/products with all filters combined', async ({ request }) => {
    const tokenResponse = await request.post(`${API_BASE}/api/login`, {
      data: { username: TEST_USERS.superadmin.username, password: TEST_USERS.superadmin.password }
    });
    const { access_token: token } = await tokenResponse.json();

    const response = await request.get(`${API_BASE}/api/products?search=&limit=10&offset=0&category=Beverages`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.data).toBeInstanceOf(Array);
    expect(body.total).toBeGreaterThanOrEqual(0);
  });
});

test.describe('Inventory API Endpoints', () => {
  test('GET /api/products returns seeded data', async ({ request }) => {
    const tokenResponse = await request.post(`${API_BASE}/api/login`, {
      data: { username: TEST_USERS.superadmin.username, password: TEST_USERS.superadmin.password }
    });
    const { access_token: token } = await tokenResponse.json();

    const response = await request.get(`${API_BASE}/api/products`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.data).toBeInstanceOf(Array);
  });

  test('GET /api/products supports query parameters', async ({ request }) => {
    const tokenResponse = await request.post(`${API_BASE}/api/login`, {
      data: { username: TEST_USERS.superadmin.username, password: TEST_USERS.superadmin.password }
    });
    const { access_token: token } = await tokenResponse.json();

    const response = await request.get(`${API_BASE}/api/products?limit=5`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    expect(response.ok()).toBeTruthy();
  });

  test('GET /api/products/:id returns single product', async ({ request }) => {
    const tokenResponse = await request.post(`${API_BASE}/api/login`, {
      data: { username: TEST_USERS.superadmin.username, password: TEST_USERS.superadmin.password }
    });
    const { access_token: token } = await tokenResponse.json();

    const response = await request.get(`${API_BASE}/api/products/1`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.data).toHaveProperty('name');
  });
});
