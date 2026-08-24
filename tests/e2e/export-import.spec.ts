import { test, expect } from './fixtures';
import { apiAs, ApiDriver } from './api-driver';
import * as XLSX from 'xlsx';

// ============================================================================
// Import/Export (template download, modules, export formats, CSV/XLSX import
// roundtrips, RBAC). All assertions run on the API driver — no browser.
// ============================================================================

const data = (b: any) => (b && b.data !== undefined ? b.data : b);

test.describe('Import/Export - Template Download', () => {
  let api: ApiDriver;
  test.beforeEach(async ({ request }) => {
    api = await apiAs(request, 'superadmin');
  });

  test('GET /api/import-export/modules returns available modules', async () => {
    const res = await api.get('/api/import-export/modules');
    expect(res.ok, `modules failed: ${res.status}`).toBeTruthy();
    const modules = Array.isArray(res.body) ? res.body : res.body.data;
    expect(Array.isArray(modules)).toBeTruthy();
    expect(modules.length).toBeGreaterThan(0);
  });

  test('GET /api/import-export/template/products returns xlsx template', async () => {
    const res = await api.get('/api/import-export/template/products');
    expect(res.ok, `template failed: ${res.status}`).toBeTruthy();
    const ct = res.headers['content-type'] || '';
    expect(ct.includes('spreadsheetml') || ct.includes('octet-stream')).toBeTruthy();
  });

  test('GET /api/import-export/template/customers returns xlsx template', async () => {
    const res = await api.get('/api/import-export/template/customers');
    expect(res.ok).toBeTruthy();
  });

  test('GET /api/import-export/template/categories returns xlsx template', async () => {
    const res = await api.get('/api/import-export/template/categories');
    expect(res.ok).toBeTruthy();
  });

  test('GET /api/import-export/template/brands returns xlsx template', async () => {
    const res = await api.get('/api/import-export/template/brands');
    expect(res.ok).toBeTruthy();
  });

  test('GET /api/import-export/template/products without auth returns 401', async ({ request }) => {
    const res = await new ApiDriver(request, '').get('/api/import-export/template/products');
    expect(res.status).toBe(401);
  });
});

test.describe('Import/Export - Cancel Import', () => {
  let api: ApiDriver;
  test.beforeEach(async ({ request }) => {
    api = await apiAs(request, 'superadmin');
  });

  test('POST /api/import-export/cancel/:jobId with nonexistent job returns error', async () => {
    const res = await api.post('/api/import-export/cancel/nonexistent-job-id', {});
    expect([400, 404, 410]).toContain(res.status);
  });

  test('POST /api/import-export/cancel/:jobId without auth returns 401', async ({ request }) => {
    const res = await new ApiDriver(request, '').post('/api/import-export/cancel/some-job', {});
    expect(res.status).toBe(401);
  });
});

