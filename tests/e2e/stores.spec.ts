import { test, expect } from './fixtures';
import { TEST_USERS, loginUI, logoutUI } from './fixtures';

const BASE = 'http://localhost:5173';

test.describe('Store Management', () => {
  test.beforeEach(async ({ page }) => {
    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    await page.goto(`${BASE}/stores`);
    await expect(page).toHaveURL(/\/stores/);
    await expect(page.locator('text=STORE NAME')).toBeVisible({ timeout: 10000 });
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('should display stores table with columns', async ({ page }) => {
    await expect(page.getByRole('columnheader', { name: 'STORE NAME' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'ADDRESS' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'PHONE' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'STATUS' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'CREATED' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'ACTIONS' })).toBeVisible();
  });

  test('should render Home / Administration / Stores breadcrumb', async ({ page }) => {
    const breadcrumb = page.locator('nav[aria-label="Breadcrumb"]');
    await expect(breadcrumb).toBeVisible();
    await expect(breadcrumb.locator('h1')).toHaveText('Stores');
    await expect(breadcrumb.locator('a', { hasText: 'Home' })).toBeVisible();
    await expect(breadcrumb.locator('a', { hasText: 'Administration' })).toBeVisible();
  });

  // NOTE: store create/edit/deactivate/delete *behaviour* is covered at the
  // API layer in entities-api.spec.ts. The tests below cover only genuine UI
  // behaviour (rendering, breadcrumb, search, status chips, bulk actions).

  test('should search stores and show empty state', async ({ page }) => {
    await page.getByPlaceholder('Search by name, address, or phone...').fill('ZZZ_NONEXISTENT_STORE_12345');
    await page.waitForTimeout(1500);
    await expect(page.locator('text=No stores found')).toBeVisible({ timeout: 5000 });
  });

  test('should filter by status chips', async ({ page }) => {
    await page.getByRole('button', { name: 'Active', exact: true }).click();
    await page.waitForTimeout(1000);
    const visibleRows = page.locator('tbody tr').filter({ visible: true });
    await expect(visibleRows.first()).toBeVisible({ timeout: 5000 });

    await page.getByRole('button', { name: 'Inactive', exact: true }).click();
    await page.waitForTimeout(1000);

    await page.getByRole('button', { name: 'All', exact: true }).click();
    await page.waitForTimeout(1000);
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 5000 });
  });

  test('should show import/export dropdown', async ({ page }) => {
    await page.locator('button').filter({ hasText: 'Bulk Actions' }).first().click();
    await expect(page.getByRole('menuitem', { name: 'Export CSV' })).toBeVisible({ timeout: 5000 });
    await expect(page.getByRole('menuitem', { name: 'Export XLSX' })).toBeVisible();
  });

  test('should open Stores import history', async ({ page }) => {
    await page.locator('button').filter({ hasText: 'Bulk Actions' }).first().click();
    await page.getByRole('menuitem', { name: 'Import History' }).click();
    await expect(page).toHaveURL(/\/stores\/import-history/, { timeout: 5000 });
    await expect(page.locator('h2', { hasText: 'Import History' })).toBeVisible({ timeout: 10000 });
    await expect(page.locator('p', { hasText: /^Stores$/ }).first()).toBeVisible();
  });

  test('should validate required fields', async ({ page }) => {
    await page.getByRole('button', { name: 'Add Store' }).first().click();
    await expect(page.getByRole('dialog', { name: 'Add Store' })).toBeVisible();

    await page.fill('#store-name', '');
    await page.getByRole('dialog', { name: 'Add Store' }).getByRole('button', { name: 'Add Store' }).click();

    await expect(page.getByRole('dialog', { name: 'Add Store' })).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Store name is required')).toBeVisible({ timeout: 10000 });
  });
});
