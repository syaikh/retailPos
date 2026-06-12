import { test, expect } from '@playwright/test';
import { TEST_USERS, API_BASE, authHeader, waitForAPI, FRONTEND_BASE } from './fixtures';

// ============================================================================
// Helpers
// ============================================================================

async function getToken(request: any, username: string, password: string) {
  const res = await request.post(`${API_BASE}/api/login`, {
    data: { username, password },
  });
  expect(res.ok(), `login failed for ${username}: ${res.status()}`).toBeTruthy();
  const body = await res.json();
  return body.access_token;
}

async function createCustomerAPI(request: any, token: string, data: Record<string, any>) {
  const res = await request.post(`${API_BASE}/api/customers`, {
    headers: authHeader(token),
    data,
  });
  const body = await res.text();
  if (!res.ok()) {
    // If unique constraint violation, retry with a different phone
    if (body.includes('failed to create customer') && data.phone) {
      const retryRes = await request.post(`${API_BASE}/api/customers`, {
        headers: authHeader(token),
        data: { ...data, phone: `08${Date.now()}${Math.floor(Math.random() * 10000)}`.slice(0, 13) },
      });
      const retryBody = await retryRes.text();
      expect(retryRes.ok(), `create customer retry failed: status=${retryRes.status()} body=${retryBody}`).toBeTruthy();
      return JSON.parse(retryBody).data;
    }
  }
  expect(res.ok(), `create customer failed: status=${res.status()} body=${body}`).toBeTruthy();
  return JSON.parse(body).data;
}

function uniquePhone() {
  return `08${Date.now()}${Math.floor(Math.random() * 1000)}`.slice(0, 13);
}

function uniqueEmail() {
  return `e2e.${Date.now()}@test.com`;
}

async function login(page: any, username: string, password: string) {
  await page.goto(`${FRONTEND_BASE}/login`);
  await page.waitForTimeout(1000);
  // Clear any stale session
  await page.evaluate(() => sessionStorage.clear());
  await page.reload();
  await page.waitForSelector('#username', { state: 'visible', timeout: 15000 });
  await page.fill('#username', username);
  await page.fill('#password', password);
  await page.click('button[type="submit"]');
  await page.waitForTimeout(2000);
  // Wait for redirect to home
  await page.waitForFunction(() => {
    const path = window.location.hash || window.location.pathname;
    return path === '/' || path === '' || !path.includes('login');
  }, { timeout: 15000 });
}

async function logout(page: any) {
  await page.goto(`${FRONTEND_BASE}/login`);
  await page.evaluate(() => sessionStorage.clear());
  await page.reload();
  await page.waitForSelector('#username', { state: 'visible', timeout: 10000 });
}

// ============================================================================
// 1. API tests: RBAC for CRUD across all roles
// ============================================================================

