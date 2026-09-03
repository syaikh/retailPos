import { test, expect } from './fixtures';
import { apiAs, ApiDriver } from './api-driver';

/**
 * Parked-sale (Hold & Recall) data contract driven at the API layer. This is the
 * API half of hold-recall.spec.ts — the browser F6/F7 park-recall-checkout flow
 * stays in hold-recall.spec.ts as genuine UI coverage.
 */
const data = (b: any) => (b && b.data !== undefined ? b.data : b);

test.describe('Hold & Recall API Flow', () => {
  let api: ApiDriver;
  let productA: { id: number; price: number };
  let productB: { id: number; price: number };

  test.beforeEach(async ({ request }) => {
    api = await apiAs(request, 'superadmin');

    const prodRes = await api.get('/api/products?limit=2');
    const prodBody = data(prodRes.body);
    if (!prodBody || prodBody.length < 2) {
      throw new Error('Need at least 2 products in DB for hold-recall tests');
    }
    productA = { id: prodBody[0].id, price: prodBody[0].price };
    productB = { id: prodBody[1].id, price: prodBody[1].price };

    await api.post('/api/inventory/adjust', {
      product_id: productA.id,
      quantity_change: 500,
      notes: 'E2E stock boost for hold-recall tests',
    });
    await api.post('/api/inventory/adjust', {
      product_id: productB.id,
      quantity_change: 500,
      notes: 'E2E stock boost for hold-recall tests',
    });
  });

  async function park(product: { id: number; price: number } = productA, qty = 1, extra: any = {}) {
    const res = await api.post('/api/sales/parked', {
      items: [{ product_id: product.id, quantity: qty }],
      payment_method: 'CASH',
      ...extra,
    });
    return res;
  }

  test('should park → recall → complete with parked_sale_id, verify consumption', async () => {
    const qty = 1;

    const parkRes = await api.post('/api/sales/parked', {
      items: [{ product_id: productA.id, quantity: qty }],
      payment_method: 'CASH',
    });
    expect(parkRes.ok, `park failed: ${parkRes.status}`).toBeTruthy();
    const parkedSale = data(parkRes.body);
    expect(parkedSale.status).toBe('parked');
    expect(parkedSale.id).toBeGreaterThan(0);

    const listRes1 = await api.get('/api/sales/parked');
    expect(listRes1.ok).toBeTruthy();
    const parkedList = data(listRes1.body);
    expect(Array.isArray(parkedList)).toBeTruthy();
    expect(parkedList.some((s: any) => s.id === parkedSale.id)).toBeTruthy();

    const recallRes = await api.post(`/api/sales/parked/${parkedSale.id}/recall`);
    expect(recallRes.ok).toBeTruthy();
    expect(data(recallRes.body).status).toBe('recalled');
    expect(data(recallRes.body).items.length).toBe(qty);

    const completeRes = await api.post('/api/sales', {
      payment_method: 'CASH',
      items: [{ product_id: productA.id, quantity: qty }],
      parked_sale_id: parkedSale.id,
    });
    expect(completeRes.ok).toBeTruthy();
    expect(data(completeRes.body).status).toBe('completed');

    const listRes2 = await api.get('/api/sales/parked');
    const listAfterRecall = data(listRes2.body);
    expect(Array.isArray(listAfterRecall)).toBeTruthy();
    expect(listAfterRecall.some((s: any) => s.id === parkedSale.id)).toBeFalsy();

    const reRecallRes = await api.post(`/api/sales/parked/${parkedSale.id}/recall`);
    expect(reRecallRes.ok).toBeFalsy();
    expect(reRecallRes.status).toBe(404);
  });

  test('should park → recall → park again cancels previous recalled sale', async () => {
    const parkResA = await api.post('/api/sales/parked', {
      items: [{ product_id: productA.id, quantity: 1 }],
      payment_method: 'CASH',
    });
    expect(parkResA.ok).toBeTruthy();
    const saleA = data(parkResA.body);

    await api.post(`/api/sales/parked/${saleA.id}/recall`);

    const parkResB = await api.post('/api/sales/parked', {
      items: [{ product_id: productB.id, quantity: 1 }],
      payment_method: 'CASH',
      recalled_sale_id: saleA.id,
    });
    expect(parkResB.ok).toBeTruthy();
    const saleB = data(parkResB.body);
    expect(saleB.status).toBe('parked');
    expect(saleB.id).not.toBe(saleA.id);

    const listRes = await api.get('/api/sales/parked');
    const list = data(listRes.body);
    expect(Array.isArray(list)).toBeTruthy();
    expect(list.some((s: any) => s.id === saleA.id)).toBeFalsy();
    expect(list.some((s: any) => s.id === saleB.id)).toBeTruthy();
  });

  test('should recall already-recalled sale (re-recall without consumption)', async () => {
    const res = await park(productA);
    expect(res.ok).toBeTruthy();
    const sale = data(res.body);

    const recall1 = await api.post(`/api/sales/parked/${sale.id}/recall`);
    expect(recall1.ok).toBeTruthy();
    expect(data(recall1.body).status).toBe('recalled');

    const recall2 = await api.post(`/api/sales/parked/${sale.id}/recall`);
    expect(recall2.ok).toBeTruthy();
    expect(data(recall2.body).status).toBe('recalled');
  });

  test('should fail to complete with parked_sale_id without prior recall (409)', async () => {
    const res = await park(productA);
    expect(res.ok).toBeTruthy();
    const sale = data(res.body);

    const completeRes = await api.post('/api/sales', {
      payment_method: 'CASH',
      items: [{ product_id: productA.id, quantity: 1 }],
      parked_sale_id: sale.id,
    });
    expect(completeRes.ok).toBeFalsy();
    expect(completeRes.status).toBe(409);
    const body = completeRes.body;
    expect(body.error?.message || body.error).toContain('checked out or cancelled');
  });

  test('should fail to recall a cancelled sale (404)', async () => {
    const res = await park(productA);
    expect(res.ok).toBeTruthy();
    const sale = data(res.body);

    const delRes = await api.del(`/api/sales/parked/${sale.id}`);
    expect(delRes.ok).toBeTruthy();

    const recallRes = await api.post(`/api/sales/parked/${sale.id}/recall`);
    expect(recallRes.ok).toBeFalsy();
    expect(recallRes.status).toBe(404);
  });

  test('should fail to recall non-existent sale (404)', async () => {
    const recallRes = await api.post('/api/sales/parked/99999999/recall');
    expect(recallRes.ok).toBeFalsy();
    expect(recallRes.status).toBe(404);
  });

  test('should cancel (DELETE) a parked sale', async () => {
    const res = await park(productA);
    expect(res.ok).toBeTruthy();
    const sale = data(res.body);

    const delRes = await api.del(`/api/sales/parked/${sale.id}`);
    expect(delRes.ok).toBeTruthy();

    const listRes = await api.get('/api/sales/parked');
    expect(data(listRes.body).some((s: any) => s.id === sale.id)).toBeFalsy();
  });

  test('should cancel (DELETE) a recalled sale', async () => {
    const res = await park(productA);
    expect(res.ok).toBeTruthy();
    const sale = data(res.body);

    await api.post(`/api/sales/parked/${sale.id}/recall`);

    const delRes = await api.del(`/api/sales/parked/${sale.id}`);
    expect(delRes.ok).toBeTruthy();

    const listRes = await api.get('/api/sales/parked');
    expect(data(listRes.body).some((s: any) => s.id === sale.id)).toBeFalsy();
  });

  test('should fail to double-cancel a sale (404)', async () => {
    const res = await park(productA);
    expect(res.ok).toBeTruthy();
    const sale = data(res.body);

    await api.del(`/api/sales/parked/${sale.id}`);

    const del2 = await api.del(`/api/sales/parked/${sale.id}`);
    expect(del2.ok).toBeFalsy();
    expect(del2.status).toBe(404);
  });

  test('should reject park with empty items (400)', async () => {
    const res = await api.post('/api/sales/parked', { items: [], payment_method: 'CASH' });
    expect(res.ok).toBeFalsy();
    expect(res.status).toBe(400);
  });

  test('should list parked sales and return valid array', async () => {
    const listRes = await api.get('/api/sales/parked');
    expect(listRes.ok).toBeTruthy();
    expect(Array.isArray(data(listRes.body))).toBeTruthy();
  });

  test('should get parked sale by ID returns full details', async () => {
    const res = await park(productA);
    expect(res.ok).toBeTruthy();
    const sale = data(res.body);

    const getRes = await api.get(`/api/sales/parked/${sale.id}`);
    expect(getRes.ok).toBeTruthy();
    const body = data(getRes.body);
    expect(body.id).toBe(sale.id);
    expect(body.status).toBe('parked');
    expect(body.items).toBeDefined();
    expect(body.items.length).toBe(1);
    expect(body.items[0].product_id).toBe(productA.id);
  });

  test('should reject park without items field (400)', async () => {
    const res = await api.post('/api/sales/parked', { payment_method: 'CASH' });
    expect(res.ok).toBeFalsy();
    expect(res.status).toBe(400);
  });

  test('should park after recall without recalled_sale_id leaves recalled sale unchanged', async () => {
    const resA = await park(productA);
    expect(resA.ok).toBeTruthy();
    const saleA = data(resA.body);

    await api.post(`/api/sales/parked/${saleA.id}/recall`);

    const resB = await park(productB);
    expect(resB.ok).toBeTruthy();

    const listRes = await api.get('/api/sales/parked');
    const list = data(listRes.body);
    expect(list.some((s: any) => s.id === saleA.id)).toBeTruthy();
  });
});
