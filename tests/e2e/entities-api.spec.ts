import { test, expect } from './fixtures';
import { apiAs } from './api-driver';

/**
 * Entity CRUD behaviour, tested at the API layer. The corresponding browser
 * specs keep only genuine UI behaviour (dialog rendering, validation messages,
 * search/breadcrumb). Here we assert the *data* contract the UI reflects:
 * create persists, it is retrievable, update persists, delete removes. We
 * verify retrieval by search / GET-by-id rather than list-pagination, so the
 * assertion is stable no matter how large the table grows.
 */
test.describe('Entity CRUD behaviour (API driver)', () => {
  test.describe('Brands', () => {
    test('create → retrieve → update → delete lifecycle', async ({ request }) => {
      const api = await apiAs(request, 'superadmin');
      const name = `E2E Brand ${Date.now()}`;

      const create = await api.post('/api/brands', { name, description: 'api driver' });
      expect(create.status).toBe(201);
      const id = create.body.data.id;
      expect(create.body.data.name).toBe(name);

      const found = await api.get(`/api/brands?search=${encodeURIComponent(name)}&limit=100`);
      expect((found.body.data || []).map((b: any) => b.id)).toContain(id);

      const updated = `${name} updated`;
      const upd = await api.put(`/api/brands/${id}`, { name: updated });
      expect(upd.ok).toBeTruthy();
      expect(upd.body.data.name).toBe(updated);

      const found2 = await api.get(`/api/brands?search=${encodeURIComponent(updated)}&limit=100`);
      expect((found2.body.data || []).map((b: any) => b.id)).toContain(id);

      const del = await api.del(`/api/brands/${id}`);
      expect(del.ok).toBeTruthy();
      const found3 = await api.get(`/api/brands?search=${encodeURIComponent(updated)}&limit=100`);
      expect((found3.body.data || []).map((b: any) => b.id)).not.toContain(id);
    });

    test('create without name → 400', async ({ request }) => {
      const api = await apiAs(request, 'superadmin');
      expect((await api.post('/api/brands', {})).status).toBe(400);
    });

    test('cashier cannot create (no product.create) → 403', async ({ request }) => {
      const api = await apiAs(request, 'cashier');
      expect((await api.post('/api/brands', { name: 'x' })).status).toBe(403);
    });
  });

  test.describe('Units of Measure', () => {
    test('create → retrieve → update → delete lifecycle', async ({ request }) => {
      const api = await apiAs(request, 'superadmin');
      const code = `U${Date.now()}`.slice(0, 10).toUpperCase();
      const name = `E2E Unit ${Date.now()}`;

      const create = await api.post('/api/units-of-measure', { code, name, description: 'api driver' });
      expect(create.status).toBe(201);
      const id = create.body.data.id;
      expect(create.body.data.code).toBe(code);

      const found = await api.get(`/api/units-of-measure?search=${encodeURIComponent(code)}&limit=100`);
      expect((found.body.data || []).map((u: any) => u.id)).toContain(id);

      const updated = `${name} updated`;
      const upd = await api.put(`/api/units-of-measure/${id}`, { code, name: updated });
      expect(upd.ok).toBeTruthy();
      expect(upd.body.data.name).toBe(updated);

      const found2 = await api.get(`/api/units-of-measure?search=${encodeURIComponent(code)}&limit=100`);
      expect((found2.body.data || []).map((u: any) => u.id)).toContain(id);

      const del = await api.del(`/api/units-of-measure/${id}`);
      expect(del.ok).toBeTruthy();
      const found3 = await api.get(`/api/units-of-measure?search=${encodeURIComponent(code)}&limit=100`);
      expect((found3.body.data || []).map((u: any) => u.id)).not.toContain(id);
    });

    test('create without code/name → 400', async ({ request }) => {
      const api = await apiAs(request, 'superadmin');
      expect((await api.post('/api/units-of-measure', {})).status).toBe(400);
    });

    test('cashier cannot create (no product.create) → 403', async ({ request }) => {
      const api = await apiAs(request, 'cashier');
      expect((await api.post('/api/units-of-measure', { code: 'X', name: 'x' })).status).toBe(403);
    });
  });

  test.describe('Stores', () => {
    test('create → retrieve → update → delete lifecycle', async ({ request }) => {
      const api = await apiAs(request, 'superadmin');
      const name = `E2E Store ${Date.now()}`;

      const create = await api.post('/api/stores', { name, address: 'Jl. E2E', phone: '022-1' });
      expect(create.status).toBe(201);
      const id = create.body.data.id;
      expect(create.body.data.name).toBe(name);

      const got = await api.get(`/api/stores/${id}`);
      expect(got.ok).toBeTruthy();
      expect(got.body.data.id).toBe(id);

      const upd = await api.put(`/api/stores/${id}`, { is_active: false });
      expect(upd.ok).toBeTruthy();
      expect(upd.body.data.name).toBe(name);
      expect(upd.body.data.is_active).toBe(false);

      const del = await api.del(`/api/stores/${id}`);
      expect(del.ok).toBeTruthy();
      const gone = await api.get(`/api/stores/${id}`);
      expect(gone.status).toBe(404);
    });

    test('create without name → 400', async ({ request }) => {
      const api = await apiAs(request, 'superadmin');
      expect((await api.post('/api/stores', {})).status).toBe(400);
    });

    test('cashier cannot create (no store.create) → 403', async ({ request }) => {
      const api = await apiAs(request, 'cashier');
      expect((await api.post('/api/stores', { name: 'x' })).status).toBe(403);
    });
  });
});
