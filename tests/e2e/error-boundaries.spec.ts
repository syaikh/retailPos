import { test, expect } from '@playwright/test';

test.describe('Error Boundaries', () => {
  test('should redirect to login when unauthenticated', async ({ page }) => {
    await page.goto('http://localhost:5173/admin/users');
    await page.waitForTimeout(1000);
    await expect(page.locator('#username')).toBeVisible({ timeout: 5000 });
    expect(page.url()).toContain('/login');
  });

  test('should show page content for unknown routes', async ({ page }) => {
    await page.goto('http://localhost:5173/login');
    await page.fill('#username', 'superadmin');
    await page.fill('#password', 'admin123');
    await page.click('button[type="submit"]');
    await expect(page).toHaveURL(/\/$/, { timeout: 5000 });

    await page.goto('http://localhost:5173/nonexistent-route');
    await page.waitForTimeout(1000);
    await expect(page).not.toHaveURL(/\/login/);
  });
});
