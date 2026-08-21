# Test Specification: Cashier Scenario Coverage

Status: **draft** (Phase 1 of `.opencode/plans/cashier-scenario-coverage-plan.md`)
Scope: all testable behavior for the **Cashier** role — POS register, cart sessions, payments,
own transactions, own shifts. Complements `docs/guides/user-manual.md` §3–5 and Appendix A.

Conventions:
- Scenario IDs: `CS-<area><n>` (A=cart session, B=stock, C=pricing/customer, D=payment, E=shift, F=UI edge).
- "API" steps use `POST/GET/PATCH` against `/api/...` with a cashier Bearer token unless stated.
- Expected results marked **[D-fix]** depend on Phase-2 defect fixes (see §3); they assert fixed behavior.
- Code citations are `file:line` at time of writing.

---

## 1. Scope & Strategy

Cashier permission surface (`docs/guides/user-manual.md` Appendix A):
`sale.create`, `sale.view` (intended: own-only), `sale.park`, `shift.view/create`,
`customer.view`, `product.view`, `pricing.view` (read-only), `dashboard.view`,
stock-opname count/submit.

Strategy layers:
1. **UI (Playwright)** — cashier-visible flows and edge cases in `web/src/modules/pos`, `shifts`.
2. **API (Playwright request context)** — backend business rules that are hard/impossible to drive from UI
   (expiry backdate, tampered payloads, cross-cashier isolation, concurrency).
3. **Negative RBAC** — endpoints/pages a cashier must not reach.

Environment: dev server (backend :9095, frontend :5173), seeded DB (`seed-dev.sh`),
English locale forced via `localStorage.pos.locale='en'` (fixtures.ts).

---

## 2. Scenario Catalog

### Area A — Cart Session Lifecycle

| ID | Scenario | Steps | Expected | Citation |
|----|----------|-------|----------|----------|
| CS-A1 | Open cart persists across page reload | Add item → reload POS | Cart restored with item; no duplicate cart created | `PosPage.svelte:645-648`; `cart_repository.go:18-49` |
| CS-A2 | Hold sets expiry TTL | Hold sale → inspect cart row via API/DB | status=`held`, `expired_at = now+24h` (default TTL) | `cart_service.go:307-337`; config `CART_HOLD_TTL_HOURS` |
| CS-A3 | Expired held cart cannot be resumed | Backdate `expired_at` to past (SQL) → Recall | Resume rejected `ErrCartExpired`; row stays `held` forever (lazy expiry) | `cart_service.go:355-376,362-364` |
| CS-A4 | Held carts are owner-scoped (list) | Cashier B lists held carts while A holds one | B's list excludes A's cart | `cart_repository.go:103-108` |
| CS-A5 | Held carts are owner-scoped (resume) | Cashier B resumes A's held cart ID | 403 `ErrCartNotOwned` | `cart_service.go:32-37,370-372` |
| CS-A6 | No cap on held carts | Hold 3 sales sequentially | All listed; Recall badge shows count; disabled at 0 | `ParkedSalesModal.svelte:64-85`; `CartPanel.svelte:204-217` |
| CS-A7 | No cancel/delete for held carts in UI | Open Parked modal | Only Recall action exists (documented limitation) | `ParkedSalesModal.svelte:72-83` |
| CS-A8 | Shift close blocked by open cart | Keep cart open → Close Shift | Close rejected (open-cart gate) | `shift/repository.go:171-178` |
| CS-A9 | Held cart survives shift close; checkout after close fails | Hold cart → close shift → resume + checkout | Resume OK; checkout fails **[D-fix]** friendly error (was raw 500) | `shift/repository.go:171-178`; `total_updater.go:23-39`; D2 |
| CS-A10 | Concurrent first-cart race | Two parallel first-adds for same cashier | Exactly one open cart; loser gets clean error **[D-fix]** (was generic 500) | `000_squash.sql:1143`; `cart_repository.go:21-38`; doc-only verdict |
| CS-A11 | Double-checkout prevented | Two sequential checkouts of same cart ID | Second → 409 `ErrCartAlreadyCheckedOut` | `cart_service.go:407-416` |
| CS-A12 | Price-tamper guard | PATCH line subtotal ≠ unit_price × qty (direct API) → checkout | 409 `ErrPriceMismatch` | `cart_service.go:440-441`; `cart_handler.go:89-90` |
| CS-A13 | Duplicate lines aggregate at checkout | Add same product twice (same snapshot) → checkout | Single aggregated sale line; stock deducted once for total qty | `cart_service.go:627-661` |

