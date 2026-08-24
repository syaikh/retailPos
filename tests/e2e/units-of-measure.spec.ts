import { test, expect } from './fixtures';
import { loginUI, logoutUI } from './fixtures';

// NOTE: unit-of-measure CRUD *behaviour* is covered at the API layer in
// entities-api.spec.ts. These browser tests cover only genuine UI behaviour:
// the table renders and the "code/name required" validation message appears.
test.describe('Units of Measure Management (UI only)', () => {
  test.beforeEach(async ({ page }) => {
    await loginUI(page, 'superadmin', 'admin123');
    await page.goto('http://localhost:5173/units-of-measure');
    await expect(page).toHaveURL(/\/units-of-measure/);
    await expect(page.locator('text=CODE')).toBeVisible({ timeout: 10000 });
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('should display unit of measure list table', async ({ page }) => {
    await expect(page.locator('text=CODE')).toBeVisible();
    await expect(page.locator('text=UNIT NAME')).toBeVisible();
    await expect(page.locator('text=DESCRIPTION')).toBeVisible();
    await expect(page.locator('text=CREATED')).toBeVisible();
  });

  test('should validate required fields', async ({ page }) => {
    await page.getByRole('button', { name: 'Add Unit' }).first().click();
    await expect(page.getByRole('dialog', { name: 'Add Unit' })).toBeVisible();

    await page.fill('#uom-code', '');
    await page.fill('#uom-name', '');
    await page.getByRole('dialog', { name: 'Add Unit' }).getByRole('button', { name: 'Add Unit' }).click();

    await expect(page.getByRole('dialog', { name: 'Add Unit' })).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Unit code is required')).toBeVisible({ timeout: 10000 });
  });
});
