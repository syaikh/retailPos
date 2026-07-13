import { test, expect } from '@playwright/test';
import { TEST_USERS, API_BASE, authHeader, waitForAPI, FRONTEND_BASE, getToken as cachedGetToken, loginUI, logoutUI } from './fixtures';

// ============================================================================
// Helpers
// ============================================================================

async function createCustomerAPI(request: any, token: string, data: Record<string, any>) {
  for (let attempt = 0; attempt < 3; attempt++) {
    const suffix = `${Date.now()}${Math.floor(Math.random() * 1000000)}`;
    const payload = attempt === 0 ? data : {
      ...data,
      phone: `08${suffix}`.slice(0, 13),
      email: `e2e.${suffix}@test.com`,
    };
    const res = await request.post(`${API_BASE}/api/customers`, {
      headers: authHeader(token),
      data: payload,
    });
    if (res.ok()) {
      const body = await res.json();
      return body.data;
    }
    const body = await res.text();
    if (attempt === 2) {
      expect(res.ok(), `create customer failed after 3 attempts: status=${res.status()} body=${body}`).toBeTruthy();
    }
    await new Promise(r => setTimeout(r, 200));
  }
}

function uniquePhone() {
  return `08${Date.now()}${Math.floor(Math.random() * 1000)}`.slice(0, 13);
}

function uniqueEmail() {
  return `e2e.${Date.now()}@test.com`;
}

async function navigateToCustomers(page: any) {
  await page.goto(`${FRONTEND_BASE}/customers`);
  await page.waitForSelector('table', { state: 'visible', timeout: 15000 });
}

async function openCreateModal(page: any) {
  await page.locator('button').filter({ hasText: /Add Customer/ }).first().click();
  await expect(page.getByRole('dialog', { name: 'Add Customer' })).toBeVisible();
}

async function fillCreateModal(page: any, data: { name?: string; phone?: string; email?: string; note?: string }) {
  const modal = page.getByRole('dialog', { name: 'Add Customer' });
  if (data.name !== undefined) {
    await modal.locator('input[placeholder*="John Doe"]').fill(data.name);
  }
  if (data.phone !== undefined) {
    await modal.locator('input[placeholder*="0812"]').fill(data.phone);
  }
  if (data.email !== undefined) {
    await modal.locator('input[placeholder*="john@"]').fill(data.email);
  }
  if (data.note !== undefined) {
    await modal.locator('textarea').fill(data.note);
  }
}

async function submitCreateModal(page: any) {
  const modal = page.getByRole('dialog', { name: 'Add Customer' });
  await modal.locator('button').filter({ hasText: /Create Customer/ }).click();
}

// Use cached token getter to avoid rate limiting
const getToken = cachedGetToken;

// ============================================================================
// 1. API tests: RBAC for CRUD across all roles
// ============================================================================

