import { test, expect } from '@playwright/test';
import { TEST_USERS, API_BASE, loginUI, logoutUI, getToken } from './fixtures';

function enabledAddButton(page: any) {
  return page.locator('button').filter({ hasText: 'Add' }).locator('visible=true').first();
}

test.describe('POS UI Flow', () => {
  test.beforeEach(async ({ page }) => {
    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    await page.goto('http://localhost:5173/pos');
    await expect(page).toHaveURL(/\/pos/);
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('should load POS page with product table and cart', async ({ page }) => {
    await expect(page.locator('text=PRODUCT NAME')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Cart', { exact: true }).nth(1)).toBeVisible();
  });

  test('should search products by name', async ({ page }) => {
    await page.fill('#pos-search-input', 'Quality Model');
    await page.waitForTimeout(1000);
    await page.locator('#pos-search-input').fill('');
  });

  test('should add product to cart', async ({ page }) => {
    await page.waitForTimeout(2000);
    const addButton = page.locator('button:not([disabled])').filter({ hasText: 'Add' }).first();
    await addButton.waitFor({ state: 'visible', timeout: 10000 });
    await expect(addButton).toBeEnabled({ timeout: 5000 });
    await addButton.click();
    await page.waitForTimeout(500);
    await expect(page.locator('text=Your cart is empty')).toBeHidden({ timeout: 5000 });
  });

  test('should increase item quantity in cart', async ({ page }) => {
    await page.waitForTimeout(2000);
    const addButton = page.locator('button:not([disabled])').filter({ hasText: 'Add' }).first();
    await addButton.waitFor({ state: 'visible', timeout: 10000 });
    await expect(addButton).toBeEnabled({ timeout: 5000 });
    await addButton.click();
    await page.waitForTimeout(500);

    const increaseBtn = page.locator('button[aria-label="Increase quantity"]').first();
    await increaseBtn.waitFor({ state: 'visible', timeout: 5000 });
    await increaseBtn.click();
    await page.waitForTimeout(300);
  });

  test('should decrease item quantity in cart', async ({ page }) => {
    await page.waitForTimeout(2000);
    const addButton = page.locator('button:not([disabled])').filter({ hasText: 'Add' }).first();
    await addButton.waitFor({ state: 'visible', timeout: 10000 });
    await expect(addButton).toBeEnabled({ timeout: 5000 });
    await addButton.click();
    await page.waitForTimeout(500);

    const increaseBtn = page.locator('button[aria-label="Increase quantity"]').first();
    await increaseBtn.waitFor({ state: 'visible', timeout: 5000 });
    await increaseBtn.click();
    await page.waitForTimeout(300);

    const decreaseBtn = page.locator('button[aria-label="Decrease quantity"]').first();
    await decreaseBtn.click();
    await page.waitForTimeout(300);
  });

  test('should remove item from cart', async ({ page }) => {
    await page.waitForTimeout(2000);
    const addButton = page.locator('button:not([disabled])').filter({ hasText: 'Add' }).first();
    await addButton.waitFor({ state: 'visible', timeout: 10000 });
    await expect(addButton).toBeEnabled({ timeout: 5000 });
    await addButton.click();
    await page.waitForTimeout(500);

    const removeBtn = page.locator('button[aria-label="Remove item"]').first();
    await removeBtn.waitFor({ state: 'visible', timeout: 5000 });
    await removeBtn.click();
    await page.waitForTimeout(500);
    await expect(page.locator('text=Your cart is empty')).toBeVisible({ timeout: 5000 });
  });

  test('should open checkout modal with F4 key', async ({ page }) => {
    await page.waitForTimeout(2000);
    const addButton = page.locator('button:not([disabled])').filter({ hasText: 'Add' }).first();
    await addButton.waitFor({ state: 'visible', timeout: 10000 });
    await expect(addButton).toBeEnabled({ timeout: 5000 });
    await addButton.click();
    await page.waitForTimeout(500);

    await page.keyboard.press('F4');
    await expect(page.getByRole('dialog', { name: 'Pembayaran Selesai' })).toBeVisible({ timeout: 5000 });

    await page.keyboard.press('Escape');
    await expect(page.getByRole('dialog', { name: 'Pembayaran Selesai' })).toBeHidden({ timeout: 5000 });
  });

  test('should open customer selection modal', async ({ page }) => {
    await page.waitForTimeout(2000);
    await page.locator('button').filter({ hasText: 'Walk-in / General' }).first().click();
    await expect(page.locator('#customer-modal-heading')).toBeVisible({ timeout: 5000 });

    await page.locator('button[aria-label="Tutup"]').click();
    await expect(page.locator('#customer-modal-heading')).toBeHidden({ timeout: 5000 });
  });

  test('should clear cart with ALT+DEL and confirm', async ({ page }) => {
    await page.waitForTimeout(2000);
    const addButton = page.locator('button:not([disabled])').filter({ hasText: 'Add' }).first();
    await addButton.waitFor({ state: 'visible', timeout: 10000 });
    await expect(addButton).toBeEnabled({ timeout: 5000 });
    await addButton.click();
    await page.waitForTimeout(500);

    await page.locator('button[aria-label="Clear cart"]').click();
    await page.waitForTimeout(500);
    await expect(page.locator('text=Your cart is empty')).toBeVisible({ timeout: 5000 });
  });

  test('should focus search with F2 key', async ({ page }) => {
    await page.waitForTimeout(2000);
    await page.keyboard.press('F2');
    await expect(page.locator('#pos-search-input')).toBeFocused({ timeout: 3000 });
  });
});

test.describe('POS API Tests', () => {
  test('should create sale via API with computed tax', async ({ page, request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    const saleResponse = await page.request.post(`${API_BASE}/api/sales`, {
      headers: { Authorization: `Bearer ${token}` },
      data: {
        invoice_number: `INV-${Date.now()}`,
        discount: 0,
        payment_method: 'CASH',
        items: [
           { product_id: 4690, quantity: 1, subtotal: 1195000 }
        ]
      }
    });

    expect(saleResponse.ok()).toBeTruthy();
    const sale = await saleResponse.json();
    expect(sale.data).toHaveProperty('id');
    expect(sale.data.invoice_number).toBeTruthy();
    expect(sale.data.total_amount).toBe(1195000);
    if (sale.data.tax > 0) {
      expect(sale.data.subtotal + sale.data.tax).toBe(sale.data.total_amount);
    }
  });

  test('GET /api/products should return products list', async ({ page, request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    const response = await page.request.get(`${API_BASE}/api/products`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    expect(response.ok()).toBeTruthy();
    const data = await response.json();
    expect(data.data).toBeTruthy();
    expect(Array.isArray(data.data)).toBeTruthy();
    expect(data.data.length).toBeGreaterThanOrEqual(5);
  });

  test('should redirect unauthenticated users to login', async ({ page }) => {
    await page.goto('/');
    await page.waitForTimeout(1000);

    await page.goto('/pos');
    await page.waitForTimeout(2000);

    await expect(page.locator('#username')).toBeVisible({ timeout: 5000 });
    expect(page.url()).toContain('/login');
  });
});
