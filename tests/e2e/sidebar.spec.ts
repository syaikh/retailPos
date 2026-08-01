import { test, expect } from '@playwright/test';
import { TEST_USERS, loginUI, logoutUI } from './fixtures';

test.describe('Sidebar RBAC Visibility', () => {
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
    await expect(sidebar.getByRole('button', { name: 'Transactions' })).toBeVisible();
    await expect(sidebar.getByRole('button', { name: 'Reports' })).toBeVisible();
    await expect(sidebar.getByRole('button', { name: 'Master Data' })).toBeVisible();
    await expect(sidebar.getByRole('button', { name: 'Administration' })).toBeVisible();
  });

  test('admin sees admin section but not POS', async ({ page }) => {
    await loginUI(page, TEST_USERS.admin.username, TEST_USERS.admin.password);
    await page.goto('http://localhost:5173/');
    const sidebar = page.locator('aside');
    await expect(sidebar).toBeVisible();

    await expect(sidebar.getByRole('button', { name: 'Dashboard' })).toBeVisible();
    await expect(sidebar.getByRole('button', { name: 'Transactions' })).toBeVisible();
    await expect(sidebar.getByRole('button', { name: 'Reports' })).toBeVisible();
    await expect(sidebar.getByRole('button', { name: 'Administration' })).toBeVisible();
  });

  test('manager sees dashboard, transactions, and reports', async ({ page }) => {
    await loginUI(page, 'manager', 'admin123');
    await page.goto('http://localhost:5173/');
    const sidebar = page.locator('aside');
    await expect(sidebar).toBeVisible();

    await expect(sidebar.getByRole('button', { name: 'Dashboard' })).toBeVisible();
    await expect(sidebar.getByRole('button', { name: 'Transactions' })).toBeVisible();
    await expect(sidebar.getByRole('button', { name: 'Reports' })).toBeVisible();
  });

  test('cashier sees POS and transactions', async ({ page }) => {
    await loginUI(page, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
    await page.goto('http://localhost:5173/');
    const sidebar = page.locator('aside');
    await expect(sidebar).toBeVisible();

    await expect(sidebar.getByRole('button', { name: 'Dashboard' })).toHaveCount(0);
    await expect(sidebar.getByRole('button', { name: 'Point of Sale' })).toBeVisible();
    await expect(sidebar.getByRole('button', { name: 'Transactions' })).toBeVisible();
    await expect(sidebar.getByRole('button', { name: 'Shifts' })).toBeVisible();
  });

  test('superadmin sees Stores under Administration', async ({ page }) => {
    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    await page.goto('http://localhost:5173/');
    const sidebar = page.locator('aside');
    await expect(sidebar).toBeVisible();

    await sidebar.getByRole('button', { name: 'Administration' }).click();
    await page.waitForTimeout(500);
    await expect(sidebar.getByRole('button', { name: 'Stores' })).toBeVisible();
  });

  test('admin sees Stores under Administration', async ({ page }) => {
    await loginUI(page, TEST_USERS.admin.username, TEST_USERS.admin.password);
    await page.goto('http://localhost:5173/');
    const sidebar = page.locator('aside');
    await expect(sidebar).toBeVisible();

    await sidebar.getByRole('button', { name: 'Administration' }).click();
    await page.waitForTimeout(500);
    await expect(sidebar.getByRole('button', { name: 'Stores' })).toBeVisible();
  });

  test('manager does not see Stores navigation item', async ({ page }) => {
    await loginUI(page, 'manager', 'admin123');
    await page.goto('http://localhost:5173/');
    const sidebar = page.locator('aside');
    await expect(sidebar).toBeVisible();

    await expect(sidebar.getByRole('button', { name: 'Administration' })).toHaveCount(0);
    await expect(sidebar.getByRole('button', { name: 'Stores' })).toHaveCount(0);
  });

  test('cashier does not see Stores navigation item', async ({ page }) => {
    await loginUI(page, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
    await page.goto('http://localhost:5173/');
    const sidebar = page.locator('aside');
    await expect(sidebar).toBeVisible();

    await expect(sidebar.getByRole('button', { name: 'Administration' })).toHaveCount(0);
    await expect(sidebar.getByRole('button', { name: 'Stores' })).toHaveCount(0);
  });

  test('sidebar navigates to correct pages', async ({ page }) => {
    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    await page.goto('http://localhost:5173/');

    const sidebar = page.locator('aside');
    const masterDataBtn = sidebar.getByRole('button', { name: 'Master Data' });
    await masterDataBtn.click();
    await page.waitForTimeout(500);

    await sidebar.getByRole('button', { name: 'Products' }).click();
    await expect(page).toHaveURL(/\/inventory\/products/);
  });
});
