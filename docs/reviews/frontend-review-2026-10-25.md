# Full Frontend Review — 2026-10-25

Scope: `/my-review frontend` (full frontend codebase, not a diff review).
Tracks: security, performance, business logic, deploy safety, duplication, dead code.
Recommendation: **NEEDS CHANGES** — 8 WARNING, 9 SUGGESTION.

## Summary

The frontend is a Svelte 5 / Vite single-page app (`web/src`, ~431 files) with a hand-rolled
router in `app/main.svelte`, a route→permission map in `app/config/permissions.ts`, a session
kernel (`shared/auth/refresh-kernel.ts` + `shared/utils/tab-coordination.ts`) around a
`sessionStorage` bearer token, and 22 feature modules talking to the Go API through
`shared/api/http-client.ts`. Authorization is consistently enforced server-side (route guards
are UI-only by design, and that is stated as such in the code), and the Jakarta-time /
completed-period invariants are correctly implemented. The material issues are in
state-consistency of the POS and import flows, error paths that render
valid-looking-but-wrong results, and a cross-tab refresh path that can strand or degrade
requests.

## Issues Found

| Severity | File:Line | Issue |
| -------- | --------- | ----- |
| WARNING | `web/src/modules/pos/components/PosPage.svelte:1014` | Failed customer change is swallowed; UI/receipt show the new customer while the server cart keeps the old one and can end up mixing two pricing groups |
| WARNING | `web/src/modules/purchase-orders/components/GoodsReceiptModal.svelte:247` | Submit disabled when total good qty is 0, making an all-damaged receipt impossible to record |
| WARNING | `web/src/shared/ui/ImportWizard.svelte:155` | One transient progress-poll failure permanently freezes the wizard; `onComplete()` never fires |
| WARNING | `web/src/modules/sales/components/FindTransaction.svelte:91` | Any lookup failure is rendered as "no results for `<query>`" |
| WARNING | `web/src/modules/admin/components/UsersPage.svelte:275` | Duplicate error-unwrapping passes an object to `toast.error`, rendering `[object Object]` |
| WARNING | `web/src/shared/auth/refresh-kernel.ts:43` | Follower tab never persists the leader's refreshed token → every later request does 401 → cross-tab refresh → retry |
| WARNING | `web/src/shared/utils/tab-coordination.ts:284` | Single-slot refresh resolver with no timeout; concurrent 401s or a dying leader strand requests forever |
| WARNING | `web/src/modules/stores/components/StoresPage.svelte:127` | N+1 readiness requests (up to 100) fired on every list refetch |
| SUGGESTION | `web/src/modules/product/components/ProductsPage.svelte:380` | ~25 catch sites surface Axios's generic message instead of the backend's error message |
| SUGGESTION | `web/src/modules/pos/index.ts:1` | Entire POS barrel (service + utils) is dead; live code re-implements it inline and already diverges |
| SUGGESTION | `web/src/modules/product/lib/product-utils.ts:43` | Dead validation/payload helpers superseded by divergent live implementations |
| SUGGESTION | `web/src/modules/settings/services/settings-service.ts:189` | Dead export helper concatenates the bearer token into a URL opened via `window.open` |
| SUGGESTION | `web/src/modules/dashboard/components/Home.svelte:46` | Second, divergent RBAC implementation; card gates disagree with `routePermissions` |
| SUGGESTION | `web/src/app/layouts/NotificationBell.svelte:46` | Module store subscriptions are never unsubscribed |
| SUGGESTION | `web/src/modules/admin/components/AuditLogsPage.svelte:266` | Self-writing `$effect` issues a duplicate (immediately aborted) request on filter change after paging |
| SUGGESTION | `web/src/shared/ui/Dropdown.svelte:106` | Un-throttled window `scroll` handler does a forced layout read + reactive write per event |
| SUGGESTION | `web/src/modules/pos/components/PosPage.svelte:489` | Clearing the cart issues serial per-item DELETE round trips while the POS is blocked |

## Detailed Findings

### 1. POS customer change is committed locally before the server confirms it — WARNING

