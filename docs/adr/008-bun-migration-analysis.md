# ADR-008: Node.js to Bun Migration — E2E Test Suite

**Date:** 2026-09-01
**Status:** Deferred
**Decision Makers:** Engineering Team

---

## Context

This document covers the full migration of the E2E test suite from Playwright (running on Node.js via `bunx`) to Bun's native `bun:test` runner + `Bun.WebView` browser automation (via Bunwright). The goal is to eliminate the Node.js dependency entirely for local dev tooling.

**Key insight:** Node.js is only used for local dev tooling. It never touches production. Docker images use `golang:alpine` (backend) and `nginx:alpine` (frontend static files). No Node.js runtime in any container.

---

## Current State

| Layer | Technology | Node.js Usage |
|-------|-----------|---------------|
| **Backend** | Go 1.26 | None |
| **Frontend** | Svelte 5 + Vite 6 + Tailwind CSS 4 | Dev server only |
| **Unit Tests** | Vitest 4.1.7 | Test runner only |
| **E2E Tests** | Playwright 1.59.1 | Test runner + `fs`/`os`/`path`/`child_process` in fixtures |
| **Production** | Nginx + Go | **Zero Node.js** |

### Files Using Node.js

```
retail-pos-system/
├── package.json                    # Root: Playwright E2E deps
├── package-lock.json               # Root lockfile
├── playwright.config.js            # E2E test config (Node/Playwright, CommonJS)
├── tests/e2e/                      # 60 E2E test files + 4 support files
│   ├── fixtures.ts                 # Token caching (fs, os, path), 344 lines
│   ├── api-driver.ts               # API test driver (pure TS), 95 lines
│   ├── pos-api.ts                  # POS API helpers (pure TS), 153 lines
│   ├── db-helper.ts                # DB access via child_process.execFileSync, 121 lines
│   └── *.spec.ts                   # 60 spec files
├── web/
│   ├── package.json                # Frontend deps (Svelte, Vite, Vitest)
│   ├── package-lock.json           # Frontend lockfile
│   ├── vite.config.js              # Vite config (imports node:url, dotenv)
│   ├── svelte.config.js            # Svelte preprocessor config
│   ├── tsconfig.json               # TypeScript config (types: ["node", "svelte"])
│   └── src/
│       ├── test-setup.ts           # Vitest setup (sessionStorage mock)
│       └── ...100+ unit test files
├── scripts/
│   ├── kill-port.sh
│   └── lint-rbac.sh
├── deploy/
│   └── docker-compose.yml          # NO Node.js
├── Makefile                        # References npm/npx
└── run-e2e.sh                      # References npx playwright
```

---

## Decision

**Replace the entire Playwright E2E test suite with `bun:test` + `Bun.WebView` (via Bunwright).**

### Rationale

| Factor | Analysis |
|--------|----------|
| Bunwright API coverage | ~95% of Playwright features used in this project are available |
| Gaps | `waitForResponse`, `page.route`, `websocket` frames — solvable via CDP or raw WS client |
| `.filter({ has: locator })` | 8 occurrences — refactor to `evaluate()` + DOM traversal |
| Test runner | `bun:test` provides `describe`, `test`, `expect`, `beforeEach/AfterEach` — sufficient |
| HTTP client | `Bun.request()` replaces Playwright's `APIRequestContext` |
| Assertion strategy | `page.evaluate()` polling — portable, reliable, auto-retrying |
| Migration strategy | Switch in batches, remove Playwright incrementally |

### Alternatives Considered

| Option | Effort | Why Not |
|--------|--------|---------|
| Bun as package manager only (keep Playwright) | 2-4 hours | Doesn't eliminate Node.js from E2E |
| Fork Playwright to add Bun.WebView backend | 2-4 weeks | Massive effort, Playwright works via `bunx`, Bun.WebView is experimental |
| **Bunwright + bun:test (chosen)** | 10-13 days | Best balance of effort, long-term simplicity, and Node.js elimination |

---

## Bunwright API Verification

### What Bunwright Provides (Verified from Source)

