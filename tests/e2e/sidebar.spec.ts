import { test, expect } from './fixtures';
import { TEST_USERS, loginUI, logoutUI } from './fixtures';

// NOTE: the *permission grants* that gate this navigation live in
// rbac-api.spec.ts (tested at the API layer). These browser tests only cover
// the genuine UI behaviour: that the nav renders from those grants, that hidden
// items are actually hidden, that groups expand, and that clicks navigate.
test.describe('Sidebar RBAC Visibility (UI rendering)', () => {
  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('superadmin sees all navigation items', async ({ page }) => {
    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    await page.goto('http://localhost:5173/');
    const sidebar = page.locator('aside');
    await expect(sidebar).toBeVisible();

    await expect(sidebar.getByRole('button', { name: 'Dashboard' })).toBeVisible();
    await expect(sidebar.getByRole('button', { name: 'Point of Sale' })).toBeVisible();
    await expect(sidebar.getByRole('button', { name: 'Transaction History' })).toBeVisible();
    await expect(sidebar.getByRole('button', { name: 'Reports' })).toBeVisible();
    await expect(sidebar.getByRole('button', { name: 'Master Data' })).toBeVisible();
    await expect(sidebar.getByRole('button', { name: 'Administration' })).toBeVisible();
  });

  test('cashier sees POS and transactions but not Dashboard', async ({ page }) => {
    await loginUI(page, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
    await page.goto('http://localhost:5173/');
    const sidebar = page.locator('aside');
    await expect(sidebar).toBeVisible();

    await expect(sidebar.getByRole('button', { name: 'Dashboard' })).toHaveCount(0);
    await expect(sidebar.getByRole('button', { name: 'Point of Sale' })).toBeVisible();
    await expect(sidebar.getByRole('button', { name: 'Transaction History' })).toBeVisible();
    await expect(sidebar.getByRole('button', { name: 'Shift Management' })).toBeVisible();
  });

  test('cashier does not see Store Management navigation item', async ({ page }) => {
    await loginUI(page, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
    await page.goto('http://localhost:5173/');
    const sidebar = page.locator('aside');
    await expect(sidebar).toBeVisible();

    await expect(sidebar.getByRole('button', { name: 'Administration' })).toHaveCount(0);
    await expect(sidebar.getByRole('button', { name: 'Store Management' })).toHaveCount(0);
  });

  test('superadmin sees Stores under Administration (group expansion)', async ({ page }) => {
    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    await page.goto('http://localhost:5173/');
    const sidebar = page.locator('aside');
    await expect(sidebar).toBeVisible();

    await sidebar.getByRole('button', { name: 'Administration' }).click();
    await expect(sidebar.getByRole('button', { name: 'Store Management' })).toBeVisible();
  });

  test('sidebar navigates to correct pages', async ({ page }) => {
    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    await page.goto('http://localhost:5173/');

    const sidebar = page.locator('aside');
    const masterDataBtn = sidebar.getByRole('button', { name: 'Master Data' });
    await masterDataBtn.click();

    await sidebar.getByRole('button', { name: 'Products' }).click();
    await expect(page).toHaveURL(/\/inventory\/products/);
  });
});
