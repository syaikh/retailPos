import { test, expect } from './fixtures';
import { loginUI } from './fixtures';

test.describe('Error Boundaries', () => {
  test('should redirect to login when unauthenticated', async ({ page }) => {
    await page.goto('http://localhost:5173/admin/users');
    await page.waitForTimeout(1000);
    await expect(page.locator('#username')).toBeVisible({ timeout: 5000 });
    expect(page.url()).toContain('/login');
  });

  test('should show page content for unknown routes', async ({ page }) => {
    await loginUI(page, 'superadmin', 'admin123');

    await page.goto('http://localhost:5173/nonexistent-route');
    await page.waitForTimeout(1000);
    await expect(page).not.toHaveURL(/\/login/);
  });
});
