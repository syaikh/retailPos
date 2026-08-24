import { test, expect } from './fixtures';
import { loginUI, logoutUI } from './fixtures';

// NOTE: brand CRUD *behaviour* (create persists, list surfaces it, update,
// delete) is covered at the API layer in entities-api.spec.ts. These browser
// tests cover only genuine UI behaviour: the table renders and the "name
// required" validation message appears.
test.describe('Brands Management (UI only)', () => {
  test.beforeEach(async ({ page }) => {
    await loginUI(page, 'superadmin', 'admin123');
    await page.goto('http://localhost:5173/brands');
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

  test('should validate required name field', async ({ page }) => {
    await page.getByRole('button', { name: 'Add Brand' }).first().click();
    await expect(page.getByRole('dialog', { name: 'Add Brand' })).toBeVisible();

    await page.fill('#brand-name', '');
    await page.getByRole('dialog', { name: 'Add Brand' }).getByRole('button', { name: 'Add Brand' }).click();

    await expect(page.getByRole('dialog', { name: 'Add Brand' })).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Brand name is required')).toBeVisible({ timeout: 10000 });
  });
});