| Feature | Playwright | Bunwright | Notes |
|---------|-----------|-----------|-------|
| Navigation | `page.goto(url)` | `page.navigate(url)` | Syntax only |
| Click | `locator.click()` | `page.click("selector")` or `locator.click()` | Syntax only |
| Type/Fill | `locator.fill()` / `page.fill()` | `locator.fill()` / `locator.type()` | Syntax only |
| Keyboard | `page.keyboard.press('F4')` | `page.press("F4")` or `locator.press("F4")` | Syntax only |
| Locator | `page.locator("css")` | `page.locator("css:css")` or `$("css:css")` | Prefix syntax |
| getByRole | `page.getByRole('button', {name:'X'})` | `page.locator("role:button[name='X']")` | Prefix syntax |
| getByText | `page.getByText('X')` | `page.locator("text:X")` | Prefix syntax |
| getByLabel | `page.getByLabel('X')` | `page.locator("label:X")` | Prefix syntax |
| filter/first/last/nth | `.filter({hasText}).first()` | `.filter("text:X").first()` | `.filter({has})` NOT supported |
| evaluate | `page.evaluate(fn)` | `page.evaluate(fn)` | Direct |
| waitForURL | `expect(page).toHaveURL()` | `page.waitForURL("**/path")` | Different pattern |
| screenshot | `page.screenshot()` | `page.screenshot()` | Direct |
| Auto-wait | Built-in (5s) | Built-in (10s configurable) | Configurable via `retryTimeout` |
| Chaining | Not chainable | Chainable (lazy queue) | Bunwright-specific feature |
| CDP access | `page.context().newCDPSession()` | `page.cdp(method, params)` | Direct CDP access |
| **Network mocking** | `page.route()` | **MISSING** | Need CDP `Fetch.enable` |
| **Response interception** | `page.waitForResponse()` | **MISSING** | Need CDP `Network.enable` |
| **WebSocket frames** | `page.on('websocket')` | **MISSING** | Need raw WS client |
| **evaluateAll** | `page.evaluateAll()` | **MISSING** | Use `evaluate` with `querySelectorAll` |
| **addInitScript** | `page.addInitScript()` | **MISSING** | Navigate → evaluate → reload |

### Bunwright Selector Syntax Reference

| Prefix | Example | Matches |
|--------|---------|---------|
| `css:` | `css:button[type=submit]` | CSS selector |
| `role:` | `role:button[name='Login']` | ARIA role + name |
| `label:` | `label:Username` | Associated `<label>` text |
| `text:` | `text:Sign in` | Visible text content |
| `xpath:` | `xpath://button[1]` | XPath expression |
| _(unprefixed)_ | `button.submit` | Treated as CSS |

### Features NOT in Bunwright

| Missing Feature | Impact | Files Affected | Workaround |
|----------------|--------|---------------|------------|
| `test.extend()` custom fixtures | `page.authAs()`, `page.logout()`, `page.getJwtPayload()` | All 60 (via `fixtures.ts`) | Plain async helper functions |
| `expect(locator).toBeVisible()` etc. | Auto-retrying assertions | 34 files (454 occurrences) | `page.evaluate()` polling helpers |
| `expect(page).toHaveURL()` | URL assertion | 17 files | `page.waitForURL()` or polling |
| `request.get/post/put/delete` | HTTP client for API setup/teardown | 31 files | `Bun.request()` or `fetch()` |
| `page.route()` | Network mocking | 1 file | CDP `Fetch.enable` |
| `expect.poll()` | Polling assertions | 1 file | Custom polling helper |
| `page.on('websocket')` | WebSocket frame interception | 1 file (7 tests) | Raw `WebSocket` client |
| `page.addInitScript()` | Pre-page-load script injection | Core to `fixtures.ts` | Navigate → evaluate → reload |
| `locator.or()` | Locator composition | 1 file | Restructure selector logic |
| `locator.boundingBox()` | Geometry queries | 1 file | CDP `DOM.getBoxModel` |
| `allTextContents()` | Bulk text extraction | 1 file | `evaluate()` with `querySelectorAll` |
| Parallel workers, retries | Test isolation | Config | `bun test` sequential mode |

