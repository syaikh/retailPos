import { test, expect } from '@playwright/test';

test.describe('Brands Management', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('http://localhost:5173/login');
    await page.fill('#username', 'superadmin');
    await page.fill('#password', 'admin123');
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL(/\/$/, { timeout: 10000 });
    await page.goto('http://localhost:5173/brands');
    await expect(page).toHaveURL(/\/brands/);
    await expect(page.locator('text=BRAND NAME')).toBeVisible({ timeout: 10000 });
  });

  test('should display brand list table', async ({ page }) => {
    await expect(page.locator('text=BRAND NAME')).toBeVisible();
    await expect(page.locator('text=DESCRIPTION')).toBeVisible();
    await expect(page.locator('text=CREATED')).toBeVisible();
  });

  test('should create a new brand', async ({ page }) => {
    const brandName = `E2E Brand ${Date.now()}`;

    await page.getByRole('button', { name: 'Tambah Brand' }).first().click();
    await expect(page.getByRole('dialog', { name: 'Tambah Brand' })).toBeVisible();

    await page.fill('#brand-name', brandName);
    await page.fill('#brand-desc', 'Auto-generated e2e brand');

    await page.getByRole('dialog', { name: 'Tambah Brand' }).getByRole('button', { name: 'Tambah Brand' }).click();
    await expect(page.getByRole('dialog', { name: 'Tambah Brand' })).toBeHidden({ timeout: 10000 });
  });

  test('should edit the first brand', async ({ page }) => {
    await page.locator('button[aria-label="Edit"]').first().click();
    await expect(page.getByRole('dialog', { name: 'Edit Brand' })).toBeVisible();

    await page.fill('#brand-name', `Updated Brand ${Date.now()}`);
    await page.getByRole('button', { name: 'Simpan Perubahan' }).click();
    await expect(page.getByRole('dialog', { name: 'Edit Brand' })).toBeHidden({ timeout: 10000 });
  });

  test('should delete the first brand', async ({ page }) => {
    await page.locator('button[aria-label="Hapus"]').first().click();
    await expect(page.getByRole('dialog', { name: 'Hapus Brand' })).toBeVisible();

    await page.getByRole('dialog', { name: 'Hapus Brand' }).getByRole('button', { name: 'Hapus' }).click();
    await expect(page.getByRole('dialog', { name: 'Hapus Brand' })).toBeHidden({ timeout: 10000 });
  });

  test('should validate required name field', async ({ page }) => {
    await page.getByRole('button', { name: 'Tambah Brand' }).first().click();
    await expect(page.getByRole('dialog', { name: 'Tambah Brand' })).toBeVisible();

    await page.fill('#brand-name', '');
    await page.getByRole('dialog', { name: 'Tambah Brand' }).getByRole('button', { name: 'Tambah Brand' }).click();

    await expect(page.getByRole('dialog', { name: 'Tambah Brand' })).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Nama brand wajib diisi')).toBeVisible({ timeout: 10000 });
  });
});