test.describe('Import/Export - Import History', () => {
  let api: ApiDriver;
  test.beforeEach(async ({ request }) => {
    api = await apiAs(request, 'superadmin');
  });

  test('GET /api/import-export/history/products returns history', async () => {
    const res = await api.get('/api/import-export/history/products');
    expect(res.ok, `history failed: ${res.status}`).toBeTruthy();
    const history = Array.isArray(res.body) ? res.body : res.body.data || [];
    expect(Array.isArray(history)).toBeTruthy();
  });

  test('GET /api/import-export/history/products without auth returns 401', async ({ request }) => {
    const res = await new ApiDriver(request, '').get('/api/import-export/history/products');
    expect(res.status).toBe(401);
  });
});

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
  api: ApiDriver,
  module: string,
  headers: string[],
  rows: (string | number)[][],
  verifyPath: string,
  verifyField: string,
  expectedValue: string,
) {
  // Step 1: preview
  const buf = createXLSX(headers, rows);
  await new Promise(r => setTimeout(r, 300));
  const previewRes = await api.multipart(
    `/api/import-export/preview/${module}`,
    {
      name: `${module}.xlsx`,
      mimeType: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
      buffer: buf,
    },
  );
  expect(previewRes.ok, `preview failed (${previewRes.status}): ${JSON.stringify(previewRes.body)}`).toBeTruthy();
  const preview = previewRes.body;
  expect(preview.module).toBe(module);
  expect(preview.total_rows).toBe(rows.length);
  if (preview.error_count > 0) {
    console.log('Preview errors:', JSON.stringify(preview.rows?.filter((r: any) => r.errors?.length > 0)?.slice(0, 3), null, 2));
  }
  expect(preview.insert_count).toBe(rows.length);
  expect(preview.token).toBeTruthy();
  console.log(`Preview for ${module}: total=${preview.total_rows} insert=${preview.insert_count} update=${preview.update_count} errors=${preview.error_count}`);
  if (preview.error_count > 0) {
    console.log('Preview errors:', JSON.stringify(preview.rows?.filter((r: any) => r.errors?.length > 0)?.slice(0, 3), null, 2));
  }

  // Step 2: confirm
  await new Promise(r => setTimeout(r, 300));
  const confirmRes = await api.post(`/api/import-export/confirm/${module}?token=${preview.token}`, {});
  expect(confirmRes.ok, `confirm failed: ${JSON.stringify(confirmRes.body)}`).toBeTruthy();
  const confirmBody = confirmRes.body;
  const jobId = confirmBody.job_id;
  console.log(`Confirm response for ${module}: job_id=${jobId} status=${confirmBody.status}`);

  // Step 3: poll progress until completed
  let status = confirmBody.status;
  const maxPolls = 30;
  for (let i = 0; i < maxPolls; i++) {
    if (status === 'completed') break;
    await new Promise(r => setTimeout(r, 500));
    const progRes = await api.get(`/api/import-export/progress/${jobId}`);
    if (progRes.ok) {
      status = progRes.body.status;
      console.log(`Progress ${module} poll ${i}: status=${status} inserted=${progRes.body.inserted_rows} total=${progRes.body.total_rows}`);
    }
  }
  expect(status, 'import should complete within timeout').toBe('completed');

  await new Promise(r => setTimeout(r, 500));

  // Step 4: verify via API (retry a few times to handle eventual consistency / cache)
  let found = false;
  for (let attempt = 0; attempt < 5; attempt++) {
    const verifyRes = await api.get(verifyPath);
    expect(verifyRes.ok).toBeTruthy();
    const items = data(verifyRes.body) ?? [];
    found = items.some((item: any) => String(item[verifyField]) === expectedValue);
    if (found) break;
    await new Promise(r => setTimeout(r, 1000));
  }
  expect(found, `expected ${verifyField}=${expectedValue} to exist after import`).toBeTruthy();
}

// ============================================================================
// Unified Import/Export Framework
// ============================================================================
test.describe('Import/Export Framework — Modules', () => {
  let api: ApiDriver;
  test.beforeEach(async ({ request }) => {
    api = await apiAs(request, 'superadmin');
  });

  test('GET /api/import-export/modules returns registered modules', async () => {
    const res = await api.get('/api/import-export/modules');
    expect(res.ok).toBeTruthy();
    const body = res.body;
    expect(Array.isArray(body)).toBeTruthy();
    const names = body.map((m: any) => m.name);
    expect(names).toContain('brands');
    expect(names).toContain('categories');
    expect(names).toContain('uoms');
    expect(names).toContain('customers');
    expect(names).toContain('products');
  });

  test('GET /api/import-export/modules without auth returns 401', async ({ request }) => {
    const res = await new ApiDriver(request, '').get('/api/import-export/modules');
    expect(res.status).toBe(401);
  });
});

