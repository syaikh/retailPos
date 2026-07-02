import { test, expect } from '@playwright/test';
import { TEST_USERS, API_BASE, authHeader } from './fixtures';

async function getToken(request: any) {
  const res = await request.post(`${API_BASE}/api/login`, {
    data: { username: TEST_USERS.superadmin.username, password: TEST_USERS.superadmin.password },
  });
  const body = await res.json();
  return body.access_token;
}

test.describe('Products API - Next SKU', () => {

  test('GET /api/products/next-sku returns SKU string', async ({ request }) => {
    const token = await getToken(request);
    const res = await request.get(`${API_BASE}/api/products/next-sku`, {
      headers: authHeader(token),
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data).toBeTruthy();
    expect(typeof body.data).toBe('string');
    expect(body.data.length).toBeGreaterThan(0);
  });

  test('GET /api/products/next-sku returns unique values on successive calls', async ({ request }) => {
    const token = await getToken(request);
    const res1 = await request.get(`${API_BASE}/api/products/next-sku`, {
      headers: authHeader(token),
    });
    const res2 = await request.get(`${API_BASE}/api/products/next-sku`, {
      headers: authHeader(token),
    });
    expect(res1.ok()).toBeTruthy();
    expect(res2.ok()).toBeTruthy();
    const body1 = await res1.json();
    const body2 = await res2.json();
    expect(body1.data).not.toBe(body2.data);
  });

  test('GET /api/products/next-sku without auth returns 401', async ({ request }) => {
    const res = await request.get(`${API_BASE}/api/products/next-sku`);
    expect(res.status()).toBe(401);
  });

});

test.describe('Products API - Bulk Status Update', () => {

  test('POST /api/products/bulk/status updates product statuses', async ({ request }) => {
    const token = await getToken(request);
    const listRes = await request.get(`${API_BASE}/api/products?limit=3`, {
      headers: authHeader(token),
    });
    expect(listRes.ok()).toBeTruthy();
    const listBody = await listRes.json();
    const products = listBody.data || [];
    if (products.length === 0) return;
    const ids = products.map((p: any) => p.id);

    const res = await request.post(`${API_BASE}/api/products/bulk/status`, {
      headers: authHeader(token),
      data: { ids, is_active: false },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.status).toBe('updated');
    expect(body.updated_count).toBe(ids.length);

    const revertRes = await request.post(`${API_BASE}/api/products/bulk/status`, {
      headers: authHeader(token),
      data: { ids, is_active: true },
    });
    expect(revertRes.ok()).toBeTruthy();
  });

  test('POST /api/products/bulk/status with empty IDs returns 400', async ({ request }) => {
    const token = await getToken(request);
    const res = await request.post(`${API_BASE}/api/products/bulk/status`, {
      headers: authHeader(token),
      data: { ids: [], is_active: false },
    });
    expect(res.status()).toBe(400);
  });

  test('POST /api/products/bulk/status without auth returns 401', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/products/bulk/status`, {
      data: { ids: [1], is_active: false },
    });
    expect(res.status()).toBe(401);
  });

  test('POST /api/products/bulk/status with restricted role returns 403', async ({ request }) => {
    const loginRes = await request.post(`${API_BASE}/api/login`, {
      data: { username: TEST_USERS.cashier.username, password: TEST_USERS.cashier.password },
    });
    const loginBody = await loginRes.json();
    const res = await request.post(`${API_BASE}/api/products/bulk/status`, {
      headers: authHeader(loginBody.access_token),
      data: { ids: [1], is_active: false },
    });
    expect(res.status()).toBe(403);
  });

});
