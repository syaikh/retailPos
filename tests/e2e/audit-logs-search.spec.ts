import { test, expect, type Page } from '@playwright/test';
import { TEST_USERS, API_BASE, waitForAPI, authHeader, loginUI, logoutUI, getToken } from './fixtures';

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

  test.beforeEach(async ({ page, request }) => {
    await waitForAPI(page);

    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    // Get the auth token for direct API calls
    authToken = await page.evaluate(() => sessionStorage.getItem('access_token')) || '';
    expect(authToken).toBeTruthy();

    // Navigate to Audit Logs page
    await ensureSectionExpanded(page, 'Administration');
    await page.click('button:has-text("Audit Logs")');
    await expect(page.locator('h1:text("Audit Logs")')).toBeVisible({ timeout: 10000 });
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  /**
   * Helper: call the audit-logs API directly via Playwright request context.
   */
  async function apiSearch(request: any, term: string, extraParams = '') {
    const url = `${API_BASE}/api/audit-logs?limit=50&offset=0&search=${encodeURIComponent(term)}${extraParams}`;
    const res = await request.get(url, { headers: authHeader(authToken) });
    expect(res.ok()).toBeTruthy();
    return await res.json();
  }

  // --- Basic page load ---

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
    await expect(page.locator('button:has-text("All Resources")')).toBeVisible();
    await expect(page.locator('button:has-text("All Actions")')).toBeVisible();
  });

  // --- Search correctness via direct API verification ---

  test('searching "login" returns only records where login appears in searchable columns', async ({ request }) => {
    const body = await apiSearch(request, 'login');
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

  test('searching "auth" returns only records where auth appears in searchable columns', async ({ request }) => {
    const body = await apiSearch(request, 'auth');
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

  test('searching "superadmin" returns only records where superadmin appears', async ({ request }) => {
    const body = await apiSearch(request, 'superadmin');
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

  test('searching "cashier" returns only records where cashier appears', async ({ request }) => {
    const body = await apiSearch(request, 'cashier');
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

  test('searching "update" returns only records with update action', async ({ request }) => {
    const body = await apiSearch(request, 'update');
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

  test('searching "create" returns only records with create action', async ({ request }) => {
    const body = await apiSearch(request, 'create');
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

  test('searching "user" returns only records with user resource type', async ({ request }) => {
    const body = await apiSearch(request, 'user');
    expect(body.data).toBeInstanceOf(Array);

    for (const log of body.data) {
      const username = (log.username || '').toLowerCase();
      const role = (log.role || '').toLowerCase();
      const action = (log.action || '').toLowerCase();
      const entityType = (log.entity_type || '').toLowerCase();
      const ip = (log.ip_address || '').toLowerCase();
      const combined = `${username} ${role} ${action} ${entityType} ${ip}`;
      expect(combined).toContain('user');
    }
  });

  test('searching nonexistent term returns empty results', async ({ request }) => {
    const body = await apiSearch(request, 'zzzznonexistent999zzz');
    expect(body.data).toBeInstanceOf(Array);
    expect(body.data.length).toBe(0);
    expect(body.total).toBe(0);
  });

  // --- UI search: type in search bar and verify DOM updates ---

  test('typing in search bar filters results in the UI', async ({ page }) => {
    await page.fill('input[placeholder="Search by actor, role, action, entity, or IP..."]', '');
    await page.waitForTimeout(1500);

    await page.fill('input[placeholder="Search by actor, role, action, entity, or IP..."]', 'login');
    await page.waitForTimeout(2000);

    const paginationText = await page.locator('text=/Showing.*of/').textContent();
    expect(paginationText).toBeTruthy();
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
    await page.fill('input[placeholder="Search by actor, role, action, entity, or IP..."]', 'zzzznonexistent999zzz');
    await page.waitForTimeout(2500);

    await page.fill('input[placeholder="Search by actor, role, action, entity, or IP..."]', '');
    await page.waitForTimeout(2000);

    const rows = page.locator('tbody tr');
    const rowCount = await rows.count();
    expect(rowCount).toBeGreaterThan(0);
  });

  // --- Resource filter dropdown ---

  test('filtering by resource "auth" shows only auth rows in UI', async ({ page }) => {
    await page.getByRole('button', { name: 'All Resources' }).first().click();
    await page.waitForTimeout(300);
    await page.locator('[role="menu"] button:has-text("Auth")').click();
    await page.waitForTimeout(1500);

    const resourceCells = await page.locator('td:nth-child(3)').allTextContents();
    expect(resourceCells.length).toBeGreaterThan(0);
    for (const cell of resourceCells) {
      expect(cell.toLowerCase()).toContain('auth');
    }
  });

  // --- Action filter dropdown ---

  test('filtering by action "login" shows only login rows in UI', async ({ page }) => {
    await page.getByRole('button', { name: 'All Actions' }).first().click();
    await page.waitForTimeout(300);
    await page.locator('[role="menu"] button:has-text("Login")').click();
    await page.waitForTimeout(1500);

    const actionCells = await page.locator('td:nth-child(4)').allTextContents();
    expect(actionCells.length).toBeGreaterThan(0);
    for (const cell of actionCells) {
      expect(cell.toLowerCase()).toContain('login');
    }
  });

  // --- Date range filter ---

  test('changing date range filter works without errors', async ({ page }) => {
    await page.locator('.date-picker-container button.date-picker-trigger').click();
    const pickerDropdown = page.getByLabel('Date range picker');
    await expect(pickerDropdown.getByRole('button', { name: 'Last 7 Days' })).toBeVisible();

    await pickerDropdown.getByRole('button', { name: 'Last 7 Days' }).click();
    await page.waitForTimeout(1500);

    const errorToast = page.locator('text=Failed to load audit logs');
    await expect(errorToast).toBeHidden({ timeout: 3000 });
  });

  // --- Combined search + filter ---

  test('combining search text with resource filter returns correct results via API', async ({ request }) => {
    const body = await apiSearch(request, 'superadmin', '&entity_type=auth');
    expect(body.data).toBeInstanceOf(Array);
    expect(body.data.length).toBeGreaterThan(0);

    for (const log of body.data) {
      expect((log.entity_type || '').toLowerCase()).toContain('auth');
      const username = (log.username || '').toLowerCase();
      const role = (log.role || '').toLowerCase();
      const action = (log.action || '').toLowerCase();
      const entityType = (log.entity_type || '').toLowerCase();
      const ip = (log.ip_address || '').toLowerCase();
      const combined = `${username} ${role} ${action} ${entityType} ${ip}`;
      expect(combined).toContain('superadmin');
    }
  });

  // --- Search does not cause 500 error ---

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

  // --- Export button is visible ---

  test('should show export button', async ({ page }) => {
    await expect(page.locator('button:text("Export")')).toBeVisible();
  });

  // --- Refresh button works ---

  test('should refresh data when refresh button is clicked', async ({ page }) => {
    await page.click('button[title="Refresh"]');
    await page.waitForTimeout(1500);

    const errorToast = page.locator('text=Failed to load audit logs');
    await expect(errorToast).toBeHidden({ timeout: 3000 });
  });

  // --- Pagination is visible ---

  test('should show pagination controls', async ({ page }) => {
    const pagination = page.locator('text=Rows per page:');
    await expect(pagination).toBeVisible({ timeout: 5000 });
  });

  // --- Non-superadmin cannot access audit logs ---

  test('should deny access to non-superadmin users', async ({ page }) => {
    await logoutUI(page);

    await page.fill('#username', TEST_USERS.cashier.username);
    await page.fill('#password', TEST_USERS.cashier.password);
    await page.click('button[type="submit"]');
    await page.waitForTimeout(1000);
    await page.waitForFunction(() => {
      const path = window.location.hash || window.location.pathname;
      return path === '/' || path === '' || !path.includes('login');
    }, { timeout: 15000 });

    await page.goto('/admin/audit-logs');
    await page.waitForTimeout(2000);

    await expect(page).not.toHaveURL(/\/admin\/audit-logs/, { timeout: 5000 });
  });
});
