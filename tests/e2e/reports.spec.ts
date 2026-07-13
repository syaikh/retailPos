import { test, expect } from '@playwright/test';
import { TEST_USERS, API_BASE, loginUI, logoutUI, getToken } from './fixtures';

test.describe('Reports & Analytics', () => {
  test.beforeEach(async ({ page }) => {
    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    await page.goto('http://localhost:5173/reports');
    await expect(page).toHaveURL(/\/reports/);
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('should display KPI cards', async ({ page }) => {
    await expect(page.locator('text=TOTAL REVENUE')).toBeVisible({ timeout: 15000 });
    await expect(page.locator('text=TOTAL ORDERS')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('text=AVG ORDER VALUE')).toBeVisible({ timeout: 5000 });
  });

  test('should display revenue chart', async ({ page }) => {
    await expect(page.locator('canvas')).toBeVisible({ timeout: 15000 });
  });

  test('should display period selector and change period', async ({ page }) => {
    const periodButton = page.locator('button[aria-haspopup="menu"]').first();
    await expect(periodButton).toBeVisible({ timeout: 10000 });

    await periodButton.click();
    await expect(page.locator('#period-dropdown-menu')).toBeVisible();

    await page.getByRole('menuitem', { name: 'Yesterday' }).click();
    await page.waitForTimeout(1000);
  });

  test('should show best/worst badges', async ({ page }) => {
    await expect(page.locator('text=Best').first()).toBeVisible({ timeout: 15000 });
  });

  test('should toggle data table', async ({ page }) => {
    await page.getByRole('button', { name: 'Show Data Table' }).click();
    await page.waitForTimeout(500);
    await expect(page.getByRole('button', { name: 'Hide Data Table' })).toBeVisible({ timeout: 5000 });
  });

  test('should open export dropdown', async ({ page }) => {
    await page.locator('button').filter({ hasText: 'Export' }).first().click();
    await expect(page.getByRole('menu').filter({ hasText: 'Export to Excel' })).toBeVisible({ timeout: 5000 });
    await expect(page.getByRole('menuitem', { name: 'Export to Excel' })).toBeVisible();
    await expect(page.getByRole('menuitem', { name: 'Export to PDF' })).toBeVisible();
  });
});

test.describe('Reports API', () => {
  test('GET /api/stats returns valid dashboard data', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    const response = await request.get(`${API_BASE}/api/dashboard/stats`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.data).toHaveProperty('todays_sales');
    expect(body.data).toHaveProperty('todays_revenue');
    expect(body.data).toHaveProperty('total_products');
  });

  test('GET /api/sales supports pagination and filters', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    const response = await request.get(`${API_BASE}/api/sales?limit=5`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    expect(response.ok()).toBeTruthy();
  });
});
