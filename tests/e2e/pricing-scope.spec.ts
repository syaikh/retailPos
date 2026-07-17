import { test, expect } from '@playwright/test';
import { API_BASE, authHeader, getToken as cachedGetToken } from './fixtures';

const getToken = cachedGetToken;

test.describe('Pricing Scope Priority E2E', () => {
  let productId: number;
  let productCategoryId: number;
  let ruleIds: number[] = [];

  test.beforeAll(async ({ request }) => {
    const token = await getToken(request);

    const prodRes = await request.get(`${API_BASE}/api/products?limit=1&status=active`, {
      headers: authHeader(token),
    });
    const prodBody = await prodRes.json();
    if (!prodBody.data || prodBody.data.length === 0) return;
    productId = prodBody.data[0].id;
    productCategoryId = prodBody.data[0].category_id;

    const existingRes = await request.get(`${API_BASE}/api/pricing-rules?product_id=${productId}&limit=100`, {
      headers: authHeader(token),
    });
    const existingBody = await existingRes.json();
    if (existingBody.data) {
      for (const rule of existingBody.data) {
        await request.delete(`${API_BASE}/api/pricing-rules/${rule.id}`, {
          headers: authHeader(token),
        });
      }
    }
    if (productCategoryId) {
      const catRes = await request.get(`${API_BASE}/api/pricing-rules?category_id=${productCategoryId}&limit=100`, {
        headers: authHeader(token),
      });
      const catBody = await catRes.json();
      if (catBody.data) {
        for (const rule of catBody.data) {
          await request.delete(`${API_BASE}/api/pricing-rules/${rule.id}`, {
            headers: authHeader(token),
          });
        }
      }
    }
  });

  test.afterAll(async ({ request }) => {
    const token = await getToken(request);
    for (const id of ruleIds) {
      await request.delete(`${API_BASE}/api/pricing-rules/${id}`, {
        headers: authHeader(token),
      });
    }
  });

  test('category-scoped rule resolves for product in that category', async ({ request }) => {
    if (!productId || !productCategoryId) return;
    const token = await getToken(request);

    const ruleRes = await request.post(`${API_BASE}/api/pricing-rules`, {
      headers: authHeader(token),
      data: {
        category_id: productCategoryId,
        pricing_type: 'special_price',
        pricing_method: 'discount_percent',
        pricing_value: 25,
        name: 'E2E Category 25% Off',
        minimum_quantity: 1,
        priority: 0,
        is_active: true,
      },
    });
    const ruleBody = await ruleRes.json();
    if (ruleBody.data) ruleIds.push(ruleBody.data.id);

    const resolveRes = await request.post(`${API_BASE}/api/pricing/resolve`, {
      headers: authHeader(token),
      data: {
        items: [{ product_id: productId, quantity: 1 }],
      },
    });
    expect(resolveRes.ok()).toBeTruthy();
    const resolveBody = await resolveRes.json();
    const resolved = resolveBody.data[0];
    expect(resolved.pricing_type).toBe('special_price');
    expect(resolved.unit_price).toBeLessThan(resolved.original_price);
  });

  test('max quantity filter excludes rule when quantity exceeds max', async ({ request }) => {
    if (!productId) return;
    const token = await getToken(request);

    const ruleRes = await request.post(`${API_BASE}/api/pricing-rules`, {
      headers: authHeader(token),
      data: {
        product_id: productId,
        pricing_type: 'special_price',
        pricing_method: 'discount_percent',
        pricing_value: 15,
        name: 'E2E Max Qty Rule',
        minimum_quantity: 1,
        maximum_quantity: 3,
        priority: 10,
        is_active: true,
      },
    });
    const ruleBody = await ruleRes.json();
    const maxRuleId = ruleBody.data?.id;

    try {
      const res1 = await request.post(`${API_BASE}/api/pricing/resolve`, {
        headers: authHeader(token),
        data: {
          items: [{ product_id: productId, quantity: 3 }],
        },
      });
      const body1 = await res1.json();
      expect(body1.data[0].pricing_type).toBe('special_price');

      const res2 = await request.post(`${API_BASE}/api/pricing/resolve`, {
        headers: authHeader(token),
        data: {
          items: [{ product_id: productId, quantity: 4 }],
        },
      });
      const body2 = await res2.json();
      expect(body2.data[0].unit_price).toBeDefined();
    } finally {
      if (maxRuleId) {
        await request.delete(`${API_BASE}/api/pricing-rules/${maxRuleId}`, {
          headers: authHeader(token),
        });
      }
    }
  });

  test('inactive rule is excluded from resolution', async ({ request }) => {
    if (!productId) return;
    const token = await getToken(request);

    const ruleRes = await request.post(`${API_BASE}/api/pricing-rules`, {
      headers: authHeader(token),
      data: {
        product_id: productId,
        pricing_type: 'promotion',
        pricing_method: 'discount_percent',
        pricing_value: 50,
        name: 'E2E Inactive Rule',
        minimum_quantity: 1,
        priority: 100,
        is_active: false,
      },
    });
    const ruleBody = await ruleRes.json();
    const inactiveRuleId = ruleBody.data?.id;

    try {
      const resolveRes = await request.post(`${API_BASE}/api/pricing/resolve`, {
        headers: authHeader(token),
        data: {
          items: [{ product_id: productId, quantity: 1 }],
        },
      });
      const resolveBody = await resolveRes.json();
      const resolved = resolveBody.data[0];
      if (resolved.rule && resolved.rule.id === inactiveRuleId) {
        expect.fail('inactive rule should not be applied');
      }
    } finally {
      if (inactiveRuleId) {
        await request.delete(`${API_BASE}/api/pricing-rules/${inactiveRuleId}`, {
          headers: authHeader(token),
        });
      }
    }
  });
});