---

## Detailed Audit: Playwright Usage Across 60 Spec Files

### Locator Patterns (485 total `page.locator()` calls across 33 files)

| Pattern | Occurrences | Notes |
|---------|------------:|-------|
| `text=` selectors inside `page.locator()` | 155 | `page.locator('text=Your cart is empty')` |
| `.filter({ hasText: ... })` | 163 | **Most common chain** — narrow by visible text |
| `.filter({ has: ... })` | 8 | Uses locator as filter — NOT supported in Bunwright |
| `.filter({ hasNotText: ... })` | 6 | |
| `.filter({ hasNot: ... })` | 1 | |
| `.filter({ visible: true })` | 1 | |
| `.first()` | 112 | **Second most common** — disambiguate multiple matches |
| `.last()` | 7 | |
| `.nth(n)` | 5 | |
| `.locator()` (nested) | 8 | |
| `.getByText()` | 6 | |
| `.getByRole()` | 1 | |
| `.count()` | 4 | |
| `.textContent()` | 4 | |
| `.isVisible()` | 6 | |
| `.evaluate()` | 1 | |
| Tag selectors (`button`, `table`, etc.) | ~97 | |
| Tag + attribute (`button[role=...]`) | ~60 | |
| ID selectors (`#my-id`) | 41 | |
| Class selectors (`.my-class`) | 14 | |
| Attribute-only (`[role="dialog"]`) | ~30 | |
| Complex CSS chains (`table tbody tr`) | ~55 | |
| Multi-attribute | ~5 | |

### getByRole Breakdown (199 total across 22 files)

| Role | Occurrences |
|------|------------:|
| `dialog` | 101 |
| `button` | 55 |
| `columnheader` | 22 |
| `menuitem` | 17 |
| `heading` | 3 |
| `menu` | 1 |

### Other Locator Types

| Pattern | Files | Occurrences |
|---------|------:|------------:|
| `page.getByText()` | 11 | 46 |
| `page.getByPlaceholder()` | 7 | 31 |
| `page.getByLabel()` | 2 | 3 |
| `page.getByTestId()` | 0 | 0 |

### Page Actions

| Action | Files | Occurrences |
|--------|------:|------------:|
| `page.fill()` | 14 | 51 |
| `page.click()` | 5 | 15 |
| `page.keyboard.press()` | 9 | 29 |

#### Keyboard Keys Used

| Key | Count |
|-----|------:|
| `F4` | 11 |
| `F7` | 8 |
| `Escape` | 6 |
| `F6` | 2 |
| `F2` | 2 |

### Wait Patterns

| Pattern | Files | Occurrences |
|---------|------:|------------:|
| `page.waitForTimeout()` | 24 | 220 |
| `page.waitForSelector()` | 7 | 8 |
| `page.waitForFunction()` | 2 | 4 |
| `page.waitForLoadState()` | 1 | 4 |

### Assertion Patterns

| Pattern | Files | Occurrences |
|---------|------:|------------:|
| `expect(locator).toBeVisible()` | 34 | 454 |
| `expect(locator).toBeHidden()` | 13 | 64 |
| `expect(locator).toBeEnabled()` | 10 | 25 |
| `expect(locator).toBeDisabled()` | 4 | 5 |
| `expect(locator).toHaveText()` | 2 | 2 |
| `expect(locator).toContainText()` | 3 | 8 |
| `expect(locator).toHaveValue()` | 2 | 7 |
| `expect(locator).toBeFocused()` | 2 | 2 |
| `expect(locator).toHaveCount()` | 7 | 17 |
| `expect(locator).toHaveAttribute()` | 1 | 10 |
| `expect(page).toHaveURL()` | 17 | — |

### Special Features

