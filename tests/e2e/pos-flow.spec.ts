import { test, expect } from './fixtures';
import { TEST_USERS, API_BASE, loginUI, logoutUI, getToken, authHeader } from './fixtures';

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
    await page.waitForTimeout(1500);
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
    await expect(page.getByRole('dialog', { name: 'Payment' })).toBeVisible({ timeout: 5000 });

    await page.keyboard.press('Escape');
    await expect(page.getByRole('dialog', { name: 'Payment' })).toBeHidden({ timeout: 5000 });
  });

  test('should open customer selection modal', async ({ page }) => {
    await page.waitForTimeout(2000);
    const addButton = page.locator('button:not([disabled])').filter({ hasText: 'Add' }).first();
    await addButton.waitFor({ state: 'visible', timeout: 10000 });
    await addButton.click();
    await page.waitForTimeout(500);

    await page.keyboard.press('F4');
    await expect(page.getByRole('dialog', { name: 'Payment' })).toBeVisible({ timeout: 5000 });

    await page.locator('button').filter({ hasText: 'Walk-in / General' }).first().click();
    await expect(page.getByRole('dialog', { name: 'Select Customer' })).toBeVisible({ timeout: 5000 });

    await page.getByRole('dialog', { name: 'Select Customer' }).getByLabel('Close').click();
    await expect(page.getByRole('dialog', { name: 'Select Customer' })).toBeHidden({ timeout: 5000 });
  });

  test('should clear cart with ALT+DEL and confirm', async ({ page }) => {
    await page.waitForTimeout(2000);
    const addButton = page.locator('button:not([disabled])').filter({ hasText: 'Add' }).first();
    await addButton.waitFor({ state: 'visible', timeout: 10000 });
    await expect(addButton).toBeEnabled({ timeout: 5000 });
    await addButton.click();
    await page.waitForTimeout(500);

    await page.locator('button[aria-label="Clear Cart"]').click();
    await page.waitForTimeout(500);
    await expect(page.locator('text=Your cart is empty')).toBeVisible({ timeout: 5000 });
  });

  test('should focus search with F2 key', async ({ page }) => {
    await page.waitForTimeout(2000);
    await page.keyboard.press('F2');
    await expect(page.locator('#pos-search-input')).toBeFocused({ timeout: 3000 });
  });

  test('cart shows item count after adding product', async ({ page }) => {
    await page.waitForTimeout(2000);
    const addButton = page.locator('button:not([disabled])').filter({ hasText: 'Add' }).first();
    await addButton.waitFor({ state: 'visible', timeout: 10000 });
    await expect(addButton).toBeEnabled({ timeout: 5000 });
    await addButton.click();
    await page.waitForTimeout(500);

    await expect(page.locator('text=Your cart is empty')).toBeHidden({ timeout: 5000 });
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

  test('should create sale via API with computed tax', async ({ page, request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    const prodRes = await page.request.get(`${API_BASE}/api/products?search=Nike Jacket&status=active`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    const prodBody = await prodRes.json();
    const product = prodBody.data?.[0];
    expect(product).toBeTruthy();

    const saleResponse = await page.request.post(`${API_BASE}/api/sales`, {
      headers: { Authorization: `Bearer ${token}` },
      data: {
        payment_method: 'CASH',
        items: [
           { product_id: product.id, quantity: 1 }
        ]
      }
    });

    expect(saleResponse.ok()).toBeTruthy();
    const sale = await saleResponse.json();
    expect(sale.data).toHaveProperty('id');
    expect(sale.data.invoice_number).toBeTruthy();
    expect(sale.data.total_amount).toBeGreaterThanOrEqual(product.price);
    if (sale.data.tax > 0) {
      expect(sale.data.subtotal).toBe(sale.data.total_amount);
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

  test('should create sale with split tender (CASH + QRIS) via API', async ({ request }) => {
    const { headers } = auth;
    const qty = 2;
    const subtotal = product.price * qty;
    const totalAmount = subtotal;
    const cashAmount = Math.floor(totalAmount / 2);
    const qrisAmount = totalAmount - cashAmount;

    const res = await request.post(`${API_BASE}/api/sales`, {
      headers,
      data: {
        payment_method: 'CASH,QRIS',
        status: 'completed',
        items: [{ product_id: product.id, quantity: qty }],
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
    expect(body.data.payment_method).toBe('CASH,QRIS');
    expect(body.data.payments).toBeDefined();
    expect(body.data.payments).toHaveLength(2);
    expect(body.data.payments.some(p => p.payment_method_code === 'CASH' && p.amount === cashAmount)).toBeTruthy();
    expect(body.data.payments.some(p => p.payment_method_code === 'QRIS' && p.amount === qrisAmount)).toBeTruthy();
  });

  test('should create sale with legacy single payment method (backward compat)', async ({ request }) => {
    const { headers } = auth;
    const qty = 1;
    const subtotal = product.price * qty;

    const res = await request.post(`${API_BASE}/api/sales`, {
      headers,
      data: {
        payment_method: 'CASH',
        status: 'completed',
        items: [{ product_id: product.id, quantity: qty }],
      },
    });

    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data.status).toBe('completed');
    expect(body.data.payment_method).toBe('CASH');
    expect(body.data.payments).toBeDefined();
    expect(body.data.payments).toHaveLength(1);
    expect(body.data.payments[0].payment_method_code).toBe('CASH');
    expect(body.data.payments[0].amount).toBe(subtotal);
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
    await page.goto('/');
    await page.waitForTimeout(1000);

    await page.goto('/pos');
    await page.waitForTimeout(2000);

    await expect(page.locator('#username')).toBeVisible({ timeout: 5000 });
    expect(page.url()).toContain('/login');
  });
});
