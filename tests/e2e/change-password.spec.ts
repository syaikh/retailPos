import { test, expect } from './fixtures';
import { TEST_USERS } from './fixtures';
import { apiAs, ApiDriver } from './api-driver';

const data = (b: any) => (b && b.data !== undefined ? b.data : b);

test.describe('Change Password API', () => {
  test('POST /api/change-password succeeds with correct current password', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');

    const res = await api.post('/api/change-password', {
      current_password: TEST_USERS.superadmin.password,
      new_password: 'newpass123',
    });
    expect(res.ok, `expected 200, got ${res.status}: ${JSON.stringify(res.body)}`).toBeTruthy();

    // Revert to original password
    const revertRes = await api.post('/api/change-password', {
      current_password: 'newpass123',
      new_password: TEST_USERS.superadmin.password,
    });
    expect(revertRes.ok, `revert failed: ${revertRes.status}`).toBeTruthy();
  });

  test('POST /api/change-password fails with wrong current password', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');

    const res = await api.post('/api/change-password', {
      current_password: 'wrongpassword',
      new_password: 'newpass123',
    });
    expect(res.status).toBe(401);
    expect(data(res.body).error ?? res.body?.error).toBeDefined();
  });

  test('POST /api/change-password rejects new password shorter than 6 chars', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');

    const res = await api.post('/api/change-password', {
      current_password: TEST_USERS.superadmin.password,
      new_password: 'abc',
    });
    expect(res.status).toBe(400);
  });

  test('POST /api/change-password rejects missing current_password', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');

    const res = await api.post('/api/change-password', { new_password: 'newpass123' });
    expect(res.status).toBe(400);
  });

  test('POST /api/change-password rejects missing new_password', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');

    const res = await api.post('/api/change-password', { current_password: TEST_USERS.superadmin.password });
    expect(res.status).toBe(400);
  });

  test('POST /api/change-password returns 401 without auth token', async ({ request }) => {
    const res = await new ApiDriver(request, '').post('/api/change-password', {
      current_password: 'test',
      new_password: 'newpass123',
    });
    expect(res.status).toBe(401);
  });

  test('POST /api/change-password returns 401 with invalid token', async ({ request }) => {
    const res = await new ApiDriver(request, 'invalid-token').post('/api/change-password', {
      current_password: 'test',
      new_password: 'newpass123',
    });
    expect(res.status).toBe(401);
  });

  test('setting password to the same value is accepted or rejected gracefully', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');

    const res = await api.post('/api/change-password', {
      current_password: TEST_USERS.superadmin.password,
      new_password: TEST_USERS.superadmin.password,
    });
    // Server may accept (200) or reject (400/422) — both are valid behaviors
    expect([200, 400, 422]).toContain(res.status);
  });
});
