import { test, expect } from './fixtures';
import { apiAs, ApiDriver } from './api-driver';

/**
 * Stock-adjustment data contract (POST /api/inventory/adjust) driven at the API
 * layer. This is the API half of inventory-adjust-stock.spec.ts — the modal
 * open / cancel / role-visibility / UI-validation tests stay in that browser
 * spec as genuine UI coverage.
 */
const data = (b: any) => (b && b.data !== undefined ? b.data : b);

/**
 * Get any product and boost its stock to a known level before testing.
 * Eliminates fragile assumptions about seeded stock levels.
 */
async function getOrBoostProduct(api: ApiDriver, { request, token }: { request: any; token: string }): Promise<{ id: number; stock: number }> {
  const res = await api.get('/api/products?limit=1');
  expect(res.ok).toBeTruthy();
  const products = data(res.body) || [];
  expect(products.length, 'need at least 1 product').toBeGreaterThan(0);
  const prod = products[0] as { id: number; stock: number };

  // Boost stock to a known safe level
  await api.post('/api/inventory/adjust', {
    product_id: prod.id,
    quantity_change: 200,
    notes: 'E2E ensure stock for inventory-adjust tests',
  });

  // Re-fetch to confirm
  const after = data((await api.get(`/api/products/${prod.id}`)).body);
  return { id: prod.id, stock: after.stock };
}

test.describe('Inventory Adjust API', () => {
  test('positive adjustment increases stock by the delta', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const prod = await getOrBoostProduct(api, { request, token: '' });
    const delta = 15;

    const res = await api.post('/api/inventory/adjust', {
      product_id: prod.id,
      quantity_change: delta,
      notes: `E2E +${delta}`,
    });
    expect(res.ok, `adjust failed: ${res.status}: ${JSON.stringify(res.body)}`).toBeTruthy();

    const after = data((await api.get(`/api/products/${prod.id}`)).body);
    expect(after.stock).toBe(prod.stock + delta);

    // revert
    await api.post('/api/inventory/adjust', {
      product_id: prod.id,
      quantity_change: -delta,
      notes: 'E2E revert',
    });
    const reverted = data((await api.get(`/api/products/${prod.id}`)).body);
    expect(reverted.stock).toBe(prod.stock);
  });

  test('negative adjustment decreases stock by the delta', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const prod = await getOrBoostProduct(api, { request, token: '' });
    const delta = -5;

    const res = await api.post('/api/inventory/adjust', {
      product_id: prod.id,
      quantity_change: delta,
      notes: `E2E ${delta}`,
    });
    expect(res.ok).toBeTruthy();

    const after = data((await api.get(`/api/products/${prod.id}`)).body);
    expect(after.stock).toBe(prod.stock + delta);

    await api.post('/api/inventory/adjust', {
      product_id: prod.id,
      quantity_change: -delta,
      notes: 'E2E revert',
    });
  });

  test('rejects zero quantity change with 400', async ({ request }) => {
    const api = await apiAs(request, 'superadmin');
    const prod = await getOrBoostProduct(api, { request, token: '' });
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
    const prod = await getOrBoostProduct(api, { request, token: '' });
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
    const prod = await getOrBoostProduct(api, { request, token: '' });
    const res = await api.post('/api/inventory/adjust', {
      product_id: prod.id,
      quantity_change: 5,
      notes: 'hack',
    });
    expect(res.status).toBe(403);
  });
});
