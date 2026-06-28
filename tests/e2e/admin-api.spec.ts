import { test, expect } from '@playwright/test';
import { TEST_USERS, API_BASE, authHeader } from './fixtures';

async function getToken(request: any) {
  const res = await request.post(`${API_BASE}/api/login`, {
    data: { username: TEST_USERS.superadmin.username, password: TEST_USERS.superadmin.password },
  });
  const body = await res.json();
  return body.access_token;
}

test.describe('Admin API - Delete User', () => {

  async function createTempUser(request: any, token: string) {
    const suffix = `${Date.now()}`;
    const res = await request.post(`${API_BASE}/api/admin/users`, {
      headers: authHeader(token),
      data: {
        username: `deleteme_${suffix}`,
        password: 'test123',
        role_id: 4,
        is_active: true,
      },
    });
    if (!res.ok()) return null;
    const body = await res.json();
    return body.data || body.user || body;
  }

  test('DELETE /api/admin/users/:id deletes a user', async ({ request }) => {
    const token = await getToken(request);
    const user = await createTempUser(request, token);
    if (!user) return;

    const res = await request.delete(`${API_BASE}/api/admin/users/${user.id}`, {
      headers: authHeader(token),
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.status).toBe('deleted');
  });

  test('DELETE /api/admin/users/:id with invalid id returns 400', async ({ request }) => {
    const token = await getToken(request);
    const res = await request.delete(`${API_BASE}/api/admin/users/abc`, {
      headers: authHeader(token),
    });
    expect(res.status()).toBe(400);
  });

  test('DELETE /api/admin/users/:id without auth returns 401', async ({ request }) => {
    const res = await request.delete(`${API_BASE}/api/admin/users/1`);
    expect(res.status()).toBe(401);
  });

  test('DELETE /api/admin/users/:id with restricted role returns 403', async ({ request }) => {
    const loginRes = await request.post(`${API_BASE}/api/login`, {
      data: { username: TEST_USERS.cashier.username, password: TEST_USERS.cashier.password },
    });
    const loginBody = await loginRes.json();
    const res = await request.delete(`${API_BASE}/api/admin/users/1`, {
      headers: authHeader(loginBody.access_token),
    });
    expect(res.status()).toBe(403);
  });

});
