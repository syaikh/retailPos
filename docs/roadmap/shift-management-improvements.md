# Shift Management Improvement Plan

**Document Version:** 1.0
**Date:** 2026-09-04
**Status:** PLANNING
**Review basis:** `docs/reviews/shift-management-review.md`

---

## Overview

This plan addresses 16 findings from the shift management review, organized into 5 phases. Each phase is independently deployable and builds on the previous one.

---

## Phase 1: Critical Fixes

Small-effort fixes for correctness bugs and data integrity gaps.

### 1.1 — `CloseAll` open-cart guard

**Problem:** `CloseAll` (`repository.go:481-535`) skips the open-cart check that `CloseShiftTx` performs. Force-closing a shift with active carts orphans cart sessions.

**Change:**
- File: `internal/shift/repository.go` — inside the `CloseAll` loop (line 507), before closing each shift, query `SELECT COUNT(*) FROM cart_sessions WHERE shift_id = $1 AND status = 'open'`
- If open carts exist, skip that shift and collect its ID in a `skippedIDs` list
- File: `internal/shift/handler.go` — update `CloseAll` response to include `skipped_shift_ids` alongside the closed IDs

**Files modified:** `internal/shift/repository.go`, `internal/shift/handler.go`
**Tests:** Add test case in `repository_test.go` — open cart prevents close in CloseAll, other shifts still close

### 1.2 — `CloseAll` record actual expected balance

**Problem:** `CloseAll` hardcodes `closing_balance = 0` and `discrepancy = 0`, discarding accounting data.

**Change:**
- File: `internal/shift/repository.go:526-527` — set `closing_balance = opening_balance + cash_sales` (the expected cash amount)
- Set `needs_review = true` for all force-closed shifts so a manager must reconcile
- Remove hardcoded `discrepancy = 0` — let it remain NULL (uncounted)

**Files modified:** `internal/shift/repository.go`

### 1.3 — `ReviewShift` status guard

**Problem:** `ReviewShift` UPDATE runs unconditionally — can review already-reviewed or open shifts.

**Change:**
- File: `internal/shift/repository.go:612-619` — add `WHERE needs_review = true AND status = 'closed'` to the UPDATE
- Check `RowsAffected()` — return "shift not pending review" error if 0

**Files modified:** `internal/shift/repository.go`, `internal/shift/service.go` (error handling)
**Tests:** Add test in `service_test.go` — review non-pending shift returns error

### 1.4 — Surprise audit flags for review

**Problem:** Audit result is read-only — not visible on the shift record.

**Change:**
- File: `internal/shift/handler.go:507-521` — after computing `off_by`, if `|off_by| > discrepancyThreshold`, update `shifts SET needs_review = true WHERE id = $1`
- Return `flagged_for_review: true` in the response so the UI can show a warning

**Files modified:** `internal/shift/handler.go`
**Tests:** Add test in `handler_extra_test.go` — large off_by triggers needs_review

---

## Phase 2: Cash Movement Tracking

Medium-effort feature. Adds the industry-standard cash drop and paid-in/paid-out tracking that is currently missing.

### 2.1 — Database: `cash_movements` table

**New migration:** `040_shift_cash_movements.sql`

```sql
CREATE TABLE cash_movements (
    id              SERIAL PRIMARY KEY,
    shift_id        INTEGER NOT NULL REFERENCES shifts(id) ON DELETE RESTRICT,
    user_id         INTEGER NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    type            VARCHAR(20) NOT NULL CHECK (type IN ('cash_drop', 'paid_in', 'paid_out')),
    amount          INTEGER NOT NULL CHECK (amount > 0),
    description     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_cash_movements_shift_id ON cash_movements(shift_id);
CREATE INDEX idx_cash_movements_shift_type ON cash_movements(shift_id, type);
```

Seed permissions:
```sql
INSERT INTO permissions (code, description) VALUES
('shift.cash_movement', 'Record cash drop / paid in / paid out');

-- cashier: record own movements
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'cashier' AND p.code = 'shift.cash_movement';

-- manager, admin, superadmin: view + record
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name IN ('manager', 'admin', 'superadmin')
AND p.code = 'shift.cash_movement';
```

### 2.2 — Backend: `cash_movement` sub-module

**New files:**
- `internal/shift/cash_movement.go` — domain type (`CashMovement`), repository (CRUD), handler
- `internal/shift/cash_movement_test.go`

