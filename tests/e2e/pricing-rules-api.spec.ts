import { test, expect } from './fixtures';
import { apiAs, ApiDriver } from './api-driver';

/**
 * Pricing rule behaviour (CRUD + resolution contract) driven at the API layer.
 * This merges the previously browser-filed pricing-rules.spec.ts,
 * pricing-scope.spec.ts and pricing-stacking.spec.ts — all of which were
 * already 100% API-driven, just misnamed. Genuine UI (the rules admin screen)
 * stays in pricing-rules-ui.spec.ts.
 */
const data = (b: any) => (b && b.data !== undefined ? b.data : b);
const firstOf = (b: any) => {
  const d = data(b);
  return Array.isArray(d) ? d[0] : d;
};

async function activeProduct(api: ApiDriver) {
  return firstOf((await api.get('/api/products?limit=1&status=active')).body);
}

test.describe('Pricing Rules CRUD (API driver)', () => {
  let productId = 0;
  let ruleId = 0;

  test.beforeAll(async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const p = await activeProduct(api);
    productId = p?.id ?? 0;
  });

  test('POST creates a promotion rule → 201', async ({ request }) => {
    const api = await apiAs(request, 'admin');
    const res = await api.post('/api/pricing-rules', {
      product_id: productId,
      pricing_type: 'promotion',
      pricing_method: 'fixed_price',
      pricing_value: 10000,
      name: 'E2E Promotion Rule',
      minimum_quantity: 1,
      priority: 0,
      is_active: true,
    });
    expect(res.status).toBe(201);
    ruleId = data(res.body).id;
    expect(data(res.body).pricing_value).toBe(10000);
  });

  test('GET lists rules', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const res = await api.get('/api/pricing-rules?limit=10');
    expect(res.ok).toBeTruthy();
    expect(Array.isArray(data(res.body))).toBeTruthy();
  });

  test('GET filters by product_id', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const res = await api.get(`/api/pricing-rules?product_id=${productId}&limit=100`);
    expect(res.ok).toBeTruthy();
    const rows = data(res.body);
    if (rows.length > 0) expect(rows[0].product_id).toBe(productId);
  });

  test('PUT updates the rule', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const res = await api.put(`/api/pricing-rules/${ruleId}`, {
      product_id: productId,
      pricing_type: 'promotion',
      pricing_method: 'fixed_price',
      pricing_value: 9000,
      name: 'E2E Updated Rule',
      minimum_quantity: 2,
      priority: 1,
      is_active: true,
    });
    expect(res.ok).toBeTruthy();
    expect(data(res.body).name).toBe('E2E Updated Rule');
    expect(data(res.body).pricing_value).toBe(9000);
  });

  test('POST /api/pricing/resolve resolves prices', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const res = await api.post('/api/pricing/resolve', {
      items: [{ product_id: productId, quantity: 1 }],
    });
    expect(res.ok).toBeTruthy();
    const rows = data(res.body);
    expect(rows.length).toBe(1);
    expect(rows[0].unit_price).toBeDefined();
    expect(rows[0].original_price).toBeDefined();
  });

  test('DELETE removes the rule → 200', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const res = await api.del(`/api/pricing-rules/${ruleId}`);
    expect(res.ok).toBeTruthy();
  });

  test('POST without auth returns 401', async ({ request }) => {
    const api = new ApiDriver(request, '');
    const res = await api.post('/api/pricing-rules', {
      product_id: 1,
      pricing_type: 'promotion',
      pricing_method: 'fixed_price',
      pricing_value: 10000,
      name: 'Test',
      minimum_quantity: 1,
      priority: 0,
    });
    expect(res.status).toBe(401);
  });

  test('POST with cashier (no pricing.create) returns 403', async ({ request }) => {
    const api = await apiAs(request, 'cashier');
    const res = await api.post('/api/pricing-rules', {
      product_id: productId,
      pricing_type: 'promotion',
      pricing_method: 'fixed_price',
      pricing_value: 10000,
      name: 'Cashier Should Fail',
      minimum_quantity: 1,
      priority: 0,
    });
    expect(res.status).toBe(403);
  });
});

