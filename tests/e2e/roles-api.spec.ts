import { test, expect } from './fixtures';
import { apiAs, ApiDriver } from './api-driver';

/**
 * Role management behaviour (update / permissions / delete) driven at the API
 * layer. This is the API half of roles.spec.ts — the browser half (create-role
 * modal, permission-group toggling, unsaved-changes guard) stays in roles.spec.ts
 * as genuine UI coverage. RBAC for role *create/list* lives in rbac-api.spec.ts.
 */
const data = (b: any) => (b && b.data !== undefined ? b.data : b);

async function createRole(api: ApiDriver, suffix: string) {
  const res = await api.post('/api/admin/roles', {
    name: `e2e_role_${suffix}`,
    description: 'api driver',
  });
  expect(res.ok, `create failed: ${res.status}`).toBeTruthy();
  return data(res.body).id as number;
}

test.describe('Roles API - Update Role', () => {
  test('PUT updates role name', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const id = await createRole(api, `upd_${Date.now()}`);
    const newName = `updated_${Date.now()}`;
    const res = await api.put(`/api/admin/roles/${id}`, { name: newName });
    expect(res.ok, `update failed: ${res.status}: ${JSON.stringify(res.body)}`).toBeTruthy();
    expect(data(res.body).name).toBe(newName);
  });

  test('PUT updates description', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const id = await createRole(api, `desc_${Date.now()}`);
    const res = await api.put(`/api/admin/roles/${id}`, { description: 'new description' });
    expect(res.ok).toBeTruthy();
    expect(data(res.body).description).toBe('new description');
  });

  test('PUT returns 404 for nonexistent role', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const res = await api.put('/api/admin/roles/999999', { name: 'nope' });
    expect(res.status).toBe(404);
  });

  test('PUT with cashier (no role.update) returns 403', async ({ request }) => {
    const api = await apiAs(request, 'cashier');
    const res = await api.put('/api/admin/roles/1', { name: 'hack' });
    expect(res.status).toBe(403);
  });
});

test.describe('Roles API - Update Permissions', () => {
  test('PUT /permissions updates permission set', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const permRes = await api.get('/api/admin/permissions');
    expect(permRes.ok).toBeTruthy();
    const perms = data(permRes.body) || [];
    expect(perms.length).toBeGreaterThan(0);
    const permIds = perms.slice(0, 3).map((p: any) => p.id);

    const id = await createRole(api, `perm_${Date.now()}`);
    const res = await api.put(`/api/admin/roles/${id}/permissions`, { permission_ids: permIds });
    expect(res.ok, `perm update failed: ${res.status}: ${JSON.stringify(res.body)}`).toBeTruthy();
    expect(data(res.body).permissions).toBeDefined();
  });

  test('PUT /permissions with empty array is accepted', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const id = await createRole(api, `emptyperm_${Date.now()}`);
    const res = await api.put(`/api/admin/roles/${id}/permissions`, { permission_ids: [] });
    expect(res.ok).toBeTruthy();
  });
});

test.describe('Roles API - Delete Role', () => {
  test('DELETE removes a role with no users', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const id = await createRole(api, `del_${Date.now()}`);
    const del = await api.del(`/api/admin/roles/${id}`);
    expect(del.ok, `delete failed: ${del.status}: ${JSON.stringify(del.body)}`).toBeTruthy();
    expect(data(del.body).status).toBe('deleted');

    const list = data((await api.get('/api/admin/roles?limit=100')).body) || [];
    expect(list.find((r: any) => r.id === id)).toBeUndefined();
  });

  test('DELETE returns 400 for role with assigned users (admin, id=2)', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const res = await api.del('/api/admin/roles/2');
    expect(res.status).toBe(400);
    expect(String(data(res.body).error ?? res.body?.error ?? '')).toContain('users are assigned');
  });

  test('DELETE with cashier (no role.delete) returns 403', async ({ request }) => {
    const api = await apiAs(request, 'cashier');
    const res = await api.del('/api/admin/roles/1');
    expect(res.status).toBe(403);
  });
});
