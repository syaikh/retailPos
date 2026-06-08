import { test, expect } from '@playwright/test';

test.describe('Admin Panel - User Management', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('http://localhost:5173/login');
    await page.fill('#username', 'superadmin');
    await page.fill('#password', 'admin123');
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL(/\/$/, { timeout: 10000 });
    await page.goto('http://localhost:5173/admin/users');
    await expect(page).toHaveURL(/\/admin\/users/);
    await expect(page.getByRole('heading', { name: 'User Management' })).toBeVisible({ timeout: 10000 });
  });

  test('should create user and normalize mixed-case username to lowercase', async ({ page }) => {
    const uniqueId = Date.now();
    const inputUsername = `TestUser${uniqueId}`;

    await page.getByRole('button', { name: 'Add User' }).click();
    await expect(page.getByRole('dialog', { name: 'Add New User' })).toBeVisible();

    await page.fill('#usr-username', inputUsername);
    await page.fill('#usr-email', `${inputUsername.toLowerCase()}@example.com`);
    await page.fill('#usr-password', 'password123');
    await page.selectOption('#usr-role', '2');

    await page.getByRole('button', { name: 'Create User' }).click();

    await expect(page.getByRole('dialog', { name: 'Add New User' })).toBeHidden({ timeout: 10000 });
    await expect(page.getByText(inputUsername.toLowerCase())).toBeVisible({ timeout: 10000 });
  });

  test('should reject username with invalid characters', async ({ page }) => {
    await page.getByRole('button', { name: 'Add User' }).click();
    await expect(page.getByRole('dialog', { name: 'Add New User' })).toBeVisible();

    await page.fill('#usr-username', 'test user!');
    await page.fill('#usr-email', 'testuser@example.com');
    await page.fill('#usr-password', 'password123');

    const createButton = page.getByRole('button', { name: 'Create User' });
    await expect(createButton).toBeDisabled();
  });

  test('should show error when submitting empty username', async ({ page }) => {
    await page.getByRole('button', { name: 'Add User' }).click();
    await expect(page.getByRole('dialog', { name: 'Add New User' })).toBeVisible();

    await page.fill('#usr-email', 'testuser@example.com');
    await page.fill('#usr-password', 'password123');

    await page.getByRole('button', { name: 'Create User' }).click();

    await expect(page.getByRole('dialog', { name: 'Add New User' })).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Please fill all required fields').or(page.getByText('required'))).toBeVisible({ timeout: 10000 });
  });
});
