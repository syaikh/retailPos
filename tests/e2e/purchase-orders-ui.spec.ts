import { test, expect } from '@playwright/test';
import { TEST_USERS, API_BASE, authHeader, loginUI, logoutUI, getToken } from './fixtures';

test.describe('Purchase Orders - UI Flow', () => {
  let headers: Record<string, string>;
  let supplier: { id: number; name: string };
  let product: { id: number; name: string; sku: string; price: number };
  let initialStock: number;
  let unitCost: number;

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

    // Fetch product price and stock
    const prodDetailRes = await request.get(`${API_BASE}/api/products/${product.id}`, { headers });
    expect(prodDetailRes.ok()).toBeTruthy();
    const prodDetailBody = await prodDetailRes.json();
    const prodDetail = prodDetailBody.data || prodDetailBody;
    product = { ...product, price: prodDetail.price };
    initialStock = prodDetail.stock;

    unitCost = Math.round(product.price * 0.65);
  });

  test.beforeEach(async ({ page }) => {
    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('full PO creation -> confirm -> receive -> stock increased -> table verification', async ({ page, request }) => {
    await page.goto('/purchase-orders');
    await page.waitForTimeout(1000);

    // ---- Create PO via UI form ----
    await page.locator('button').filter({ hasText: 'Create PO' }).click();
    await page.waitForTimeout(500);

    const formModal = page.locator('[role="dialog"][aria-label="Create Purchase Order"]');
    await expect(formModal).toBeVisible({ timeout: 5000 });

    // --- Step 1: PO Details ---
    // Supplier - SelectSearch
    const supplierSelect = formModal.locator('label:has-text("Supplier")').locator('[aria-haspopup="listbox"]');
    await supplierSelect.click();
    await page.waitForTimeout(300);
    const listbox = page.locator('[role="listbox"]');
    await expect(listbox).toBeVisible({ timeout: 3000 });
    await listbox.locator('[role="option"]').first().click();
    await page.waitForTimeout(300);

    // Expected Date
    const expectedDate = new Date(Date.now() + 7 * 86400000).toISOString().split('T')[0];
    const dateInput = formModal.locator('input[type="date"]');
    await dateInput.fill(expectedDate);

    // Payment Term - select (use selectOption which works with <select>)
    await formModal.locator('select').selectOption('Cash on Delivery');

    // Click Next
    await formModal.locator('button').filter({ hasText: 'Next' }).click();
    await page.waitForTimeout(500);

    // --- Step 2: Items ---
    // Product - SelectSearch (in items table, not wrapped in a label)
    const productSelect = formModal.locator('button').filter({ hasText: 'Select product' });
    await productSelect.first().click();
    await page.waitForTimeout(300);
    const productListbox = page.locator('[role="listbox"]');
    await expect(productListbox).toBeVisible({ timeout: 3000 });
    await productListbox.locator('[role="option"]').first().click();
    await page.waitForTimeout(300);

    // Qty - first input[inputmode="numeric"] in step 2
    const qtyInput = formModal.locator('input[inputmode="numeric"]').first();
    await qtyInput.click();
    await qtyInput.fill('10');

    // Unit Cost - CurrencyInput (use pressSequentially to trigger $bindable)
    const unitCostInput = formModal.locator('input[inputmode="numeric"]').nth(1);
    await unitCostInput.click();
    await unitCostInput.pressSequentially(String(unitCost), { delay: 50 });

    // Create draft
    const createBtn = formModal.locator('button').filter({ hasText: 'Create Draft' });
    await expect(createBtn).toBeEnabled({ timeout: 3000 });

    const createResponsePromise = page.waitForResponse(
      resp => resp.url().includes('/api/purchase-orders') && resp.request().method() === 'POST'
    );
    await createBtn.click();
    const createResponse = await createResponsePromise;
    expect(createResponse.status(), `Create PO failed: ${createResponse.status()} ${await createResponse.text()}`).toBe(201);
    const createBody = await createResponse.json();
    const createdPO = createBody.data || createBody;
    const poId = createdPO.id;
    expect(poId).toBeGreaterThan(0);

    // Wait for success toast and modal close
    await expect(page.locator('text=Purchase order created')).toBeVisible({ timeout: 5000 });
    await expect(formModal).not.toBeVisible({ timeout: 5000 });
    await page.waitForTimeout(500);

    // Get PO number via API
    const poDetailRes = await request.get(`${API_BASE}/api/purchase-orders/${poId}`, { headers });
    expect(poDetailRes.ok()).toBeTruthy();
    const poDetailBody = await poDetailRes.json();
    const poDetail = poDetailBody.data || poDetailBody;
    const poNumber = poDetail.po_number;
    expect(poNumber).toBeTruthy();

    // ---- Verify stock unchanged after PO creation (not yet received) ----
    const stockAfterPO = (await (await request.get(`${API_BASE}/api/products/${product.id}`, { headers })).json()).data?.stock ?? initialStock;
    expect(stockAfterPO).toBe(initialStock);

    // ---- Verify table row is correct ----
    const table = page.locator('[role="grid"][aria-label="Purchase orders"]');
    const draftRow = table.locator('tbody tr').filter({ hasText: poNumber }).first();
    await expect(draftRow).toBeVisible({ timeout: 10000 });

    // PO number column
    await expect(draftRow.locator('td').first()).toHaveText(poNumber);

    // Status badge shows "Draft"
    await expect(draftRow.locator('span').filter({ hasText: 'Draft' })).toBeVisible({ timeout: 3000 });

    // ---- Confirm the draft PO ----
    const actionBtn = draftRow.locator('button[aria-label*="Actions for"]');
    await actionBtn.click();
    await page.waitForTimeout(300);

    page.once('dialog', async (dialog) => {
      expect(dialog.message()).toContain('Confirm purchase order');
      await dialog.accept();
    });
    await page.locator('button').filter({ hasText: 'Confirm' }).click({ noWaitAfter: true });
    await page.waitForTimeout(2000);

    await expect(page.locator('text=Confirmed').first()).toBeVisible({ timeout: 10000 });
    await page.waitForTimeout(500);

    // ---- Verify table shows confirmed status ----
    const confirmedRow = table.locator('tbody tr').filter({ hasText: poNumber }).first();
    await expect(confirmedRow).toBeVisible({ timeout: 5000 });
    await expect(confirmedRow.locator('span').filter({ hasText: 'Confirmed' })).toBeVisible({ timeout: 3000 });

    // ---- Receive Goods ----
    const actionBtn2 = confirmedRow.locator('button[aria-label*="Actions for"]');
    await actionBtn2.click();
    await page.waitForTimeout(300);

    await page.locator('button').filter({ hasText: 'Receive' }).click();
    await page.waitForTimeout(500);

    const grModal = page.locator('[role="dialog"][aria-label*="Receive Goods"]');
    await expect(grModal).toBeVisible({ timeout: 5000 });

    await expect(grModal.locator('input[type="number"]').first()).toBeVisible({ timeout: 10000 });

    // Fill GR form
    const doInput = grModal.locator('input[placeholder="DO-001"]');
    await doInput.click();
    await doInput.fill(`DO-E2E-UI-${Date.now()}`);

    const qtyGoodInput = grModal.locator('input[type="number"]').first();
    await qtyGoodInput.click();
    await qtyGoodInput.fill('10');
    await page.waitForTimeout(300);

    const submitBtn = grModal.locator('button').filter({ hasText: 'Create Goods Receipt' });
    await expect(submitBtn).not.toBeDisabled({ timeout: 3000 });

    const responsePromise = page.waitForResponse(
      resp => resp.url().includes('/api/goods-receipts') && resp.request().method() === 'POST'
    );
    await submitBtn.click();
    const grResponse = await responsePromise;
    expect(grResponse.status(), `GR API returned ${grResponse.status()}: ${await grResponse.text()}`).toBe(201);
    const grBody = await grResponse.json();
    const grData = grBody.data || grBody;
    expect(grData.purchase_order_id).toBe(poId);
    await page.waitForTimeout(500);

    await expect(page.locator('text=Goods receipt created')).toBeVisible({ timeout: 5000 });
    await expect(grModal).not.toBeVisible({ timeout: 5000 });

    // ---- Verify stock increased via API ----
    const finalStockRes = await request.get(`${API_BASE}/api/products/${product.id}`, { headers });
    expect(finalStockRes.ok()).toBeTruthy();
    const finalStockBody = await finalStockRes.json();
    const finalStock = (finalStockBody.data || finalStockBody).stock;
    expect(finalStock).toBe(initialStock + 10);

    // ---- Verify table shows fully_received status (reload to get fresh data) ----
    await page.goto('/purchase-orders');
    await page.waitForTimeout(1000);
    const finalRow = table.locator('tbody tr').filter({ hasText: poNumber }).first();
    await expect(finalRow).toBeVisible({ timeout: 5000 });
    await expect(finalRow.locator('span').filter({ hasText: 'Fully Received' })).toBeVisible({ timeout: 5000 });
  });
});
