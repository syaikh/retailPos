import { test, expect } from '@playwright/test';
import { TEST_USERS, API_BASE, authHeader, getToken as cachedGetToken } from './fixtures';

const getToken = cachedGetToken;

test.describe('Dashboard API - Available Years', () => {

  test('GET /api/dashboard/years returns array of years', async ({ request }) => {
    const token = await getToken(request);
    const res = await request.get(`${API_BASE}/api/dashboard/years`, {
      headers: authHeader(token),
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(Array.isArray(body.data)).toBeTruthy();
    if (body.data.length > 0) {
      expect(typeof body.data[0]).toBe('number');
    }
  });

  test('GET /api/dashboard/years without auth returns 401', async ({ request }) => {
    const res = await request.get(`${API_BASE}/api/dashboard/years`);
    expect(res.status()).toBe(401);
  });

});

test.describe('Dashboard API - Export Report', () => {

  test('POST /api/dashboard/export returns file with correct headers', async ({ request }) => {
    const token = await getToken(request);
    const res = await request.post(`${API_BASE}/api/dashboard/export`, {
      headers: authHeader(token),
      form: {
        chartData: JSON.stringify({ labels: ['Jan'], values: [1000] }),
        period: 'daily',
        mode: 'current',
      },
    });
    expect(res.ok()).toBeTruthy();
    const contentType = res.headers()['content-type'] || '';
    expect(contentType).toContain('openxmlformats');
  });

  test('POST /api/dashboard/export without auth returns 401', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/dashboard/export`);
    expect(res.status()).toBe(401);
  });

  test('POST /api/dashboard/export with invalid date returns 400', async ({ request }) => {
    const token = await getToken(request);
    const res = await request.post(`${API_BASE}/api/dashboard/export`, {
      headers: authHeader(token),
      form: {
        chartData: '{}',
        period: 'daily',
        mode: 'current',
        date: 'invalid-date',
      },
    });
    expect(res.status()).toBe(400);
  });

});
