import { test, expect } from './fixtures';
import { apiAs, ApiDriver } from './api-driver';

/**
 * Stock-adjustment data contract (POST /api/inventory/adjust) driven at the API
 * layer. This is the API half of inventory-adjust-stock.spec.ts — the modal
 * open / cancel / role-visibility / UI-validation tests stay in that browser
 * spec as genuine UI coverage.
 */
const data = (b: any) => (b && b.data !== undefined ? b.data : b);

async function firstProductWithStock(api: ApiDriver, minStock = 6) {
  const res = await api.get('/api/products?limit=20&offset=0');
  expect(res.ok).toBeTruthy();
  const products = data(res.body) || [];
  const target = products.find((p: any) => (p.stock ?? 0) >= minStock);
  expect(target, 'no product with sufficient stock').toBeTruthy();
  return target as { id: number; stock: number };
}

test.describe('Inventory Adjust API', () => {
  test('positive adjustment increases stock by the delta', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const prod = await firstProductWithStock(api, 6);
    const before = prod.stock;
    const delta = 15;

    const res = await api.post('/api/inventory/adjust', {
      product_id: prod.id,
      quantity_change: delta,
      notes: `E2E +${delta}`,
    });
    expect(res.ok, `adjust failed: ${res.status}: ${JSON.stringify(res.body)}`).toBeTruthy();

    const after = data((await api.get(`/api/products/${prod.id}`)).body);
    expect(after.stock).toBe(before + delta);

    // revert
    await api.post('/api/inventory/adjust', {
      product_id: prod.id,
      quantity_change: -delta,
      notes: 'E2E revert',
    });
    const reverted = data((await api.get(`/api/products/${prod.id}`)).body);
    expect(reverted.stock).toBe(before);
  });

  test('negative adjustment decreases stock by the delta', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const prod = await firstProductWithStock(api, 10);
    const before = prod.stock;
    const delta = -5;

    const res = await api.post('/api/inventory/adjust', {
      product_id: prod.id,
      quantity_change: delta,
      notes: `E2E ${delta}`,
    });
    expect(res.ok).toBeTruthy();

    const after = data((await api.get(`/api/products/${prod.id}`)).body);
    expect(after.stock).toBe(before + delta);

    await api.post('/api/inventory/adjust', {
      product_id: prod.id,
      quantity_change: -delta,
      notes: 'E2E revert',
    });
  });

  test('rejects zero quantity change with 400', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const prod = await firstProductWithStock(api, 1);
    const res = await api.post('/api/inventory/adjust', {
      product_id: prod.id,
      quantity_change: 0,
      notes: 'zero',
    });
    expect(res.status).toBe(400);
    expect(String(data(res.body).error ?? res.body?.error ?? '')).toContain('quantity change must not be zero');
  });

  test('rejects missing notes with 400', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const prod = await firstProductWithStock(api, 1);
    const res = await api.post('/api/inventory/adjust', {
      product_id: prod.id,
      quantity_change: 10,
      notes: '',
    });
    expect(res.status).toBe(400);
    expect(String(data(res.body).error ?? res.body?.error ?? '')).toContain('notes are required');
  });

  test('cashier (no inventory.adjust) returns 403', async ({ request }) => {
    const api = await apiAs(request, 'cashier');
    const prod = await firstProductWithStock(api, 1);
    const res = await api.post('/api/inventory/adjust', {
      product_id: prod.id,
      quantity_change: 5,
      notes: 'hack',
    });
    expect(res.status).toBe(403);
  });
});