test.describe('Customers API - RBAC', () => {
  // ─── READ ────────────────────────────────────────────────────────────

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

  // ─── CREATE ──────────────────────────────────────────────────────────

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

  // ─── UPDATE ──────────────────────────────────────────────────────────

  test('superadmin can update customer', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const created = await createCustomerAPI(request, token, { name: 'SA Update Me', phone: uniquePhone() });
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
    const created = await createCustomerAPI(request, token, { name: 'Admin Update Me', phone: uniquePhone() });
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
    const created = await createCustomerAPI(request, token, { name: 'Manager Update Me', phone: uniquePhone() });
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
    const created = await createCustomerAPI(request, saToken, { name: 'No Update Cashier', phone: uniquePhone() });
    const cashierToken = await getToken(request, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
    const res = await request.put(`${API_BASE}/api/customers/${created.id}`, {
      headers: authHeader(cashierToken),
      data: { name: 'Hacked', is_active: true },
    });
    expect(res.status()).toBe(403);
  });

  // ─── DELETE / DEACTIVATE ─────────────────────────────────────────────

  test('superadmin can deactivate customer', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const created = await createCustomerAPI(request, token, { name: 'SA Deactivate Me', phone: uniquePhone() });
    const delRes = await request.delete(`${API_BASE}/api/customers/${created.id}`, { headers: authHeader(token) });
    expect(delRes.ok()).toBeTruthy();
    const getRes = await request.get(`${API_BASE}/api/customers/${created.id}`, { headers: authHeader(token) });
    const getBody = await getRes.json();
    expect(getBody.data.is_active).toBe(false);
  });

  test('admin can deactivate customer', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.admin.username, TEST_USERS.admin.password);
    const created = await createCustomerAPI(request, token, { name: 'Admin Deactivate Me', phone: uniquePhone() });
    const delRes = await request.delete(`${API_BASE}/api/customers/${created.id}`, { headers: authHeader(token) });
    expect(delRes.ok()).toBeTruthy();
    const getRes = await request.get(`${API_BASE}/api/customers/${created.id}`, { headers: authHeader(token) });
    const getBody = await getRes.json();
    expect(getBody.data.is_active).toBe(false);
  });

  test('manager CANNOT deactivate customer (403)', async ({ request }) => {
    const saToken = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const created = await createCustomerAPI(request, saToken, { name: 'No Delete Manager', phone: uniquePhone() });
    const managerToken = await getToken(request, TEST_USERS.manager.username, TEST_USERS.manager.password);
    const res = await request.delete(`${API_BASE}/api/customers/${created.id}`, { headers: authHeader(managerToken) });
    expect(res.status()).toBe(403);
  });

  test('cashier CANNOT deactivate customer (403)', async ({ request }) => {
    const saToken = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const created = await createCustomerAPI(request, saToken, { name: 'No Delete Cashier', phone: uniquePhone() });
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
      data: { name: '', phone: uniquePhone() },
    });
    expect(res.status()).toBe(400);
    const body = await res.json();
    expect(body.error).toContain('name');
  });

  test('rejects whitespace-only name', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/customers`, {
      headers: authHeader(token),
      data: { name: '   ', phone: uniquePhone() },
    });
    expect(res.status()).toBe(400);
  });

  test('rejects invalid email format', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/customers`, {
      headers: authHeader(token),
      data: { name: 'Valid Name', email: 'not-an-email' },
    });
    expect(res.status()).toBe(400);
    const body = await res.json();
    expect(body.error.toLowerCase()).toContain('email');
  });

  test('accepts valid email', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/customers`, {
      headers: authHeader(token),
      data: { name: 'Valid Email Test', email: 'valid@example.com' },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data.email).toBe('valid@example.com');
  });

  test('accepts no email (optional field)', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/customers`, {
      headers: authHeader(token),
      data: { name: 'No Email Test' },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    // Optional fields may be null or undefined when not provided
    expect(body.data.email == null).toBeTruthy();
  });

  test('rejects invalid phone format', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/customers`, {
      headers: authHeader(token),
      data: { name: 'Bad Phone', phone: 'abc' },
    });
    expect(res.status()).toBe(400);
    const body = await res.json();
    expect(body.error.toLowerCase()).toContain('phone');
  });

  test('accepts various phone formats', async ({ request }) => {
    const formats = [`08${Date.now()}`, `+62${Date.now()}`, `0812-345-${Date.now().toString().slice(-4)}`];
    for (const phone of formats) {
      const uniqueName = `Phone ${Date.now()}`;
      const res = await request.post(`${API_BASE}/api/customers`, {
        headers: authHeader(token),
        data: { name: uniqueName, phone },
      });
      expect(res.ok(), `phone "${phone}" should be valid: ${await res.text()}`).toBeTruthy();
    }
  });

  test('accepts no phone (optional field)', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/customers`, {
      headers: authHeader(token),
      data: { name: 'No Phone Test' },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data.phone == null).toBeTruthy();
  });

  test('long name (>200 chars) is rejected', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/customers`, {
      headers: authHeader(token),
      data: { name: 'A'.repeat(256) },
    });
    expect(res.status()).toBe(400);
  });

  test('name exactly 200 chars is accepted', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/customers`, {
      headers: authHeader(token),
      data: { name: 'A'.repeat(200) },
    });
    expect(res.ok()).toBeTruthy();
  });

  test('name >200 chars is rejected', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/customers`, {
      headers: authHeader(token),
      data: { name: 'A'.repeat(201) },
    });
    expect(res.status()).toBe(400);
  });

  test('update validates email too', async ({ request }) => {
    const created = await createCustomerAPI(request, token, { name: 'Update Validation', phone: uniquePhone() });
    const res = await request.put(`${API_BASE}/api/customers/${created.id}`, {
      headers: authHeader(token),
      data: { email: 'bad-email' },
    });
    expect(res.status()).toBe(400);
  });

  test('update validates name is not empty', async ({ request }) => {
    const created = await createCustomerAPI(request, token, { name: 'Update Name Validation', phone: uniquePhone() });
    const res = await request.put(`${API_BASE}/api/customers/${created.id}`, {
      headers: authHeader(token),
      data: { name: '' },
    });
    expect(res.status()).toBe(400);
  });

  test('name is trimmed on create', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/customers`, {
      headers: authHeader(token),
      data: { name: '  Trim Me  ' },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data.name).toBe('Trim Me');
  });
});

