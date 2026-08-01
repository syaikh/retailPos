import { test, expect } from '@playwright/test';
import { API_BASE, FRONTEND_BASE, getToken, authHeader } from './fixtures';

const API_URLS = {
  PRODUCTS: `${API_BASE}/api/products`,
  CART: `${API_BASE}/api/pos/cart`,
  PRICING_RULES: `${API_BASE}/api/pricing-rules`,
  SALES: `${API_BASE}/api/sales`,
};

const BASE_PRICE = 10000;
const PRODUCT_NAME = 'Test Price Consistency Product';

function cartItemUrl(cartId: number) {
  return `${API_URLS.CART}/${cartId}`;
}

function checkoutUrl(cartId: number) {
  return `${API_URLS.CART}/${cartId}/checkout`;
}

function todayJakarta(): string {
  const now = new Date(Date.now() + 7 * 3600 * 1000);
  return now.toISOString().slice(0, 10);
}

// Ensure a fresh, empty open cart exists for the cashier. Any existing open
// cart (left over from a previous scenario) is parked first so we never pick
// up stale items.
async function ensureFreshCart(token: string, request: any): Promise<number> {
  const openRes = await request.get(`${API_URLS.CART}`, {
    headers: authHeader(token),
  });
  if (openRes.ok()) {
    const openCart = (await openRes.json()).data;
    if (openCart && openCart.id) {
      await request.post(`${API_URLS.CART}/${openCart.id}/hold`, {
        headers: authHeader(token),
      });
    }
  }
  const createRes = await request.post(`${API_URLS.CART}`, {
    headers: authHeader(token),
    data: { store_id: 1, shift_id: 1, customer_id: 1 },
  });
  expect(createRes.ok()).toBeTruthy();
  return (await createRes.json()).data.id;
}

async function addItem(token: string, request: any, productId: number, quantity = 1) {
  const res = await request.post(`${API_URLS.CART}/items`, {
    headers: authHeader(token),
    data: { product_id: productId, quantity },
  });
  expect(res.ok()).toBeTruthy();
  return (await res.json()).data;
}

async function getCart(token: string, request: any, cartId: number) {
  const res = await request.get(cartItemUrl(cartId), {
    headers: authHeader(token),
  });
  expect(res.ok()).toBeTruthy();
  return (await res.json()).data;
}

async function updateProductPrice(token: string, request: any, productId: number, sku: string, price: number) {
  const res = await request.put(`${API_URLS.PRODUCTS}/${productId}`, {
    headers: authHeader(token),
    data: {
      name: PRODUCT_NAME,
      sku,
      price,
      cost: 5000,
      stock: 100,
      status: 'active',
    },
  });
  expect(res.ok()).toBeTruthy();
}

async function createPromoRule(token: string, request: any, productId: number, method: string, value: number) {
  const res = await request.post(`${API_URLS.PRICING_RULES}`, {
    headers: authHeader(token),
    data: {
      product_id: productId,
      pricing_type: 'promotion',
      pricing_method: method,
      pricing_value: value,
      name: `E2E Promo ${Date.now()}`,
      minimum_quantity: 1,
      priority: 0,
      is_active: true,
    },
  });
  expect(res.ok()).toBeTruthy();
  return (await res.json()).data;
}

async function setRuleActive(token: string, request: any, rule: any, isActive: boolean) {
  const res = await request.put(`${API_URLS.PRICING_RULES}/${rule.id}`, {
    headers: authHeader(token),
    data: {
      product_id: rule.product_id,
      pricing_type: rule.pricing_type,
      pricing_method: rule.pricing_method,
      pricing_value: rule.pricing_value,
      name: rule.name,
      minimum_quantity: rule.minimum_quantity,
      priority: rule.priority,
      is_active: isActive,
    },
  });
  expect(res.ok()).toBeTruthy();
}

