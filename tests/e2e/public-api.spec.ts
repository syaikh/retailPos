import { test, expect } from '@playwright/test';
import { API_BASE } from './fixtures';

test.describe('Public API - Tax Classes', () => {

  test('GET /api/tax-classes returns array of tax classes', async ({ request }) => {
    const res = await request.get(`${API_BASE}/api/tax-classes`);
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data).toBeInstanceOf(Array);
    if (body.data.length > 0) {
      const item = body.data[0];
      expect(item.id).toBeDefined();
      expect(item.name).toBeDefined();
      expect(item.rate_percent).toBeDefined();
    }
  });

  test('GET /api/tax-classes is accessible without auth', async ({ request }) => {
    const res = await request.get(`${API_BASE}/api/tax-classes`);
    expect(res.ok()).toBeTruthy();
  });

});

test.describe('Public API - Stock Thresholds', () => {

  test('GET /api/stock-thresholds returns warning and critical values', async ({ request }) => {
    const res = await request.get(`${API_BASE}/api/stock-thresholds`);
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.warning).toBe(10);
    expect(body.critical).toBe(5);
  });

  test('GET /api/stock-thresholds is accessible without auth', async ({ request }) => {
    const res = await request.get(`${API_BASE}/api/stock-thresholds`);
    expect(res.ok()).toBeTruthy();
  });

});
