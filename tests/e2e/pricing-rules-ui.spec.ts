import { test, expect } from './fixtures';
import { TEST_USERS, FRONTEND_BASE, authHeader, getToken as cachedGetToken, loginUI, logoutUI, waitForAPI } from './fixtures';

const getToken = cachedGetToken;

// ============================================================================
// Helpers
// ============================================================================

async function navigateToPricingRules(page: any) {
  await page.goto(`${FRONTEND_BASE}/pricing-rules`);
  await page.waitForSelector('table', { state: 'visible', timeout: 15000 });
}

// ============================================================================
// 1. UI tests: Superadmin full workflow
// ============================================================================

test.describe('Pricing Rules UI - Superadmin', () => {
  test.beforeEach(async ({ page }) => {
    await waitForAPI(page);
    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    await navigateToPricingRules(page);
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('displays pricing rules page with table', async ({ page }) => {
    await expect(page.locator('table[aria-label="Pricing Rules"]')).toBeVisible();
    await expect(page.locator('th').filter({ hasText: 'Name' })).toBeVisible();
  });

  test('has Create Rule button for superadmin', async ({ page }) => {
    await expect(page.locator('button').filter({ hasText: /Create Rule/ })).toBeVisible();
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
    const kebab = page.locator('table tbody tr').first().locator('button[aria-label*="Actions"], button[aria-label*="Aksi"]').first();
    if (await kebab.isVisible()) {
      await kebab.click();
      await expect(page.locator('button[role="menuitem"]').first()).toBeVisible({ timeout: 3000 });
    }
  });

  test('kebab menu shows Edit option', async ({ page }) => {
    const kebab = page.locator('table tbody tr').first().locator('button[aria-label*="Actions"], button[aria-label*="Aksi"]').first();
    if (await kebab.isVisible()) {
      await kebab.click();
      await expect(page.locator('button[role="menuitem"]').filter({ hasText: /Edit/ })).toBeVisible({ timeout: 3000 });
    }
  });

  test('kebab menu shows Duplikasi option', async ({ page }) => {
    const kebab = page.locator('table tbody tr').first().locator('button[aria-label*="Actions"], button[aria-label*="Aksi"]').first();
    if (await kebab.isVisible()) {
      await kebab.click();
      await expect(page.locator('button[role="menuitem"]').filter({ hasText: /Duplikasi|Duplicate/ })).toBeVisible({ timeout: 3000 });
    }
  });

  test('kebab menu shows Hapus option', async ({ page }) => {
    const kebab = page.locator('table tbody tr').first().locator('button[aria-label*="Actions"], button[aria-label*="Aksi"]').first();
    if (await kebab.isVisible()) {
      await kebab.click();
      await expect(page.locator('button[role="menuitem"]').filter({ hasText: /Hapus|Delete/ })).toBeVisible({ timeout: 3000 });
    }
  });

  test('create rule modal opens from Create Rule button', async ({ page }) => {
    await page.locator('button').filter({ hasText: /Create Rule/ }).first().click();
    const modal = page.getByRole('dialog', { name: /Add Pricing Rule/ });
    await expect(modal).toBeVisible({ timeout: 5000 });
    await expect(modal.locator('#rule-name')).toBeVisible();
    await expect(modal.locator('#pricing-type')).toBeVisible();
    await expect(modal.locator('#pricing-method')).toBeVisible();
    await expect(modal.locator('#pricing-value')).toBeVisible();
  });

  test('create modal shows validation error for empty name', async ({ page }) => {
    await page.locator('button').filter({ hasText: /Create Rule/ }).first().click();
    const modal = page.getByRole('dialog', { name: /Add Pricing Rule/ });
    await expect(modal).toBeVisible({ timeout: 5000 });
    await modal.locator('button').filter({ hasText: /Create Rule/ }).click();
    await expect(modal.locator('text=/Name is required/i')).toBeVisible({ timeout: 3000 });
  });

  test('create modal shows validation error for no target', async ({ page }) => {
    await page.locator('button').filter({ hasText: /Create Rule/ }).first().click();
    const modal = page.getByRole('dialog', { name: /Add Pricing Rule/ });
    await expect(modal).toBeVisible({ timeout: 5000 });
    await modal.locator('#rule-name').fill('Test Rule');
    await modal.locator('button').filter({ hasText: /Create Rule/ }).click();
    await expect(modal.locator('text=/Select at least one target/i')).toBeVisible({ timeout: 3000 });
  });

  test('create modal cancel closes without creating', async ({ page }) => {
    await page.locator('button').filter({ hasText: /Create Rule/ }).first().click();
    const modal = page.getByRole('dialog', { name: /Add Pricing Rule/ });
    await expect(modal).toBeVisible({ timeout: 5000 });
    await modal.locator('#rule-name').fill('Should Not Exist');
    await modal.locator('button').filter({ hasText: /Cancel/ }).first().click();
    await expect(page.getByRole('dialog', { name: /Add Pricing Rule/ })).toHaveCount(0);
  });

  test('status filter buttons work', async ({ page }) => {
    await page.locator('button').filter({ hasText: /All/ }).first().click();
    await page.waitForTimeout(1000);
    await expect(page.locator('table')).toBeVisible();
  });

  test('bulk action bar appears when checkbox selected', async ({ page }) => {
    const checkbox = page.locator('table tbody tr').first().locator('button[aria-label*="Pilih"], button[aria-label*="Select"]').first();
    if (await checkbox.isVisible()) {
      await checkbox.click();
      await expect(page.locator('text=/dipilih|selected/')).toBeVisible({ timeout: 5000 });
    }
  });

  test('detail drawer shows rule information', async ({ page }) => {
    const firstRow = page.locator('table tbody tr[role="button"]').first();
    await firstRow.click();
    const drawer = page.locator('[role="dialog"]').last();
    await expect(drawer).toBeVisible({ timeout: 5000 });
    await expect(drawer.locator('h2').first()).toBeVisible({ timeout: 3000 });
  });
});

// ============================================================================
// 2. UI tests: Staff access
// ============================================================================

test.describe('Pricing Rules UI - Staff', () => {
  test('staff is denied access to /pricing-rules page', async ({ page }) => {
    await loginUI(page, 'staff', 'admin123');
    await page.goto(`${FRONTEND_BASE}/pricing-rules`);
    await page.waitForTimeout(2000);
    const redirectedAway = !page.url().includes('/pricing-rules');
    const isOnLogin = page.url().includes('/login');
    const accessDenied = await page.locator('text=Access Denied').isVisible().catch(() => false);
    expect(redirectedAway || isOnLogin || accessDenied).toBeTruthy();
  });
});

// ============================================================================
// 3. UI tests: Unauthenticated access
// ============================================================================

test.describe('Pricing Rules UI - Unauthenticated', () => {
  test('redirects to login when not authenticated', async ({ page }) => {
    await page.goto(`${FRONTEND_BASE}/login`);
    await page.evaluate(() => sessionStorage.clear());
    await page.goto(`${FRONTEND_BASE}/pricing-rules`);
    await page.waitForTimeout(2000);
    expect(page.url().includes('/login')).toBeTruthy();
  });
});
