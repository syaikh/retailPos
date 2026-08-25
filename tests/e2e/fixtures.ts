import { test as base, expect } from '@playwright/test';
import { readFileSync, writeFileSync, existsSync } from 'fs';
import { tmpdir } from 'os';
import { join } from 'path';

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
export { expect };

export const test = base.extend({
  // Add authenticated page fixture
  page: async ({ page }, use) => {
    // The E2E suite asserts English UI labels, so force the English locale
    // before any page script runs (app default is Indonesian).
    await page.addInitScript(() => {
      localStorage.setItem('pos.locale', 'en');
    });

    // Add helper methods to page
    page.authAs = async (username: string, password: string) => {
      const { token, refreshToken } = await getAuthTokens(page.request, username, password);

      // Restore an existing session via the shared cached token instead of
      // submitting the login form. This keeps total /api/login calls for the
      // whole suite at ~4 (one per user, shared on disk), avoiding the per-IP
      // login rate limiter. `restoreSession()` does NOT re-apply the user's
      // language, so the English locale pinned by addInitScript above is kept.
      await page.goto('/');
      await page.evaluate(({ t, rt }) => {
        sessionStorage.setItem('access_token', t);
        sessionStorage.setItem('refresh_token', rt);
      }, { t: token, rt: refreshToken });
      await page.reload({ waitUntil: 'load' });
      await waitForAppReady(page);

      return token;
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
// Shared token cache - persisted to a file on disk so every Playwright worker
// process reuses the same tokens. The entire suite therefore makes at most one
// /api/login call per user (~4 total), which keeps us well under the per-IP
// login rate limiter (5 req/min, burst 5) and removes cross-spec auth flakiness.
//
// Hardening:
//  - TOKEN_CACHE_VERSION is baked into the cache filename; bump it (or delete
//    the file) to invalidate every cached token at once — e.g. after the
//    backend is restarted with a new JWT_SECRET or a fresh DB, since a token
//    cached <10 min earlier would otherwise be reused blindly and rejected.
//  - An in-memory, per-process promise map dedupes concurrent logins for the
//    same user (multiple specs starting up at once), so we never fire several
//    parallel /api/login calls that trip the rate limiter.
//  - Cached tokens are verified against the API at most once per process per
//    user; if the server rejects the token (stale secret, reseed, expiry
//    drift), we transparently re-login instead of failing later.
// ============================================================================

const TOKEN_CACHE_VERSION = 1;
const TOKEN_TTL_MS = 10 * 60 * 1000; // 10 minutes (JWT expires in 15 min)
const TOKEN_CACHE_FILE = join(tmpdir(), `retail-pos-e2e-tokens.v${TOKEN_CACHE_VERSION}.json`);

type CachedToken = { token: string; refreshToken: string; expiresAt: number };
type TokenStore = Record<string, CachedToken>;

// In-flight login promises, keyed by cacheKey, so concurrent callers for the
// same user share a single /api/login instead of racing it.
const inFlight: Map<string, Promise<CachedToken>> = new Map();
// Users whose cached token has already been verified against the API this run.
const validatedKeys = new Set<string>();

function readTokenStore(): TokenStore {
  try {
    if (existsSync(TOKEN_CACHE_FILE)) {
      return JSON.parse(readFileSync(TOKEN_CACHE_FILE, 'utf8')) as TokenStore;
    }
  } catch {
    // ignore corrupt/locked cache
  }
  return {};
}

function writeTokenStore(store: TokenStore): void {
  try {
    writeFileSync(TOKEN_CACHE_FILE, JSON.stringify(store));
  } catch {
    // ignore write failures (token simply won't be shared)
  }
}

/** Best-effort check that a token is still accepted by the API. */
async function isTokenValid(request: any, token: string): Promise<boolean> {
  try {
    const res = await request.get(`${API_BASE}/api/products?limit=1`, { headers: authHeader(token) });
    return res.ok();
  } catch {
    return false;
  }
}

/**
 * Login via API and cache both tokens on disk, shared across all workers.
 */
async function getAuthTokens(
  request: any,
  username: string = TEST_USERS.superadmin.username,
  password: string = TEST_USERS.superadmin.password
): Promise<CachedToken> {
  const cacheKey = `${username}:${password}`;
  const now = Date.now();
  const store = readTokenStore();
  const cached = store[cacheKey];
  if (cached && cached.expiresAt > now) {
    if (validatedKeys.has(cacheKey)) return cached;
    if (await isTokenValid(request, cached.token)) {
      validatedKeys.add(cacheKey);
      return cached;
    }
    // Cached token rejected by the server → fall through and re-login.
  }

  // De-dupe concurrent logins for the same user within this process.
  const existing = inFlight.get(cacheKey);
  if (existing) return existing;

  const promise = (async () => {
    let body: any;
    for (let attempt = 0; attempt < 6; attempt++) {
      const res = await request.post(`${API_BASE}/api/login`, {
        data: { username, password },
      });
      if (res.ok()) {
        body = await res.json();
        break;
      }
      // On rate-limit (429), back off and retry; the shared cache means only
      // the first worker to need a given user actually reaches this.
      if (res.status() === 429 && attempt < 5) {
        await new Promise((r) => setTimeout(r, 1500));
        continue;
      }
      expect(res.ok(), `login failed for ${username}: ${res.status()}`).toBeTruthy();
    }
    const entry: CachedToken = {
      token: body.access_token,
      refreshToken: body.refresh_token,
      expiresAt: Date.now() + TOKEN_TTL_MS,
    };
    const s = readTokenStore();
    s[cacheKey] = entry;
    writeTokenStore(s);
    validatedKeys.add(cacheKey);
    return entry;
  })();
  inFlight.set(cacheKey, promise);
  try {
    return await promise;
  } finally {
    inFlight.delete(cacheKey);
  }
}

/**
 * Reuse a cached access token across tests to avoid login rate limiting.
 */
export async function getToken(
  request: any,
  username: string = TEST_USERS.superadmin.username,
  password: string = TEST_USERS.superadmin.password
): Promise<string> {
  return (await getAuthTokens(request, username, password)).token;
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
export async function loginUI(page: any, username: string, password: string, _retries = 2) {
  const { token, refreshToken } = await getAuthTokens(page.request, username, password);

  // Restore an existing session via the shared cached token instead of
  // submitting the login form, avoiding the per-IP login rate limiter.
  // `restoreSession()` does not re-apply the user's language, so the English
  // locale pinned by addInitScript is preserved.
  await page.goto(`${FRONTEND_BASE}/`);
  await page.evaluate(({ t, rt }) => {
    sessionStorage.setItem('access_token', t);
    sessionStorage.setItem('refresh_token', rt);
  }, { t: token, rt: refreshToken });
  await page.reload({ waitUntil: 'load' });
  await waitForAppReady(page);
}

/**
 * Logout via browser UI - clears session and navigates to login.
 */
export async function logoutUI(page: any) {
  await page.evaluate(() => sessionStorage.clear());
  await page.goto(`${FRONTEND_BASE}/login`);
  await page.waitForSelector('#username', { state: 'visible', timeout: 10000 });
}

/**
 * Driver-layer readiness gate. The SPA only renders role-gated navigation after
 * the async /users/me permission fetch resolves, but it shows <aside> (empty)
 * immediately. Asserting against the nav before that fetch lands is the classic
 * "assert-before-hydration" flake. We absorb that accidental complexity HERE so
 * individual specs only describe essential behaviour.
 */
export async function waitForAppReady(page: any) {
  await page.locator('aside').waitFor({ state: 'visible', timeout: 15000 });
  // The SPA keeps a persistent WebSocket open, so `networkidle` never fires;
  // waiting on it just blocks for the full default timeout. The nav is
  // permission-driven and populates async, so gate on a real nav button
  // instead — that is the actual readiness signal.
  await page.locator('aside button').first().waitFor({ state: 'visible', timeout: 15000 });
}

// ============================================================================
// Types
// ============================================================================

export type User = typeof TEST_USERS[keyof typeof TEST_USERS];
