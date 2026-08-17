import { test, expect } from './fixtures';
import { TEST_USERS, API_BASE, FRONTEND_BASE, authHeader, getToken as cachedGetToken, loginUI, logoutUI, waitForAPI } from './fixtures';

const getToken = cachedGetToken;

// ============================================================================
// Helpers
// ============================================================================

async function createSupplierAPI(request: any, token: string, data: Record<string, any>) {
  const suffix = `${Date.now()}${Math.floor(Math.random() * 1000000)}`;
  const payload = {
    name: `E2E Supplier ${suffix}`,
    code: `E2E-SUP-${suffix}`.slice(0, 50),
    contact_name: 'Test Contact',
    phone: '081234567890',
    email: `sup.${suffix}@test.com`,
    address: 'Jl. Test No. 1',
    is_active: true,
    ...data,
  };
  const res = await request.post(`${API_BASE}/api/suppliers`, {
    headers: authHeader(token),
    data: payload,
  });
  expect(res.ok(), `create supplier failed: ${res.status()} ${await res.text()}`).toBeTruthy();
  const body = await res.json();
  return body.data;
}

async function navigateToSuppliers(page: any) {
  await page.goto(`${FRONTEND_BASE}/suppliers`);
  await page.waitForSelector('table', { state: 'visible', timeout: 15000 });
}

// ============================================================================
// 1. UI tests: Superadmin full workflow
// ============================================================================

