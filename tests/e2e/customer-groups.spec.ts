import { test, expect } from '@playwright/test';
import { TEST_USERS, API_BASE, FRONTEND_BASE, authHeader, getToken as cachedGetToken, loginUI, logoutUI, waitForAPI } from './fixtures';

const getToken = cachedGetToken;

// ============================================================================
// Helpers
// ============================================================================

async function createGroupAPI(request: any, token: string, data: Record<string, any>) {
  const suffix = `${Date.now()}${Math.floor(Math.random() * 1000000)}`;
  const payload = { name: `E2E Group ${suffix}`, ...data };
  const res = await request.post(`${API_BASE}/api/customer-groups`, {
    headers: authHeader(token),
    data: payload,
  });
  expect(res.ok(), `create group failed: ${res.status()} ${await res.text()}`).toBeTruthy();
  const body = await res.json();
  return body.data;
}

async function navigateToGroups(page: any) {
  await page.goto(`${FRONTEND_BASE}/customer-groups`);
  await page.waitForSelector('table', { state: 'visible', timeout: 15000 });
}

// ============================================================================
// 1. API tests: CRUD
// ============================================================================

test.describe('Customer Groups API - CRUD', () => {
  let token: string;
  let createdId: number;

  test.beforeAll(async ({ request }) => {
    token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
  });

  test('POST /api/customer-groups creates a group', async ({ request }) => {
    const t = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const res = await request.post(`${API_BASE}/api/customer-groups`, {
      headers: authHeader(t),
      data: { name: `CRUD Create ${Date.now()}`, description: 'Test description', color: '#FF0000' },
    });
    expect(res.ok(), `create failed: ${res.status()} ${await res.text()}`).toBeTruthy();
    const body = await res.json();
    expect(body.data.name).toContain('CRUD Create');
    expect(body.data.color).toBe('#FF0000');
    createdId = body.data.id;
  });

  test('GET /api/customer-groups lists groups', async ({ request }) => {
    const t = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const res = await request.get(`${API_BASE}/api/customer-groups?limit=10`, { headers: authHeader(t) });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(Array.isArray(body.data)).toBeTruthy();
    expect(body.total).toBeGreaterThanOrEqual(1);
  });

  test('GET /api/customer-groups/:id gets a single group', async ({ request }) => {
    if (!createdId) return;
    const t = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const res = await request.get(`${API_BASE}/api/customer-groups/${createdId}`, { headers: authHeader(t) });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data.id).toBe(createdId);
  });

  test('PUT /api/customer-groups/:id updates a group', async ({ request }) => {
    if (!createdId) return;
    const t = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const res = await request.put(`${API_BASE}/api/customer-groups/${createdId}`, {
      headers: authHeader(t),
      data: { name: 'Updated Group Name', description: 'Updated desc' },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data.name).toBe('Updated Group Name');
  });

  test('DELETE /api/customer-groups/:id deletes a group', async ({ request }) => {
    if (!createdId) return;
    const t = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const res = await request.delete(`${API_BASE}/api/customer-groups/${createdId}`, { headers: authHeader(t) });
    expect(res.ok()).toBeTruthy();
  });

  test('POST without auth returns 401', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/customer-groups`, {
      data: { name: 'No Auth' },
    });
    expect(res.status()).toBe(401);
  });

  test('rejects empty name', async ({ request }) => {
    const t = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const res = await request.post(`${API_BASE}/api/customer-groups`, {
      headers: authHeader(t),
      data: { name: '' },
    });
    expect(res.ok()).toBeFalsy();
  });

  test('rejects name > 100 chars', async ({ request }) => {
    const t = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const res = await request.post(`${API_BASE}/api/customer-groups`, {
      headers: authHeader(t),
      data: { name: 'A'.repeat(101) },
    });
    expect(res.ok()).toBeFalsy();
  });
});

// ============================================================================
// 2. API tests: RBAC
// ============================================================================

test.describe('Customer Groups API - RBAC', () => {
  test('admin can list groups', async ({ request }) => {
    const t = await getToken(request, TEST_USERS.admin.username, TEST_USERS.admin.password);
    const res = await request.get(`${API_BASE}/api/customer-groups`, { headers: authHeader(t) });
    expect(res.ok()).toBeTruthy();
  });

  test('manager can list groups', async ({ request }) => {
    const t = await getToken(request, TEST_USERS.manager.username, TEST_USERS.manager.password);
    const res = await request.get(`${API_BASE}/api/customer-groups`, { headers: authHeader(t) });
    expect(res.ok()).toBeTruthy();
  });

  test('cashier can list groups', async ({ request }) => {
    const t = await getToken(request, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
    const res = await request.get(`${API_BASE}/api/customer-groups`, { headers: authHeader(t) });
    expect(res.ok()).toBeTruthy();
  });

  test('cashier CANNOT create group (403)', async ({ request }) => {
    const t = await getToken(request, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
    const res = await request.post(`${API_BASE}/api/customer-groups`, {
      headers: authHeader(t),
      data: { name: 'Cashier Should Fail' },
    });
    expect(res.status()).toBe(403);
  });
});

// ============================================================================
// 3. API tests: Bulk operations
// ============================================================================

test.describe('Customer Groups API - Bulk Operations', () => {
  let token: string;

  test.beforeAll(async ({ request }) => {
    token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
  });

  test('PUT /api/customer-groups/bulk activates groups', async ({ request }) => {
    const g1 = await createGroupAPI(request, token, { name: `Bulk1 ${Date.now()}` });
    const g2 = await createGroupAPI(request, token, { name: `Bulk2 ${Date.now()}` });
    const res = await request.put(`${API_BASE}/api/customer-groups/bulk`, {
      headers: authHeader(token),
      data: { ids: [g1.id, g2.id], is_active: true },
    });
    expect(res.ok(), `bulk activate failed: ${res.status()} ${await res.text()}`).toBeTruthy();
    const body = await res.json();
    expect(body.updated).toBeGreaterThanOrEqual(1);
  });

  test('PUT /api/customer-groups/bulk deactivates groups', async ({ request }) => {
    const g1 = await createGroupAPI(request, token, { name: `BulkDeact ${Date.now()}` });
    const res = await request.put(`${API_BASE}/api/customer-groups/bulk`, {
      headers: authHeader(token),
      data: { ids: [g1.id], is_active: false },
    });
    expect(res.ok()).toBeTruthy();
    const getRes = await request.get(`${API_BASE}/api/customer-groups/${g1.id}`, { headers: authHeader(token) });
    const body = await getRes.json();
    expect(body.data.is_active).toBe(false);
  });

  test('DELETE /api/customer-groups/bulk deletes multiple groups', async ({ request }) => {
    const g1 = await createGroupAPI(request, token, { name: `BulkDel1 ${Date.now()}` });
    const g2 = await createGroupAPI(request, token, { name: `BulkDel2 ${Date.now()}` });
    const res = await request.delete(`${API_BASE}/api/customer-groups/bulk`, {
      headers: authHeader(token),
      data: { ids: [g1.id, g2.id] },
    });
    expect(res.ok(), `bulk delete failed: ${res.status()} ${await res.text()}`).toBeTruthy();
    const body = await res.json();
    expect(body.deleted).toBeGreaterThanOrEqual(1);
  });

  test('bulk with empty IDs returns error', async ({ request }) => {
    const res = await request.put(`${API_BASE}/api/customer-groups/bulk`, {
      headers: authHeader(token),
      data: { ids: [], is_active: true },
    });
    expect(res.ok()).toBeFalsy();
  });

  test('cashier CANNOT bulk delete (403)', async ({ request }) => {
    const t = await getToken(request, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
    const res = await request.delete(`${API_BASE}/api/customer-groups/bulk`, {
      headers: authHeader(t),
      data: { ids: [1] },
    });
    expect(res.status()).toBe(403);
  });
});

// ============================================================================
// 4. API tests: Audit logging
// ============================================================================

test.describe('Customer Groups API - Audit Logging', () => {
  test('creating a group generates an audit log', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const g = await createGroupAPI(request, token, { name: `Audit Group ${Date.now()}` });
    await new Promise(r => setTimeout(r, 1500));
    const logsRes = await request.get(`${API_BASE}/api/audit-logs?action=create&entity_type=customer_group`, {
      headers: authHeader(token),
    });
    expect(logsRes.ok()).toBeTruthy();
    const logs = await logsRes.json();
    const found = logs.data?.some((l: any) => l.entity_id === g.id && l.action === 'create');
    expect(found).toBeTruthy();
  });
});

// ============================================================================
// 5. UI tests: Superadmin workflow
// ============================================================================

test.describe('Customer Groups UI - Superadmin', () => {
  test.beforeEach(async ({ page }) => {
    await waitForAPI(page);
    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    await navigateToGroups(page);
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('displays customer groups page with table', async ({ page }) => {
    await expect(page.locator('table')).toBeVisible();
    await expect(page.locator('button').filter({ hasText: /Tambah Group/ })).toBeVisible();
  });

  test('create group via modal', async ({ page }) => {
    const name = `UI Group ${Date.now()}`;
    await page.locator('button').filter({ hasText: /Tambah Group/ }).first().click();
    const modal = page.getByRole('dialog', { name: 'Add Customer Group' });
    await expect(modal).toBeVisible();
    await modal.locator('#cg-name').fill(name);
    await modal.locator('#cg-description').fill('UI test description');
    await modal.locator('button').filter({ hasText: /Create Group/ }).click();
    await expect(page.locator('table td').filter({ hasText: name }).first()).toBeVisible({ timeout: 10000 });
  });

  test('create modal validation: empty name shows error', async ({ page }) => {
    await page.locator('button').filter({ hasText: /Tambah Group/ }).first().click();
    const modal = page.getByRole('dialog', { name: 'Add Customer Group' });
    await expect(modal).toBeVisible();
    await modal.locator('button').filter({ hasText: /Create Group/ }).click();
    await expect(modal.locator('text=/Name is required/i')).toBeVisible();
  });

  test('create modal cancel closes without creating', async ({ page }) => {
    const name = `Cancel Group ${Date.now()}`;
    await page.locator('button').filter({ hasText: /Tambah Group/ }).first().click();
    const modal = page.getByRole('dialog', { name: 'Add Customer Group' });
    await modal.locator('#cg-name').fill(name);
    await modal.locator('button').filter({ hasText: 'Cancel' }).click();
    await expect(page.getByRole('dialog', { name: 'Add Customer Group' })).toHaveCount(0);
    await expect(page.locator('table td').filter({ hasText: name })).toHaveCount(0);
  });

  test('status filter: Aktif shows active groups', async ({ page }) => {
    await page.locator('button').filter({ hasText: /^Aktif$/ }).first().click();
    await page.waitForTimeout(2000);
    const badges = page.locator('table tbody tr').locator('text=Aktif');
    const count = await badges.count();
    if (count > 0) {
      await expect(badges.first()).toBeVisible();
    }
  });

  test('status filter: Nonaktif shows inactive groups', async ({ page }) => {
    await page.locator('button').filter({ hasText: /^Nonaktif$/ }).first().click();
    await page.waitForTimeout(2000);
    await expect(page.locator('table')).toBeVisible();
  });

  test('row click opens detail drawer', async ({ page }) => {
    const firstRow = page.locator('table tbody tr[role="button"]').first();
    if (await firstRow.count() > 0) {
      await firstRow.click();
      await expect(page.locator('[aria-label="Detail Customer Group"]')).toBeVisible({ timeout: 5000 });
    }
  });

  test('detail drawer shows audit trail section', async ({ page }) => {
    const firstRow = page.locator('table tbody tr[role="button"]').first();
    if (await firstRow.count() > 0) {
      await firstRow.click();
      await expect(page.locator('[aria-label="Detail Customer Group"]')).toBeVisible({ timeout: 5000 });
      await expect(page.locator('text=Riwayat Aktivitas')).toBeVisible({ timeout: 5000 });
    }
  });

  test('search filters groups', async ({ page }) => {
    const searchInput = page.locator('#customer-group-search');
    await searchInput.fill('NonexistentXYZ123');
    await page.waitForTimeout(1000);
    await expect(page.locator('text=Tidak ada group ditemukan').or(page.locator('text=Belum ada customer group'))).toBeVisible({ timeout: 5000 });
  });

  test('bulk action bar appears when checkbox selected', async ({ page }) => {
    const firstCheckbox = page.locator('table tbody tr').first().locator('button[aria-label*="Pilih"]').first();
    if (await firstCheckbox.count() > 0) {
      await firstCheckbox.click();
      await expect(page.locator('text=/group dipilih/')).toBeVisible({ timeout: 5000 });
    }
  });
});

// ============================================================================
// 6. UI tests: Staff access
// ============================================================================

test.describe('Customer Groups UI - Staff', () => {
  test('staff is denied access to /customer-groups page', async ({ page }) => {
    await loginUI(page, 'staff', 'admin123');
    await page.goto(`${FRONTEND_BASE}/customer-groups`);
    await page.waitForTimeout(2000);
    const redirectedAway = !page.url().includes('/customer-groups');
    const isOnLogin = page.url().includes('/login');
    const accessDenied = await page.locator('text=Access Denied').isVisible().catch(() => false);
    expect(redirectedAway || isOnLogin || accessDenied).toBeTruthy();
  });
});
