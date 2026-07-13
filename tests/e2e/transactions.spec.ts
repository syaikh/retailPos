import { test, expect } from '@playwright/test';
import { TEST_USERS, loginUI, logoutUI } from './fixtures';

test.describe('Transactions Page', () => {
  test.beforeEach(async ({ page }) => {
    await loginUI(page, 'superadmin', 'admin123');
    await page.goto('http://localhost:5173/transactions');
    await expect(page).toHaveURL(/\/transactions/);
    await expect(page.locator('text=INVOICE')).toBeVisible({ timeout: 10000 });
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('should display transaction table with columns', async ({ page }) => {
    await expect(page.locator('text=INVOICE')).toBeVisible();
    await expect(page.locator('text=DATE')).toBeVisible();
    await expect(page.locator('text=CUSTOMER')).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'PAYMENT' })).toBeVisible();
    await expect(page.locator('text=TOTAL (RP)')).toBeVisible();
  });

  test('should search transactions', async ({ page }) => {
    await page.getByPlaceholder('Search by invoice, product, or customer...').fill('INV');
    await page.waitForTimeout(1000);
    await page.getByPlaceholder('Search by invoice, product, or customer...').fill('');
  });

  test('should filter by date range preset', async ({ page }) => {
    await page.locator('button.date-picker-trigger').first().click();
    await expect(page.locator('.date-picker-container')).toBeVisible();

    await page.getByRole('button', { name: 'Last 7 Days' }).click();
    await page.waitForTimeout(500);
  });

  test('should filter by payment method', async ({ page }) => {
    await page.getByRole('button', { name: 'All methods' }).click();
    await expect(page.locator('text=All methods')).toBeVisible();

    await page.getByText('CASH').first().click();
    await page.getByRole('button', { name: 'Clear' }).click();
  });

  test('should open export dropdown', async ({ page }) => {
    await page.locator('button').filter({ hasText: 'Export' }).first().click();
    await expect(page.getByRole('menuitem', { name: 'Export to CSV' })).toBeVisible();
    await expect(page.getByRole('menuitem', { name: 'Export to Excel' })).toBeVisible();
  });

  test('should open transaction detail drawer', async ({ page }) => {
    const firstRow = page.locator('table tbody tr').first();
    await firstRow.waitFor({ state: 'visible', timeout: 10000 });

    await firstRow.click();
    const drawer = page.getByRole('dialog', { name: 'Transaction Details' }).or(page.locator('#transaction-details-heading'));
    await expect(drawer.first()).toBeVisible({ timeout: 5000 });

    await page.locator('button[aria-label="Close drawer"]').click();
    await expect(page.getByRole('dialog', { name: 'Transaction Details' })).toBeHidden({ timeout: 5000 });
  });

  test('should search with no errors', async ({ page }) => {
    await page.getByPlaceholder('Search by invoice, product, or customer...').fill('NONEXISTENT_XYZ_12345');
    await page.waitForTimeout(1000);
    await page.getByPlaceholder('Search by invoice, product, or customer...').fill('');
    await page.waitForTimeout(500);
  });
});