- **File:** `web/src/modules/pos/components/PosPage.svelte:1014`
- **Confidence:** 90%
- **Problem:** `selectedCustomerId = id` is committed to local state before the server call, and
  the only handler for a rejected `updateCartCustomer` is `.catch(() => {})` at `:1022-1024`. On
  failure the server cart keeps the previous customer, but checkout sends only the cart id
  (`checkoutCart(activeCartId, …)` at `:463`), so the sale is recorded with the old
  `customer_id` while the checkout modal label (`:101-104`) and the printed receipt
  (`:659-661`) show the newly selected customer. The divergence also reaches pricing: items
  added afterwards pass the *new* group (`customer_group_id` at `:365`/`:382`), while the
  backend re-prices existing lines only inside the `UpdateCartCustomer` transaction that just
  failed (`internal/sale/cart_service.go:124-157`), so the cart ends up holding lines priced
  under two different customer groups in one sale.
- **Suggestion:** Handle the rejection like every other cart mutation in this file: toast the
  failure and revert `selectedCustomerId` to `cartSession?.customer_id ?? null` (or await the
  update and only then commit the local selection).

### 2. All-damaged goods receipt cannot be recorded — WARNING

- **File:** `web/src/modules/purchase-orders/components/GoodsReceiptModal.svelte:247`
- **Confidence:** 90%
- **Problem:** `disabled={saving || getTotalGood() === 0}` blocks submission, but the path it
  guards explicitly accepts damaged-only lines: `handleSubmit` filters with
  `qty_good > 0 || qty_damaged > 0` (`:79-85`), the inputs allow `qty_damaged > 0` with
  `qty_good = 0` (`:200-219`), and the backend accepts it (only
  `QtyGood+QtyDamaged > remaining` is checked, `internal/purchase/service.go:495-501`; only
  `len(grItems) == 0` is rejected at `:524`). A shipment that arrived entirely damaged cannot
  be recorded, and the `pleaseEnterReceivingQuantities` branch it appears to guard is
  unreachable for this case.
- **Suggestion:** Disable only when nothing has been entered:
  `disabled={saving || items.every((i) => i.qty_good === 0 && i.qty_damaged === 0)}`.

### 3. One transient progress-poll failure permanently freezes the import wizard — WARNING

- **File:** `web/src/shared/ui/ImportWizard.svelte:155`
- **Confidence:** 90%
- **Problem:** The polling interval's `catch` only clears `pollTimer`. `progress` and `step`
  are unchanged, and neither is read again, so the `$effect` (`:141-162`) never re-runs: a
  single transient `getProgress` failure stops polling permanently. The wizard stays frozen on
  the progress step (which has no footer buttons, `:283-286`), `onComplete()` is never called
  (`:136`), and the parent's data is never refreshed even though the import job continues
  server-side. The `error` state is untouched and is not rendered on that step.
- **Suggestion:** Retry instead of clearing the timer on first error, surface an error after N
  consecutive failures, and guarantee a path to `onComplete`.

### 4. Find Transaction shows failures as "no results" — WARNING

- **File:** `web/src/modules/sales/components/FindTransaction.svelte:91`
- **Confidence:** 90%
- **Problem:** `load()`'s `catch` silently resets `data = []` / `total = 0`, and `runSearch()`
  sets `hasSearched = true` unconditionally (`:107`). The template renders
  `data.length === 0 && hasSearched` as `noResultsFor "<query>"` (`:185-194`), so a network
  error, 500, 403 or rate limit is presented to the cashier as proof that the receipt does not
  exist — an incorrect business conclusion drawn from a failed lookup.
- **Suggestion:** Track an `error` state in the `catch` (using `getApiErrorMessage`), show it
  in the empty state, and only set `hasSearched` when the request succeeded.

### 5. Duplicate error-unwrapping renders `[object Object]` — WARNING

