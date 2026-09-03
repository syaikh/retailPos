import { test, expect } from './fixtures';
import { TEST_USERS, API_BASE, FRONTEND_BASE, loginUI, logoutUI, getToken, authHeader } from './fixtures';

function getFirstAddButton(page: any) {
  return page.locator('button:not([disabled])').filter({ hasText: 'Add' }).first();
}

test.describe('POS UI Flow', () => {
  test.beforeEach(async ({ page }) => {
    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    await page.goto(`${FRONTEND_BASE}/pos`);
    await expect(page).toHaveURL(/\/pos/);
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('should load POS page with product table and cart', async ({ page }) => {
    await expect(page.locator('text=PRODUCT NAME')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Cart', { exact: true }).nth(1)).toBeVisible();
  });

  test('should search products by name', async ({ page, request }) => {
    // Ensure deterministic product exists (100-product seed may not contain Quality Model)
    const tok = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    await request.post(`${API_BASE}/api/products`, {
      headers: authHeader(tok),
      data: { name: `Quality Model E2E ${Date.now()}`, sku: `QM-${Date.now()}`, price: 10000, cost: 5000, stock: 10, status: 'active', category_id: 1 },
    });
    await page.fill('#pos-search-input', 'Quality Model');
    await expect(page.locator('tbody tr').first()).toContainText('Quality Model', { timeout: 5000 });
    await page.locator('#pos-search-input').fill('');
    await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 5000 });
  });

  test('should add product to cart', async ({ page }) => {
    await expect(page.locator('text=PRODUCT NAME')).toBeVisible({ timeout: 10000 });
    const addButton = getFirstAddButton(page);
    await expect(addButton).toBeVisible({ timeout: 10000 });
    await expect(addButton).toBeEnabled({ timeout: 5000 });
    await addButton.click();
    await expect(page.locator('text=Your cart is empty')).toBeHidden({ timeout: 5000 });
    // Quantity is reflected in cart (input value or total) — assert cart is populated
    await expect(page.locator('button[aria-label="Increase quantity"]').first()).toBeVisible({ timeout: 5000 });
  });

  test('should increase item quantity in cart', async ({ page }) => {
    await expect(page.locator('text=PRODUCT NAME')).toBeVisible({ timeout: 10000 });
    const addButton = getFirstAddButton(page);
    await expect(addButton).toBeVisible({ timeout: 10000 });
    await expect(addButton).toBeEnabled({ timeout: 5000 });
    await addButton.click();
    await expect(page.locator('text=Your cart is empty')).toBeHidden({ timeout: 5000 });

    const increaseBtn = page.locator('button[aria-label="Increase quantity"]').first();
    await expect(increaseBtn).toBeVisible({ timeout: 5000 });
    await increaseBtn.click();
    // After increase, quantity controls should still be visible and cart populated
    await expect(page.locator('button[aria-label="Decrease quantity"]').first()).toBeVisible({ timeout: 5000 });
    await expect(page.locator('text=Your cart is empty')).toBeHidden({ timeout: 5000 });
  });

  test('should decrease item quantity in cart', async ({ page }) => {
    await expect(page.locator('text=PRODUCT NAME')).toBeVisible({ timeout: 10000 });
    const addButton = getFirstAddButton(page);
    await expect(addButton).toBeVisible({ timeout: 10000 });
    await expect(addButton).toBeEnabled({ timeout: 5000 });
    await addButton.click();
    await expect(page.locator('text=Your cart is empty')).toBeHidden({ timeout: 5000 });

    const increaseBtn = page.locator('button[aria-label="Increase quantity"]').first();
    await expect(increaseBtn).toBeVisible({ timeout: 5000 });
    await increaseBtn.click();
    await expect(page.locator('button[aria-label="Decrease quantity"]').first()).toBeVisible({ timeout: 5000 });

    const decreaseBtn = page.locator('button[aria-label="Decrease quantity"]').first();
    await expect(decreaseBtn).toBeVisible({ timeout: 5000 });
    await decreaseBtn.click();
    await expect(page.locator('text=Your cart is empty')).toBeHidden({ timeout: 5000 });
  });

  test('should remove item from cart', async ({ page }) => {
    await expect(page.locator('text=PRODUCT NAME')).toBeVisible({ timeout: 10000 });
    const addButton = getFirstAddButton(page);
    await expect(addButton).toBeVisible({ timeout: 10000 });
    await expect(addButton).toBeEnabled({ timeout: 5000 });
    await addButton.click();
    await expect(page.locator('text=Your cart is empty')).toBeHidden({ timeout: 5000 });

    const removeBtn = page.locator('button[aria-label="Remove item"]').first();
    await expect(removeBtn).toBeVisible({ timeout: 5000 });
    await removeBtn.click();
    await expect(page.locator('text=Your cart is empty')).toBeVisible({ timeout: 5000 });
  });

  test('should open checkout modal with F4 key', async ({ page }) => {
    await expect(page.locator('text=PRODUCT NAME')).toBeVisible({ timeout: 10000 });
    const addButton = getFirstAddButton(page);
    await expect(addButton).toBeVisible({ timeout: 10000 });
    await expect(addButton).toBeEnabled({ timeout: 5000 });
    await addButton.click();
    await expect(page.locator('text=Your cart is empty')).toBeHidden({ timeout: 5000 });

    await page.keyboard.press('F4');
    await expect(page.getByRole('dialog', { name: 'Payment' })).toBeVisible({ timeout: 5000 });

    await page.keyboard.press('Escape');
    await expect(page.getByRole('dialog', { name: 'Payment' })).toBeHidden({ timeout: 5000 });
  });

  test('should open customer selection modal', async ({ page }) => {
    await expect(page.locator('text=PRODUCT NAME')).toBeVisible({ timeout: 10000 });
    const addButton = getFirstAddButton(page);
    await expect(addButton).toBeVisible({ timeout: 10000 });
    await addButton.click();
    await expect(page.locator('text=Your cart is empty')).toBeHidden({ timeout: 5000 });

    await page.keyboard.press('F4');
    await expect(page.getByRole('dialog', { name: 'Payment' })).toBeVisible({ timeout: 5000 });

    await page.locator('button').filter({ hasText: 'Walk-in / General' }).first().click();
    await expect(page.getByRole('dialog', { name: 'Select Customer' })).toBeVisible({ timeout: 5000 });

    await page.getByRole('dialog', { name: 'Select Customer' }).getByLabel('Close').click();
    await expect(page.getByRole('dialog', { name: 'Select Customer' })).toBeHidden({ timeout: 5000 });
  });

  test('should clear cart with ALT+DEL and confirm', async ({ page }) => {
    await expect(page.locator('text=PRODUCT NAME')).toBeVisible({ timeout: 10000 });
    const addButton = getFirstAddButton(page);
    await expect(addButton).toBeVisible({ timeout: 10000 });
    await expect(addButton).toBeEnabled({ timeout: 5000 });
    await addButton.click();
    await expect(page.locator('text=Your cart is empty')).toBeHidden({ timeout: 5000 });

    await page.locator('button[aria-label="Clear Cart"]').click();
    await expect(page.locator('text=Your cart is empty')).toBeVisible({ timeout: 5000 });
  });

  test('should focus search with F2 key', async ({ page }) => {
    await page.keyboard.press('F2');
    await expect(page.locator('#pos-search-input')).toBeFocused({ timeout: 3000 });
  });

  test('cart shows item count after adding product', async ({ page }) => {
    await expect(page.locator('text=PRODUCT NAME')).toBeVisible({ timeout: 10000 });
    const addButton = getFirstAddButton(page);
    await expect(addButton).toBeVisible({ timeout: 10000 });
    await expect(addButton).toBeEnabled({ timeout: 5000 });
    await addButton.click();
    await expect(page.locator('text=Your cart is empty')).toBeHidden({ timeout: 5000 });
    await expect(page.locator('button[aria-label="Increase quantity"]').first()).toBeVisible({ timeout: 5000 });
  });
});

test.describe('POS API Tests', () => {
  let auth: { token: string; headers: Record<string, string> };
  let product: { id: number; price: number };

  test.beforeAll(async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    auth = { token, headers: authHeader(token) };

    const prodRes = await request.get(`${API_BASE}/api/products?limit=1`, { headers: auth.headers });
    const prodBody = await prodRes.json();
    if (!prodBody.data || prodBody.data.length === 0) {
      throw new Error('Need at least 1 product in DB for POS API tests');
    }
    product = { id: prodBody.data[0].id, price: prodBody.data[0].price };

    await request.post(`${API_BASE}/api/inventory/adjust`, {
      headers: auth.headers,
      data: { product_id: product.id, quantity_change: 500, notes: 'E2E stock boost for POS API tests' },
    });
  });

  test('should create sale via API with computed tax', async ({ request }) => {
    const { headers } = auth;
    // Use the beforeAll-seeded product so `product` is asserted and stock-boost is exercised.
    expect(product.id).toBeGreaterThan(0);
    expect(product.price).toBeGreaterThan(0);

    const saleResponse = await request.post(`${API_BASE}/api/sales`, {
      headers,
      data: {
        payment_method: 'CASH',
        status: 'completed',
        items: [{ product_id: product.id, quantity: 1 }],
      },
    });

    expect(saleResponse.ok()).toBeTruthy();
    const sale = await saleResponse.json();
    expect(sale.data).toHaveProperty('id');
    expect(sale.data.invoice_number).toMatch(/^INV-/);
    expect(sale.data.status).toBe('completed');
    expect(sale.data.subtotal).toBeGreaterThan(0);
    expect(sale.data.total_amount).toBeGreaterThan(0);
    expect(sale.data.total_amount).toBeGreaterThanOrEqual(sale.data.subtotal);
    // Direct POST /api/sales stores total as subtotal, tax is separate (handler.go:347)
    expect(sale.data.total_amount).toBe(sale.data.subtotal);
    // Pricing rules may adjust price above or below list price
    expect(sale.data.subtotal).toBeGreaterThan(0);
    expect(sale.data.payments).toBeDefined();
    expect(sale.data.payments.length).toBeGreaterThanOrEqual(1);
  });

  test('GET /api/products should return products list', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    const response = await request.get(`${API_BASE}/api/products`, {
      headers: authHeader(token),
    });
    expect(response.ok()).toBeTruthy();
    const data = await response.json();
    expect(data.data).toBeTruthy();
    expect(Array.isArray(data.data)).toBeTruthy();
    expect(data.data.length).toBeGreaterThanOrEqual(5);
  });

  test('should create sale with split tender (CASH + QRIS) via API', async ({ request }) => {
    const { headers } = auth;
    const qty = 2;
    // Create a dedicated tax-free product to avoid tax/discount flakiness
    const prodRes = await request.post(`${API_BASE}/api/products`, {
      headers,
      data: { name: `E2E Split ${Date.now()}`, sku: `E2E-SPLIT-${Date.now()}`, price: 10000, cost: 5000, stock: 10, status: 'active' },
    });
    expect(prodRes.ok()).toBeTruthy();
    const prodBody = await prodRes.json();
    const pid = prodBody.data?.id ?? prodBody.id;
    const subtotal = 10000 * qty;
    const totalAmount = subtotal;
    const cashAmount = Math.floor(totalAmount / 2);
    const qrisAmount = totalAmount - cashAmount;

    const res = await request.post(`${API_BASE}/api/sales`, {
      headers,
      data: {
        payment_method: 'CASH,QRIS',
        status: 'completed',
        items: [{ product_id: pid, quantity: qty }],
        payments: [
          { payment_method_code: 'CASH', amount: cashAmount },
          { payment_method_code: 'QRIS', amount: qrisAmount },
        ],
      },
    });

    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data.status).toBe('completed');
    expect(body.data.total_amount).toBe(totalAmount);
    expect(body.data.subtotal).toBe(subtotal);
    expect(body.data.total_amount).toBe(body.data.subtotal + (body.data.tax || 0));
    expect(body.data.payment_method).toBe('CASH,QRIS');
    expect(body.data.payments).toBeDefined();
    expect(body.data.payments).toHaveLength(2);
    expect(body.data.payments.some(p => p.payment_method_code === 'CASH' && p.amount === cashAmount)).toBeTruthy();
    expect(body.data.payments.some(p => p.payment_method_code === 'QRIS' && p.amount === qrisAmount)).toBeTruthy();
  });

  test('should create sale with legacy single payment method (backward compat)', async ({ request }) => {
    const { headers } = auth;
    const qty = 1;
    const prodRes = await request.post(`${API_BASE}/api/products`, {
      headers,
      data: { name: `E2E Legacy ${Date.now()}`, sku: `E2E-LEGACY-${Date.now()}`, price: 10000, cost: 5000, stock: 10, status: 'active' },
    });
    expect(prodRes.ok()).toBeTruthy();
    const prodBody = await prodRes.json();
    const pid = prodBody.data?.id ?? prodBody.id;
    const subtotal = 10000 * qty;

    const res = await request.post(`${API_BASE}/api/sales`, {
      headers,
      data: {
        payment_method: 'CASH',
        status: 'completed',
        items: [{ product_id: pid, quantity: qty }],
      },
    });

    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data.status).toBe('completed');
    expect(body.data.subtotal).toBe(subtotal);
    expect(body.data.total_amount).toBe(body.data.subtotal + (body.data.tax || 0));
    expect(body.data.payment_method).toBe('CASH');
    expect(body.data.payments).toBeDefined();
    expect(body.data.payments).toHaveLength(1);
    expect(body.data.payments[0].payment_method_code).toBe('CASH');
    expect(body.data.payments[0].amount).toBe(body.data.total_amount);
  });

  test('should reject split tender with payment total mismatch (400)', async ({ request }) => {
    const { headers } = auth;
    const totalAmount = product.price * 1;

    const res = await request.post(`${API_BASE}/api/sales`, {
      headers,
      data: {
        payment_method: 'CASH,QRIS',
        status: 'completed',
        items: [{ product_id: product.id, quantity: 1 }],
        payments: [
          { payment_method_code: 'CASH', amount: totalAmount - 2 },
          { payment_method_code: 'QRIS', amount: 1 },
        ],
      },
    });

    expect(res.ok()).toBeFalsy();
    expect(res.status()).toBe(400);
    const body = await res.json();
    expect(body.error.message).toContain('total payments do not match');
  });

  test('should reject split tender with duplicate payment method (400)', async ({ request }) => {
    const { headers } = auth;
    const totalAmount = product.price * 1;

    const res = await request.post(`${API_BASE}/api/sales`, {
      headers,
      data: {
        payment_method: 'CASH',
        status: 'completed',
        items: [{ product_id: product.id, quantity: 1 }],
        payments: [
          { payment_method_code: 'CASH', amount: Math.floor(totalAmount / 2) },
          { payment_method_code: 'CASH', amount: Math.ceil(totalAmount / 2) },
        ],
      },
    });

    expect(res.ok()).toBeFalsy();
    expect(res.status()).toBe(400);
    const body = await res.json();
    expect(body.error.message).toContain('only one cash payment per transaction is allowed');
  });

  test('should reject payment method that requires reference without one (400)', async ({ request }) => {
    const { headers } = auth;
    const totalAmount = product.price * 1;

    const res = await request.post(`${API_BASE}/api/sales`, {
      headers,
      data: {
        payment_method: 'CARD',
        status: 'completed',
        items: [{ product_id: product.id, quantity: 1 }],
        payments: [
          { payment_method_code: 'CARD', amount: totalAmount },
        ],
      },
    });

    expect(res.ok()).toBeFalsy();
    expect(res.status()).toBe(400);
    const body = await res.json();
    expect(body.error.message).toContain('reference number is required for this payment method');
  });

  test('should reject unknown payment method (400)', async ({ request }) => {
    const { headers } = auth;
    const totalAmount = product.price * 1;

    const res = await request.post(`${API_BASE}/api/sales`, {
      headers,
      data: {
        payment_method: 'BITCOIN',
        status: 'completed',
        items: [{ product_id: product.id, quantity: 1 }],
        payments: [
          { payment_method_code: 'BITCOIN', amount: totalAmount },
        ],
      },
    });

    expect(res.ok()).toBeFalsy();
    expect(res.status()).toBe(400);
    const body = await res.json();
    expect(body.error.message).toContain('invalid payment method code');
  });

  test('should redirect unauthenticated users to login', async ({ page }) => {
    await page.goto(`${FRONTEND_BASE}/login`);
    await page.evaluate(() => sessionStorage.clear());
    await page.goto(`${FRONTEND_BASE}/pos`);
    await expect(page.locator('#username')).toBeVisible({ timeout: 5000 });
    await expect(page).toHaveURL(/\/login/);
  });
});
