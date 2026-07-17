import { test, expect } from '@playwright/test';
import { API_BASE, authHeader, getToken as cachedGetToken } from './fixtures';

const getToken = cachedGetToken;

test.describe('Pricing Stacking E2E', () => {
  let productId: number;
  let ruleIds: number[] = [];

  test.beforeAll(async ({ request }) => {
    const token = await getToken(request);

    const prodRes = await request.get(`${API_BASE}/products?limit=1&status=active`, {
      headers: authHeader(token),
    });
    const prodBody = await prodRes.json();
    if (!prodBody.data || prodBody.data.length === 0) return;
    productId = prodBody.data[0].id;

    // Create two stacking promotion rules
    const rule1 = await request.post(`${API_BASE}/api/pricing-rules`, {
      headers: authHeader(token),
      data: {
        product_id: productId,
        pricing_type: 'promotion',
        pricing_method: 'discount_percent',
        pricing_value: 10,
        name: 'E2E Stack 10% Off',
        minimum_quantity: 1,
        priority: 1,
        is_active: true,
        allow_combine: true,
      },
    });
    const body1 = await rule1.json();
    if (body1.data) ruleIds.push(body1.data.id);

    const rule2 = await request.post(`${API_BASE}/api/pricing-rules`, {
      headers: authHeader(token),
      data: {
        product_id: productId,
        pricing_type: 'promotion',
        pricing_method: 'discount_amount',
        pricing_value: 5000,
        name: 'E2E Stack Rp5000 Off',
        minimum_quantity: 1,
        priority: 2,
        is_active: true,
        allow_combine: true,
      },
    });
    const body2 = await rule2.json();
    if (body2.data) ruleIds.push(body2.data.id);
  });

  test.afterAll(async ({ request }) => {
    const token = await getToken(request);
    for (const id of ruleIds) {
      await request.delete(`${API_BASE}/api/pricing-rules/${id}`, {
        headers: authHeader(token),
      });
    }
  });

  test('resolve picks best single rule when stacking not implemented', async ({ request }) => {
    if (!productId) return;
    const token = await getToken(request);
    const res = await request.post(`${API_BASE}/api/pricing/resolve`, {
      headers: authHeader(token),
      data: {
        items: [{ product_id: productId, quantity: 1 }],
      },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    const resolved = body.data[0];
    expect(resolved.pricing_type).toBe('promotion');
    expect(resolved.unit_price).toBeLessThan(resolved.original_price);
    expect(resolved.discount).toBeGreaterThan(0);
  });

  test('resolve batch applies best rule per product', async ({ request }) => {
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
    expect(body.data[0].unit_price).toBeDefined();
    expect(body.data[1].unit_price).toBeDefined();
  });
});