| Feature | Files |
|---------|-------|
| `page.goto()` with query params | 2 (`products`, `transaction-notifications`) |
| `page.waitForResponse()` | 3 (`consignment-flow`, `hold-recall`, `purchase-orders-ui`) |
| `page.route()` (network mocking) | 1 (`transaction-notifications`) |
| `page.on('websocket')` | 1 (`websocket`) |
| `expect.poll()` | 1 (`websocket`) |
| `page.evaluateAll()` | 1 (`transaction-notifications`) |
| `page.on('console')` | 1 (`audit-logs-search`) |
| `page.on('request')` | 1 (`price-consistency`) |
| `page.on('response')` | 1 (`reports`) |

### Fixture Usage Per File

| Pattern | Files |
|---------|-------|
| Uses `{ page }` fixture | ~43 (UI tests) |
| Uses `{ request }` fixture | ~31 (API tests) |
| Uses both `{ page, request }` | ~12 (mixed tests) |
| Uses `test.describe.configure({ mode: 'serial' })` | 0 |
| Uses `test.setTimeout()` | 0 |
| Uses `test.slow()` | 0 |
| Uses `test.skip()` | 1 |
| Defines local helper functions | 30 of 60 |

### Category Breakdown

| Category | Files | Characteristics |
|----------|-------|----------------|
| **Pure API tests** | 17 | Zero `page.*` locator calls, HTTP-only |
| **Simple UI tests** | ~20 | `page.goto` + locators + assertions, no `waitForResponse` |
| **Complex UI tests** | ~10 | `waitForResponse`, `page.route`, complex locator chains |
| **Mixed API+UI** | ~13 | API setup + UI assertions in same file |

---

## Implementation Plan

### Phase 0: Build Infrastructure Layer (2-3 days)

**New files to create:**

```
tests/e2e/infra/
├── index.ts          — barrel export
├── browser.ts        — Bunwright browser lifecycle (singleton)
├── auth.ts           — token cache + authPage/logoutPage
├── http.ts           — Bun.request()-based API client
├── assertions.ts     — auto-retrying DOM assertions via evaluate()
├── selectors.ts      — Bunwright selector helpers with chaining
├── wait.ts           — waitForResponse via CDP, sleep, polling
├── constants.ts      — FRONTEND_BASE, API_BASE, TEST_USERS, API_URLS
└── types.ts          — shared TypeScript types
```

#### `browser.ts` — Bunwright singleton

```ts
import { browser as bunwright } from "bunwright";

let initialized = false;

export async function getBrowser() {
  if (!initialized) {
    initialized = true;
  }
  return bunwright;
}

export async function newPage() {
  const b = await getBrowser();
  return b.newPage();
}

export async function closeBrowser() {
  if (initialized) {
    await bunwright.close();
    initialized = false;
  }
}
```

#### `auth.ts` — Token cache (ported from fixtures.ts)

Port the existing token cache logic (file-based, TTL, dedup, retry) with these changes:

```ts
export async function authPage(page: any, username: string, password?: string) {
  const token = await getToken(username, password);
  await page.navigate(FRONTEND_BASE);
  await page.evaluate((t: string) => {
    sessionStorage.setItem('access_token', t);
    localStorage.setItem('pos.locale', 'en');
  }, token);
  await page.reload();
  await waitForAppReady(page);
}

export async function logoutPage(page: any) {
  await page.evaluate(() => sessionStorage.clear());
  await page.reload();
  await expectVisible(page, '#username');
}
```

#### `http.ts` — Bun.request()-based API client

```ts
export class HttpClient {
  constructor(private token: string) {}

  private headers() { return { Authorization: `Bearer ${this.token}` }; }

  async get(path: string): Promise<ApiResult> {
    const res = await Bun.request({
      method: "GET", url: API_BASE + path, headers: this.headers()
    });
    return this.resolve(res);
  }

  async post(path: string, data?: any): Promise<ApiResult> { ... }
  async put(path: string, data?: any): Promise<ApiResult> { ... }
  async patch(path: string, data?: any): Promise<ApiResult> { ... }
  async delete(path: string): Promise<ApiResult> { ... }

  async multipart(path: string, file: File, fieldName = "file"): Promise<ApiResult> {
    const form = new FormData();
    form.append(fieldName, file);
    const res = await Bun.request({
      method: "POST", url: API_BASE + path,
      headers: this.headers(), body: form
    });
    return this.resolve(res);
  }

  private async resolve(res: any): Promise<ApiResult> {
    const text = await res.text();
    let body: any;
    try { body = JSON.parse(text); } catch { body = text; }
    return {
      status: res.status, ok: res.ok, body,
      headers: Object.fromEntries(res.headers.entries()),
    };
  }
}
```