### Area B — Stock Validation Timing

| ID | Scenario | Steps | Expected | Citation |
|----|----------|-------|----------|----------|
| CS-B1 | Stock NOT checked at add-to-cart | API add qty 999 of 5-stock product | Accepted into cart without error | `cart_service.go:146-158` |
| CS-B2 | Oversell blocked at checkout | Checkout cart exceeding stock | 409 "insufficient stock"; stock unchanged | `stock_deducer.go:71-79`; `cart_handler.go:82-83` |
| CS-B3 | Concurrent checkout of last unit | Two terminals checkout same last-unit product in parallel | Exactly one succeeds; stock never negative | `stock_deducer.go:46-73` (FOR UPDATE + conditional decrement) |
| CS-B4 | UI stock badge is cart-adjusted & live | Add item → observe badge; trigger `stock_update` WS event | Badge decrements in-cart qty; WS update mutates ceiling live | `PosPage.svelte:109-112,670-675` |
| CS-B5 | Qty input clamps to available stock | Type qty > stock in cart input | Clamped to [1, stock]; unchanged value restored | `CartPanel.svelte:135-145` |
| CS-B6 | Decimal quantity accepted (gap) | Type `1.5` in qty input | Currently passed unclamped to PATCH — document actual behavior; decide clamp fix | `CartPanel.svelte:141-143`; `PosPage.svelte:294-307` |
| CS-B7 | Qty ≤ 0 removes item | Decrement below 1 / type 0 | Item removed via DELETE | `PosPage.svelte:294-302` |
| CS-B8 | Clamp ceiling uses raw catalog stock | Product stock 10, add 4 to cart, type 8 in input | Input max shows 10 (raw), not 6 (adjusted) — document as known quirk | `PosPage.svelte:298,303` |

### Area C — Pricing / Customer-Group Semantics

| ID | Scenario | Steps | Expected | Citation |
|----|----------|-------|----------|----------|
| CS-C1 | Price frozen at add time | Add item → change product price → checkout | Sale uses snapshot price; new adds use new price | `cart_service.go:161-186,432-447`; covered partly by E2E-01..08 |
| CS-C2 | Frozen indicator shown | Item added before price change | "harga dibekukan"/"price frozen" note visible | `CartPanel.svelte:123-125` |
| CS-C3 | Group pricing is per-add-request | Add item passing VIP group id WITHOUT attaching customer | VIP price applied; cart customer stays walk-in | `internal/sale/cart.go:92`; `cart_service.go:80-111` |
| CS-C4 | Changing cart customer does not re-price | Attach customer after items added | Existing items keep prices; only future adds (if client passes group) differ | `UpdateCartCustomer` `cart_service.go:80-111` |
| CS-C5 | Direct POST /sales rejects client pricing fields | Send invoice_number/discount/tax/store_id/subtotal/total/unit_price | Each → 400 | `handler.go:241-280` |
| CS-C6 | Invoice generated server-side only | Successful checkout | Format `INV-<JakartaYear>-<seq %06d>`; failures don't burn sequence | `repository.go:605-616`; `handler.go:315-322` |
| CS-C7 | Tax-inclusive totals | Checkout taxable item | DPP = round(subtotal×100/111); tax = subtotal − DPP | `cart_service.go:684-692` |

### Area D — Payment Validation

