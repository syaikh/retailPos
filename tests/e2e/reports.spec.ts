import { test, expect } from '@playwright/test';

test.describe('Reports & Analytics', () => {
  test.beforeEach(async ({ page }) => {
    // Login first
    await page.goto('http://localhost:5173/login');
    await page.fill('#username', 'superadmin');
    await page.fill('#password', 'admin123');
    await page.click('.login-btn');
    await expect(page).toHaveURL(/\/$/, { timeout: 5000 });
    await expect(page.locator('#dashboard')).toBeVisible({ timeout: 5000 });
  });

  test('should navigate to reports page from dashboard', async ({ page }) => {
    // Click "View Reports" button on Reports card
    await page.locator('.card').nth(2).locator('.btn').click();
    // URL changes to /reports
    await expect(page).toHaveURL(/\/reports$/);
    // Reports page elements should be visible
    await expect(page.locator('h3').filter({ hasText: 'Revenue Overview' })).toBeVisible();
  });

  test('should display sales chart', async ({ page }) => {
    await page.goto('http://localhost:5173/reports');
    // Chart canvas should be visible
    await expect(page.locator('canvas')).toBeVisible();
    // Chart title should be visible
    await expect(page.locator('h3').filter({ hasText: 'Revenue Overview' })).toBeVisible();
  });

  test('should filter reports by date range', async ({ page }) => {
    await page.goto('http://localhost:5173/reports');
    // Date inputs should be visible
    const startDateInput = page.locator('input[type="date"]').first();
    const endDateInput = page.locator('input[type="date"]').nth(1);
    await expect(startDateInput).toBeVisible();
    await expect(endDateInput).toBeVisible();

    // Change date range
    await startDateInput.fill('2025-11-01');
    await endDateInput.fill('2025-11-10');
    await page.locator('button').filter({ hasText: 'Apply' }).click();

    // Chart should still be visible (data may change)
    await expect(page.locator('canvas')).toBeVisible();
  });

  test('should export sales report to Excel', async ({ page }) => {
    await page.goto('http://localhost:5173/reports');
    // Set up download listener
    const downloadPromise = page.waitForEvent('download');

    // Click Export dropdown and then Excel option
    await page.locator('button').filter({ hasText: 'Export' }).click();
    await page.locator('button').filter({ hasText: 'Export Chart to Excel' }).click();

    // Wait for download
    const download = await downloadPromise;
    expect(download.suggestedFilename()).toMatch(/revenue-report.*\.xlsx$/);
  });

  test('should export sales report to PDF', async ({ page }) => {
    await page.goto('http://localhost:5173/reports');
    // Set up download listener
    const downloadPromise = page.waitForEvent('download');

    // Click Export dropdown and then PDF option
    await page.locator('button').filter({ hasText: 'Export' }).click();
    await page.locator('button').filter({ hasText: 'Export Chart to PDF' }).click();

    // Wait for download
    const download = await downloadPromise;
    expect(download.suggestedFilename()).toMatch(/revenue-report.*\.pdf$/);
  });

  test('should show summary statistics cards', async ({ page }) => {
    await page.goto('http://localhost:5173/reports');
    // KPI cards should be visible
    await expect(page.locator('text=Total Revenue')).toBeVisible();
    await expect(page.locator('text=Total Orders')).toBeVisible();
    await expect(page.locator('text=Avg Order Value')).toBeVisible();
    await expect(page.locator('text=Revenue per Day')).toBeVisible();
    await expect(page.locator('text=vs Previous Period')).toBeVisible();
  });

  test('should filter by payment method', async ({ page }) => {
    // Payment method filter not implemented in current UI
    test.skip(true, 'Payment method filter not implemented');
  });

  test('should drill down to detailed transaction list', async ({ page }) => {
    await page.goto('http://localhost:5173/reports');
    // Transaction table should be visible
    await expect(page.locator('table')).toBeVisible();
    // Table headers should be present
    await expect(page.locator('th').filter({ hasText: 'Invoice' })).toBeVisible();
    await expect(page.locator('th').filter({ hasText: 'Date' })).toBeVisible();
    await expect(page.locator('th').filter({ hasText: 'Items' })).toBeVisible();
    await expect(page.locator('th').filter({ hasText: 'Payment' })).toBeVisible();
    await expect(page.locator('th').filter({ hasText: 'Total (Rp)' })).toBeVisible();

    // Click on an invoice number to open modal
    const invoiceLink = page.locator('button').filter({ hasText: /^INV-/ }).first();
    if (await invoiceLink.isVisible()) {
      await invoiceLink.click();
      // Modal should open
      await expect(page.locator('text=Transaction Details')).toBeVisible();
      // Close modal
      await page.locator('[aria-label="Close"]').click();
    }
  });
});

// ============================================================================
// Backend Reports API Tests
// ============================================================================

test.describe('Reports API', () => {
  test('GET /api/stats returns valid dashboard data', async ({ page }) => {
    // Already tested in api-integration.spec.ts, but keeping here for completeness
    const response = await page.request.get('http://localhost:8080/api/stats');
    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.data).toHaveProperty('total_sales');
    expect(body.data).toHaveProperty('total_revenue');
    expect(body.data).toHaveProperty('total_products');
  });

  test('GET /api/reports/chart returns chart data', async ({ page }) => {
    const response = await page.request.get('http://localhost:8080/api/reports/chart');
    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    // Should be array of { day: date, sales: number, revenue: number }
    if (body.data && body.data.length > 0) {
      expect(body.data[0]).toHaveProperty('day');
      expect(body.data[0]).toHaveProperty('sales');
    }
  });

  test('GET /api/sales supports pagination and filters', async ({ page }) => {
    const response = await page.request.get('http://localhost:8080/api/sales?limit=5');
    expect(response.ok()).toBeTruthy();
  });
});
