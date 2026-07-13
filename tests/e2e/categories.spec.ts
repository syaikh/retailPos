import { test, expect } from '@playwright/test';
import { loginUI, logoutUI } from './fixtures';

test.describe('Categories Management', () => {
  test.beforeEach(async ({ page }) => {
    await loginUI(page, 'superadmin', 'admin123');
    await page.goto('http://localhost:5173/categories');
    await page.waitForTimeout(2000);
    await expect(page.getByRole('columnheader', { name: 'CATEGORY NAME' })).toBeVisible({ timeout: 10000 });
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('should display category list table', async ({ page }) => {
    await expect(page.getByRole('columnheader', { name: 'CATEGORY NAME' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'SLUG' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'PRODUCTS' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'CREATED' })).toBeVisible();
  });

  test('should create a new category', async ({ page }) => {
    const categoryName = `E2E Category ${Date.now()}`;

    await page.getByRole('button', { name: 'Tambah Kategori' }).first().click();
    await expect(page.getByRole('dialog', { name: 'Tambah Kategori' })).toBeVisible();

    await page.fill('#cat-name', categoryName);
    await page.fill('#cat-desc', 'Auto-generated e2e category');

    await page.getByRole('dialog', { name: 'Tambah Kategori' }).getByRole('button', { name: 'Tambah Kategori' }).click();
    await expect(page.getByRole('dialog', { name: 'Tambah Kategori' })).toBeHidden({ timeout: 10000 });
  });

  test('should edit the first category', async ({ page }) => {
    await page.locator('button[aria-label="Edit"]').first().click();
    await expect(page.getByRole('dialog', { name: 'Edit Kategori' })).toBeVisible();

    await page.fill('#cat-name', `Updated Category ${Date.now()}`);
    await page.getByRole('button', { name: 'Simpan Perubahan' }).click();
    await expect(page.getByRole('dialog', { name: 'Edit Kategori' })).toBeHidden({ timeout: 10000 });
  });

  test('should validate required name field', async ({ page }) => {
    await page.getByRole('button', { name: 'Tambah Kategori' }).first().click();
    await expect(page.getByRole('dialog', { name: 'Tambah Kategori' })).toBeVisible();

    await page.fill('#cat-name', '');
    await page.getByRole('dialog', { name: 'Tambah Kategori' }).getByRole('button', { name: 'Tambah Kategori' }).click();

    await expect(page.getByRole('dialog', { name: 'Tambah Kategori' })).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Nama kategori wajib diisi')).toBeVisible({ timeout: 10000 });
  });
});
