import { test, expect } from '@playwright/test';
import { TEST_USERS, API_BASE, authHeader, getToken as cachedGetToken, loginUI, logoutUI } from './fixtures';

const getToken = cachedGetToken;

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

// ============================================================================
// Categories API - Delete
// ============================================================================

test.describe('Categories API - Delete', () => {

  test('DELETE /api/categories/:id deletes a category', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    // Create a category to delete
    const createRes = await request.post(`${API_BASE}/api/categories`, {
      headers: authHeader(token),
      data: { name: `DelCat ${Date.now()}` },
    });
    expect(createRes.ok(), `create failed: ${createRes.status()}: ${await createRes.text()}`).toBeTruthy();
    const created = await createRes.json();
    const catId = created.data?.id || created.id;
    expect(catId).toBeTruthy();

    const deleteRes = await request.delete(`${API_BASE}/api/categories/${catId}`, {
      headers: authHeader(token),
    });
    expect(deleteRes.ok(), `delete failed: ${deleteRes.status()}: ${await deleteRes.text()}`).toBeTruthy();
  });

  test('DELETE /api/categories/:id returns 400 for invalid id', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const res = await request.delete(`${API_BASE}/api/categories/not-a-number`, {
      headers: authHeader(token),
    });
    expect(res.status()).toBe(400);
  });

  test('DELETE /api/categories/:id without auth returns 401', async ({ request }) => {
    const res = await request.delete(`${API_BASE}/api/categories/1`);
    expect(res.status()).toBe(401);
  });

  test('DELETE /api/categories/:id with restricted role returns 403', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
    const res = await request.delete(`${API_BASE}/api/categories/1`, {
      headers: authHeader(token),
    });
    expect(res.status()).toBe(403);
  });
});
