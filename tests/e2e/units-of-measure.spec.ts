import { test, expect } from '@playwright/test';
import { loginUI, logoutUI } from './fixtures';

test.describe('Units of Measure Management', () => {
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

  test('should create a new unit of measure', async ({ page }) => {
    const code = `U${Date.now()}`.slice(0, 10).toUpperCase();
    const name = `E2E Unit ${Date.now()}`;

    await page.getByRole('button', { name: 'Tambah Unit' }).first().click();
    await expect(page.getByRole('dialog', { name: 'Tambah Unit' })).toBeVisible();

    await page.locator('#uom-code').click();
    await page.locator('#uom-code').fill(code);
    await page.locator('#uom-name').click();
    await page.locator('#uom-name').fill(name);
    await page.locator('#uom-desc').click();
    await page.locator('#uom-desc').fill('Auto-generated e2e unit');

    const submitBtn = page.getByRole('dialog', { name: 'Tambah Unit' }).getByRole('button', { name: 'Tambah Unit' });
    await expect(submitBtn).toBeEnabled({ timeout: 5000 });
    const responsePromise = page.waitForResponse(resp => resp.url().includes('/api/units-of-measure') && resp.request().method() === 'POST');
    await submitBtn.click();
    await responsePromise;
    await expect(page.getByRole('dialog', { name: 'Tambah Unit' })).toBeHidden({ timeout: 15000 });
  });

  test('should edit the first unit of measure', async ({ page }) => {
    await page.locator('button[aria-label="Edit"]').first().click();
    await expect(page.getByRole('dialog', { name: 'Edit Unit' })).toBeVisible();

    await page.fill('#uom-name', `Updated Unit ${Date.now()}`);
    await page.getByRole('button', { name: 'Simpan Perubahan' }).click();
    await expect(page.getByRole('dialog', { name: 'Edit Unit' })).toBeHidden({ timeout: 10000 });
  });

  test('should delete the first unit of measure', async ({ page }) => {
    await page.locator('button[aria-label="Hapus"]').first().click();
    await expect(page.getByRole('dialog', { name: 'Hapus Unit' })).toBeVisible();

    await page.getByRole('dialog', { name: 'Hapus Unit' }).getByRole('button', { name: 'Hapus' }).click();
    await expect(page.getByRole('dialog', { name: 'Hapus Unit' })).toBeHidden({ timeout: 10000 });
  });

  test('should validate required fields', async ({ page }) => {
    await page.getByRole('button', { name: 'Tambah Unit' }).first().click();
    await expect(page.getByRole('dialog', { name: 'Tambah Unit' })).toBeVisible();

    await page.fill('#uom-code', '');
    await page.fill('#uom-name', '');
    await page.getByRole('dialog', { name: 'Tambah Unit' }).getByRole('button', { name: 'Tambah Unit' }).click();

    await expect(page.getByRole('dialog', { name: 'Tambah Unit' })).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Kode unit wajib diisi')).toBeVisible({ timeout: 10000 });
  });
});
