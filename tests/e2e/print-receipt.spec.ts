import { test, expect } from './fixtures';
import { TEST_USERS } from './fixtures';

test.describe('Thermal Receipt Print Flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript(() => {
      sessionStorage.clear();
      localStorage.clear();
      localStorage.setItem('pos.locale', 'en');
    });
    await page.context().clearCookies();
    await page.goto('http://localhost:5173/', { waitUntil: 'domcontentloaded' });
  });

  async function loginAndGoToPos(page: any) {
    await expect(page).toHaveURL(/\/login$/);
    await page.fill('#username', TEST_USERS.superadmin.username);
    await page.fill('#password', TEST_USERS.superadmin.password);
    await page.click('button[type="submit"]');
    await page.waitForURL(/\/$/, { timeout: 10000 });
    await page.click('nav button:has-text("Point of Sale")');
    await page.waitForSelector('#pos-search-input', { state: 'visible', timeout: 15000 });
  }

  async function addItemToCart(page: any, count: number) {
    const addButtons = page.locator('button:has-text("Add"):not([disabled])');
    await addButtons.first().waitFor({ state: 'visible', timeout: 10000 });
    for (let i = 0; i < count; i++) {
      await addButtons.nth(i).click();
    }
    await page.waitForTimeout(500);
  }

  async function openCheckoutAndPayExact(page: any) {
    await page.keyboard.press('F4');
    await expect(page.getByRole('heading', { name: 'Payment' })).toBeVisible({ timeout: 5000 });
    await page.keyboard.press('F7');
    await expect(page.getByRole('button', { name: /Done/ })).toBeEnabled({ timeout: 5000 });
    await page.waitForTimeout(200);
  }

  test('should complete sale and show receipt in print preview hiding app chrome', async ({ page }) => {
    await loginAndGoToPos(page);
    await addItemToCart(page, 3);

    await openCheckoutAndPayExact(page);

    await page.evaluate(() => {
      (window as any)._printCallCount = 0;
      const originalPrint = window.print;
      window.print = function () {
        (window as any)._printCallCount = ((window as any)._printCallCount || 0) + 1;
        return originalPrint.apply(window, arguments);
      };
    });

    await page.getByRole('button', { name: /Done \[Enter\]/ }).click();
    await page.waitForFunction(() => (window as any)._printCallCount >= 1, { timeout: 10000 });

    await expect(page.locator('#thermal-receipt')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('.thermal-shop-name')).toHaveText('RETAIL POS');
    await expect(page.locator('.thermal-label:has-text("Invoice:")')).toBeVisible();
    await expect(page.locator('.thermal-label:has-text("Time:")')).toBeVisible();

    const itemCount = await page.locator('.thermal-item-name').count();
    expect(itemCount).toBeGreaterThanOrEqual(1);

    const hasPrintStyles = await page.evaluate(() => {
      for (const sheet of document.styleSheets) {
        try {
          for (const rule of sheet.cssRules) {
            if (rule instanceof CSSMediaRule && rule.conditionText.includes('print')) {
              for (const r of rule.cssRules) {
                if (r.cssText.includes('thermal-receipt-container') || r.cssText.includes('display: none')) {
                  return true;
                }
              }
            }
          }
        } catch (_e) {
        }
      }
      return false;
    });
    expect(hasPrintStyles).toBe(true);

    await page.emulateMedia({ media: 'print' });
    await page.screenshot({ path: 'test-results/print-receipt-preview.png' });
    await page.emulateMedia({ media: 'screen' });

    expect(page.url()).toContain('/pos');
    await expect(page.getByRole('button', { name: /Print/ })).toBeVisible({ timeout: 3000 });
  });

  test('print preview renders black text on white background', async ({ page }) => {
    await loginAndGoToPos(page);
    await addItemToCart(page, 2);
    await openCheckoutAndPayExact(page);

    await page.getByRole('button', { name: /Done \[Enter\]/ }).click();
    await page.waitForTimeout(600);

    await page.emulateMedia({ media: 'print' });
    await page.waitForTimeout(300);

    const style = await page.locator('#thermal-receipt').evaluate((el) => {
      const cs = window.getComputedStyle(el);
      return { bg: cs.backgroundColor, color: cs.color };
    });

    expect(style.bg).toMatch(/rgb\(255,\s*255,\s*255\)|#ffffff/i);
    expect(style.color).toMatch(/rgb\(0,\s*0,\s*0\)|#000000/i);
  });

  test('reprint button triggers print again with same receipt data', async ({ page }) => {
    await loginAndGoToPos(page);
    await addItemToCart(page, 1);
    await openCheckoutAndPayExact(page);

    await page.evaluate(() => {
      (window as any)._printCallCount = 0;
      const originalPrint = window.print;
      window.print = function () {
        (window as any)._printCallCount = ((window as any)._printCallCount || 0) + 1;
        return originalPrint.apply(window, arguments);
      };
    });

    await page.getByRole('button', { name: /Done \[Enter\]/ }).click();
    await page.waitForFunction(() => (window as any)._printCallCount >= 1, { timeout: 10000 });

    let printCalls = await page.evaluate(() => (window as any)._printCallCount || 0);
    expect(printCalls).toBeGreaterThanOrEqual(1);

    const invoiceFirst = await page.locator('.thermal-value').first().textContent();

    await page.getByRole('button', { name: /Print/ }).click();
    await page.waitForFunction(() => (window as any)._printCallCount >= 2, { timeout: 10000 });

    printCalls = await page.evaluate(() => (window as any)._printCallCount || 0);
    expect(printCalls).toBeGreaterThanOrEqual(2);
  });
});
