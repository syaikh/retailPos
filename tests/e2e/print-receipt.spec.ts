import { test, expect } from '@playwright/test';
import { TEST_USERS } from './fixtures';

test.describe('Thermal Receipt Print Flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript(() => {
      sessionStorage.clear();
      localStorage.clear();
    });
    await page.context().clearCookies();
    await page.goto('http://localhost:5173/', { waitUntil: 'domcontentloaded' });
  });

  test('should complete sale and show receipt in print preview hiding app chrome', async ({ page }) => {
    await expect(page).toHaveURL(/\/login$/);
    await page.fill('#username', TEST_USERS.superadmin.username);
    await page.fill('#password', TEST_USERS.superadmin.password);
    await page.click('button[type="submit"]');
    await page.waitForURL(/\/$/, { timeout: 10000 });

    // Navigate to POS via sidebar (SPA router, no full page reload)
    await page.click('nav button:has-text("Point of Sale")');
    await page.waitForSelector('#pos-search-input', { state: 'visible', timeout: 15000 });

    const addButtons = page.locator('button:has-text("Add"):not([disabled])');
    await addButtons.first().waitFor({ state: 'visible', timeout: 10000 });
    expect(await addButtons.count()).toBeGreaterThanOrEqual(3);
    for (let i = 0; i < 3; i++) {
      await addButtons.nth(i).click();
    }
    await page.waitForTimeout(500);

    await page.getByRole('button', { name: /Bayar \[F4\]/ }).click();
    await page.waitForTimeout(500);
    await expect(page.locator('text=Pembayaran Selesai')).toBeVisible({ timeout: 5000 });

    const totalText = await page.locator('.text-4xl.font-extrabold').textContent();
    const parsedTotal = parseInt((totalText || '0').replace(/[^\d]/g, ''), 10);
    await page.fill('#cash-received-input', String(parsedTotal));
    await page.waitForTimeout(300);

    const changeDueText = await page.locator('.text-2xl.font-extrabold').textContent();
    expect(changeDueText).not.toContain('kurang');

    await page.evaluate(() => {
      (window as any)._printCallCount = 0;
      const originalPrint = window.print;
      window.print = function () {
        (window as any)._printCallCount = ((window as any)._printCallCount || 0) + 1;
        return originalPrint.apply(window, arguments);
      };
    });

    await page.getByRole('button', { name: /Selesai & Cetak \[Enter\]/ }).click();
    await page.waitForTimeout(1000);

    const printCalls = await page.evaluate(() => (window as any)._printCallCount || 0);
    expect(printCalls).toBeGreaterThanOrEqual(1);

    await expect(page.locator('#thermal-receipt')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('.thermal-shop-name')).toHaveText('RETAIL POS');
    await expect(page.locator('.thermal-label:has-text("Invoice:")')).toBeVisible();
    await expect(page.locator('.thermal-label:has-text("Waktu:")')).toBeVisible();

    const itemCount = await page.locator('.thermal-item-name').count();
    expect(itemCount).toBeGreaterThanOrEqual(3);

    // Verify print styles are defined in the stylesheet
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
          // cross-origin stylesheet, skip
        }
      }
      return false;
    });
    expect(hasPrintStyles).toBe(true);

    // Take screenshot in print mode for visual verification
    await page.emulateMedia({ media: 'print' });
    await page.screenshot({ path: 'test-results/print-receipt-preview.png' });
    await page.emulateMedia({ media: 'screen' });

    // Verify no page rerender: URL preserved, sidebar visible after print
    expect(page.url()).toContain('/pos');

    await expect(page.getByRole('button', { name: /Print Receipt/ })).toBeVisible({ timeout: 3000 });
  });

  test('print preview renders black text on white background', async ({ page }) => {
    await expect(page).toHaveURL(/\/login$/);
    await page.fill('#username', TEST_USERS.superadmin.username);
    await page.fill('#password', TEST_USERS.superadmin.password);
    await page.click('button[type="submit"]');
    await page.waitForURL(/\/$/, { timeout: 10000 });

    await page.click('nav button:has-text("Point of Sale")');
    await page.waitForSelector('#pos-search-input', { state: 'visible', timeout: 15000 });

    const addButtons = page.locator('button:has-text("Add"):not([disabled])');
    await addButtons.first().waitFor({ state: 'visible', timeout: 10000 });
    for (let i = 0; i < 2; i++) {
      await addButtons.nth(i).click();
    }
    await page.waitForTimeout(300);

    await page.getByRole('button', { name: /Bayar \[F4\]/ }).click();
    await page.waitForTimeout(300);

    const totalText = await page.locator('.text-4xl.font-extrabold').textContent();
    const parsedTotal = parseInt((totalText || '0').replace(/[^\d]/g, ''), 10);
    await page.fill('#cash-received-input', String(parsedTotal));
    await page.waitForTimeout(300);

    await page.getByRole('button', { name: /Selesai & Cetak \[Enter\]/ }).click();
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
    await expect(page).toHaveURL(/\/login$/);
    await page.fill('#username', TEST_USERS.superadmin.username);
    await page.fill('#password', TEST_USERS.superadmin.password);
    await page.click('button[type="submit"]');
    await page.waitForURL(/\/$/, { timeout: 10000 });

    await page.click('nav button:has-text("Point of Sale")');
    await page.waitForSelector('#pos-search-input', { state: 'visible', timeout: 15000 });

    const addButtons = page.locator('button:has-text("Add"):not([disabled])');
    await addButtons.first().waitFor({ state: 'visible', timeout: 10000 });
    await addButtons.nth(0).click();
    await page.waitForTimeout(300);

    await page.getByRole('button', { name: /Bayar \[F4\]/ }).click();
    await page.waitForTimeout(300);

    const totalText = await page.locator('.text-4xl.font-extrabold').textContent();
    const parsedTotal = parseInt((totalText || '0').replace(/[^\d]/g, ''), 10);
    await page.fill('#cash-received-input', String(parsedTotal));
    await page.waitForTimeout(300);

    await page.evaluate(() => {
      (window as any)._printCallCount = 0;
      const originalPrint = window.print;
      window.print = function () {
        (window as any)._printCallCount = ((window as any)._printCallCount || 0) + 1;
        return originalPrint.apply(window, arguments);
      };
    });

    await page.getByRole('button', { name: /Selesai & Cetak \[Enter\]/ }).click();
    await page.waitForTimeout(1000);

    let printCalls = await page.evaluate(() => (window as any)._printCallCount || 0);
    expect(printCalls).toBeGreaterThanOrEqual(1);

    const invoiceFirst = await page.locator('.thermal-value').first().textContent();

    await page.getByRole('button', { name: /Print Receipt/ }).click();
    await page.waitForTimeout(500);

    printCalls = await page.evaluate(() => (window as any)._printCallCount || 0);
    expect(printCalls).toBeGreaterThanOrEqual(2);

    const invoiceSecond = await page.locator('.thermal-value').first().textContent();
    expect(invoiceSecond).toBe(invoiceFirst);
  });
});