// ============================================================================
// Export
// ============================================================================
test.describe('Import/Export Framework — Export', () => {
  let api: ApiDriver;
  let cashierApi: ApiDriver;
  test.beforeEach(async ({ request }) => {
    api = await apiAs(request, 'superadmin');
    cashierApi = await apiAs(request, 'cashier');
  });

  for (const mod of ['categories', 'brands', 'uoms', 'customers', 'products']) {
    test(`GET /api/import-export/export/${mod} CSV returns expected headers`, async () => {
      const res = await api.get(`/api/import-export/export/${mod}?format=csv`);
      expect(res.ok).toBeTruthy();
      expect(res.headers['content-type']).toContain('text/csv');
    });

    test(`GET /api/import-export/export/${mod} XLSX returns file`, async () => {
      const res = await api.get(`/api/import-export/export/${mod}?format=xlsx`);
      expect(res.ok).toBeTruthy();
      expect(res.headers['content-type']).toContain('spreadsheet');
    });
  }

  test('GET /api/import-export/export/brands without auth returns 401', async ({ request }) => {
    const res = await new ApiDriver(request, '').get('/api/import-export/export/brands');
    expect(res.status).toBe(401);
  });

  test('GET /api/import-export/export/brands with cashier returns 403', async () => {
    const res = await cashierApi.get('/api/import-export/export/brands');
    expect(res.status).toBe(403);
  });

  test('GET /api/import-export/export/unknown-module returns 400', async () => {
    const res = await api.get('/api/import-export/export/unknown-module');
    expect(res.status).toBe(400);
  });
});

// ============================================================================
// Import — Brands
// ============================================================================
test.describe('Import/Export Framework — Import Brands', () => {
  let api: ApiDriver;
  test.beforeEach(async ({ request }) => {
    api = await apiAs(request, 'superadmin');
  });

  test('POST /api/import-export/preview/brands previews new brands', async () => {
    const name = uniqueName('Brand');
    const csvContent = `Name,Description,IsActive\n${name},E2E test import,true\n`;
    const res = await api.multipart('/api/import-export/preview/brands', {
      name: 'brands.csv',
      mimeType: 'text/csv',
      buffer: Buffer.from(csvContent),
    });
    expect(res.ok).toBeTruthy();
    expect(res.body.module).toBe('brands');
    expect(res.body.total_rows).toBe(1);
    expect(res.body.insert_count).toBe(1);
    expect(res.body.token).toBeTruthy();
  });

  test('POST /api/import-export/confirm/brands confirms import', async () => {
    const name = uniqueName('Brand');
    const csvContent = `Name,Description,IsActive\n${name},E2E test confirm,true\n`;

    // Step 1: preview
    const previewRes = await api.multipart('/api/import-export/preview/brands', {
      name: 'brands.csv',
      mimeType: 'text/csv',
      buffer: Buffer.from(csvContent),
    });
    expect(previewRes.ok).toBeTruthy();
    const preview = previewRes.body;

    // Step 2: confirm
    const confirmRes = await api.post(`/api/import-export/confirm/brands?token=${preview.token}`, {});
    expect(confirmRes.ok).toBeTruthy();
    const confirmBody = confirmRes.body;
    expect(confirmBody.job_id).toBeGreaterThan(0);
    const jobId = confirmBody.job_id;

    // Step 3: poll progress until completed
    let status = confirmBody.status;
    const maxPolls = 30;
    for (let i = 0; i < maxPolls; i++) {
      if (status === 'completed') break;
      await new Promise(r => setTimeout(r, 500));
      const progRes = await api.get(`/api/import-export/progress/${jobId}`);
      if (progRes.ok) status = progRes.body.status;
    }
    expect(status, 'import should complete within timeout').toBe('completed');

    // Step 4: verify via progress endpoint
    const finalRes = await api.get(`/api/import-export/progress/${jobId}`);
    expect(finalRes.ok).toBeTruthy();
    expect(finalRes.body.inserted).toBeGreaterThanOrEqual(1);
  });

  test('POST /api/import-export/confirm/brands without token returns 400', async () => {
    const res = await api.post('/api/import-export/confirm/brands', {});
    expect(res.status).toBe(400);
  });
});

