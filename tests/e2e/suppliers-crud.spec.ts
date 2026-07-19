import { test, expect } from '@playwright/test';
import { TEST_USERS, API_BASE, authHeader, getToken as cachedGetToken } from './fixtures';

const getToken = cachedGetToken;

test.describe('Suppliers Product Assignment', () => {
  let supplierAId: number;
  let supplierBId: number;
  let productId: number;

  test.beforeEach(async ({ request }) => {
    const token = await getToken(request);
    const ts = Date.now();

    // Create two suppliers
    const aRes = await request.post(`${API_BASE}/api/suppliers`, {
      headers: authHeader(token),
      data: { name: `E2E SupA ${ts}-${Math.random().toString(36).slice(2, 6)}`, code: `E2E-PSA-${ts}-${Math.random().toString(36).slice(2, 6)}`, is_active: true },
    });
    expect(aRes.ok()).toBeTruthy();
    supplierAId = (await aRes.json()).data.id;

    const bRes = await request.post(`${API_BASE}/api/suppliers`, {
      headers: authHeader(token),
      data: { name: `E2E SupB ${ts}-${Math.random().toString(36).slice(2, 6)}`, code: `E2E-PSB-${ts}-${Math.random().toString(36).slice(2, 6)}`, is_active: true },
    });
    expect(bRes.ok()).toBeTruthy();
    supplierBId = (await bRes.json()).data.id;

    // Find an active product without existing suppliers to avoid unique constraint issues
    const prodRes = await request.get(`${API_BASE}/api/products?limit=10&status=active`, {
      headers: authHeader(token),
    });
    expect(prodRes.ok()).toBeTruthy();
    const prodBody = await prodRes.json();
    let foundProduct = false;
    for (const p of prodBody.data) {
      const supCheck = await request.get(`${API_BASE}/api/products/${p.id}/suppliers`, {
        headers: authHeader(token),
      });
      const supCheckBody = await supCheck.json();
      if (!supCheckBody.data || supCheckBody.data.length === 0) {
        productId = p.id;
        foundProduct = true;
        break;
      }
    }
    if (!foundProduct) {
      productId = prodBody.data[0].id;
    }

    // Link supplier A to product (not preferred, to avoid unique constraint)
    const linkRes = await request.post(`${API_BASE}/api/suppliers/${supplierAId}/products`, {
      headers: authHeader(token),
      data: { product_id: productId, unit_cost: 5000, lead_time_days: 7, is_preferred: false },
    });
    expect(linkRes.ok()).toBeTruthy();
  });

  test.afterEach(async ({ request }) => {
    if (!supplierAId || !productId) return;
    const token = await getToken(request);
    await request.delete(`${API_BASE}/api/suppliers/${supplierAId}/products/${productId}`, { headers: authHeader(token) });
    await request.delete(`${API_BASE}/api/suppliers/${supplierBId}/products/${productId}`, { headers: authHeader(token) });
    await request.delete(`${API_BASE}/api/suppliers/${supplierAId}`, { headers: authHeader(token) });
    await request.delete(`${API_BASE}/api/suppliers/${supplierBId}`, { headers: authHeader(token) });
  });

  test('POST /suppliers/:id/products links product to supplier', async ({ request }) => {
    const token = await getToken(request);
    // Find a second product to test linking (first is already linked by beforeEach)
    const prodRes = await request.get(`${API_BASE}/api/products?limit=20&status=active`, {
      headers: authHeader(token),
    });
    expect(prodRes.ok()).toBeTruthy();
    const prodBody = await prodRes.json();
    const secondProduct = prodBody.data.find((p: any) => p.id !== productId);
    const testProductId = secondProduct ? secondProduct.id : productId;
    const res = await request.post(`${API_BASE}/api/suppliers/${supplierAId}/products`, {
      headers: authHeader(token),
      data: { product_id: testProductId, unit_cost: 5000, lead_time_days: 7, is_preferred: false },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data.product_id).toBe(testProductId);
    expect(body.data.supplier_id).toBe(supplierAId);
    expect(body.data.is_preferred).toBeFalsy();
    // Cleanup: unlink
    await request.delete(`${API_BASE}/api/suppliers/${supplierAId}/products/${testProductId}`, { headers: authHeader(token) });
  });

  test('GET /suppliers/:id/products lists linked products', async ({ request }) => {
    const token = await getToken(request);
    const res = await request.get(`${API_BASE}/api/suppliers/${supplierAId}/products`, {
      headers: authHeader(token),
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data.length).toBeGreaterThanOrEqual(1);
    const link = body.data.find((p: any) => p.product_id === productId);
    expect(link).toBeTruthy();
    expect(link.product_name).toBeTruthy();
    expect(link.product_sku).toBeTruthy();
  });

  test('GET /products/:id/suppliers lists suppliers for a product', async ({ request }) => {
    const token = await getToken(request);
    const res = await request.get(`${API_BASE}/api/products/${productId}/suppliers`, {
      headers: authHeader(token),
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data.length).toBeGreaterThanOrEqual(1);
    const link = body.data.find((s: any) => s.supplier_id === supplierAId);
    expect(link).toBeTruthy();
    expect(link.supplier_name).toBeTruthy();
    expect(link.supplier_code).toBeTruthy();
  });

  test('POST /suppliers/:id/products/:productId/preferred sets preferred supplier', async ({ request }) => {
    const token = await getToken(request);
    // First link supplier B to same product
    const linkRes = await request.post(`${API_BASE}/api/suppliers/${supplierBId}/products`, {
      headers: authHeader(token),
      data: { product_id: productId, unit_cost: 6000, lead_time_days: 10, is_preferred: false },
    });
    expect(linkRes.ok()).toBeTruthy();

    // Set supplier B as preferred
    const prefRes = await request.post(`${API_BASE}/api/suppliers/${supplierBId}/products/${productId}/preferred`, {
      headers: authHeader(token),
    });
    expect(prefRes.ok()).toBeTruthy();

    // Verify: supplier B is now preferred, supplier A is not
    const supsRes = await request.get(`${API_BASE}/api/products/${productId}/suppliers`, {
      headers: authHeader(token),
    });
    expect(supsRes.ok()).toBeTruthy();
    const supsBody = await supsRes.json();
    const supA = supsBody.data.find((s: any) => s.supplier_id === supplierAId);
    const supB = supsBody.data.find((s: any) => s.supplier_id === supplierBId);
    expect(supA.is_preferred).toBeFalsy();
    expect(supB.is_preferred).toBeTruthy();
  });

  test('PUT /suppliers/:id/products/:productId updates link metadata', async ({ request }) => {
    const token = await getToken(request);
    const res = await request.put(`${API_BASE}/api/suppliers/${supplierAId}/products/${productId}`, {
      headers: authHeader(token),
      data: { supplier_sku: 'CUSTOM-SKU-001', unit_cost: 7500, lead_time_days: 14, is_preferred: false },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data.supplier_sku).toBe('CUSTOM-SKU-001');
    expect(body.data.unit_cost).toBe(7500);
    expect(body.data.lead_time_days).toBe(14);
  });

  test('DELETE /suppliers/:id/products/:productId unlinks product', async ({ request }) => {
    const token = await getToken(request);
    const res = await request.delete(`${API_BASE}/api/suppliers/${supplierAId}/products/${productId}`, {
      headers: authHeader(token),
    });
    expect(res.ok()).toBeTruthy();

    // Verify: supplier A no longer linked
    const supsRes = await request.get(`${API_BASE}/api/products/${productId}/suppliers`, {
      headers: authHeader(token),
    });
    expect(supsRes.ok()).toBeTruthy();
    const supsBody = await supsRes.json();
    const supA = supsBody.data.find((s: any) => s.supplier_id === supplierAId);
    expect(supA).toBeUndefined();
  });

  test('POST /suppliers/:id/products without auth returns 401', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/suppliers/${supplierBId}/products`, {
      data: { product_id: productId, unit_cost: 5000, lead_time_days: 7, is_preferred: false },
    });
    expect(res.status()).toBe(401);
  });
});

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