// Add a specific product through the POS UI by searching for its SKU first.
async function addProductByUI(page: any, sku: string) {
  await page.fill('#pos-search-input', sku);
  await page.waitForTimeout(1200);
  const row = page.locator('tr', { hasText: sku }).first();
  await row.locator('button', { hasText: 'Add' }).click();
}

// Pre-authenticate the POS UI by injecting a valid token into sessionStorage
// before the app boots, then navigate straight to /pos. This avoids the shared
// loginUI() reload path (which waits on the page 'load' event and can time out
// on memory-constrained machines).
async function loginViaSessionStorage(page: any, token: string) {
  await page.addInitScript((tok: string) => {
    sessionStorage.setItem('access_token', tok);
  }, token);
  await page.goto(`${FRONTEND_BASE}/pos`, { waitUntil: 'domcontentloaded' });
  await page.waitForSelector('#pos-search-input', { state: 'visible', timeout: 15000 });
}

test.describe('Price Consistency During Active Transactions', () => {
  let productId: number;
  let productSku: string;

  test.beforeAll(async ({ request }) => {
    const token = await getToken(request);

    const res = await request.post(`${API_URLS.PRODUCTS}`, {
      headers: authHeader(token),
      data: {
        name: PRODUCT_NAME,
        sku: `TEST-PRICE-${Date.now()}`,
        price: BASE_PRICE,
        cost: 5000,
        stock: 100,
        status: 'active',
      },
    });

    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data).toBeTruthy();
    productId = body.data.id;
    productSku = body.data.sku;
  });

  test.afterEach(async ({ request }) => {
    if (!productId) return;
    const token = await getToken(request);
    // Clean up any pricing rules created for the shared test product so no
    // active rule leaks into a later scenario.
    const listRes = await request.get(`${API_URLS.PRICING_RULES}?product_id=${productId}`, {
      headers: authHeader(token),
    });
    if (listRes.ok()) {
      const rules = (await listRes.json()).data || [];
      for (const rule of rules) {
        await request.delete(`${API_URLS.PRICING_RULES}/${rule.id}`, {
          headers: authHeader(token),
        });
      }
    }
    // Restore the master price for the next test.
    await updateProductPrice(token, request, productId, productSku, BASE_PRICE);
  });

  // E2E-01 — price does not change mid-transaction (BR-01/02)
  test('E2E-01: should preserve snapshot price in cart after product price change', async ({ request }) => {
    const token = await getToken(request);

    const cartId = await ensureFreshCart(token, request);

    const addRes = await request.post(`${API_URLS.CART}/items`, {
      headers: authHeader(token),
      data: { product_id: productId, quantity: 1 },
    });
    expect(addRes.ok()).toBeTruthy();

    const cartDetail = await request.get(cartItemUrl(cartId), {
      headers: authHeader(token),
    });
    expect(cartDetail.ok()).toBeTruthy();
    const items = (await cartDetail.json()).data.items;
    const item = items.find((i: any) => i.product_id === productId);
    expect(item).toBeDefined();
    expect(item.unit_price).toBe(BASE_PRICE);
    expect(item.snapshot_created_at).toBeTruthy();

    await updateProductPrice(token, request, productId, productSku, 15000);

    const updatedCart = await request.get(cartItemUrl(cartId), {
      headers: authHeader(token),
    });
    expect(updatedCart.ok()).toBeTruthy();
    const updatedItems = (await updatedCart.json()).data.items;
    const updatedItem = updatedItems.find((i: any) => i.product_id === productId);
    expect(updatedItem).toBeDefined();
    expect(updatedItem.unit_price).toBe(BASE_PRICE);
    expect(updatedItem.snapshot_created_at).toBeTruthy();
  });

  // Checkout uses the item snapshot verbatim.
  test('E2E-01b: should use snapshot prices on checkout', async ({ request }) => {
    const token = await getToken(request);

    const cartId = await ensureFreshCart(token, request);

    await addItem(token, request, productId, 2);

    const cartData = await getCart(token, request, cartId);
    const cartItem = cartData.items.find((i: any) => i.product_id === productId);
    expect(cartItem).toBeDefined();
    const expectedUnitPrice = cartItem.unit_price;
    const expectedSubtotal = cartData.total_amount;

    const checkoutRes = await request.post(checkoutUrl(cartId), {
      headers: authHeader(token),
      data: {
        payments: [{ payment_method_code: 'CASH', amount: expectedSubtotal }],
      },
    });
    expect(checkoutRes.ok()).toBeTruthy();
    const sale = (await checkoutRes.json()).data;
    expect(sale.status).toBe('completed');

    const saleItem = sale.items.find((i: any) => i.product_id === productId);
    expect(saleItem).toBeDefined();
    expect(saleItem.unit_price).toBe(expectedUnitPrice);
    expect(saleItem.quantity).toBe(2);
    expect(saleItem.subtotal).toBe(expectedSubtotal);
  });

  // E2E-02 — hold does not refresh price; recall resumes with snapshot (BR-05, Edge #1)
  test('E2E-02: hold then recall keeps the snapshot price and never re-resolves', async ({ page, request }) => {
    const token = await getToken(request);

    const cartId = await ensureFreshCart(token, request);
    await addItem(token, request, productId, 1);

    await loginViaSessionStorage(page, token);
    await expect(page.locator('text=Your cart is empty')).toBeHidden({ timeout: 10000 });

    const resolveRequests: string[] = [];
    page.on('request', req => {
      if (req.method() === 'POST' && req.url().includes('/api/pricing/resolve')) {
        resolveRequests.push(req.url());
      }
    });

    // F6 → hold sale via UI.
    await page.keyboard.press('F6');
    await expect(page.locator('text=Your cart is empty')).toBeVisible({ timeout: 10000 });

    // Master data price change while the sale is held.
    await updateProductPrice(token, request, productId, productSku, 8000);

    // F5 → open parked sales modal and recall our cart.
    await page.keyboard.press('F5');
    await expect(page.locator('text=Held Sales')).toBeVisible({ timeout: 5000 });
    const recallButton = page
      .locator('div:has(button[data-action="recall"])', { hasText: `Cart #${cartId}` })
      .locator('button[data-action="recall"]')
      .first();
    await recallButton.click();
    await expect(page.locator('text=Your cart is empty')).toBeHidden({ timeout: 5000 });

    // Recalled cart must keep the snapshot price and must not have triggered resolve.
    const cart = await getCart(token, request, cartId);
    expect(cart.items.length).toBe(1);
    expect(cart.items[0].unit_price).toBe(BASE_PRICE);
    expect(resolveRequests.length).toBe(0);

    // Checkout consumes the snapshot price.
    const checkoutRes = await request.post(checkoutUrl(cartId), {
      headers: authHeader(token),
      data: { payments: [{ payment_method_code: 'CASH', amount: cart.total_amount }] },
    });
    expect(checkoutRes.ok()).toBeTruthy();
    const sale = (await checkoutRes.json()).data;
    expect(sale.status).toBe('completed');
    const saleItem = sale.items.find((i: any) => i.product_id === productId);
    expect(saleItem).toBeDefined();
    expect(saleItem.unit_price).toBe(BASE_PRICE);
  });

  // E2E-03 — item added after resume uses latest master price (BR-06, Edge #2)
  test('E2E-03: new item after resume uses the latest master price', async ({ page, request }) => {
    const token = await getToken(request);
    const cartId = await ensureFreshCart(token, request);

    await addItem(token, request, productId, 1);
    await updateProductPrice(token, request, productId, productSku, 8000);
    await request.post(`${API_URLS.CART}/${cartId}/hold`, { headers: authHeader(token) });
    await request.post(`${API_URLS.CART}/${cartId}/resume`, { headers: authHeader(token) });
    await addItem(token, request, productId, 1);

    const cart = await getCart(token, request, cartId);
    expect(cart.items.length).toBe(2);
    expect(cart.items[0].unit_price).toBe(BASE_PRICE);
    expect(cart.items[1].unit_price).toBe(8000);
    expect(cart.subtotal).toBe(BASE_PRICE + 8000);

    // UI reflects two cart rows with the two different prices.
    await loginViaSessionStorage(page, token);
    await expect(page.locator('input[type="number"]')).toHaveCount(2, { timeout: 10000 });
  });

  // E2E-04 — quantity change does not alter the snapshot (BR-07)
  test('E2E-04: changing quantity does not change the snapshot price', async ({ request }) => {
    const token = await getToken(request);
    const cartId = await ensureFreshCart(token, request);

    await addItem(token, request, productId, 1);
    const cartBefore = await getCart(token, request, cartId);
    const item = cartBefore.items.find((i: any) => i.product_id === productId);
    const snapshotAt = item.snapshot_created_at;

    const patchRes = await request.patch(`${API_URLS.CART}/items/${item.id}`, {
      headers: authHeader(token),
      data: { quantity: 3 },
    });
    expect(patchRes.ok()).toBeTruthy();

    const cartAfter = await getCart(token, request, cartId);
    const updated = cartAfter.items.find((i: any) => i.product_id === productId);
    expect(updated.quantity).toBe(3);
    expect(updated.unit_price).toBe(BASE_PRICE);
    expect(updated.subtotal).toBe(BASE_PRICE * 3);
    expect(updated.snapshot_created_at).toBe(snapshotAt);
  });

  // E2E-05 — void then rescan creates a fresh snapshot with latest price (BR-08, Edge #4)
  test('E2E-05: voiding an item then re-adding uses the latest price', async ({ request }) => {
    const token = await getToken(request);
    const cartId = await ensureFreshCart(token, request);

    await addItem(token, request, productId, 1);
    const cart1 = await getCart(token, request, cartId);
    const itemId = cart1.items.find((i: any) => i.product_id === productId).id;

    const delRes = await request.delete(`${API_URLS.CART}/items/${itemId}`, {
      headers: authHeader(token),
    });
    expect(delRes.ok()).toBeTruthy();

    await updateProductPrice(token, request, productId, productSku, 8000);
    await addItem(token, request, productId, 1);

    const cart = await getCart(token, request, cartId);
    expect(cart.items.length).toBe(1);
    expect(cart.items[0].unit_price).toBe(8000);
  });

  // E2E-06 — promo activated mid-transaction does not affect existing items (BR-09, Edge #5)
  test('E2E-06: promo activated mid-transaction only applies to new items', async ({ request }) => {
    const token = await getToken(request);
    const cartId = await ensureFreshCart(token, request);

    await addItem(token, request, productId, 1);
    const rule = await createPromoRule(token, request, productId, 'discount_percent', 10);
    await addItem(token, request, productId, 1);

    const cart = await getCart(token, request, cartId);
    expect(cart.items.length).toBe(2);
    const first = cart.items[0];
    expect(first.unit_price).toBe(BASE_PRICE);
    expect(first.pricing_type).toBe('default');
    const second = cart.items[1];
    expect(second.unit_price).toBe(9000);
    expect(second.original_price).toBe(BASE_PRICE);
    expect(second.pricing_type).toBe('promotion');
    expect(second.pricing_rule_name).toBe(rule.name);
  });

  // E2E-07 — promo active before scan applies; deactivating only affects new items (Edge #6)
  test('E2E-07: deactivating a promo mid-transaction only affects new items', async ({ request }) => {
    const token = await getToken(request);
    const rule = await createPromoRule(token, request, productId, 'discount_percent', 10);
    const cartId = await ensureFreshCart(token, request);

    await addItem(token, request, productId, 1);
    await setRuleActive(token, request, rule, false);
    await addItem(token, request, productId, 1);

    const cart = await getCart(token, request, cartId);
    expect(cart.items.length).toBe(2);
    expect(cart.items[0].unit_price).toBe(9000);
    expect(cart.items[0].pricing_type).toBe('promotion');
    expect(cart.items[1].unit_price).toBe(BASE_PRICE);
    expect(cart.items[1].pricing_type).toBe('default');
  });

  // E2E-08 — repeated admin price changes only affect items added afterwards (Edge #7)
  test('E2E-08: repeated price changes only affect items added afterwards', async ({ request }) => {
    const token = await getToken(request);
    const cartId = await ensureFreshCart(token, request);

    await addItem(token, request, productId, 1);
    await updateProductPrice(token, request, productId, productSku, 8000);
    await updateProductPrice(token, request, productId, productSku, 9000);
    await updateProductPrice(token, request, productId, productSku, 7500);
    await addItem(token, request, productId, 1);

    const cart = await getCart(token, request, cartId);
    expect(cart.items.length).toBe(2);
    expect(cart.items[0].unit_price).toBe(BASE_PRICE);
    expect(cart.items[1].unit_price).toBe(7500);
  });

  // E2E-09 — architecture regression: no client re-resolve after adding items
  test('E2E-09: no POST /api/pricing/resolve after adding items', async ({ page, request }) => {
    const token = await getToken(request);

    const cartId = await ensureFreshCart(token, request);
    await loginViaSessionStorage(page, token);

    const resolveRequests: string[] = [];
    page.on('request', req => {
      if (req.method() === 'POST' && req.url().includes('/api/pricing/resolve')) {
        resolveRequests.push(req.url());
      }
    });

    await addProductByUI(page, productSku);
    await expect(page.locator('text=Your cart is empty')).toBeHidden({ timeout: 5000 });
    await page.waitForTimeout(500);
    const resolveCountAfterAdd = resolveRequests.length;

    // Master data changes while the item is in the cart must not trigger resolve.
    await updateProductPrice(token, request, productId, productSku, 8000);
    await page.waitForTimeout(1000);
    expect(resolveRequests.length).toBe(resolveCountAfterAdd);

    // Cart still shows the snapshot price.
    const cart = await getCart(token, request, cartId);
    expect(cart.items[0].unit_price).toBe(BASE_PRICE);
  });

  // E2E-10 — full POS checkout flow regression (add → F4 → F7 → Selesai)
  test('E2E-10: full POS checkout flow still works', async ({ page, request }) => {
    const token = await getToken(request);
    await ensureFreshCart(token, request);
    await loginViaSessionStorage(page, token);

    await addProductByUI(page, productSku);
    await expect(page.locator('text=Your cart is empty')).toBeHidden({ timeout: 5000 });

    await page.keyboard.press('F4');
    const selesaiBtn = page.getByRole('button', { name: /Selesai/ });
    await expect(selesaiBtn).toBeVisible({ timeout: 5000 });

    await page.keyboard.press('F7');
    await expect(selesaiBtn).toBeEnabled({ timeout: 5000 });
    await selesaiBtn.click();

    await expect(page.locator('text=Your cart is empty')).toBeVisible({ timeout: 8000 });

    // The completed sale shows up in the POS history (last sale → Print button).
    const printBtn = page.locator('button', { hasText: 'Print · INV-' });
    await expect(printBtn).toBeVisible({ timeout: 8000 });
    const printText = (await printBtn.innerText()).trim();
    const invoiceMatch = printText.match(/INV-\S+/);
    expect(invoiceMatch).toBeTruthy();
    const invoice = invoiceMatch![0];

    const histRes = await request.get(
      `${API_URLS.SALES}?limit=10&offset=0&startDate=2025-01-01&endDate=${todayJakarta()}`,
      { headers: authHeader(token) },
    );
    expect(histRes.ok()).toBeTruthy();
    const histBody = await histRes.json();
    const raw = histBody.data ?? histBody;
    const list = Array.isArray(raw) ? raw : [];
    expect(list.length).toBeGreaterThanOrEqual(1);
    expect(
      list.some((s: any) => s.invoice_number === invoice && s.status === 'completed'),
    ).toBeTruthy();
  });
});
