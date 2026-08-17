import { test, expect } from './fixtures';
import { TEST_USERS, API_BASE, authHeader, getToken } from './fixtures';

test.describe('Payment Methods', () => {
  test('lists active payment methods publicly', async ({ request }) => {
    const res = await request.get(`${API_BASE}/api/payment-methods`);
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(Array.isArray(data.data)).toBeTruthy();
    expect(data.data.length).toBeGreaterThanOrEqual(5);
    const codes = data.data.map((m: any) => m.code);
    expect(codes).toContain('CASH');
    expect(codes).toContain('CARD');
    expect(codes).toContain('E_WALLET');
  });

  test('returns cash method by code', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const res = await request.get(`${API_BASE}/api/payment-methods/CASH`, {
      headers: authHeader(token),
    });
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.data.code).toBe('CASH');
    expect(data.data.name).toBe('Cash');
    expect(data.data.is_active).toBe(true);
  });

  test('return 404 for unknown code', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const res = await request.get(`${API_BASE}/api/payment-methods/UNKNOWN`, {
      headers: authHeader(token),
    });
    expect(res.status()).toBe(404);
  });
});
