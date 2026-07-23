import { test, expect } from '@playwright/test';
import { TEST_USERS, API_BASE, loginUI, logoutUI, getToken, authHeader } from './fixtures';

test.describe('Hold & Recall UI Flow', () => {
  test.beforeEach(async ({ page }) => {
    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    await page.goto('http://localhost:5173/pos');
    await expect(page).toHaveURL(/\/pos/);
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('should park sale via F6, recall via F5, and complete checkout', async ({ page }) => {
    await page.waitForTimeout(2000);
    const addButton = page.locator('button:not([disabled])').filter({ hasText: 'Add' }).first();
    await addButton.waitFor({ state: 'visible', timeout: 10000 });
    await expect(addButton).toBeEnabled({ timeout: 5000 });
    await addButton.click();
    await page.waitForTimeout(500);
    await expect(page.locator('text=Your cart is empty')).toBeHidden({ timeout: 5000 });

    await page.keyboard.press('F6');
    await page.waitForTimeout(1000);
    await expect(page.locator('text=Your cart is empty')).toBeVisible({ timeout: 5000 });

    await page.keyboard.press('F5');
    await page.waitForTimeout(1000);
    await expect(page.locator('text=Held Sales')).toBeVisible({ timeout: 5000 });

    const recallBtn = page.locator('button[data-action="recall"]').first();
    await recallBtn.waitFor({ state: 'visible', timeout: 5000 });
    await recallBtn.click();
    await page.waitForTimeout(1000);
    await expect(page.locator('text=Your cart is empty')).toBeHidden({ timeout: 5000 });

    await page.keyboard.press('F4');
    await page.waitForTimeout(1000);

    await page.keyboard.press('F7');
    await page.waitForTimeout(500);

    const selesaiBtn = page.locator('button').filter({ hasText: 'Selesai' });
    await selesaiBtn.waitFor({ state: 'visible', timeout: 5000 });
    await expect(selesaiBtn).toBeEnabled({ timeout: 3000 });
    await selesaiBtn.click();
    await page.waitForTimeout(2000);

    await expect(page.locator('text=Your cart is empty')).toBeVisible({ timeout: 8000 });
  });
});

test.describe('Hold & Recall API Flow', () => {
  let productA: { id: number; price: number };
  let productB: { id: number; price: number };
  let auth: { token: string; headers: Record<string, string> };

  test.beforeAll(async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const headers = authHeader(token);
    auth = { token, headers };

    const prodRes = await request.get(`${API_BASE}/api/products?limit=2`, { headers });
    const prodBody = await prodRes.json();
    if (!prodBody.data || prodBody.data.length < 2) {
      throw new Error('Need at least 2 products in DB for hold-recall tests');
    }
    productA = { id: prodBody.data[0].id, price: prodBody.data[0].price };
    productB = { id: prodBody.data[1].id, price: prodBody.data[1].price };
  });

  async function park(request: any, headers: any, product: { id: number; price: number } = productA, qty = 1) {
    const res = await request.post(`${API_BASE}/api/sales/parked`, {
      headers,
      data: { items: [{ product_id: product.id, quantity: qty, subtotal: product.price * qty }], payment_method: 'CASH' },
    });
    return res;
  }

  test('should park → recall → complete with parked_sale_id, verify consumption', async ({ request }) => {
    const { headers } = auth;
    const qty = 1;
    const subtotal = productA.price * qty;

    const parkRes = await request.post(`${API_BASE}/api/sales/parked`, {
      headers,
      data: { items: [{ product_id: productA.id, quantity: qty, subtotal }], payment_method: 'CASH' },
    });
    expect(parkRes.ok()).toBeTruthy();
    const parkBody = await parkRes.json();
    const parkedSale = parkBody.data;
    expect(parkedSale.status).toBe('parked');
    expect(parkedSale.id).toBeGreaterThan(0);

    const listRes1 = await request.get(`${API_BASE}/api/sales/parked`, { headers });
    expect(listRes1.ok()).toBeTruthy();
    const list1 = await listRes1.json();
    expect(list1.data.some((s: any) => s.id === parkedSale.id)).toBeTruthy();

    const recallRes = await request.post(`${API_BASE}/api/sales/parked/${parkedSale.id}/recall`, { headers });
    expect(recallRes.ok()).toBeTruthy();
    const recallBody = await recallRes.json();
    expect(recallBody.data.status).toBe('recalled');
    expect(recallBody.data.items.length).toBe(qty);

    const completeRes = await request.post(`${API_BASE}/api/sales`, {
      headers,
      data: {
        payment_method: 'CASH',
        items: [{ product_id: productA.id, quantity: qty, subtotal }],
        parked_sale_id: parkedSale.id,
      },
    });
    expect(completeRes.ok()).toBeTruthy();
    const completeBody = await completeRes.json();
    expect(completeBody.data.status).toBe('completed');

    const listRes2 = await request.get(`${API_BASE}/api/sales/parked`, { headers });
    expect(listRes2.ok()).toBeTruthy();
    const list2 = await listRes2.json();
    expect(list2.data.some((s: any) => s.id === parkedSale.id)).toBeFalsy();

    const reRecallRes = await request.post(`${API_BASE}/api/sales/parked/${parkedSale.id}/recall`, { headers });
    expect(reRecallRes.ok()).toBeFalsy();
    expect(reRecallRes.status()).toBe(404);
  });

  test('should park → recall → park again cancels previous recalled sale', async ({ request }) => {
    const { headers } = auth;

    const parkResA = await request.post(`${API_BASE}/api/sales/parked`, {
      headers,
      data: { items: [{ product_id: productA.id, quantity: 1, subtotal: productA.price }], payment_method: 'CASH' },
    });
    expect(parkResA.ok()).toBeTruthy();
    const saleA = (await parkResA.json()).data;

    await request.post(`${API_BASE}/api/sales/parked/${saleA.id}/recall`, { headers });

    const parkResB = await request.post(`${API_BASE}/api/sales/parked`, {
      headers,
      data: {
        items: [{ product_id: productB.id, quantity: 1, subtotal: productB.price }],
        payment_method: 'CASH',
        recalled_sale_id: saleA.id,
      },
    });
    expect(parkResB.ok()).toBeTruthy();
    const saleB = (await parkResB.json()).data;
    expect(saleB.status).toBe('parked');
    expect(saleB.id).not.toBe(saleA.id);

    const listRes = await request.get(`${API_BASE}/api/sales/parked`, { headers });
    expect(listRes.ok()).toBeTruthy();
    const list = await listRes.json();
    expect(list.data.some((s: any) => s.id === saleA.id)).toBeFalsy();
    expect(list.data.some((s: any) => s.id === saleB.id)).toBeTruthy();
  });

  test('should recall already-recalled sale (re-recall without consumption)', async ({ request }) => {
    const { headers } = auth;
    const res = await park(request, headers, productA);
    expect(res.ok()).toBeTruthy();
    const sale = (await res.json()).data;

    const recall1 = await request.post(`${API_BASE}/api/sales/parked/${sale.id}/recall`, { headers });
    expect(recall1.ok()).toBeTruthy();
    expect((await recall1.json()).data.status).toBe('recalled');

    const recall2 = await request.post(`${API_BASE}/api/sales/parked/${sale.id}/recall`, { headers });
    expect(recall2.ok()).toBeTruthy();
    expect((await recall2.json()).data.status).toBe('recalled');
  });

  test('should fail to complete with parked_sale_id without prior recall (409)', async ({ request }) => {
    const { headers } = auth;
    const res = await park(request, headers, productA);
    expect(res.ok()).toBeTruthy();
    const sale = (await res.json()).data;

    const completeRes = await request.post(`${API_BASE}/api/sales`, {
      headers,
      data: {
        payment_method: 'CASH',
        items: [{ product_id: productA.id, quantity: 1, subtotal: productA.price }],
        parked_sale_id: sale.id,
      },
    });
    expect(completeRes.ok()).toBeFalsy();
    expect(completeRes.status()).toBe(409);
    const body = await completeRes.json();
    expect(body.error).toContain('checked out or cancelled');
  });

  test('should fail to recall a cancelled sale (404)', async ({ request }) => {
    const { headers } = auth;
    const res = await park(request, headers, productA);
    expect(res.ok()).toBeTruthy();
    const sale = (await res.json()).data;

    const delRes = await request.delete(`${API_BASE}/api/sales/parked/${sale.id}`, { headers });
    expect(delRes.ok()).toBeTruthy();

    const recallRes = await request.post(`${API_BASE}/api/sales/parked/${sale.id}/recall`, { headers });
    expect(recallRes.ok()).toBeFalsy();
    expect(recallRes.status()).toBe(404);
  });

  test('should fail to recall non-existent sale (404)', async ({ request }) => {
    const { headers } = auth;
    const recallRes = await request.post(`${API_BASE}/api/sales/parked/99999999/recall`, { headers });
    expect(recallRes.ok()).toBeFalsy();
    expect(recallRes.status()).toBe(404);
  });

  test('should cancel (DELETE) a parked sale', async ({ request }) => {
    const { headers } = auth;
    const res = await park(request, headers, productA);
    expect(res.ok()).toBeTruthy();
    const sale = (await res.json()).data;

    const delRes = await request.delete(`${API_BASE}/api/sales/parked/${sale.id}`, { headers });
    expect(delRes.ok()).toBeTruthy();

    const listRes = await request.get(`${API_BASE}/api/sales/parked`, { headers });
    const list = await listRes.json();
    expect(list.data.some((s: any) => s.id === sale.id)).toBeFalsy();
  });

  test('should cancel (DELETE) a recalled sale', async ({ request }) => {
    const { headers } = auth;
    const res = await park(request, headers, productA);
    expect(res.ok()).toBeTruthy();
    const sale = (await res.json()).data;

    await request.post(`${API_BASE}/api/sales/parked/${sale.id}/recall`, { headers });

    const delRes = await request.delete(`${API_BASE}/api/sales/parked/${sale.id}`, { headers });
    expect(delRes.ok()).toBeTruthy();

    const listRes = await request.get(`${API_BASE}/api/sales/parked`, { headers });
    const list = await listRes.json();
    expect(list.data.some((s: any) => s.id === sale.id)).toBeFalsy();
  });

  test('should fail to double-cancel a sale (404)', async ({ request }) => {
    const { headers } = auth;
    const res = await park(request, headers, productA);
    expect(res.ok()).toBeTruthy();
    const sale = (await res.json()).data;

    await request.delete(`${API_BASE}/api/sales/parked/${sale.id}`, { headers });

    const del2 = await request.delete(`${API_BASE}/api/sales/parked/${sale.id}`, { headers });
    expect(del2.ok()).toBeFalsy();
    expect(del2.status()).toBe(404);
  });

  test('should reject park with empty items (400)', async ({ request }) => {
    const { headers } = auth;
    const res = await request.post(`${API_BASE}/api/sales/parked`, {
      headers,
      data: { items: [], payment_method: 'CASH' },
    });
    expect(res.ok()).toBeFalsy();
    expect(res.status()).toBe(400);
  });

  test('should list parked sales and return valid array', async ({ request }) => {
    const { headers } = auth;
    const listRes = await request.get(`${API_BASE}/api/sales/parked`, { headers });
    expect(listRes.ok()).toBeTruthy();
    const list = await listRes.json();
    expect(Array.isArray(list.data)).toBeTruthy();
  });

  test('should reject park without items field (400)', async ({ request }) => {
    const { headers } = auth;
    const res = await request.post(`${API_BASE}/api/sales/parked`, {
      headers,
      data: { payment_method: 'CASH' },
    });
    expect(res.ok()).toBeFalsy();
    expect(res.status()).toBe(400);
  });

  test('should park after recall without recalled_sale_id leaves recalled sale unchanged', async ({ request }) => {
    const { headers } = auth;
    const resA = await park(request, headers, productA);
    expect(resA.ok()).toBeTruthy();
    const saleA = (await resA.json()).data;

    await request.post(`${API_BASE}/api/sales/parked/${saleA.id}/recall`, { headers });

    const resB = await park(request, headers, productB);
    expect(resB.ok()).toBeTruthy();

    const listRes = await request.get(`${API_BASE}/api/sales/parked`, { headers });
    const list = await listRes.json();
    expect(list.data.some((s: any) => s.id === saleA.id)).toBeTruthy();
  });
});
