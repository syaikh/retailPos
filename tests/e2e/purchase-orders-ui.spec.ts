import { test, expect } from '@playwright/test';
import { TEST_USERS, API_BASE, authHeader, loginUI, logoutUI, getToken } from './fixtures';

test.describe('Purchase Orders - UI Flow', () => {
  let headers: Record<string, string>;
  let supplier: { id: number; name: string };
  let product: { id: number; name: string; sku: string };
  let initialStock: number;

  let draftPOId: number;

  test.beforeAll(async ({ request }) => {
    const token = await getToken(request);
    headers = authHeader(token);

    const supRes = await request.get(`${API_BASE}/api/suppliers?limit=1`, { headers });
    expect(supRes.ok()).toBeTruthy();
    const supBody = await supRes.json();
    supplier = (supBody.data || supBody)[0];
    expect(supplier).toBeTruthy();

    // Find a product linked to this supplier; if none, link the first product
    let linkedRes = await request.get(`${API_BASE}/api/suppliers/${supplier.id}/products`, { headers });
    let linkedBody = await linkedRes.json();
    let linkedProducts = linkedBody.data || [];
    if (linkedProducts.length === 0) {
      const allProdRes = await request.get(`${API_BASE}/api/products?limit=1`, { headers });
      const allProdBody = await allProdRes.json();
      const anyProduct = (allProdBody.data || [])[0];
      expect(anyProduct).toBeTruthy();
      await request.post(`${API_BASE}/api/suppliers/${supplier.id}/products`, {
        headers,
        data: { product_id: anyProduct.id, unit_cost: Math.round(anyProduct.price * 0.6) },
      });
      linkedRes = await request.get(`${API_BASE}/api/suppliers/${supplier.id}/products`, { headers });
      linkedBody = await linkedRes.json();
      linkedProducts = linkedBody.data || [];
    }
    expect(linkedProducts.length).toBeGreaterThan(0);
    product = { id: linkedProducts[0].product_id, name: linkedProducts[0].product_name, sku: linkedProducts[0].product_sku };
    // Fetch product price
    const prodDetailRes = await request.get(`${API_BASE}/api/products/${product.id}`, { headers });
    const prodDetailBody = await prodDetailRes.json();
    const prodDetail = prodDetailBody.data || prodDetailBody;
    product = { ...product, price: prodDetail.price };

    const stockRes = await request.get(`${API_BASE}/api/products/${product.id}`, { headers });
    const stockBody = await stockRes.json();
    initialStock = (stockBody.data || stockBody).stock;

    // Create draft PO via API (CurrencyInput binding doesn't work in Playwright)
    const expectedDate = new Date(Date.now() + 7 * 86400000).toISOString().split('T')[0];
    const unitCost = Math.round(product.price * 0.65);
    const storeRes = await request.get(`${API_BASE}/api/stores/active`, { headers });
    expect(storeRes.ok()).toBeTruthy();
    const storeBody = await storeRes.json();
    const stores = storeBody.data || storeBody;
    const store = stores[0];
    const poRes = await request.post(`${API_BASE}/api/purchase-orders`, {
      headers,
      data: {
        supplier_id: supplier.id,
        store_id: store.id,
        expected_date: expectedDate,
        items: [{ product_id: product.id, qty_ordered: 10, unit_cost: unitCost }],
      },
    });
    expect(poRes.ok()).toBeTruthy();
    const poBody = await poRes.json();
    const po = poBody.data || poBody;
    draftPOId = po.id;
    expect(draftPOId).toBeGreaterThan(0);
    // Fetch PO detail to verify financial fields
    const poDetailRes = await request.get(`${API_BASE}/api/purchase-orders/${draftPOId}`, { headers });
    expect(poDetailRes.ok()).toBeTruthy();
    const poDetailBody = await poDetailRes.json();
    const poDetail = poDetailBody.data || poDetailBody;
    expect(poDetail.items[0].unit_cost).toBeGreaterThan(0);
    expect(poDetail.items[0].subtotal).toBeGreaterThan(0);
    expect(poDetail.items[0].discount_amount).toBeGreaterThanOrEqual(0);
    expect(poDetail.subtotal).toBeGreaterThan(0);
    expect(poDetail.discount_amount).toBeGreaterThanOrEqual(0);
    expect(poDetail.grand_total).toBeGreaterThan(0);
  });

  test.beforeEach(async ({ page }) => {
    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('full PO creation -> confirm -> receive -> stock increased', async ({ page, request }) => {
    await page.goto('/purchase-orders');
    await page.waitForTimeout(1000);

    // ---- Confirm the draft PO ----
    const table = page.locator('[role="grid"][aria-label="Purchase orders"]');

    // Get our PO number from API
    const poInfoRes = await request.get(`${API_BASE}/api/purchase-orders/${draftPOId}`, { headers });
    expect(poInfoRes.ok()).toBeTruthy();
    const poInfoBody = await poInfoRes.json();
    const poInfo = poInfoBody.data || poInfoBody;
    const poNumber = poInfo.po_number;
    expect(poNumber).toBeTruthy();

    // Find our draft PO row by po_number
    const draftRow = table.locator(`tbody tr`).filter({ hasText: poNumber }).first();
    await expect(draftRow).toBeVisible({ timeout: 10000 });

    const actionBtn = draftRow.locator('button[aria-label*="Actions for"]');
    await actionBtn.click();
    await page.waitForTimeout(300);

    page.once('dialog', async (dialog) => {
      expect(dialog.message()).toContain('Confirm purchase order');
      await dialog.accept();
    });
    await page.locator('button').filter({ hasText: 'Confirm' }).click({ noWaitAfter: true });
    await page.waitForTimeout(2000);

    // Wait for confirmation to process
    await expect(page.locator('text=Confirmed').first()).toBeVisible({ timeout: 10000 });
    await page.waitForTimeout(500);

    // ---- Receive Goods ----
    const confRow = table.locator('tbody tr').filter({ hasText: poNumber }).first();
    await expect(confRow).toBeVisible({ timeout: 5000 });

    const actionBtn2 = confRow.locator('button[aria-label*="Actions for"]');
    await actionBtn2.click();
    await page.waitForTimeout(300);

    await page.locator('button').filter({ hasText: 'Receive' }).click();
    await page.waitForTimeout(500);

    const grModal = page.locator('[role="dialog"][aria-label*="Receive Goods"]');
    await expect(grModal).toBeVisible({ timeout: 5000 });

    // Wait for items to load (input fields appear)
    await expect(grModal.locator('input[type="number"]').first()).toBeVisible({ timeout: 10000 });

    // Fill GR form
    const doInput = grModal.locator('input[placeholder="DO-001"]');
    await doInput.click();
    await doInput.fill(`DO-E2E-UI-${Date.now()}`);

    // Enter qty good
    const qtyGoodInput = grModal.locator('input[type="number"]').first();
    await qtyGoodInput.click();
    await qtyGoodInput.fill('10');
    await page.waitForTimeout(300);

    // Verify button is enabled (total good > 0)
    const submitBtn = grModal.locator('button').filter({ hasText: 'Create Goods Receipt' });
    await expect(submitBtn).not.toBeDisabled({ timeout: 3000 });

    // Wait for the API response
    const responsePromise = page.waitForResponse(resp => resp.url().includes('/api/goods-receipts') && resp.request().method() === 'POST');
    await submitBtn.click();
    const grResponse = await responsePromise;
    expect(grResponse.status(), `GR API returned ${grResponse.status()}: ${await grResponse.text()}`).toBe(201);
    const grBody = await grResponse.json();
    const grData = grBody.data || grBody;
    const ourPOId = grData.purchase_order_id;
    expect(ourPOId).toBeGreaterThan(0);
    await page.waitForTimeout(500);

    // Check success toast
    await expect(page.locator('text=Goods receipt created')).toBeVisible({ timeout: 5000 });
    await expect(grModal).not.toBeVisible({ timeout: 5000 });
    await page.waitForTimeout(500);

    // Verify PO financial fields
    const poDetailRes = await request.get(`${API_BASE}/api/purchase-orders/${ourPOId}`, { headers });
    expect(poDetailRes.ok()).toBeTruthy();
    const poDetailBody = await poDetailRes.json();
    const finalizedPO = poDetailBody.data || poDetailBody;
    expect(finalizedPO.items[0].unit_cost).toBeGreaterThan(0);
    expect(finalizedPO.items[0].subtotal).toBeGreaterThan(0);
    expect(finalizedPO.items[0].discount_amount).toBeGreaterThanOrEqual(0);
    expect(finalizedPO.subtotal).toBeGreaterThan(0);
    expect(finalizedPO.discount_amount).toBeGreaterThanOrEqual(0);
    expect(finalizedPO.grand_total).toBeGreaterThan(0);

    // ---- Verify stock increased via API ----
    const finalStockRes = await request.get(`${API_BASE}/api/products/${product.id}`, { headers });
    const finalStockBody = await finalStockRes.json();
    const finalStock = (finalStockBody.data || finalStockBody).stock;
    expect(finalStock).toBe(initialStock + 10);
  });
});
