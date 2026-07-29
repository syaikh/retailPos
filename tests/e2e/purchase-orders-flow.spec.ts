import { test, expect } from '@playwright/test';
import { TEST_USERS, API_BASE, authHeader, getToken as cachedGetToken } from './fixtures';

const getToken = cachedGetToken;
let poId: number;
let poItemId: number;

test.describe('Purchase Orders & Goods Receipts - Happy Path', () => {
  let headers: Record<string, string>;
  let supplier: { id: number };
  let product: { id: number; price: number };
  let store: { id: number };

  test.beforeAll(async ({ request }) => {
    const token = await getToken(request);
    headers = authHeader(token);

    const supRes = await request.get(`${API_BASE}/api/suppliers?limit=1`, { headers });
    expect(supRes.ok()).toBeTruthy();
    const supBody = await supRes.json();
    const suppliers = supBody.data || supBody;
    expect(suppliers.length).toBeGreaterThan(0);
    supplier = suppliers[0];

    const prodRes = await request.get(`${API_BASE}/api/products?limit=1`, { headers });
    expect(prodRes.ok()).toBeTruthy();
    const prodBody = await prodRes.json();
    const products = prodBody.data || [];
    expect(products.length).toBeGreaterThan(0);
    product = products[0];

    const storeRes = await request.get(`${API_BASE}/api/stores/active`, { headers });
    expect(storeRes.ok()).toBeTruthy();
    const storeBody = await storeRes.json();
    const stores = storeBody.data || storeBody;
    expect(stores.length).toBeGreaterThan(0);
    store = stores[0];
  });

  test('1. POST /api/purchase-orders - create draft PO', async ({ request }) => {
    const unitCost = Math.round(product.price * 0.65);
    const qtyOrdered = 10;
    const expectedDate = new Date(Date.now() + 7 * 86400000).toISOString().split('T')[0];

    const res = await request.post(`${API_BASE}/api/purchase-orders`, {
      headers,
      data: {
        supplier_id: supplier.id,
        store_id: store.id,
        expected_date: expectedDate,
        items: [
          {
            product_id: product.id,
            qty_ordered: qtyOrdered,
            unit_cost: unitCost,
          },
        ],
      },
    });
    expect(res.ok(), `create PO failed: ${res.status()} ${await res.text()}`).toBeTruthy();
    const body = await res.json();
    const po = body.data || body;
    poId = po.id;
    expect(po.po_number).toBeTruthy();
    expect(po.status).toBe('draft');
    expect(po.supplier_id).toBe(supplier.id);
    expect(po.subtotal).toBeGreaterThan(0);
    expect(po.grand_total).toBeGreaterThan(0);
  });

  test('2. POST /api/purchase-orders/:id/confirm - confirm PO', async ({ request }) => {
    expect(poId).toBeGreaterThan(0);
    const res = await request.post(`${API_BASE}/api/purchase-orders/${poId}/confirm`, {
      headers,
    });
    expect(res.ok(), `confirm PO failed: ${res.status()} ${await res.text()}`).toBeTruthy();
    const body = await res.json();
    const po = body.data || body;
    expect(po.status).toBe('confirmed');
  });

  test('3. GET /api/purchase-orders/:id - verify PO details', async ({ request }) => {
    expect(poId).toBeGreaterThan(0);
    const res = await request.get(`${API_BASE}/api/purchase-orders/${poId}`, {
      headers,
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    const po = body.data || body;
    expect(po.status).toBe('confirmed');
    expect(po.items.length).toBe(1);
    expect(po.items[0].qty_received).toBe(0);
    expect(po.items[0].unit_cost).toBeGreaterThan(0);
    expect(po.items[0].subtotal).toBeGreaterThan(0);
    expect(po.items[0].discount_amount).toBeGreaterThanOrEqual(0);
    expect(po.subtotal).toBeGreaterThan(0);
    expect(po.discount_amount).toBeGreaterThanOrEqual(0);
    expect(po.grand_total).toBeGreaterThan(0);
    poItemId = po.items[0].id;
  });

  let initialStock: number;

  test('4. GET /api/products/:id - capture initial stock', async ({ request }) => {
    const res = await request.get(`${API_BASE}/api/products/${product.id}`, { headers });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    const prod = body.data || body;
    initialStock = prod.stock;
    expect(initialStock).toBeGreaterThanOrEqual(0);
  });

  test('5. POST /api/goods-receipts - full goods receipt', async ({ request }) => {
    expect(poId).toBeGreaterThan(0);
    expect(poItemId).toBeGreaterThan(0);
    const res = await request.post(`${API_BASE}/api/goods-receipts`, {
      headers,
      data: {
        purchase_order_id: poId,
        delivery_order_number: `DO-E2E-${Date.now()}`,
        items: [
          {
            purchase_order_item_id: poItemId,
            qty_good: 10,
            qty_damaged: 0,
          },
        ],
      },
    });
    expect(res.ok(), `create GR failed: ${res.status()} ${await res.text()}`).toBeTruthy();
    const body = await res.json();
    const gr = body.data || body;
    expect(gr.gr_number).toBeTruthy();
    expect(gr.purchase_order_id).toBe(poId);
    expect(gr.items).toBeDefined();
    expect(gr.items.length).toBe(1);
    expect(gr.items[0].qty_good).toBe(10);
    expect(gr.items[0].qty_damaged).toBe(0);
  });

  test('6. GET /api/purchase-orders/:id - verify PO marked as fully_received', async ({ request }) => {
    expect(poId).toBeGreaterThan(0);
    const res = await request.get(`${API_BASE}/api/purchase-orders/${poId}`, {
      headers,
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    const po = body.data || body;
    expect(po.status).toBe('fully_received');
    expect(po.items[0].qty_received).toBe(10);
    expect(po.items[0].unit_cost).toBeGreaterThan(0);
    expect(po.items[0].subtotal).toBeGreaterThan(0);
    expect(po.items[0].discount_amount).toBeGreaterThanOrEqual(0);
    expect(po.subtotal).toBeGreaterThan(0);
    expect(po.discount_amount).toBeGreaterThanOrEqual(0);
    expect(po.grand_total).toBeGreaterThan(0);
  });

  test('7. GET /api/purchase-orders/:id/receipts - list GRs for PO', async ({ request }) => {
    expect(poId).toBeGreaterThan(0);
    const res = await request.get(`${API_BASE}/api/purchase-orders/${poId}/receipts`, {
      headers,
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    const receipts = body.data || [];
    expect(receipts.length).toBe(1);
    expect(receipts[0].purchase_order_id).toBe(poId);
    expect(receipts[0].items.length).toBeGreaterThan(0);
  });

  test('8. GET /api/products/:id - verify stock increased', async ({ request }) => {
    const res = await request.get(`${API_BASE}/api/products/${product.id}`, { headers });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    const prod = body.data || body;
    expect(prod.stock).toBe(initialStock + 10);
  });
});

test.describe('Purchase Orders - Auth & Validation', () => {
  let headers: Record<string, string>;

  test.beforeAll(async ({ request }) => {
    const token = await getToken(request);
    headers = authHeader(token);
  });

  test('POST /api/purchase-orders without auth returns 401', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/purchase-orders`, {
      data: { supplier_id: 1, items: [{ product_id: 1, qty_ordered: 1, unit_cost: 1000 }] },
    });
    expect(res.status()).toBe(401);
  });

  test('POST /api/purchase-orders with restricted role returns 403', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
    const res = await request.post(`${API_BASE}/api/purchase-orders`, {
      headers: authHeader(token),
      data: { supplier_id: 1, items: [{ product_id: 1, qty_ordered: 1, unit_cost: 1000 }] },
    });
    expect(res.status()).toBe(403);
  });

  test('POST /api/purchase-orders/:id/confirm on non-existent PO returns 404', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/purchase-orders/999999/confirm`, {
      headers,
    });
    expect(res.status()).toBe(404);
  });

  test('POST /api/purchase-orders/:id/confirm without auth returns 401', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/purchase-orders/1/confirm`);
    expect(res.status()).toBe(401);
  });
});

test.describe('Goods Receipts - Auth & Validation', () => {
  let headers: Record<string, string>;

  test.beforeAll(async ({ request }) => {
    const token = await getToken(request);
    headers = authHeader(token);
  });

  test('POST /api/goods-receipts without auth returns 401', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/goods-receipts`, {
      data: { purchase_order_id: 1, items: [] },
    });
    expect(res.status()).toBe(401);
  });

  test('POST /api/goods-receipts with restricted role returns 403', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
    const res = await request.post(`${API_BASE}/api/goods-receipts`, {
      headers: authHeader(token),
      data: { purchase_order_id: 1, items: [{ purchase_order_item_id: 1, qty_good: 1 }] },
    });
    expect(res.status()).toBe(403);
  });

  test('POST /api/goods-receipts for draft PO returns 400', async ({ request }) => {
    const unitCost = 1000;
    const qtyOrdered = 5;

    const supRes = await request.get(`${API_BASE}/api/suppliers?limit=1`, { headers });
    const supBody = await supRes.json();
    const supplier = (supBody.data || supBody)[0];

    const prodRes = await request.get(`${API_BASE}/api/products?limit=1`, { headers });
    const prodBody = await prodRes.json();
    const product = (prodBody.data || [])[0];

    const storeRes = await request.get(`${API_BASE}/api/stores/active`, { headers });
    const storeBody = await storeRes.json();
    const store = (storeBody.data || storeBody)[0];

    const poRes = await request.post(`${API_BASE}/api/purchase-orders`, {
      headers,
      data: {
        supplier_id: supplier.id,
        store_id: store.id,
        items: [{ product_id: product.id, qty_ordered: qtyOrdered, unit_cost: unitCost }],
      },
    });
    expect(poRes.ok()).toBeTruthy();
    const poBody = await poRes.json();
    const draftPO = poBody.data || poBody;

    const detailRes = await request.get(`${API_BASE}/api/purchase-orders/${draftPO.id}`, { headers });
    const detailBody = await detailRes.json();
    const detailPO = detailBody.data || detailBody;

    const grRes = await request.post(`${API_BASE}/api/goods-receipts`, {
      headers,
      data: {
        purchase_order_id: draftPO.id,
        items: [{ purchase_order_item_id: detailPO.items[0].id, qty_good: 5 }],
      },
    });
    expect(grRes.status()).toBe(400);
    const grBody = await grRes.json();
    expect(grBody.error?.message || grBody.message || '').toContain('not in a receivable status');
  });
});

test.describe('Purchase Orders - Delete Draft', () => {
  let headers: Record<string, string>;
  let store: { id: number };

  test.beforeAll(async ({ request }) => {
    const token = await getToken(request);
    headers = authHeader(token);

    const storeRes = await request.get(`${API_BASE}/api/stores/active`, { headers });
    const storeBody = await storeRes.json();
    store = (storeBody.data || storeBody)[0];
  });

  test('DELETE /api/purchase-orders/:id deletes a draft PO', async ({ request }) => {
    const supRes = await request.get(`${API_BASE}/api/suppliers?limit=1`, { headers });
    const supBody = await supRes.json();
    const supplier = (supBody.data || supBody)[0];

    const prodRes = await request.get(`${API_BASE}/api/products?limit=1`, { headers });
    const prodBody = await prodRes.json();
    const product = (prodBody.data || [])[0];

    const poRes = await request.post(`${API_BASE}/api/purchase-orders`, {
      headers,
      data: {
        supplier_id: supplier.id,
        store_id: store.id,
        items: [{ product_id: product.id, qty_ordered: 1, unit_cost: 1000 }],
      },
    });
    expect(poRes.ok()).toBeTruthy();
    const poBody = await poRes.json();
    const po = poBody.data || poBody;

    const delRes = await request.delete(`${API_BASE}/api/purchase-orders/${po.id}`, {
      headers,
    });
    expect(delRes.ok(), `delete PO failed: ${delRes.status()} ${await delRes.text()}`).toBeTruthy();
  });

  test('DELETE /api/purchase-orders/:id on confirmed PO returns 409', async ({ request }) => {
    const supRes = await request.get(`${API_BASE}/api/suppliers?limit=1`, { headers });
    const supBody = await supRes.json();
    const supplier = (supBody.data || supBody)[0];

    const prodRes = await request.get(`${API_BASE}/api/products?limit=1`, { headers });
    const prodBody = await prodRes.json();
    const product = (prodBody.data || [])[0];

    const poRes = await request.post(`${API_BASE}/api/purchase-orders`, {
      headers,
      data: {
        supplier_id: supplier.id,
        store_id: store.id,
        items: [{ product_id: product.id, qty_ordered: 1, unit_cost: 1000 }],
      },
    });
    expect(poRes.ok()).toBeTruthy();
    const poBody = await poRes.json();
    const po = poBody.data || poBody;

    const confirmRes = await request.post(`${API_BASE}/api/purchase-orders/${po.id}/confirm`, {
      headers,
    });
    expect(confirmRes.ok()).toBeTruthy();

    const delRes = await request.delete(`${API_BASE}/api/purchase-orders/${po.id}`, {
      headers,
    });
    expect(delRes.status()).toBe(409);
  });

  test('DELETE /api/purchase-orders/:id without auth returns 401', async ({ request }) => {
    const res = await request.delete(`${API_BASE}/api/purchase-orders/1`);
    expect(res.status()).toBe(401);
  });
});
