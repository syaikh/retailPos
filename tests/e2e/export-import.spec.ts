import { test, expect } from '@playwright/test';
import { TEST_USERS, API_BASE, authHeader } from './fixtures';
import * as XLSX from 'xlsx';

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

function createXLSX(headers: string[], rows: (string | number)[][]): Buffer {
  const wb = XLSX.utils.book_new();
  const ws = XLSX.utils.aoa_to_sheet([headers, ...rows]);
  XLSX.utils.book_append_sheet(wb, ws, 'Data');
  return Buffer.from(XLSX.write(wb, { type: 'buffer', bookType: 'xlsx' }));
}

async function xlsxRoundtrip(
  request: any,
  token: string,
  module: string,
  headers: string[],
  rows: (string | number)[][],
  verifyEndpoint: string,
  verifyField: string,
  expectedValue: string,
) {
  // Step 1: preview
  const buf = createXLSX(headers, rows);
  await new Promise(r => setTimeout(r, 300));
  const previewRes = await request.post(`${API_BASE}/api/import-export/preview/${module}`, {
    headers: authHeader(token),
    multipart: {
      file: {
        name: `${module}.xlsx`,
        mimeType: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        buffer: buf,
      },
    },
  });
  expect(previewRes.ok(), `preview failed (${previewRes.status()}): ${await previewRes.text()}`).toBeTruthy();
  const preview = await previewRes.json();
  expect(preview.module).toBe(module);
  expect(preview.total_rows).toBe(rows.length);
  if (preview.error_count > 0) {
    console.log('Preview errors:', JSON.stringify(preview.rows?.filter((r: any) => r.errors?.length > 0)?.slice(0, 3), null, 2));
  }
  expect(preview.insert_count).toBe(rows.length);
  expect(preview.token).toBeTruthy();

  // Step 2: confirm
  await new Promise(r => setTimeout(r, 300));
  const confirmRes = await request.post(`${API_BASE}/api/import-export/confirm/${module}?token=${preview.token}`, {
    headers: authHeader(token),
  });
  expect(confirmRes.ok(), `confirm failed: ${await confirmRes.text()}`).toBeTruthy();
  const confirmBody = await confirmRes.json();
  const jobId = confirmBody.job_id;

  // Step 3: poll progress until completed
  let status = confirmBody.status;
  const maxPolls = 30;
  for (let i = 0; i < maxPolls; i++) {
    if (status === 'completed') break;
    await new Promise(r => setTimeout(r, 500));
    const progRes = await request.get(`${API_BASE}/api/import-export/progress/${jobId}`, {
      headers: authHeader(token),
    });
    if (progRes.ok()) {
      const prog = await progRes.json();
      status = prog.status;
    }
  }
  expect(status, 'import should complete within timeout').toBe('completed');

  await new Promise(r => setTimeout(r, 300));

  // Step 4: verify via API
  const verifyRes = await request.get(`${verifyEndpoint}`, {
    headers: authHeader(token),
  });
  expect(verifyRes.ok()).toBeTruthy();
  const body = await verifyRes.json();
  const items = body.data ?? [];
  const found = items.some((item: any) => String(item[verifyField]) === expectedValue);
  expect(found, `expected ${verifyField}=${expectedValue} to exist after import`).toBeTruthy();
}

