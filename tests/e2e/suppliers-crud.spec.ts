import { test, expect } from '@playwright/test';
import { TEST_USERS, API_BASE, authHeader, getToken as cachedGetToken } from './fixtures';

const getToken = cachedGetToken;

test.describe('Suppliers CRUD API', () => {
  let createdSupplierId: number;
  const testCode = `E2E-SUP-${Date.now()}`;

  test('POST /api/suppliers creates a supplier', async ({ request }) => {
    const token = await getToken(request);
    const res = await request.post(`${API_BASE}/api/suppliers`, {
      headers: authHeader(token),
      data: {
        name: 'E2E Test Supplier',
        code: testCode,
        contact_name: 'Test Contact',
        phone: '081234567890',
        email: 'test@supplier.com',
        address: 'Jl. Test No. 1',
        is_active: true,
      },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data).toBeTruthy();
    expect(body.data.id).toBeTruthy();
    expect(body.data.name).toBe('E2E Test Supplier');
    expect(body.data.code).toBe(testCode);
    createdSupplierId = body.data.id;
  });

  test('GET /api/suppliers lists suppliers', async ({ request }) => {
    const token = await getToken(request);
    const res = await request.get(`${API_BASE}/api/suppliers?limit=10`, {
      headers: authHeader(token),
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data).toBeDefined();
    expect(Array.isArray(body.data)).toBeTruthy();
    expect(body.total).toBeGreaterThanOrEqual(1);
  });

  test('GET /api/suppliers searches by name', async ({ request }) => {
    const token = await getToken(request);
    const res = await request.get(`${API_BASE}/api/suppliers?search=E2E`, {
      headers: authHeader(token),
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data.length).toBeGreaterThanOrEqual(1);
  });

  test('PUT /api/suppliers/:id updates a supplier', async ({ request }) => {
    if (!createdSupplierId) return;
    const token = await getToken(request);
    const res = await request.put(`${API_BASE}/api/suppliers/${createdSupplierId}`, {
      headers: authHeader(token),
      data: {
        name: 'E2E Updated Supplier',
        contact_name: 'Updated Contact',
      },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data.name).toBe('E2E Updated Supplier');
  });

  test('DELETE /api/suppliers/:id soft deletes a supplier', async ({ request }) => {
    if (!createdSupplierId) return;
    const token = await getToken(request);
    const res = await request.delete(`${API_BASE}/api/suppliers/${createdSupplierId}`, {
      headers: authHeader(token),
    });
    expect(res.ok()).toBeTruthy();
  });

  test('POST /api/suppliers without auth returns 401', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/suppliers`, {
      data: { name: 'Test', code: 'TEST', is_active: true },
    });
    expect(res.status()).toBe(401);
  });

  test('POST /api/suppliers with duplicate code returns error', async ({ request }) => {
    const token = await getToken(request);
    const res = await request.post(`${API_BASE}/api/suppliers`, {
      headers: authHeader(token),
      data: {
        name: 'Duplicate Supplier',
        code: testCode,
        is_active: true,
      },
    });
    expect(res.ok()).toBeFalsy();
  });
});