| ID | Scenario | Steps | Expected | Citation |
|----|----------|-------|----------|----------|
| CS-D1 | Allocations must equal total exactly | Underpay / overpay | Done button disabled (UI); API 400 `ErrPaymentTotalMismatch` | `CheckoutModal.svelte:49-51,393`; `service.go:247-249` |
| CS-D2 | Max 10 payment rows | API send 11 allocations | 400 `ErrMaxPaymentsExceeded` | `service.go:190-192`; `domain.go:22` |
| CS-D3 | Single CASH entry only | API send two CASH rows | 400 | `service.go:222-226` |
| CS-D4 | Duplicate non-cash method rejected | Two QRIS rows | 400 (covered pos-flow.spec) | `service.go:227-231` |
| CS-D5 | Zero-amount rows allowed if sum=total | One row Rp 0 + one exact row | Completes (CurrencyInput forbids negatives) | `CurrencyInput.svelte:37-53` |
| CS-D6 | Reference prefill per method | Select CARD / E_WALLET / TRANSFER | Prefills `EDC/{ddmmyy}/{rand}` / `EW/…` / `REF/…`; editable | `CheckoutModal.svelte:71-96,329-342` |
| CS-D7 | requires_reference enforced | Method flagged, blank ref | 400 | `service.go:234-236` |
| CS-D8 | Fallback methods on API failure | Intercept `/payment-methods` → abort | Static Cash/Card/E-Wallet list used | `PosPage.svelte:171-187` |
| CS-D9 | Single toast on failed checkout | Force checkout failure (e.g. insufficient stock) | Exactly ONE error toast **[D-fix]** (was double toast) | D1; `PosPage.svelte:349-353,514-517` |
| CS-D10 | Empty payments rejected | API checkout with empty array | 400 `ErrZeroPaymentAmount` | `service.go:187-189` |

### Area E — Shift Rules

| ID | Scenario | Steps | Expected | Citation |
|----|----------|-------|----------|----------|
| CS-E1 | POS gated on open shift (cashier) | Login cashier without shift → go to /pos | Toast + redirect to /shifts; data loading aborted | `PosPage.svelte:630-638` |
| CS-E2 | One open shift per user | Open second shift same user | Friendly "already has an open shift" (pre-check + unique index) | `shift/repository.go:66-98`; `000_squash.sql:1145` |
| CS-E3 | Opening balance must be > 0 | Try open with 0 | Button disabled; API 400 | `ShiftsPage.svelte:88,428`; `shift/service.go:30-32` |
| CS-E4 | Closing balance ≥ 0; owner-scoped close | Close own shift; try closing another user's shift ID | Own OK; other → "not found or not open" | `shift/service.go:37-39`; `shift/repository.go:155-166` |
| CS-E5 | Discrepancy formula & needs_review boundary | Close with Δ = ±50 000 and ±50 001 | needs_review flips at strictly greater than 50 000 | `shift/repository.go:195-198` |
| CS-E6 | Logout blocked with open shift (frontend-only) | Cashier with open shift: sidebar logout disabled; direct API logout | UI blocks; backend logout SUCCEEDS — document gap | `Sidebar.svelte:63-64`; `auth_service.go:213-215` |
| CS-E7 | Topbar shift indicator | Open/close shift | "Shift: Open/Closed" updates (hardcoded EN string) | `Topbar.svelte:178-183` |
| CS-E8 | Sale into closed shift fails cleanly | Checkout after shift closed | **[D-fix]** friendly 4xx (was raw 500) | D2; `total_updater.go:23-39` |
| CS-E9 | shift_id ownership validated | API create cart/sale with another user's open shift_id | **[D-fix]** 409 generic error (fixed; no existence leak) | D3; `total_updater.go`; `handler.go:349`; `cart_handler.go:119` |
| CS-E10 | Cashier sees only own shifts | List shifts as cashier | Only own rows even when requesting other user_id filter | `shift/handler.go:50-56`; `ownership.go:29-45` |

### Area F — POS UI Edge Cases

