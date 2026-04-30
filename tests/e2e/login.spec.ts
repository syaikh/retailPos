import { test, expect } from '@playwright/test';

test.describe('Authentication Flow (SPA)', () => {
  test.beforeEach(async ({ page }) => {
    // Clear storage BEFORE any page script runs
    await page.addInitScript(() => {
      sessionStorage.clear();
      localStorage.clear();
    });
    await page.context().clearPermissions();
    await page.goto('http://localhost:5173/', { waitUntil: 'domcontentloaded' });
  });

  test('should redirect unauthenticated user from "/" to "/login"', async ({ page }) => {
    // After beforeEach, SPA should redirect to /login
    await expect(page).toHaveURL(/\/login$/);
  });

  test('should display login form', async ({ page }) => {
    await expect(page).toHaveURL(/\/login$/);
    await expect(page.locator('#login-section')).toBeVisible();
    await expect(page.locator('text=Login to Retail POS')).toBeVisible();
    await expect(page.locator('#username')).toBeVisible();
    await expect(page.locator('#password')).toBeVisible();
    await expect(page.locator('.login-btn')).toBeVisible();
  });

  test('should login with valid credentials', async ({ page }) => {
    await expect(page).toHaveURL(/\/login$/);
    await page.fill('#username', 'superadmin');
    await page.fill('#password', 'admin123');
    await page.click('.login-btn');

    // After successful login, redirect to '/'
    await expect(page).toHaveURL(/\/$/, { timeout: 15000 });

    // Login section hidden, dashboard visible
    await expect(page.locator('#login-section')).not.toBeVisible({ timeout: 5000 });
    await expect(page.locator('#dashboard')).toBeVisible({ timeout: 5000 });

    // Dashboard header
    await expect(page.locator('h1')).toHaveText('Retail POS System');
    // Cards present
    const dashboard = page.locator('#dashboard');
    await expect(dashboard.locator('h3').first()).toHaveText('Point of Sale');
    await expect(dashboard.locator('h3').nth(1)).toHaveText('Inventory');
    await expect(dashboard.locator('h3').nth(2)).toHaveText('Reports');
    await expect(dashboard.locator('h3').nth(3)).toHaveText('Administration');
  });

  test('should show error for invalid credentials', async ({ page }) => {
    await expect(page).toHaveURL(/\/login$/);
    await page.fill('#username', 'wronguser');
    await page.fill('#password', 'wrongpass');
    await page.click('.login-btn');

    await expect(page.locator('#error-msg')).toBeVisible({ timeout: 5000 });
    const errorText = await page.locator('#error-msg').innerText();
    const hasInvalid = errorText.toLowerCase().includes('invalid username or password');
    const hasNetwork = errorText.includes('Network error. Please try again.');
    expect(hasInvalid || hasNetwork).toBeTruthy();
  });

  test('should clear error on new login attempt', async ({ page }) => {
    await expect(page).toHaveURL(/\/login$/);
    // Trigger error first
    await page.click('.login-btn');
    await expect(page.locator('#error-msg')).toBeVisible();

    // Fill credentials correctly
    await page.fill('#username', 'superadmin');
    await page.fill('#password', 'admin123');
    await page.click('.login-btn');

    // Should succeed
    await expect(page.locator('#dashboard')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('#error-msg')).not.toBeVisible();
  });
});
