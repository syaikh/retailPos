import { test, expect, type Page } from '@playwright/test';
import { TEST_USERS, API_BASE } from './fixtures';

async function navigateToStock(page: Page) {
  const sidebar = page.locator('aside');
  const stockBtn = sidebar.locator('button', { hasText: 'Stock' }).first();
  await stockBtn.click();
  await expect(page.locator('text=PRODUCT NAME')).toBeVisible({ timeout: 10000 });
}

test.describe('Stock Management', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.fill('#username', TEST_USERS.superadmin.username);
    await page.fill('#password', TEST_USERS.superadmin.password);
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL(/\/$/, { timeout: 5000 });
  });

  test('should load stock page with product table', async ({ page }) => {
    await navigateToStock(page);
    await expect(page.locator('text=PRODUCT NAME')).toBeVisible();
    await expect(page.locator('text=No stock data found')).toBeHidden();
  });

  test('should show stock values in table', async ({ page }) => {
    await navigateToStock(page);
    await page.waitForTimeout(2000);
    const stockCells = page.locator('table tbody tr td:nth-child(3)');
    const cellCount = await stockCells.count();
    expect(cellCount).toBeGreaterThan(0);
    for (let i = 0; i < Math.min(cellCount, 5); i++) {
      const cellText = await stockCells.nth(i).textContent();
      expect(parseInt(cellText ?? '0', 10)).toBeGreaterThanOrEqual(0);
    }
  });

  test('should show stock status badges', async ({ page }) => {
    await navigateToStock(page);
    await page.waitForTimeout(2000);
    const statusCells = page.locator('table tbody tr td:nth-child(4)');
    const cellCount = await statusCells.count();
    expect(cellCount).toBeGreaterThan(0);
  });

  test('should toggle low stock filter', async ({ page }) => {
    await navigateToStock(page);
    const lowStockBtn = page.locator('button').filter({ hasText: 'Low Stock' }).first();
    await lowStockBtn.click();
    await page.waitForTimeout(1000);
    await lowStockBtn.click();
    await page.waitForTimeout(1000);
  });

  test('should open stock adjustment modal', async ({ page }) => {
    await navigateToStock(page);
    await page.waitForTimeout(2000);
    const adjustBtn = page.locator('table tbody tr:first-child td:last-child button').first();
    if (await adjustBtn.isVisible()) {
      await adjustBtn.click();
      await expect(page.locator('text=Adjust Stock')).toBeVisible({ timeout: 5000 });
    }
  });

  test('should adjust stock with valid input', async ({ page }) => {
    await navigateToStock(page);
    await page.waitForTimeout(2000);
    const adjustBtn = page.locator('table tbody tr:first-child td:last-child button').first();
    if (await adjustBtn.isVisible()) {
      await adjustBtn.click();
      await expect(page.locator('text=Adjust Stock')).toBeVisible({ timeout: 5000 });
      await page.fill('#adjust-qty', '10');
      await page.fill('#adjust-notes', 'Restocking from supplier');
      await page.click('button', { hasText: 'Adjust Stock' });
      await expect(page.locator('text=Stock adjusted successfully')).toBeVisible({ timeout: 5000 });
    }
  });

  test('should search products on stock page', async ({ page }) => {
    await navigateToStock(page);
    await page.fill('input[placeholder="Search products by name or SKU..."]', 'Quality');
    await page.waitForTimeout(1000);
  });

  test('should NOT have Add Product button on stock page', async ({ page }) => {
    await navigateToStock(page);
    await expect(page.locator('button', { hasText: 'Add Product' })).toBeHidden();
  });

  test('should NOT have PRICE column in stock table', async ({ page }) => {
    await navigateToStock(page);
    await expect(page.locator('text=PRICE')).toBeHidden();
  });
});

test.describe('Stock API Endpoints', () => {
  test('GET /api/products returns stock data', async ({ request }) => {
    const tokenResponse = await request.post(`${API_BASE}/api/login`, {
      data: { username: TEST_USERS.superadmin.username, password: TEST_USERS.superadmin.password }
    });
    const { access_token: token } = await tokenResponse.json();
    const response = await request.get(`${API_BASE}/api/products?limit=200`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.data).toBeInstanceOf(Array);
    expect(body.data.length).toBeGreaterThan(0);
    for (const product of body.data) {
      expect(product.stock).toBeGreaterThan(0);
    }
  });

  test('GET /api/stock-thresholds returns threshold config', async ({ request }) => {
    const tokenResponse = await request.post(`${API_BASE}/api/login`, {
      data: { username: TEST_USERS.superadmin.username, password: TEST_USERS.superadmin.password }
    });
    const { access_token: token } = await tokenResponse.json();
    const response = await request.get(`${API_BASE}/api/stock-thresholds`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body).toHaveProperty('warning');
    expect(body).toHaveProperty('critical');
  });
});
