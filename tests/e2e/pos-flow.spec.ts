import { test, expect } from '@playwright/test';
import { TEST_USERS } from './fixtures';

async function getAuthToken(page) {
  const tokenResponse = await page.request.post('http://localhost:9095/api/login', {
    data: {
      username: TEST_USERS.superadmin.username,
      password: TEST_USERS.superadmin.password
    }
  });
  const tokenData = await tokenResponse.json();
  return tokenData.access_token;
}

test.describe('POS API Tests', () => {
  test('should create sale via API', async ({ page }) => {
    const token = await getAuthToken(page);

    const saleResponse = await page.request.post('http://localhost:9095/api/sales', {
      headers: { Authorization: `Bearer ${token}` },
      data: {
        invoice_number: `INV-${Date.now()}`,
        cashier_id: 1,
        subtotal: 1262000,
        discount: 0,
        tax: 113580,
        total_amount: 1375580,
        payment_method: 'cash',
items: [
           { product_id: 4690, quantity: 1, unit_price: 1262000, subtotal: 1262000 }
         ]
      }
    });

    expect(saleResponse.ok()).toBeTruthy();
    const sale = await saleResponse.json();
    expect(sale.data).toHaveProperty('id');
    expect(sale.data.invoice_number).toBeTruthy();
  });

  test('GET /api/products should return products list', async ({ page }) => {
    const token = await getAuthToken(page);

    const response = await page.request.get('http://localhost:9095/api/products', {
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