// ============================================================================
// Unified Import/Export Framework
// ============================================================================
test.describe('Import/Export Framework — Modules', () => {
  test('GET /api/import-export/modules returns registered modules', async ({ request }) => {
    const token = await getToken(request);
    const res = await request.get(`${API_BASE}/api/import-export/modules`, {
      headers: authHeader(token),
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(Array.isArray(body)).toBeTruthy();
    const names = body.map((m: any) => m.name);
    expect(names).toContain('brands');
    expect(names).toContain('categories');
    expect(names).toContain('uoms');
    expect(names).toContain('customers');
    expect(names).toContain('products');
  });

  test('GET /api/import-export/modules without auth returns 401', async ({ request }) => {
    const res = await request.get(`${API_BASE}/api/import-export/modules`);
    expect(res.status()).toBe(401);
  });
});

// ============================================================================
// Export
// ============================================================================
test.describe('Import/Export Framework — Export', () => {
  for (const mod of ['categories', 'brands', 'uoms', 'customers', 'products']) {
    test(`GET /api/import-export/export/${mod} CSV returns expected headers`, async ({ request }) => {
      const token = await getToken(request);
      const res = await request.get(`${API_BASE}/api/import-export/export/${mod}?format=csv`, {
        headers: authHeader(token),
      });
      expect(res.ok()).toBeTruthy();
      expect(res.headers()['content-type']).toContain('text/csv');
    });

    test(`GET /api/import-export/export/${mod} XLSX returns file`, async ({ request }) => {
      const token = await getToken(request);
      const res = await request.get(`${API_BASE}/api/import-export/export/${mod}?format=xlsx`, {
        headers: authHeader(token),
      });
      expect(res.ok()).toBeTruthy();
      expect(res.headers()['content-type']).toContain('spreadsheet');
    });
  }

  test('GET /api/import-export/export/brands without auth returns 401', async ({ request }) => {
    const res = await request.get(`${API_BASE}/api/import-export/export/brands`);
    expect(res.status()).toBe(401);
  });

  test('GET /api/import-export/export/brands with cashier returns 403', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
    const res = await request.get(`${API_BASE}/api/import-export/export/brands`, {
      headers: authHeader(token),
    });
    expect(res.status()).toBe(403);
  });

  test('GET /api/import-export/export/unknown-module returns 400', async ({ request }) => {
    const token = await getToken(request);
    const res = await request.get(`${API_BASE}/api/import-export/export/unknown-module`, {
      headers: authHeader(token),
    });
    expect(res.status()).toBe(400);
  });
});

// ============================================================================
// Import — Brands
// ============================================================================
test.describe('Import/Export Framework — Import Brands', () => {
  test('POST /api/import-export/preview/brands previews new brands', async ({ request }) => {
    const token = await getToken(request);
    const name = uniqueName('Brand');
    const csvContent = `Name,Description,IsActive\n${name},E2E test import,true\n`;
    const res = await request.post(`${API_BASE}/api/import-export/preview/brands`, {
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
    expect(body.module).toBe('brands');
    expect(body.total_rows).toBe(1);
    expect(body.insert_count).toBe(1);
    expect(body.token).toBeTruthy();
  });

  test('POST /api/import-export/confirm/brands confirms import', async ({ request }) => {
    const token = await getToken(request);
    const name = uniqueName('Brand');
    const csvContent = `Name,Description,IsActive\n${name},E2E test confirm,true\n`;

    // Step 1: preview
    const previewRes = await request.post(`${API_BASE}/api/import-export/preview/brands`, {
      headers: authHeader(token),
      multipart: {
        file: {
          name: 'brands.csv',
          mimeType: 'text/csv',
          buffer: Buffer.from(csvContent),
        },
      },
    });
    expect(previewRes.ok()).toBeTruthy();
    const preview = await previewRes.json();

    // Step 2: confirm
    const confirmRes = await request.post(`${API_BASE}/api/import-export/confirm/brands?token=${preview.token}`, {
      headers: authHeader(token),
    });
    expect(confirmRes.ok()).toBeTruthy();
    const confirmBody = await confirmRes.json();
    expect(confirmBody.job_id).toBeGreaterThan(0);
    const jobId = confirmBody.job_id;

    // Step 3: poll progress until completed
    let status = confirmBody.status;
    const maxPolls = 30;
    for (let i = 0; i < maxPolls; i++) {
      if (status === 'completed') break;
      await new Promise(r => setTimeout(r, 500));
      const progRes = await request.get(`${API_BASE}/api/import-export/progress/${jobId}`, {
        headers: authHeader(token),
      });
      if (progRes.ok()) {
        const prog = await progRes.json();
        status = prog.status;
      }
    }
    expect(status, 'import should complete within timeout').toBe('completed');

    // Step 4: verify via progress endpoint
    const finalRes = await request.get(`${API_BASE}/api/import-export/progress/${jobId}`, {
      headers: authHeader(token),
    });
    expect(finalRes.ok()).toBeTruthy();
    const final = await finalRes.json();
    expect(final.inserted).toBeGreaterThanOrEqual(1);
  });

  test('POST /api/import-export/confirm/brands without token returns 400', async ({ request }) => {
    const token = await getToken(request);
    const res = await request.post(`${API_BASE}/api/import-export/confirm/brands`, {
      headers: authHeader(token),
    });
    expect(res.status()).toBe(400);
  });
});

// ============================================================================
// Import — Categories
// ============================================================================
test.describe('Import/Export Framework — Import Categories', () => {
  test('POST /api/import-export/preview/categories previews new categories', async ({ request }) => {
    const token = await getToken(request);
    const name = uniqueName('Category');
    const csvContent = `Name,Description,IsActive\n${name},E2E test import,true\n`;
    const res = await request.post(`${API_BASE}/api/import-export/preview/categories`, {
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
    expect(body.module).toBe('categories');
    expect(body.total_rows).toBe(1);
    expect(body.insert_count).toBe(1);
    expect(body.token).toBeTruthy();
  });

  test('POST /api/import-export/preview/categories with cashier returns 403', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
    const csvContent = `Name,Description,IsActive\nTest,Test,true\n`;
    const res = await request.post(`${API_BASE}/api/import-export/preview/categories`, {
      headers: authHeader(token),
      multipart: {
        file: {
          name: 'categories.csv',
          mimeType: 'text/csv',
          buffer: Buffer.from(csvContent),
        },
      },
    });
    expect(res.status()).toBe(403);
  });
});

// ============================================================================
// Import — Units of Measure
// ============================================================================
test.describe('Import/Export Framework — Import UOMs', () => {
  test('POST /api/import-export/preview/uoms previews new UOMs', async ({ request }) => {
    const token = await getToken(request);
    const code = uniqueCode();
    const name = `${code} Name`;
    const csvContent = `Code,Name,Description,IsActive\n${code},${name},E2E test import,true\n`;
    const res = await request.post(`${API_BASE}/api/import-export/preview/uoms`, {
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
    expect(body.module).toBe('uoms');
    expect(body.total_rows).toBe(1);
    expect(body.insert_count).toBe(1);
    expect(body.token).toBeTruthy();
  });
});

// ============================================================================
// Import — Customers
// ============================================================================
test.describe('Import/Export Framework — Import Customers', () => {
  test('POST /api/import-export/preview/customers previews new customers', async ({ request }) => {
    const token = await getToken(request);
    const phone = uniquePhone();
    const csvContent = `Name,Phone,Email,Address,Note,IsActive\nE2E Customer,${phone},e2e.${Date.now()}@test.com,Test Address,Import test,true\n`;
    const res = await request.post(`${API_BASE}/api/import-export/preview/customers`, {
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
    expect(body.module).toBe('customers');
    expect(body.total_rows).toBe(1);
    expect(body.insert_count).toBe(1);
    expect(body.token).toBeTruthy();
  });

  test('POST /api/import-export/preview/customers with cashier returns 403', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
    const csvContent = `Name,Phone,Email,Address,Note,IsActive\nTest,081111,test@test.com,,,true\n`;
    const res = await request.post(`${API_BASE}/api/import-export/preview/customers`, {
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
// Import — Products
// ============================================================================
test.describe('Import/Export Framework — Import Products', () => {
  test('POST /api/import-export/preview/products previews new products', async ({ request }) => {
    const token = await getToken(request);
    const sku = `SKU${Date.now()}`;
    const csvContent = `SKU,Name,Barcode,Category,Brand,Price,Cost,Stock,Status,UnitOfMeasure,WeightGrams,Description\n${sku},E2E Product,,General,General,15000,10000,10,active,PCS,250,E2E test product\n`;
    const res = await request.post(`${API_BASE}/api/import-export/preview/products`, {
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
    expect(body.module).toBe('products');
    expect(body.token).toBeTruthy();
  });

  test('POST /api/import-export/preview/products invalid CSV returns validation errors', async ({ request }) => {
    const token = await getToken(request);
    const csvContent = `SKU,Name,Barcode,Category,Brand,Price,Cost,Stock,Status,UnitOfMeasure,WeightGrams,Description\n,Incomplete Product,,,,,,,,,,\n`;
    const res = await request.post(`${API_BASE}/api/import-export/preview/products`, {
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
    expect(body.error_count).toBeGreaterThanOrEqual(1);
  });

  test('POST /api/import-export/preview/products without auth returns 401', async ({ request }) => {
    const csvContent = 'SKU,Name\nTEST,Test\n';
    const res = await request.post(`${API_BASE}/api/import-export/preview/products`, {
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

// ============================================================================
// XLSX Roundtrip
// ============================================================================
test.describe('Import/Export Framework — XLSX Roundtrip', () => {
  test('Categories XLSX: preview → confirm → verify', async ({ request }) => {
    const token = await getToken(request);
    const name = uniqueName('CatXLSX');
    await xlsxRoundtrip(
      request, token,
      'categories',
      ['Name', 'Description', 'Active'],
      [[name, 'XLSX roundtrip test', true]],
      `${API_BASE}/api/categories`,
      'name',
      name,
    );
  });

  test('Brands XLSX: preview → confirm → verify', async ({ request }) => {
    const token = await getToken(request);
    const name = uniqueName('BrandXLSX');
    await xlsxRoundtrip(
      request, token,
      'brands',
      ['Name', 'Description', 'IsActive'],
      [[name, 'XLSX roundtrip test', true]],
      `${API_BASE}/api/brands`,
      'name',
      name,
    );
  });

  test('Products XLSX: preview → confirm → verify', async ({ request }) => {
    const token = await getToken(request);
    const sku = `SKUXLSX${Date.now()}`;
    await xlsxRoundtrip(
      request, token,
      'products',
      ['SKU', 'Product Name', 'Barcode', 'Category', 'Brand', 'Price', 'Cost', 'Status', 'Unit of Measure'],
      [[sku, 'XLSX Product', '', 'Accessories', 'Indofood', 15000, 10000, 'active', 'pcs']],
      `${API_BASE}/api/products?limit=200&search=${sku}`,
      'sku',
      sku,
    );
  });
});
