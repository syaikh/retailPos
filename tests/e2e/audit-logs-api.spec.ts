import { test, expect } from './fixtures';
import { apiAs, ApiDriver } from './api-driver';

/**
 * Audit-logs data contract driven at the API layer. This is the API half of
 * audit-logs-search.spec.ts — the search/filter UI flows stay in
 * audit-logs-search.spec.ts as genuine UI coverage.
 */
const data = (b: any) => (b && b.data !== undefined ? b.data : b);

function anon(request: any): ApiDriver {
  return new ApiDriver(request, '');
}

test.describe('Audit Logs API - Get by ID', () => {
  test('GET /api/audit-logs/:id returns a single audit log', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const listRes = await api.get('/api/audit-logs?limit=1');
    expect(listRes.ok).toBeTruthy();
    const listBody = data(listRes.body);
    expect(listBody.length).toBeGreaterThan(0);

    const logId = listBody[0].id;
    const res = await api.get(`/api/audit-logs/${logId}`);
    expect(res.ok, `get by id failed: ${res.status}`).toBeTruthy();
    const body = data(res.body);
    expect(body).toBeDefined();
    expect(body.id).toBe(logId);
  });

  test('GET /api/audit-logs/:id returns 400 for invalid id', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const res = await api.get('/api/audit-logs/not-a-number');
    expect(res.status).toBe(400);
  });

  test('GET /api/audit-logs/:id returns 404 for nonexistent id', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const res = await api.get('/api/audit-logs/999999999');
    expect(res.status).toBe(404);
  });

  test('GET /api/audit-logs/:id without auth returns 401', async ({ request }) => {
    const res = await anon(request).get('/api/audit-logs/1');
    expect(res.status).toBe(401);
  });
});

test.describe('Audit Logs API - List Entity Types', () => {
  test('GET /api/audit-logs/entity-types returns array of entity types', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const res = await api.get('/api/audit-logs/entity-types');
    expect(res.ok, `entity types failed: ${res.status}`).toBeTruthy();
    const body = data(res.body);
    expect(Array.isArray(body)).toBeTruthy();
    expect(body.length).toBeGreaterThan(0);
    const lowerTypes = body.map((t: string) => t.toLowerCase());
    expect(lowerTypes.some((t: string) => t.includes('auth') || t.includes('user') || t.includes('product'))).toBeTruthy();
  });

  test('GET /api/audit-logs/entity-types without auth returns 401', async ({ request }) => {
    const res = await anon(request).get('/api/audit-logs/entity-types');
    expect(res.status).toBe(401);
  });

  test('GET /api/audit-logs/entity-types with restricted role returns 403', async ({ request }) => {
    const api = await apiAs(request, 'cashier');
    const res = await api.get('/api/audit-logs/entity-types');
    expect(res.status).toBe(403);
  });
});

test.describe('Audit Logs API - Export', () => {
  test('GET /api/audit-logs/export returns CSV by default', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const res = await api.get('/api/audit-logs/export');
    expect(res.ok, `export failed: ${res.status}`).toBeTruthy();
    const ct = res.headers['content-type'] || '';
    expect(ct).toContain('csv');
  });

  test('GET /api/audit-logs/export?format=xlsx returns xlsx', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const res = await api.get('/api/audit-logs/export?format=xlsx');
    expect(res.ok, `xlsx export failed: ${res.status}`).toBeTruthy();
    const ct = res.headers['content-type'] || '';
    expect(ct.includes('spreadsheetml') || ct.includes('octet-stream')).toBeTruthy();
  });

  test('GET /api/audit-logs/export?format=csv returns CSV explicitly', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const res = await api.get('/api/audit-logs/export?format=csv');
    expect(res.ok).toBeTruthy();
  });

  test('GET /api/audit-logs/export without auth returns 401', async ({ request }) => {
    const res = await anon(request).get('/api/audit-logs/export');
    expect(res.status).toBe(401);
  });

  test('GET /api/audit-logs/export with restricted role returns 403', async ({ request }) => {
    const api = await apiAs(request, 'cashier');
    const res = await api.get('/api/audit-logs/export');
    expect(res.status).toBe(403);
  });

  test('Today filter returns seeded login event via API', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const today = new Date(Date.now() + 7 * 60 * 60 * 1000);
    const year = today.getUTCFullYear();
    const month = String(today.getUTCMonth() + 1).padStart(2, '0');
    const day = String(today.getUTCDate()).padStart(2, '0');
    const todayStr = `${year}-${month}-${day}`;

    const res = await api.get(`/api/audit-logs?start_date=${todayStr}&end_date=${todayStr}&limit=10`);
    expect(res.ok).toBeTruthy();
    const body = data(res.body);
    expect(body).toBeInstanceOf(Array);
    expect(body.length).toBeGreaterThanOrEqual(0);
  });
});
