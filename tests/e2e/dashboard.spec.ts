import { test, expect } from '@playwright/test';

test.describe('Dashboard (Home Page)', () => {
  test.beforeEach(async ({ page }) => {
    // Login first - navigate to login page directly
    await page.goto('http://localhost:5173/login');
    await page.fill('#username', 'superadmin');
    await page.fill('#password', 'admin123');
    await page.click('.login-btn');
    // Wait for dashboard to load
    await expect(page).toHaveURL(/\/$/, { timeout: 5000 });
    await expect(page.locator('#dashboard')).toBeVisible({ timeout: 5000 });
  });

  test('should display dashboard header', async ({ page }) => {
    await expect(page.locator('h1')).toHaveText('Retail POS System');
    await expect(page.locator('header p')).toHaveText('Modern Point of Sale Management');
  });

  test('should display all 4 feature cards with correct content', async ({ page }) => {
    const cards = page.locator('.card');
    await expect(cards).toHaveCount(4);

    // POS Card
    await expect(cards.nth(0).locator('h3')).toHaveText('Point of Sale');
    await expect(cards.nth(0).locator('p')).toHaveText('Process customer transactions and manage sales');
    await expect(cards.nth(0).locator('.btn')).toHaveText('Open POS');

    // Inventory Card
    await expect(cards.nth(1).locator('h3')).toHaveText('Inventory');
    await expect(cards.nth(1).locator('p')).toHaveText('Manage products, stock levels, and categories');
    await expect(cards.nth(1).locator('.btn')).toHaveText('View Inventory');

    // Reports Card
    await expect(cards.nth(2).locator('h3')).toHaveText('Reports');
    await expect(cards.nth(2).locator('p')).toHaveText('View sales analytics and business insights');
    await expect(cards.nth(2).locator('.btn')).toHaveText('View Reports');

    // Admin Card
    await expect(cards.nth(3).locator('h3')).toHaveText('Administration');
    await expect(cards.nth(3).locator('p')).toHaveText('Manage users, roles, and system settings');
    await expect(cards.nth(3).locator('.btn')).toHaveText('Open Admin');
  });

  test('should display system status banner', async ({ page }) => {
    await expect(page.locator('.status h3')).toHaveText('✅ System Status: Operational');
  });

  test('should maintain user session after page reload', async ({ page }) => {
    const tokenBefore = await page.evaluate(() => sessionStorage.getItem('access_token'));
    expect(tokenBefore).toBeTruthy();

    await page.reload();
    await expect(page.locator('#dashboard')).toBeVisible({ timeout: 3000 });

    const tokenAfter = await page.evaluate(() => sessionStorage.getItem('access_token'));
    expect(tokenAfter).toBe(tokenBefore);
  });

  test('should persist JWT tokens', async ({ page }) => {
    const storage = await page.evaluate(() => ({
      access: sessionStorage.getItem('access_token'),
      refresh: sessionStorage.getItem('refresh_token')
    }));
    expect(storage.access).toBeTruthy();
    expect(storage.refresh).toBeTruthy();
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
    expect(payload.role_id).toBe(1);
    expect(payload.exp).toBeGreaterThan(Math.floor(Date.now() / 1000));
  });

  test('should logout and redirect to login', async ({ page }) => {
    // Clear session and reload
    await page.evaluate(() => sessionStorage.clear());
    await page.reload();
    // Should be redirected to login
    await expect(page).toHaveURL(/\/login$/);
    await expect(page.locator('#login-section')).toBeVisible();
    await expect(page.locator('#dashboard')).toBeHidden();
  });
});