#### `assertions.ts` — Auto-retrying DOM assertions

```ts
export async function expectVisible(page: any, selector: string, timeout = 5000) {
  const start = Date.now();
  while (Date.now() - start < timeout) {
    const visible = await page.evaluate((s: string) => {
      const el = document.querySelector(s);
      return el && el.offsetParent !== null;
    }, selector);
    if (visible) return;
    await sleep(100);
  }
  throw new Error(`Element "${selector}" not visible after ${timeout}ms`);
}

export async function expectHidden(page: any, selector: string, timeout = 5000) { ... }
export async function expectText(page: any, selector: string, expected: string, timeout = 5000) { ... }
export async function expectURL(page: any, pattern: string | RegExp, timeout = 5000) { ... }
export async function expectEnabled(page: any, selector: string, timeout = 5000) { ... }
export async function expectCount(page: any, selector: string, count: number, timeout = 5000) { ... }
```

#### `selectors.ts` — Bunwright selector helpers

```ts
export function byRole(role: string, name?: string): string {
  return name ? `role:${role}[name='${name}']` : `role:${role}`;
}

export function byText(text: string): string { return `text:${text}`; }
export function byLabel(label: string): string { return `label:${label}`; }

// DOM reading helpers via evaluate()
export async function count(page: any, selector: string): Promise<number> {
  return page.evaluate((s: string) => document.querySelectorAll(s).length, selector);
}

export async function isVisible(page: any, selector: string): Promise<boolean> {
  return page.evaluate((s: string) => {
    const el = document.querySelector(s);
    return el ? el.offsetParent !== null : false;
  }, selector);
}

export async function textContent(page: any, selector: string): Promise<string | null> {
  return page.evaluate((s: string) => document.querySelector(s)?.textContent ?? null, selector);
}

export async function allTextContents(page: any, selector: string): Promise<string[]> {
  return page.evaluate((s: string) =>
    Array.from(document.querySelectorAll(s)).map(el => el.textContent ?? ''), selector);
}
```

#### `wait.ts` — CDP-based response interception + polling

```ts
export async function waitForResponse(
  page: any,
  predicate: (resp: { url: string; status: number; method: string }) => boolean,
  timeout = 10000
): Promise<{ url: string; status: number; method: string; body: () => Promise<any> }> {
  // Use CDP Network.enable + page.addEventListener("Network.responseReceived", ...)
  // Collect events, return first match
}

export async function waitForWSMessage(
  predicate: (msg: any) => boolean,
  token: string,
  wsUrl: string,
  timeout = 10000
): Promise<any> {
  return new Promise((resolve, reject) => {
    const ws = new WebSocket(`${wsUrl}?token=${token}`);
    const timer = setTimeout(() => { ws.close(); reject(new Error("WS timeout")); }, timeout);
    ws.onmessage = (event) => {
      const msg = JSON.parse(event.data);
      if (predicate(msg)) { clearTimeout(timer); ws.close(); resolve(msg); }
    };
  });
}

export function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}
```

---

### Phase 1: Migrate API-Only Tests (17 files, ~1 day)

**Files:** `admin-api`, `audit-logs-api`, `auth-api`, `categories-api`, `dashboard-api`, `entities-api`, `hold-recall-api`, `inventory-adjust-api`, `pricing-rules-api`, `products-api`, `purchase-orders-api`, `public-api`, `rbac-api`, `reports-api`, `roles-api`, `sales-api`, `simple-test`

**Conversion pattern:**