// ============================================================================
// Import — Categories
// ============================================================================
test.describe('Import/Export Framework — Import Categories', () => {
  let api: ApiDriver;
  let cashierApi: ApiDriver;
  test.beforeEach(async ({ request }) => {
    api = await apiAs(request, 'superadmin');
    cashierApi = await apiAs(request, 'cashier');
  });

  test('POST /api/import-export/preview/categories previews new categories', async () => {
    const name = uniqueName('Category');
    const csvContent = `Name,Description,IsActive\n${name},E2E test import,true\n`;
    const res = await api.multipart('/api/import-export/preview/categories', {
      name: 'categories.csv',
      mimeType: 'text/csv',
      buffer: Buffer.from(csvContent),
    });
    expect(res.ok).toBeTruthy();
    expect(res.body.module).toBe('categories');
    expect(res.body.total_rows).toBe(1);
    expect(res.body.insert_count).toBe(1);
    expect(res.body.token).toBeTruthy();
  });

  test('POST /api/import-export/preview/categories with cashier returns 403', async () => {
    const csvContent = `Name,Description,IsActive\nTest,Test,true\n`;
    const res = await cashierApi.multipart('/api/import-export/preview/categories', {
      name: 'categories.csv',
      mimeType: 'text/csv',
      buffer: Buffer.from(csvContent),
    });
    expect(res.status).toBe(403);
  });
});

// ============================================================================
// Import — Units of Measure
// ============================================================================
test.describe('Import/Export Framework — Import UOMs', () => {
  let api: ApiDriver;
  test.beforeEach(async ({ request }) => {
    api = await apiAs(request, 'superadmin');
  });

  test('POST /api/import-export/preview/uoms previews new UOMs', async () => {
    const code = uniqueCode();
    const name = `${code} Name`;
    const csvContent = `Code,Name,Description,IsActive\n${code},${name},E2E test import,true\n`;
    const res = await api.multipart('/api/import-export/preview/uoms', {
      name: 'uoms.csv',
      mimeType: 'text/csv',
      buffer: Buffer.from(csvContent),
    });
    expect(res.ok).toBeTruthy();
    expect(res.body.module).toBe('uoms');
    expect(res.body.total_rows).toBe(1);
    expect(res.body.insert_count).toBe(1);
    expect(res.body.token).toBeTruthy();
  });
});

// ============================================================================
// Import — Customers
// ============================================================================
test.describe('Import/Export Framework — Import Customers', () => {
  let api: ApiDriver;
  let cashierApi: ApiDriver;
  test.beforeEach(async ({ request }) => {
    api = await apiAs(request, 'superadmin');
    cashierApi = await apiAs(request, 'cashier');
  });

  test('POST /api/import-export/preview/customers previews new customers', async () => {
    const phone = uniquePhone();
    const csvContent = `Name,Phone,Email,Address,Note,IsActive\nE2E Customer,${phone},e2e.${Date.now()}@test.com,Test Address,Import test,true\n`;
    const res = await api.multipart('/api/import-export/preview/customers', {
      name: 'customers.csv',
      mimeType: 'text/csv',
      buffer: Buffer.from(csvContent),
    });
    expect(res.ok).toBeTruthy();
    expect(res.body.module).toBe('customers');
    expect(res.body.total_rows).toBe(1);
    expect(res.body.insert_count).toBe(1);
    expect(res.body.token).toBeTruthy();
  });

  test('POST /api/import-export/preview/customers with cashier returns 403', async () => {
    const csvContent = `Name,Phone,Email,Address,Note,IsActive\nTest,081111,test@test.com,,,true\n`;
    const res = await cashierApi.multipart('/api/import-export/preview/customers', {
      name: 'customers.csv',
      mimeType: 'text/csv',
      buffer: Buffer.from(csvContent),
    });
    expect(res.status).toBe(403);
  });
});