// ============================================================================
// 3. API tests: Audit logging
// ============================================================================

test.describe('Customers API - Audit Logging', () => {
  let token: string;

  test.beforeAll(async ({ request }) => {
    token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
  });

  test('creating a customer generates an audit log', async ({ request }) => {
    const created = await createCustomerAPI(request, token, { name: 'Audit Create Test', phone: uniquePhone() });
    // Wait for async audit log write
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
    const created = await createCustomerAPI(request, token, { name: 'Audit Update Before', phone: uniquePhone() });
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
    const created = await createCustomerAPI(request, token, { name: 'Audit Delete Test', phone: uniquePhone() });
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
// 4. API tests: Walk-in customer filtering
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
    const created = await createCustomerAPI(request, token, { name: 'Listed Customer', phone: uniquePhone() });
    const res = await request.get(`${API_BASE}/api/customers?limit=200`, { headers: authHeader(token) });
    const body = await res.json();
    const found = body.data?.find((c: any) => c.id === created.id);
    expect(found).toBeTruthy();
  });

  test('cannot modify walk-in customer via API', async ({ request, page }) => {
    // Find the walk-in customer by querying with include_walk_in or by known ID
    // Walk-in customer is typically created during seed with a known pattern
    // Try to get all customers including walk-in by checking the DB directly
    // Since GetAllCustomers filters walk-in, we use a direct approach: try common IDs
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
// 5. UI tests: superadmin full workflow
// ============================================================================

test.describe('Customers UI - Superadmin', () => {
  test.beforeEach(async ({ page }) => {
    await waitForAPI(page);
    await login(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    await page.goto('/customers');
    await page.waitForTimeout(2000);
  });

  test.afterEach(async ({ page }) => {
    await logout(page);
  });

  test('displays customers page with table', async ({ page }) => {
    // Wait for SPA to fully load and auth to restore
    await page.waitForFunction(() => {
      // Check if the page has rendered by looking for the heading
      const h1 = document.querySelector('h1');
      return h1 && h1.textContent === 'Customers';
    }, { timeout: 15000 });
    await page.waitForTimeout(1000);
    await expect(page.locator('table')).toBeVisible();
    await expect(page.locator('button').filter({ hasText: /Create/ })).toBeVisible();
  });

  test('create customer via UI', async ({ page }) => {
    const timestamp = Date.now();
    const name = `UI Customer ${timestamp}`;
    const phone = uniquePhone();

    await page.fill('input[placeholder*="Name"]', name);
    await page.fill('input[placeholder*="Phone"]', phone);
    await page.fill('input[placeholder*="Email"]', `${timestamp}@test.com`);
    await page.locator('button').filter({ hasText: /Create/ }).click();
    await page.waitForTimeout(1500);

    const row = page.locator('td').filter({ hasText: name });
    await expect(row).toBeVisible();
  });

  test('create button requires name', async ({ page }) => {
    await page.fill('input[placeholder*="Name"]', '');
    await page.locator('button').filter({ hasText: /Create/ }).click();
    await page.waitForTimeout(500);
    const error = page.locator('text=/name is required/i');
    await expect(error).toBeVisible();
  });

  test('edit customer inline', async ({ page }) => {
    const token = await page.evaluate(() => sessionStorage.getItem('access_token'));
    const name = `Edit Me ${Date.now()}`;
    await createCustomerAPI(page.request, token!, { name, phone: uniquePhone() });
    await page.reload();
    await page.waitForTimeout(1500);

    const row = page.locator('tr').filter({ has: page.locator(`text=${name}`) });
    await row.locator('button[title="Edit"]').click();
    await page.waitForTimeout(500);

    // Row should turn into edit mode with inputs
    const editInputs = page.locator('table input');
    await expect(editInputs.first()).toBeVisible();

    // Clear and type new name
    await editInputs.first().fill(`Edited ${name}`);
    await page.locator('button[title="Save"]').click();
    await page.waitForTimeout(1500);

    await expect(page.locator(`text=Edited ${name}`)).toBeVisible();
  });

  test('cancel editing reverts row', async ({ page }) => {
    const token = await page.evaluate(() => sessionStorage.getItem('access_token'));
    const name = `Cancel Me ${Date.now()}`;
    await createCustomerAPI(page.request, token!, { name, phone: uniquePhone() });
    await page.reload();
    await page.waitForTimeout(1500);

    const row = page.locator('tr').filter({ has: page.locator(`text=${name}`) });
    await row.locator('button[title="Edit"]').click();
    await page.waitForTimeout(500);

    await page.locator('button[title="Cancel"]').click();
    await page.waitForTimeout(500);

    const editBtn = row.locator('button[title="Edit"]');
    await expect(editBtn).toBeVisible();
  });

  test('deactivate customer with confirmation', async ({ page }) => {
    const token = await page.evaluate(() => sessionStorage.getItem('access_token'));
    const name = `Deactivate Me ${Date.now()}`;
    await createCustomerAPI(page.request, token!, { name, phone: uniquePhone() });
    await page.reload();
    await page.waitForTimeout(1500);

    const row = page.locator('tr').filter({ has: page.locator(`text=${name}`) });

    page.once('dialog', async (dialog) => {
      expect(dialog.message()).toContain('Deactivate');
      await dialog.accept();
    });

    await row.locator('button[title="Deactivate"]').click();
    await page.waitForTimeout(1500);

    const updatedRow = page.locator('tr').filter({ has: page.locator(`text=${name}`) });
    await expect(updatedRow.locator('text=Inactive')).toBeVisible();
  });
});

// ============================================================================
// 6. UI tests: admin role
// ============================================================================

test.describe('Customers UI - Admin', () => {
  test.beforeEach(async ({ page }) => {
    await login(page, TEST_USERS.admin.username, TEST_USERS.admin.password);
    await page.goto('/customers');
    await page.waitForTimeout(1500);
  });

  test.afterEach(async ({ page }) => {
    await logout(page);
  });

  test('admin can see create form and table', async ({ page }) => {
    await expect(page.locator('h1')).toHaveText('Customers');
    await expect(page.locator('button').filter({ hasText: /Create/ })).toBeVisible();
    await expect(page.locator('table')).toBeVisible();
  });

  test('admin sees edit and deactivate buttons', async ({ page }) => {
    const token = await page.evaluate(() => sessionStorage.getItem('access_token'));
    const name = `Admin Test ${Date.now()}`;
    await createCustomerAPI(page.request, token!, { name, phone: uniquePhone() });
    await page.reload();
    await page.waitForTimeout(1500);

    const row = page.locator('tr').filter({ has: page.locator(`text=${name}`) });
    await expect(row.locator('button[title="Edit"]')).toBeVisible();
    await expect(row.locator('button[title="Deactivate"]')).toBeVisible();
  });
});

// ============================================================================
// 7. UI tests: manager role (read + create + update, NO delete)
// ============================================================================

test.describe('Customers UI - Manager', () => {
  test.beforeEach(async ({ page }) => {
    await login(page, TEST_USERS.manager.username, TEST_USERS.manager.password);
    await page.goto('/customers');
    await page.waitForTimeout(1500);
  });

  test.afterEach(async ({ page }) => {
    await logout(page);
  });

  test('manager can see create form', async ({ page }) => {
    await expect(page.locator('h1')).toHaveText('Customers');
    await expect(page.locator('button').filter({ hasText: /Create/ })).toBeVisible();
  });

  test('manager sees edit button but NOT deactivate button', async ({ page }) => {
    const saToken = await getToken(page.request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const name = `Manager View ${Date.now()}`;
    await createCustomerAPI(page.request, saToken, { name, phone: uniquePhone() });
    await page.reload();
    await page.waitForTimeout(1500);

    const row = page.locator('tr').filter({ has: page.locator(`text=${name}`) });
    await expect(row.locator('button[title="Edit"]')).toBeVisible();
    const deactivateBtn = row.locator('button[title="Deactivate"]');
    await expect(deactivateBtn).toHaveCount(0);
  });

  test('manager can create customer', async ({ page }) => {
    const timestamp = Date.now();
    const name = `Manager Created ${timestamp}`;

    await page.fill('input[placeholder*="Name"]', name);
    await page.fill('input[placeholder*="Phone"]', uniquePhone());
    await page.locator('button').filter({ hasText: /Create/ }).click();
    await page.waitForTimeout(1500);

    await expect(page.locator(`text=${name}`)).toBeVisible();
  });

  test('manager can edit customer', async ({ page }) => {
    const saToken = await getToken(page.request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const name = `Mgr Edit ${Date.now()}`;
    await createCustomerAPI(page.request, saToken, { name, phone: uniquePhone() });
    await page.reload();
    await page.waitForTimeout(1500);

    const row = page.locator('tr').filter({ has: page.locator(`text=${name}`) });
    await row.locator('button[title="Edit"]').click();
    await page.waitForTimeout(500);

    await page.locator('table input').first().fill(`Mgr Edited ${name}`);
    await page.locator('button[title="Save"]').click();
    await page.waitForTimeout(1500);

    await expect(page.locator(`text=Mgr Edited ${name}`)).toBeVisible();
  });
});

// ============================================================================
// 8. UI tests: cashier role (read only — no create, no edit, no delete)
// ============================================================================

test.describe('Customers UI - Cashier', () => {
  test.beforeEach(async ({ page }) => {
    await login(page, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
    await page.goto('/customers');
    await page.waitForTimeout(1500);
  });

  test.afterEach(async ({ page }) => {
    await logout(page);
  });

  test('cashier sees table but NO create form', async ({ page }) => {
    await expect(page.locator('h1')).toHaveText('Customers');
    // Cashier does NOT have customer:create permission -> no create form
    const createBtn = page.locator('button').filter({ hasText: /Create/ });
    await expect(createBtn).toHaveCount(0);
  });

  test('cashier sees NO action buttons on rows', async ({ page }) => {
    const saToken = await getToken(page.request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const name = `Cashier View ${Date.now()}`;
    await createCustomerAPI(page.request, saToken, { name, phone: uniquePhone() });
    await page.reload();
    await page.waitForTimeout(1500);

    const row = page.locator('tr').filter({ has: page.locator(`text=${name}`) });
    const editBtn = row.locator('button[title="Edit"]');
    const deleteBtn = row.locator('button[title="Deactivate"]');
    await expect(editBtn).toHaveCount(0);
    await expect(deleteBtn).toHaveCount(0);
  });

  test('cashier API CANNOT create customer', async ({ page }) => {
    const token = await page.evaluate(() => sessionStorage.getItem('access_token'));
    const res = await page.request.post(`${API_BASE}/api/customers`, {
      headers: authHeader(token!),
      data: { name: 'Cashier Should Fail', phone: uniquePhone() },
    });
    expect(res.status()).toBe(403);
  });
});

// ============================================================================
// 9. UI tests: staff role (no customer permissions at all)
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
      data: { name: 'Staff Should Fail', phone: uniquePhone() },
    });
    expect(res.status()).toBe(403);
  });

  test('staff is denied access to /customers page', async ({ page }) => {
    await login(page, 'staff', 'admin123');
    await page.goto('/customers');
    await page.waitForTimeout(2000);

    // Staff has no customer:read permission, so API returns 403
    // The SPA shows the page but with no customer data (empty table or "No customers")
    const isOnLogin = page.url().includes('/login');
    const accessDenied = await page.locator('text=Access Denied').isVisible().catch(() => false);
    const noCustomers = await page.locator('text=No customers').isVisible().catch(() => false);
    const errorMsg = await page.locator('text=/failed to load|insufficient|error/i').isVisible().catch(() => false);

    // Staff should either be redirected, see access denied, or see empty/error state
    // They should NOT see actual customer data
    expect(isOnLogin || accessDenied || noCustomers || errorMsg).toBeTruthy();

    await logout(page);
  });

  test('staff sidebar does not have Customers link', async ({ page }) => {
    await login(page, 'staff', 'admin123');
    await page.waitForTimeout(1500);

    // Staff sidebar does not include Customers
    const customersLink = page.locator('button').filter({ hasText: /^Customers$/ });
    await expect(customersLink).toHaveCount(0);

    await logout(page);
  });
});

// ============================================================================
// 10. Unauthenticated access
// ============================================================================

test.describe('Customers UI - Unauthenticated', () => {
  test('redirects to login when not authenticated', async ({ page }) => {
    await page.goto('/login');
    await page.evaluate(() => sessionStorage.clear());
    await page.goto('/customers');
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
