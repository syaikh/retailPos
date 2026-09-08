import { test, expect } from './fixtures';
import { TEST_USERS, API_BASE, authHeader, getToken as cachedGetToken } from './fixtures';

const getToken = cachedGetToken;

test.describe('POS Wholesale Flow', () => {
  let productId: number;
  let ruleId: number;

  test.beforeAll(async ({ request }) => {
    const token = await getToken(request);

    const ts = Date.now();
    const sku = `E2E-WHOLE-${ts}`;
    const prodRes = await request.post(`${API_BASE}/api/products`, {
      headers: authHeader(token),
      data: {
        sku,
        name: `E2E Wholesale Test ${ts}`,
        price: 50000,
        cost: 30000,
        stock: 0,
        status: 'active',
      },
    });
    if (!prodRes.ok()) return;
    const prodBody = await prodRes.json();
    if (!prodBody.data) return;
    productId = prodBody.data.id;

    const ruleRes = await request.post(`${API_BASE}/api/pricing-rules`, {
      headers: authHeader(token),
      data: {
        product_id: productId,
        pricing_type: 'special_price',
        pricing_method: 'fixed_price',
        pricing_value: 10000,
        name: 'E2E Wholesale Test',
        minimum_quantity: 3,
        priority: 0,
        is_active: true,
      },
    });
    const ruleBody = await ruleRes.json();
    if (ruleBody.data) {
      ruleId = ruleBody.data.id;
    }
  });

  test.afterAll(async ({ request }) => {
    const token = await getToken(request);
    if (ruleId) {
      await request.delete(`${API_BASE}/api/pricing-rules/${ruleId}`, {
        headers: authHeader(token),
      });
    }
    if (productId) {
      await request.delete(`${API_BASE}/api/products/${productId}`, {
        headers: authHeader(token),
      });
    }
  });

  test('resolve returns base price for quantity below minimum', async ({ request }) => {
    if (!productId) return;
    const token = await getToken(request);
    const res = await request.post(`${API_BASE}/api/pricing/resolve`, {
      headers: authHeader(token),
      data: {
        items: [{ product_id: productId, quantity: 2 }],
      },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    const resolved = body.data[0];
    expect(resolved.pricing_type).toBe('normal');
    expect(resolved.unit_price).toBe(resolved.original_price);
  });

  test('resolve returns special_price for quantity meeting minimum', async ({ request }) => {
    if (!productId) return;
    const token = await getToken(request);
    const res = await request.post(`${API_BASE}/api/pricing/resolve`, {
      headers: authHeader(token),
      data: {
        items: [{ product_id: productId, quantity: 3 }],
      },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    const resolved = body.data[0];
    expect(resolved.pricing_type).toBe('special_price');
    expect(resolved.unit_price).toBe(10000);
    expect(resolved.original_price).toBeGreaterThanOrEqual(10000);
  });

  test('resolve batch handles multiple products', async ({ request }) => {
    if (!productId) return;
    const token = await getToken(request);
    const res = await request.post(`${API_BASE}/api/pricing/resolve`, {
      headers: authHeader(token),
      data: {
        items: [
          { product_id: productId, quantity: 1 },
          { product_id: productId, quantity: 5 },
        ],
      },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data.length).toBe(2);
    expect(body.data[0].pricing_type).toBe('normal');
    expect(body.data[1].pricing_type).toBe('special_price');
  });
});
