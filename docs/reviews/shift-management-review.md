# Shift Management — Architecture & Feature Review

**Document Version:** 1.0
**Review Date:** 2026-09-04
**Reviewer:** opencode
**Scope:** `internal/shift/`, `web/src/modules/shifts/`, `internal/sale/` (shift contribution)
**Status:** ASSESSMENT COMPLETE

---

## Executive Summary

Shift management is implemented as a clean modular monolith with well-separated concerns: domain model, repository (SQL), service (business rules), handler (HTTP + export), and a unit-of-work accumulator for sale→shift totals. The frontend uses Svelte 5 runes with a reactive store, modals, and role-based UI gating.

**Strengths:** atomic sale→shift accumulation, one-open-shift-per-user DB constraint, ownership scoping, open-cart guard on close, cross-module port pattern (shift never imports sale).

**Key gaps:** no cash drop/paid-in/paid-out tracking (industry-standard feature), `CloseAll` skips open-cart check, surprise audit is read-only, hardcoded discrepancy threshold, no Z-report generation. These are detailed below.

---

## Architecture Overview

### Backend Module: `internal/shift/`

| File | Lines | Purpose |
|------|-------|---------|
| `domain.go` | 25 | `Shift` struct (16 fields, JSON-tagged) |
| `ports.go` | 44 | Consumer-side interfaces: `SalesSummaryProvider`, `StoreNameProvider`, `UsernameProvider` |
| `repository.go` | 639 | All SQL operations: Open, Close, CloseAll, List, Review, GetByID, GetActive, GetWithLiveSales |
| `service.go` | 107 | Business rules + thin delegation to repository |
| `handler.go` | 529 | HTTP handlers, route registration, CSV/XLSX export, audit log creation |
| `total_updater.go` | 43 | Unit-of-work contributor: accumulates sale amounts onto shift running totals |
| `repository_test.go` | 783 | Comprehensive DB-level tests |
| `service_test.go` | 231 | Service-level tests |
| `total_updater_test.go` | 138 | Unit-of-work tests |
| `handler_extra_test.go` | 165 | Handler edge cases, export tests |

### Cross-Module Contracts

| File | Purpose |
|------|---------|
| `internal/shared/shift.go` | `ShiftSaleContribution` struct — sale→shift accumulator contract |
| `internal/shared/shift_sale_summary.go` | `ShiftSaleSummary` struct — sale-side aggregate used at shift close |
| `internal/sale/shift_summary.go` | `ShiftSummaryProvider` — sale module's implementation of `SalesSummaryProvider` |
| `internal/sale/service.go:554-574` | `shiftContribution()` — computes cash/non-cash split per sale |
| `internal/sale/ports.go:31-42` | `ShiftTotalUpdater` interface — sale module's consumer-side port |

### Frontend Module: `web/src/modules/shifts/`

| File | Purpose |
|------|---------|
| `types/index.ts` | TypeScript interfaces: `Shift`, `ShiftFilters` |
| `services/shift-service.ts` | API client: `openShift`, `closeShift`, `getActiveShift`, `listShifts`, `reviewShift`, `auditShift`, `exportShifts` |
| `stores/shift-store.svelte.ts` | Svelte 5 runes-based reactive store (136 lines) |
| `components/ShiftsPage.svelte` | Main page: table, active shift banner, open/close/audit modals, export, pagination |
| `components/ShiftDetailDrawer.svelte` | Detail slide-over with review/audit buttons (role-gated) |

### Database Schema: `shifts` table

