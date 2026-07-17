import { test, expect } from '@playwright/test';
import { TEST_USERS, API_BASE, authHeader, getToken as cachedGetToken } from './fixtures';

const getToken = cachedGetToken;

test.describe('Product-Supplier Linking API', () => {
  let supplierId: number;
  let productId: number;

  test.beforeAll(async ({ request }) => {
    const token = await getToken(request);

    const supRes = await request.get(`${API_BASE}/api/suppliers?limit=1&is_active=true`, {
      headers: authHeader(token),
    });
    const supBody = await supRes.json();
    if (supBody.data && supBody.data.length > 0) {
      supplierId = supBody.data[0].id;
    }

    const prodRes = await request.get(`${API_BASE}/api/products?limit=1&status=active`, {
      headers: authHeader(token),
    });
    const prodBody = await prodRes.json();
    if (prodBody.data && prodBody.data.length > 0) {
      productId = prodBody.data[0].id;
    }
  });

  test('GET /products/:id/suppliers lists linked suppliers for a product', async ({ request }) => {
    if (!productId) return;
    const token = await getToken(request);
    const res = await request.get(`${API_BASE}/api/products/${productId}/suppliers`, {
      headers: authHeader(token),
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data).toBeDefined();
    expect(Array.isArray(body.data)).toBeTruthy();
  });

  test('POST /suppliers/:id/products links supplier to product', async ({ request }) => {
    if (!productId || !supplierId) return;
    const token = await getToken(request);
    await request.delete(`${API_BASE}/api/suppliers/${supplierId}/products/${productId}`, {
      headers: authHeader(token),
    });
    const res = await request.post(`${API_BASE}/api/suppliers/${supplierId}/products`, {
      headers: authHeader(token),
      data: {
        product_id: productId,
        supplier_sku: 'E2E-SKU-001',
        unit_cost: 8500,
        lead_time_days: 7,
        is_preferred: false,
      },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data).toBeTruthy();
  });

  test('GET /suppliers/:id/products lists products for supplier', async ({ request }) => {
    if (!supplierId) return;
    const token = await getToken(request);
    const res = await request.get(`${API_BASE}/api/suppliers/${supplierId}/products`, {
      headers: authHeader(token),
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data).toBeDefined();
    expect(Array.isArray(body.data)).toBeTruthy();
  });

  test('PUT /suppliers/:id/products/:productId updates link', async ({ request }) => {
    if (!supplierId || !productId) return;
    const token = await getToken(request);
    const res = await request.put(
      `${API_BASE}/api/suppliers/${supplierId}/products/${productId}`,
      {
        headers: authHeader(token),
        data: {
          supplier_sku: 'E2E-SKU-001-UPDATED',
          unit_cost: 8200,
        },
      }
    );
    expect(res.ok()).toBeTruthy();
  });

  test('DELETE /suppliers/:id/products/:productId unlinks supplier', async ({ request }) => {
    if (!supplierId || !productId) return;
    const token = await getToken(request);
    const res = await request.delete(
      `${API_BASE}/api/suppliers/${supplierId}/products/${productId}`,
      {
        headers: authHeader(token),
      }
    );
    expect(res.ok()).toBeTruthy();
  });

  test('POST /suppliers/:id/products without auth returns 401', async ({ request }) => {
    if (!supplierId) return;
    const res = await request.post(`${API_BASE}/api/suppliers/${supplierId}/products`, {
      data: { product_id: 1 },
    });
    expect(res.status()).toBe(401);
  });
});
