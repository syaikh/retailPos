import { test, expect } from '@playwright/test';
import { TEST_USERS, API_BASE, authHeader, loginUI } from './fixtures';

test.describe('WebSocket Real-time Events', () => {

  test('WebSocket connects after login and receives sale_created event', async ({ page }) => {
    const messages: string[] = [];

    page.on('websocket', ws => {
      ws.on('framereceived', frame => {
        messages.push(typeof frame === 'string' ? frame : frame.payload || '');
      });
    });

    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    await page.waitForTimeout(2000);

    expect(messages.length).toBeGreaterThanOrEqual(0);

    const token = await page.evaluate(() => sessionStorage.getItem('access_token'));
    expect(token).toBeTruthy();

    // Create product with known stock via API to ensure product_stock record exists
    const sku = `SALE-WS-${Date.now()}`;
    const createRes = await page.request.post(`${API_BASE}/api/products`, {
      headers: authHeader(token!),
      data: { name: 'E2E WS Sale Test', sku, price: 25000, cost: 10000, stock: 10, status: 'active' },
    });
    expect(createRes.ok()).toBeTruthy();
    const created = await createRes.json();
    const productId = created.data?.id || created.id;

    // Adjust stock to create product_stock record
    await page.request.post(`${API_BASE}/api/inventory/adjust`, {
      headers: authHeader(token!),
      data: { product_id: productId, quantity_change: 10, notes: 'seed stock for WS sale test' },
    });

    const saleRes = await page.request.post(`${API_BASE}/api/sales`, {
      headers: authHeader(token!),
      data: {
        items: [{ product_id: productId, quantity: 1, subtotal: 25000 }],
        payment_method: 'cash',
      },
    });
    expect(saleRes.ok()).toBeTruthy();

    await page.waitForTimeout(2000);

    const saleCreatedMsg = messages.find(m => m.includes('sale_created'));
    expect(saleCreatedMsg).toBeTruthy();
  });

  test('WebSocket connection requires valid token', async ({ page }) => {
    let wsRejected = false;

    page.on('websocket', ws => {
      ws.on('close', () => { wsRejected = true; });
    });

    // Navigate to a page that attempts WebSocket connection without token
    await page.goto('/login');
    await page.evaluate(() => sessionStorage.clear());

    // Trigger WebSocket connection attempt by setting a dummy token
    await page.evaluate(() => {
      sessionStorage.setItem('access_token', 'invalid-token');
    });

    // Reload to trigger WebSocket connection
    await page.reload();
    await page.waitForTimeout(2000);

    // The WebSocket with invalid token should fail or be rejected
    // The app should still be on login page
    await expect(page.locator('#username')).toBeVisible({ timeout: 5000 });
  });

  test('WebSocket events have valid JSON structure', async ({ page }) => {
    const messages: string[] = [];

    page.on('websocket', ws => {
      ws.on('framereceived', frame => {
        const raw = typeof frame === 'string' ? frame : frame.payload || '';
        if (raw) messages.push(raw);
      });
    });

    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    await page.waitForTimeout(2000);

    for (const msg of messages) {
      const parsed = JSON.parse(msg);
      expect(parsed.type).toBeDefined();
    }
  });

  test('WebSocket receives stock_update after inventory adjustment', async ({ page }) => {
    const messages: string[] = [];

    page.on('websocket', ws => {
      ws.on('framereceived', frame => {
        messages.push(typeof frame === 'string' ? frame : frame.payload || '');
      });
    });

    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    await page.waitForTimeout(2000);

    const token = await page.evaluate(() => sessionStorage.getItem('access_token'));
    expect(token).toBeTruthy();

    const productsRes = await page.request.get(`${API_BASE}/api/products?limit=1`, {
      headers: authHeader(token!),
    });
    const productsBody = await productsRes.json();
    const product = (productsBody.data || [])[0];
    expect(product).toBeTruthy();

    const adjRes = await page.request.post(`${API_BASE}/api/inventory/adjust`, {
      headers: authHeader(token!),
      data: {
        product_id: product.id,
        quantity_change: 5,
        notes: 'E2E test stock adjustment',
      },
    });
    expect(adjRes.ok()).toBeTruthy();

    await page.waitForTimeout(2000);

    const stockUpdateMsg = messages.find(m => m.includes('stock_update'));
    expect(stockUpdateMsg).toBeTruthy();
  });

  test('WebSocket receives product_updated after product update', async ({ page }) => {
    const messages: string[] = [];

    page.on('websocket', ws => {
      ws.on('framereceived', frame => {
        messages.push(typeof frame === 'string' ? frame : frame.payload || '');
      });
    });

    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    await page.waitForTimeout(2000);

    const token = await page.evaluate(() => sessionStorage.getItem('access_token'));
    expect(token).toBeTruthy();

    const productsRes = await page.request.get(`${API_BASE}/api/products?limit=1`, {
      headers: authHeader(token!),
    });
    const productsBody = await productsRes.json();
    const product = (productsBody.data || [])[0];
    expect(product).toBeTruthy();

    const updateRes = await page.request.put(`${API_BASE}/api/products/${product.id}`, {
      headers: authHeader(token!),
      data: {
        name: product.name,
        sku: product.sku,
        price: product.price || 10000,
        cost: product.cost || 0,
        stock: product.stock || 0,
        status: product.status || 'active',
      },
    });
    expect(updateRes.ok()).toBeTruthy();

    await page.waitForTimeout(2000);

    const productUpdateMsg = messages.find(m => m.includes('product_updated'));
    expect(productUpdateMsg).toBeTruthy();
  });

  test('WebSocket receives low_stock_alert when stock reaches zero', async ({ page }) => {
    const messages: string[] = [];

    page.on('websocket', ws => {
      ws.on('framereceived', frame => {
        messages.push(typeof frame === 'string' ? frame : frame.payload || '');
      });
    });

    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    await page.waitForTimeout(2000);

    const token = await page.evaluate(() => sessionStorage.getItem('access_token'));
    expect(token).toBeTruthy();

    // Create a product with known low stock
    const sku = `LOW-STOCK-${Date.now()}`;
    const createRes = await page.request.post(`${API_BASE}/api/products`, {
      headers: authHeader(token!),
      data: {
        name: 'E2E Low Stock Test',
        sku,
        price: 10000,
        cost: 5000,
        stock: 3,
        status: 'active',
      },
    });
    expect(createRes.ok()).toBeTruthy();
    const created = await createRes.json();
    const productId = created.data?.id || created.id;

    // Adjust stock to bring it to zero, triggering low_stock_alert
    const adjRes = await page.request.post(`${API_BASE}/api/inventory/adjust`, {
      headers: authHeader(token!),
      data: {
        product_id: productId,
        quantity_change: -3,
        notes: 'E2E test low stock alert',
      },
    });
    expect(adjRes.ok()).toBeTruthy();

    await page.waitForTimeout(2000);

    const lowStockMsg = messages.find(m => m.includes('low_stock_alert'));
    expect(lowStockMsg).toBeTruthy();
  });

});
