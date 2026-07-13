import { test, expect } from '@playwright/test';
import { TEST_USERS, API_BASE, authHeader, getToken as cachedGetToken } from './fixtures';

const getToken = cachedGetToken;

test.describe('Reports API - Chart Endpoints', () => {

  test('GET /api/dashboard/chart returns sales data', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const res = await request.get(`${API_BASE}/api/dashboard/chart?period=day`, { headers: authHeader(token) });
    expect(res.ok(), `chart failed: ${res.status()}: ${await res.text()}`).toBeTruthy();
    const body = await res.json();
    expect(body).toBeDefined();
  });

  test('GET /api/dashboard/chart/weekly returns weekly data', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const res = await request.get(`${API_BASE}/api/dashboard/chart/weekly`, { headers: authHeader(token) });
    expect(res.ok(), `weekly failed: ${res.status()}: ${await res.text()}`).toBeTruthy();
    const body = await res.json();
    expect(body).toBeDefined();
  });

  test('GET /api/dashboard/chart/monthly returns monthly data', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const res = await request.get(`${API_BASE}/api/dashboard/chart/monthly`, { headers: authHeader(token) });
    expect(res.ok(), `monthly failed: ${res.status()}: ${await res.text()}`).toBeTruthy();
    const body = await res.json();
    expect(body).toBeDefined();
  });

  test('GET /api/dashboard/comparison returns period comparison', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const res = await request.get(`${API_BASE}/api/dashboard/comparison`, { headers: authHeader(token) });
    expect(res.ok(), `comparison failed: ${res.status()}: ${await res.text()}`).toBeTruthy();
    const body = await res.json();
    expect(body).toBeDefined();
  });

  test('GET /api/dashboard/years returns available years', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const res = await request.get(`${API_BASE}/api/dashboard/years`, { headers: authHeader(token) });
    expect(res.ok(), `years failed: ${res.status()}: ${await res.text()}`).toBeTruthy();
    const body = await res.json();
    expect(body).toBeDefined();
  });

  test('GET /api/dashboard/chart without auth returns 401', async ({ request }) => {
    const res = await request.get(`${API_BASE}/api/dashboard/chart?period=day`);
    expect(res.status()).toBe(401);
  });

  test('GET /api/dashboard/chart with restricted role returns 403', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
    const res = await request.get(`${API_BASE}/api/dashboard/chart?period=day`, { headers: authHeader(token) });
    expect(res.status()).toBe(403);
  });
});

test.describe('Reports API - Dashboard Export', () => {

  test('POST /api/dashboard/export returns file', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const res = await request.post(`${API_BASE}/api/dashboard/export`, {
      headers: authHeader(token),
      data: { format: 'xlsx' },
    });
    expect(res.ok(), `export failed: ${res.status()}: ${await res.text()}`).toBeTruthy();
  });

  test('POST /api/dashboard/export without auth returns 401', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/dashboard/export`, {
      data: { format: 'xlsx' },
    });
    expect(res.status()).toBe(401);
  });
});
