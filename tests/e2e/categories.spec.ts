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

  test('should create a new category', async ({ page }) => {
    const categoryName = `E2E Category ${Date.now()}`;

    await page.getByRole('button', { name: 'Add Category' }).first().click();
    await expect(page.getByRole('dialog', { name: 'Add Category' })).toBeVisible();

    await page.fill('#cat-name', categoryName);
    await page.fill('#cat-desc', 'E2E test category description');
    await page.getByRole('button', { name: 'Add Category' }).last().click();

    await expect(page.getByRole('dialog', { name: 'Add Category' })).toBeHidden({ timeout: 15000 });

    await page.getByPlaceholder('Search by name or slug...').fill(categoryName);
    await page.waitForTimeout(2000);
    await expect(page.getByText(categoryName, { exact: true })).toBeVisible({ timeout: 15000 });
  });

  test('should edit a category', async ({ page }) => {
    const categoryName = `Edit Category ${Date.now()}`;

    await page.getByRole('button', { name: 'Add Category' }).first().click();
    await page.fill('#cat-name', categoryName);
    await page.fill('#cat-desc', 'Original description');
    await page.getByRole('button', { name: 'Add Category' }).last().click();
    await expect(page.getByRole('dialog', { name: 'Add Category' })).toBeHidden({ timeout: 15000 });

    await page.getByPlaceholder('Search by name or slug...').fill(categoryName);
    await page.waitForTimeout(1000);
    await expect(page.locator(`text=${categoryName}`).first()).toBeVisible({ timeout: 10000 });

    const editButton = page.locator('tr').filter({ hasText: categoryName }).locator('button[aria-label="Edit"]');
    await editButton.click();
    await expect(page.getByRole('dialog', { name: 'Edit Category' })).toBeVisible();

    await page.fill('#cat-name', `${categoryName} Updated`);
    await page.getByRole('button', { name: 'Save Changes' }).click();
    await expect(page.getByRole('dialog', { name: 'Edit Category' })).toBeHidden({ timeout: 10000 });

    await expect(page.locator(`text=${categoryName} Updated`).first()).toBeVisible({ timeout: 10000 });
  });

  test('should delete a category', async ({ page }) => {
    const categoryName = `Delete Category ${Date.now()}`;

    await page.getByRole('button', { name: 'Add Category' }).first().click();
    await page.fill('#cat-name', categoryName);
    await page.getByRole('button', { name: 'Add Category' }).last().click();
    await expect(page.getByRole('dialog', { name: 'Add Category' })).toBeHidden({ timeout: 15000 });

    await page.getByPlaceholder('Search by name or slug...').fill(categoryName);
    await page.waitForTimeout(1000);
    await expect(page.locator(`text=${categoryName}`).first()).toBeVisible({ timeout: 10000 });

    const deleteButton = page.locator('tr').filter({ hasText: categoryName }).locator('button[aria-label="Delete"]');
    await deleteButton.click();
    await expect(page.getByRole('dialog', { name: 'Delete Category' })).toBeVisible();

    await page.getByRole('dialog', { name: 'Delete Category' }).getByRole('button', { name: 'Delete' }).click();
    await expect(page.getByRole('dialog', { name: 'Delete Category' })).toBeHidden({ timeout: 10000 });

    await page.getByPlaceholder('Search by name or slug...').fill('');
    await page.waitForTimeout(1000);
    await expect(page.locator(`text=${categoryName}`).first()).toBeHidden({ timeout: 5000 });
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
