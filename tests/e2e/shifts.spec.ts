import { test, expect } from './fixtures';
import { TEST_USERS, API_BASE, loginUI, logoutUI, getToken } from './fixtures';

test.describe('Shifts Page', () => {
  test.beforeEach(async ({ page }) => {
    const token = await getToken(page.request, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
    await page.request.post(`${API_BASE}/api/shifts/close-all`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    await loginUI(page, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('should load shift table without getting stuck on loading', async ({ page }) => {
    await page.goto('/shifts');
    await expect(page).toHaveURL(/\/shifts/);

    await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 15000 });
  });

  test('should show shifts table with correct columns for cashier', async ({ page }) => {
    await page.goto('/shifts');
    await expect(page).toHaveURL(/\/shifts/);

    await expect(page.locator('text=Loading shifts...')).toBeHidden({ timeout: 15000 });

    await expect(page.locator('text=OPENED AT')).toBeVisible();
    await expect(page.locator('text=OPENING (Rp)')).toBeVisible();
    await expect(page.locator('text=CASH SALES (Rp)')).toBeVisible();
    await expect(page.locator('text=TOTAL SALES (Rp)')).toBeVisible();
    await expect(page.locator('text=TXN')).toBeVisible();
    await expect(page.locator('text=DISCREPANCY')).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'STATUS' })).toBeVisible();
    await expect(page.locator('text=CLOSED AT')).toBeVisible();

    await expect(page.locator('th').filter({ hasText: 'CASHIER' })).toHaveCount(0);
  });

  test('should open shift modal and display active shift banner', async ({ page, request }) => {
    const token = await getToken(request, TEST_USERS.cashier.username, TEST_USERS.cashier.password);

    const closeRes = await page.request.post(`${API_BASE}/api/shifts/close-all`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(closeRes.ok()).toBe(true);

    await page.goto('/shifts');
    await expect(page).toHaveURL(/\/shifts/);
    await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 15000 });

    const openBtn = page.getByRole('button', { name: 'Open Shift' }).first();
    await expect(openBtn).toBeEnabled({ timeout: 5000 });
    await openBtn.click();

    const dialog = page.getByRole('dialog', { name: 'Open Shift' });
    await expect(dialog).toBeVisible();

    await page.locator('#opening-balance').fill('100000');
    await dialog.getByRole('button', { name: 'Open Shift' }).click();

    await expect(page.locator('text=Active Shift')).toBeVisible({ timeout: 10000 });
  });

  test('loading state does not persist after navigating back to shifts page', async ({ page }) => {
    await page.goto('/shifts');
    await expect(page).toHaveURL(/\/shifts/);

    const loadingSpinner = page.locator('text=Loading shifts...');

    await page.goto('/customers');
    await expect(page).toHaveURL(/\/customers/);

    await page.goto('/shifts');
    await expect(page).toHaveURL(/\/shifts/);
    await expect(loadingSpinner).toBeHidden({ timeout: 15000 });

    await expect(page.locator('text=OPENED AT')).toBeVisible({ timeout: 5000 });
  });

  test('admin sees CASHIER column and all shifts', async ({ page }) => {
    await logoutUI(page);
    await loginUI(page, TEST_USERS.admin.username, TEST_USERS.admin.password);

    await page.goto('/shifts');
    await expect(page).toHaveURL(/\/shifts/);
    await expect(page.locator('text=Loading shifts...')).toBeHidden({ timeout: 15000 });

    await expect(page.locator('th').filter({ hasText: 'CASHIER' })).toBeVisible();
  });

  test('superadmin sees CASHIER column and all shifts', async ({ page }) => {
    await logoutUI(page);
    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    await page.goto('/shifts');
    await expect(page).toHaveURL(/\/shifts/);
    await expect(page.locator('text=Loading shifts...')).toBeHidden({ timeout: 15000 });

    await expect(page.locator('th').filter({ hasText: 'CASHIER' })).toBeVisible();
  });
});

test.describe('Shifts API - cashier_id filter', () => {
  test('should list shifts filtered by cashier_id for cashier role', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.cashier.username, TEST_USERS.cashier.password);

    const res = await request.get(`${API_BASE}/api/shifts?limit=10&offset=0&sort_by=opened_at&sort_dir=desc&user_id=4`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();

    expect(body).toHaveProperty('data');
    expect(body).toHaveProperty('total');
  });

  test('superadmin sees all shifts via API', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    const res = await request.get(`${API_BASE}/api/shifts?limit=10&offset=0&sort_by=opened_at&sort_dir=desc`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body).toHaveProperty('data');
  });
});