- **File:** `web/src/modules/admin/components/UsersPage.svelte:275`
- **Confidence:** 95%
- **Problem:** `error?.response?.data?.error || error?.message || labels.networkError` feeds
  `toast.error(...)` directly, but the canonical error body is an object —
  `shared.JSONError` emits `{"error":{"code","message"}}`
  (`internal/shared/response.go:37-44`), and the duplicate-username conflict hits exactly
  that shape (`internal/user/handler.go:355-357`). `Toast.svelte:39` interpolates
  `{t.message}`, so the user sees `[object Object]`. The shared `getApiErrorMessage`
  (`shared/utils/error-utils.ts:7-20`) already unwraps string-or-object. The same duplicated
  pattern exists at `web/src/modules/shifts/components/CashMovementModal.svelte:67`.
- **Suggestion:** Replace both inline expressions with `getApiErrorMessage(error, fallback)`.

### 6. Follower tab never persists the leader's refreshed token — WARNING

- **File:** `web/src/shared/auth/refresh-kernel.ts:43`
- **Confidence:** 85%
- **Problem:** In a follower tab, `const token = await requestRefresh(); if (token) return
  token;` returns the leader's new token without calling `setAccessToken()`. `sessionStorage`
  is per-tab, so the follower's stored token stays stale: that one retried request succeeds
  (`:187`/`:206`) but every subsequent request re-reads the stale token from `session.ts:3`,
  gets a 401, and repeats a full cross-tab refresh round trip — double API traffic and added
  latency on every call for the tab's remaining life. The leader side has the mirror gap:
  `performRefreshForFollowers` broadcasts the result but never persists it
  (`tab-coordination.ts:231-233`). This path is untested — the auth tests mock
  `isTabLeader: () => true`.
- **Suggestion:** Call `setAccessToken(token)` in the follower branch of `doRefresh()` before
  returning, and persist on the leader in `performRefreshForFollowers`.

### 7. Single-slot refresh resolver with no timeout can strand requests forever — WARNING

- **File:** `web/src/shared/utils/tab-coordination.ts:284`
- **Confidence:** 85%
- **Problem:** `refreshResolvers` is a single slot, not a list, and neither `requestRefresh()`
  nor `REFRESH_RESULT`/`REFRESH_FAILED` has a timeout. Two concurrent 401s in a follower both
  reach it because `refreshPromise` is still `null` while the first awaits the leader
  (`refresh-kernel.ts:41-57`, interceptor check at `:181`), so the second overwrites the slot
  and the first promise never settles. Likewise, if the leader closes after receiving
  `REFRESH_REQUEST` but before answering, nothing settles it: `startLeaderCheck`
  (`:194-207`) elects a new leader but never re-sends or rejects the pending request, so the
  awaiting API call hangs indefinitely and no logout ever happens.
- **Suggestion:** Keep `Array<{resolve,reject}>` and settle all waiters on
  `REFRESH_RESULT`/`REFRESH_FAILED`, plus reject them after a bounded timeout (falling through
  to the direct-refresh path already present in `doRefresh`) or on leader change.

### 8. N+1 readiness requests on every Stores list refetch — WARNING

- **File:** `web/src/modules/stores/components/StoresPage.svelte:127`
- **Confidence:** 85%
- **Problem:** `fetchStores()` fires `fetchReadiness(res.data.map(s => s.id))`, which does
  `Promise.allSettled(ids.map(id => getReadiness(id)))` (`:101-114`) — one uncached
  `GET /api/stores/{id}/readiness` per row, on every mount, page change, status filter and
  debounced search. `Pagination.svelte:115-122` allows `limit=100`, so a single page/search
  action issues up to 100 concurrent requests on the same origin, saturating the browser's
  per-origin connection budget ahead of the user's next navigation and repeatedly hitting a
  backend endpoint that computes blockers per store.
