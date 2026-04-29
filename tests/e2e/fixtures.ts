import { test as base } from '@playwright/test';

// ============================================================================
// Global Fixtures & Helpers for E2E Tests
// ============================================================================

/**
 * Authenticate and return authenticated page context
 * Usage: const { page, token } = await authAs('superadmin', 'admin123');
 */
export const test = base.extend({
  // Add authenticated page fixture
  page: async ({ page }, use) => {
    // Store original page
    const originalPage = page;
    
    // Add helper methods to page
    page.authAs = async (username: string, password: string) => {
      await page.goto('/');
      await page.fill('#username', username);
      await page.fill('#password', password);
      await page.click('.login-btn');
      await expect(page.locator('#dashboard')).toBeVisible({ timeout: 5000 });
      
      // Return token
      return await page.evaluate(() => sessionStorage.getItem('access_token'));
    };

    page.logout = async () => {
      await page.evaluate(() => sessionStorage.clear());
      await page.reload();
      await expect(page.locator('#login-section')).toBeVisible();
    };

    page.getJwtPayload = async () => {
      return await page.evaluate(() => {
        const token = sessionStorage.getItem('access_token');
        if (!token) return null;
        const parts = token.split('.');
        return JSON.parse(atob(parts[1]));
      });
    };

    page.isLoggedIn = async () => {
      const token = await page.evaluate(() => sessionStorage.getItem('access_token'));
      return !!token;
    };

    await use(page);
  },
});

// ============================================================================
// Test Data
// ============================================================================

export const TEST_USERS = {
  superadmin: {
    username: 'superadmin',
    password: 'admin123',
    role: 'superadmin',
    id: 1
  },
  admin: {
    username: 'admin',
    password: 'admin123',
    role: 'admin',
    id: 2
  },
  manager: {
    username: 'manager',
    password: 'admin123',
    role: 'manager',
    id: 3
  },
  cashier: {
    username: 'cashier',
    password: 'admin123',
    role: 'cashier',
    id: 4
  }
};

export const API_ENDPOINTS = {
  LOGIN: '/api/login',
  STATS: '/api/stats',
  PRODUCTS: '/api/products',
  SALES: '/api/sales',
  INVENTORY_EXPORT: '/api/inventory/export',
  ADMIN_USERS: '/api/admin/users',
  ADMIN_ROLES: '/api/admin/roles'
};

// ============================================================================
// Utility Functions
// ============================================================================

/**
 * Wait for API to be ready
 */
export async function waitForAPI(page, maxAttempts = 30) {
  for (let i = 0; i < maxAttempts; i++) {
    try {
      const response = await page.request.get(API_ENDPOINTS.STATS);
      if (response.ok()) return true;
    } catch (e) {
      // ignore
    }
    await new Promise(r => setTimeout(r, 2000));
  }
  throw new Error('API did not become ready');
}

/**
 * Extract JWT payload from token
 */
export function decodeJWT(token: string) {
  const parts = token.split('.');
  return JSON.parse(atob(parts[1]));
}

/**
 * Create a valid auth header
 */
export function authHeader(token: string) {
  return { Authorization: `Bearer ${token}` };
}

// ============================================================================
// Types
// ============================================================================

export type User = typeof TEST_USERS[keyof typeof TEST_USERS];
