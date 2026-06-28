import { test, expect } from '@playwright/test';
import { TEST_USERS, API_BASE, authHeader } from './fixtures';

async function getToken(request: any, username: string = TEST_USERS.superadmin.username, password: string = TEST_USERS.superadmin.password) {
  const res = await request.post(`${API_BASE}/api/login`, {
    data: { username, password },
  });
  expect(res.ok(), `login failed for ${username}: ${res.status()}`).toBeTruthy();
  const body = await res.json();
  return body.access_token;
}

function uniqueName(prefix: string) {
  return `${prefix} E2E ${Date.now()}`;
}

function uniquePhone() {
  return `08${Date.now()}${Math.floor(Math.random() * 1000)}`.slice(0, 13);
}

function uniqueCode() {
  return `U${Date.now()}`.slice(0, 10);
}

// ============================================================================
// Category Export/Import
// ============================================================================
test.describe('Category Export/Import', () => {
  test('GET /api/categories/export CSV returns expected headers', async ({ request }) => {
    const token = await getToken(request);
    const res = await request.get(`${API_BASE}/api/categories/export?format=csv`, {
      headers: authHeader(token),
    });
    expect(res.ok()).toBeTruthy();
    expect(res.headers()['content-type']).toContain('text/csv');
    const text = await res.text();
    expect(text).toContain('Name');
    expect(text).toContain('Slug');
    expect(text).toContain('IsActive');
  });

  test('GET /api/categories/export XLSX returns file', async ({ request }) => {
    const token = await getToken(request);
    const res = await request.get(`${API_BASE}/api/categories/export?format=xlsx`, {
      headers: authHeader(token),
    });
    expect(res.ok()).toBeTruthy();
    expect(res.headers()['content-type']).toContain('spreadsheet');
  });

  test('POST /api/categories/import creates new category', async ({ request }) => {
    const token = await getToken(request);
    const name = uniqueName('Category');
    const csvContent = `Name,Description,IsActive\n${name},E2E test import,true\n`;
    const res = await request.post(`${API_BASE}/api/categories/import`, {
      headers: authHeader(token),
      multipart: {
        file: {
          name: 'categories.csv',
          mimeType: 'text/csv',
          buffer: Buffer.from(csvContent),
        },
      },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.inserted).toBeGreaterThanOrEqual(1);
    expect(body.errors).toHaveLength(0);
  });

  test('GET /api/categories/export without auth returns 401', async ({ request }) => {
    const res = await request.get(`${API_BASE}/api/categories/export`);
    expect(res.status()).toBe(401);
  });

  test('POST /api/categories/export with cashier returns 403', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
    const res = await request.get(`${API_BASE}/api/categories/export`, {
      headers: authHeader(token),
    });
    expect(res.status()).toBe(403);
  });
});