| ID | Scenario | Steps | Expected | Citation |
|----|----------|-------|----------|----------|
| CS-F1 | F7 opens Held Sales modal without navigating | Press F7 on POS | Parked modal opens; page does NOT reload (F5 stays browser reload); while a modal is open F7 belongs to that modal (checkout = Exact fill) | `PosPage.svelte` keydown F7 branch w/ modal guard; `CartPanel.svelte` kbd hint |
| CS-F2 | Escape clears search only when no modal | Esc with each modal open vs closed | Search cleared only when all modals closed; keys swallowed otherwise | `PosPage.svelte:547-574` |
| CS-F3 | Debounce 400 ms; empty query bypasses | Type quickly; clear query | Single fetch after idle; clearing refetches immediately | `PosPage.svelte:157-160,199-211` |
| CS-F4 | No barcode-scanner buffer | Simulate rapid HID typing unfocused | Nothing added; scanner works only into focused search box | grep: no scanner logic in web/src |
| CS-F5 | Enter adds first result; empty search safe no-op | Enter with results / with none | Adds first match; no-op when empty | `PosPage.svelte:229-233` |
| CS-F6 | Double-click row adds; single selects | Interact with product table | Selection vs add distinction | `PosProductTable.svelte:72-73` |
| CS-F7 | sale_created WS overwrites reprint target | Complete sale on terminal B while A idle | A's Print button now points at B's invoice (cross-cashier reprint) — document | `PosPage.svelte:676-680` |
| CS-F8 | Reprint window = last 7 Jakarta days | Last sale older than 7 days | Print disabled | `PosPage.svelte:649-666`; `CartPanel.svelte:222-230` |
| CS-F9 | Receipt content & branding | Complete sale → print preview | Store name/header/footer from settings; fallback thanks line; app chrome hidden | `ReceiptPrintOverlay.svelte:11-19,77-86`; `app.css:8-43` |
| CS-F10 | Mobile bottom-sheet cart toggle | Viewport < lg | Toggle shows item-count pill; Show/Hide switches panel | `PosPage.svelte:715-751` |
| CS-F11 | i18n hardcoded strings spot-check | Switch locale id/en | Topbar Online/Offline, breadcrumbs, "Shift:", SearchBar placeholder stay EN; currency always id-ID/Rp | `Topbar.svelte:156-160,178-183`; `SearchBar.svelte:7,67` |
| CS-F12 | Shifts page uses native alert() | Trigger export/open/close failure | Native alert, not toast — document inconsistency | `ShiftsPage.svelte:81,97,115,131,168` |
| CS-F13 | Clear-cart partial failure leaves remainder | Mock one DELETE failing during ALT+Del | Remaining items kept; error toasted | `PosPage.svelte:359-375` |
| CS-F14 | SKU/barcode copy feedback | Click copy buttons | ✓ feedback ~2 s | `PosPage.svelte:318-330`; `PosProductTable.svelte:82-109` |
| CS-F15 | Negative RBAC sweep | Cashier hits /reports, /purchase-orders, /konsinyasi, product edit, inventory adjust | Blocked (no menu + direct URL denied) | permission matrix; `middleware/auth.go:77-96` |

**Area F automation status** (`tests/e2e/pos-ui-edge.spec.ts`, all passing):

- **Automated**: CS-F1, CS-F2, CS-F5, CS-F8, CS-F14, CS-F15 — plus six renamed smoke tests POS-UI-01…06 (F2 focus, F4 payment modal + Esc, ALT+DEL clear, decrease-at-qty-1 removes the line by design (`updateQty` → `removeFromCart`), Done gating, reload consistency after API-side add — there is no cart websocket topic, so realtime cart push is not a feature).
- **Inspection-only** (manual checklist, not automated): CS-F3 debounce timing (network-mock flakiness), CS-F4 scanner buffer (needs HID simulation), CS-F6 double-click vs single-click nuance, CS-F7 cross-cashier WS reprint target (needs two sessions), CS-F9 receipt branding (covered by print-receipt.spec basics), CS-F10 mobile viewport sheet, CS-F11 i18n spot-check, CS-F12 native alert() inconsistency (documented as-is), CS-F13 partial clear-cart failure (needs route mock).