| Column | Type | Default | Notes |
|--------|------|---------|-------|
| `id` | SERIAL | auto | PK |
| `user_id` | INTEGER NOT NULL | — | FK → `users(id)` ON DELETE RESTRICT |
| `store_id` | INTEGER | NULL | FK → `stores(id)` ON DELETE SET NULL |
| `status` | VARCHAR(20) NOT NULL | `'open'` | CHECK `IN ('open','closed')` |
| `opening_balance` | INTEGER NOT NULL | `0` | IDR cents |
| `closing_balance` | INTEGER | NULL | IDR cents |
| `cash_sales` | INTEGER NOT NULL | `0` | Accumulated by `TotalUpdater` |
| `non_cash_sales` | INTEGER NOT NULL | `0` | Accumulated by `TotalUpdater` |
| `total_sales` | INTEGER NOT NULL | `0` | Accumulated by `TotalUpdater` |
| `transaction_count` | INTEGER NOT NULL | `0` | Incremented per sale |
| `discrepancy` | INTEGER | NULL | Computed at close |
| `notes` | TEXT | NULL | Optional close notes |
| `needs_review` | BOOLEAN NOT NULL | `false` | `true` if `|discrepancy| > 50,000` |
| `reviewed_by` | INTEGER | NULL | FK → `users(id)` |
| `reviewed_at` | TIMESTAMPTZ | NULL | |
| `opened_at` | TIMESTAMPTZ NOT NULL | `now()` | |
| `closed_at` | TIMESTAMPTZ | NULL | |
| `created_at` | TIMESTAMPTZ NOT NULL | `now()` | |
| `updated_at` | TIMESTAMPTZ NOT NULL | `now()` | |

**Key constraint:** `uq_open_shift_per_user` — UNIQUE partial index on `(user_id) WHERE status = 'open'`, enforcing exactly one open shift per user at the DB level.

**Relationships:**
- `sales.shift_id` → `shifts.id` (FK) — each sale linked to a shift
- `cart_sessions.shift_id` → `shifts.id` (FK) — carts linked to shifts

---

## API Endpoints

| Method | Path | Permission | Handler | Description |
|--------|------|-----------|---------|-------------|
| `POST` | `/shifts/open` | `shift.create` | `OpenShift` | Open new shift (atomic with audit log) |
| `POST` | `/shifts/:id/close` | `shift.create` | `CloseShift` | Close shift with sales summary snapshot |
| `POST` | `/shifts/close-all` | `shift.create` | `CloseAll` | Admin: force-close all open shifts |
| `POST` | `/shifts/:id/review` | `shift.review` | `ReviewShift` | Review/approve discrepancy |
| `POST` | `/shifts/:id/audit` | `shift.audit` | `AuditShift` | Surprise cash audit (read-only) |
| `GET` | `/shifts/active` | (auth only) | `GetActiveShift` | Current user's open shift with live totals |
| `GET` | `/shifts` | `shift.view` | `ListShifts` | Paginated list with filters |
| `GET` | `/shifts/export` | `shift.view` | `ExportShifts` | CSV/XLSX download |
| `GET` | `/shifts/:id` | `shift.view` | `GetShiftByID` | Single shift detail |

---

## Business Logic Flow

### Open Shift

1. Validation: `opening_balance > 0`
2. Duplicate check: queries for existing open shift for same user (application-level)
3. DB constraint: partial unique index prevents race conditions
4. Insert row with `status = 'open'`, `opened_at = NOW()`
5. Audit log: `shift_opened` created atomically in same transaction

### Close Shift

1. Validation: `closing_balance >= 0`
2. Lock row: `SELECT ... FOR UPDATE OF s` — prevents concurrent close
3. Ownership check: `WHERE user_id = $2 AND status = 'open'`
4. **Open cart guard:** rejects close if any `cart_sessions` with `shift_id = this shift` has `status = 'open'`
5. Sales summary: calls `ShiftSummaryInTx()` to compute cash_sales, non_cash_sales, total_sales, transaction_count within the same transaction (snapshot consistency)
6. Discrepancy: `closing_balance - opening_balance - cash_sales`
7. Review flag: `needs_review = true` if `|discrepancy| > 50,000` IDR
8. Update shift row with all computed totals, `status = 'closed'`
9. Audit log: `shift_closed` created atomically

### Sale → Shift Accumulation (Unit of Work)

