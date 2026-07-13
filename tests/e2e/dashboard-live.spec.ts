import { test, expect } from '@playwright/test';
import { TEST_USERS, API_URLS, API_BASE, authHeader, decodeJWT, loginUI, logoutUI, getToken } from './fixtures';

async function createTestSale(request: any, token: string, productId = 1) {
  const invoiceNumber = `INV-LIVE-${Date.now()}`;
  const totalAmount = 25000;
  const subtotal = 25000;

  const res = await request.post(`${API_BASE}/api/sales`, {
    headers: authHeader(token),
    data: {
      invoice_number: invoiceNumber,
      cashier_id: 1,
      subtotal,
      discount: 0,
      tax: 0,
      total_amount: totalAmount,
      payment_method: 'cash',
      items: [{ product_id: productId, quantity: 1, unit_price: totalAmount, subtotal }],
    },
  });

  expect(res.ok()).toBeTruthy();
  const data = await res.json();
  if (!res.ok()) {
    console.log('Sale creation failed:', JSON.stringify(data));
  }
  expect(data.data).toBeTruthy();
  return { saleId: data.data.id, invoiceNumber, totalAmount };
}

test.describe('Dashboard Live Stats', () => {
  test.beforeEach(async ({ page }) => {
    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('displays live dashboard header with connection indicator', async ({ page }) => {
    await expect(page.getByText("Live Dashboard", { exact: true })).toBeVisible();
    await expect(page.getByText("Live", { exact: true })).toBeVisible();
  });

  test('shows real stat cards on initial load', async ({ page }) => {
    await expect(page.getByText("Today's Revenue")).toBeVisible();
    await expect(page.locator('#main-content').getByText('Transactions', { exact: true })).toBeVisible();
    await expect(page.getByText('Total Products')).toBeVisible();
    await expect(page.getByText('Low Stock Alerts')).toBeVisible();
  });

  test('updates revenue and transactions after a new sale is broadcast', async ({ page, request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const productRes = await request.get(`${API_BASE}/api/products?limit=50`, {
      headers: authHeader(token),
    });
    expect(productRes.ok()).toBeTruthy();
    const productData = await productRes.json();
    const productWithStock = productData.data?.find((p: any) => (p.stock ?? 0) > 0);
    expect(productWithStock, 'no product with stock found').toBeTruthy();
    const productId = productWithStock.id;
    const productPrice = productWithStock.price ?? productWithStock.selling_price ?? 25000;

    const beforeStats = await request.get(`${API_BASE}/api/dashboard/live`, {
      headers: authHeader(token),
    });
    const beforeJson = await beforeStats.json();
    const beforeRevenue = (beforeJson.data?.todays_revenue ?? 0) as number;

    const saleRes = await request.post(`${API_BASE}/api/sales`, {
      headers: authHeader(token),
      data: {
        invoice_number: `INV-LIVE-${Date.now()}`,
        cashier_id: 1,
        subtotal: productPrice,
        discount: 0,
        tax: 0,
        total_amount: productPrice,
        payment_method: 'cash',
        items: [{ product_id: productId, quantity: 1, unit_price: productPrice, subtotal: productPrice }],
      },
    });

    const saleBody = await saleRes.text();
    expect(saleRes.ok(), `sale creation failed: status=${saleRes.status()} body=${saleBody}`).toBeTruthy();
    const saleJson = JSON.parse(saleBody);
    expect(saleJson.data).toBeTruthy();

    const afterRes = await request.get(`${API_BASE}/api/dashboard/live`, {
      headers: authHeader(token),
    });
    const afterJson = await afterRes.json();
    const afterRevenue = (afterJson.data?.todays_revenue ?? 0) as number;
    const afterSales = (afterJson.data?.todays_sales ?? 0) as number;

    expect(afterRevenue).toBeGreaterThan(beforeRevenue);
    expect(afterRevenue).toBeGreaterThanOrEqual(productPrice);
    expect(afterSales).toBeGreaterThanOrEqual(1);
  });
});
