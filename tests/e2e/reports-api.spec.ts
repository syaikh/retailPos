import { test, expect } from '@playwright/test';
import { TEST_USERS, API_BASE, authHeader, getToken as cachedGetToken } from './fixtures';

const getToken = cachedGetToken;

const JAKARTA_OFFSET_MS = 7 * 60 * 60 * 1000;

function getTodayInJakarta(): string {
  const shifted = new Date(Date.now() + JAKARTA_OFFSET_MS);
  return `${shifted.getUTCFullYear()}-${String(shifted.getUTCMonth() + 1).padStart(2, '0')}-${String(shifted.getUTCDate()).padStart(2, '0')}`;
}

function getDateNDaysAgoInJakarta(daysAgo: number): string {
  const shifted = new Date(Date.now() + JAKARTA_OFFSET_MS);
  const todayMidnightJKT =
    Date.UTC(shifted.getUTCFullYear(), shifted.getUTCMonth(), shifted.getUTCDate(), 0, 0, 0, 0) -
    JAKARTA_OFFSET_MS;
  const targetMs = todayMidnightJKT - daysAgo * 86400000;
  const target = new Date(targetMs + JAKARTA_OFFSET_MS);
  return `${target.getUTCFullYear()}-${String(target.getUTCMonth() + 1).padStart(2, '0')}-${String(target.getUTCDate()).padStart(2, '0')}`;
}

test.describe('Reports API - Chart Endpoints', () => {

  test('GET /api/dashboard/chart returns sales data', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const startDate = getDateNDaysAgoInJakarta(6);
    const endDate = getTodayInJakarta();
    const res = await request.get(`${API_BASE}/api/dashboard/chart?period=day&startDate=${startDate}&endDate=${endDate}`, { headers: authHeader(token) });
    expect(res.ok(), `chart failed: ${res.status()}: ${await res.text()}`).toBeTruthy();
    const body = await res.json();
    expect(body).toBeDefined();
    const currentData = (body?.data as any[]) || [];
    expect(currentData.length).toBe(7);
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

  test('GET /api/dashboard/chart 30 days with Jakarta dates returns 30 data points', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const startDate = getDateNDaysAgoInJakarta(29);
    const endDate = getTodayInJakarta();
    const res = await request.get(`${API_BASE}/api/dashboard/chart?period=day&startDate=${startDate}&endDate=${endDate}`, { headers: authHeader(token) });
    expect(res.ok(), `30day chart failed: ${res.status()}: ${await res.text()}`).toBeTruthy();
    const body = await res.json();
    const currentData = (body?.data as any[]) || [];
    expect(currentData.length).toBe(30);
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

  test('POST /api/dashboard/export uses realtime filename for realtime period', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const res = await request.post(`${API_BASE}/api/dashboard/export`, {
      headers: authHeader(token),
      multipart: {
        format: 'xlsx',
        selectedPeriodType: 'realtime',
        chartType: 'hourly',
        currentTimeHour: '12:00',
      },
    });
    expect(res.ok(), `export failed: ${res.status()}: ${await res.text()}`).toBeTruthy();
    const disposition = res.headers()['content-disposition'];
    expect(disposition).toContain('revenue-report-realtime-');
  });

  test('POST /api/dashboard/export uses start-to-end filename for multi-day period', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const res = await request.post(`${API_BASE}/api/dashboard/export`, {
      headers: authHeader(token),
      multipart: {
        format: 'xlsx',
        selectedPeriodType: '7days',
        chartType: 'daily',
        startDate: '2026-08-07',
        endDate: '2026-08-13',
      },
    });
    expect(res.ok(), `export failed: ${res.status()}: ${await res.text()}`).toBeTruthy();
    const disposition = res.headers()['content-disposition'];
    expect(disposition).toContain('revenue-report-7days-2026-08-07-to-2026-08-13.xlsx');
  });

  test('POST /api/dashboard/export without auth returns 401', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/dashboard/export`, {
      data: { format: 'xlsx' },
    });
    expect(res.status()).toBe(401);
  });
});