Called from `sale.service.finalizeSaleCreation()` (`internal/sale/service.go:546`) **within the same DB transaction** as the sale:

```sql
UPDATE shifts
SET cash_sales = cash_sales + $1,
    non_cash_sales = cash_sales + $2,
    total_sales = total_sales + $3,
    transaction_count = transaction_count + 1,
    updated_at = NOW()
WHERE id = $4 AND status = 'open' AND user_id = $5
```

- Rejects if shift is not open or doesn't belong to the cashier
- `shiftContribution()` computes cash/non-cash split; `changeDue` subtracted from cash (over-tender only on CASH line)

### CloseAll (Admin)

1. Fetches all open shifts with `FOR UPDATE`
2. For each shift: computes sales summary, sets `closing_balance = 0`, `discrepancy = 0`
3. Does **not** check for open carts (gap — see CRIT-01)
4. Commits all in one transaction

### Surprise Audit

1. Computes `expected = opening_balance + cash_sales` (live query for open shifts)
2. Returns `off_by = actual_balance - expected`
3. Logged as audit entry only — does **not** modify the shift row or flag for review

---

## Findings

### CRIT-01: `CloseAll` Skips Open Cart Check

**Location:** `internal/shift/repository.go:481-535`

`CloseShiftTx` (line 191-198) checks `cart_sessions` for open carts before allowing close. `CloseAll` does **not** perform this check. This can orphan open cart sessions — a shift is force-closed while a cart is mid-checkout, leaving the cart in limbo with a `shift_id` pointing at a closed shift.

**Recommendation:** Add the same open-cart guard inside the `CloseAll` loop before closing each shift. Either skip shifts with open carts (and report which shifts were skipped), or abort the entire operation.

---

### CRIT-02: `CloseAll` Hardcodes `closing_balance=0` and `discrepancy=0`

**Location:** `internal/shift/repository.go:513-528`

When an admin force-closes shifts via `CloseAll`, the closing balance is hardcoded to `0` and discrepancy to `0`. This discards the actual cash position. For end-of-day reconciliation, the finance team has no record of how much cash was expected vs. what was in the drawer.

**Recommendation:** Compute the expected cash (`opening_balance + cash_sales`) and use it as the closing balance. Set discrepancy to `0` only if the admin confirms the actual count matches expected. Alternatively, prompt for a closing balance input per shift.

---

### CRIT-03: No Cash Drop / Safe Drop Tracking

**Industry standard:** Every major POS system (Square, Toast, Clover, Dynamics 365, HotelBee, AWRA) supports mid-shift cash drops — recording cash removed from the drawer and placed in a safe. This keeps the expected cash calculation realistic during a shift and at close.

**Current behavior:** The only cash movements tracked are opening balance and sales. If a cashier drops excess cash to the safe during a busy shift, the `cash_sales` still reflects the total cash tendered, but the physical drawer has less cash. At close, the formula `closing - opening - cash_sales` reports a phantom shortage.

**Recommendation:** Add a `cash_drops` table or a `cash_movements` table tracking type (drop/paid-in/paid-out), amount, timestamp, and user. Update the expected cash formula to account for drops.

---

### CRIT-04: No Paid-In / Paid-Out Tracking

**Industry standard:** Non-sale cash movements (petty cash, vendor refunds, delivery payments) are standard features in POS shift management. Without them, any cash entering or leaving the drawer outside of sales creates unexplained discrepancies.

**Current behavior:** No mechanism to record paid-in or paid-out events. The shift summary only considers sale-related cash flows.

**Recommendation:** Extend the cash movement system (see CRIT-03 recommendation) to include `paid_in` and `paid_out` types. Include these in the expected cash calculation at close.

---

### HIGH-01: Surprise Audit Is Read-Only

**Location:** `internal/shift/handler.go:483-529`

The audit endpoint computes the difference and logs it, but never:
- Updates the shift record with the audited balance
- Flags the shift for review if the variance exceeds a threshold
- Records the audit result on the shift itself

