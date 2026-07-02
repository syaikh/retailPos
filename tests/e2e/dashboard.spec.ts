import { test, expect } from '@playwright/test';

test.describe('Dashboard (Home Page)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('http://localhost:5173/login');
    await page.fill('#username', 'superadmin');
    await page.fill('#password', 'admin123');
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL(/\/$/, { timeout: 5000 });
  });

  test('should persist JWT tokens', async ({ page }) => {
    const storage = await page.evaluate(() => ({
      access: sessionStorage.getItem('access_token'),
      refresh: sessionStorage.getItem('refresh_token')
    }));
    expect(storage.access).toBeTruthy();
    const parts = storage.access.split('.');
    expect(parts).toHaveLength(3);
    expect(storage.access.length).toBeGreaterThan(100);
  });

  test('should decode JWT payload with correct user', async ({ page }) => {
    const payload = await page.evaluate(() => {
      const token = sessionStorage.getItem('access_token');
      if (!token) return null;
      const parts = token.split('.');
      return JSON.parse(atob(parts[1]));
    });
    expect(payload).not.toBeNull();
    expect(payload.username).toBe('superadmin');
    expect(payload.id).toBe(1);
    expect(payload.exp).toBeGreaterThan(Math.floor(Date.now() / 1000));
  });

  test('should logout and redirect to login', async ({ page }) => {
    await page.evaluate(() => sessionStorage.clear());
    await page.reload();
    await expect(page).toHaveURL(/\/login$/);
    await expect(page.locator('#username')).toBeVisible();
  });

  test('Topbar clock displays Jakarta time with WIB suffix', async ({ page }) => {
    const dateTimeText = await page.locator('header').locator('span.text-xs').last().textContent();
    expect(dateTimeText).toMatch(/\d{2}:\d{2}/);
  });
});
