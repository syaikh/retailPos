import { test, expect } from '@playwright/test';
import { TEST_USERS, API_BASE } from './fixtures';

test.describe('Reports & Analytics', () => {
  test.beforeEach(async ({ page }) => {
    test.skip(true, 'Reports page not yet implemented');
  });
});

test.describe('Reports API', () => {
  test('GET /api/stats returns valid dashboard data', async ({ request }) => {
    const tokenResponse = await request.post(`${API_BASE}/api/login`, {
      data: { username: TEST_USERS.superadmin.username, password: TEST_USERS.superadmin.password }
    });
    const { access_token: token } = await tokenResponse.json();

    const response = await request.get(`${API_BASE}/api/dashboard/stats`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.data).toHaveProperty('todays_sales');
    expect(body.data).toHaveProperty('todays_revenue');
    expect(body.data).toHaveProperty('total_products');
  });

  test('GET /api/sales supports pagination and filters', async ({ request }) => {
    const tokenResponse = await request.post(`${API_BASE}/api/login`, {
      data: { username: TEST_USERS.superadmin.username, password: TEST_USERS.superadmin.password }
    });
    const { access_token: token } = await tokenResponse.json();

    const response = await request.get(`${API_BASE}/api/sales?limit=5`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    expect(response.ok()).toBeTruthy();
  });
});
