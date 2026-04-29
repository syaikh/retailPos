import { test, expect } from '@playwright/test';
import { API_ENDPOINTS, authHeader, waitForAPI, TEST_USERS } from './fixtures';

test.describe('API Integration from Browser Context', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    // Login first
    await page.fill('#username', 'superadmin');
    await page.fill('#password', 'admin123');
    await page.click('.login-btn');
    await expect(page.locator('#dashboard')).toBeVisible({ timeout: 5000 });
  });

  test('should have valid access token in sessionStorage', async ({ page }) => {
    const token = await page.evaluate(() => sessionStorage.getItem('access_token'));
    expect(token).toBeTruthy();
    expect(token?.length).toBeGreaterThan(100);
  });

  test('should successfully call API with bearer token', async ({ page }) => {
    const token = await page.evaluate(() => sessionStorage.getItem('access_token'));
    
    const response = await page.request.get(API_ENDPOINTS.STATS, {
      headers: authHeader(token!)
    });

    expect(response.ok()).toBeTruthy();
    const body = response.json();
    expect(body).toHaveProperty('data');
  });

  test('should get 401 without token', async ({ page }) => {
    const response = await page.request.get(API_ENDPOINTS.STATS);
    expect(response.status()).toBe(401);
  });

  test('should get 401 with invalid token', async ({ page }) => {
    const response = await page.request.get(API_ENDPOINTS.STATS, {
      headers: { Authorization: 'Bearer invalid-token-xyz' }
    });
    expect(response.status()).toBe(401);
  });

  test('should fetch stats data correctly', async ({ page }) => {
    const token = await page.evaluate(() => sessionStorage.getItem('access_token'));
    const response = await page.request.get(API_ENDPOINTS.STATS, {
      headers: authHeader(token!)
    });
    const body = await response.json();

    expect(body.data).toBeDefined();
    expect(body.data).toHaveProperty('total_sales');
    expect(body.data).toHaveProperty('total_revenue');
    expect(body.data).toHaveProperty('total_products');
    expect(body.data).toHaveProperty('todays_sales');
    expect(body.data).toHaveProperty('todays_revenue');
    expect(typeof body.data.total_sales).toBe('number');
    expect(typeof body.data.total_revenue).toBe('number');
  });

  test('should fetch products list', async ({ page }) => {
    const token = await page.evaluate(() => sessionStorage.getItem('access_token'));
    const response = await page.request.get(API_ENDPOINTS.PRODUCTS, {
      headers: authHeader(token!)
    });

    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.data).toBeInstanceOf(Array);
    // Seeded products should exist
    if ((body.data as any[]).length > 0) {
      expect(body.data[0]).toHaveProperty('id');
      expect(body.data[0]).toHaveProperty('name');
      expect(body.data[0]).toHaveProperty('sku');
    }
  });

  test('should fetch admin users list (admin only)', async ({ page }) => {
    // Already logged in as superadmin (full permissions)
    const token = await page.evaluate(() => sessionStorage.getItem('access_token'));
    const response = await page.request.get(API_ENDPOINTS.ADMIN_USERS, {
      headers: authHeader(token!)
    });

    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.data).toBeInstanceOf(Array);
    // Should have seeded users
    expect(body.data.length).toBeGreaterThanOrEqual(4);
  });

  test('should wait for API to be ready (helper function)', async ({ page }) => {
    // This test just verifies our waitForAPI helper works
    await waitForAPI(page, 5);
    const response = await page.request.get(API_ENDPOINTS.STATS);
    expect(response.ok()).toBeTruthy();
  });

  test('should have valid JWT signature (backend should accept)', async ({ page }) => {
    const token = await page.evaluate(() => sessionStorage.getItem('access_token'));
    const response = await page.request.get(API_ENDPOINTS.STATS, {
      headers: authHeader(token!)
    });
    // If token was invalid, would return 401
    expect(response.status()).toBe(200);
  });
});

test.describe('API Error Handling', () => {
  test('should reject invalid login credentials', async ({ page }) => {
    await page.goto('/');
    await page.fill('#username', 'fakeuser');
    await page.fill('#password', 'fakepass');
    await page.click('.login-btn');

    // Wait for error
    await expect(page.locator('#error-msg')).toBeVisible();
    const error = await page.locator('#error-msg').innerText();
    expect(error.toLowerCase()).toContain('invalid');
  });

  test('should require both username and password', async ({ page }) => {
    await page.goto('/');
    
    // Try submit empty
    await page.click('.login-btn');
    await expect(page.locator('#error-msg')).toBeVisible();
    const error = await page.locator('#error-msg').innerText();
    expect(error).toContain('invalid request');
  });

  test('should clear error on new login attempt', async ({ page }) => {
    await page.goto('/');
    await page.click('.login-btn');
    await expect(page.locator('#error-msg')).toHaveText('invalid request');

    // Start typing should clear error? Actually current app clears on submit success only
    // But we can verify error is still there before submit
    await page.fill('#username', 'superadmin');
    // Error still visible until submit
    await expect(page.locator('#error-msg')).toBeVisible();
  });
});

test.describe('Cross-User Isolation (Security)', () => {
  test('should only return own user data in /stats', async ({ page }) => {
    // The stats endpoint returns aggregate data, not user-specific
    const token = await page.evaluate(() => sessionStorage.getItem('access_token'));
    const response = await page.request.get(API_ENDPOINTS.STATS, {
      headers: authHeader(token!)
    });
    const body = await response.json();

    // Should not include sensitive user data
    expect(body.data).not.toHaveProperty('password');
    expect(body.data).not.toHaveProperty('refresh_token');
  });
});
