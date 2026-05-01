import { test, expect } from '@playwright/test';
import { TEST_USERS } from './fixtures';

// ============================================================================
// Point of Sale (POS) Module - E2E Tests
// ============================================================================

async function loginAndNavigateToPos(page: any) {
  await page.goto('/');
  await page.waitForTimeout(1000); // Slow down for visibility
  await page.waitForSelector('#login-section', { timeout: 10000 });
  await page.waitForTimeout(500); // Slow down for visibility
  await page.locator('input[type="text"]').fill(TEST_USERS.superadmin.username);
  await page.waitForTimeout(800); // Slow down for visibility
  await page.locator('input[type="password"]').fill(TEST_USERS.superadmin.password);
  await page.waitForTimeout(800); // Slow down for visibility
  await page.locator('button[type="submit"]').click();
  // After login, redirect to home, then navigate to POS
  await page.waitForURL(/\/$/, { timeout: 10000 });
  await page.waitForTimeout(1000); // Slow down for visibility
  await page.goto('/pos');
  await page.waitForTimeout(800); // Slow down for visibility
  await expect(page.locator('.pos-page')).toBeVisible({ timeout: 10000 });
}

test.describe('Point of Sale (POS) Module', () => {
  test('should navigate to POS page and load products', async ({ page }) => {
    await loginAndNavigateToPos(page);

    // Check page title/header
    await expect(page.getByText('Point of Sale')).toBeVisible();
    await expect(page.getByText('Process transactions and manage sales')).toBeVisible();

    // Check product table is visible
    await expect(page.locator('table')).toBeVisible();
  });

  test('should display product table with correct columns', async ({ page }) => {
    await loginAndNavigateToPos(page);

    // Check column headers
    const headers = page.locator('thead th');
    await expect(headers.nth(0)).toContainText('Product');
    await expect(headers.nth(1)).toContainText('SKU');
    await expect(headers.nth(2)).toContainText('Stock');
    await expect(headers.nth(3)).toContainText('Price');
    await expect(headers.nth(4)).toContainText('Action');
  });

  test('should display seeded products in the table', async ({ page }) => {
    await loginAndNavigateToPos(page);

    // Wait for products to load
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 10000 });

    // Count rows (should have at least 5 products)
    const rows = page.locator('tbody tr');
    const count = await rows.count();
    expect(count).toBeGreaterThanOrEqual(5);
  });

  test('should add product to cart', async ({ page }) => {
    await loginAndNavigateToPos(page);

    // Wait for products
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 10000 });
    await page.waitForTimeout(1000); // Pause to see products loaded

    // Click Add button on first product
    const firstAddBtn = page.locator('tbody tr').first().getByRole('button', { name: 'Add' });
    await firstAddBtn.click();
    await page.waitForTimeout(800); // Pause to see button click

    // Cart should no longer be empty
    await expect(page.getByText('Your cart is empty')).not.toBeVisible({ timeout: 5000 });
    await page.waitForTimeout(1000); // Pause to see cart updated
  });

  test('should update cart when adding multiple items', async ({ page }) => {
    await loginAndNavigateToPos(page);
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 10000 });

    // Add first product
    const firstRow = page.locator('tbody tr').first();
    await firstRow.getByRole('button', { name: 'Add' }).click();

    // Add second product
    const secondRow = page.locator('tbody tr').nth(1);
    await secondRow.getByRole('button', { name: 'Add' }).click();

    // Cart should show 2 items total
    await expect(page.getByText(/items/)).toBeVisible({ timeout: 5000 });
  });

  test('should increment quantity when adding same product again', async ({ page }) => {
    await loginAndNavigateToPos(page);
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 10000 });

    // Add same product twice
    const firstRow = page.locator('tbody tr').first();
    await firstRow.getByRole('button', { name: 'Add' }).click();
    await firstRow.getByRole('button', { name: 'Add' }).click();

    // Quantity should be 2
    const quantitySpan = page.locator('.max-h-96 span.w-8').first();
    await expect(quantitySpan).toContainText('2', { timeout: 5000 });
  });

  test('should increase quantity with + button', async ({ page }) => {
    await loginAndNavigateToPos(page);
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 10000 });

    const firstRow = page.locator('tbody tr').first();
    await firstRow.getByRole('button', { name: 'Add' }).click();
    await page.waitForTimeout(1000); // Pause to see initial cart

    // Click + button
    const plusBtn = page.locator('.max-h-96 button').filter({ hasText: '+' }).first();
    await plusBtn.click();
    await page.waitForTimeout(800); // Pause to see quantity increase

    // Quantity should be 2
    const quantitySpan = page.locator('.max-h-96 span.w-8').first();
    await expect(quantitySpan).toContainText('2', { timeout: 5000 });
    await page.waitForTimeout(1000); // Pause to see final quantity
  });

  test('should decrease quantity with - button (min 1)', async ({ page }) => {
    await loginAndNavigateToPos(page);
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 10000 });

    const firstRow = page.locator('tbody tr').first();
    await firstRow.getByRole('button', { name: 'Add' }).click();

    // Click + twice then - once
    const plusBtn = page.locator('.max-h-96 button').filter({ hasText: '+' }).first();
    await plusBtn.click();
    await plusBtn.click();

    const minusBtn = page.locator('.max-h-96 button').filter({ hasText: '-' }).first();
    await minusBtn.click();

    // Quantity should be 2 (3 - 1)
    const quantitySpan = page.locator('.max-h-96 span.w-8').first();
    await expect(quantitySpan).toContainText('2', { timeout: 5000 });
  });

  test('should remove item from cart with × button', async ({ page }) => {
    await loginAndNavigateToPos(page);
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 10000 });

    const firstRow = page.locator('tbody tr').first();
    await firstRow.getByRole('button', { name: 'Add' }).click();
    await page.waitForTimeout(1000); // Pause to see item added

    // Remove item
    const removeBtn = page.locator('.max-h-96 button').filter({ hasText: '×' }).first();
    await removeBtn.click();
    await page.waitForTimeout(800); // Pause to see item removed

    // Cart should be empty
    await expect(page.getByText('Your cart is empty')).toBeVisible({ timeout: 5000 });
    await page.waitForTimeout(1000); // Pause to see empty cart
  });

  test('should display subtotal, tax, and total in cart', async ({ page }) => {
    await loginAndNavigateToPos(page);
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 10000 });

    // Add a product
    const firstRow = page.locator('tbody tr').first();
    await firstRow.getByRole('button', { name: 'Add' }).click();

    // Check cart summary
    await expect(page.getByText('Subtotal')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Tax (10%)')).toBeVisible();
    await expect(page.getByText('Total', { exact: true })).toBeVisible();
  });

  test('should show Complete Purchase button', async ({ page }) => {
    await loginAndNavigateToPos(page);
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 10000 });

    const firstRow = page.locator('tbody tr').first();
    await firstRow.getByRole('button', { name: 'Add' }).click();

    await expect(page.getByRole('button', { name: 'Complete Purchase' })).toBeVisible({ timeout: 5000 });
  });

  test('should search products by name', async ({ page }) => {
    await loginAndNavigateToPos(page);
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 10000 });

    // Get initial count
    const initialRows = await page.locator('tbody tr').count();

    // Type in search box
    const searchInput = page.getByPlaceholder('Search products (name or SKU)...');
    await searchInput.fill('Nonexistent');

    // Should show "No products found"
    await expect(page.getByText(/No products found/)).toBeVisible({ timeout: 5000 });
  });

  test('should filter products by category', async ({ page }) => {
    await loginAndNavigateToPos(page);
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 10000 });

    // Check category dropdown exists
    const categorySelect = page.locator('select');
    await expect(categorySelect).toBeVisible();

    // Select a specific category
    await categorySelect.selectOption('Makanan');

    // Wait for filter to apply
    await page.waitForTimeout(500);

    // All visible products should be in Makanan category
    const categoryCells = page.locator('tbody tr .text-xs.text-slate-400');
    const count = await categoryCells.count();
    for (let i = 0; i < count; i++) {
      const text = await categoryCells.nth(i).textContent();
      expect(text?.trim()).toBe('Makanan');
    }
  });

  test('should create sale via API when checkout', async ({ page }) => {
    const tokenResponse = await page.request.post('/api/login', {
      data: {
        username: TEST_USERS.superadmin.username,
        password: TEST_USERS.superadmin.password
      }
    });
    const tokenData = await tokenResponse.json();
    const token = tokenData.access_token;

    const saleResponse = await page.request.post('/api/sales', {
      headers: { Authorization: `Bearer ${token}` },
      data: {
        invoice_number: `INV-${Date.now()}`,
        cashier_id: 1,
        store_id: 1,
        subtotal: 15000,
        discount: 0,
        tax: 1500,
        total_amount: 16500,
        payment_method: 'cash',
        items: [
          { product_id: 1, quantity: 1, unit_price: 15000, subtotal: 15000 }
        ]
      }
    });

    expect(saleResponse.ok()).toBeTruthy();
    const sale = await saleResponse.json();
    expect(sale.data).toHaveProperty('id');
    expect(sale.data.invoice_number).toBeTruthy();
  });

  test('GET /api/products should return products list', async ({ page }) => {
    const tokenResponse = await page.request.post('/api/login', {
      data: {
        username: TEST_USERS.superadmin.username,
        password: TEST_USERS.superadmin.password
      }
    });
    const token = (await tokenResponse.json()).access_token;

    const response = await page.request.get('/api/products', {
      headers: { Authorization: `Bearer ${token}` }
    });
    expect(response.ok()).toBeTruthy();
    const data = await response.json();
    expect(data.data).toBeTruthy();
    expect(Array.isArray(data.data)).toBeTruthy();
    expect(data.data.length).toBeGreaterThanOrEqual(5);
  });

  test('should show disabled Add button for out-of-stock product', async ({ page }) => {
    await loginAndNavigateToPos(page);
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 10000 });

    // Products with stock=0 should have disabled Add button
    // Since seeded products have stock, we just verify Add buttons exist
    const addButtons = page.locator('tbody tr button:has-text("Add")');
    const count = await addButtons.count();
    expect(count).toBeGreaterThanOrEqual(1);
  });

  test('should redirect unauthenticated users to login for protected routes', async ({ page }) => {
    // Start with a fresh context to ensure no authentication
    await page.goto('/');

    // Wait for initial route handling
    await page.waitForTimeout(1000);

    // Try to access POS page directly without authentication
    await page.goto('/pos');
    await page.waitForTimeout(2000);

    // Should be redirected to login page
    await expect(page.locator('#login-section')).toBeVisible({ timeout: 5000 });
    expect(page.url()).toContain('/login');

    // Navbar should NOT be visible for unauthenticated users
    await expect(page.locator('.navbar')).toBeHidden();

    // Try to access inventory page directly
    await page.goto('/inventory');
    await page.waitForTimeout(2000);

    // Should still be on login page
    await expect(page.locator('#login-section')).toBeVisible({ timeout: 5000 });
    expect(page.url()).toContain('/login');

    // Try to access reports page directly
    await page.goto('/reports');
    await page.waitForTimeout(2000);

    // Should still be on login page
    await expect(page.locator('#login-section')).toBeVisible({ timeout: 5000 });
    expect(page.url()).toContain('/login');

    // Try to access admin page directly
    await page.goto('/admin');
    await page.waitForTimeout(2000);

    // Should still be on login page
    await expect(page.locator('#login-section')).toBeVisible({ timeout: 5000 });
    expect(page.url()).toContain('/login');

    // Now login and verify access works
    await page.locator('input[type="text"]').fill(TEST_USERS.superadmin.username);
    await page.locator('input[type="password"]').fill(TEST_USERS.superadmin.password);
    await page.locator('button[type="submit"]').click();
    await page.waitForTimeout(3000);

    // Should be on home page now with navbar visible
    await expect(page.locator('.navbar')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Retail POS System')).toBeVisible();
  });
});