**Endpoints:**
| Method | Path | Permission | Description |
|--------|------|-----------|-------------|
| `POST` | `/shifts/:id/cash-movements` | `shift.cash_movement` | Record a cash drop, paid-in, or paid-out |
| `GET` | `/shifts/:id/cash-movements` | `shift.view` | List movements for a shift |

**Business rules:**
- Only the shift owner can record movements on their own open shift
- `cash_drop` reduces expected cash; `paid_in` increases it; `paid_out` reduces it
- Each movement creates an atomic audit log entry
- Cannot record movements on a closed shift

**Files modified:** `internal/shift/repository.go`, `internal/shift/service.go`, `internal/shift/handler.go` (route registration), `cmd/server/main.go` (route wiring)

### 2.3 — Update shift close formula

**Change:**
- File: `internal/shift/repository.go:215`
- Current: `discrepancy = closingBalance - openingBalance - cashSales`
- New: `discrepancy = closingBalance - (openingBalance + cashSales - totalCashDrops + totalPaidIns - totalPaidOuts)`
- Add a helper `shiftCashMovementSummary(ctx, tx, shiftID)` that queries `SUM(CASE WHEN type='cash_drop' THEN amount ELSE 0 END)` etc.

**Include in close response:** `cash_drop_total`, `paid_in_total`, `paid_out_total` so the frontend can display the breakdown.

### 2.4 — Frontend: Cash movement UI

**New files:**
- `web/src/modules/shifts/components/CashMovementModal.svelte` — form with type selector (cash drop / paid in / paid out), amount input (CurrencyInput), description field

**Modified files:**
- `web/src/modules/shifts/components/ShiftsPage.svelte` — add "Record Movement" button to the active shift banner; show cash movement summary in the close modal
- `web/src/modules/shifts/components/ShiftDetailDrawer.svelte` — add cash movement list section
- `web/src/modules/shifts/stores/shift-store.svelte.ts` — add `doRecordCashMovement`, `loadCashMovements`
- `web/src/modules/shifts/services/shift-service.ts` — add `recordCashMovement`, `listCashMovements`

**Labels to add:** `cashDrop`, `paidIn`, `paidOut`, `recordMovement`, `cashMovements`, `movementType`, `movementDescription`

---

## Phase 3: Configurable Settings

Small-effort. Makes the discrepancy threshold and other shift behaviors configurable.

### 3.1 — Database: seed shift settings

**New migration:** `041_shift_settings.sql`

```sql
INSERT INTO app_settings (key, value) VALUES
('shift_discrepancy_threshold', '50000'),
('shift_blind_close', 'false');
```

### 3.2 — Backend: read settings at close time

**Changes:**
- File: `internal/shift/repository.go:217` — replace `const discrepancyThreshold = 50000` with value read from `appsettings.Service.GetMultiple`
- Inject `appsettings` dependency into shift service (via `SetAppSettingsService` pattern, consistent with existing port wiring)
- Cache settings in-memory with 60-second TTL (same pattern as `handler.go` public branding endpoint)

**Files modified:** `internal/shift/service.go`, `internal/shift/repository.go`, `internal/wiring/wiring.go`

### 3.3 — Frontend: settings UI

**Changes:**
- File: `web/src/modules/admin/components/SettingsPage.svelte` — add "Shift Management" section with:
  - Discrepancy threshold input (IDR, numeric)
  - Blind close toggle (boolean)
- File: `web/src/modules/admin/services/app-settings-service.ts` — ensure new keys are included in fetch/update
- File: `web/src/shared/stores/settings.svelte.ts` — add new keys to the store type

---

## Phase 4: Blind Close & Z-Report

Medium-effort. Two related UX improvements for shift close workflow.

### 4.1 — Blind close

**Changes:**
- Backend: no schema change needed — controlled by `app_settings.shift_blind_close`
- File: `web/src/modules/shifts/components/ShiftsPage.svelte` — in the close modal (line 462), conditionally hide the `expected_cash` display based on the setting
- Add a `showExpected` state toggle that the cashier can optionally reveal after counting
- Fetch `shift_blind_close` from settings on modal open

### 4.2 — Z-report (per-shift report)

**New files:**
- `web/src/modules/shifts/components/ShiftReport.svelte` — printable layout with:
  - Header: cashier name, store, open/close timestamps, duration
  - Payment method breakdown table (cash, card, e-wallet, etc.)
  - Cash reconciliation: opening float, cash sales, cash drops, paid in/out, expected, actual, variance
  - Void/refund summary (if applicable)
  - Transaction count

**New endpoint:** `GET /shifts/:id/report` — returns structured report data (same info as close response, enriched with payment method breakdown from `sale_payments`)

