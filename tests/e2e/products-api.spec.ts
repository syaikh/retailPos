import { test, expect } from './fixtures';
import { apiAs, ApiDriver } from './api-driver';

/**
 * Product listing/filter data contract driven at the API layer. This is the API
 * half of products.spec.ts — the 'Products Management' UI flows and the
 * 'Products - Supplier Features' scenario (which needs shared supplier setup)
 * stay in products.spec.ts.
 */
const data = (b: any) => (b && b.data !== undefined ? b.data : b);

test.describe('Products API - Category Filter', () => {
  test('GET /api/products?category=single returns filtered products', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const response = await api.get('/api/products?category=Personal+Care');
    expect(response.ok).toBeTruthy();
    const body = data(response.body);
    expect(body).toBeInstanceOf(Array);
    if (body.length > 0) {
      for (const product of body) {
        expect(product.category_name).toBe('Personal Care');
      }
    }
  });

  test('GET /api/products?category=multiple returns products matching any category', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const response = await api.get('/api/products?category=Gaming,Condiments');
    expect(response.ok).toBeTruthy();
    const body = data(response.body);
    expect(body).toBeInstanceOf(Array);
    if (body.length > 0) {
      const validCategories = ['Gaming', 'Condiments'];
      for (const product of body) {
        expect(validCategories).toContain(product.category_name);
      }
    }
  });
});

test.describe('Products API - Brand Filter', () => {
  test('GET /api/products?brand_id=single returns only products of that brand', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const brandsRes = await api.get('/api/brands');
    expect(brandsRes.ok).toBeTruthy();
    const brands = data(brandsRes.body) || [];
    test.skip(brands.length === 0, 'no brands seeded');
    const brandId = brands[0].id;

    const response = await api.get(`/api/products?brand_id=${brandId}`);
    expect(response.ok).toBeTruthy();
    const body = data(response.body);
    expect(body).toBeInstanceOf(Array);
    for (const product of body) {
      expect(product.brand_id).toBe(brandId);
    }
  });

  test('GET /api/products?brand_id=multiple returns products matching any brand', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const brandsRes = await api.get('/api/brands');
    expect(brandsRes.ok).toBeTruthy();
    const brands = (data(brandsRes.body) || []).slice(0, 2);
    test.skip(brands.length < 2, 'fewer than two brands seeded');
    const brandIds = brands.map((b: any) => b.id).join(',');

    const response = await api.get(`/api/products?brand_id=${brandIds}`);
    expect(response.ok).toBeTruthy();
    const body = data(response.body);
    expect(body).toBeInstanceOf(Array);
    for (const product of body) {
      expect(brands.map((b: any) => b.id)).toContain(product.brand_id);
    }
  });

  test('GET /api/products?brand_id=unknown returns empty list', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const response = await api.get('/api/products?brand_id=99999999');
    expect(response.ok).toBeTruthy();
    const body = data(response.body);
    expect(body).toBeInstanceOf(Array);
    expect(body.length).toBe(0);
  });
});

test.describe('Products API - Stock', () => {
  test('GET /api/products returns stock data', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const response = await api.get('/api/products?limit=200');
    expect(response.ok).toBeTruthy();
    const body = data(response.body);
    expect(body).toBeInstanceOf(Array);
    expect(body.length).toBeGreaterThan(0);
    for (const product of body) {
      expect(product.stock).toBeGreaterThanOrEqual(0);
    }
  });

  test('GET /api/stock-thresholds returns threshold config', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const response = await api.get('/api/stock-thresholds');
    expect(response.ok).toBeTruthy();
    const body = response.body;
    expect(body).toHaveProperty('warning');
    expect(body).toHaveProperty('critical');
  });
});
