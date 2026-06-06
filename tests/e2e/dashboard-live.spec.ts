import { test, expect } from '@playwright/test';
import { TEST_USERS, API_URLS, API_BASE, authHeader, decodeJWT } from './fixtures';

async function getAuthToken(request: any) {
  const res = await request.post(`${API_BASE}/api/login`, {
    data: {
      username: TEST_USERS.superadmin.username,
      password: TEST_USERS.superadmin.password,
    },
  });
  const body = await res.json();
  expect(res.ok()).toBeTruthy();
  return body.access_token;
}

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
  expect(data.data).toBeTruthy();
  return { saleId: data.data.id, invoiceNumber, totalAmount };
}

test.describe('Dashboard Live Stats', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('http://localhost:5173/login');
    await page.fill('#username', TEST_USERS.superadmin.username);
    await page.fill('#password', TEST_USERS.superadmin.password);
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL(/\/$/);
  });

  test('displays live dashboard header with connection indicator', async ({ page }) => {
    await expect(page.getByText("Live Dashboard")).toBeVisible();
    await expect(page.getByText("Live")).toBeVisible();
  });

  test('shows real stat cards on initial load', async ({ page }) => {
    await expect(page.getByText("Today's Revenue")).toBeVisible();
    await expect(page.getByText("Transactions")).toBeVisible();
    await expect(page.getByText("Total Products")).toBeVisible();
    await expect(page.getByText("Low Stock Alerts")).toBeVisible();
  });

  test('updates revenue and transactions after a new sale is broadcast', async ({ page, request }) => {
    const token = await getAuthToken(request);
    const productRes = await request.get(`${API_BASE}/api/products?limit=1`, {
      headers: authHeader(token),
    });
    const productData = await productRes.json();
    const firstProduct = productData.data[0];
    const productId = firstProduct?.id ?? 1;

    const beforeStats = await request.get(`${API_BASE}/api/dashboard/live`, {
      headers: authHeader(token),
    });
    const beforeJson = await beforeStats.json();
    const beforeRevenue = (beforeJson.data?.todays_revenue ?? 0) as number;

    await createTestSale(request, token, productId);

    const afterRes = await request.get(`${API_BASE}/api/dashboard/live`, {
      headers: authHeader(token),
    });
    const afterJson = await afterRes.json();
    const afterRevenue = (afterJson.data?.todays_revenue ?? 0) as number;
    const afterSales = (afterJson.data?.todays_sales ?? 0) as number;

    expect(afterRevenue).toBeGreaterThan(beforeRevenue);
    expect(afterRevenue).toBeGreaterThanOrEqual(25000);
    expect(afterSales).toBeGreaterThanOrEqual(1);
  });
});
