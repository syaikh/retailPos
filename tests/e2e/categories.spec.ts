import { test, expect } from './fixtures';
import { TEST_USERS, loginUI, logoutUI } from './fixtures';

test.describe('Categories Management', () => {
  test.beforeEach(async ({ page }) => {
    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    await page.goto('http://localhost:5173/categories');
    await expect(page).toHaveURL(/\/categories/);
    await expect(page.locator('text=CATEGORY NAME')).toBeVisible({ timeout: 10000 });
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('should display categories table with columns', async ({ page }) => {
    await expect(page.getByRole('columnheader', { name: 'CATEGORY NAME' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'SLUG' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'PRODUCTS' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'CREATED' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'ACTIONS' })).toBeVisible();
  });

  test('Add Category dialog opens with the name field', async ({ page }) => {
    await page.getByRole('button', { name: 'Add Category' }).first().click();
    const dlg = page.getByRole('dialog', { name: 'Add Category' });
    await expect(dlg).toBeVisible();
    await expect(dlg.locator('#cat-name')).toBeVisible();
  });

  test('should search categories', async ({ page }) => {
    await page.getByPlaceholder('Search by name or slug...').fill('Personal Care');
    await page.waitForTimeout(1000);
    await expect(page.locator('text=Personal Care').first()).toBeVisible({ timeout: 5000 });
  });

  test('should show empty state when no categories match search', async ({ page }) => {
    await page.getByPlaceholder('Search by name or slug...').fill('ZZZ_NONEXISTENT_CATEGORY_12345');
    await page.waitForTimeout(1500);
    await expect(page.locator('text=No categories found')).toBeVisible({ timeout: 5000 });
  });

  test('should show import/export dropdown', async ({ page }) => {
    await page.locator('button').filter({ hasText: 'Bulk Actions' }).first().click();
    await expect(page.getByRole('menuitem', { name: 'Export CSV' })).toBeVisible({ timeout: 5000 });
    await expect(page.getByRole('menuitem', { name: 'Export XLSX' })).toBeVisible();
  });
});
