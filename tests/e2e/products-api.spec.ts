import { test, expect } from './fixtures';
import { TEST_USERS, API_BASE, authHeader, getToken as cachedGetToken } from './fixtures';

const getToken = cachedGetToken;

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

// ============================================================================
// Products API - Delete
// ============================================================================

test.describe('Products API - Delete', () => {

  test('DELETE /api/products/:id deletes a product', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    const sku = `DELPROD${Date.now()}`;
    const createRes = await request.post(`${API_BASE}/api/products`, {
      headers: authHeader(token),
      data: { name: `DeleteMe ${Date.now()}`, sku, price: 1000, cost: 500, stock: 1, status: 'active', store_id: 1 },
    });
    expect(createRes.ok(), `create failed: ${createRes.status()}: ${await createRes.text()}`).toBeTruthy();
    const created = await createRes.json();
    const productId = created.data?.id || created.id;

    const deleteRes = await request.delete(`${API_BASE}/api/products/${productId}`, {
      headers: authHeader(token),
    });
    expect(deleteRes.ok(), `delete failed: ${deleteRes.status()}: ${await deleteRes.text()}`).toBeTruthy();
    const body = await deleteRes.json();
    expect(body.status).toBe('deleted');
  });

  test('DELETE /api/products/:id returns 400 for invalid id', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const res = await request.delete(`${API_BASE}/api/products/not-a-number`, {
      headers: authHeader(token),
    });
    expect(res.status()).toBe(400);
  });

  test('DELETE /api/products/:id returns valid response for nonexistent product', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const res = await request.delete(`${API_BASE}/api/products/999999999`, {
      headers: authHeader(token),
    });
    // Delete is idempotent — returns 200 or 404
    expect([200, 404]).toContain(res.status());
  });

  test('DELETE /api/products/:id without auth returns 401', async ({ request }) => {
    const res = await request.delete(`${API_BASE}/api/products/1`);
    expect(res.status()).toBe(401);
  });

  test('DELETE /api/products/:id with restricted role returns 403', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
    const res = await request.delete(`${API_BASE}/api/products/1`, {
      headers: authHeader(token),
    });
    expect(res.status()).toBe(403);
  });
});