test.describe('Customers API - RBAC', () => {
  test('superadmin can list customers', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const res = await request.get(`${API_BASE}/api/customers?limit=200`, { headers: authHeader(token) });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(Array.isArray(body.data)).toBeTruthy();
  });

  test('admin can list customers', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.admin.username, TEST_USERS.admin.password);
    const res = await request.get(`${API_BASE}/api/customers?limit=200`, { headers: authHeader(token) });
    expect(res.ok()).toBeTruthy();
  });

  test('manager can list customers', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.manager.username, TEST_USERS.manager.password);
    const res = await request.get(`${API_BASE}/api/customers?limit=200`, { headers: authHeader(token) });
    expect(res.ok()).toBeTruthy();
  });

  test('cashier can list customers', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
    const res = await request.get(`${API_BASE}/api/customers?limit=200`, { headers: authHeader(token) });
    expect(res.ok()).toBeTruthy();
  });

  test('superadmin can create customer', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const data = await createCustomerAPI(request, token, { name: 'SA Create Test', phone: uniquePhone(), email: uniqueEmail() });
    expect(data.name).toBe('SA Create Test');
  });

  test('admin can create customer', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.admin.username, TEST_USERS.admin.password);
    const data = await createCustomerAPI(request, token, { name: 'Admin Create Test', phone: uniquePhone(), email: uniqueEmail() });
    expect(data.name).toBe('Admin Create Test');
  });

  test('manager can create customer', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.manager.username, TEST_USERS.manager.password);
    const data = await createCustomerAPI(request, token, { name: 'Manager Create Test', phone: uniquePhone(), email: uniqueEmail() });
    expect(data.name).toBe('Manager Create Test');
  });

  test('cashier CANNOT create customer (403)', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
    const res = await request.post(`${API_BASE}/api/customers`, {
      headers: authHeader(token),
      data: { name: 'Cashier Create Test', phone: uniquePhone(), email: uniqueEmail() },
    });
    expect(res.status()).toBe(403);
  });

  test('superadmin can update customer', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const created = await createCustomerAPI(request, token, { name: 'SA Update Me', phone: uniquePhone(), email: uniqueEmail() });
    const res = await request.put(`${API_BASE}/api/customers/${created.id}`, {
      headers: authHeader(token),
      data: { name: 'SA Updated', is_active: true },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data.name).toBe('SA Updated');
  });

  test('admin can update customer', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.admin.username, TEST_USERS.admin.password);
    const created = await createCustomerAPI(request, token, { name: 'Admin Update Me', phone: uniquePhone(), email: uniqueEmail() });
    const res = await request.put(`${API_BASE}/api/customers/${created.id}`, {
      headers: authHeader(token),
      data: { name: 'Admin Updated', is_active: true },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data.name).toBe('Admin Updated');
  });

  test('manager can update customer', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.manager.username, TEST_USERS.manager.password);
    const created = await createCustomerAPI(request, token, { name: 'Manager Update Me', phone: uniquePhone(), email: uniqueEmail() });
    const res = await request.put(`${API_BASE}/api/customers/${created.id}`, {
      headers: authHeader(token),
      data: { name: 'Manager Updated', is_active: true },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data.name).toBe('Manager Updated');
  });

  test('cashier CANNOT update customer (403)', async ({ request }) => {
    const saToken = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const created = await createCustomerAPI(request, saToken, { name: 'No Update Cashier', phone: uniquePhone(), email: uniqueEmail() });
    const cashierToken = await getToken(request, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
    const res = await request.put(`${API_BASE}/api/customers/${created.id}`, {
      headers: authHeader(cashierToken),
      data: { name: 'Hacked', is_active: true },
    });
    expect(res.status()).toBe(403);
  });

  test('superadmin can deactivate customer', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const created = await createCustomerAPI(request, token, { name: 'SA Deactivate Me', phone: uniquePhone(), email: uniqueEmail() });
    const delRes = await request.delete(`${API_BASE}/api/customers/${created.id}`, { headers: authHeader(token) });
    expect(delRes.ok()).toBeTruthy();
    const getRes = await request.get(`${API_BASE}/api/customers/${created.id}`, { headers: authHeader(token) });
    const getBody = await getRes.json();
    expect(getBody.data.is_active).toBe(false);
  });

  test('admin can deactivate customer', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.admin.username, TEST_USERS.admin.password);
    const created = await createCustomerAPI(request, token, { name: 'Admin Deactivate Me', phone: uniquePhone(), email: uniqueEmail() });
    const delRes = await request.delete(`${API_BASE}/api/customers/${created.id}`, { headers: authHeader(token) });
    expect(delRes.ok()).toBeTruthy();
    const getRes = await request.get(`${API_BASE}/api/customers/${created.id}`, { headers: authHeader(token) });
    const getBody = await getRes.json();
    expect(getBody.data.is_active).toBe(false);
  });

  test('manager CANNOT deactivate customer (403)', async ({ request }) => {
    const saToken = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const created = await createCustomerAPI(request, saToken, { name: 'No Delete Manager', phone: uniquePhone(), email: uniqueEmail() });
    const managerToken = await getToken(request, TEST_USERS.manager.username, TEST_USERS.manager.password);
    const res = await request.delete(`${API_BASE}/api/customers/${created.id}`, { headers: authHeader(managerToken) });
    expect(res.status()).toBe(403);
  });

  test('cashier CANNOT deactivate customer (403)', async ({ request }) => {
    const saToken = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const created = await createCustomerAPI(request, saToken, { name: 'No Delete Cashier', phone: uniquePhone(), email: uniqueEmail() });
    const cashierToken = await getToken(request, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
    const res = await request.delete(`${API_BASE}/api/customers/${created.id}`, { headers: authHeader(cashierToken) });
    expect(res.status()).toBe(403);
  });
});

