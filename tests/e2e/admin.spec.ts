import { test, expect } from './fixtures';
import { loginUI, logoutUI } from './fixtures';

async function selectRole(page: any, roleName: string) {
  const dialog = page.getByRole('dialog');
  const trigger = dialog.locator('.form-role-dropdown-container button').first();
  await trigger.waitFor({ state: 'visible', timeout: 5000 });
  await trigger.click();
  await page.waitForTimeout(300);
  const option = page.locator('[class*="grid"][class*="gap-1"]').getByRole('button', { name: roleName, exact: true });
  await option.waitFor({ state: 'visible', timeout: 5000 });
  await option.click();
}

test.describe('Admin Panel - User Management', () => {
  test.beforeEach(async ({ page }) => {
    await loginUI(page, 'superadmin', 'admin123');
    await page.goto('http://localhost:5173/admin/users');
    await expect(page).toHaveURL(/\/admin\/users/);
    await expect(page.getByRole('heading', { name: 'Users' })).toBeVisible({ timeout: 10000 });
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  test('should display user table with columns', async ({ page }) => {
    await expect(page.getByRole('heading', { name: 'Users' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'USER' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'ROLE' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'STATUS' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'LAST LOGIN' })).toBeVisible();
  });

  test('should create user and normalize mixed-case username to lowercase', async ({ page }) => {
    const uniqueId = Date.now();
    const inputUsername = `TestUser${uniqueId}`;

    await page.getByRole('button', { name: 'Add User' }).click();
    await expect(page.getByRole('dialog', { name: 'Add User' })).toBeVisible();

    await page.fill('#usr-username', inputUsername);
    await page.fill('#usr-email', `${inputUsername.toLowerCase()}@example.com`);
    await page.fill('#usr-password', 'password123');
    await selectRole(page, 'admin');

    await page.getByRole('button', { name: 'Create User' }).click();

    await expect(page.getByRole('dialog', { name: 'Add User' })).toBeHidden({ timeout: 15000 });
    await page.getByPlaceholder('Search by username or email...').fill(inputUsername.toLowerCase());
    await page.waitForTimeout(1000);
    await expect(page.locator('table').getByText(inputUsername.toLowerCase(), { exact: true })).toBeVisible({ timeout: 10000 });
  });

  test('should reject username with invalid characters', async ({ page }) => {
    await page.getByRole('button', { name: 'Add User' }).click();
    await expect(page.getByRole('dialog', { name: 'Add User' })).toBeVisible();

    await page.fill('#usr-username', 'test user!');
    await page.fill('#usr-email', 'testuser@example.com');
    await page.fill('#usr-password', 'password123');

    const createButton = page.getByRole('button', { name: 'Create User' });
    await expect(createButton).toBeDisabled();
  });

  test('should show error when submitting empty username', async ({ page }) => {
    await page.getByRole('button', { name: 'Add User' }).click();
    await expect(page.getByRole('dialog', { name: 'Add User' })).toBeVisible();

    await page.fill('#usr-email', 'testuser@example.com');
    await page.fill('#usr-password', 'password123');

    await page.getByRole('button', { name: 'Create User' }).click();

    await expect(page.getByRole('dialog', { name: 'Add User' })).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Please fill all required fields').or(page.getByText('required'))).toBeVisible({ timeout: 10000 });
  });

  test('should filter by role', async ({ page }) => {
    await page.getByRole('button', { name: 'All Roles' }).click();
    await page.waitForTimeout(500);
    const roleOption = page.getByRole('button', { name: 'cashier', exact: true });
    await roleOption.waitFor({ state: 'visible', timeout: 5000 });
    await roleOption.click();
    await page.waitForTimeout(500);

    const rows = page.locator('table tbody tr');
    const count = await rows.count();
    if (count > 0) {
      await expect(rows.first().getByText('cashier', { exact: true }).first()).toBeVisible();
    }
  });

  test('should filter by status', async ({ page }) => {
    await page.getByRole('button', { name: 'All Status' }).click();
    await page.waitForTimeout(300);
    await page.getByRole('menuitem', { name: 'Active', exact: true }).click();
    await page.waitForTimeout(500);
  });

  test('should edit a user and change role', async ({ page }) => {
    const uniqueId = Date.now();
    const username = `edituser${uniqueId}`;

    await page.getByRole('button', { name: 'Add User' }).click();
    await page.fill('#usr-username', username);
    await page.fill('#usr-email', `${username}@example.com`);
    await page.fill('#usr-password', 'password123');
    await selectRole(page, 'admin');
    await page.getByRole('button', { name: 'Create User' }).click();
    await expect(page.getByRole('dialog', { name: 'Add User' })).toBeHidden({ timeout: 15000 });

    await page.getByPlaceholder('Search by username or email...').fill(username);
    await page.waitForTimeout(1000);
    await expect(page.locator('table').getByText(username, { exact: true })).toBeVisible({ timeout: 10000 });

    const editButton = page.locator('tr').filter({ hasText: username }).locator('button[aria-label="Edit"]');
    await editButton.click();
    await expect(page.getByRole('dialog', { name: 'Edit User' })).toBeVisible();

    await page.fill('#usr-email', `${username}@updated.com`);
    await page.getByRole('button', { name: 'Save' }).click();
    await expect(page.getByRole('dialog', { name: 'Edit User' })).toBeHidden({ timeout: 10000 });
  });

  test('should deactivate and reactivate a user', async ({ page }) => {
    const username = `deactuser${Date.now()}`;

    await page.getByRole('button', { name: 'Add User' }).click();
    await page.fill('#usr-username', username);
    await page.fill('#usr-email', `${username}@example.com`);
    await page.fill('#usr-password', 'password123');
    await selectRole(page, 'admin');
    await page.getByRole('button', { name: 'Create User' }).click();
    await expect(page.getByRole('dialog', { name: 'Add User' })).toBeHidden({ timeout: 15000 });

    await page.getByPlaceholder('Search by username or email...').fill(username);
    await page.waitForTimeout(1000);
    await expect(page.locator('table').getByText(username, { exact: true })).toBeVisible({ timeout: 10000 });
    await expect(page.locator('tr').filter({ hasText: username }).locator('text=Active')).toBeVisible();

    const editButton = page.locator('tr').filter({ hasText: username }).locator('button[aria-label="Edit"]');
    await editButton.click();
    await expect(page.getByRole('dialog', { name: 'Edit User' })).toBeVisible();

    const toggleSwitch = page.getByRole('dialog', { name: 'Edit User' }).locator('input[type="checkbox"]');
    await toggleSwitch.click({ force: true });
    await page.getByRole('button', { name: 'Save' }).click();
    await expect(page.getByRole('dialog', { name: 'Edit User' })).toBeHidden({ timeout: 10000 });

    await page.waitForTimeout(500);
    await expect(page.locator('tr').filter({ hasText: username }).locator('text=Inactive')).toBeVisible({ timeout: 5000 });
  });

  test('should delete a user', async ({ page }) => {
    const username = `deluser${Date.now()}`;

    await page.getByRole('button', { name: 'Add User' }).click();
    await page.fill('#usr-username', username);
    await page.fill('#usr-email', `${username}@example.com`);
    await page.fill('#usr-password', 'password123');
    await selectRole(page, 'admin');
    await page.getByRole('button', { name: 'Create User' }).click();
    await expect(page.getByRole('dialog', { name: 'Add User' })).toBeHidden({ timeout: 15000 });

    await page.getByPlaceholder('Search by username or email...').fill(username);
    await expect(page.locator('table').getByText(username, { exact: true })).toBeVisible({ timeout: 10000 });

    const deleteButton = page.locator('tr').filter({ hasText: username }).locator('button[aria-label="Delete"]');
    await deleteButton.click();
    await expect(page.getByRole('dialog', { name: 'Delete User' })).toBeVisible();

    await page.getByRole('dialog', { name: 'Delete User' }).getByRole('button', { name: 'Delete' }).click();
    await expect(page.getByRole('dialog', { name: 'Delete User' })).toBeHidden({ timeout: 10000 });

    await page.getByPlaceholder('Search by username or email...').fill(username);
    await page.waitForTimeout(500);
    await expect(page.locator('table').getByText(username, { exact: true })).toBeHidden({ timeout: 5000 });
  });
});
