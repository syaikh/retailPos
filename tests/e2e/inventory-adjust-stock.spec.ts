import { test, expect, type Page } from '@playwright/test';
import { TEST_USERS, API_BASE, authHeader, loginUI, logoutUI, getToken } from './fixtures';

async function navigateToInventory(page: Page) {
  const sidebar = page.locator('aside');
  const masterDataBtn = sidebar.locator('button', { hasText: 'Master Data' }).first();
  await masterDataBtn.click();
  await page.waitForTimeout(300);
  const productsBtn = sidebar.locator('button', { hasText: 'Products' }).first();
  await productsBtn.click();
  await expect(page.locator('text=PRODUCT NAME')).toBeVisible({ timeout: 10000 });
}

test.describe('Inventory Stock Adjustment', () => {
  test.beforeEach(async ({ page }) => {
    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    await navigateToInventory(page);
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('opens adjust stock modal when clicking Adjust Stock in action dropdown', async ({ page }) => {
    // Find first product row
    const productRow = page.locator('tbody tr').filter({ hasNotText: 'No products' }).first();

    // Click the action button (MoreVertical icon) in the first row
    const actionBtn = productRow.locator('button[title="Actions"]');
    await actionBtn.click();

    // Wait for dropdown to appear and click Adjust Stock
    const adjustOption = page.locator('text=Adjust Stock');
    await expect(adjustOption).toBeVisible({ timeout: 5000 });
    await adjustOption.click();

    // Modal should be visible
    await expect(page.getByText('Adjust Stock').first()).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Product:')).toBeVisible();
    await expect(page.getByText('Current Stock:')).toBeVisible();
    await expect(page.locator('#adjust-qty')).toBeVisible();
    await expect(page.locator('#adjust-notes')).toBeVisible();
  });

  test('adjusts stock positively and reflects via API', async ({ page, request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    // Find a product with existing stock
    const prodRes = await request.get(`${API_BASE}/api/products?limit=10&offset=0`, {
      headers: authHeader(token),
    });
    const prodData = await prodRes.json();
    const targetProd = prodData.data.find((p: any) => p.stock > 5);
    expect(targetProd).toBeTruthy();
    const originalStock = targetProd.stock;
    const adjustmentQty = 15;

    // Find the product row in table by searching for the SKU
    await page.fill('input[placeholder="Search by name, SKU, or barcode..."]', targetProd.sku);
    await page.waitForTimeout(1500); // Wait for debounce + API response

    const productRow = page.locator('tbody tr').filter({ hasText: targetProd.sku }).first();
    await expect(productRow).toBeVisible({ timeout: 5000 });

    // Open the adjust stock modal for this product
    const actionBtn = productRow.getByRole('button', { name: 'Actions' });
    await actionBtn.click();

    // Click Adjust Stock in dropdown
    await page.locator('text=Adjust Stock').click();

    // Wait for modal
    await expect(page.getByText('Adjust Stock').first()).toBeVisible({ timeout: 5000 });

    // Verify product info shown in modal (use locator within modal to avoid strict mode)
    const modal = page.getByRole('dialog', { name: 'Adjust Stock' });
    await expect(modal.getByText(targetProd.name)).toBeVisible();

    // Fill in the adjustment
    await page.fill('#adjust-qty', String(adjustmentQty));
    await page.fill('#adjust-notes', `E2E test +${adjustmentQty}`);

    // Click the Adjust Stock button in the footer
    const adjustBtn = page.locator('button').filter({ hasText: 'Adjust Stock' }).last();
    await adjustBtn.click();

    // Modal should close
    await expect(page.getByText('Adjust Stock').first()).toBeHidden({ timeout: 10000 });

    // Verify success toast
    await expect(page.locator('text=Stock adjusted successfully')).toBeVisible({ timeout: 5000 });

    // Verify via API that stock was actually updated
    const afterRes = await request.get(`${API_BASE}/api/products/${targetProd.id}`, {
      headers: authHeader(token),
    });
    const afterData = await afterRes.json();
    expect(afterData.data.stock).toBe(originalStock + adjustmentQty);

    // Revert the adjustment
    await request.post(`${API_BASE}/api/inventory/adjust`, {
      headers: authHeader(token),
      data: {
        product_id: targetProd.id,
        quantity_change: -adjustmentQty,
        notes: 'E2E test revert',
      },
    });
  });

  test('adjusts stock negatively (reduction)', async ({ page, request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    // Find a product with enough stock to reduce
    const prodRes = await request.get(`${API_BASE}/api/products?limit=20&offset=0`, {
      headers: authHeader(token),
    });
    const prodData = await prodRes.json();
    const targetProd = prodData.data.find((p: any) => p.stock >= 10);
    expect(targetProd).toBeTruthy();
    const originalStock = targetProd.stock;
    const reductionQty = -5;

    // Search for product
    await page.fill('input[placeholder="Search by name, SKU, or barcode..."]', targetProd.sku);
    await page.waitForTimeout(1500);

    const productRow = page.locator('tbody tr').filter({ hasText: targetProd.sku }).first();
    await expect(productRow).toBeVisible({ timeout: 5000 });

    // Open action dropdown
    const actionBtn = productRow.locator('button[title="Actions"]');
    await actionBtn.click();
    await page.locator('text=Adjust Stock').click();

    // Wait for modal
    await expect(page.getByText('Adjust Stock').first()).toBeVisible({ timeout: 5000 });

    // Fill negative adjustment
    await page.fill('#adjust-qty', String(reductionQty));
    await page.fill('#adjust-notes', `E2E test ${reductionQty}`);

    // Submit
    const adjustBtn = page.locator('button').filter({ hasText: 'Adjust Stock' }).last();
    await adjustBtn.click();

    // Modal should close and show success
    await expect(page.getByText('Adjust Stock').first()).toBeHidden({ timeout: 10000 });
    await expect(page.locator('text=Stock adjusted successfully')).toBeVisible({ timeout: 5000 });

    // Verify via API
    const afterRes = await request.get(`${API_BASE}/api/products/${targetProd.id}`, {
      headers: authHeader(token),
    });
    const afterData = await afterRes.json();
    expect(afterData.data.stock).toBe(originalStock + reductionQty);

    // Revert
    await request.post(`${API_BASE}/api/inventory/adjust`, {
      headers: authHeader(token),
      data: {
        product_id: targetProd.id,
        quantity_change: -reductionQty,
        notes: 'E2E test revert',
      },
    });
  });

  test('rejects zero quantity change', async ({ page }) => {
    // Find any product row
    const productRow = page.locator('tbody tr').filter({ hasNotText: 'No products' }).first();

    // Open action dropdown
    const actionBtn = productRow.locator('button[title="Actions"]');
    await actionBtn.click();
    await page.locator('text=Adjust Stock').click();

    // Wait for modal
    await expect(page.getByText('Adjust Stock').first()).toBeVisible({ timeout: 5000 });

    // Fill notes but leave quantity as zero (default)
    await page.fill('#adjust-notes', 'Test notes');

    // Try to submit with zero quantity
    const adjustBtn = page.locator('button').filter({ hasText: 'Adjust Stock' }).last();
    await adjustBtn.click();

    // Should show error toast, modal stays open
    await expect(page.locator('text=Quantity change must be non-zero')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Adjust Stock').first()).toBeVisible();
  });

  test('rejects missing notes', async ({ page }) => {
    // Find any product row
    const productRow = page.locator('tbody tr').filter({ hasNotText: 'No products' }).first();

    // Open action dropdown
    const actionBtn = productRow.locator('button[title="Actions"]');
    await actionBtn.click();
    await page.locator('text=Adjust Stock').click();

    // Wait for modal
    await expect(page.getByText('Adjust Stock').first()).toBeVisible({ timeout: 5000 });

    // Fill quantity but leave notes empty
    await page.fill('#adjust-qty', '10');

    // Try to submit without notes
    const adjustBtn = page.locator('button').filter({ hasText: 'Adjust Stock' }).last();
    await adjustBtn.click();

    // Should show error toast, modal stays open
    await expect(page.locator('text=Notes are required')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Adjust Stock').first()).toBeVisible();
  });

  test('can cancel the adjust stock modal', async ({ page }) => {
    const productRow = page.locator('tbody tr').filter({ hasNotText: 'No products' }).first();

    const actionBtn = productRow.locator('button[title="Actions"]');
    await actionBtn.click();
    await page.locator('text=Adjust Stock').click();

    await expect(page.getByText('Adjust Stock').first()).toBeVisible({ timeout: 5000 });

    // Click Cancel
    const cancelBtn = page.locator('button').filter({ hasText: 'Cancel' }).last();
    await cancelBtn.click();

    // Modal should close
    await expect(page.getByText('Adjust Stock').first()).toBeHidden({ timeout: 5000 });
  });

  test('manager role can also adjust stock', async ({ page }) => {
    // Logout and login as manager
    await page.evaluate(() => sessionStorage.clear());
    await page.reload();

    await loginUI(page, TEST_USERS.manager.username, TEST_USERS.manager.password);
    await navigateToInventory(page);

    // Find product row and open dropdown
    const productRow = page.locator('tbody tr').filter({ hasNotText: 'No products' }).first();
    const actionBtn = productRow.locator('button[title="Actions"]');
    await actionBtn.click();

    // Adjust Stock option should be visible for manager
    const adjustOption = page.locator('text=Adjust Stock');
    await expect(adjustOption).toBeVisible({ timeout: 5000 });
    await adjustOption.click();

    // Modal should open
    await expect(page.getByText('Adjust Stock').first()).toBeVisible({ timeout: 5000 });

    // Close modal
    const cancelBtn = page.locator('button').filter({ hasText: 'Cancel' }).last();
    await cancelBtn.click();
    await expect(page.getByText('Adjust Stock').first()).toBeHidden({ timeout: 5000 });
  });

  test('cashier role cannot see Adjust Stock option', async ({ page }) => {
    // Logout and login as cashier
    await page.evaluate(() => sessionStorage.clear());
    await page.reload();

    await loginUI(page, TEST_USERS.cashier.username, TEST_USERS.cashier.password);

    // Cashier doesn't have Inventory in sidebar, navigate directly
    await page.goto('/inventory');
    await expect(page.locator('text=PRODUCT NAME')).toBeVisible({ timeout: 10000 });

    // Find product row and open dropdown
    const productRow = page.locator('tbody tr').filter({ hasNotText: 'No products' }).first();
    const actionBtn = productRow.locator('button[title="Actions"]');
    await actionBtn.click();

    // Adjust Stock option should NOT be visible for cashier
    const adjustOption = page.locator('text=Adjust Stock');
    await expect(adjustOption).toBeHidden({ timeout: 5000 });
  });
});
