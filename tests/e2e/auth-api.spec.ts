import { test, expect } from './fixtures';
import { TEST_USERS, API_BASE } from './fixtures';

async function login(request: any, username: string, password: string) {
  const res = await request.post(`${API_BASE}/api/login`, {
    data: { username, password },
  });
  if (!res.ok() && res.status() === 429) {
    await new Promise(r => setTimeout(r, 13000));
    const retry = await request.post(`${API_BASE}/api/login`, {
      data: { username, password },
    });
    expect(retry.ok(), `login failed after retry: ${retry.status()}`).toBeTruthy();
    const body = await retry.json();
    return { ...body, _headers: retry.headers() };
  }
  expect(res.ok(), `login failed: ${res.status()}`).toBeTruthy();
  const body = await res.json();
  return { ...body, _headers: res.headers() };
}

function decodeJWT(token: string) {
  const parts = token.split('.');
  return JSON.parse(atob(parts[1]));
}

test.describe('Auth API - Refresh Token', () => {

  test('POST /api/refresh returns new access token', async ({ request }) => {
    const loginResp = await login(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    expect(loginResp.access_token).toBeTruthy();

    const cookieHeader = loginResp._headers['set-cookie'];
    const refreshToken = cookieHeader?.match(/refresh_token=([^;]+)/)?.[1];
    expect(refreshToken, 'refresh token should be in set-cookie header').toBeTruthy();

    const res = await request.post(`${API_BASE}/api/refresh`, {
      headers: {
        'X-Refresh-Token': refreshToken,
        'X-Requested-With': 'XMLHttpRequest',
      },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.access_token).toBeTruthy();

    const decoded = decodeJWT(body.access_token);
    expect(decoded.username).toBe(TEST_USERS.superadmin.username);
    expect(decoded.role).toBe(TEST_USERS.superadmin.role);
  });

  test('POST /api/refresh with invalid token returns 401', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/refresh`, {
      headers: {
        'X-Refresh-Token': 'invalid-refresh-token',
        'X-Requested-With': 'XMLHttpRequest',
      },
    });
    expect(res.status()).toBe(401);
  });

  test('POST /api/refresh without token returns 400', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/refresh`, {
      headers: { 'X-Requested-With': 'XMLHttpRequest' },
    });
    expect(res.status()).toBe(400);
    const body = await res.json();
    expect(body.error).toContain('refresh token is required');
  });

  test('refresh token rotation: chained refreshes succeed', async ({ request }) => {
    const loginResp = await login(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const cookieHeader = loginResp._headers['set-cookie'];
    const refreshToken = cookieHeader?.match(/refresh_token=([^;]+)/)?.[1];
    expect(refreshToken).toBeTruthy();

    const res1 = await request.post(`${API_BASE}/api/refresh`, {
      headers: {
        'X-Refresh-Token': refreshToken,
        'X-Requested-With': 'XMLHttpRequest',
      },
    });
    expect(res1.ok()).toBeTruthy();

    const res1Cookie = res1.headers()['set-cookie'];
    const newRefreshToken = res1Cookie?.match(/refresh_token=([^;]+)/)?.[1];
    expect(newRefreshToken, 'rotation should return a new refresh token').toBeTruthy();
    expect(newRefreshToken).not.toBe(refreshToken);

    await new Promise(r => setTimeout(r, 1000));
    const res2 = await request.post(`${API_BASE}/api/refresh`, {
      headers: {
        'X-Refresh-Token': newRefreshToken,
        'X-Requested-With': 'XMLHttpRequest',
      },
    });
    expect(res2.ok()).toBeTruthy();
  });

});

test.describe('Auth API - Validate Session', () => {

  async function getToken(request: any) {
    const loginResp = await login(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    return loginResp.access_token;
  }

  test('POST /api/validate with valid token returns user info', async ({ request }) => {
    const token = await getToken(request);
    const res = await request.post(`${API_BASE}/api/validate`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.user.username).toBe(TEST_USERS.superadmin.username);
    expect(body.user.role).toBe(TEST_USERS.superadmin.role);
    expect(body.user.id).toBe(TEST_USERS.superadmin.id);
    expect(Array.isArray(body.permissions)).toBeTruthy();
  });

  test('POST /api/validate without token returns 401', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/validate`);
    expect(res.status()).toBe(401);
  });

  test('POST /api/validate with malformed token returns 401', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/validate`, {
      headers: { Authorization: 'Bearer this.is.not.a.valid.jwt' },
    });
    expect(res.status()).toBe(401);
  });

  test('validate for different roles', async ({ request }) => {
    const users = [TEST_USERS.admin, TEST_USERS.manager, TEST_USERS.cashier];
    for (let i = 0; i < users.length; i++) {
      const user = users[i];
      if (i > 0) await new Promise(r => setTimeout(r, 13000));
      const loginResp = await login(request, user.username, user.password);
      const res = await request.post(`${API_BASE}/api/validate`, {
        headers: { Authorization: `Bearer ${loginResp.access_token}` },
      });
      expect(res.ok(), `validation failed for ${user.username}`).toBeTruthy();
      const body = await res.json();
      expect(body.user.username).toBe(user.username);
      expect(body.user.role).toBe(user.role);
      expect(body.user.id).toBe(user.id);
    }
  });

});

test.describe('Auth API - Logout', () => {

  test('POST /api/logout returns success', async ({ request }) => {
    const loginResp = await login(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const cookieHeader = loginResp._headers['set-cookie'];
    const refreshToken = cookieHeader?.match(/refresh_token=([^;]+)/)?.[1];

    const res = await request.post(`${API_BASE}/api/logout`, {
      headers: {
        Authorization: `Bearer ${loginResp.access_token}`,
        ...(refreshToken ? { Cookie: `refresh_token=${refreshToken}` } : {}),
      },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.status).toBe('logged out');
  });

  test('POST /api/logout without auth returns 401', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/logout`);
    expect(res.status()).toBe(401);
  });

  test('after logout, refresh token is invalid', async ({ request }) => {
    await new Promise(r => setTimeout(r, 13000));
    const loginResp = await login(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const cookieHeader = loginResp._headers['set-cookie'];
    const refreshToken = cookieHeader?.match(/refresh_token=([^;]+)/)?.[1];
    expect(refreshToken).toBeTruthy();

    const logoutRes = await request.post(`${API_BASE}/api/logout`, {
      headers: {
        Authorization: `Bearer ${loginResp.access_token}`,
        Cookie: `refresh_token=${refreshToken}`,
      },
    });
    expect(logoutRes.ok()).toBeTruthy();

    const refreshRes = await request.post(`${API_BASE}/api/refresh`, {
      headers: {
        'X-Refresh-Token': refreshToken,
        'X-Requested-With': 'XMLHttpRequest',
      },
    });
    expect(refreshRes.status()).toBe(401);
  });

});
