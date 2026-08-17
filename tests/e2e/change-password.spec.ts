import { test, expect } from './fixtures';
import { TEST_USERS, API_BASE, authHeader, getToken as cachedGetToken } from './fixtures';

const getToken = cachedGetToken;

test.describe('Change Password API', () => {

  test('POST /api/change-password succeeds with correct current password', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    const res = await request.post(`${API_BASE}/api/change-password`, {
      headers: authHeader(token),
      data: { current_password: TEST_USERS.superadmin.password, new_password: 'newpass123' },
    });
    expect(res.ok(), `expected 200, got ${res.status()}: ${await res.text()}`).toBeTruthy();

    // Revert to original password
    const revertRes = await request.post(`${API_BASE}/api/change-password`, {
      headers: authHeader(token),
      data: { current_password: 'newpass123', new_password: TEST_USERS.superadmin.password },
    });
    expect(revertRes.ok(), `revert failed: ${revertRes.status()}`).toBeTruthy();
  });

  test('POST /api/change-password fails with wrong current password', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    const res = await request.post(`${API_BASE}/api/change-password`, {
      headers: authHeader(token),
      data: { current_password: 'wrongpassword', new_password: 'newpass123' },
    });
    expect(res.status()).toBe(401);
    const body = await res.json();
    expect(body.error).toBeDefined();
  });

  test('POST /api/change-password rejects new password shorter than 6 chars', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    const res = await request.post(`${API_BASE}/api/change-password`, {
      headers: authHeader(token),
      data: { current_password: TEST_USERS.superadmin.password, new_password: 'abc' },
    });
    expect(res.status()).toBe(400);
  });

  test('POST /api/change-password rejects missing current_password', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    const res = await request.post(`${API_BASE}/api/change-password`, {
      headers: authHeader(token),
      data: { new_password: 'newpass123' },
    });
    expect(res.status()).toBe(400);
  });

  test('POST /api/change-password rejects missing new_password', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    const res = await request.post(`${API_BASE}/api/change-password`, {
      headers: authHeader(token),
      data: { current_password: TEST_USERS.superadmin.password },
    });
    expect(res.status()).toBe(400);
  });

  test('POST /api/change-password returns 401 without auth token', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/change-password`, {
      data: { current_password: 'test', new_password: 'newpass123' },
    });
    expect(res.status()).toBe(401);
  });

  test('POST /api/change-password returns 401 with invalid token', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/change-password`, {
      headers: authHeader('invalid-token'),
      data: { current_password: 'test', new_password: 'newpass123' },
    });
    expect(res.status()).toBe(401);
  });

  test('setting password to the same value is accepted or rejected gracefully', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    const res = await request.post(`${API_BASE}/api/change-password`, {
      headers: authHeader(token),
      data: { current_password: TEST_USERS.superadmin.password, new_password: TEST_USERS.superadmin.password },
    });
    // Server may accept (200) or reject (400/422) — both are valid behaviors
    expect([200, 400, 422]).toContain(res.status());
  });
});