- **Suggestion:** Cache results per store id and only request rows whose verdict is
  unknown/stale (bulk endpoint is a backend change outside this review's scope).

### 9. Axios generic message shown instead of the backend error message — SUGGESTION

- **File:** `web/src/modules/product/components/ProductsPage.svelte:380`
- **Confidence:** 85%
- **Problem:** ~25 call sites catch a raw Axios rejection and surface `err.message`
  (`ProductsPage.svelte:380/416/440`, `PosPage.svelte:388/403/475/495`,
  `StoreOnboardingWizard.svelte:273`, `RolesPage.svelte:453`). Axios's `message` is its own
  generic text ("Request failed with status code 400"), never the backend `error.message`, so
  failed saves/deletes/checkout show transport jargon instead of the actual business error.
  Two competing error-parsing implementations have already drifted — `getApiErrorMessage` is
  used at 70 sites, this pattern at 25.
- **Suggestion:** Route these through `getApiErrorMessage(err, labels.<fallback>)`.

### 10. Dead POS barrel with live-divergent duplicates — SUGGESTION

- **File:** `web/src/modules/pos/index.ts:1`
- **Confidence:** 90%
- **Problem:** The POS barrel is imported by nothing (including tests), so
  `getPosProducts`/`getCustomers`/`searchCustomers`/`getSaleById`/`getLastSale`
  (`pos-service.ts:13-58`) and every `pos-utils` export (`pos-utils.ts:6-31`) are dead in
  production while `PosPage.svelte` re-implements each request inline (`:166-168`, `:194`,
  `:246-248`, `:613`, `:849-851`). The copies already contradict each other:
  `QUICK_CASH_PRESETS = [50000,100000,150000,200000]` vs the live denominations
  `[5000,10000,20000,50000,100000]` (`CheckoutModal.svelte:13`), and
  `QUICK_FILL_TOTAL_KEY = "F6"` while F6 is actually "hold sale" (`PosPage.svelte:745-750`).
- **Suggestion:** Delete the dead exports, barrel and their tests.

### 11. Superseded product validation/payload helpers kept as dead exports — SUGGESTION

- **File:** `web/src/modules/product/lib/product-utils.ts:43`
- **Confidence:** 90%
- **Problem:** `validateProductForm` (`:43`) and `buildProductPayload` (`:62`) are referenced
  only by their own tests plus the unused `product/index.ts:22-24` re-export. The live copies
  are `ProductFormModal.validate()` (`ProductFormModal.svelte:115-123`) and the inline payload
  in `ProductsPage.handleAdd`, and they already diverge: the live validation adds the
  `sku.length > 50` rule the exported one lacks.
- **Suggestion:** Delete both dead exports, their tests and the re-export.

### 12. Dead export helper embeds the bearer token in a URL — SUGGESTION

- **File:** `web/src/modules/settings/services/settings-service.ts:189`
- **Confidence:** 90%
- **Problem:** `downloadExport` builds `window.open(\`${url}&token=${token}\`)` with the raw
  access token from `sessionStorage` (`:183-185`), which would put the bearer token into
  browser history, proxy/backend access logs and any intermediary request log. It is currently
  reachable only from its own test — `exportBrands`/`importBrands` have no production caller,
  and `/api/brands/export` is not registered by `internal/brand/handler.go` — so it is dead
  code carrying a latent credential leak (the only `token=` URL in `web/src`).
- **Suggestion:** Delete `downloadExport`/`exportBrands`/`importBrands` (the real path is
  `shared/services/import-export-service.ts:85`, which uses the `Authorization` header).

### 13. Second, divergent RBAC implementation on the dashboard — SUGGESTION

- **File:** `web/src/modules/dashboard/components/Home.svelte:46`
- **Confidence:** 80%
- **Problem:** Home re-implements RBAC (`hasPermission`/`hasAnyPermission`, `:46-54`) with
  hardcoded permission strings instead of `useRBAC().can` + `Permissions.*`, creating a second
  copy of the authorization policy that can drift. It demonstrably has: the "Administration"
  card is shown for **any** of `["user.view","role.view","store.view"]` (`:131`, filtered at
  `:136`) while the `/admin` route requires `user.view` (`app/config/permissions.ts:23`), and
  the "Pending Settlements" card is gated on `consignment.pay` (`:140`) while `/consignment`
  requires `consignment.view` (`:39`) — in both cases a card renders that bounces with "no
  permission" when clicked. Impact is UI-only (the route guard and backend both re-check).
- **Suggestion:** Derive card visibility from the same `Permissions.*` arrays used by
  `routePermissions`.

### 14. NotificationBell store subscriptions are never unsubscribed — SUGGESTION

- **File:** `web/src/app/layouts/NotificationBell.svelte:46`
- **Confidence:** 85%
- **Problem:** `notifications.subscribe(...)` and `notifications.unreadCount.subscribe(...)`
  run at component init and the returned unsubscribe functions are discarded.
  `NotificationBell` lives inside `Layout`, which `main.svelte:336-340` unmounts on every
  logout/login and on the `mustChangePassword` branch (`:330-333`), so each auth cycle leaks
  two subscribers plus the retained Topbar subtree, and each leaked subscriber keeps running
  on every notification push.
- **Suggestion:** Move the subscriptions into `onMount`/`$effect` that returns the
  unsubscribe functions.

### 15. Duplicate (aborted) request on filter change after paging — SUGGESTION

- **File:** `web/src/modules/admin/components/AuditLogsPage.svelte:266`
- **Confidence:** 85%
- **Problem:** The filter branch (`:256-260`) writes `offset = 0` and then assigns
  `prevOff = so` (the old offset, `:266`). Because the effect reads `offset`, it re-runs, sees
  `pageChanged` (`0 !== 20`) and issues a second `fetchLogs()` for the identical query,
  immediately aborting the first (`:139`) — a wasted request plus a `loading` flicker on every
  filter change made while the user has paged away from page 0.
- **Suggestion:** Set `prevOff = 0` alongside `offset = 0`, as `UsersPage.svelte:173-174`
  already does.

### 16. Un-throttled scroll reposition does layout thrash per event — SUGGESTION

- **File:** `web/src/shared/ui/Dropdown.svelte:106`
- **Confidence:** 80%
- **Problem:** While open, a window-level capture `scroll` listener calls `reposition()` →
  `computePosition()` → `container.getBoundingClientRect()` (`:49`) and assigns the `$state`
  `menuStyle` on **every** scroll event with no rAF coalescing — a forced layout read plus a
  reactive write per frame, per open menu, on the busiest list pages (dropdowns are
  instantiated per table row). The same pattern exists at `SelectSearch.svelte:106` and
  `ProductFormModal.svelte:148/250`.
- **Suggestion:** Set a pending flag in the handler and run `computePosition()` inside a
  single `requestAnimationFrame`, cancelling it in the effect cleanup.

### 17. Serial per-item DELETE when clearing the cart — SUGGESTION

- **File:** `web/src/modules/pos/components/PosPage.svelte:489`
- **Confidence:** 80%
- **Problem:** `clearCart()` does `for (const item of cartItems) { lastSession = await
  removeCartItem(...) }` — one HTTP round trip and one backend transaction per line,
  serialized, with the POS blocked behind `cartLoading = true` for the whole sequence. A 30–50
  line wholesale cart on a 50–100 ms RTT link is 1.5–5 s of frozen checkout screen triggered
  by a single Alt+Delete.
- **Suggestion:** Issue the deletes concurrently and fetch the resulting session once at the
  end (a bulk backend endpoint would be a separate backend change).

## Recommendation

**NEEDS CHANGES** — the POS customer/pricing desync, the blocked all-damaged goods receipt,
the frozen import wizard, and the error paths that present failures as valid results are
material business-logic defects; the cross-tab refresh gaps degrade or strand requests in
production.

## Remediation

All 17 findings were fixed on 2026-10-25 in the same working tree as this review
(no commits made). Notes on two fixes where the obvious remedy was wrong:

- **Finding 3 (ImportWizard)** — stopping the poll and jumping to the summary step was
  rejected: `ImportSummary` renders any status other than `completed`/`failed` as a green
  "Cancelled" checkmark, a false terminal state. The wizard instead keeps polling with
  `setTimeout` (1 s cadence, 5 s backoff after 5 consecutive failures), surfaces the error
  in a banner shared by every step, and still reaches `onComplete()` when the job ends.
- **Finding 15 (AuditLogsPage)** — writing `prevOff = 0` in the filter branch alone is not
  enough: the effect's bottom sync (`prevOff = so`) restored the stale offset. The local
  `so` is re-pointed to the reset offset so the re-run triggered by writing `offset` is a
  no-op instead of a duplicate, aborted `fetchLogs`.