Key-map change record: parked-sales modal moved F5→F7 (F5 freed for browser reload). CheckoutModal already used F7 for "Exact" fill while open, so the global handler now defers F7 to any open modal (`showCheckoutModal || showParkedModal || showCustomerModal` guard) — both shortcuts coexist.

---

## 3. Defect Watch (Phase 2 outcomes)

| ID | Finding | Verdict | Fix | Test |
|----|---------|---------|-----|------|
| D1 | Failed checkout shows two error toasts (`processCheckout` toasts then rethrows; `finalizeSale` catch toasts again) | **Confirmed — fixed** | Removed duplicate toast in `finalizeSale` catch (`PosPage.svelte`); dropped dead i18n key `toastCheckoutFailedRetry` (en/id). Vitest PosPage 30/30 pass | CS-D9 |
| D2 | Checkout into closed/nonexistent shift returns unmapped 500 | **Confirmed — fixed** | New sentinel `shared.ErrShiftNotOpen`; `UpdateShiftTotals` wraps it when the UPDATE misses (`total_updater.go`); mapped to 409 "shift is closed or no longer exists" in `cartError` and both direct-sale handler paths. `go test ./internal/shift ./internal/sale` pass | CS-A9, CS-E8 |
| D3 | `shift_id` taken from payload without ownership validation — sale attributable to another user's open shift | **Confirmed — fixed** | Ownership enforced atomically in `UpdateShiftTotals`: `ShiftSaleContribution.CashierID` added, WHERE clause now `id = $4 AND status = 'open' AND user_id = $5`; sale service passes `sale.CashierID`. Foreign/closed/missing shift all reject via `ErrShiftNotOpen` → same generic 409 (no existence leak). New subtest "another user's open shift rejects contribution" passes | CS-E9 |
| D4 | `sale.view` store-scoped not owner-scoped: arbitrary `cashier_id` param honored; GetSaleByID filters store only — cashier can read other cashiers' sales | **Confirmed — fixed** | `ownership.Resolve` applied in `GetSalesHistory` (non-`report.view` callers clamped to own `cashier_id`, requested filter never widens) and `GetSaleByID` (foreign sale → 404, no existence leak); elevation permission = `report.view`. Regression tests updated + new clamp/404/elevated cases pass; full `internal/sale` suite green | CS-E10 extension |
| D5 | Add-to-cart on an open cart with non-NULL `expired_at` returns unmapped 500: `AtomicGetOrCreateOpenCart` scanned `expired_at` (timestamptz) into `*string` (pgx binary-format scan error), and the CTE reused expired carts instead of starting fresh | **Confirmed — fixed** (found by E2E CS-A6) | `cart_repository.go`: scan via `sql.NullTime`; new `ExpireStaleOpenCarts` marks stale open carts `status='expired'` before the CTE; CTE + `GetOpenCartByCashier` now filter `expired_at IS NULL OR expired_at > NOW()`; unique-index race (O1) retried once on 23505. Regression tests in `cart_repository_expiry_test.go` pass | CS-A6, CS-A7, O1 |
| O1 | Concurrent first-cart race surfaces generic 500 (unique index not friendly-mapped) | **Confirmed — fixed** (via D5 retry) | `AtomicGetOrCreateOpenCart` retries once on unique-violation (23505) and re-reads the winner's row | CS-A10 |
| O2 | No rate limit on sale/shift endpoints; no idempotency keys anywhere (state-based safety only) | Confirmed — doc-only (accepted risk) | — | — |

Note on D3/D4 response codes: both surface as 404/409-style generic errors rather than 403 so row existence is not leaked to unauthorized callers (consistent with parked-sale recall behavior).

## 4. Traceability Matrix

