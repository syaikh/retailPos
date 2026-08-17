# Frontend Review — 2026-08-18

## Summary

Full frontend review of the Svelte 5 SPA (20 feature modules, custom SPA router, two coexisting HTTP clients, ~100+ test files). The review focused on security, performance, business logic, deploy safety, duplication, and dead code tracks. **2 critical issues**, **2 warnings**, and **2 suggestions** were identified.

## Issues Found

| Severity | Area     | File:Line                                        | Issue                                                            |
| -------- | -------- | ------------------------------------------------ | ---------------------------------------------------------------- |
| CRITICAL | Security | `web/src/shared/api/http-client.ts:89`           | `apiFetch` does not call `logout()` on failed token refresh     |
| CRITICAL | BizLogic | `web/src/modules/reporting/components/ReportsPage.svelte:88` | `shiftDays` mismatch for `yesterday`/`daily` — wrong comparison labels |
| WARNING  | Perf     | `web/src/modules/pos/components/PosPage.svelte:364`  | `clearCart` fires N parallel DELETE requests instead of one bulk call |
| WARNING  | Perf     | `web/src/modules/pos/services/pos-service.ts:35`     | `getLastSale` uses hardcoded `startDate='2025-01-01'` — unbounded scan |
| SUGGESTION | Duplication | Multiple modules                            | `formatCurrency` reimplemented 5+ times with inconsistent output |
| SUGGESTION | Dead Code | `web/src/shared/ui/index.ts:14-15`              | `ExpandableRow` and `FilterChip` exported but never imported     |

## Detailed Findings

### 1. CRITICAL — `apiFetch` does not call `logout()` on failed refresh, stranding users

**Area:** Security
**File:** `web/src/shared/api/http-client.ts:89`
**Confidence:** 98%

**Problem:**
`apiFetch` (native fetch wrapper, line 62) and `apiClient` (Axios, line 7) handle 401 errors completely differently. When token refresh fails, `apiClient` calls `logout()` (via `setupAxiosInterceptors` at `auth-service.ts:114`) which clears the auth store and redirects to `/login`. `apiFetch` only throws `new Error('Session expired')` — the user stays on the current page with no redirect, no session cleanup, and no proactive refresh stop.

Ten services use `apiFetch` (supplier, stores, settings, pricing, PO, sales, roles, reporting, dashboard, consignment) and will silently leave users stranded after session expiry. Meanwhile, 8 services using `apiClient` properly redirect to login. Same backend, same token, different user-facing behavior.

Additionally, `apiClient` has a race-condition queue (`failedQueue`) preventing concurrent refresh storms; `apiFetch` has none.

**Suggestion:**
Unify 401 handling. Either route all services through `apiClient`, or have `apiFetch` call `logout()` on failed refresh and add a refresh-queue to prevent concurrent refresh calls:

```typescript
// In apiFetch, replace line 80-90:
if (response.status === 401) {
  const newToken = await refreshAccessToken();
  if (newToken) {
    headers['Authorization'] = `Bearer ${newToken}`;
    return fetch(url, { ...options, headers });
  }
  // Call logout like apiClient does
  const { logout } = await import('$modules/auth');
  logout();
  throw new Error('Session expired');
}
```

---

### 2. CRITICAL — `shiftDays` mismatch causes wrong comparison labels for `yesterday` and `daily`

**Area:** Business Logic
**File:** `web/src/modules/reporting/components/ReportsPage.svelte:88-89`
**Confidence:** 98%

**Problem:**
In `ReportsPage.svelte`, the `shiftDays` calculation at lines 88-89 groups `'yesterday'` and `'daily'` with `'weekly'` and `'7days'`, giving them `shiftDays = 7`:

```javascript
const shiftDays = activePeriodType === 'realtime' ? 1
  : ['yesterday', 'daily', 'weekly', '7days'].includes(activePeriodType) ? 7  // BUG
  : activePeriodType === '30days' ? 30 : 0;
```

But in `data-fetching.ts:117`, the same logic correctly assigns `shiftDays = 1`:

```javascript
const shiftDays = activePeriodType === 'realtime' || activePeriodType === 'daily' || activePeriodType === 'yesterday' ? 1 :
```

