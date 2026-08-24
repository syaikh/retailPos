import { test, expect } from './fixtures';
import { apiAs } from './api-driver';
import { API_BASE } from './fixtures';

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
  test('every role is issued a non-empty permission set', async ({ request }) => {
    for (const role of ['superadmin', 'admin', 'manager', 'cashier'] as const) {
      const api = await apiAs(request, role);
      expect(api.permissions().length, `${role} should have permissions`).toBeGreaterThan(0);
    }
  });

  test('superadmin (full privilege) can create brands and stores', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    expect((await api.post('/api/brands', { name: `E2E Brand ${Date.now()}` })).status).toBe(201);
    expect((await api.post('/api/stores', { name: `E2E Store ${Date.now()}` })).status).toBe(201);
  });

  test('admin can create stores (store.create)', async ({ request }) => {
    const api = await apiAs(request, 'admin');
    expect((await api.post('/api/stores', { name: `E2E Store ${Date.now()}` })).status).toBe(201);
  });

  test('manager is rejected from store and brand creation (no store/product create)', async ({ request }) => {
    const api = await apiAs(request, 'manager');
    expect((await api.post('/api/stores', { name: 'x' })).status).toBe(403);
    expect((await api.post('/api/brands', { name: 'x' })).status).toBe(403);
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