| Area | Scenarios | New E2E file | Overlap with existing specs |
|------|-----------|--------------|------------------------------|
| A | CS-A1…A13 | `tests/e2e/cart-session.spec.ts` | hold-recall.spec covers legacy parked-sale flow; price-consistency E2E-02/03 cover hold/resume pricing |
| B | CS-B1…B8 | `tests/e2e/cart-session.spec.ts` | — |
| C | CS-C1…C7 | `tests/e2e/cart-session.spec.ts` | price-consistency.spec covers C1 partially |
| D | CS-D1…D10 | `tests/e2e/payment-validation.spec.ts` | pos-flow.spec covers split-tender mismatch/duplicate/ref/unknown at API level |
| E | CS-E1…E10 | `tests/e2e/shift-rules.spec.ts` | shifts.spec covers filters/export/pagination only |
| F | CS-F1…F15 (F1/F2/F5/F8/F14/F15 automated; rest inspection-only) | `tests/e2e/pos-ui-edge.spec.ts` (POS-UI-01…06 + CS subset) | print-receipt.spec covers F9 basics |

## 5. Test Data & Cleanup

- Reuse seeded products/categories/customers; create only: second cashier user (via admin API)
- Track created IDs (sales, carts, shifts, user) from API responses
- `tests/e2e/db-helper.ts`: psql/docker-exec wrapper for expiry backdate + cleanup
- Purge order: `sale_payments` → `sale_items` → `sales` → `cart_sessions` → `shifts` → test user → audit logs (if FK-restricted); runs in `afterAll`
- Shifts opened for the shared cashier are tracked (`trackShift`) so cleanup cascade-deletes them — an abandoned shift keeps stale denormalized `total_sales` counters after its tracked sales are deleted, which broke the CS-E4/E7 cross-check on subsequent runs

## 6. Deferred Test Improvements (test-review outcomes)

Findings from the post-implementation test review whose fixes were deliberately
deferred: they are polish/robustness items, not correctness bugs, and each
carries non-trivial change risk relative to its payoff. All P1 findings
(vacuous subtests, unpinned status contracts, missing 409 coverage) and the
actionable P2s (POS-UI-05 silent skip, magic user IDs, shift-counter pollution)
were fixed before commit.

| # | Finding | Location | Why deferred | Fix approach |
|---|---------|----------|--------------|--------------|
| 1 | Blind `waitForTimeout` sleeps (~15–20s cumulative) instead of condition-based waits | E2E specs (`pos-ui-edge.spec.ts`, `payment-validation.spec.ts`) | Specs are currently stable (42/42 twice consecutively); converting waits is a timing refactor that can introduce new flakiness mid-workstream | Replace with `expect(...).toBeVisible()` / `waitForFunction` polling on the awaited state |
| 2 | Loose error-message assertions (presence-only, not content) | CS-D7/D8/D1b (`payment-validation.spec.ts`, `pos-ui-edge.spec.ts`) | Status codes are already pinned; asserting exact copy couples tests to UI wording that i18n/copy edits would break | Assert stable substrings (e.g. "reference", method code) or backend error codes if exposed |
| 3 | CS-F8 backdating affects all of the shared cashier's sales, not just the fixture's | `pos-ui-edge.spec.ts` (reprint-window test) | Works today; narrowing requires per-fixture sale isolation rework | Create a dedicated cashier + sale for the backdate, or scope the SQL by tracked sale ID |
| 4 | POS-UI-01/02/03 duplicate scenarios already covered by `pos-flow.spec.ts` | `pos-ui-edge.spec.ts` vs `pos-flow.spec.ts` | Deduplication requires deciding which spec owns each scenario, revising this doc's coverage inventory | Merge or delete duplicates; update §4 traceability accordingly |
| 5 | `_ = setupSaleRouterWithPerms(...)` discarded-result smell | `internal/sale/security_regression_test.go` | Cosmetic; no behavioral impact | Use the returned router or drop the assignment |
| 6 | Naming consistency and string-built SQL interpolation in test helpers | `tests/e2e/db-helper.ts` | Style-only; queries take no untrusted input | Parameterize via psql `-v` vars or an identifier allowlist; align helper naming |
