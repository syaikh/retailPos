import { test, expect } from './fixtures';
import { TEST_USERS, API_BASE, authHeader, getToken as cachedGetToken } from './fixtures';

const getToken = cachedGetToken;

test.describe('Pricing Rules API', () => {
  let createdProductId: number;
  let createdRuleId: number;

  test.beforeAll(async ({ request }) => {
    const token = await getToken(request);
    const res = await request.get(`${API_BASE}/api/products?limit=1&status=active`, {
      headers: authHeader(token),
    });
    const body = await res.json();
    if (body.data && body.data.length > 0) {
      createdProductId = body.data[0].id;
    }
  });

  test('POST /api/pricing-rules creates a promotion rule', async ({ request }) => {
    if (!createdProductId) return;
    const token = await getToken(request);
    const res = await request.post(`${API_BASE}/api/pricing-rules`, {
      headers: authHeader(token),
      data: {
        product_id: createdProductId,
        pricing_type: 'promotion',
        pricing_method: 'fixed_price',
        pricing_value: 10000,
        name: 'E2E Promotion Rule',
        minimum_quantity: 1,
        priority: 0,
        is_active: true,
      },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data).toBeTruthy();
    expect(body.data.id).toBeTruthy();
    expect(body.data.pricing_type).toBe('promotion');
    expect(body.data.pricing_value).toBe(10000);
    createdRuleId = body.data.id;
  });

  test('GET /api/pricing-rules lists rules', async ({ request }) => {
    const token = await getToken(request);
    const res = await request.get(`${API_BASE}/api/pricing-rules?limit=10`, {
      headers: authHeader(token),
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data).toBeDefined();
    expect(Array.isArray(body.data)).toBeTruthy();
  });

  test('GET /api/pricing-rules filters by product_id', async ({ request }) => {
    if (!createdProductId) return;
    const token = await getToken(request);
    const res = await request.get(`${API_BASE}/api/pricing-rules?product_id=${createdProductId}`, {
      headers: authHeader(token),
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data).toBeDefined();
    if (body.data.length > 0) {
      expect(body.data[0].product_id).toBe(createdProductId);
    }
  });

  test('PUT /api/pricing-rules/:id updates a rule', async ({ request }) => {
    if (!createdRuleId) return;
    const token = await getToken(request);
    const res = await request.put(`${API_BASE}/api/pricing-rules/${createdRuleId}`, {
      headers: authHeader(token),
      data: {
        product_id: createdProductId,
        pricing_type: 'promotion',
        pricing_method: 'fixed_price',
        pricing_value: 9000,
        name: 'E2E Updated Rule',
        minimum_quantity: 2,
        priority: 1,
        is_active: true,
      },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data.name).toBe('E2E Updated Rule');
    expect(body.data.pricing_value).toBe(9000);
  });

  test('POST /api/pricing/resolve resolves prices', async ({ request }) => {
    if (!createdProductId) return;
    const token = await getToken(request);
    const res = await request.post(`${API_BASE}/api/pricing/resolve`, {
      headers: authHeader(token),
      data: {
        items: [{ product_id: createdProductId, quantity: 1 }],
      },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data).toBeDefined();
    expect(Array.isArray(body.data)).toBeTruthy();
    expect(body.data.length).toBe(1);
    expect(body.data[0].unit_price).toBeDefined();
    expect(body.data[0].original_price).toBeDefined();
  });

  test('DELETE /api/pricing-rules/:id deletes a rule', async ({ request }) => {
    if (!createdRuleId) return;
    const token = await getToken(request);
    const res = await request.delete(`${API_BASE}/api/pricing-rules/${createdRuleId}`, {
      headers: authHeader(token),
    });
    expect(res.ok()).toBeTruthy();
  });

  test('POST /api/pricing-rules without auth returns 401', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/pricing-rules`, {
      data: { product_id: 1, pricing_type: 'promotion', pricing_method: 'fixed_price', pricing_value: 10000, name: 'Test', minimum_quantity: 1, priority: 0 },
    });
    expect(res.status()).toBe(401);
  });
});