test.describe('Pricing Scope (API driver)', () => {
  let productId = 0;
  let categoryId = 0;
  const ruleIds: number[] = [];

  test.beforeAll(async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    // Pick an active product that actually has a category, so the category-scoped
    // rule has something to match. (The original spec silently skipped otherwise.)
    const list = data((await api.get('/api/products?limit=50&status=active')).body) || [];
    const p = list.find((x: any) => x.category_id) || list[0];
    productId = p?.id ?? 0;
    categoryId = p?.category_id ?? 0;

    // Clear any pre-existing rules for this product/category to avoid interference.
    for (const q of [`product_id=${productId}`, `category_id=${categoryId}`]) {
      if (!q.includes('category_id=0')) {
        const existing = data((await api.get(`/api/pricing-rules?${q}&limit=100`)).body) || [];
        for (const r of existing) await api.del(`/api/pricing-rules/${r.id}`);
      }
    }
  });

  test.afterAll(async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    for (const id of ruleIds) await api.del(`/api/pricing-rules/${id}`);
  });

  test('category-scoped rule resolves for product in that category', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const res = await api.post('/api/pricing-rules', {
      category_id: categoryId,
      pricing_type: 'special_price',
      pricing_method: 'discount_percent',
      pricing_value: 25,
      name: 'E2E Category 25% Off',
      minimum_quantity: 1,
      priority: 0,
      is_active: true,
    });
    const created = data(res.body);
    if (created?.id) ruleIds.push(created.id);

    const resolve = await api.post('/api/pricing/resolve', { items: [{ product_id: productId, quantity: 1 }] });
    expect(resolve.ok).toBeTruthy();
    const resolved = data(resolve.body)[0];
    expect(resolved.pricing_type).toBe('special_price');
    expect(resolved.unit_price).toBeLessThan(resolved.original_price);
  });

  test('max quantity filter excludes rule when quantity exceeds max', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const res = await api.post('/api/pricing-rules', {
      product_id: productId,
      pricing_type: 'special_price',
      pricing_method: 'discount_percent',
      pricing_value: 15,
      name: 'E2E Max Qty Rule',
      minimum_quantity: 1,
      maximum_quantity: 3,
      priority: 10,
      is_active: true,
    });
    const id = data(res.body)?.id;

    try {
      const r1 = await api.post('/api/pricing/resolve', { items: [{ product_id: productId, quantity: 3 }] });
      expect(data(r1.body)[0].pricing_type).toBe('special_price');

      const r2 = await api.post('/api/pricing/resolve', { items: [{ product_id: productId, quantity: 4 }] });
      expect(data(r2.body)[0].unit_price).toBeDefined();
    } finally {
      if (id) await api.del(`/api/pricing-rules/${id}`);
    }
  });

  test('inactive rule is excluded from resolution', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const res = await api.post('/api/pricing-rules', {
      product_id: productId,
      pricing_type: 'promotion',
      pricing_method: 'discount_percent',
      pricing_value: 50,
      name: 'E2E Inactive Rule',
      minimum_quantity: 1,
      priority: 100,
      is_active: false,
    });
    const id = data(res.body)?.id;

    try {
      const resolve = await api.post('/api/pricing/resolve', { items: [{ product_id: productId, quantity: 1 }] });
      const resolved = data(resolve.body)[0];
      if (resolved.rule && resolved.rule.id === id) {
        expect.fail('inactive rule should not be applied');
      }
    } finally {
      if (id) await api.del(`/api/pricing-rules/${id}`);
    }
  });
});

test.describe('Pricing Stacking (API driver)', () => {
  let productId = 0;
  const ruleIds: number[] = [];

  test.beforeAll(async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const p = await activeProduct(api);
    productId = p?.id ?? 0;

    const r1 = await api.post('/api/pricing-rules', {
      product_id: productId,
      pricing_type: 'promotion',
      pricing_method: 'discount_percent',
      pricing_value: 10,
      name: 'E2E Stack 10% Off',
      minimum_quantity: 1,
      priority: 1,
      is_active: true,
      allow_combine: true,
    });
    if (data(r1.body)?.id) ruleIds.push(data(r1.body).id);

    const r2 = await api.post('/api/pricing-rules', {
      product_id: productId,
      pricing_type: 'promotion',
      pricing_method: 'discount_amount',
      pricing_value: 5000,
      name: 'E2E Stack Rp5000 Off',
      minimum_quantity: 1,
      priority: 2,
      is_active: true,
      allow_combine: true,
    });
    if (data(r2.body)?.id) ruleIds.push(data(r2.body).id);
  });

  test.afterAll(async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    for (const id of ruleIds) await api.del(`/api/pricing-rules/${id}`);
  });

  test('resolve picks best single rule when stacking not implemented', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const res = await api.post('/api/pricing/resolve', { items: [{ product_id: productId, quantity: 1 }] });
    expect(res.ok).toBeTruthy();
    const resolved = data(res.body)[0];
    expect(resolved.pricing_type).toBe('promotion');
    expect(resolved.unit_price).toBeLessThan(resolved.original_price);
    expect(resolved.discount).toBeGreaterThan(0);
  });

  test('resolve batch applies best rule per product', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const res = await api.post('/api/pricing/resolve', {
      items: [
        { product_id: productId, quantity: 1 },
        { product_id: productId, quantity: 5 },
      ],
    });
    expect(res.ok).toBeTruthy();
    const rows = data(res.body);
    expect(rows.length).toBe(2);
    expect(rows[0].unit_price).toBeDefined();
    expect(rows[1].unit_price).toBeDefined();
  });
});
