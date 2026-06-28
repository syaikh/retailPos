import { test, expect, type Page } from '@playwright/test';
import { TEST_USERS, API_BASE, waitForAPI, authHeader } from './fixtures';

async function ensureSectionExpanded(page: Page, name: string) {
  const btn = page.locator('aside').locator('button').filter({ hasText: name });
  const expanded = await btn.getAttribute('aria-expanded');
  if (expanded !== 'true') {
    await btn.click();
    await page.waitForTimeout(300);
  }
}

test.describe('Audit Logs Search', () => {
  let authToken: string;

  test.beforeEach(async ({ page }) => {
    // Wait for backend to be ready
    await waitForAPI(page);

    // Login as superadmin
    await page.goto('/');
    await page.fill('#username', TEST_USERS.superadmin.username);
    await page.fill('#password', TEST_USERS.superadmin.password);
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL(/\/$/, { timeout: 10000 });

    // Get the auth token for direct API calls
    authToken = await page.evaluate(() => sessionStorage.getItem('access_token')) || '';
    expect(authToken).toBeTruthy();

    // Navigate to Audit Logs page
    await ensureSectionExpanded(page, 'Administration');
    await page.click('button:has-text("Audit Logs")');
    await expect(page.locator('h1:text("Audit Logs")')).toBeVisible({ timeout: 10000 });
  });

  /**
   * Helper: call the audit-logs API directly with a search term.
   */
  async function apiSearch(term: string, extraParams = '') {
    const url = `${API_BASE}/api/audit-logs?limit=50&offset=0&search=${encodeURIComponent(term)}${extraParams}`;
    const res = await fetch(url, { headers: authHeader(authToken) });
    expect(res.ok).toBeTruthy();
    return await res.json();
  }

  // ─── Basic page load ────────────────────────────────────────────────

  test('should load audit logs page with table visible', async ({ page }) => {
    await expect(page.locator('th:text("Timestamp")')).toBeVisible();
    await expect(page.locator('th:text("Actor")')).toBeVisible();
    await expect(page.locator('th:text("Resource")')).toBeVisible();
    await expect(page.locator('th:text("Action")')).toBeVisible();
    await expect(page.locator('th:text("Description")')).toBeVisible();
    await expect(page.locator('th:text("IP Address")')).toBeVisible();
  });

  test('should show search input and filter controls', async ({ page }) => {
    await expect(page.locator('input[placeholder="Search by actor, role, action, entity, or IP..."]')).toBeVisible();
    await expect(page.locator('.date-picker-container')).toBeVisible();
    await expect(page.locator('#resource-dropdown-container')).toBeVisible();
    await expect(page.locator('#action-dropdown-container')).toBeVisible();
  });

  // ─── Search correctness via direct API verification ─────────────────

  test('searching "login" returns only records where login appears in searchable columns', async () => {
    const body = await apiSearch('login');
    expect(body.data).toBeInstanceOf(Array);
    expect(body.data.length).toBeGreaterThan(0);

    for (const log of body.data) {
      const username = (log.username || '').toLowerCase();
      const role = (log.role || '').toLowerCase();
      const action = (log.action || '').toLowerCase();
      const entityType = (log.entity_type || '').toLowerCase();
      const ip = (log.ip_address || '').toLowerCase();
      const combined = `${username} ${role} ${action} ${entityType} ${ip}`;
      expect(combined).toContain('login');
    }
  });

  test('searching "auth" returns only records where auth appears in searchable columns', async () => {
    const body = await apiSearch('auth');
    expect(body.data).toBeInstanceOf(Array);
    expect(body.data.length).toBeGreaterThan(0);

    for (const log of body.data) {
      const username = (log.username || '').toLowerCase();
      const role = (log.role || '').toLowerCase();
      const action = (log.action || '').toLowerCase();
      const entityType = (log.entity_type || '').toLowerCase();
      const ip = (log.ip_address || '').toLowerCase();
      const combined = `${username} ${role} ${action} ${entityType} ${ip}`;
      expect(combined).toContain('auth');
    }
  });

  test('searching "superadmin" returns only records where superadmin appears', async () => {
    const body = await apiSearch('superadmin');
    expect(body.data).toBeInstanceOf(Array);
    expect(body.data.length).toBeGreaterThan(0);

    for (const log of body.data) {
      const username = (log.username || '').toLowerCase();
      const role = (log.role || '').toLowerCase();
      const action = (log.action || '').toLowerCase();
      const entityType = (log.entity_type || '').toLowerCase();
      const ip = (log.ip_address || '').toLowerCase();
      const combined = `${username} ${role} ${action} ${entityType} ${ip}`;
      expect(combined).toContain('superadmin');
    }
  });

  test('searching "cashier" returns only records where cashier appears', async () => {
    const body = await apiSearch('cashier');
    expect(body.data).toBeInstanceOf(Array);
    expect(body.data.length).toBeGreaterThan(0);

    for (const log of body.data) {
      const username = (log.username || '').toLowerCase();
      const role = (log.role || '').toLowerCase();
      const action = (log.action || '').toLowerCase();
      const entityType = (log.entity_type || '').toLowerCase();
      const ip = (log.ip_address || '').toLowerCase();
      const combined = `${username} ${role} ${action} ${entityType} ${ip}`;
      expect(combined).toContain('cashier');
    }
  });

  test('searching "update" returns only records with update action', async () => {
    const body = await apiSearch('update');
    expect(body.data).toBeInstanceOf(Array);

    for (const log of body.data) {
      const username = (log.username || '').toLowerCase();
      const role = (log.role || '').toLowerCase();
      const action = (log.action || '').toLowerCase();
      const entityType = (log.entity_type || '').toLowerCase();
      const ip = (log.ip_address || '').toLowerCase();
      const combined = `${username} ${role} ${action} ${entityType} ${ip}`;
      expect(combined).toContain('update');
    }
  });

  test('searching "create" returns only records with create action', async () => {
    const body = await apiSearch('create');
    expect(body.data).toBeInstanceOf(Array);

    for (const log of body.data) {
      const username = (log.username || '').toLowerCase();
      const role = (log.role || '').toLowerCase();
      const action = (log.action || '').toLowerCase();
      const entityType = (log.entity_type || '').toLowerCase();
      const ip = (log.ip_address || '').toLowerCase();
      const combined = `${username} ${role} ${action} ${entityType} ${ip}`;
      expect(combined).toContain('create');
    }
  });

  test('searching "user" returns only records with user resource type', async () => {
    const body = await apiSearch('user');
    expect(body.data).toBeInstanceOf(Array);

    for (const log of body.data) {
      const username = (log.username || '').toLowerCase();
      const role = (log.role || '').toLowerCase();
      const action = (log.action || '').toLowerCase();
      const entityType = (log.entity_type || '').toLowerCase();
      const ip = (log.ip_address || '').toLowerCase();
      const combined = `${username} ${role} ${action} ${entityType} ${ip}`;
      // "user" should match entity_type=user OR username containing "user" OR role containing "user"
      expect(combined).toContain('user');
    }
  });

  test('searching nonexistent term returns empty results', async () => {
    const body = await apiSearch('zzzznonexistent999zzz');
    expect(body.data).toBeInstanceOf(Array);
    expect(body.data.length).toBe(0);
    expect(body.total).toBe(0);
  });

  // ─── UI search: type in search bar and verify DOM updates ───────────

  test('typing in search bar filters results in the UI', async ({ page }) => {
    // Clear any existing search
    await page.fill('input[placeholder="Search by actor, role, action, entity, or IP..."]', '');
    await page.waitForTimeout(1500);

    // Type "login" in the search bar
    await page.fill('input[placeholder="Search by actor, role, action, entity, or IP..."]', 'login');
    await page.waitForTimeout(2000);

    // Pagination should show results (not 0)
    const paginationText = await page.locator('text=/Showing.*of/').textContent();
    expect(paginationText).toBeTruthy();
    // Should not be "Showing 0-0 of 0"
    expect(paginationText).not.toContain('0-0 of 0');
  });

  test('searching nonexistent term in UI shows empty state', async ({ page }) => {
    await page.fill('input[placeholder="Search by actor, role, action, entity, or IP..."]', '');
    await page.waitForTimeout(1500);

    await page.fill('input[placeholder="Search by actor, role, action, entity, or IP..."]', 'zzzznonexistent999zzz');
    await page.waitForTimeout(2500);

    const emptyState = page.locator('text=No audit logs found');
    const showingZero = page.locator('text=Showing 0-0 of 0');

    const hasEmpty = await emptyState.isVisible().catch(() => false);
    const hasZeroPagination = await showingZero.isVisible().catch(() => false);
    expect(hasEmpty || hasZeroPagination).toBeTruthy();
  });

  test('clearing search in UI restores results', async ({ page }) => {
    // Search for something with no results
    await page.fill('input[placeholder="Search by actor, role, action, entity, or IP..."]', 'zzzznonexistent999zzz');
    await page.waitForTimeout(2500);

    // Clear the search
    await page.fill('input[placeholder="Search by actor, role, action, entity, or IP..."]', '');
    await page.waitForTimeout(2000);

    // Should see results again
    const rows = page.locator('tbody tr');
    const rowCount = await rows.count();
    expect(rowCount).toBeGreaterThan(0);
  });

  // ─── Resource filter dropdown ───────────────────────────────────────

  test('filtering by resource "auth" shows only auth rows in UI', async ({ page }) => {
    await page.locator('#resource-dropdown-container button').click();
    await page.waitForTimeout(300);
    await page.locator('#resource-dropdown-container button:has-text("Auth")').click();
    await page.waitForTimeout(1500);

    const resourceCells = await page.locator('td:nth-child(3)').allTextContents();
    expect(resourceCells.length).toBeGreaterThan(0);
    for (const cell of resourceCells) {
      expect(cell.toLowerCase()).toContain('auth');
    }
  });

  // ─── Action filter dropdown ─────────────────────────────────────────

  test('filtering by action "login" shows only login rows in UI', async ({ page }) => {
    await page.locator('#action-dropdown-container button').click();
    await page.waitForTimeout(300);
    await page.locator('#action-dropdown-container button:has-text("Login")').click();
    await page.waitForTimeout(1500);

    const actionCells = await page.locator('td:nth-child(4)').allTextContents();
    expect(actionCells.length).toBeGreaterThan(0);
    for (const cell of actionCells) {
      expect(cell.toLowerCase()).toContain('login');
    }
  });

  // ─── Date range filter ──────────────────────────────────────────────

  test('changing date range filter works without errors', async ({ page }) => {
    await page.locator('.date-picker-container button.date-picker-trigger').click();
    await expect(page.locator('button:has-text("Last 7 Days")')).toBeVisible();

    await page.click('button:has-text("Last 7 Days")');
    await page.waitForTimeout(1500);

    const errorToast = page.locator('text=Failed to load audit logs');
    await expect(errorToast).toBeHidden({ timeout: 3000 });
  });

  // ─── Combined search + filter ───────────────────────────────────────

  test('combining search text with resource filter returns correct results via API', async () => {
    const body = await apiSearch('superadmin', '&entity_type=auth');
    expect(body.data).toBeInstanceOf(Array);
    expect(body.data.length).toBeGreaterThan(0);

    for (const log of body.data) {
      // Must match resource filter
      expect((log.entity_type || '').toLowerCase()).toContain('auth');
      // Must match search term
      const username = (log.username || '').toLowerCase();
      const role = (log.role || '').toLowerCase();
      const action = (log.action || '').toLowerCase();
      const entityType = (log.entity_type || '').toLowerCase();
      const ip = (log.ip_address || '').toLowerCase();
      const combined = `${username} ${role} ${action} ${entityType} ${ip}`;
      expect(combined).toContain('superadmin');
    }
  });

  // ─── Search does not cause 500 error ────────────────────────────────

  test('search should not produce 500 Internal Server Error', async ({ page }) => {
    const errors: string[] = [];
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        errors.push(msg.text());
      }
    });

    await page.fill('input[placeholder="Search by actor, role, action, entity, or IP..."]', 'logg');
    await page.waitForTimeout(2500);

    const errorToast = page.locator('text=Failed to load audit logs');
    await expect(errorToast).toBeHidden({ timeout: 3000 });

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
    await page.waitForTimeout(1500);

    const errorToast = page.locator('text=Failed to load audit logs');
    await expect(errorToast).toBeHidden({ timeout: 3000 });
  });

  // ─── Pagination is visible ──────────────────────────────────────────

  test('should show pagination controls', async ({ page }) => {
    const pagination = page.locator('text=Rows per page:');
    await expect(pagination).toBeVisible({ timeout: 5000 });
  });

  // ─── Non-superadmin cannot access audit logs ────────────────────────

  test('should deny access to non-superadmin users', async ({ page }) => {
    await page.evaluate(() => sessionStorage.clear());
    await page.reload();

    await page.fill('#username', TEST_USERS.cashier.username);
    await page.fill('#password', TEST_USERS.cashier.password);
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL(/\/$/, { timeout: 10000 });

    await page.goto('/admin/audit-logs');
    await page.waitForTimeout(2000);

    const accessDenied = page.locator('text=Access Denied');
    const isOnLogin = page.url().includes('/login');
    const isDenied = await accessDenied.isVisible().catch(() => false);
    expect(isDenied || isOnLogin).toBeTruthy();
  });
});