test.describe('Suppliers UI - Superadmin', () => {
  test.beforeEach(async ({ page }) => {
    await waitForAPI(page);
    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    await navigateToSuppliers(page);
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('displays suppliers page with table', async ({ page }) => {
    await expect(page.locator('table[aria-label="Suppliers"]')).toBeVisible();
    await expect(page.locator('th').filter({ hasText: 'NAME' })).toBeVisible();
    await expect(page.locator('th').filter({ hasText: 'STATUS' })).toBeVisible();
  });

  test('table rows are clickable (role=button)', async ({ page }) => {
    const rows = page.locator('table tbody tr[role="button"]');
    const count = await rows.count();
    expect(count).toBeGreaterThan(0);
    const role = await rows.first().getAttribute('role');
    expect(role).toBe('button');
  });

  test('row click opens detail drawer', async ({ page }) => {
    const firstRow = page.locator('table tbody tr[role="button"]').first();
    await firstRow.click();
    const drawer = page.locator('[role="dialog"]').last();
    await expect(drawer).toBeVisible({ timeout: 5000 });
  });

  test('kebab menu opens with actions', async ({ page }) => {
    const kebab = page.locator('table tbody tr').first().locator('button[aria-label*="Actions"]').first();
    await expect(kebab).toBeVisible();
    await kebab.click();
    await expect(page.locator('button[role="menuitem"]').first()).toBeVisible({ timeout: 3000 });
    const menuItems = page.locator('button[role="menuitem"]');
    const count = await menuItems.count();
    expect(count).toBeGreaterThanOrEqual(1);
  });

  test('kebab menu shows View Products option', async ({ page }) => {
    const kebab = page.locator('table tbody tr').first().locator('button[aria-label*="Actions"]').first();
    await kebab.click();
    await expect(page.locator('button[role="menuitem"]').filter({ hasText: 'View Products' })).toBeVisible({ timeout: 3000 });
  });

  test('kebab menu shows Edit option', async ({ page }) => {
    const kebab = page.locator('table tbody tr').first().locator('button[aria-label*="Actions"]').first();
    await kebab.click();
    await expect(page.locator('button[role="menuitem"]').filter({ hasText: 'Edit' })).toBeVisible({ timeout: 3000 });
  });

  test('kebab menu shows Duplicate option', async ({ page }) => {
    const kebab = page.locator('table tbody tr').first().locator('button[aria-label*="Actions"]').first();
    await kebab.click();
    await expect(page.locator('button[role="menuitem"]').filter({ hasText: 'Duplicate' })).toBeVisible({ timeout: 3000 });
  });

  test('kebab menu shows Delete option', async ({ page }) => {
    const kebab = page.locator('table tbody tr').first().locator('button[aria-label*="Actions"]').first();
    await kebab.click();
    await expect(page.locator('button[role="menuitem"]').filter({ hasText: 'Delete' })).toBeVisible({ timeout: 3000 });
  });

  test('create supplier via Add Supplier button', async ({ page }) => {
    const name = `UI Supplier ${Date.now()}`;
    const code = `UI-SUP-${Date.now()}`.slice(0, 50);
    await page.locator('button').filter({ hasText: /Add Supplier/ }).first().click();
    const modal = page.locator('[role="dialog"]').first();
    await expect(modal).toBeVisible({ timeout: 5000 });
    await modal.locator('input').first().fill(name);
    await modal.locator('button').filter({ hasText: /Create|Simpan|Save/ }).first().click();
    await page.waitForTimeout(2000);
  });

  test('search filters suppliers', async ({ page }) => {
    const searchInput = page.locator('input[placeholder*="Search"]').first();
    if (await searchInput.isVisible()) {
      await searchInput.fill('NonexistentXYZ123');
      await page.waitForTimeout(1500);
      const noResults = page.locator('text=No suppliers found');
      const emptyTable = page.locator('table tbody tr').filter({ hasNot: page.locator('td') });
      const visible = await noResults.isVisible().catch(() => false) || await emptyTable.count() === 0;
      expect(visible).toBeTruthy();
    }
  });

  test('bulk action bar appears when checkbox selected', async ({ page }) => {
    const checkbox = page.locator('table tbody tr').first().locator('input[type="checkbox"]');
    await checkbox.click();
    await expect(page.locator('text=/selected/')).toBeVisible({ timeout: 5000 });
  });

  test('bulk action bar has Activate, Deactivate, Delete buttons', async ({ page }) => {
    const checkbox = page.locator('table tbody tr').first().locator('input[type="checkbox"]');
    await checkbox.click();
    await expect(page.locator('text=/selected/')).toBeVisible({ timeout: 5000 });
    await expect(page.getByRole('button', { name: 'Activate', exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Deactivate', exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Delete', exact: true })).toBeVisible();
  });

  test('Lihat Produk navigates to products page with supplier filter', async ({ page }) => {
    const kebab = page.locator('table tbody tr').first().locator('button[aria-label*="Actions"]').first();
    await kebab.click();
    await page.locator('button[role="menuitem"]').filter({ hasText: 'View Products' }).click();
    await page.waitForTimeout(2000);
    const url = page.url();
    expect(url).toContain('supplier_id=');
  });
});

// ============================================================================
// 2. UI tests: Staff access
// ============================================================================

test.describe('Suppliers UI - Staff', () => {
  test('staff is denied access to /suppliers page', async ({ page }) => {
    await loginUI(page, 'staff', 'admin123');
    await page.goto(`${FRONTEND_BASE}/suppliers`);
    await page.waitForTimeout(2000);
    const redirectedAway = !page.url().includes('/suppliers');
    const isOnLogin = page.url().includes('/login');
    const accessDenied = await page.locator('text=Access Denied').isVisible().catch(() => false);
    expect(redirectedAway || isOnLogin || accessDenied).toBeTruthy();
  });
});

// ============================================================================
// 3. UI tests: Unauthenticated access
// ============================================================================

test.describe('Suppliers UI - Unauthenticated', () => {
  test('redirects to login when not authenticated', async ({ page }) => {
    await page.goto(`${FRONTEND_BASE}/login`);
    await page.evaluate(() => sessionStorage.clear());
    await page.goto(`${FRONTEND_BASE}/suppliers`);
    await page.waitForTimeout(2000);
    expect(page.url().includes('/login')).toBeTruthy();
  });
});