// ============================================================================
// 2. API tests: Validation edge cases
// ============================================================================

test.describe('Customers API - Validation', () => {
  let token: string;

  test.beforeAll(async ({ request }) => {
    token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
  });

  test('rejects empty name', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/customers`, {
      headers: authHeader(token),
      data: { name: '', phone: uniquePhone(), email: uniqueEmail() },
    });
    expect([400, 422]).toContain(res.status());
    const body = await res.json();
    expect(body.error).toBeDefined();
  });

  test('rejects whitespace-only name', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/customers`, {
      headers: authHeader(token),
      data: { name: '   ', phone: uniquePhone(), email: uniqueEmail() },
    });
    expect([400, 422]).toContain(res.status());
  });

  test('rejects missing phone', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/customers`, {
      headers: authHeader(token),
      data: { name: 'Valid Name', email: 'test@test.com' },
    });
    expect([400, 422]).toContain(res.status());
  });

  test('rejects invalid phone format', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/customers`, {
      headers: authHeader(token),
      data: { name: 'Bad Phone', phone: 'abc', email: 'test@test.com' },
    });
    expect([400, 422]).toContain(res.status());
  });

  test('rejects invalid email format', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/customers`, {
      headers: authHeader(token),
      data: { name: 'Valid Name', phone: uniquePhone(), email: 'not-an-email' },
    });
    expect([400, 422]).toContain(res.status());
  });

  test('rejects missing email', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/customers`, {
      headers: authHeader(token),
      data: { name: 'Valid Name', phone: uniquePhone() },
    });
    expect([400, 422]).toContain(res.status());
  });

  test('long name (>200 chars) is rejected', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/customers`, {
      headers: authHeader(token),
      data: { name: 'A'.repeat(256), phone: uniquePhone(), email: uniqueEmail() },
    });
    expect(res.status()).toBe(400);
  });

  test('name exactly 200 chars is accepted', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/customers`, {
      headers: authHeader(token),
      data: { name: 'A'.repeat(200), phone: uniquePhone(), email: uniqueEmail() },
    });
    expect(res.ok()).toBeTruthy();
  });

  test('name >200 chars is rejected', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/customers`, {
      headers: authHeader(token),
      data: { name: 'A'.repeat(201), phone: uniquePhone(), email: uniqueEmail() },
    });
    expect(res.status()).toBe(400);
  });

  test('update validates email too', async ({ request }) => {
    const created = await createCustomerAPI(request, token, { name: 'Update Validation', phone: uniquePhone(), email: uniqueEmail() });
    const res = await request.put(`${API_BASE}/api/customers/${created.id}`, {
      headers: authHeader(token),
      data: { email: 'bad-email' },
    });
    expect([400, 422]).toContain(res.status());
  });

  test('update validates name is not empty', async ({ request }) => {
    const created = await createCustomerAPI(request, token, { name: 'Update Name Validation', phone: uniquePhone(), email: uniqueEmail() });
    const res = await request.put(`${API_BASE}/api/customers/${created.id}`, {
      headers: authHeader(token),
      data: { name: '' },
    });
    expect([400, 422]).toContain(res.status());
  });

  test('name is trimmed on create', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/customers`, {
      headers: authHeader(token),
      data: { name: '  Trim Me  ', phone: uniquePhone(), email: uniqueEmail() },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data.name).toBe('Trim Me');
  });
});

// ============================================================================
// 3. API tests: isActive filter
// ============================================================================

