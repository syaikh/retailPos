import { test, expect } from '@playwright/test';
import { TEST_USERS, API_BASE } from './fixtures';

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
    await page.click('a[href="/inventory"]');
    await expect(page.locator('text=PRODUCT NAME')).toBeVisible();
    await expect(page.locator('text=No products found')).toBeHidden();
  });

  test('should search products', async ({ page }) => {
    await page.click('a[href="/inventory"]');
    await page.fill('input[placeholder="Search products..."]', 'Indomie');
    await expect(page.locator('text=Indomie')).toBeVisible({ timeout: 5000 });
  });

  test('should filter by single category', async ({ page }) => {
    await page.click('a[href="/inventory"]');
    
    // Click the category filter input to open dropdown
    await page.click('.relative.flex-1');
    
    // Select a category checkbox
    await page.locator('label:has-text("Makanan")').click();
    
    // Wait for filter to apply
    await page.waitForTimeout(500);
    
    // Verify products are filtered (should show only Makanan category products)
    const products = await page.locator('tbody tr').count();
    if (products > 0) {
      await expect(page.locator('text=Makanan').first()).toBeVisible();
    }
  });

  test('should filter by multiple categories', async ({ page }) => {
    await page.click('a[href="/inventory"]');
    
    // Click the category filter input to open dropdown
    await page.click('.relative.flex-1');
    
    // Select multiple category checkboxes
    await page.locator('label:has-text("Makanan")').click();
    await page.locator('label:has-text("Minuman")').click();
    
    // Wait for filter to apply
    await page.waitForTimeout(500);
    
    // Verify both categories show up in selected tags
    await expect(page.locator('text=Makanan')).toBeVisible();
    await expect(page.locator('text=Minuman')).toBeVisible();
  });
});

test.describe('Inventory API - Category Filter', () => {
  test('GET /api/products?category=single returns filtered products', async ({ request }) => {
    const tokenResponse = await request.post(`${API_BASE}/api/login`, {
      data: { username: TEST_USERS.superadmin.username, password: TEST_USERS.superadmin.password }
    });
    const { access_token: token } = await tokenResponse.json();

    const response = await request.get(`${API_BASE}/api/products?category=Makanan`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.data).toBeInstanceOf(Array);
    
    // If there are products, verify they're from Makanan category
    if (body.data.length > 0) {
      for (const product of body.data) {
        expect(product.category_name).toBe('Makanan');
      }
    }
  });

  test('GET /api/products?category=multiple returns products matching any category', async ({ request }) => {
    const tokenResponse = await request.post(`${API_BASE}/api/login`, {
      data: { username: TEST_USERS.superadmin.username, password: TEST_USERS.superadmin.password }
    });
    const { access_token: token } = await tokenResponse.json();

    const response = await request.get(`${API_BASE}/api/products?category=Makanan,Minuman`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.data).toBeInstanceOf(Array);
    
    // Verify products are from either Makanan or Minuman category
    if (body.data.length > 0) {
      const validCategories = ['Makanan', 'Minuman'];
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

    const response = await request.get(`${API_BASE}/api/products?search=&limit=10&offset=0&category=Makanan`, {
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
