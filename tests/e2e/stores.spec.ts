import { test, expect } from '@playwright/test';
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

  test('should create a new store', async ({ page }) => {
    const storeName = `E2E Store ${Date.now()}`;

    await page.getByRole('button', { name: 'Tambah Toko' }).first().click();
    await expect(page.getByRole('dialog', { name: 'Tambah Toko' })).toBeVisible();

    await page.fill('#store-name', storeName);
    await page.fill('#store-address', 'Jl. E2E No. 1');
    await page.fill('#store-phone', '022-987654');

    const responsePromise = page.waitForResponse(resp => resp.url().includes('/api/stores') && resp.request().method() === 'POST');
    await page.getByRole('dialog', { name: 'Tambah Toko' }).getByRole('button', { name: 'Tambah Toko' }).click();
    await responsePromise;
    await expect(page.getByRole('dialog', { name: 'Tambah Toko' })).toBeHidden({ timeout: 15000 });

    await page.getByPlaceholder('Search by name, address, or phone...').fill(storeName);
    await page.waitForTimeout(2000);
    await expect(page.getByText(storeName, { exact: true })).toBeVisible({ timeout: 15000 });
  });

  test('should edit a store name', async ({ page }) => {
    const storeName = `Edit Store ${Date.now()}`;

    await page.getByRole('button', { name: 'Tambah Toko' }).first().click();
    await page.fill('#store-name', storeName);
    await page.getByRole('dialog', { name: 'Tambah Toko' }).getByRole('button', { name: 'Tambah Toko' }).click();
    await expect(page.getByRole('dialog', { name: 'Tambah Toko' })).toBeHidden({ timeout: 15000 });

    await page.getByPlaceholder('Search by name, address, or phone...').fill(storeName);
    await page.waitForTimeout(1000);
    await expect(page.locator(`text=${storeName}`).first()).toBeVisible({ timeout: 10000 });

    const editButton = page.locator('tr').filter({ hasText: storeName }).locator('button[aria-label="Edit"]');
    await editButton.click();
    await expect(page.getByRole('dialog', { name: 'Edit Toko' })).toBeVisible();

    await page.fill('#store-name', `${storeName} Updated`);
    await page.getByRole('dialog', { name: 'Edit Toko' }).getByRole('button', { name: 'Simpan Perubahan' }).click();
    await expect(page.getByRole('dialog', { name: 'Edit Toko' })).toBeHidden({ timeout: 10000 });

    await expect(page.locator(`text=${storeName} Updated`).first()).toBeVisible({ timeout: 10000 });
  });

  test('should deactivate a store via edit toggle', async ({ page }) => {
    const storeName = `Inactive Store ${Date.now()}`;

    await page.getByRole('button', { name: 'Tambah Toko' }).first().click();
    await page.fill('#store-name', storeName);
    await page.getByRole('dialog', { name: 'Tambah Toko' }).getByRole('button', { name: 'Tambah Toko' }).click();
    await expect(page.getByRole('dialog', { name: 'Tambah Toko' })).toBeHidden({ timeout: 15000 });

    await page.getByPlaceholder('Search by name, address, or phone...').fill(storeName);
    await page.waitForTimeout(1000);
    const row = page.locator('tr').filter({ hasText: storeName });
    await expect(row.first()).toBeVisible({ timeout: 10000 });
    await expect(row.first().getByText('Aktif', { exact: true })).toBeVisible();

    await row.first().locator('button[aria-label="Edit"]').click();
    await expect(page.getByRole('dialog', { name: 'Edit Toko' })).toBeVisible();

    const toggleSwitch = page.getByRole('dialog', { name: 'Edit Toko' }).locator('input[type="checkbox"]');
    await toggleSwitch.click({ force: true });
    await page.getByRole('dialog', { name: 'Edit Toko' }).getByRole('button', { name: 'Simpan Perubahan' }).click();
    await expect(page.getByRole('dialog', { name: 'Edit Toko' })).toBeHidden({ timeout: 10000 });

    await expect(row.first().getByText('Nonaktif', { exact: true })).toBeVisible({ timeout: 10000 });
  });

  test('should delete a store', async ({ page }) => {
    const storeName = `Delete Store ${Date.now()}`;

    await page.getByRole('button', { name: 'Tambah Toko' }).first().click();
    await page.fill('#store-name', storeName);
    await page.getByRole('dialog', { name: 'Tambah Toko' }).getByRole('button', { name: 'Tambah Toko' }).click();
    await expect(page.getByRole('dialog', { name: 'Tambah Toko' })).toBeHidden({ timeout: 15000 });

    await page.getByPlaceholder('Search by name, address, or phone...').fill(storeName);
    await page.waitForTimeout(1000);
    await expect(page.locator(`text=${storeName}`).first()).toBeVisible({ timeout: 10000 });

    const deleteButton = page.locator('tr').filter({ hasText: storeName }).locator('button[aria-label="Hapus"]');
    await deleteButton.click();
    await expect(page.getByRole('dialog', { name: 'Hapus Toko' })).toBeVisible();

    await page.getByRole('dialog', { name: 'Hapus Toko' }).getByRole('button', { name: 'Hapus' }).click();
    await expect(page.getByRole('dialog', { name: 'Hapus Toko' })).toBeHidden({ timeout: 10000 });

    await page.getByPlaceholder('Search by name, address, or phone...').fill('');
    await page.waitForTimeout(1000);
    await expect(page.locator(`text=${storeName}`).first()).toBeHidden({ timeout: 5000 });
  });

  test('should search stores and show empty state', async ({ page }) => {
    await page.getByPlaceholder('Search by name, address, or phone...').fill('ZZZ_NONEXISTENT_STORE_12345');
    await page.waitForTimeout(1500);
    await expect(page.locator('text=No stores found')).toBeVisible({ timeout: 5000 });
  });

  test('should filter by status chips', async ({ page }) => {
    await page.getByRole('button', { name: 'Aktif', exact: true }).click();
    await page.waitForTimeout(1000);
    const visibleRows = page.locator('tbody tr').filter({ visible: true });
    await expect(visibleRows.first()).toBeVisible({ timeout: 5000 });

    await page.getByRole('button', { name: 'Nonaktif', exact: true }).click();
    await page.waitForTimeout(1000);

    await page.getByRole('button', { name: 'Semua', exact: true }).click();
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
    await page.getByRole('button', { name: 'Tambah Toko' }).first().click();
    await expect(page.getByRole('dialog', { name: 'Tambah Toko' })).toBeVisible();

    await page.fill('#store-name', '');
    await page.getByRole('dialog', { name: 'Tambah Toko' }).getByRole('button', { name: 'Tambah Toko' }).click();

    await expect(page.getByRole('dialog', { name: 'Tambah Toko' })).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Nama toko wajib diisi')).toBeVisible({ timeout: 10000 });
  });
});