| Playwright | bun:test + infra |
|-----------|-----------------|
| `import { test, expect } from '@playwright/test'` | `import { describe, test, expect } from 'bun:test'` |
| `async ({ request })` fixture | Remove — use `HttpClient` directly |
| `apiAs(request, 'superadmin')` | `apiAs('superadmin')` |
| `expect(res.ok()).toBeTruthy()` | `expect(res.ok).toBe(true)` |
| `test.describe.serial` | `describe` (Bun runs in order) |

---

### Phase 2: Migrate Simple UI Tests (~20 files, ~3-4 days)

**Files:** `admin`, `brands`, `categories`, `customers`, `customer-groups`, `dashboard`, `dashboard-live`, `error-boundaries`, `login`, `pos-ui-edge` (UI sections), `print-receipt`, `products`, `reports`, `roles`, `shifts`, `sidebar`, `silent-print`, `stores`, `suppliers-ui`, `transactions`, `units-of-measure`

**Conversion pattern per file:**

1. Replace Playwright imports with `bun:test` + `../infra` imports
2. `test.beforeEach(async ({ page }) => { loginUI(page, ...) })` → `beforeEach(async () => { page = await newPage(); await authPage(page, ...) })`
3. `test.afterEach(async ({ page }) => { logoutUI(page) })` → `afterEach(async () => { await logoutPage(page); page.close(); })`
4. `page.goto('/path')` → `page.navigate(FRONTEND_BASE + '/path')`

**Locator conversion table:**

| Playwright | Bunwright |
|-----------|-----------|
| `page.getByRole('button', { name: 'Submit' })` | `page.locator("role:button[name='Submit']")` |
| `page.getByText('Hello')` | `page.locator("text:Hello")` |
| `page.getByPlaceholder('Search')` | `page.locator("label:Search")` |
| `page.locator('#my-id')` | `page.locator("css:#my-id")` |
| `page.locator('button')` | `page.locator("css:button")` |
| `page.locator('button').filter({ hasText: 'Export' })` | `page.locator("css:button").filter("text:Export")` |
| `page.locator('tr').first()` | `page.locator("css:tr").first()` |
| `page.locator('tr').nth(2)` | `page.locator("css:tr").nth(2)` |
| `page.locator('button').last()` | `page.locator("css:button").last()` |
| `page.locator('dialog', { hasText: 'Save' })` | `page.locator("role:dialog").filter("text:Save")` |

**Assertion conversion table:**

| Playwright | bun:test + infra |
|-----------|-----------------|
| `expect(locator).toBeVisible()` | `await expectVisible(page, selector)` |
| `expect(locator).toBeHidden()` | `await expectHidden(page, selector)` |
| `expect(locator).toHaveText('X')` | `await expectText(page, selector, 'X')` |
| `expect(page).toHaveURL(/regex/)` | `await expectURL(page, /regex/)` |
| `expect(locator).toBeEnabled()` | `await expectEnabled(page, selector)` |
| `expect(locator).toContainText('X')` | `await expectContainsText(page, selector, 'X')` |

**Other conversions:**

| Playwright | bun:test + infra |
|-----------|-----------------|
| `page.keyboard.press('F4')` | `page.press("F4")` |
| `page.fill('#input', 'value')` | `page.locator("css:#input").fill("value")` |
| `page.waitForTimeout(500)` | `await sleep(500)` |
| `page.waitForSelector('#el')` | `page.locator("css:#el").waitFor({ timeout: 5000 })` |

**Handling `.filter({ has: locator })` (8 occurrences):**
Replace with `page.evaluate()` + manual DOM traversal, or restructure the selector.

---

### Phase 3: Migrate Complex UI Tests (~10 files, ~3-4 days)

**Files:** `consignment-flow`, `hold-recall`, `pos-flow`, `pos-wholesale-flow`, `payment-validation`, `cart-session`, `purchase-orders-dropdown`, `purchase-orders-ui`, `transaction-notifications`, `websocket`

#### `waitForResponse` (3 files)

Before:
```ts
const [checkoutResponse] = await Promise.all([
  page.waitForResponse(resp =>
    resp.url().includes('/pos/cart/') && resp.url().includes('/checkout') && resp.request().method() === 'POST'
  ),
  page.keyboard.press('F4'),
]);
```