test.describe('Shifts API - needs_review filter', () => {
  test('should filter shifts by needs_review=true', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const headers = { Authorization: `Bearer ${token}` };

    await request.post(`${API_BASE}/api/shifts/close-all`, { headers });

    const openRes = await request.post(`${API_BASE}/api/shifts/open`, {
      headers,
      data: { opening_balance: 100000, store_id: null },
    });
    expect(openRes.ok()).toBeTruthy();
    const openBody = await openRes.json();
    const shiftId = openBody.data.id;

    const closeRes = await request.post(`${API_BASE}/api/shifts/${shiftId}/close`, {
      headers: { Authorization: `Bearer ${token}` },
      data: { closing_balance: 200000 },
    });
    expect(closeRes.ok()).toBeTruthy();

    const res = await request.get(`${API_BASE}/api/shifts?limit=50&needs_review=true`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();

    expect(body).toHaveProperty('data');
    expect(body.data.length).toBeGreaterThan(0);
    for (const shift of body.data) {
      expect(shift.needs_review).toBe(true);
    }
  });

  test('should filter shifts by needs_review=false', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    const res = await request.get(`${API_BASE}/api/shifts?limit=50&needs_review=false`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();

    expect(body).toHaveProperty('data');
    expect(body.data.length).toBeGreaterThan(0);
    for (const shift of body.data) {
      expect(shift.needs_review).toBe(false);
    }
  });
});

test.describe('Shifts API - discrepancy filter', () => {
  test('should filter shifts by discrepancy=balanced', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const headers = { Authorization: `Bearer ${token}` };

    await request.post(`${API_BASE}/api/shifts/close-all`, { headers });

    const openRes = await request.post(`${API_BASE}/api/shifts/open`, {
      headers,
      data: { opening_balance: 100000, store_id: null },
    });
    expect(openRes.ok()).toBeTruthy();
    const openBody = await openRes.json();
    const shiftId = openBody.data.id;

    const closeRes = await request.post(`${API_BASE}/api/shifts/${shiftId}/close`, {
      headers: { Authorization: `Bearer ${token}` },
      data: { closing_balance: 100000 },
    });
    expect(closeRes.ok()).toBeTruthy();

    const res = await request.get(`${API_BASE}/api/shifts?limit=50&discrepancy=balanced`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();

    expect(body).toHaveProperty('data');
    expect(body.data.length).toBeGreaterThan(0);
    for (const shift of body.data) {
      if (shift.discrepancy !== null && shift.discrepancy !== undefined) {
        expect(shift.discrepancy).toBe(0);
      }
    }
  });

  test('should filter shifts by discrepancy=surplus', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const headers = { Authorization: `Bearer ${token}` };

    await request.post(`${API_BASE}/api/shifts/close-all`, { headers });

    const openRes = await request.post(`${API_BASE}/api/shifts/open`, {
      headers,
      data: { opening_balance: 100000, store_id: null },
    });
    expect(openRes.ok()).toBeTruthy();
    const openBody = await openRes.json();
    const shiftId = openBody.data.id;

    const closeRes = await request.post(`${API_BASE}/api/shifts/${shiftId}/close`, {
      headers: { Authorization: `Bearer ${token}` },
      data: { closing_balance: 200000 },
    });
    expect(closeRes.ok()).toBeTruthy();

    const res = await request.get(`${API_BASE}/api/shifts?limit=50&discrepancy=surplus`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();

    expect(body).toHaveProperty('data');
    expect(body.data.length).toBeGreaterThan(0);
    for (const shift of body.data) {
      expect(shift.discrepancy).toBeGreaterThan(0);
    }
  });

  test('should filter shifts by discrepancy=shortage', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const headers = { Authorization: `Bearer ${token}` };

    await request.post(`${API_BASE}/api/shifts/close-all`, { headers });

    const openRes = await request.post(`${API_BASE}/api/shifts/open`, {
      headers,
      data: { opening_balance: 100000, store_id: null },
    });
    expect(openRes.ok()).toBeTruthy();
    const openBody = await openRes.json();
    const shiftId = openBody.data.id;

    const closeRes = await request.post(`${API_BASE}/api/shifts/${shiftId}/close`, {
      headers: { Authorization: `Bearer ${token}` },
      data: { closing_balance: 0 },
    });
    expect(closeRes.ok()).toBeTruthy();

    const res = await request.get(`${API_BASE}/api/shifts?limit=50&discrepancy=shortage`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();

    expect(body).toHaveProperty('data');
    expect(body.data.length).toBeGreaterThan(0);
    for (const shift of body.data) {
      expect(shift.discrepancy).toBeLessThan(0);
    }
  });
});

test.describe('Shifts API - pagination', () => {
  test('should respect limit and offset pagination', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    const res = await request.get(`${API_BASE}/api/shifts?limit=5&offset=0`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();

    expect(body).toHaveProperty('data');
    expect(body.data.length).toBeLessThanOrEqual(5);
    expect(body).toHaveProperty('total');
    expect(body).toHaveProperty('limit', 5);
    expect(body).toHaveProperty('offset', 0);
    expect(body).toHaveProperty('total_pages');
  });
});

test.describe('Shifts API - export', () => {
  test('should export shifts as CSV', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    const res = await request.get(`${API_BASE}/api/shifts/export?format=csv`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.ok()).toBeTruthy();
    expect(res.headers()['content-type']).toContain('text/csv');
  });

  test('should export shifts as XLSX', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    const res = await request.get(`${API_BASE}/api/shifts/export?format=xlsx`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.ok()).toBeTruthy();
    expect(res.headers()['content-type']).toContain('spreadsheetml');
  });
});