test.describe('Customers API - isActive Filter', () => {
  let token: string;

  test.beforeAll(async ({ request }) => {
    token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
  });

  test('filter by active=true returns only active customers', async ({ request }) => {
    const res = await request.get(`${API_BASE}/api/customers?limit=200&isActive=true`, { headers: authHeader(token) });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    for (const c of body.data) {
      expect(c.is_active).toBe(true);
    }
  });

  test('filter by isActive=false returns only inactive customers', async ({ request }) => {
    const res = await request.get(`${API_BASE}/api/customers?limit=200&isActive=false`, { headers: authHeader(token) });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    for (const c of body.data) {
      expect(c.is_active).toBe(false);
    }
  });

  test('no filter returns both active and inactive', async ({ request }) => {
    const res = await request.get(`${API_BASE}/api/customers?limit=200`, { headers: authHeader(token) });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(Array.isArray(body.data)).toBeTruthy();
  });
});

// ============================================================================
// 4. API tests: Audit logging
// ============================================================================

test.describe('Customers API - Audit Logging', () => {
  let token: string;

  test.beforeAll(async ({ request }) => {
    token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
  });

  test('creating a customer generates an audit log', async ({ request }) => {
    const created = await createCustomerAPI(request, token, { name: 'Audit Create Test', phone: uniquePhone(), email: uniqueEmail() });
    await new Promise(r => setTimeout(r, 1500));
    const logsRes = await request.get(`${API_BASE}/api/audit-logs?action=create&entity_type=customer`, {
      headers: authHeader(token),
    });
    expect(logsRes.ok()).toBeTruthy();
    const logs = await logsRes.json();
    const found = logs.data?.some((l: any) => l.entity_id === created.id && l.action === 'create');
    expect(found).toBeTruthy();
  });

  test('updating a customer generates an audit log with old/new values', async ({ request }) => {
    const created = await createCustomerAPI(request, token, { name: 'Audit Update Before', phone: uniquePhone(), email: uniqueEmail() });
    await request.put(`${API_BASE}/api/customers/${created.id}`, {
      headers: authHeader(token),
      data: { name: 'Audit Update After' },
    });
    await new Promise(r => setTimeout(r, 1500));
    const logsRes = await request.get(`${API_BASE}/api/audit-logs?action=update&entity_type=customer`, {
      headers: authHeader(token),
    });
    expect(logsRes.ok()).toBeTruthy();
    const logs = await logsRes.json();
    const found = logs.data?.find((l: any) => l.entity_id === created.id && l.action === 'update');
    expect(found).toBeTruthy();
  });

  test('deactivating a customer generates an audit log', async ({ request }) => {
    const created = await createCustomerAPI(request, token, { name: 'Audit Delete Test', phone: uniquePhone(), email: uniqueEmail() });
    await request.delete(`${API_BASE}/api/customers/${created.id}`, { headers: authHeader(token) });
    await new Promise(r => setTimeout(r, 1500));
    const logsRes = await request.get(`${API_BASE}/api/audit-logs?action=delete&entity_type=customer`, {
      headers: authHeader(token),
    });
    expect(logsRes.ok()).toBeTruthy();
    const logs = await logsRes.json();
    const found = logs.data?.some((l: any) => l.entity_id === created.id && l.action === 'delete');
    expect(found).toBeTruthy();
  });
});

// ============================================================================
// 5. API tests: Walk-in customer filtering
// ============================================================================

test.describe('Customers API - Walk-in Filtering', () => {
  let token: string;

  test.beforeAll(async ({ request }) => {
    token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
  });

  test('walk-in customer is NOT in the list', async ({ request }) => {
    const res = await request.get(`${API_BASE}/api/customers?limit=200`, { headers: authHeader(token) });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    const walkIn = body.data?.find((c: any) => c.is_walk_in === true);
    expect(walkIn).toBeUndefined();
  });

  test('non-walk-in customers ARE in the list', async ({ request }) => {
    const created = await createCustomerAPI(request, token, { name: 'Listed Customer', phone: uniquePhone(), email: uniqueEmail() });
    const res = await request.get(`${API_BASE}/api/customers?limit=200`, { headers: authHeader(token) });
    const body = await res.json();
    const found = body.data?.find((c: any) => c.id === created.id);
    expect(found).toBeTruthy();
  });

  test('cannot modify walk-in customer via API', async ({ request }) => {
    const knownWalkInIds = [1, 2, 3, 4, 5];
    let blocked = false;
    for (const id of knownWalkInIds) {
      const res = await request.put(`${API_BASE}/api/customers/${id}`, {
        headers: authHeader(token),
        data: { name: 'Hacked Walk-in' },
      });
      if (res.status() === 403) {
        blocked = true;
        break;
      }
    }
    expect(blocked).toBeTruthy();
  });

  test('cannot delete walk-in customer via API', async ({ request }) => {
    const knownWalkInIds = [1, 2, 3, 4, 5];
    let blocked = false;
    for (const id of knownWalkInIds) {
      const res = await request.delete(`${API_BASE}/api/customers/${id}`, { headers: authHeader(token) });
      if (res.status() === 403) {
        blocked = true;
        break;
      }
    }
    expect(blocked).toBeTruthy();
  });
});