The audit result only exists in the `audit_logs` table. A manager reviewing the shift list later has no visibility into "this shift was audited and was off by X" without querying audit logs separately.

**Recommendation:** After computing `off_by`, update the shift with a `last_audit_at`, `last_audit_off_by` field, or at minimum set `needs_review = true` if the variance exceeds the threshold.

---

### HIGH-02: `CloseAll` Creates Single Audit Entry

**Location:** `internal/shift/handler.go:204-213`

`CloseAll` creates a single `shift_close_all` audit entry with the list of shift IDs, rather than individual `shift_closed` entries per shift. This makes per-shift audit trails incomplete — there's no atomic `shift_closed` record with the closing balance and discrepancy for each shift closed via this path.

**Recommendation:** Inside the `CloseAll` transaction loop, create individual `shift_closed` audit entries for each shift (with the shift-specific data), or at minimum include per-shift closing balance/discrepancy in the aggregated audit entry.

---

### MED-01: Hardcoded Discrepancy Threshold

**Location:** `internal/shift/repository.go:217`

```go
const discrepancyThreshold = 50000
```

The threshold for flagging shifts for review is hardcoded at 50,000 IDR. Different stores or business contexts may need different thresholds.

**Recommendation:** Make this configurable via the `app_settings` table (already exists in the schema) or environment variable. Read it at close time from the configuration.

---

### MED-02: No Blind Close Option

**Location:** `web/src/modules/shifts/components/ShiftsPage.svelte:462`

The close modal shows `expected_cash` (opening balance + cash sales) before the cashier enters their count. This creates anchoring bias — the cashier may unconsciously adjust their count toward the expected number.

**Industry best practice:** Offer a "blind close" mode where the cashier enters their count without seeing the expected amount. The system then computes the discrepancy after submission.

**Recommendation:** Add a configurable option (per-store or per-role) to hide the expected cash until after the cashier submits their count.

---

### MED-03: No Z-Report / Per-Shift Report Generation

**Industry standard:** A Z-report (shift report) is a printable/PDF summary of a single shift, including payment method breakdown, voids, tax summary, and cash reconciliation. This is a standard deliverable in every retail POS.

**Current behavior:** Export functionality generates CSV/XLSX for all shifts matching filters. There is no per-shift report.

**Recommendation:** Add a per-shift report endpoint that generates a formatted summary (HTML/PDF) with:
- Cashier name, store, open/close timestamps
- Payment method breakdown (cash, card, e-wallet, etc.)
- Cash reconciliation: opening float, cash sales, cash drops, expected, actual, variance
- Voids and refunds processed during the shift

---

### MED-04: No Store Selector in Open Shift Modal

**Location:** `web/src/modules/shifts/components/ShiftsPage.svelte:91`

```javascript
await store.doOpenShift(null, openingBalance);
```

The UI always passes `null` for `store_id`, even though the backend supports it. In a multi-store environment, shifts should be associated with a store.

**Recommendation:** Add a store selector dropdown to the open shift modal, populated from the stores list.

---

### MED-05: `ReviewShift` Has No Status Guard

**Location:** `internal/shift/repository.go:611-624`

The `ReviewShift` UPDATE runs unconditionally — it can review an already-reviewed shift, or attempt to review an open shift. While not harmful (setting `needs_review = false` on an already-reviewed shift is idempotent), it's semantically incorrect.

**Recommendation:** Add a `WHERE needs_review = true AND status = 'closed'` guard to the UPDATE query.

---

### LOW-01: No Shift Duration Tracking

`opened_at` and `closed_at` exist, but no `duration_minutes` computed column. Reporting on shift length requires re-computation from timestamps.

**Recommendation:** Either add a generated column (`duration_minutes = EXTRACT(EPOCH FROM (closed_at - opened_at)) / 60`) or compute it at read time.

---

