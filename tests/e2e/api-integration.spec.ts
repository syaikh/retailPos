import { test, expect } from '@playwright/test';
import { authHeader, waitForAPI, TEST_USERS } from './fixtures';

test.describe('API Integration (Backend)', () => {
  test('should successfully call API with bearer token', async ({ page }) => {
    // Login first to get token
    await page.goto('http://localhost:5173/login');
    await page.fill('#username', TEST_USERS.superadmin.username);
    await page.fill('#password', TEST_USERS.superadmin.password);
    await page.click('.login-btn');
    await expect(page).toHaveURL(/\/$/, { timeout: 5000 });

    const token = await page.evaluate(() => sessionStorage.getItem('access_token'));
    expect(token).toBeTruthy();

    const response = await page.request.get('http://localhost:5173/api/stats', {
      headers: authHeader(token)
    });
    expect(response.ok()).toBeTruthy();
  });

  test('should fetch admin users list (admin only)', async ({ page }) => {
    await page.goto('http://localhost:5173/login');
    await page.fill('#username', TEST_USERS.superadmin.username);
    await page.fill('#password', TEST_USERS.superadmin.password);
    await page.click('.login-btn');
    await expect(page).toHaveURL(/\/$/, { timeout: 5000 });

    const token = await page.evaluate(() => sessionStorage.getItem('access_token'));
    const response = await page.request.get('http://localhost:5173/api/admin/users', {
      headers: authHeader(token)
    });
    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.data).toBeInstanceOf(Array);
    expect(body.data.length).toBeGreaterThanOrEqual(4);
  });

  test('should wait for API to be ready', async ({ page }) => {
    await waitForAPI(page, 10);
    const response = await page.request.get('http://localhost:5173/api/stats');
    expect(response.ok()).toBeTruthy();
  });

  test.describe('API Error Handling', () => {
    test('should reject invalid login credentials', async ({ page }) => {
      await page.goto('http://localhost:5173/login');
      await page.fill('#username', 'fakeuser');
      await page.fill('#password', 'fakepass');
      await page.click('.login-btn');

      await expect(page.locator('#error-msg')).toBeVisible();
      const error = await page.locator('#error-msg').innerText();
      expect(error.toLowerCase()).toContain('invalid');
    });

    test('should require both username and password', async ({ page }) => {
      await page.goto('http://localhost:5173/login');
      await page.click('.login-btn');
      await expect(page.locator('#error-msg')).toBeVisible();
      const error = await page.locator('#error-msg').innerText();
      expect(error.toLowerCase()).toContain('invalid request');
    });

    test('should clear error on successful login', async ({ page }) => {
      await page.goto('http://localhost:5173/login');
      // Trigger error
      await page.click('.login-btn');
      await expect(page.locator('#error-msg')).toBeVisible();

      // Now login properly
      await page.fill('#username', TEST_USERS.superadmin.username);
      await page.fill('#password', TEST_USERS.superadmin.password);
      await page.click('.login-btn');

      await expect(page.locator('#dashboard')).toBeVisible({ timeout: 5000 });
      await expect(page.locator('#error-msg')).not.toBeVisible();
    });
  });

  test.describe('Security & Isolation', () => {
    test('should not expose sensitive data in /stats', async ({ page }) => {
      await page.goto('http://localhost:5173/login');
      await page.fill('#username', TEST_USERS.superadmin.username);
      await page.fill('#password', TEST_USERS.superadmin.password);
      await page.click('.login-btn');
      await expect(page).toHaveURL(/\/$/, { timeout: 5000 });

      const token = await page.evaluate(() => sessionStorage.getItem('access_token'));
      const response = await page.request.get('http://localhost:5173/api/stats', {
        headers: authHeader(token)
      });
      const body = await response.json();
      expect(body.data).not.toHaveProperty('password');
      expect(body.data).not.toHaveProperty('refresh_token');
    });
  });
});
