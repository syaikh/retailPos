import { test, expect } from './fixtures';
import type { Page } from '@playwright/test';
import { TEST_USERS, FRONTEND_BASE, loginUI, logoutUI } from './fixtures';

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
    await expect(page.locator('text=quantity change must not be zero')).toBeVisible({ timeout: 5000 });
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
    await page.goto(`${FRONTEND_BASE}/inventory`);
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
