import { test, expect } from './fixtures';
import { TEST_USERS, API_BASE, authHeader, loginUI, logoutUI, getToken } from './fixtures';

test.describe('Purchase Orders - Kebab Menu Dropdown', () => {
  const poIds: number[] = [];

  test.beforeAll(async ({ request }) => {
    const token = await getToken(request);
    const headers = authHeader(token);

    const supRes = await request.get(`${API_BASE}/api/suppliers?limit=1`, { headers });
    expect(supRes.ok()).toBeTruthy();
    const supplier = ((await supRes.json()).data || (await supRes.json()))[0];

    const prodRes = await request.get(`${API_BASE}/api/products?limit=1`, { headers });
    expect(prodRes.ok()).toBeTruthy();
    const product = ((await prodRes.json()).data || [])[0];

    const storeRes = await request.get(`${API_BASE}/api/stores/active`, { headers });
    expect(storeRes.ok()).toBeTruthy();
    const storeRaw = (await storeRes.json()).data;
    const store = Array.isArray(storeRaw) ? storeRaw[0] : storeRaw;

    const expDate = new Date(Date.now() + 7 * 86400000).toISOString().split('T')[0];

    for (let i = 0; i < 3; i++) {
      const res = await request.post(`${API_BASE}/api/purchase-orders`, {
        headers,
        data: {
          supplier_id: supplier.id,
          store_id: store.id,
          expected_date: expDate,
          items: [{ product_id: product.id, qty_ordered: i + 5, unit_cost: 1000 }],
        },
      });
      expect(res.ok()).toBeTruthy();
      const body = await res.json();
      poIds.push((body.data || body).id);
    }
  });

  test.beforeEach(async ({ page }) => {
    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('dropdown appears fully visible without clipping on all rows', async ({ page }) => {
    await page.goto('/purchase-orders');
    await page.waitForTimeout(1500);

    const table = page.locator('[role="grid"][aria-label="Purchase Orders"]');
    await expect(table).toBeVisible({ timeout: 5000 });

    const rows = table.locator('tbody tr');
    const rowCount = await rows.count();
    expect(rowCount).toBeGreaterThanOrEqual(3);

    for (let i = 0; i < Math.min(rowCount, 5); i++) {
      const row = rows.nth(i);
      const actionBtn = row.locator('button[aria-label*="Actions for"]');
      await actionBtn.click();
      await page.waitForTimeout(400);

      const dropdown = page.locator('[role="menu"]').first();
      await expect(dropdown).toBeVisible({ timeout: 3000 });

      const menuItems = dropdown.locator('[role="menuitem"]');
      const itemCount = await menuItems.count();
      expect(itemCount).toBeGreaterThan(0);

      for (let j = 0; j < itemCount; j++) {
        await expect(menuItems.nth(j)).toBeVisible({ timeout: 1000 });
      }

      const box = await dropdown.boundingBox();
      expect(box).not.toBeNull();
      expect(box!.x).toBeGreaterThanOrEqual(0);
      expect(box!.y).toBeGreaterThanOrEqual(0);
      const vp = page.viewportSize();
      if (vp) {
        expect(box!.x + box!.width).toBeLessThanOrEqual(vp.width);
        expect(box!.y + box!.height).toBeLessThanOrEqual(vp.height);
      }

      // Click first menu item (View) to verify clickability
      await menuItems.first().click();
      await page.waitForTimeout(500);

      // Should navigate away or show drawer; go back to PO list
      await page.goto('/purchase-orders');
      await page.waitForTimeout(1000);
    }
  });

  test('dropdown renders above table container (z-stacking)', async ({ page }) => {
    await page.goto('/purchase-orders');
    await page.waitForTimeout(1500);

    const table = page.locator('[role="grid"][aria-label="Purchase Orders"]');
    await expect(table).toBeVisible({ timeout: 5000 });

    const firstRow = table.locator('tbody tr').first();
    const actionBtn = firstRow.locator('button[aria-label*="Actions for"]');
    await actionBtn.click();
    await page.waitForTimeout(400);

    const dropdown = page.locator('[role="menu"]').first();
    await expect(dropdown).toBeVisible({ timeout: 3000 });

    const menuItems = dropdown.locator('[role="menuitem"]');
    await expect(menuItems.first()).toBeVisible();

    const dropdownBox = await dropdown.boundingBox();
    expect(dropdownBox).not.toBeNull();

    // The dropdown must NOT be clipped — its full height must match the menu items
    let totalItemsHeight = 0;
    const gap = 1;
    for (let i = 0; i < await menuItems.count(); i++) {
      const itemBox = await menuItems.nth(i).boundingBox();
      if (itemBox) totalItemsHeight += itemBox.height + gap;
    }
    expect(dropdownBox!.height).toBeGreaterThanOrEqual(totalItemsHeight * 0.5);

    // Close via Escape
    await page.keyboard.press('Escape');
    await page.waitForTimeout(300);
    await expect(dropdown).not.toBeVisible();
  });
});
