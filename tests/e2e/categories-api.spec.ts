import { test, expect } from './fixtures';
import { apiAs } from './api-driver';

/**
 * Category CRUD behaviour, tested at the API layer. The browser spec
 * (categories.spec.ts) keeps only genuine UI behaviour: dialog rendering,
 * search/empty state, import/export. Here we assert the data contract the UI
 * reflects: create persists, it is retrievable, update persists, delete
 * removes. Retrieval goes through /categories/manage?search= so the assertion
 * stays stable regardless of table size.
 */
test.describe('Categories CRUD behaviour (API driver)', () => {
  test('superadmin: create → retrieve → update → delete lifecycle', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const name = `E2E Category ${Date.now()}`;

    const create = await api.post('/api/categories', { name, description: 'api driver' });
    expect(create.status).toBe(201);
    const id = create.body.data.id;
    expect(create.body.data.name).toBe(name);

    const found = await api.get(`/api/categories/manage?search=${encodeURIComponent(name)}&limit=100`);
    expect((found.body.data || []).map((c: any) => c.id)).toContain(id);

    const updated = `${name} updated`;
    const upd = await api.put(`/api/categories/${id}`, { name: updated, description: 'api driver' });
    expect(upd.ok).toBeTruthy();
    expect(upd.body.data.name).toBe(updated);

    const found2 = await api.get(`/api/categories/manage?search=${encodeURIComponent(updated)}&limit=100`);
    expect((found2.body.data || []).map((c: any) => c.id)).toContain(id);

    const del = await api.del(`/api/categories/${id}`);
    expect(del.ok).toBeTruthy();
    const found3 = await api.get(`/api/categories/manage?search=${encodeURIComponent(updated)}&limit=100`);
    expect((found3.body.data || []).map((c: any) => c.id)).not.toContain(id);
  });

  test('admin can create a category (product.create) → 201', async ({ request }) => {
    const api = await apiAs(request, 'admin');
    const name = `E2E Category Admin ${Date.now()}`;
    const res = await api.post('/api/categories', { name });
    expect(res.status).toBe(201);
    expect(res.body.data.name).toBe(name);
    await api.del(`/api/categories/${res.body.data.id}`);
  });

  test('create without name → 400', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    expect((await api.post('/api/categories', {})).status).toBe(400);
  });

  test('cashier cannot create (no product.create) → 403', async ({ request }) => {
    const api = await apiAs(request, 'cashier');
    expect((await api.post('/api/categories', { name: 'x' })).status).toBe(403);
  });
});
