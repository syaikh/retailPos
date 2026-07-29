import { test, expect } from '@playwright/test';
import { TEST_USERS, API_BASE, authHeader, loginUI, logoutUI, getToken } from './fixtures';

test.describe('Purchase Orders - UI Flow', () => {
  let headers: Record<string, string>;
  let supplier: { id: number; name: string };
  let product: { id: number; name: string; sku: string };
  let initialStock: number;

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

    const stockRes = await request.get(`${API_BASE}/api/products/${product.id}`, { headers });
    const stockBody = await stockRes.json();
    initialStock = (stockBody.data || stockBody).stock;
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

    // Click Create PO
    await page.locator('button').filter({ hasText: 'Create PO' }).click();
    await page.waitForTimeout(500);

    // Modal should appear
    const modal = page.locator('[role="dialog"][aria-label="Create Purchase Order"]');
    await expect(modal).toBeVisible({ timeout: 5000 });

    // ---- Step 1: PO Details ----
    // Select supplier
    const supplierSelect = modal.locator('button').filter({ hasText: 'Select supplier' });
    await expect(supplierSelect).toBeVisible({ timeout: 5000 });
    await supplierSelect.click();
    await page.waitForTimeout(300);

    const supplierOption = page.locator('[role="option"]').filter({ hasText: supplier.name }).first();
    await expect(supplierOption).toBeVisible({ timeout: 5000 });
    await supplierOption.click();
    await page.waitForTimeout(300);

    // Fill expected date
    const expectedDate = new Date(Date.now() + 7 * 86400000).toISOString().split('T')[0];
    const dateInput = modal.locator('input[type="date"]');
    await dateInput.fill(expectedDate);

    // Select payment term
    const paymentSelect = modal.locator('select');
    await paymentSelect.selectOption('Cash on Delivery');

    // Click Next
    await modal.locator('button').filter({ hasText: 'Next' }).click();

    // Wait for step 2 to load (spinner disappears, Add Item button enabled)
    await expect(modal.locator('button').filter({ hasText: 'Add Item' })).not.toBeDisabled({ timeout: 10000 });
    await page.waitForTimeout(300);

    // ---- Step 2: Items ----
    // Click Add Item
    await modal.locator('button').filter({ hasText: 'Add Item' }).click();
    await page.waitForTimeout(300);

    // Select product
    const productSelect = modal.locator('button').filter({ hasText: 'Select product' }).first();
    await expect(productSelect).toBeVisible({ timeout: 5000 });
    await productSelect.click();
    await page.waitForTimeout(300);

    // Wait for dropdown options to load, then select the product
    const productOption = modal.locator('[role="option"]').filter({ hasText: new RegExp(product.sku, 'i') }).first();
    await expect(productOption).toBeVisible({ timeout: 10000 });
    await productOption.click();
    await page.waitForTimeout(300);

    // Enter qty
    const itemRow = modal.locator('table tbody tr').first();
    const qtyCell = itemRow.locator('td').nth(1);
    await qtyCell.locator('input').fill('10');

    // Enter unit cost (CurrencyInput - 3rd column)
    const costCell = itemRow.locator('td').nth(2);
    await costCell.locator('input').click();
    await costCell.locator('input').fill(String(Math.round(product.price * 0.65)));

    // Click Create Draft
    await modal.locator('button').filter({ hasText: 'Create Draft' }).click();

    // Wait for success toast and modal to close
    await expect(page.locator('text=Purchase order created')).toBeVisible({ timeout: 10000 });
    await expect(modal).not.toBeVisible({ timeout: 5000 });
    await page.waitForTimeout(500);

    // ---- Confirm PO ----
    const table = page.locator('[role="grid"][aria-label="Purchase orders"]');
    const firstRow = table.locator('tbody tr').first();
    await expect(firstRow).toBeVisible({ timeout: 5000 });

    const actionBtn = firstRow.locator('button[aria-label*="Actions for"]');
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
    const confRow = table.locator('tbody tr').filter({ hasText: 'Confirmed' }).first();
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
    await page.waitForTimeout(500);

    // Check success toast
    await expect(page.locator('text=Goods receipt created')).toBeVisible({ timeout: 5000 });
    await expect(grModal).not.toBeVisible({ timeout: 5000 });
    await page.waitForTimeout(500);

    // ---- Verify stock increased via API ----
    const finalStockRes = await request.get(`${API_BASE}/api/products/${product.id}`, { headers });
    const finalStockBody = await finalStockRes.json();
    const finalStock = (finalStockBody.data || finalStockBody).stock;
    expect(finalStock).toBe(initialStock + 10);
  });
});
