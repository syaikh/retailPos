import { test, expect } from '@playwright/test';
import { TEST_USERS, API_BASE, authHeader, loginUI, logoutUI, getToken } from './fixtures';

test.describe('Purchase Orders - Notification Bell on Goods Receipt', () => {
  let headers: Record<string, string>;
  let supplier: { id: number; name: string };
  let product: { id: number; name: string; sku: string };
  let initialStock: number;
  const GR_QTY = 10;

  test.beforeAll(async ({ request }) => {
    const token = await getToken(request);
    headers = authHeader(token);

    const supRes = await request.get(`${API_BASE}/api/suppliers?limit=1`, { headers });
    expect(supRes.ok()).toBeTruthy();
    supplier = ((await supRes.json()).data || (await supRes.json()))[0];

    const prodRes = await request.get(`${API_BASE}/api/products?limit=1`, { headers });
    expect(prodRes.ok()).toBeTruthy();
    const p = ((await prodRes.json()).data || (await prodRes.json()))[0];
    product = { id: p.id, name: p.name, sku: p.sku };

    const detailRes = await request.get(`${API_BASE}/api/products/${product.id}`, { headers });
    expect(detailRes.ok()).toBeTruthy();
    initialStock = ((await detailRes.json()).data || (await detailRes.json())).stock;
  });

  test.beforeEach(async ({ page }) => {
    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('partial GR -> bell notification -> click -> product page; then full GR -> second notification', async ({ page, request }) => {
    // ---- Create and confirm PO ----
    const store = ((await (await request.get(`${API_BASE}/api/stores/active`, { headers })).json()).data ||
      (await (await request.get(`${API_BASE}/api/stores/active`, { headers })).json()));

    const expDate = new Date(Date.now() + 7 * 86400000).toISOString().split('T')[0];

    const createRes = await request.post(`${API_BASE}/api/purchase-orders`, {
      headers,
      data: {
        supplier_id: supplier.id,
        store_id: store.id,
        expected_date: expDate,
        items: [{ product_id: product.id, qty_ordered: GR_QTY * 2, unit_cost: 1000 }],
      },
    });
    expect(createRes.ok()).toBeTruthy();
    const poId = ((await createRes.json()).data || (await createRes.json())).id;

    const confirmRes = await request.post(`${API_BASE}/api/purchase-orders/${poId}/confirm`, { headers });
    expect(confirmRes.ok()).toBeTruthy();

    // Get PO item ID for GR
    const poDetail = ((await (await request.get(`${API_BASE}/api/purchase-orders/${poId}`, { headers })).json()).data ||
      (await (await request.get(`${API_BASE}/api/purchase-orders/${poId}`, { headers })).json()));
    const itemId = poDetail.items[0].id;
    expect(itemId).toBeGreaterThan(0);

    // ---- PARTIAL GR ----
    const gr1Res = await request.post(`${API_BASE}/api/goods-receipts`, {
      headers,
      data: {
        purchase_order_id: poId,
        items: [{ purchase_order_item_id: itemId, qty_good: GR_QTY, qty_damaged: 0 }],
      },
    });
    expect(gr1Res.ok()).toBeTruthy();
    const partialStock = initialStock + GR_QTY;

    // Verify stock via API
    const stock1 = ((await (await request.get(`${API_BASE}/api/products/${product.id}`, { headers })).json()).data ||
      (await (await request.get(`${API_BASE}/api/products/${product.id}`, { headers })).json())).stock;
    expect(stock1).toBe(partialStock);

    // ---- Check bell notification (WS was just received, still in memory) ----
    const bellBtn = page.locator('button[aria-label="Notifications"]');
    await expect(bellBtn.locator('span').first()).toBeVisible({ timeout: 20000 });

    // Open bell dropdown
    await bellBtn.click();
    await page.waitForTimeout(500);

    // Find stock notification for this product
    const menu = page.locator('[role="menu"]');
    await expect(menu).toBeVisible({ timeout: 3000 });

    const stockNotif = menu.locator('[role="menuitem"]').filter({ hasText: product.sku }).first();
    await expect(stockNotif).toBeVisible({ timeout: 5000 });
    await expect(stockNotif).toContainText('Stok Diubah');

    // Click notification -> should navigate to product page
    await stockNotif.click();
    await page.waitForURL('**/products/**', { timeout: 10000 });
    await expect(page.locator('h2, h3, [role="heading"]').filter({ hasText: product.name })).toBeVisible({ timeout: 5000 });

    // ---- FULL GR ----
    await page.goto('/purchase-orders');
    await page.waitForTimeout(1500);

    const gr2Res = await request.post(`${API_BASE}/api/goods-receipts`, {
      headers,
      data: {
        purchase_order_id: poId,
        items: [{ purchase_order_item_id: itemId, qty_good: GR_QTY, qty_damaged: 0 }],
      },
    });
    expect(gr2Res.ok()).toBeTruthy();
    const finalStock = partialStock + GR_QTY;

    const stock2 = ((await (await request.get(`${API_BASE}/api/products/${product.id}`, { headers })).json()).data ||
      (await (await request.get(`${API_BASE}/api/products/${product.id}`, { headers })).json())).stock;
    expect(stock2).toBe(finalStock);

    // Check second bell notification (WS was just received on current page)
    await expect(bellBtn.locator('span').first()).toBeVisible({ timeout: 20000 });
    await bellBtn.click();
    await page.waitForTimeout(500);

    const notif2 = page.locator('[role="menu"]').locator('[role="menuitem"]').filter({ hasText: product.sku }).first();
    await expect(notif2).toBeVisible({ timeout: 5000 });
    await expect(notif2).toContainText('Stok Diubah');

    await notif2.click();
    await page.waitForURL('**/products/**', { timeout: 10000 });
    await expect(page.locator('h2, h3, [role="heading"]').filter({ hasText: product.name })).toBeVisible({ timeout: 5000 });

    // Verify table shows Fully Received
    await page.goto('/purchase-orders');
    await page.waitForTimeout(1500);
    const table = page.locator('[role="grid"][aria-label="Purchase orders"]');
    const finalRow = table.locator('tbody tr').filter({ hasText: poDetail.po_number }).first();
    await expect(finalRow).toBeVisible({ timeout: 5000 });
    await expect(finalRow.locator('span').filter({ hasText: 'Fully Received' })).toBeVisible({ timeout: 5000 });
  });
});