After:
```ts
await page.press("F4");
const resp = await waitForResponse(page, r =>
  r.url.includes('/pos/cart/') && r.url.includes('/checkout') && r.method === 'POST'
);
```

#### `page.route` (1 file: transaction-notifications)

Replace with CDP `Fetch.enable` or capture via `Network.requestWillBeSent`:

```ts
await page.cdp("Fetch.enable", { patterns: [{ urlPattern: "*api/sales/lookup*" }] });
```

#### WebSocket (1 file: websocket.spec.ts)

Replace `page.on('websocket')` with raw `WebSocket` client:

```ts
const token = await getToken("cashier");
const ws = new WebSocket(`ws://localhost:9095/ws?token=${token}`);
const messages: any[] = [];
ws.onmessage = (event) => messages.push(JSON.parse(event.data));
// ... test logic ...
ws.close();
```

#### `page.evaluateAll` (1 file: transaction-notifications)

Replace with `page.evaluate()` using `querySelectorAll`:

```ts
const rows = await page.evaluate(() =>
  Array.from(document.querySelectorAll('table tbody tr td:first-child'))
    .map(el => el.textContent)
);
```

---

### Phase 4: Update Build Tooling (~2 hours)

1. **Root `package.json`**: Remove `@playwright/test`, add `bunwright`, update scripts
2. **Delete `playwright.config.js`**
3. **Create `bunfig.toml`** with test configuration
4. **Update `Makefile`**: `npx playwright test` → `bun test`
5. **Update `run-e2e.sh`**: `npx playwright test` → `bun test`
6. **Delete `package-lock.json`** files, generate `bun.lockb`
7. **Update `.gitignore`** for `bun.lockb`

---

### Migration Effort Summary

| Phase | Files | Effort | Running in parallel with Playwright? |
|-------|-------|--------|--------------------------------------|
| Phase 0: Infrastructure | 8 new files | 2-3 days | Yes (new code, doesn't touch existing) |
| Phase 1: API-only tests | 17 files | 1 day | Yes |
| Phase 2: Simple UI tests | ~20 files | 3-4 days | Yes |
| Phase 3: Complex UI tests | ~10 files | 3-4 days | Yes |
| Phase 4: Build tooling | 5 files | 2 hours | Yes (last step) |
| **Total** | **60 specs + 8 infra + 5 config** | **10-13 days** | Delete Playwright only after all pass |

---

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|-----------|
| Bunwright `.filter({ has })` missing | Confirmed | 8 occurrences | Refactor to `evaluate()` with DOM traversal |
| `waitForResponse` via CDP unreliable | Medium | 3 files | Add polling fallback |
| `page.waitForTimeout` removal breaks timing | Low | 24 files | Replace with assertion-based waits where possible |
| Bunwright auto-wait too aggressive | Low | Various | Configure `retryTimeout` in `bunwright.config.ts` |
| `Bun.request()` multipart differences | Low | 2-3 files | Test early, use `FormData` directly |
| Concurrent `page` creation in tests | Low | All UI tests | Create/close in beforeEach/afterEach |
| Bunwright is small library (42 stars) | Medium | All | Pin version, vendor critical helpers, test thoroughly |
| `Bun.WebView` API experimental | Medium | All | Pin Bun version, test before upgrading |

---

## Testing Strategy During Migration

Run both Playwright and Bun tests in parallel during migration:

```bash
# Run existing Playwright tests (unchanged)
npx playwright test

# Run migrated Bun tests (new)
bun test tests/e2e/

# Run both
make test-e2e
```

Delete Playwright dependencies only after ALL tests are migrated and passing.

---

## References

- [Bun 1.4 Release Notes](https://bun.sh/blog/bun-v1.4)
- [Bun.WebView Documentation](https://bun.sh/docs/runtime/webview)
- [Bunwright GitHub](https://github.com/jonaspm/bunwright)
- [Bunwright API Reference](https://github.com/jonaspm/bunwright/blob/main/packages/app/README.md)
- [Playwright + Bun Discussion](https://github.com/microsoft/playwright/issues/27139)
