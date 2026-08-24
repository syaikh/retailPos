import { test, expect } from './fixtures';
import { apiAs, ApiDriver } from './api-driver';

const data = (b: any) => (b && b.data !== undefined ? b.data : b);

test.describe('Product-Supplier Linking API', () => {
  let supplierId: number;
  let productId: number;

  test.beforeAll(async ({ request }) => {
    const api = await apiAs(request, 'superadmin');

    const supBody = data((await api.get('/api/suppliers?limit=1&is_active=true')).body);
    if (supBody && supBody.length > 0) {
      supplierId = supBody[0].id;
    }

    const prodBody = data((await api.get('/api/products?limit=1&status=active')).body);
    if (prodBody && prodBody.length > 0) {
      productId = prodBody[0].id;
    }
  });

  test('GET /products/:id/suppliers lists linked suppliers for a product', async ({ request }) => {
    if (!productId) return;
    const api = await apiAs(request, 'superadmin');
    const res = await api.get(`/api/products/${productId}/suppliers`);
    expect(res.ok).toBeTruthy();
    const body = data(res.body);
    expect(body).toBeDefined();
    expect(Array.isArray(body)).toBeTruthy();
  });

  test('POST /suppliers/:id/products links supplier to product', async ({ request }) => {
    if (!productId || !supplierId) return;
    const api = await apiAs(request, 'superadmin');
    await api.del(`/api/suppliers/${supplierId}/products/${productId}`);
    const res = await api.post(`/api/suppliers/${supplierId}/products`, {
      product_id: productId,
      supplier_sku: 'E2E-SKU-001',
      unit_cost: 8500,
      lead_time_days: 7,
      is_preferred: false,
    });
    expect(res.ok).toBeTruthy();
    const body = data(res.body);
    expect(body).toBeTruthy();
  });

  test('GET /suppliers/:id/products lists products for supplier', async ({ request }) => {
    if (!supplierId) return;
    const api = await apiAs(request, 'superadmin');
    const res = await api.get(`/api/suppliers/${supplierId}/products`);
    expect(res.ok).toBeTruthy();
    const body = data(res.body);
    expect(body).toBeDefined();
    expect(Array.isArray(body)).toBeTruthy();
  });

  test('PUT /suppliers/:id/products/:productId updates link', async ({ request }) => {
    if (!supplierId || !productId) return;
    const api = await apiAs(request, 'superadmin');
    const res = await api.put(`/api/suppliers/${supplierId}/products/${productId}`, {
      supplier_sku: 'E2E-SKU-001-UPDATED',
      unit_cost: 8200,
    });
    expect(res.ok).toBeTruthy();
  });

  test('DELETE /suppliers/:id/products/:productId unlinks supplier', async ({ request }) => {
    if (!supplierId || !productId) return;
    const api = await apiAs(request, 'superadmin');
    const res = await api.del(`/api/suppliers/${supplierId}/products/${productId}`);
    expect(res.ok).toBeTruthy();
  });

  test('POST /suppliers/:id/products without auth returns 401', async ({ request }) => {
    if (!supplierId) return;
    const res = await new ApiDriver(request, '').post(`/api/suppliers/${supplierId}/products`, {
      product_id: 1,
    });
    expect(res.status).toBe(401);
  });
});
