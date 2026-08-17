import { test, expect } from './fixtures';
import { loginUI, logoutUI } from './fixtures';

test.describe('Brands Management', () => {
  test.beforeEach(async ({ page }) => {
    await loginUI(page, 'superadmin', 'admin123');
    await page.goto('http://localhost:5173/brands');
    await page.waitForTimeout(2000);
    await expect(page.getByRole('columnheader', { name: 'BRAND NAME' })).toBeVisible({ timeout: 10000 });
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('should display brand list table', async ({ page }) => {
    await expect(page.getByRole('columnheader', { name: 'BRAND NAME' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'DESCRIPTION' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'CREATED' })).toBeVisible();
  });

  test('should create a new brand', async ({ page }) => {
    const brandName = `E2E Brand ${Date.now()}`;

    await page.getByRole('button', { name: 'Add Brand' }).first().click();
    await expect(page.getByRole('dialog', { name: 'Add Brand' })).toBeVisible();

    await page.fill('#brand-name', brandName);
    await page.fill('#brand-desc', 'Auto-generated e2e brand');

    await page.getByRole('dialog', { name: 'Add Brand' }).getByRole('button', { name: 'Add Brand' }).click();
    await expect(page.getByRole('dialog', { name: 'Add Brand' })).toBeHidden({ timeout: 10000 });
  });

  test('should edit the first brand', async ({ page }) => {
    await page.locator('button[aria-label="Edit"]').first().click();
    await expect(page.getByRole('dialog', { name: 'Edit Brand' })).toBeVisible();

    await page.fill('#brand-name', `Updated Brand ${Date.now()}`);
    await page.getByRole('button', { name: 'Save Changes' }).click();
    await expect(page.getByRole('dialog', { name: 'Edit Brand' })).toBeHidden({ timeout: 10000 });
  });

  test('should delete the first brand', async ({ page }) => {
    await page.locator('button[aria-label="Delete"]').first().click();
    await expect(page.getByRole('dialog', { name: 'Delete Brand' })).toBeVisible();

    await page.getByRole('dialog', { name: 'Delete Brand' }).getByRole('button', { name: 'Delete' }).click();
    await expect(page.getByRole('dialog', { name: 'Delete Brand' })).toBeHidden({ timeout: 10000 });
  });

  test('should validate required name field', async ({ page }) => {
    await page.getByRole('button', { name: 'Add Brand' }).first().click();
    await expect(page.getByRole('dialog', { name: 'Add Brand' })).toBeVisible();

    await page.fill('#brand-name', '');
    await page.getByRole('dialog', { name: 'Add Brand' }).getByRole('button', { name: 'Add Brand' }).click();

    await expect(page.getByRole('dialog', { name: 'Add Brand' })).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Brand name is required')).toBeVisible({ timeout: 10000 });
  });
});
