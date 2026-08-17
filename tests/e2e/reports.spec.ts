import { test, expect } from './fixtures';
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
    const periodButton = page.locator('button[aria-controls="period-dropdown-menu"]');
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

  test('should change period to 7 Days without errors', async ({ page }) => {
    const periodButton = page.locator('button[aria-controls="period-dropdown-menu"]');
    await expect(periodButton).toBeVisible({ timeout: 10000 });

    await periodButton.click();
    await expect(page.locator('#period-dropdown-menu')).toBeVisible();

    await page.getByRole('menuitem', { name: '7 Days' }).click();
    await page.waitForTimeout(1500);

    const errorToast = page.locator('text=Failed to load report data');
    await expect(errorToast).toBeHidden({ timeout: 3000 });
  });

  test('should change period to 30 Days without errors', async ({ page }) => {
    const periodButton = page.locator('button[aria-controls="period-dropdown-menu"]');
    await expect(periodButton).toBeVisible({ timeout: 10000 });

    await periodButton.click();
    await expect(page.locator('#period-dropdown-menu')).toBeVisible();

    await page.getByRole('menuitem', { name: '30 Days' }).click();
    await page.waitForTimeout(1500);

    const errorToast = page.locator('text=Failed to load report data');
    await expect(errorToast).toBeHidden({ timeout: 3000 });
  });

  test('should change period to Monthly without errors', async ({ page }) => {
    const periodButton = page.locator('button[aria-controls="period-dropdown-menu"]');
    await expect(periodButton).toBeVisible({ timeout: 10000 });

    await periodButton.click();
    await expect(page.locator('#period-dropdown-menu')).toBeVisible();

    await page.getByRole('menuitem', { name: 'Monthly' }).click();
    await page.waitForTimeout(1500);

    const errorToast = page.locator('text=Failed to load report data');
    await expect(errorToast).toBeHidden({ timeout: 3000 });
  });

  test('should display chart canvas after period change', async ({ page }) => {
    const periodButton = page.locator('button[aria-controls="period-dropdown-menu"]');
    await expect(periodButton).toBeVisible({ timeout: 10000 });

    await periodButton.click();
    await page.getByRole('menuitem', { name: '7 Days' }).click();
    await page.waitForTimeout(2000);

    await expect(page.locator('canvas')).toBeVisible({ timeout: 10000 });
  });

  test('should load KPI values from dashboard stats API', async ({ page }) => {
    let statsPayload: any = null;
    page.on('response', async (response) => {
      if (response.url().includes('/api/dashboard/comparison')) {
        statsPayload = await response.json().catch(() => null);
      }
    });

    await page.reload();
    await page.waitForTimeout(3000);

    expect(statsPayload).toBeTruthy();
    const data = statsPayload?.data || statsPayload;
    expect(data).toHaveProperty('current_revenue');
    expect(data).toHaveProperty('current_orders');
    expect(typeof data.current_revenue).toBe('number');
    expect(typeof data.current_orders).toBe('number');
  });

  test('7 Days period returns 7 data points from chart API', async ({ page }) => {
    let chartPayload: any = null;
    page.on('response', async (response) => {
      if (response.url().includes('/api/dashboard/chart') && !response.url().includes('monthly')) {
        chartPayload = await response.json().catch(() => null);
      }
    });

    const periodButton = page.locator('button[aria-controls="period-dropdown-menu"]');
    await expect(periodButton).toBeVisible({ timeout: 10000 });

    await periodButton.click();
    await page.getByRole('menuitem', { name: '7 Days' }).click();
    await page.waitForTimeout(3000);

    expect(chartPayload).toBeTruthy();
    const currentData = chartPayload?.data?.current || chartPayload?.current || [];
    expect(currentData.length).toBe(7);
  });

  test('Best/Worst badges display period labels from API data', async ({ page }) => {
    const periodButton = page.locator('button[aria-controls="period-dropdown-menu"]');
    await expect(periodButton).toBeVisible({ timeout: 10000 });

    await periodButton.click();
    await page.getByRole('menuitem', { name: '7 Days' }).click();
    await page.waitForTimeout(3000);

    const bestBadge = page.locator('text=Best').first();
    await expect(bestBadge).toBeVisible({ timeout: 10000 });

    const badgeText = await bestBadge.textContent();
    expect(badgeText).toBeTruthy();
    expect(badgeText!.length).toBeGreaterThan(5);
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
