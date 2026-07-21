import { test, expect } from '@playwright/test';
import { TEST_USERS, API_BASE, loginUI, logoutUI, getToken } from './fixtures';

test.describe('Shifts Page', () => {
  test.beforeEach(async ({ page }) => {
    await loginUI(page, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('should load shift table without getting stuck on loading', async ({ page }) => {
    await page.goto('/shifts');
    await expect(page).toHaveURL(/\/shifts/);

    const loadingSpinner = page.locator('text=Loading shifts...');
    await expect(loadingSpinner).toBeVisible({ timeout: 5000 });

    await expect(loadingSpinner).toBeHidden({ timeout: 15000 });

    await expect(page.locator('text=No shifts found').or(page.locator('table tbody tr'))).toBeVisible({ timeout: 5000 });
  });

  test('should show shifts table with correct columns for cashier', async ({ page }) => {
    await page.goto('/shifts');
    await expect(page).toHaveURL(/\/shifts/);

    await expect(page.locator('text=Loading shifts...')).toBeHidden({ timeout: 15000 });

    await expect(page.locator('text=OPENED AT')).toBeVisible();
    await expect(page.locator('text=OPENING (Rp)')).toBeVisible();
    await expect(page.locator('text=CASH SALES (Rp)')).toBeVisible();
    await expect(page.locator('text=TOTAL SALES (Rp)')).toBeVisible();
    await expect(page.locator('text=TXN')).toBeVisible();
    await expect(page.locator('text=DISCREPANCY (Rp)')).toBeVisible();
    await expect(page.locator('text=STATUS')).toBeVisible();
    await expect(page.locator('text=CLOSED AT')).toBeVisible();

    await expect(page.locator('text=CASHIER')).toBeHidden();
  });

  test('should open shift modal and display active shift banner', async ({ page, request }) => {
    const token = await getToken(request, TEST_USERS.cashier.username, TEST_USERS.cashier.password);

    await page.request.post(`${API_BASE}/api/shifts/close-all`, {
      headers: { Authorization: `Bearer ${token}` },
    }).catch(() => {});

    await page.goto('/shifts');
    await expect(page).toHaveURL(/\/shifts/);
    await expect(page.locator('text=Loading shifts...')).toBeHidden({ timeout: 15000 });

    const openBtn = page.locator('button').filter({ hasText: 'Open Shift' });
    await expect(openBtn).toBeEnabled({ timeout: 5000 });
    await openBtn.click();

    await expect(page.getByRole('dialog', { name: 'Open Shift' })).toBeVisible();

    await page.locator('#opening-balance').fill('100000');
    await page.locator('button').filter({ hasText: 'Open Shift' }).click();

    await expect(page.locator('text=Active Shift')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('text=Loading shifts...')).toBeHidden({ timeout: 5000 });
  });

  test('loading state does not persist after navigating to shifts page', async ({ page }) => {
    await page.goto('/shifts');
    await expect(page).toHaveURL(/\/shifts/);

    const loadingSpinner = page.locator('text=Loading shifts...');
    await expect(loadingSpinner).toBeVisible({ timeout: 5000 });
    await expect(loadingSpinner).toBeHidden({ timeout: 15000 });

    await page.goto('/pos');
    await expect(page).toHaveURL(/\/pos/);

    await page.goto('/shifts');
    await expect(page).toHaveURL(/\/shifts/);
    await expect(loadingSpinner).toBeHidden({ timeout: 15000 });

    await expect(page.locator('text=OPENED AT')).toBeVisible({ timeout: 5000 });
  });

  test('admin sees CASHIER column and all shifts', async ({ page }) => {
    await logoutUI(page);
    await loginUI(page, TEST_USERS.admin.username, TEST_USERS.admin.password);

    await page.goto('/shifts');
    await expect(page).toHaveURL(/\/shifts/);
    await expect(page.locator('text=Loading shifts...')).toBeHidden({ timeout: 15000 });

    await expect(page.locator('th').filter({ hasText: 'CASHIER' })).toBeVisible();
  });

  test('superadmin sees CASHIER column and all shifts', async ({ page }) => {
    await logoutUI(page);
    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    await page.goto('/shifts');
    await expect(page).toHaveURL(/\/shifts/);
    await expect(page.locator('text=Loading shifts...')).toBeHidden({ timeout: 15000 });

    await expect(page.locator('th').filter({ hasText: 'CASHIER' })).toBeVisible();
  });
});

test.describe('Shifts API - cashier_id filter', () => {
  test('should list shifts filtered by cashier_id for cashier role', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.cashier.username, TEST_USERS.cashier.password);

    const res = await request.get(`${API_BASE}/api/shifts?limit=10&offset=0&sort_by=opened_at&sort_dir=desc&user_id=4`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();

    expect(body).toHaveProperty('data');
    expect(Array.isArray(body.data)).toBeTruthy();
    expect(body).toHaveProperty('total');
  });

  test('superadmin sees all shifts via API', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    const res = await request.get(`${API_BASE}/api/shifts?limit=10&offset=0&sort_by=opened_at&sort_dir=desc`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body).toHaveProperty('data');
  });
});
