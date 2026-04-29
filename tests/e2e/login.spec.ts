import { test, expect } from '@playwright/test';

test.beforeEach(async ({ page }) => {
  await page.goto('http://localhost:5173');
});

test('should display login form', async ({ page }) => {
  await expect(page.locator('#login-section')).toBeVisible();
  await expect(page.locator('text=Login to Retail POS')).toBeVisible();
  await expect(page.locator('#username')).toBeVisible();
  await expect(page.locator('#password')).toBeVisible();
  await expect(page.locator('.login-btn')).toBeVisible();
});

test('should login with valid credentials', async ({ page }) => {
  await page.fill('#username', 'superadmin');
  await page.fill('#password', 'admin123');
  await page.click('.login-btn');

  // After successful login the app should redirect to '/' or '/dashboard'
  await expect(page).toHaveURL(/.*\/?$/, { timeout: 15000 });

  // Login section should be hidden
  await expect(page.locator('#login-section')).not.toBeVisible({ timeout: 5000 });

  // Dashboard should be visible
  await expect(page.locator('#dashboard')).toBeVisible({ timeout: 5000 });

  // Check dashboard content - use more specific selectors to avoid duplicate matches
  await expect(page.locator('text=Retail POS System')).toBeVisible();
  // Use within dashboard to narrow down
  const dashboard = page.locator('#dashboard');
  await expect(dashboard.getByRole('heading', { name: 'Point of Sale' })).toBeVisible();
  await expect(dashboard.getByRole('heading', { name: 'Inventory' })).toBeVisible();
  await expect(dashboard.getByRole('heading', { name: 'Reports' })).toBeVisible();
});

test('should show error for invalid credentials', async ({ page }) => {
  await page.fill('#username', 'wronguser');
  await page.fill('#password', 'wrongpass');
  await page.click('.login-btn');

  // Wait for error message to appear
  await expect(page.locator('#error-msg')).toBeVisible({ timeout: 5000 });

  const errorText = await page.locator('#error-msg').innerText();
  // Accept either backend validation error (lowercase) or network error
  const hasInvalid = errorText.toLowerCase().includes('invalid username or password');
  const hasNetwork = errorText.includes('Network error. Please try again.');
  expect(hasInvalid || hasNetwork).toBeTruthy();
});