**Modified files:**
- `internal/shift/handler.go` — add `GetShiftReport` handler
- `internal/shift/repository.go` — add `GetShiftReportData` query joining `sales`, `sale_payments`, `cash_movements`
- `web/src/modules/shifts/components/ShiftDetailDrawer.svelte` — add "Print Report" button
- `web/src/modules/shifts/services/shift-service.ts` — add `getShiftReport`

**Print CSS:** Add `@media print` styles to `ShiftReport.svelte` for clean printing via `window.print()`

---

## Phase 5: Polish & Hardening

Low-effort nice-to-haves, can be done incrementally.

### 5.1 — Shift duration display

- File: `internal/shift/domain.go` — add `DurationMinutes *int` field (computed, not stored)
- File: `internal/shift/repository.go` — compute `duration = closed_at - opened_at` in `GetActiveShiftByUserID`, `ListShifts`, `GetShiftByID` queries (only when `closed_at IS NOT NULL`)
- Frontend: display duration in shift list table and detail drawer

### 5.2 — Store selector in open shift modal

- File: `web/src/modules/shifts/components/ShiftsPage.svelte` — fetch active stores on modal open, add dropdown
- Pass selected `store_id` to `doOpenShift()`

### 5.3 — `CloseAll` per-shift audit entries

- File: `internal/shift/repository.go:507-531` — inside the loop, create individual `shift_closed` audit entries with per-shift closing balance and discrepancy data

### 5.4 — Auto-close abandoned shifts

**New files:**
- `internal/shift/auto_close.go` — background job that runs every hour
- Query: `WHERE status = 'open' AND opened_at < NOW() - INTERVAL '1 hour' * $1`
- Auto-close with `closing_balance = opening_balance + cash_sales` (expected), `notes = 'Auto-closed due to timeout'`
- Create audit entry for each auto-closed shift
- Configurable timeout via `app_settings.shift_auto_close_hours` (0 = disabled)

**New migration:** add `shift_auto_close_hours` seed to `041_shift_settings.sql`

### 5.5 — Real-time shift totals

- File: `web/src/modules/shifts/components/ShiftsPage.svelte` — add polling (every 30 seconds) for active shift banner when on the shifts page
- Use `setInterval` + `store.loadActiveShift()` with cleanup on destroy

---

## File Change Summary

| Phase | New Files | Modified Files | New Migration |
|-------|-----------|----------------|---------------|
| 1 | — | `repository.go`, `handler.go`, `service.go` | — |
| 2 | `cash_movement.go`, `cash_movement_test.go`, `CashMovementModal.svelte` | `repository.go`, `service.go`, `handler.go`, `domain.go`, `ShiftsPage.svelte`, `ShiftDetailDrawer.svelte`, `shift-store.svelte.ts`, `shift-service.ts` | `040_shift_cash_movements.sql` |
| 3 | — | `repository.go`, `service.go`, `wiring.go`, `SettingsPage.svelte`, `settings.svelte.ts` | `041_shift_settings.sql` |
| 4 | `ShiftReport.svelte` | `handler.go`, `repository.go`, `ShiftDetailDrawer.svelte`, `shift-service.ts` | — |
| 5 | `auto_close.go` | `domain.go`, `repository.go`, `handler.go`, `ShiftsPage.svelte`, `ShiftDetailDrawer.svelte` | (add to `041`) |

---

## Testing Strategy

| Phase | Unit Tests | Integration Tests | E2E |
|-------|-----------|-------------------|-----|
| 1 | `repository_test.go` (CloseAll with carts, ReviewShift guard), `handler_extra_test.go` (audit flag) | — | — |
| 2 | `cash_movement_test.go` (CRUD, validation, shift-scoped) | `repository_test.go` (close formula with movements) | `tests/e2e/shifts.spec.ts` (record movement flow) |
| 3 | — | Verify setting read at close time | Settings page interaction |
| 4 | — | Shift report endpoint | Print report flow |
| 5 | `auto_close_test.go` (timeout, config) | Duration computation | Active shift polling |

---

## Execution Order

```
Phase 1 (Critical Fixes)
  └─► Phase 2 (Cash Movement Tracking)
       └─► Phase 3 (Configurable Settings)
            └─► Phase 4 (Blind Close & Z-Report)
                 └─► Phase 5 (Polish & Hardening)
```

Phase 1 should be merged first as it fixes correctness bugs. Phases 2-5 can be reordered or deferred independently, though Phase 3 should precede Phase 4 (blind close depends on the setting).
