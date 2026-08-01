import { test, expect } from '@playwright/test';
import { TEST_USERS, API_BASE, getToken, authHeader, loginUI, logoutUI } from './fixtures';

const API_URLS = {
  PRODUCTS: `${API_BASE}/api/products`,
  CART: `${API_BASE}/api/pos/cart`,
};

function cartItemUrl(cartId: number) {
  return `${API_URLS.CART}/${cartId}`;
}

function checkoutUrl(cartId: number) {
  return `${API_URLS.CART}/${cartId}/checkout`;
}

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

test.describe('Price Consistency During Active Transactions', () => {
  let productId: number;
  let productSku: string;

  test.beforeAll(async ({ request }) => {
    const token = await getToken(request);

    const res = await request.post(`${API_URLS.PRODUCTS}`, {
      headers: authHeader(token),
      data: {
        name: 'Test Price Consistency Product',
        sku: `TEST-PRICE-${Date.now()}`,
        price: 10000,
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

  test('should preserve snapshot price in cart after product price change', async ({ page, request }) => {
    const token = await getToken(request);
    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    await page.goto('http://localhost:5173/pos');
    await expect(page).toHaveURL(/\/pos/);

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
    expect(item.unit_price).toBe(10000);
    expect(item.snapshot_created_at).toBeTruthy();

    await request.put(`${API_URLS.PRODUCTS}/${productId}`, {
      headers: authHeader(token),
      data: {
        name: 'Test Price Consistency Product',
        sku: productSku,
        price: 15000,
        cost: 5000,
        stock: 100,
        status: 'active',
      },
    });

    const updatedCart = await request.get(cartItemUrl(cartId), {
      headers: authHeader(token),
    });
    expect(updatedCart.ok()).toBeTruthy();
    const updatedItems = (await updatedCart.json()).data.items;
    const updatedItem = updatedItems.find((i: any) => i.product_id === productId);
    expect(updatedItem).toBeDefined();
    expect(updatedItem.unit_price).toBe(10000);
    expect(updatedItem.snapshot_created_at).toBeTruthy();
  });

  test('should use snapshot prices on checkout', async ({ page, request }) => {
    const token = await getToken(request);
    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    await page.goto('http://localhost:5173/pos');
    await expect(page).toHaveURL(/\/pos/);

    const cartId = await ensureFreshCart(token, request);

    await request.post(`${API_URLS.CART}/items`, {
      headers: authHeader(token),
      data: { product_id: productId, quantity: 2 },
    });

    const cartDetail = await request.get(cartItemUrl(cartId), {
      headers: authHeader(token),
    });
    expect(cartDetail.ok()).toBeTruthy();
    const cartData = (await cartDetail.json()).data;
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
});