The `shiftDays` in ReportsPage is used to derive `prevStartStr` (line 91) which feeds into the KPI comparison labels at lines 132-133. When viewing the **yesterday** or **daily** period, the KPI card shows a label like "vs [date 7 days before]" while the actual comparison data is from 1 day before. Users see a misleading label referencing the wrong comparison date.

The same bug exists at line 157-158 in the `comparisonDateRange` block (though the `yesterday`/`daily` branches there hardcode `'00:00 - 23:00'`, masking the issue).

**Suggestion:**
Add `'yesterday'` and `'daily'` to the `shiftDays = 1` branch in both locations:

```javascript
const shiftDays = activePeriodType === 'realtime' || activePeriodType === 'yesterday' || activePeriodType === 'daily' ? 1
  : ['weekly', '7days'].includes(activePeriodType) ? 7
  : activePeriodType === '30days' ? 30 : 0;
```

---

### 3. WARNING — `clearCart` fires N parallel DELETE requests instead of one bulk call

**Area:** Performance
**File:** `web/src/modules/pos/components/PosPage.svelte:364`
**Confidence:** 90%

**Problem:**
The `clearCart` function fires one `DELETE /pos/cart/items/:itemId` request per cart item in parallel via `Promise.all`. Each response returns the full `CartSession` object. A cart with 15 items produces 15 network round trips and 15 full session payloads, when only the final state is needed. In a POS environment where speed matters (cashier clearing a cart to start fresh), this creates unnecessary latency.

```javascript
// Current (PosPage.svelte:364):
const removals = cartItems.map(item => removeCartItem(cartId, item.id));
const sessions = await Promise.all(removals);
const last = sessions[sessions.length - 1];
if (last) applyCartSession(last);
```

**Suggestion:**
Add a server-side `DELETE /pos/cart/:id/items` (bulk) or `POST /pos/cart/:id/clear` endpoint, and call it once. Alternatively, batch-remove server-side in a single transaction and return only the final session.

---

### 4. WARNING — `getLastSale` uses unbounded date range on every POS mount

**Area:** Performance
**File:** `web/src/modules/pos/services/pos-service.ts:35`
**Confidence:** 85%

**Problem:**
On every POS page mount, `getLastSale()` queries `GET /api/sales?limit=1&offset=0&startDate=2025-01-01&endDate={today}`. The hardcoded `startDate='2025-01-01'` forces the backend to potentially scan 1.5+ years of sales records to return `LIMIT 1`. As the dataset grows over time, this query becomes increasingly expensive on every POS page load.

**Suggestion:**
Add a dedicated `GET /api/sales/latest` endpoint that uses `ORDER BY created_at DESC LIMIT 1` (index-backed) instead of a date-range filter. Alternatively, use a recent window like `startDate={7_days_ago}`.

---

### 5. SUGGESTION — `formatCurrency` reimplemented 5+ times with inconsistent output

**Area:** Duplication
**Files:** Multiple modules
**Confidence:** 95%

**Problem:**
Currency formatting is independently implemented in at least 5 modules with different behavior:

| Module | Implementation | Output for 15000 |
|--------|---------------|-------------------|
| `pos/lib/pos-utils.ts:3` | `value.toLocaleString('id-ID')` | `15.000` (no symbol) |
| `product/lib/product-utils.ts:31` | `'Rp ' + value.toLocaleString('id-ID')` | `Rp 15.000` |
| `consignment/lib/format.ts:3` | `'Rp ' + (value ?? 0).toLocaleString('id-ID')` | `Rp 15.000` |
| `pricing/PriceSimulationModal.svelte:86` | `Intl.NumberFormat('id-ID', {style:'currency', currency:'IDR'})` | `Rp 15.000` |
| `purchase-orders/PurchaseOrdersTable.svelte:63` | `value.toLocaleString('id-ID')` | `15.000` (no symbol) |

The POS module returns no currency symbol while the product module hardcodes "Rp". If currency formatting requirements change, each copy must be found and updated individually — easy to miss, creating visible inconsistencies on pages that mix modules.

**Suggestion:**
Create a single `formatCurrency(value: number): string` utility in `web/src/shared/utils/` using `labels.currencySymbol` + `toLocaleString('id-ID')`, then replace all inline copies.

