import { test, expect } from './fixtures';
import { apiAs } from './api-driver';
import { API_BASE } from './fixtures';
import { TestDataTracker } from './db-helper';

/**
 * RBAC behaviour, tested at the API layer. The cheap, meaningful assertion is
 * *enforcement*: a role with a permission reaches the endpoint (2xx) and a role
 * without it is rejected (403). The browser sidebar (sidebar.spec.ts) still
 * proves those same grants render as visible/hidden nav. We deliberately avoid
 * asserting coarse permission *prefixes* — many roles legitimately hold
 * `sale.view` / `product.view` without holding the gate that opens a specific
 * page (e.g. POS), so prefix checks are both fragile and not the real contract.
 */
test.describe('RBAC behaviour (API driver)', () => {
  const tracker = new TestDataTracker();
  test.afterAll(() => tracker.cleanup());

  test('every role is issued a non-empty permission set', async ({ request }) => {
    for (const role of ['superadmin', 'admin', 'manager', 'cashier'] as const) {
      const api = await apiAs(request, role);
      expect(api.permissions().length, `${role} should have permissions`).toBeGreaterThan(0);
    }
  });

  test('superadmin (full privilege) can create brands and stores', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const b = await api.post('/api/brands', { name: `E2E Brand ${Date.now()}` });
    expect(b.status).toBe(201);
    tracker.trackBrand(b.body?.data?.id ?? b.body?.id);
    const s = await api.post('/api/stores', { name: `E2E Store ${Date.now()}` });
    expect(s.status).toBe(201);
    tracker.trackStore(s.body?.data?.id ?? s.body?.id);
  });

  test('admin can create stores (store.create)', async ({ request }) => {
    const api = await apiAs(request, 'admin');
    const r = await api.post('/api/stores', { name: `E2E Store ${Date.now()}` });
    expect(r.status).toBe(201);
    tracker.trackStore(r.body?.data?.id ?? r.body?.id);
  });

  test('manager is rejected from store creation (no store.create) but can create brands (has product.create)', async ({ request }) => {
    const api = await apiAs(request, 'manager');
    expect((await api.post('/api/stores', { name: 'x' })).status).toBe(403);
    const brandRes = await api.post('/api/brands', { name: `E2E Manager Brand ${Date.now()}` });
    expect(brandRes.status).toBe(201);
    tracker.trackBrand(brandRes.body?.data?.id ?? brandRes.body?.id);
  });

  test('cashier is rejected from store and brand creation', async ({ request }) => {
    const api = await apiAs(request, 'cashier');
    expect((await api.post('/api/stores', { name: 'x' })).status).toBe(403);
    expect((await api.post('/api/brands', { name: 'x' })).status).toBe(403);
  });

  test('unauthenticated request is rejected (401)', async ({ request }) => {
    const res = await request.get(`${API_BASE}/api/admin/roles`);
    expect(res.status()).toBe(401);
  });
});
