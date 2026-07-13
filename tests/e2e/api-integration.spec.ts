import { test, expect } from '@playwright/test';
import { authHeader, TEST_USERS, FRONTEND_BASE, API_BASE, loginUI, getToken } from './fixtures';

test.describe('API Integration (Backend)', () => {
  test('should successfully call API with bearer token', async ({ page, request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    expect(token).toBeTruthy();

    const response = await page.request.get(`${API_BASE}/health`, {
      headers: authHeader(token)
    });
    expect(response.ok()).toBeTruthy();
  });

  test('should fetch admin users list (admin only)', async ({ page, request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const response = await page.request.get(`${API_BASE}/api/admin/users`, {
      headers: authHeader(token)
    });
    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.data).toBeInstanceOf(Array);
    expect(body.data.length).toBeGreaterThanOrEqual(1);
  });

  test('should wait for API to be ready', async ({ page }) => {
    const response = await page.request.get(`${API_BASE}/health`);
    expect(response.ok()).toBeTruthy();
  });

  test.describe('API Error Handling', () => {
    test('should reject invalid login credentials', async ({ page }) => {
      await page.goto(`${FRONTEND_BASE}/login`);
      await page.fill('#username', 'fakeuser');
      await page.fill('#password', 'fakepass');
      await page.click('button[type="submit"]');

      await expect(page.locator('text=Invalid username or password')).toBeVisible();
    });

    test('should require both username and password', async ({ page }) => {
      await page.goto(`${FRONTEND_BASE}/login`);
      await page.click('button[type="submit"]');
      await expect(page.locator('text=Username and password are required')).toBeVisible();
    });

    test('should clear error on successful login', async ({ page }) => {
      await page.goto(`${FRONTEND_BASE}/login`);
      // Trigger error first (empty password)
      await page.click('button[type="submit"]');
      await expect(page.locator('text=Username and password are required')).toBeVisible();

      // Now login properly
      await page.fill('#username', TEST_USERS.superadmin.username);
      await page.fill('#password', TEST_USERS.superadmin.password);
      await page.click('button[type="submit"]');

      // Should succeed - wait for URL to change to home
      await page.waitForURL(/\/$/, { timeout: 10000 });
      expect(page.url()).toMatch(/\/$/);
    });
  });

  test.describe('Security & Isolation', () => {
    test('should not expose sensitive data in health endpoint', async ({ page, request }) => {
      const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
      const response = await page.request.get(`${API_BASE}/health`, {
        headers: authHeader(token)
      });
      const body = await response.json();
      expect(body).toHaveProperty('status');
      expect(body).not.toHaveProperty('password');
      expect(body).not.toHaveProperty('refresh_token');
    });
  });
});