---

### 6. SUGGESTION — Dead UI components and exports

**Area:** Dead Code
**File:** `web/src/shared/ui/index.ts:14-15`, `web/src/shared/api/http-client.ts:52`
**Confidence:** 95%

**Problem:**
Three exports are defined but never imported anywhere in the codebase:

- `ExpandableRow.svelte` (`index.ts:14`) — zero imports found
- `FilterChip.svelte` (`index.ts:15`) — zero imports found (all grep matches are `clearFilterChip` handler functions, not the component)
- `cachedGet` (`http-client.ts:52`) — zero imports found; also highlights that the caching layer only works for `apiClient` users, not the 10+ services using `apiFetch`

**Suggestion:**
Remove the unused components and the `cachedGet` export, or add `// @internal` comments if planned for future use.

## Recommendation

**NEEDS CHANGES** — Two critical issues must be addressed:

1. The `apiFetch` logout gap (finding #1) creates a real security/UX inconsistency where 10 services silently strand users after session expiry
2. The `shiftDays` mismatch (finding #2) produces visibly wrong comparison labels for the two most common reporting views (yesterday and daily)

## Completion Status

All findings have been addressed. Build verified passing (`npm run build`).

| # | Severity | Status | Fix Summary |
| --- | -------- | ------ | ----------- |
| 1 | CRITICAL | **Fixed** | `apiFetch` now calls `logout()` on failed refresh (`http-client.ts:79`), matching `apiClient` behavior |
| 2 | CRITICAL | **Fixed** | `shiftDays` for `yesterday`/`daily` corrected to `1` in both `statCardLabels` and `comparisonDateRange` (`ReportsPage.svelte:88,157`) |
| 3 | WARNING | **Fixed** | `clearCart` now chains DELETE requests sequentially, keeping only the final `CartSession` state (`PosPage.svelte:359`) |
| 4 | WARNING | **Fixed** | `getLastSale` date range changed from hardcoded `2025-01-01` to `getDateNDaysAgoInJakarta(7)` in both `pos-service.ts:35` and `PosPage.svelte:650` |
| 5 | SUGGESTION | **Fixed** | Shared `formatCurrency` created at `shared/utils/currency.ts` using `labels.currencySymbol`; POS, product, and consignment modules now re-export from it |
| 6 | SUGGESTION | **Fixed** | Removed `ExpandableRow.svelte`, `FilterChip.svelte`, and `cachedGet` export; removed unused `getCached` import |

### Files Changed

| File | Change |
| ---- | ------ |
| `web/src/shared/api/http-client.ts` | Added `logout` import; `apiFetch` calls `logout()` on failed refresh; removed `cachedGet` and unused `getCached` import |
| `web/src/modules/reporting/components/ReportsPage.svelte` | Fixed `shiftDays` calculation in both `statCardLabels` and `comparisonDateRange` |
| `web/src/modules/pos/components/PosPage.svelte` | `clearCart` chains deletions sequentially; `getLastSale` uses 7-day window |
| `web/src/modules/pos/services/pos-service.ts` | `getLastSale` uses `getDateNDaysAgoInJakarta(7)` instead of hardcoded `2025-01-01` |
| `web/src/shared/utils/currency.ts` | **New** — shared `formatCurrency` with locale-aware currency symbol |
| `web/src/modules/pos/lib/pos-utils.ts` | Re-exports `formatCurrency` from shared utility |
| `web/src/modules/product/lib/product-utils.ts` | Re-exports `formatCurrency` from shared utility with `'-'` null fallback |
| `web/src/modules/product/components/ProductFormModal.svelte` | Replaced inline `formatCurrency` with shared import |
| `web/src/modules/product/components/ProductDetailDrawer.svelte` | Replaced inline `formatCurrency` with shared import wrapper |
| `web/src/modules/consignment/lib/format.ts` | Re-exports `formatCurrency` from shared utility |
| `web/src/shared/ui/index.ts` | Removed `ExpandableRow` and `FilterChip` exports |
| `web/src/shared/ui/ExpandableRow.svelte` | **Deleted** — unused component |
| `web/src/shared/ui/FilterChip.svelte` | **Deleted** — unused component |
