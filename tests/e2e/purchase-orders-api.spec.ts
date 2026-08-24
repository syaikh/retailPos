import { test, expect } from './fixtures';
import { apiAs, ApiDriver } from './api-driver';

/**
 * Purchase Order + Goods Receipt behaviour, driven at the API layer. This is a
 * straight port of the previously browser-filed purchase-orders-flow.spec.ts
 * (which was already 100% API-driven, just misnamed). It asserts the data
 * contract: draft → confirm → goods receipt → fully_received, the stock delta,
 * and the auth/validation matrix. The genuine UI (create form, confirm/receive
 * dialogs, dropdown z-stacking, websocket bell) stays in the *-ui / *-dropdown /
 * *-notification browser specs.
 */
const data = (body: any) => (body && body.data !== undefined ? body.data : body);
const firstOf = (body: any) => (Array.isArray(data(body)) ? data(body)[0] : data(body));

test.describe('Purchase Orders & Goods Receipts (API driver)', () => {
  test.describe('Happy Path', () => {
    let poId = 0;
    let poItemId = 0;
    let initialStock = 0;
    let storeId = 0;
    let productId = 0;

    test.beforeAll(async ({ request }) => {
      const api = await apiAs(request, 'superadmin');
      const supplier = firstOf((await api.get('/api/suppliers?limit=1')).body);
      const product = firstOf((await api.get('/api/products?limit=1')).body);
      const store = firstOf((await api.get('/api/stores/active')).body);
      expect(supplier?.id).toBeTruthy();
      expect(product?.id).toBeTruthy();
      expect(store?.id).toBeTruthy();
      storeId = store.id;
      productId = product.id;
    });

    test('1. POST /api/purchase-orders - create draft PO', async ({ request }) => {
      const api = await apiAs(request, 'superadmin');
      const unitCost = 1000;
      const qtyOrdered = 10;
      const expectedDate = new Date(Date.now() + 7 * 86400000).toISOString().split('T')[0];

      const res = await api.post('/api/purchase-orders', {
        supplier_id: 1,
        store_id: storeId,
        expected_date: expectedDate,
        items: [{ product_id: productId, qty_ordered: qtyOrdered, unit_cost: unitCost }],
      });
      expect(res.status).toBe(201);
      const po = data(res.body);
      poId = po.id;
      expect(po.po_number).toBeTruthy();
      expect(po.status).toBe('draft');
      expect(po.subtotal).toBeGreaterThan(0);
      expect(po.grand_total).toBeGreaterThan(0);
    });

    test('2. POST /api/purchase-orders/:id/confirm - confirm PO', async ({ request }) => {
      const api = await apiAs(request, 'superadmin');
      const res = await api.post(`/api/purchase-orders/${poId}/confirm`, {});
      expect(res.ok).toBeTruthy();
      expect(data(res.body).status).toBe('confirmed');
    });

    test('3. GET /api/purchase-orders/:id - verify PO details', async ({ request }) => {
      const api = await apiAs(request, 'superadmin');
      const res = await api.get(`/api/purchase-orders/${poId}`);
      expect(res.ok).toBeTruthy();
      const po = data(res.body);
      expect(po.status).toBe('confirmed');
      expect(po.items.length).toBe(1);
      expect(po.items[0].qty_received).toBe(0);
      expect(po.items[0].unit_cost).toBeGreaterThan(0);
      poItemId = po.items[0].id;
    });

    test('4. GET /api/products/:id - capture initial stock', async ({ request }) => {
      const api = await apiAs(request, 'superadmin');
      const res = await api.get(`/api/products/${productId}`);
      expect(res.ok).toBeTruthy();
      initialStock = data(res.body).stock;
      expect(initialStock).toBeGreaterThanOrEqual(0);
    });

    test('5. POST /api/goods-receipts - full goods receipt', async ({ request }) => {
      const api = await apiAs(request, 'superadmin');
      const res = await api.post('/api/goods-receipts', {
        purchase_order_id: poId,
        store_id: storeId,
        items: [{ purchase_order_item_id: poItemId, qty_good: 10, qty_damaged: 0 }],
      });
      expect(res.ok, `GR failed: ${res.status}`).toBeTruthy();
      const gr = data(res.body);
      expect(gr.gr_number).toBeTruthy();
      expect(gr.delivery_order_number).toMatch(/^DO-\d{4}-\d{6}$/);
      expect(gr.purchase_order_id).toBe(poId);
      expect(gr.items.length).toBe(1);
      expect(gr.items[0].qty_good).toBe(10);
    });

    test('6. GET /api/purchase-orders/:id - fully_received', async ({ request }) => {
      const api = await apiAs(request, 'superadmin');
      const res = await api.get(`/api/purchase-orders/${poId}`);
      expect(res.ok).toBeTruthy();
      const po = data(res.body);
      expect(po.status).toBe('fully_received');
      expect(po.items[0].qty_received).toBe(10);
    });

    test('7. GET /api/purchase-orders/:id/receipts - list GRs', async ({ request }) => {
      const api = await apiAs(request, 'superadmin');
      const res = await api.get(`/api/purchase-orders/${poId}/receipts`);
      expect(res.ok).toBeTruthy();
      const receipts = data(res.body);
      expect(receipts.length).toBe(1);
      expect(receipts[0].purchase_order_id).toBe(poId);
    });

    test('8. GET /api/products/:id - stock increased', async ({ request }) => {
      const api = await apiAs(request, 'superadmin');
      const res = await api.get(`/api/products/${productId}`);
      expect(res.ok).toBeTruthy();
      expect(data(res.body).stock).toBe(initialStock + 10);
    });
  });

  test.describe('Purchase Orders - Auth & Validation', () => {
    test('POST without auth returns 401', async ({ request }) => {
      const api = new ApiDriver(request, '');
      const res = await api.post('/api/purchase-orders', {
        supplier_id: 1,
        items: [{ product_id: 1, qty_ordered: 1, unit_cost: 1000 }],
      });
      expect(res.status).toBe(401);
    });

    test('POST with cashier (no purchase.create) returns 403', async ({ request }) => {
      const api = await apiAs(request, 'cashier');
      const res = await api.post('/api/purchase-orders', {
        supplier_id: 1,
        items: [{ product_id: 1, qty_ordered: 1, unit_cost: 1000 }],
      });
      expect(res.status).toBe(403);
    });

    test('confirm on non-existent PO returns 404', async ({ request }) => {
      const api = await apiAs(request, 'superadmin');
      const res = await api.post('/api/purchase-orders/999999/confirm', {});
      expect(res.status).toBe(404);
    });

    test('confirm without auth returns 401', async ({ request }) => {
      const api = new ApiDriver(request, '');
      const res = await api.post('/api/purchase-orders/1/confirm', {});
      expect(res.status).toBe(401);
    });
  });

  test.describe('Goods Receipts - Auth & Validation', () => {
    test('POST without auth returns 401', async ({ request }) => {
      const api = new ApiDriver(request, '');
      const res = await api.post('/api/goods-receipts', { purchase_order_id: 1, items: [] });
      expect(res.status).toBe(401);
    });

    test('POST with cashier (no purchase.receive) returns 403', async ({ request }) => {
      const api = await apiAs(request, 'cashier');
      const res = await api.post('/api/goods-receipts', {
        purchase_order_id: 1,
        items: [{ purchase_order_item_id: 1, qty_good: 1 }],
      });
      expect(res.status).toBe(403);
    });

    test('POST for draft PO returns 400', async ({ request }) => {
      const api = await apiAs(request, 'superadmin');
      const supplier = firstOf((await api.get('/api/suppliers?limit=1')).body);
      const product = firstOf((await api.get('/api/products?limit=1')).body);
      const store = firstOf((await api.get('/api/stores/active')).body);

      const poRes = await api.post('/api/purchase-orders', {
        supplier_id: supplier.id,
        store_id: store.id,
        expected_date: new Date(Date.now() + 7 * 86400000).toISOString().split('T')[0],
        items: [{ product_id: product.id, qty_ordered: 5, unit_cost: 1000 }],
      });
      expect(poRes.ok).toBeTruthy();
      const po = data(poRes.body);

      const detail = data((await api.get(`/api/purchase-orders/${po.id}`)).body);
      const grRes = await api.post('/api/goods-receipts', {
        purchase_order_id: po.id,
        store_id: store.id,
        items: [{ purchase_order_item_id: detail.items[0].id, qty_good: 5 }],
      });
      expect(grRes.status).toBe(400);
      const msg = grRes.body?.error?.message || grRes.body?.message || '';
      expect(msg).toContain('not in a receivable status');
    });
  });

  test.describe('Purchase Orders - Delete Draft', () => {
    let storeId = 0;

    test.beforeAll(async ({ request }) => {
      const api = await apiAs(request, 'superadmin');
      storeId = firstOf((await api.get('/api/stores/active')).body).id;
    });

    async function makeDraft(api: any) {
      const supplier = firstOf((await api.get('/api/suppliers?limit=1')).body);
      const product = firstOf((await api.get('/api/products?limit=1')).body);
      const res = await api.post('/api/purchase-orders', {
        supplier_id: supplier.id,
        store_id: storeId,
        expected_date: new Date(Date.now() + 7 * 86400000).toISOString().split('T')[0],
        items: [{ product_id: product.id, qty_ordered: 1, unit_cost: 1000 }],
      });
      expect(res.ok).toBeTruthy();
      return data(res.body).id;
    }

    test('DELETE deletes a draft PO', async ({ request }) => {
      const api = await apiAs(request, 'superadmin');
      const id = await makeDraft(api);
      const del = await api.del(`/api/purchase-orders/${id}`);
      expect(del.ok, `delete failed: ${del.status}`).toBeTruthy();
    });

    test('DELETE on confirmed PO returns 409', async ({ request }) => {
      const api = await apiAs(request, 'superadmin');
      const id = await makeDraft(api);
      const confirm = await api.post(`/api/purchase-orders/${id}/confirm`, {});
      expect(confirm.ok).toBeTruthy();
      const del = await api.del(`/api/purchase-orders/${id}`);
      expect(del.status).toBe(409);
    });

    test('DELETE without auth returns 401', async ({ request }) => {
      const api = new ApiDriver(request, '');
      const res = await api.del('/api/purchase-orders/1');
      expect(res.status).toBe(401);
    });
  });
});