// ============================================================================
// Import — Products
// ============================================================================
test.describe('Import/Export Framework — Import Products', () => {
  let api: ApiDriver;
  let cashierApi: ApiDriver;
  test.beforeEach(async ({ request }) => {
    api = await apiAs(request, 'superadmin');
    cashierApi = await apiAs(request, 'cashier');
  });

  test('POST /api/import-export/preview/products previews new products', async () => {
    const sku = `SKU${Date.now()}`;
    const csvContent = `SKU,Name,Barcode,Category,Brand,Price,Cost,Stock,Status,UnitOfMeasure,WeightGrams,Description\n${sku},E2E Product,,General,General,15000,10000,10,active,PCS,250,E2E test product\n`;
    const res = await api.multipart('/api/import-export/preview/products', {
      name: 'products.csv',
      mimeType: 'text/csv',
      buffer: Buffer.from(csvContent),
    });
    expect(res.ok).toBeTruthy();
    expect(res.body.module).toBe('products');
    expect(res.body.token).toBeTruthy();
  });

  test('POST /api/import-export/preview/products invalid CSV returns validation errors', async () => {
    const csvContent = `SKU,Name,Barcode,Category,Brand,Price,Cost,Stock,Status,UnitOfMeasure,WeightGrams,Description\n,Incomplete Product,,,,,,,,,,\n`;
    const res = await api.multipart('/api/import-export/preview/products', {
      name: 'products.csv',
      mimeType: 'text/csv',
      buffer: Buffer.from(csvContent),
    });
    expect(res.ok).toBeTruthy();
    expect(res.body.error_count).toBeGreaterThanOrEqual(1);
  });

  test('POST /api/import-export/preview/products without auth returns 401', async ({ request }) => {
    const csvContent = 'SKU,Name\nTEST,Test\n';
    const res = await new ApiDriver(request, '').multipart('/api/import-export/preview/products', {
      name: 'products.csv',
      mimeType: 'text/csv',
      buffer: Buffer.from(csvContent),
    });
    expect(res.status).toBe(401);
  });
});

// ============================================================================
// XLSX Roundtrip
// ============================================================================
test.describe('Import/Export Framework — XLSX Roundtrip', () => {
  let api: ApiDriver;
  test.beforeEach(async ({ request }) => {
    api = await apiAs(request, 'superadmin');
  });

  test('Categories XLSX: preview → confirm → verify', async () => {
    const name = uniqueName('CatXLSX');
    await xlsxRoundtrip(
      api,
      'categories',
      ['Name', 'Description', 'Active'],
      [[name, 'XLSX roundtrip test', true]],
      `/api/categories/manage?search=${encodeURIComponent(name)}`,
      'name',
      name,
    );
  });

  test('Brands XLSX: preview → confirm → verify', async () => {
    const name = uniqueName('BrandXLSX');
    console.log('Testing brand:', name);
    await xlsxRoundtrip(
      api,
      'brands',
      ['Name', 'Description', 'IsActive'],
      [[name, 'XLSX roundtrip test', true]],
      '/api/brands',
      'name',
      name,
    );
  });

  test('Products XLSX: preview → confirm → verify', async () => {
    const sku = `SKUXLSX${Date.now()}`;
    const catName = uniqueName('CatProdXLSX');
    const brandName = uniqueName('BrandProdXLSX');

    const catRes = await api.post('/api/categories', { name: catName, description: 'Product XLSX test category' });
    expect(catRes.ok).toBeTruthy();
    const brandRes = await api.post('/api/brands', { name: brandName, description: 'Product XLSX test brand' });
    expect(brandRes.ok).toBeTruthy();

    await xlsxRoundtrip(
      api,
      'products',
      ['SKU', 'Product Name', 'Barcode', 'Category', 'Brand', 'Price', 'Cost', 'Status', 'Unit of Measure'],
      [[sku, 'XLSX Product', '', catName, brandName, 15000, 10000, 'active', 'pcs']],
      `/api/products?limit=200&search=${encodeURIComponent(sku)}`,
      'sku',
      sku,
    );
  });
});
