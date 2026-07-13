import { test, expect } from '@playwright/test';
import { TEST_USERS, loginUI } from './fixtures';

function sidebarLink(page, name) {
  return page.locator('aside').locator('a, button').filter({ hasText: name });
}

async function ensureSectionExpanded(page, name) {
  const btn = page.locator('aside').locator('button').filter({ hasText: name });
  const expanded = await btn.getAttribute('aria-expanded');
  if (expanded !== 'true') {
    await btn.click();
    await page.waitForTimeout(300);
  }
}

test.describe('Sidebar RBAC', () => {
  test('superadmin sees all nav items', async ({ page }) => {
    await loginUI(page, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    await ensureSectionExpanded(page, 'Master Data');
    await ensureSectionExpanded(page, 'Administration');

    await expect(sidebarLink(page, 'Dashboard')).toBeVisible();
    await expect(sidebarLink(page, 'Point of Sale')).toBeVisible();
    await expect(sidebarLink(page, 'Transactions')).toBeVisible();
    await expect(sidebarLink(page, 'Reports')).toBeVisible();

    await expect(sidebarLink(page, 'Products')).toBeVisible();
    await expect(sidebarLink(page, 'Categories')).toBeVisible();
    await expect(sidebarLink(page, 'Brands')).toBeVisible();
    await expect(sidebarLink(page, 'Units')).toBeVisible();
    await expect(sidebarLink(page, 'Customers')).toBeVisible();

    await expect(sidebarLink(page, 'Users')).toBeVisible();
    await expect(sidebarLink(page, 'Roles')).toBeVisible();
    await expect(sidebarLink(page, 'Audit Logs')).toBeVisible();
  });

  test('admin sees all except audit logs', async ({ page }) => {
    await loginUI(page, TEST_USERS.admin.username, TEST_USERS.admin.password);
    await ensureSectionExpanded(page, 'Master Data');
    await ensureSectionExpanded(page, 'Administration');

    await expect(sidebarLink(page, 'Dashboard')).toBeVisible();
    await expect(sidebarLink(page, 'Point of Sale')).toBeVisible();
    await expect(sidebarLink(page, 'Transactions')).toBeVisible();
    await expect(sidebarLink(page, 'Reports')).toBeVisible();

    await expect(sidebarLink(page, 'Products')).toBeVisible();
    await expect(sidebarLink(page, 'Categories')).toBeVisible();
    await expect(sidebarLink(page, 'Brands')).toBeVisible();
    await expect(sidebarLink(page, 'Units')).toBeVisible();
    await expect(sidebarLink(page, 'Customers')).toBeVisible();

    await expect(sidebarLink(page, 'Users')).toBeVisible();
    await expect(sidebarLink(page, 'Roles')).toBeVisible();
    await expect(sidebarLink(page, 'Audit Logs')).toBeHidden();
  });

  test('manager sees no admin section', async ({ page }) => {
    await loginUI(page, TEST_USERS.manager.username, TEST_USERS.manager.password);
    await ensureSectionExpanded(page, 'Master Data');

    await expect(sidebarLink(page, 'Dashboard')).toBeVisible();
    await expect(sidebarLink(page, 'Transactions')).toBeVisible();
    await expect(sidebarLink(page, 'Reports')).toBeVisible();

    await expect(sidebarLink(page, 'Products')).toBeVisible();
    await expect(sidebarLink(page, 'Categories')).toBeVisible();
    await expect(sidebarLink(page, 'Brands')).toBeVisible();
    await expect(sidebarLink(page, 'Units')).toBeVisible();
    await expect(sidebarLink(page, 'Customers')).toBeVisible();

    await expect(sidebarLink(page, 'Point of Sale')).toBeHidden();
    await expect(sidebarLink(page, 'Users')).toBeHidden();
    await expect(sidebarLink(page, 'Roles')).toBeHidden();
    await expect(sidebarLink(page, 'Audit Logs')).toBeHidden();
  });

  test('cashier sees only POS, transactions, and dashboard', async ({ page }) => {
    await loginUI(page, TEST_USERS.cashier.username, TEST_USERS.cashier.password);

    await expect(sidebarLink(page, 'Dashboard')).toBeVisible();
    await expect(sidebarLink(page, 'Point of Sale')).toBeVisible();
    await expect(sidebarLink(page, 'Transactions')).toBeVisible();

    await expect(sidebarLink(page, 'Reports')).toBeHidden();
    await expect(sidebarLink(page, 'Products')).toBeHidden();
    await expect(sidebarLink(page, 'Categories')).toBeHidden();
    await expect(sidebarLink(page, 'Brands')).toBeHidden();
    await expect(sidebarLink(page, 'Units')).toBeHidden();
    await expect(sidebarLink(page, 'Customers')).toBeHidden();
    await expect(sidebarLink(page, 'Users')).toBeHidden();
    await expect(sidebarLink(page, 'Roles')).toBeHidden();
    await expect(sidebarLink(page, 'Audit Logs')).toBeHidden();
  });

  test('staff sees only dashboard and products', async ({ page }) => {
    await loginUI(page, 'staff', 'admin123');
    await ensureSectionExpanded(page, 'Master Data');

    await expect(sidebarLink(page, 'Dashboard')).toBeVisible();
    await expect(sidebarLink(page, 'Products')).toBeVisible();

    await expect(sidebarLink(page, 'Point of Sale')).toBeHidden();
    await expect(sidebarLink(page, 'Transactions')).toBeHidden();
    await expect(sidebarLink(page, 'Reports')).toBeHidden();
    await expect(sidebarLink(page, 'Categories')).toBeHidden();
    await expect(sidebarLink(page, 'Brands')).toBeHidden();
    await expect(sidebarLink(page, 'Units')).toBeHidden();
    await expect(sidebarLink(page, 'Customers')).toBeHidden();
    await expect(sidebarLink(page, 'Users')).toBeHidden();
    await expect(sidebarLink(page, 'Roles')).toBeHidden();
    await expect(sidebarLink(page, 'Audit Logs')).toBeHidden();
  });
});