### LOW-02: No Auto-Close for Abandoned Shifts

If a cashier forgets to close (walks out), the shift stays open indefinitely. This prevents the cashier from opening a new shift and blocks reporting.

**Recommendation:** Add a background job that auto-closes shifts open longer than a configurable duration (e.g., 24 hours) with a manager notification and a note "auto-closed due to timeout".

---

### LOW-03: No Real-Time Shift Totals

The active shift banner loads once on mount (`onMount(() => store.loadActiveShift())`). During an active shift, the cashier must refresh the page to see updated totals.

**Recommendation:** Add polling (every 30-60 seconds) or WebSocket updates for the active shift's running totals.

---

### LOW-04: No Variance Trend Analysis

`discrepancy` is recorded per shift but never aggregated. Best practice: track cashier variance patterns over time to identify training issues or potential theft.

**Recommendation:** Add a report or dashboard widget showing average variance per cashier over time, flagging persistent patterns.

---

### LOW-05: No Drawer Assignment

Shifts are user-scoped but not drawer-scoped. In environments with shared physical drawers, there's no way to track which drawer was used.

**Recommendation:** This is a low priority for single-drawer setups, but could be added as a `drawer_id` column for environments with multiple physical drawers per counter.

---

## Summary Table

| ID | Severity | Finding | Effort | Impact |
|----|----------|---------|--------|--------|
| CRIT-01 | Critical | `CloseAll` skips open cart check | Small | Orphaned cart sessions |
| CRIT-02 | Critical | `CloseAll` hardcodes closing_balance=0 | Small | Lost accounting data |
| CRIT-03 | Critical | No cash drop / safe drop tracking | Medium | Phantom shortages at close |
| CRIT-04 | Critical | No paid-in / paid-out tracking | Medium | Unexplained discrepancies |
| HIGH-01 | High | Surprise audit is read-only | Small | Audit results not visible on shift |
| HIGH-02 | High | `CloseAll` single audit entry | Small | Incomplete per-shift audit trail |
| MED-01 | Medium | Hardcoded discrepancy threshold | Small | Inflexible for different stores |
| MED-02 | Medium | No blind close option | Small | Anchoring bias |
| MED-03 | Medium | No Z-report generation | Medium | Missing standard retail feature |
| MED-04 | Medium | No store selector in open modal | Small | Multi-store readiness |
| MED-05 | Medium | `ReviewShift` no status guard | Small | Idempotent but semantically wrong |
| LOW-01 | Low | No shift duration tracking | Small | Reporting convenience |
| LOW-02 | Low | No auto-close for abandoned shifts | Medium | Indefinitely-open shifts |
| LOW-03 | Low | No real-time shift totals | Medium | UX for active cashiers |
| LOW-04 | Low | No variance trend analysis | Medium | Loss prevention insight |
| LOW-05 | Low | No drawer assignment | Small | Multi-drawer environments |

---

## What's Done Well

- **Atomic sale→shift accumulation** — `TotalUpdater` runs inside the sale transaction, so a sale and its shift contribution always commit or rollback together. This is a textbook Unit of Work pattern.
- **One-open-shift-per-user** enforced at both application and DB level (partial unique index), with race condition testing.
- **Cross-module port pattern** — shift never imports `internal/sale`. The `SalesSummaryProvider` interface is wired at startup, keeping modules decoupled.
- **Ownership scoping** — cashiers see only their own shifts via `ownership.Scope`, managers/admins see all.
- **Open cart guard** on normal close prevents orphaned cart sessions.
- **Live sales fallback** for active shifts — if running totals are zero, queries `sale_payments` directly for real-time accuracy.
- **Comprehensive test coverage** — repository (783 lines), service (231), total_updater (138), handler extras (165), E2E (325).
- **CSV/XLSX export** with styled headers and proper filters.
- **CashBreakdown component** in close modal for physical cash counting.
- **Logout guard** prevents cashiers from logging out while a shift is open.