// ============================================================================
// Brand Export/Import
// ============================================================================
test.describe('Brand Export/Import', () => {
  test('GET /api/brands/export CSV returns expected headers', async ({ request }) => {
    const token = await getToken(request);
    const res = await request.get(`${API_BASE}/api/brands/export?format=csv`, {
      headers: authHeader(token),
    });
    expect(res.ok()).toBeTruthy();
    const text = await res.text();
    expect(text).toContain('Name');
    expect(text).toContain('Description');
    expect(text).toContain('IsActive');
  });

  test('GET /api/brands/export XLSX returns file', async ({ request }) => {
    const token = await getToken(request);
    const res = await request.get(`${API_BASE}/api/brands/export?format=xlsx`, {
      headers: authHeader(token),
    });
    expect(res.ok()).toBeTruthy();
    expect(res.headers()['content-type']).toContain('spreadsheet');
  });

  test('POST /api/brands/import creates new brand', async ({ request }) => {
    const token = await getToken(request);
    const name = uniqueName('Brand');
    const csvContent = `Name,Description,IsActive\n${name},E2E test import,true\n`;
    const res = await request.post(`${API_BASE}/api/brands/import`, {
      headers: authHeader(token),
      multipart: {
        file: {
          name: 'brands.csv',
          mimeType: 'text/csv',
          buffer: Buffer.from(csvContent),
        },
      },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.inserted).toBeGreaterThanOrEqual(1);
    expect(body.errors).toHaveLength(0);
  });

  test('GET /api/brands/export without auth returns 401', async ({ request }) => {
    const res = await request.get(`${API_BASE}/api/brands/export`);
    expect(res.status()).toBe(401);
  });
});

// ============================================================================
// Units of Measure Export/Import
// ============================================================================
test.describe('Units of Measure Export/Import', () => {
  test('GET /api/units-of-measure/export CSV returns expected headers', async ({ request }) => {
    const token = await getToken(request);
    const res = await request.get(`${API_BASE}/api/units-of-measure/export?format=csv`, {
      headers: authHeader(token),
    });
    expect(res.ok()).toBeTruthy();
    const text = await res.text();
    expect(text).toContain('Code');
    expect(text).toContain('Name');
  });

  test('GET /api/units-of-measure/export XLSX returns file', async ({ request }) => {
    const token = await getToken(request);
    const res = await request.get(`${API_BASE}/api/units-of-measure/export?format=xlsx`, {
      headers: authHeader(token),
    });
    expect(res.ok()).toBeTruthy();
    expect(res.headers()['content-type']).toContain('spreadsheet');
  });

  test('POST /api/units-of-measure/import creates new UOM', async ({ request }) => {
    const token = await getToken(request);
    const code = uniqueCode();
    const name = `${code} Name`;
    const csvContent = `Code,Name,Description,IsActive\n${code},${name},E2E test import,true\n`;
    const res = await request.post(`${API_BASE}/api/units-of-measure/import`, {
      headers: authHeader(token),
      multipart: {
        file: {
          name: 'uoms.csv',
          mimeType: 'text/csv',
          buffer: Buffer.from(csvContent),
        },
      },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.inserted).toBeGreaterThanOrEqual(1);
    expect(body.errors).toHaveLength(0);
  });

  test('GET /api/units-of-measure/export without auth returns 401', async ({ request }) => {
    const res = await request.get(`${API_BASE}/api/units-of-measure/export`);
    expect(res.status()).toBe(401);
  });
});

// ============================================================================
// Customer Export/Import
// ============================================================================
test.describe('Customer Export/Import', () => {
  test('GET /api/customers/export CSV returns expected headers', async ({ request }) => {
    const token = await getToken(request);
    const res = await request.get(`${API_BASE}/api/customers/export?format=csv`, {
      headers: authHeader(token),
    });
    expect(res.ok()).toBeTruthy();
    const text = await res.text();
    expect(text).toContain('Name');
    expect(text).toContain('Phone');
  });

  test('GET /api/customers/export XLSX returns file', async ({ request }) => {
    const token = await getToken(request);
    const res = await request.get(`${API_BASE}/api/customers/export?format=xlsx`, {
      headers: authHeader(token),
    });
    expect(res.ok()).toBeTruthy();
    expect(res.headers()['content-type']).toContain('spreadsheet');
  });

  test('POST /api/customers/import creates new customer', async ({ request }) => {
    const token = await getToken(request);
    const phone = uniquePhone();
    const csvContent = `Name,Phone,Email,Address,Note,IsActive\nE2E Customer,${phone},e2e.${Date.now()}@test.com,Test Address,Import test,true\n`;
    const res = await request.post(`${API_BASE}/api/customers/import`, {
      headers: authHeader(token),
      multipart: {
        file: {
          name: 'customers.csv',
          mimeType: 'text/csv',
          buffer: Buffer.from(csvContent),
        },
      },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.inserted).toBeGreaterThanOrEqual(1);
    expect(body.errors).toHaveLength(0);
  });

  test('GET /api/customers/export without auth returns 401', async ({ request }) => {
    const res = await request.get(`${API_BASE}/api/customers/export`);
    expect(res.status()).toBe(401);
  });

  test('POST /api/customers/import with cashier returns 403', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
    const csvContent = `Name,Phone,Email,Address,Note,IsActive\nTest,081111,test@test.com,,,true\n`;
    const res = await request.post(`${API_BASE}/api/customers/import`, {
      headers: authHeader(token),
      multipart: {
        file: {
          name: 'customers.csv',
          mimeType: 'text/csv',
          buffer: Buffer.from(csvContent),
        },
      },
    });
    expect(res.status()).toBe(403);
  });
});

// ============================================================================
// Product Export/Import
// ============================================================================
test.describe('Product Export/Import', () => {
  test('GET /api/products/export CSV returns expected headers', async ({ request }) => {
    const token = await getToken(request);
    const res = await request.get(`${API_BASE}/api/products/export?format=csv`, {
      headers: authHeader(token),
    });
    expect(res.ok()).toBeTruthy();
    const text = await res.text();
    expect(text).toContain('SKU');
    expect(text).toContain('Name');
    expect(text).toContain('Price');
  });

  test('GET /api/products/export XLSX returns file', async ({ request }) => {
    const token = await getToken(request);
    const res = await request.get(`${API_BASE}/api/products/export?format=xlsx`, {
      headers: authHeader(token),
    });
    expect(res.ok()).toBeTruthy();
    expect(res.headers()['content-type']).toContain('spreadsheet');
  });

  test('POST /api/products/import creates new product', async ({ request }) => {
    const token = await getToken(request);
    const sku = `SKU${Date.now()}`;
    const csvContent = `SKU,Name,Barcode,Category,Brand,Price,Cost,Stock,Status,UnitOfMeasure,WeightGrams,Description\n${sku},E2E Product,,General,General,15000,10000,10,active,PCS,250,E2E test product\n`;
    const res = await request.post(`${API_BASE}/api/products/import`, {
      headers: authHeader(token),
      multipart: {
        file: {
          name: 'products.csv',
          mimeType: 'text/csv',
          buffer: Buffer.from(csvContent),
        },
      },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.inserted).toBeGreaterThanOrEqual(1);
    expect(body.errors).toHaveLength(0);
  });

  test('POST /api/products/import invalid CSV returns error details', async ({ request }) => {
    const token = await getToken(request);
    const csvContent = `SKU,Name,Barcode,Category,Brand,Price,Cost,Stock,Status,UnitOfMeasure,WeightGrams,Description\n,Incomplete Product,,,,,,,,,,\n`;
    const res = await request.post(`${API_BASE}/api/products/import`, {
      headers: authHeader(token),
      multipart: {
        file: {
          name: 'products.csv',
          mimeType: 'text/csv',
          buffer: Buffer.from(csvContent),
        },
      },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.errors.length).toBeGreaterThanOrEqual(1);
  });

  test('GET /api/products/export without auth returns 401', async ({ request }) => {
    const res = await request.get(`${API_BASE}/api/products/export`);
    expect(res.status()).toBe(401);
  });

  test('POST /api/products/import without auth returns 401', async ({ request }) => {
    const csvContent = 'SKU,Name\nTEST,Test\n';
    const res = await request.post(`${API_BASE}/api/products/import`, {
      multipart: {
        file: {
          name: 'products.csv',
          mimeType: 'text/csv',
          buffer: Buffer.from(csvContent),
        },
      },
    });
    expect(res.status()).toBe(401);
  });
});
