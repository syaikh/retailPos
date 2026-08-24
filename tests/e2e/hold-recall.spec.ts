import { test, expect } from './fixtures';
import { TEST_USERS, loginUI, logoutUI } from './fixtures';
import { purgeHeldCarts } from './db-helper';

test.describe('Hold & Recall UI Flow', () => {
  test.beforeEach(async ({ page }) => {
    // Recall uses `.first()` — stale held carts from other runs would win.
    purgeHeldCarts(TEST_USERS.superadmin.id);
    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    await page.goto('http://localhost:5173/pos');
    await expect(page).toHaveURL(/\/pos/);
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('should park sale via F6, recall via F7, and complete checkout', async ({ page }) => {
    await page.waitForTimeout(2000);
    const addButton = page.locator('button:not([disabled])').filter({ hasText: 'Add' }).first();
    await addButton.waitFor({ state: 'visible', timeout: 10000 });
    await expect(addButton).toBeEnabled({ timeout: 5000 });
    await addButton.click();
    await page.waitForTimeout(500);
    await expect(page.locator('text=Your cart is empty')).toBeHidden({ timeout: 5000 });

    await page.keyboard.press('F6');
    await page.waitForResponse(res => res.url().includes('/api/pos/cart/') && res.url().includes('/hold') && res.status() === 200, { timeout: 10000 });
    await expect(page.locator('text=Your cart is empty')).toBeVisible({ timeout: 5000 });

    await page.keyboard.press('F7');
    await page.waitForTimeout(1000);
    await expect(page.locator('text=Held Sales')).toBeVisible({ timeout: 5000 });

    const recallBtn = page.locator('button[data-action="recall"]').first();
    await recallBtn.waitFor({ state: 'visible', timeout: 5000 });
    await recallBtn.click();
    await page.waitForTimeout(1000);
    await expect(page.locator('text=Your cart is empty')).toBeHidden({ timeout: 5000 });

    await page.keyboard.press('F4');
    await page.waitForTimeout(1000);

    // F7 inside the payment modal = "Exact" cash allocation fill.
    await page.keyboard.press('F7');
    await page.waitForTimeout(500);

    const selesaiBtn = page.locator('button').filter({ hasText: /Selesai|Done/ });
    await selesaiBtn.waitFor({ state: 'visible', timeout: 5000 });
    await expect(selesaiBtn).toBeEnabled({ timeout: 3000 });
    await selesaiBtn.click();
    await page.waitForTimeout(2000);

    await expect(page.locator('text=Your cart is empty')).toBeVisible({ timeout: 8000 });
  });
});
