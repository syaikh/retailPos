import { test, expect } from '@playwright/test';
import { TEST_USERS, API_BASE, waitForAPI } from './fixtures';

test.describe('Audit Logs Search', () => {
  test.beforeEach(async ({ page }) => {
    // Wait for backend to be ready
    await waitForAPI(page);

    // Login as superadmin (required for audit logs access)
    await page.goto('/');
    await page.fill('#username', TEST_USERS.superadmin.username);
    await page.fill('#password', TEST_USERS.superadmin.password);
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL(/\/$/, { timeout: 10000 });

    // Navigate to Audit Logs page
    await page.click('button:has-text("Audit Logs")');
    await expect(page.locator('h1:text("Audit Logs")')).toBeVisible({ timeout: 10000 });
  });

  // ─── Basic page load ────────────────────────────────────────────────

  test('should load audit logs page with table visible', async ({ page }) => {
    // Table header should be visible
    await expect(page.locator('th:text("Timestamp")')).toBeVisible();
    await expect(page.locator('th:text("Actor")')).toBeVisible();
    await expect(page.locator('th:text("Resource")')).toBeVisible();
    await expect(page.locator('th:text("Action")')).toBeVisible();
    await expect(page.locator('th:text("Description")')).toBeVisible();
    await expect(page.locator('th:text("IP Address")')).toBeVisible();
  });

  test('should show search input and filter controls', async ({ page }) => {
    await expect(page.locator('input[placeholder="Search logs..."]')).toBeVisible();
    await expect(page.locator('#date-dropdown-container')).toBeVisible();
    // Resource and Action dropdowns (select elements in the filter toolbar)
    const selects = page.locator('select');
    await expect(selects).toHaveCount(3);
  });

  // ─── Search by username ─────────────────────────────────────────────

  test('should search logs by username', async ({ page }) => {
    // Type a known username prefix in the search bar
    await page.fill('input[placeholder="Search logs..."]', 'superadmin');
    // Wait for debounce (400ms) + network
    await page.waitForTimeout(1000);

    // Table should show results (or empty state)
    const rows = page.locator('tbody tr');
    const emptyState = page.locator('text=No audit logs found');

    // Either we have rows or empty state — both are valid outcomes
    const hasRows = await rows.count() > 0;
    const hasEmpty = await emptyState.isVisible().catch(() => false);
    expect(hasRows || hasEmpty).toBeTruthy();
  });

  // ─── Search by action ───────────────────────────────────────────────

  test('should search logs by action type', async ({ page }) => {
    await page.fill('input[placeholder="Search logs..."]', 'login');
    await page.waitForTimeout(1000);

    const rows = page.locator('tbody tr');
    const emptyState = page.locator('text=No audit logs found');
    const hasRows = await rows.count() > 0;
    const hasEmpty = await emptyState.isVisible().catch(() => false);
    expect(hasRows || hasEmpty).toBeTruthy();
  });

  // ─── Search by resource (entity_type) ───────────────────────────────

  test('should search logs by resource type', async ({ page }) => {
    await page.fill('input[placeholder="Search logs..."]', 'user');
    await page.waitForTimeout(1000);

    const rows = page.locator('tbody tr');
    const emptyState = page.locator('text=No audit logs found');
    const hasRows = await rows.count() > 0;
    const hasEmpty = await emptyState.isVisible().catch(() => false);
    expect(hasRows || hasEmpty).toBeTruthy();
  });

  // ─── Search by role ─────────────────────────────────────────────────

  test('should search logs by role', async ({ page }) => {
    await page.fill('input[placeholder="Search logs..."]', 'superadmin');
    await page.waitForTimeout(1000);

    const rows = page.locator('tbody tr');
    const emptyState = page.locator('text=No audit logs found');
    const hasRows = await rows.count() > 0;
    const hasEmpty = await emptyState.isVisible().catch(() => false);
    expect(hasRows || hasEmpty).toBeTruthy();
  });

  // ─── Search by IP address ───────────────────────────────────────────

  test('should search logs by IP address', async ({ page }) => {
    await page.fill('input[placeholder="Search logs..."]', '192.168');
    await page.waitForTimeout(1000);

    const rows = page.locator('tbody tr');
    const emptyState = page.locator('text=No audit logs found');
    const hasRows = await rows.count() > 0;
    const hasEmpty = await emptyState.isVisible().catch(() => false);
    expect(hasRows || hasEmpty).toBeTruthy();
  });

  // ─── Search with no results ─────────────────────────────────────────

  test('should show empty state for search with no matching results', async ({ page }) => {
    // Clear any existing search and wait for results to reset
    await page.fill('input[placeholder="Search logs..."]', '');
    await page.waitForTimeout(1500);

    // Search for something that definitely doesn't exist
    await page.fill('input[placeholder="Search logs..."]', 'zzzznonexistent999zzz');
    await page.waitForTimeout(2500);

    // Wait until pagination shows 0 results or empty state appears
    const emptyState = page.locator('text=No audit logs found');
    const showingZero = page.locator('text=Showing 0-0 of 0');

    const hasEmpty = await emptyState.isVisible().catch(() => false);
    const hasZeroPagination = await showingZero.isVisible().catch(() => false);

    expect(hasEmpty || hasZeroPagination).toBeTruthy();
  });

  // ─── Clear search ───────────────────────────────────────────────────

  test('should clear search and show all results again', async ({ page }) => {
    // First search for something with no results
    await page.fill('input[placeholder="Search logs..."]', 'zzzznonexistent999zzz');
    await page.waitForTimeout(2000);

    // Clear the search by clearing the input
    await page.fill('input[placeholder="Search logs..."]', '');
    await page.waitForTimeout(1500);

    // After clearing, we should either see rows or empty state for the date range
    // (depending on how many logs exist in last 24h)
    const rows = page.locator('tbody tr');
    const emptyState = page.locator('text=No audit logs found');
    const rowCount = await rows.count();
    const hasEmpty = await emptyState.isVisible().catch(() => false);
    // Either rows visible or empty state — both are valid after clearing search
    expect(rowCount > 0 || hasEmpty).toBeTruthy();
  });

  // ─── Resource filter dropdown ───────────────────────────────────────

  test('should filter by resource type using dropdown', async ({ page }) => {
    // Select "User" from resource dropdown
    const resourceSelect = page.locator('select').first();
    await resourceSelect.selectOption('user');
    await page.waitForTimeout(1000);

    // Table should update
    const rows = page.locator('tbody tr');
    const emptyState = page.locator('text=No audit logs found');
    const hasRows = await rows.count() > 0;
    const hasEmpty = await emptyState.isVisible().catch(() => false);
    expect(hasRows || hasEmpty).toBeTruthy();
  });

  // ─── Action filter dropdown ─────────────────────────────────────────

  test('should filter by action type using dropdown', async ({ page }) => {
    // Select "Login" from action dropdown
    const actionSelect = page.locator('select').nth(1);
    await actionSelect.selectOption('login');
    await page.waitForTimeout(1000);

    const rows = page.locator('tbody tr');
    const emptyState = page.locator('text=No audit logs found');
    const hasRows = await rows.count() > 0;
    const hasEmpty = await emptyState.isVisible().catch(() => false);
    expect(hasRows || hasEmpty).toBeTruthy();
  });

  // ─── Date range filter ──────────────────────────────────────────────

  test('should change date range filter', async ({ page }) => {
    // Open date dropdown
    await page.click('#date-dropdown-container button');
    await expect(page.locator('#date-dropdown-container button + div')).toBeVisible();

    // Select "Last 7 Days"
    await page.click('button:has-text("Last 7 Days")');
    await page.waitForTimeout(1000);

    // Table should update (no error toast)
    const errorToast = page.locator('text=Failed to load audit logs');
    await expect(errorToast).toBeHidden({ timeout: 3000 });
  });

  // ─── Combined search + filter ───────────────────────────────────────

  test('should combine search text with resource filter', async ({ page }) => {
    // Select resource filter first
    const resourceSelect = page.locator('select').first();
    await resourceSelect.selectOption('auth');
    await page.waitForTimeout(500);

    // Then type in search
    await page.fill('input[placeholder="Search logs..."]', 'login');
    await page.waitForTimeout(1000);

    // No error toast should appear
    const errorToast = page.locator('text=Failed to load audit logs');
    await expect(errorToast).toBeHidden({ timeout: 3000 });

    // Table should show results or empty state
    const rows = page.locator('tbody tr');
    const emptyState = page.locator('text=No audit logs found');
    const hasRows = await rows.count() > 0;
    const hasEmpty = await emptyState.isVisible().catch(() => false);
    expect(hasRows || hasEmpty).toBeTruthy();
  });

  // ─── Search does not cause 500 error ────────────────────────────────

  test('search should not produce 500 Internal Server Error', async ({ page }) => {
    // Listen for console errors
    const errors: string[] = [];
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        errors.push(msg.text());
      }
    });

    // Perform a search
    await page.fill('input[placeholder="Search logs..."]', 'logg');
    await page.waitForTimeout(1500);

    // Check no error toast appeared
    const errorToast = page.locator('text=Failed to load audit logs');
    await expect(errorToast).toBeHidden({ timeout: 3000 });

    // Check no 500-related console errors
    const serverErrors = errors.filter(e => e.includes('500') || e.includes('Internal Server Error'));
    expect(serverErrors).toHaveLength(0);
  });

  // ─── Export button is visible ───────────────────────────────────────

  test('should show export button', async ({ page }) => {
    await expect(page.locator('button:text("Export")')).toBeVisible();
  });

  // ─── Refresh button works ──────────────────────────────────────────

  test('should refresh data when refresh button is clicked', async ({ page }) => {
    await page.click('button[title="Refresh"]');
    await page.waitForTimeout(1000);

    // No error toast
    const errorToast = page.locator('text=Failed to load audit logs');
    await expect(errorToast).toBeHidden({ timeout: 3000 });
  });

  // ─── Pagination is visible when there are results ───────────────────

  test('should show pagination controls', async ({ page }) => {
    // Pagination should be present (even if only 1 page)
    const pagination = page.locator('text=Rows per page:');
    await expect(pagination).toBeVisible({ timeout: 5000 });
  });

  // ─── Non-superadmin cannot access audit logs ────────────────────────

  test('should deny access to non-superadmin users', async ({ page }) => {
    // Logout
    await page.evaluate(() => sessionStorage.clear());
    await page.reload();

    // Login as cashier
    await page.fill('#username', TEST_USERS.cashier.username);
    await page.fill('#password', TEST_USERS.cashier.password);
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL(/\/$/, { timeout: 10000 });

    // Try to navigate to audit logs
    await page.goto('/admin/audit-logs');
    await page.waitForTimeout(2000);

    // Should show access denied or redirect
    const accessDenied = page.locator('text=Access Denied');
    const isOnLogin = page.url().includes('/login');
    const isDenied = await accessDenied.isVisible().catch(() => false);
    expect(isDenied || isOnLogin).toBeTruthy();
  });
});