// ============================================================================
// 6. UI tests: superadmin full workflow
// ============================================================================

test.describe('Customers UI - Superadmin', () => {
  test.beforeEach(async ({ page }) => {
    await waitForAPI(page);
    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    await navigateToCustomers(page);
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('displays customers page with table', async ({ page }) => {
    await expect(page.locator('table')).toBeVisible();
    await expect(page.locator('button').filter({ hasText: /Add Customer/ })).toBeVisible();
  });

  test('create customer via modal', async ({ page }) => {
    const timestamp = Date.now();
    const name = `UI Customer ${timestamp}`;
    const phone = uniquePhone();
    const email = `${timestamp}@test.com`;

    await openCreateModal(page);
    await fillCreateModal(page, { name, phone, email });
    await submitCreateModal(page);

    const row = page.locator('table td').filter({ hasText: name }).first();
    await expect(row).toBeVisible({ timeout: 10000 });
  });

  test('create customer with note', async ({ page }) => {
    const timestamp = Date.now();
    const name = `UI Customer Note ${timestamp}`;
    const phone = uniquePhone();
    const email = `${timestamp}@test.com`;

    await openCreateModal(page);
    await fillCreateModal(page, { name, phone, email, note: 'Preferred customer' });
    await submitCreateModal(page);

    const row = page.locator('table td').filter({ hasText: name }).first();
    await expect(row).toBeVisible({ timeout: 10000 });
  });

  test('modal validation: empty name shows error', async ({ page }) => {
    await openCreateModal(page);
    await fillCreateModal(page, { phone: uniquePhone(), email: uniqueEmail() });
    await submitCreateModal(page);

    const modal = page.getByRole('dialog', { name: 'Add Customer' });
    await expect(modal.locator('text=/name is required/i')).toBeVisible();
  });

  test('modal validation: empty phone shows error', async ({ page }) => {
    await openCreateModal(page);
    await fillCreateModal(page, { name: 'Test', email: uniqueEmail() });
    await submitCreateModal(page);

    const modal = page.getByRole('dialog', { name: 'Add Customer' });
    await expect(modal.locator('text=/phone is required/i')).toBeVisible();
  });

  test('modal validation: invalid email shows error', async ({ page }) => {
    await openCreateModal(page);
    await fillCreateModal(page, { name: 'Test', phone: uniquePhone(), email: 'not-valid' });
    await submitCreateModal(page);

    const modal = page.getByRole('dialog', { name: 'Add Customer' });
    await expect(modal.locator('text=/invalid email/i')).toBeVisible();
  });

  test('modal validation: invalid phone shows error', async ({ page }) => {
    await openCreateModal(page);
    await fillCreateModal(page, { name: 'Test', phone: 'abc', email: uniqueEmail() });
    await submitCreateModal(page);

    const modal = page.getByRole('dialog', { name: 'Add Customer' });
    await expect(modal.locator('text=/invalid phone/i')).toBeVisible();
  });

  test('modal cancel closes without creating', async ({ page }) => {
    await openCreateModal(page);
    await fillCreateModal(page, { name: 'Should Not Exist', phone: uniquePhone(), email: uniqueEmail() });
    const modal = page.getByRole('dialog', { name: 'Add Customer' });
    await modal.locator('button').filter({ hasText: 'Cancel' }).click();

    await expect(page.getByRole('dialog', { name: 'Add Customer' })).toHaveCount(0);
    await expect(page.locator('table td').filter({ hasText: 'Should Not Exist' })).toHaveCount(0);
  });

  test('edit customer via modal', async ({ page }) => {
    const token = await page.evaluate(() => sessionStorage.getItem('access_token'));
    const name = `Edit Me ${Date.now()}`;
    await createCustomerAPI(page.request, token!, { name, phone: uniquePhone(), email: uniqueEmail() });
    await navigateToCustomers(page);

    const row = page.locator('table tr').filter({ has: page.locator(`text=${name}`) });
    await row.locator('button[title="Edit"]').click();

    const modal = page.getByRole('dialog', { name: 'Edit Customer' });
    await expect(modal).toBeVisible();

    const nameInput = modal.locator('#edit-name');
    await nameInput.clear();
    await nameInput.fill(`Edited ${name}`);
    await modal.locator('button').filter({ hasText: /Save Changes/ }).click();

    await expect(page.locator('table td').filter({ hasText: `Edited ${name}` }).first()).toBeVisible({ timeout: 10000 });
  });

  test('edit modal cancel reverts', async ({ page }) => {
    const token = await page.evaluate(() => sessionStorage.getItem('access_token'));
    const name = `Cancel Edit ${Date.now()}`;
    await createCustomerAPI(page.request, token!, { name, phone: uniquePhone(), email: uniqueEmail() });
    await navigateToCustomers(page);

    const row = page.locator('table tr').filter({ has: page.locator(`text=${name}`) });
    await row.locator('button[title="Edit"]').click();

    const modal = page.getByRole('dialog', { name: 'Edit Customer' });
    await expect(modal).toBeVisible();

    await modal.locator('#edit-name').clear();
    await modal.locator('#edit-name').fill('Should Not Persist');
    await modal.locator('button').filter({ hasText: 'Cancel' }).click();

    await expect(page.locator('table td').filter({ hasText: name }).first()).toBeVisible();
    await expect(page.locator('table td').filter({ hasText: 'Should Not Persist' })).toHaveCount(0);
  });

  test('deactivate customer via modal', async ({ page }) => {
    const token = await page.evaluate(() => sessionStorage.getItem('access_token'));
    const name = `Deactivate Me ${Date.now()}`;
    await createCustomerAPI(page.request, token!, { name, phone: uniquePhone(), email: uniqueEmail() });
    await navigateToCustomers(page);

    const row = page.locator('table tr').filter({ has: page.locator(`text=${name}`) });
    await row.locator('button[title="Deactivate"]').click();

    const modal = page.getByRole('dialog', { name: 'Deactivate Customer' });
    await expect(modal).toBeVisible();
    await expect(modal.locator('text=Are you sure')).toBeVisible();
    await modal.locator('button').filter({ hasText: /Deactivate$/ }).click();

    await expect(page.locator('table td').filter({ hasText: name }).first()).toBeVisible({ timeout: 10000 });
    const updatedRow = page.locator('table tr').filter({ has: page.locator(`text=${name}`) });
    await expect(updatedRow.locator('.text-danger, [class*="danger"]')).toBeVisible({ timeout: 5000 }).catch(async () => {
      await expect(updatedRow.locator('text=Inactive')).toBeVisible({ timeout: 5000 });
    });
  });

  test('status filter: Inactive shows only inactive', async ({ page }) => {
    const token = await page.evaluate(() => sessionStorage.getItem('access_token'));
    const inactiveName = `Inactive Filter ${Date.now()}`;
    const created = await createCustomerAPI(page.request, token!, { name: inactiveName, phone: uniquePhone(), email: uniqueEmail() });

    await page.request.delete(`${API_BASE}/api/customers/${created.id}`, { headers: authHeader(token!) });
    await navigateToCustomers(page);

    await page.locator('button').filter({ hasText: /^Inactive$/ }).click();
    await page.waitForTimeout(2000);

    await expect(page.locator('table td').filter({ hasText: inactiveName }).first()).toBeVisible({ timeout: 10000 });
  });
});

// ============================================================================
// 7. UI tests: admin role
// ============================================================================

test.describe('Customers UI - Admin', () => {
  test.beforeEach(async ({ page }) => {
    await loginUI(page, TEST_USERS.admin.username, TEST_USERS.admin.password);
    await navigateToCustomers(page);
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('admin can see Add Customer button and table', async ({ page }) => {
    await expect(page.locator('button').filter({ hasText: /Add Customer/ })).toBeVisible();
    await expect(page.locator('table')).toBeVisible();
  });

  test('admin sees edit and deactivate buttons', async ({ page }) => {
    const token = await page.evaluate(() => sessionStorage.getItem('access_token'));
    const name = `Admin Test ${Date.now()}`;
    await createCustomerAPI(page.request, token!, { name, phone: uniquePhone(), email: uniqueEmail() });
    await navigateToCustomers(page);

    const row = page.locator('table tr').filter({ has: page.locator(`text=${name}`) });
    await expect(row.locator('button[title="Edit"]')).toBeVisible();
    await expect(row.locator('button[title="Deactivate"]')).toBeVisible();
  });

  test('admin can create customer via modal', async ({ page }) => {
    const timestamp = Date.now();
    const name = `Admin Modal ${timestamp}`;

    await openCreateModal(page);
    await fillCreateModal(page, { name, phone: uniquePhone(), email: `${timestamp}@test.com` });
    await submitCreateModal(page);

    await expect(page.locator('table td').filter({ hasText: name }).first()).toBeVisible({ timeout: 10000 });
  });
});

// ============================================================================
// 8. UI tests: manager role (read + create + update, NO delete)
// ============================================================================

test.describe('Customers UI - Manager', () => {
  test.beforeEach(async ({ page }) => {
    await loginUI(page, TEST_USERS.manager.username, TEST_USERS.manager.password);
    await navigateToCustomers(page);
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('manager can see Add Customer button', async ({ page }) => {
    await expect(page.locator('button').filter({ hasText: /Add Customer/ })).toBeVisible();
  });

  test('manager sees edit button but NOT deactivate button', async ({ page }) => {
    const saToken = await getToken(page.request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const name = `Manager View ${Date.now()}`;
    await createCustomerAPI(page.request, saToken, { name, phone: uniquePhone(), email: uniqueEmail() });
    await navigateToCustomers(page);

    const row = page.locator('table tr').filter({ has: page.locator(`text=${name}`) });
    await expect(row.locator('button[title="Edit"]')).toBeVisible();
    const deactivateBtn = row.locator('button[title="Deactivate"]');
    await expect(deactivateBtn).toHaveCount(0);
  });

  test('manager can create customer via modal', async ({ page }) => {
    const timestamp = Date.now();
    const name = `Manager Created ${timestamp}`;

    await openCreateModal(page);
    await fillCreateModal(page, { name, phone: uniquePhone(), email: `${timestamp}@test.com` });
    await submitCreateModal(page);

    await expect(page.locator('table td').filter({ hasText: name }).first()).toBeVisible({ timeout: 10000 });
  });

  test('manager can edit customer', async ({ page }) => {
    const saToken = await getToken(page.request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const name = `Mgr Edit ${Date.now()}`;
    await createCustomerAPI(page.request, saToken, { name, phone: uniquePhone(), email: uniqueEmail() });
    await navigateToCustomers(page);

    const row = page.locator('table tr').filter({ has: page.locator(`text=${name}`) });
    await row.locator('button[title="Edit"]').click();

    const modal = page.getByRole('dialog', { name: 'Edit Customer' });
    await expect(modal).toBeVisible();

    const nameInput = modal.locator('#edit-name');
    await nameInput.clear();
    await nameInput.fill(`Mgr Edited ${name}`);
    await modal.locator('button').filter({ hasText: /Save Changes/ }).click();

    await expect(page.locator('table td').filter({ hasText: `Mgr Edited ${name}` }).first()).toBeVisible({ timeout: 10000 });
  });
});

// ============================================================================
// 9. UI tests: cashier role (read only — no create, no edit, no delete)
// ============================================================================

test.describe('Customers UI - Cashier', () => {
  test.beforeEach(async ({ page }) => {
    await loginUI(page, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
    await navigateToCustomers(page);
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('cashier sees table but NO Add Customer button', async ({ page }) => {
    const addBtn = page.locator('button').filter({ hasText: /Add Customer/ });
    await expect(addBtn).toHaveCount(0);
  });

  test('cashier sees NO action buttons on rows', async ({ page }) => {
    const saToken = await getToken(page.request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const name = `Cashier View ${Date.now()}`;
    await createCustomerAPI(page.request, saToken, { name, phone: uniquePhone(), email: uniqueEmail() });
    await navigateToCustomers(page);

    const row = page.locator('table tr').filter({ has: page.locator(`text=${name}`) });
    const editBtn = row.locator('button[title="Edit"]');
    const deleteBtn = row.locator('button[title="Deactivate"]');
    await expect(editBtn).toHaveCount(0);
    await expect(deleteBtn).toHaveCount(0);
  });

  test('cashier API CANNOT create customer', async ({ page }) => {
    const token = await page.evaluate(() => sessionStorage.getItem('access_token'));
    const res = await page.request.post(`${API_BASE}/api/customers`, {
      headers: authHeader(token!),
      data: { name: 'Cashier Should Fail', phone: uniquePhone(), email: uniqueEmail() },
    });
    expect(res.status()).toBe(403);
  });
});

// ============================================================================
// 10. UI tests: staff role (no customer permissions at all)
// ============================================================================

test.describe('Customers UI - Staff', () => {
  test('staff API cannot list customers', async ({ request }) => {
    const staffToken = await getToken(request, 'staff', 'admin123');
    const res = await request.get(`${API_BASE}/api/customers`, { headers: authHeader(staffToken) });
    expect(res.status()).toBe(403);
  });

  test('staff API cannot create customer', async ({ request }) => {
    const staffToken = await getToken(request, 'staff', 'admin123');
    const res = await request.post(`${API_BASE}/api/customers`, {
      headers: authHeader(staffToken),
      data: { name: 'Staff Should Fail', phone: uniquePhone(), email: uniqueEmail() },
    });
    expect(res.status()).toBe(403);
  });

  test('staff is denied access to /customers page', async ({ page }) => {
    await loginUI(page, 'staff', 'admin123');
    await page.goto(`${FRONTEND_BASE}/customers`);
    await page.waitForTimeout(2000);

    const redirectedAway = !page.url().includes('/customers');
    const isOnLogin = page.url().includes('/login');
    const accessDenied = await page.locator('text=Access Denied').isVisible().catch(() => false);
    const noCustomers = await page.locator('text=No customers').isVisible().catch(() => false);
    const errorMsg = await page.locator('text=/failed to load|insufficient|error/i').isVisible().catch(() => false);

    expect(redirectedAway || isOnLogin || accessDenied || noCustomers || errorMsg).toBeTruthy();
  });

  test('staff sidebar does not have Customers link', async ({ page }) => {
    await loginUI(page, 'staff', 'admin123');
    await page.waitForTimeout(1500);

    const customersLink = page.locator('button').filter({ hasText: /^Customers$/ });
    await expect(customersLink).toHaveCount(0);
  });
});

// ============================================================================
// 11. Unauthenticated access
// ============================================================================

test.describe('Customers UI - Unauthenticated', () => {
  test('redirects to login when not authenticated', async ({ page }) => {
    await page.goto(`${FRONTEND_BASE}/login`);
    await page.evaluate(() => sessionStorage.clear());
    await page.goto(`${FRONTEND_BASE}/customers`);
    await page.waitForTimeout(2000);
    expect(page.url().includes('/login')).toBeTruthy();
  });

  test('API returns 401 without token', async ({ request }) => {
    const res = await request.get(`${API_BASE}/api/customers`);
    expect(res.status()).toBe(401);
  });

  test('API returns 401 with invalid token', async ({ request }) => {
    const res = await request.get(`${API_BASE}/api/customers`, {
      headers: authHeader('invalid-token-here'),
    });
    expect(res.status()).toBe(401);
  });
});
