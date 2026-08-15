import { test, expect } from '@playwright/test';
import { API_BASE, authHeader, loginUI, logoutUI, getToken } from './fixtures';

// Konsinyasi (Consignment) Supplier module — E2E happy path.
// API setup in beforeAll (store-scoped admin user), then UI-driven flow:
//   arrangement → receipt → POS sale → pending return → return → settlement → payout
//
// The store-scoped admin (role 2, store_id 1) is required because the
// consignment list endpoints resolve the store from the JWT and seed users
// have a NULL store_id, which returns 403 for those endpoints. Role 2 holds
// all consignment.* permissions (incl. settle and pay) plus sale.create.
//
// Locale: the app defaults to Indonesian ('id'); a fresh Playwright context
// has empty localStorage, so `loadInitialLocale()` returns 'id'. All label
// selectors below use the Indonesian dictionary.

test.describe('Consignment Supplier - Full Flow', () => {
  let headers: Record<string, string>;
  let adminToken: string;
  let adminUser: { username: string; password: string };
  let supplier: { id: number; name: string };
  let product: { id: number; name: string; sku: string; price: number };
  let arrangement: { id: number };
  let receiptNumber: string;

  test.beforeAll(async ({ request }) => {
    // 1. Superadmin creates a store-scoped admin user
    const superToken = await getToken(request);
    const superHeaders = authHeader(superToken);

    const suffix = Date.now();
    const username = `e2ecn${suffix}`;
    const password = 'e2eConsign123';
    const createUserRes = await request.post(`${API_BASE}/api/admin/users`, {
      headers: superHeaders,
      data: {
        username,
        email: `${username}@retail-pos.local`,
        password,
        role_id: 2,
        store_id: 1,
      },
    });
    expect(createUserRes.ok(), `create user failed: ${createUserRes.status()} ${await createUserRes.text()}`).toBeTruthy();
    adminUser = { username, password };

    // 2. Login as the store-scoped admin for the rest of the setup
    adminToken = await getToken(request, username, password);
    headers = authHeader(adminToken);

    // 3. Consignment supplier
    const supplierName = `E2E Konsinyasi ${suffix}`;
    const supRes = await request.post(`${API_BASE}/api/suppliers`, {
      headers,
      data: { name: supplierName, is_consignment: true },
    });
    expect(supRes.ok(), `supplier failed: ${supRes.status()} ${await supRes.text()}`).toBeTruthy();
    supplier = (await supRes.json()).data;

    // 4. Product (must be store-scoped + active so it appears in the store-scoped POS list)
    const sku = `KNS-${suffix}`;
    const price = 100000;
    const prodRes = await request.post(`${API_BASE}/api/products`, {
      headers,
      data: {
        name: `Konsinyasi ${suffix}`,
        sku,
        price,
        cost: 60000,
        stock: 0,
        status: 'active',
        store_id: 1,
      },
    });
    expect(prodRes.ok(), `product failed: ${prodRes.status()} ${await prodRes.text()}`).toBeTruthy();
    product = (await prodRes.json()).data;
    expect(product.id).toBeGreaterThan(0);

    // 5. Arrangement
    const arrRes = await request.post(`${API_BASE}/api/consignment/arrangements`, {
      headers,
      data: { supplier_id: supplier.id, store_id: 1 },
    });
    expect(arrRes.ok(), `arrangement failed: ${arrRes.status()} ${await arrRes.text()}`).toBeTruthy();
    arrangement = (await arrRes.json()).data;

    // 6. Terms — PUT with an array (store share 20% percentage)
    const termsRes = await request.put(`${API_BASE}/api/consignment/arrangements/${arrangement.id}/terms`, {
      headers,
      data: [
        {
          product_id: product.id,
          price,
          store_share_type: 'percentage',
          store_share_value: 20,
        },
      ],
    });
    expect(termsRes.ok(), `terms failed: ${termsRes.status()} ${await termsRes.text()}`).toBeTruthy();

    // 7. Receipt (10 accepted units → stock becomes 10 so POS Add is enabled)
    const recRes = await request.post(`${API_BASE}/api/consignment/receipts`, {
      headers,
      data: {
        arrangement_id: arrangement.id,
        notes: 'E2E setup',
        items: [{ product_id: product.id, accepted_qty: 10, notes: '' }],
      },
    });
    expect(recRes.ok(), `receipt failed: ${recRes.status()} ${await recRes.text()}`).toBeTruthy();
    receiptNumber = (await recRes.json()).data.receipt_number;
    expect(receiptNumber).toBeTruthy();
  });

  test.beforeEach(async ({ page }) => {
    await loginUI(page, adminUser.username, adminUser.password);
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('opens arrangement detail from the list and verifies receipt history', async ({ page }) => {
    await page.goto('/consignment');
    await page.waitForTimeout(1500);

    const row = page.locator('tbody tr').filter({ hasText: supplier.name }).first();
    await expect(row).toBeVisible({ timeout: 10000 });

    await row.locator('button').filter({ hasText: 'Buka' }).click();
    await page.waitForTimeout(800);

    // Detail header shows supplier name
    await expect(page.locator('h1').filter({ hasText: supplier.name })).toBeVisible({ timeout: 5000 });

    // Tabs are present
    for (const tab of ['Penerimaan', 'Terms', 'Retur Tertunda', 'Retur', 'Settlement', 'Stok']) {
      await expect(page.locator('button').filter({ hasText: tab }).first()).toBeVisible({ timeout: 3000 });
    }

    // Receipt history shows the receipt number from beforeAll
    await expect(page.locator('tbody tr').filter({ hasText: receiptNumber }).first()).toBeVisible({ timeout: 5000 });
  });

  test('sells the consigned product via POS and stock decreases', async ({ page }) => {
    await page.goto('/pos');
    await page.waitForTimeout(1500);

    // Search for the product by SKU
    await page.locator('#pos-search-input').fill(product.sku);
    await page.waitForTimeout(1200);

    const row = page.locator('tbody tr').filter({ hasText: product.name }).first();
    await expect(row).toBeVisible({ timeout: 8000 });

    // Add the product to the cart once (qty 1)
    await row.locator('button').filter({ hasText: 'Tambah' }).click();
    await page.waitForTimeout(800);

    // Open checkout with F4
    await page.keyboard.press('F4');
    const checkoutDialog = page.getByRole('dialog', { name: 'Pembayaran' });
    await expect(checkoutDialog).toBeVisible({ timeout: 5000 });

    // CASH allocation is auto-created with amount 0; set exact amount via shortcut
    await checkoutDialog.locator('button').filter({ hasText: 'Tepat [F7]' }).click();
    await page.waitForTimeout(300);

    // Finalize
    const doneBtn = checkoutDialog.locator('button').filter({ hasText: 'Selesai [Enter]' });
    await expect(doneBtn).toBeEnabled({ timeout: 3000 });

    const checkoutResponsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/pos/cart/') && resp.url().includes('/checkout') && resp.request().method() === 'POST'
    );
    await doneBtn.click();
    const checkoutResponse = await checkoutResponsePromise;
    expect(checkoutResponse.status(), `checkout failed: ${checkoutResponse.status()} ${await checkoutResponse.text()}`).toBe(201);

    await expect(page.locator('text=Transaksi berhasil')).toBeVisible({ timeout: 5000 });

    // Verify stock decreased via API (10 - 1 = 9)
    const prodRes = await page.request.get(`${API_BASE}/api/products/${product.id}`, { headers });
    expect(prodRes.ok()).toBeTruthy();
    const prod = (await prodRes.json()).data;
    expect(prod.stock).toBe(9);
  });

  test('records a pending return from consignment stock', async ({ page }) => {
    await page.goto('/consignment');
    await page.waitForTimeout(1500);

    const row = page.locator('tbody tr').filter({ hasText: supplier.name }).first();
    await expect(row).toBeVisible({ timeout: 10000 });
    await row.locator('button').filter({ hasText: 'Buka' }).click();
    await page.waitForTimeout(800);

    // Pending Returns tab
    await page.locator('button').filter({ hasText: 'Retur Tertunda' }).first().click();
    await page.waitForTimeout(800);

    await page.locator('button').filter({ hasText: 'Catat Retur Tertunda' }).click();
    const modal = page.getByRole('dialog', { name: 'Catat Retur Tertunda' });
    await expect(modal).toBeVisible({ timeout: 5000 });

    // Select product from consignment stock
    const productSelect = modal.locator('label:has-text("Produk (dari stok konsinyasi)")').locator('[aria-haspopup="listbox"]');
    await productSelect.click();
    await page.waitForTimeout(300);
    const listbox = page.locator('[role="listbox"]');
    await expect(listbox).toBeVisible({ timeout: 3000 });
    await listbox.locator('[role="option"]').filter({ hasText: product.sku }).first().click();
    await page.waitForTimeout(300);

    // Qty (default 1)
    const qtyInput = modal.locator('input[type="number"]').first();
    await qtyInput.fill('1');

    // Reason select (default damaged)
    await modal.locator('select').selectOption('damaged');
    await page.waitForTimeout(200);

    // Submit
    const createResponsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/api/consignment/pending-returns') && resp.request().method() === 'POST'
    );
    await modal.locator('button').filter({ hasText: 'Simpan' }).click();
    const createResponse = await createResponsePromise;
    expect(createResponse.status(), `pending return failed: ${createResponse.status()} ${await createResponse.text()}`).toBe(201);

    await expect(page.locator('text=Retur tertunda tercatat')).toBeVisible({ timeout: 5000 });
    await expect(modal).not.toBeVisible({ timeout: 5000 });

    // The pending return row appears in the table
    await expect(page.locator('tbody tr').filter({ hasText: product.name }).first()).toBeVisible({ timeout: 5000 });
  });

  test('records a formal return linked to the pending return', async ({ page }) => {
    await page.goto('/consignment');
    await page.waitForTimeout(1500);

    const row = page.locator('tbody tr').filter({ hasText: supplier.name }).first();
    await expect(row).toBeVisible({ timeout: 10000 });
    await row.locator('button').filter({ hasText: 'Buka' }).click();
    await page.waitForTimeout(800);

    // Returns tab (exact name — 'Retur' would also match 'Retur Tertunda')
    await page.getByRole('button', { name: 'Retur', exact: true }).click();
    await page.waitForTimeout(800);

    // 'Catat Retur' (exact — must not match 'Catat Retur Tertunda')
    await page.getByRole('button', { name: 'Catat Retur', exact: true }).click();
    const modal = page.getByRole('dialog', { name: 'Catat Retur', exact: true });
    await expect(modal).toBeVisible({ timeout: 5000 });

    // Select product
    const productSelect = modal.locator('label:has-text("Produk")').first().locator('[aria-haspopup="listbox"]');
    await productSelect.click();
    await page.waitForTimeout(300);
    const listbox = page.locator('[role="listbox"]');
    await expect(listbox).toBeVisible({ timeout: 3000 });
    await listbox.locator('[role="option"]').filter({ hasText: product.sku }).first().click();
    await page.waitForTimeout(300);

    // Qty (default 1)
    const qtyInput = modal.locator('input[type="number"]').first();
    await qtyInput.fill('1');

    // Reason
    await modal.locator('select').first().selectOption('damaged');
    await page.waitForTimeout(200);

    // Link to the open pending return (index 1 = first real option, index 0 = no link)
    const pendingSelect = modal.locator('select').nth(1);
    await pendingSelect.selectOption({ index: 1 });
    await page.waitForTimeout(200);

    // Submit
    const createResponsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/api/consignment/returns') && resp.request().method() === 'POST'
    );
    await modal.locator('button').filter({ hasText: 'Simpan' }).click();
    const createResponse = await createResponsePromise;
    expect(
      createResponse.status(),
      `return failed: ${createResponse.status()} ${await createResponse.text()} body=${createResponse.request().postData()}`
    ).toBe(201);
    const returnBody = await createResponse.json();
    const returnNumber = (returnBody.data || returnBody).return_number;
    expect(returnNumber).toBeTruthy();

    await expect(page.locator('text=tercatat').first()).toBeVisible({ timeout: 5000 });
    await expect(modal).not.toBeVisible({ timeout: 5000 });

    // Return row appears (table shows return number / date / item count / qty)
    await expect(page.locator('tbody tr').filter({ hasText: returnNumber }).first()).toBeVisible({ timeout: 5000 });
  });

  test('creates a settlement and records the payout', async ({ page }) => {
    await page.goto('/consignment');
    await page.waitForTimeout(1500);

    const row = page.locator('tbody tr').filter({ hasText: supplier.name }).first();
    await expect(row).toBeVisible({ timeout: 10000 });
    await row.locator('button').filter({ hasText: 'Buka' }).click();
    await page.waitForTimeout(800);

    // Settlement tab
    await page.locator('button').filter({ hasText: 'Settlement' }).first().click();
    await page.waitForTimeout(1200);

    // Preview totals present
    await expect(page.locator('text=Total Penjualan').first()).toBeVisible({ timeout: 8000 });
    await expect(page.locator('text=Hak Toko').first()).toBeVisible({ timeout: 3000 });
    await expect(page.locator('text=Terhutang ke Supplier').first()).toBeVisible({ timeout: 3000 });

    // Create settlement
    const createBtn = page.locator('button').filter({ hasText: 'Buat Settlement' }).first();
    await expect(createBtn).toBeEnabled({ timeout: 5000 });
    await createBtn.click();

    const confirmModal = page.getByRole('dialog', { name: 'Konfirmasi Settlement' });
    await expect(confirmModal).toBeVisible({ timeout: 5000 });

    const createResponsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/api/consignment/settlements') && !resp.url().includes('preview') && resp.request().method() === 'POST'
    );
    await confirmModal.locator('button').filter({ hasText: 'Buat Settlement' }).click();
    const createResponse = await createResponsePromise;
    expect(createResponse.status(), `settlement failed: ${createResponse.status()} ${await createResponse.text()}`).toBe(201);
    const settlementBody = await createResponse.json();
    const settlementId = (settlementBody.data || settlementBody).id;
    expect(settlementId).toBeGreaterThan(0);

    await expect(page.locator('text=Menunggu Pembayaran').first()).toBeVisible({ timeout: 5000 });

    // Pay the settlement
    const payBtn = page.locator('tbody tr').filter({ hasText: 'Menunggu Pembayaran' }).first()
      .locator('button').filter({ hasText: 'Bayar' });
    await expect(payBtn).toBeVisible({ timeout: 5000 });
    await payBtn.click();

    const payoutModal = page.getByRole('dialog', { name: 'Catat Pembayaran' });
    await expect(payoutModal).toBeVisible({ timeout: 5000 });

    // Payment method SelectSearch: 'Cash (CASH)'
    const methodSelect = payoutModal.locator('label:has-text("Metode Pembayaran")').locator('[aria-haspopup="listbox"]');
    await methodSelect.click();
    await page.waitForTimeout(300);
    const methodListbox = page.locator('[role="listbox"]');
    await expect(methodListbox).toBeVisible({ timeout: 3000 });
    await methodListbox.locator('[role="option"]').filter({ hasText: 'Cash (CASH)' }).first().click();
    await page.waitForTimeout(300);

    // Amount is prefilled; submit
    const payoutResponsePromise = page.waitForResponse(
      (resp) => resp.url().includes(`/api/consignment/settlements/${settlementId}/payouts`) && resp.request().method() === 'POST'
    );
    await payoutModal.locator('button').filter({ hasText: 'Bayar' }).click();
    const payoutResponse = await payoutResponsePromise;
    expect(payoutResponse.status(), `payout failed: ${payoutResponse.status()} ${await payoutResponse.text()}`).toBe(201);

    await expect(page.locator('text=Dibayar').first()).toBeVisible({ timeout: 5000 });
  });
});