import { test as base, expect } from '@playwright/test';

// ============================================================================
// Configuration - Base URLs from environment variables
// ============================================================================

export const FRONTEND_BASE = process.env.FRONTEND_BASE_URL || 'http://localhost:5173';
export const API_BASE = process.env.API_BASE_URL || 'http://localhost:9095';

// Export full URLs for common endpoints
export const API_URLS = {
  LOGIN: `${API_BASE}/api/login`,
  HEALTH: `${API_BASE}/health`,
  STATS: `${API_BASE}/api/dashboard/stats`,
  PRODUCTS: `${API_BASE}/api/products`,
  SALES: `${API_BASE}/api/sales`,
  INVENTORY_EXPORT: `${API_BASE}/api/inventory/export`,
  ADMIN_USERS: `${API_BASE}/api/admin/users`,
  ADMIN_ROLES: `${API_BASE}/api/admin/roles`
};

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
    // Add helper methods to page
    page.authAs = async (username: string, password: string) => {
      await page.goto('/');
      await page.fill('#username', username);
      await page.fill('#password', password);
      await page.click('button[type="submit"]');
      await expect(page).toHaveURL(/\/$/, { timeout: 5000 });
      
      // Return token
      return await page.evaluate(() => sessionStorage.getItem('access_token'));
    };

    page.logout = async () => {
      await page.evaluate(() => sessionStorage.clear());
      await page.reload();
      await expect(page.locator('#username')).toBeVisible();
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
  HEALTH: '/health',
  STATS: '/api/dashboard/stats',
  PRODUCTS: '/api/products',
  SALES: '/api/sales',
  INVENTORY_EXPORT: '/api/inventory/export',
  ADMIN_USERS: '/api/admin/users',
  ADMIN_ROLES: '/api/admin/roles'
};

// ============================================================================
// Token Cache - Reuse tokens across tests to avoid login rate limiting
// ============================================================================

const tokenCache = new Map<string, { token: string; expiresAt: number }>();
const TOKEN_TTL_MS = 55 * 60 * 1000; // 55 minutes (tokens typically last 1 hour)

/**
 * Login via API and cache the access token to avoid hitting rate limits.
 * Uses request fixture from Playwright for HTTP calls.
 */
export async function getToken(request: any, username: string = TEST_USERS.superadmin.username, password: string = TEST_USERS.superadmin.password): Promise<string> {
  const cacheKey = `${username}:${password}`;
  const cached = tokenCache.get(cacheKey);
  if (cached && cached.expiresAt > Date.now()) {
    return cached.token;
  }

  const res = await request.post(`${API_BASE}/api/login`, {
    data: { username, password },
  });
  expect(res.ok(), `login failed for ${username}: ${res.status()}`).toBeTruthy();
  const body = await res.json();
  const token = body.access_token;
  tokenCache.set(cacheKey, { token, expiresAt: Date.now() + TOKEN_TTL_MS });
  return token;
}

/**
 * Wait for API to be ready
 */
export async function waitForAPI(page, maxAttempts = 30) {
  for (let i = 0; i < maxAttempts; i++) {
    try {
      const response = await page.request.get(`${API_BASE}/health`);
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

/**
 * Login via browser UI with retry for transient failures.
 * Clears session, fills form, submits, and waits for navigation away from login.
 */
export async function loginUI(page: any, username: string, password: string, retries = 2) {
  for (let attempt = 0; attempt <= retries; attempt++) {
    await page.goto(`${FRONTEND_BASE}/login`);
    await page.waitForTimeout(500);
    await page.evaluate(() => sessionStorage.clear());
    await page.reload();
    await page.waitForSelector('#username', { state: 'visible', timeout: 15000 });
    await page.fill('#username', username);
    await page.fill('#password', password);
    await page.click('button[type="submit"]');
    await page.waitForTimeout(1500);
    try {
      await page.waitForFunction(() => {
        const path = window.location.hash || window.location.pathname;
        return path === '/' || path === '' || !path.includes('login');
      }, { timeout: 10000 });
      return;
    } catch {
      if (attempt < retries) {
        await new Promise(r => setTimeout(r, 2000));
        continue;
      }
      throw new Error(`loginUI failed after ${retries + 1} attempts for user "${username}"`);
    }
  }
}

/**
 * Logout via browser UI - clears session and navigates to login.
 */
export async function logoutUI(page: any) {
  await page.evaluate(() => sessionStorage.clear());
  await page.goto(`${FRONTEND_BASE}/login`);
  await page.waitForSelector('#username', { state: 'visible', timeout: 10000 });
}

// ============================================================================
// Types
// ============================================================================

export type User = typeof TEST_USERS[keyof typeof TEST_USERS];
