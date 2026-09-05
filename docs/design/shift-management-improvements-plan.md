# Implementation Plan: Shift Management Improvements

| Field | Value |
|-------|-------|
| Status | **In Progress** |
| Date | 2026-09-04 |
| Updated | 2026-09-05 |
| Review | `docs/reviews/shift-management-review.md` |
| Estimated Effort | 8-10 days |

---

## Overview

This document provides a step-by-step implementation plan for shift management improvements, organized into 5 phases. Each phase is independently deployable. The plan addresses 16 findings from the review, prioritized by impact.

**Reading order:** This plan assumes familiarity with `docs/reviews/shift-management-review.md` (findings) and `docs/roadmap/shift-management-improvements.md` (high-level phases).

---

## Phase 1: Critical Fixes

> **Status:** ✅ Completed

**Effort:** 1-1.5 days
**Goal:** Fix correctness bugs in existing shift operations.

### Step 1.1 — ~~`CloseAll` open-cart guard~~ REMOVED

> **Status:** Removed — `CloseAll` endpoint deleted in commit `54fa21d`. No longer applicable.

### Step 1.2 — ~~`CloseAll` record actual expected balance~~ REMOVED

> **Status:** Removed — `CloseAll` endpoint deleted in commit `54fa21d`. No longer applicable.

### Step 1.3 — `ReviewShift` status guard

> **Status:** ✅ Completed

**File:** `internal/shift/repository.go` — `ReviewShift()` (line 554)

**What to change:**
```go
func (r *Repository) ReviewShift(ctx context.Context, shiftID, reviewerID int) (*Shift, error) {
    result, err := r.db.Exec(ctx, `
        UPDATE shifts
        SET needs_review = false,
            reviewed_by = $1,
            reviewed_at = NOW(),
            updated_at = NOW()
        WHERE id = $2 AND needs_review = true AND status = 'closed'
    `, reviewerID, shiftID)
    if err != nil {
        return nil, fmt.Errorf("failed to review shift: %w", err)
    }
    if result.RowsAffected() == 0 {
        return nil, fmt.Errorf("shift not pending review or not found")
    }
    return r.GetShiftByID(ctx, ownership.Scope{}, shiftID)
}
```

**Tests — `service_test.go`:**
- `TestReviewShift_AlreadyReviewed` — review a shift twice, second returns error
- `TestReviewShift_OpenShift` — attempt to review an open shift, returns error

### Step 1.4 — Surprise audit flags for review

> **Status:** ✅ Completed

**File:** `internal/shift/handler.go` — `AuditShift()` (line 442)

**What to change:**
After computing `off_by` (line 505), add:
```go
if offBy < 0 { offBy = -offBy } // absolute value
const discrepancyThreshold = 50000 // TODO: read from app_settings in Phase 3
if offBy > discrepancyThreshold {
    _, _ = h.svc.(*service).repo.db.Exec(c.Request.Context(),
        `UPDATE shifts SET needs_review = true, updated_at = NOW() WHERE id = $1`, shiftID,
    )
}
```

Better approach — add a `FlagForReview` method to the repository and call it from the service:
- `internal/shift/repository.go`: `FlagForReview(ctx, shiftID) error`
- `internal/shift/service.go`: `FlagForReview(ctx, shiftID) error`
- `internal/shift/handler.go`: call after audit computation

**Response change:**
```go
shared.JSONSuccess(c, gin.H{
    "shift":           shift,
    "expected_cash":   expected,
    "actual_balance":  req.ActualBalance,
    "off_by":          offBy,
    "flagged_for_review": offBy < -50000 || offBy > 50000,
})
```

**Tests — `handler_extra_test.go`:**
- `TestAuditShift_LargeVariance_TriggersReview` — audit with large off_by, verify needs_review is set

---

## Phase 2: Cash Movement Tracking

**Effort:** 3-4 days
**Goal:** Add industry-standard cash drop and paid-in/paid-out tracking.

### Step 2.1 — Database migration

**New file:** `database/migrations/040_shift_cash_movements.sql`

```sql
-- 040_shift_cash_movements.sql
-- Cash movement tracking for shift reconciliation.

CREATE TABLE IF NOT EXISTS cash_movements (
    id              SERIAL PRIMARY KEY,
    shift_id        INTEGER NOT NULL REFERENCES shifts(id) ON DELETE RESTRICT,
    user_id         INTEGER NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    type            VARCHAR(20) NOT NULL,
    amount          INTEGER NOT NULL,
    description     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT cash_movements_type_check
        CHECK (type IN ('cash_drop', 'paid_in', 'paid_out')),
    CONSTRAINT cash_movements_amount_check
        CHECK (amount > 0)
);

CREATE INDEX idx_cash_movements_shift_id ON cash_movements(shift_id);
CREATE INDEX idx_cash_movements_shift_type ON cash_movements(shift_id, type);

-- Permissions
INSERT INTO permissions (code, description) VALUES
    ('shift.cash_movement', 'Record cash drop / paid in / paid out');

-- Cashier: record own movements on own shifts
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'cashier' AND p.code = 'shift.cash_movement';

-- Manager, admin, superadmin: full access
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name IN ('manager', 'admin', 'superadmin')
AND p.code = 'shift.cash_movement';
```

**Verify:** Run migration against dev DB, confirm table and permissions created.

### Step 2.2 — Domain layer

**New file:** `internal/shift/cash_movement.go`

```go
package shift

import "time"

type CashMovement struct {
    ID          int       `json:"id"`
    ShiftID     int       `json:"shift_id"`
    UserID      int       `json:"user_id"`
    Username    string    `json:"username,omitempty"`
    Type        string    `json:"type"`        // cash_drop, paid_in, paid_out
    Amount      int       `json:"amount"`
    Description *string   `json:"description,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
}

type CashMovementSummary struct {
    CashDrops  int `json:"cash_drops"`
    PaidIns    int `json:"paid_ins"`
    PaidOuts   int `json:"paid_outs"`
    NetEffect  int `json:"net_effect"` // -cash_drops + paid_ins - paid_outs
}

var (
    ErrShiftClosed          = errors.New("cannot record movement on a closed shift")
    ErrInvalidMovementType  = errors.New("invalid movement type")
    ErrNotShiftOwner        = errors.New("only the shift owner can record movements")
)
```

### Step 2.3 — Repository layer

**File:** `internal/shift/cash_movement.go` (same file, add methods)

Methods to add to `Repository`:
- `CreateCashMovement(ctx, tx, shiftID, userID, movementType, amount, description) (*CashMovement, error)`
  - Check shift is open and belongs to user
  - INSERT into `cash_movements`
- `ListCashMovements(ctx, shiftID) ([]CashMovement, error)`
  - SELECT with username resolution via `UsernameProvider`
- `ShiftCashMovementSummary(ctx, tx, shiftID) (CashMovementSummary, error)`
  - Aggregate query:
    ```sql
    SELECT
        COALESCE(SUM(CASE WHEN type = 'cash_drop' THEN amount ELSE 0 END), 0),
        COALESCE(SUM(CASE WHEN type = 'paid_in' THEN amount ELSE 0 END), 0),
        COALESCE(SUM(CASE WHEN type = 'paid_out' THEN amount ELSE 0 END), 0)
    FROM cash_movements WHERE shift_id = $1
    ```

### Step 2.4 — Service layer

**File:** `internal/shift/service.go`

Add to `Repo` interface:
```go
CreateCashMovement(ctx context.Context, tx pgx.Tx, shiftID, userID int, movementType string, amount int, description *string) (*CashMovement, error)
ListCashMovements(ctx context.Context, shiftID int) ([]CashMovement, error)
ShiftCashMovementSummary(ctx context.Context, tx pgx.Tx, shiftID int) (CashMovementSummary, error)
```

Add service methods:
- `RecordCashMovement(ctx, shiftID, userID, movementType, amount, description) (*CashMovement, error)`
  - Validates type is one of: `cash_drop`, `paid_in`, `paid_out`
  - Validates amount > 0
  - Delegates to repo within a transaction that also creates an audit log
- `GetCashMovements(ctx, shiftID) ([]CashMovement, error)`
- `GetCashMovementSummary(ctx, shiftID) (CashMovementSummary, error)`

### Step 2.5 — Update shift close formula

**File:** `internal/shift/repository.go` — `CloseShiftTx()` (line 210-215)

**What to change:**
After computing `summary` (line 210), also compute cash movement summary:
```go
movementSummary, err := r.ShiftCashMovementSummary(ctx, tx, shiftID)
if err != nil {
    return nil, fmt.Errorf("failed to calculate cash movements: %w", err)
}

// Expected cash = opening + cash_sales - cash_drops + paid_ins - paid_outs
expectedCash := shift.OpeningBalance + summary.TotalCashSales + movementSummary.NetEffect
discrepancy := closingBalance - expectedCash
```

Update the `discrepancy` calculation and the `needs_review` check. Also add movement totals to the returned shift response.

**File:** `internal/shift/domain.go`

Add fields to `Shift` struct:
```go
CashDropTotal  int `json:"cash_drop_total,omitempty"`
PaidInTotal    int `json:"paid_in_total,omitempty"`
PaidOutTotal   int `json:"paid_out_total,omitempty"`
```

### Step 2.6 — Handler layer

**File:** `internal/shift/handler.go`

Add routes to `RegisterRoutes`:
```go
r.POST("/shifts/:id/cash-movements", auth, perm(permissions.ShiftCashMovement), h.RecordCashMovement)
r.GET("/shifts/:id/cash-movements", auth, perm(permissions.ShiftView), h.ListCashMovements)
```

Add handlers:
- `RecordCashMovement` — validates request body `{type, amount, description}`, calls service, creates audit log
- `ListCashMovements` — returns movements list for a shift

**File:** `internal/permissions/permissions.go`

Add constant:
```go
ShiftCashMovement Code = "shift.cash_movement"
```

**File:** `cmd/server/main.go`

No change needed — routes registered via existing `RegisterRoutes` pattern.

### Step 2.7 — Wiring

**File:** `internal/wiring/wiring.go`

No new wiring needed — cash movement methods are on the existing `shift.Repository` and `shift.Service`. The `appsettings` dependency will be added in Phase 3.

### Step 2.8 — Tests

**New file:** `internal/shift/cash_movement_test.go`

Tests:
- `TestCreateCashMovement_Success` — cash drop, paid_in, paid_out
- `TestCreateCashMovement_ClosedShift` — rejected
- `TestCreateCashMovement_NotOwner` — rejected
- `TestCreateCashMovement_InvalidType` — rejected
- `TestCreateCashMovement_ZeroAmount` — rejected
- `TestListCashMovements` — returns list with usernames
- `TestShiftCashMovementSummary` — aggregates correctly

**File:** `internal/shift/repository_test.go`

Add:
- `TestCloseShift_WithCashMovements` — verify discrepancy formula accounts for movements
- `TestCloseShift_CashDropReducesExpected` — cash drop means less expected cash

### Step 2.9 — Frontend: Cash movement service & store

**File:** `web/src/modules/shifts/services/shift-service.ts`

Add:
```typescript
export async function recordCashMovement(
  shiftId: number,
  type: 'cash_drop' | 'paid_in' | 'paid_out',
  amount: number,
  description: string | null,
): Promise<CashMovement> { ... }

export async function listCashMovements(shiftId: number): Promise<CashMovement[]> { ... }
```

**File:** `web/src/modules/shifts/types/index.ts`

Add:
```typescript
export interface CashMovement {
  id: number;
  shift_id: number;
  user_id: number;
  username?: string;
  type: 'cash_drop' | 'paid_in' | 'paid_out';
  amount: number;
  description?: string;
  created_at: string;
}
```

**File:** `web/src/modules/shifts/stores/shift-store.svelte.ts`

Add methods:
- `doRecordCashMovement(shiftId, type, amount, description)` — calls API, appends to local state
- `loadCashMovements(shiftId)` — fetches and caches movements

### Step 2.10 — Frontend: Cash movement modal

**New file:** `web/src/modules/shifts/components/CashMovementModal.svelte`

- Modal with type selector (3 buttons: Cash Drop, Paid In, Paid Out)
- CurrencyInput for amount
- Optional description text input
- Submit button calls `store.doRecordCashMovement()`
- Shows toast on success/error

### Step 2.11 — Frontend: Wire into shift pages

**File:** `web/src/modules/shifts/components/ShiftsPage.svelte`

- Add "Record Movement" button to the active shift banner (next to Close Shift button)
- In close modal: add cash movement summary section showing totals (cash drops, paid ins, paid outs, net effect)
- Update expected cash display to include movement net effect

**File:** `web/src/modules/shifts/components/ShiftDetailDrawer.svelte`

- Add "Cash Movements" section listing all movements with type, amount, description, timestamp, user
- Show summary totals at the bottom

---

## Phase 3: Configurable Settings

**Effort:** 1 day
**Goal:** Make discrepancy threshold and blind close configurable.

### Step 3.1 — Database migration

**New file:** `database/migrations/041_shift_settings.sql`

```sql
INSERT INTO app_settings (key, value) VALUES
    ('shift_discrepancy_threshold', '50000'),
    ('shift_blind_close', 'false');
```

### Step 3.2 — Backend: inject appsettings dependency

**File:** `internal/shift/service.go`

Add a port for settings access:
```go
type SettingsProvider interface {
    GetMultiple(ctx context.Context, keys []string) (map[string]string, error)
}
```

Add to `service` struct:
```go
type service struct {
    repo     *Repository
    settings SettingsProvider  // NEW
}

func (s *service) SetSettingsProvider(p SettingsProvider) {
    s.settings = p
}
```

Add helper:
```go
func (s *service) getDiscrepancyThreshold(ctx context.Context) int {
    if s.settings == nil {
        return 50000 // default
    }
    settings, err := s.settings.GetMultiple(ctx, []string{"shift_discrepancy_threshold"})
    if err != nil {
        return 50000
    }
    if v, ok := settings["shift_discrepancy_threshold"]; ok {
        if n, err := strconv.Atoi(v); err == nil && n > 0 {
            return n
        }
    }
    return 50000
}
```

**File:** `internal/shift/repository.go` — `CloseShiftTx()` (line 217)

Replace hardcoded threshold with parameter passed from service. Change `CloseShiftTx` signature to accept threshold, or have the service call a `SetDiscrepancyThreshold` on the repository before calling close.

Better approach — pass threshold from service to repository:
```go
func (r *Repository) CloseShiftTx(ctx context.Context, tx pgx.Tx, shiftID, userID int, closingBalance int, notes *string, discrepancyThreshold int) (*Shift, error) {
```

### Step 3.3 — Wiring

**File:** `internal/wiring/wiring.go`

After creating `d.ShiftSvc` (line 506), add:
```go
d.ShiftSvc.SetSettingsProvider(d.AppSettingsSvc)
```

### Step 3.4 — Frontend: settings UI

**File:** `web/src/modules/admin/components/SettingsPage.svelte`

Add a "Shift Management" section after the existing settings sections:
- Discrepancy Threshold: number input (IDR), default 50000
- Blind Close: toggle switch, default false

**File:** `web/src/modules/admin/services/app-settings-service.ts`

Ensure `shift_discrepancy_threshold` and `shift_blind_close` are included in the settings payload sent to/from `PUT /api/settings`.

**File:** `web/src/shared/stores/settings.svelte.ts`

Add to the store type:
```typescript
shiftDiscrepancyThreshold: number;
shiftBlindClose: boolean;
```

### Step 3.5 — Tests

- Verify close uses configured threshold (unit test with mock settings provider)
- Verify default threshold used when settings unavailable
- Frontend: settings page saves and loads new keys

---

## Phase 4: Blind Close & Z-Report

**Effort:** 2-3 days
**Goal:** Reduce anchoring bias in cash counting; add per-shift printable report.

### Step 4.1 — Blind close

**File:** `web/src/modules/shifts/components/ShiftsPage.svelte`

In the close modal (around line 462):
1. Fetch `shift_blind_close` from settings on modal open
2. Conditionally hide the `expected_cash` display:
   ```svelte
   {#if !blindCloseEnabled || showExpected}
     <div>
       <p class="text-xs text-text-muted">{labels.expectedCash}</p>
       <p class="text-lg font-bold text-primary">{formatMoney(expected)}</p>
     </div>
   {/if}
   ```
3. Add a "Show Expected" toggle button that sets `showExpected = true`
4. After the cashier submits their count, always show the discrepancy result

**No backend changes needed** — controlled entirely by the `app_settings.shift_blind_close` value.

### Step 4.2 — Z-report endpoint

**File:** `internal/shift/repository.go`

Add method:
```go
func (r *Repository) GetShiftReportData(ctx context.Context, shiftID int) (*ShiftReportData, error) {
    // 1. Get shift row (all fields)
    // 2. Get payment method breakdown from sale_payments via summary provider
    // 3. Get cash movements summary
    // 4. Compute duration from opened_at / closed_at
    // 5. Return enriched report data
}
```

**File:** `internal/shift/domain.go`

Add domain type:
```go
type ShiftReportData struct {
    Shift
    DurationMinutes    int                    `json:"duration_minutes"`
    PaymentBreakdown   []PaymentMethodTotal   `json:"payment_breakdown"`
    CashMovementSummary CashMovementSummary   `json:"cash_movement_summary"`
}

type PaymentMethodTotal struct {
    Method string `json:"method"`
    Amount int    `json:"amount"`
    Count  int    `json:"count"`
}
```

**File:** `internal/sale/shift_summary.go`

Add a new method to `ShiftSummaryProvider` (or create a new port):
```go
func (ShiftSummaryProvider) PaymentMethodBreakdown(ctx context.Context, db shared.DBPool, shiftID int) ([]PaymentMethodTotal, error) {
    // Query sale_payments grouped by payment_method_code
}
```

**File:** `internal/shift/handler.go`

Add route:
```go
r.GET("/shifts/:id/report", auth, perm(permissions.ShiftView), h.GetShiftReport)
```

Add handler:
```go
func (h *Handler) GetShiftReport(c *gin.Context) {
    // Parse shift ID, fetch report data, return JSON
}
```

### Step 4.3 — Frontend: Z-report component

**New file:** `web/src/modules/shifts/components/ShiftReport.svelte`

Printable layout with:
- Header: "Shift Report" title, cashier name, store name
- Summary grid: opened at, closed at, duration
- Payment breakdown table: method, amount, count
- Cash reconciliation section: opening balance, cash sales, cash drops, paid in/out, expected, actual, variance
- Print button using `window.print()`

Add `@media print` styles for clean printing.

**File:** `web/src/modules/shifts/services/shift-service.ts`

Add:
```typescript
export async function getShiftReport(shiftId: number): Promise<ShiftReportData> { ... }
```

**File:** `web/src/modules/shifts/components/ShiftDetailDrawer.svelte`

Add "Print Report" button that opens `ShiftReport` in a new window or modal.

---

## Phase 5: Polish & Hardening

**Effort:** 1.5-2 days
**Goal:** UX improvements and edge case handling.

### Step 5.1 — Shift duration display

**File:** `internal/shift/domain.go`

Add field:
```go
DurationMinutes *int `json:"duration_minutes,omitempty"`
```

**File:** `internal/shift/repository.go`

In `GetActiveShiftByUserID`, `ListShifts`, `GetShiftByID` — after scanning `opened_at` and `closed_at`, compute:
```go
if closedAt.Valid {
    dur := int(closedAt.Time.Sub(openedAt).Minutes())
    shift.DurationMinutes = &dur
}
```

**Frontend:** Display in shift list table column and detail drawer.

### Step 5.2 — Store selector in open shift modal

**File:** `web/src/modules/shifts/components/ShiftsPage.svelte`

In the open modal (line 416):
1. Fetch active stores on modal open via `GET /api/stores/active`
2. Add a dropdown selector (hidden if only 1 store)
3. Pass selected `store_id` to `doOpenShift()`

### Step 5.3 — ~~`CloseAll` per-shift audit entries~~ REMOVED

> **Status:** Removed — `CloseAll` endpoint deleted in commit `54fa21d`. No longer applicable.

### Step 5.4 — Auto-close abandoned shifts

**New file:** `internal/shift/auto_close.go`

```go
package shift

type AutoCloser struct {
    repo    *Repository
    threshold time.Duration
}

func (a *AutoCloser) Run(ctx context.Context) error {
    // 1. Query: WHERE status = 'open' AND opened_at < NOW() - $1
    // 2. For each: compute expected closing balance, close with notes
    // 3. Create audit entry per shift
    // 4. Return count of auto-closed shifts
}
```

**File:** `cmd/server/main.go`

Register as a background goroutine that runs every hour:
```go
go func() {
    ticker := time.NewTicker(1 * time.Hour)
    for range ticker.C {
        deps.ShiftAutoCloser.Run(context.Background())
    }
}()
```

**Settings:** `shift_auto_close_hours` in `app_settings` (0 = disabled). Add to `041_shift_settings.sql`.

### Step 5.5 — Real-time shift totals

**File:** `web/src/modules/shifts/components/ShiftsPage.svelte`

Add polling for the active shift banner:
```svelte
let pollInterval: ReturnType<typeof setInterval> | null = null;

onMount(() => {
    store.loadActiveShift();
    if (store.activeShift) {
        pollInterval = setInterval(() => store.loadActiveShift(), 30000);
    }
});

$effect(() => {
    if (pollInterval) clearInterval(pollInterval);
    if (store.activeShift) {
        pollInterval = setInterval(() => store.loadActiveShift(), 30000);
    }
});

onDestroy(() => {
    if (pollInterval) clearInterval(pollInterval);
});
```

---

## Files Summary

### New Files

| Phase | File | Purpose |
|-------|------|---------|
| 2 | `database/migrations/040_shift_cash_movements.sql` | Cash movements table + permissions |
| 2 | `internal/shift/cash_movement.go` | Domain, repository, service methods |
| 2 | `internal/shift/cash_movement_test.go` | Tests for cash movement CRUD |
| 2 | `web/src/modules/shifts/components/CashMovementModal.svelte` | Record movement modal |
| 3 | `database/migrations/041_shift_settings.sql` | Discrepancy threshold + blind close settings |
| 4 | `web/src/modules/shifts/components/ShiftReport.svelte` | Printable Z-report |
| 5 | `internal/shift/auto_close.go` | Background auto-close job |

### Modified Files

| Phase | File | Changes |
|-------|------|---------|
| 1 | `internal/shift/repository.go` | ~~CloseAll cart guard + expected balance~~, ReviewShift guard, FlagForReview method |
| 1 | `internal/shift/service.go` | ~~CloseAll return type update~~, FlagForReview |
| 1 | `internal/shift/handler.go` | ~~CloseAll response~~, AuditShift flag, FlagForReview |
| 2 | `internal/shift/repository.go` | Cash movement CRUD, updated close formula |
| 2 | `internal/shift/service.go` | Cash movement service methods, updated interface |
| 2 | `internal/shift/handler.go` | Cash movement endpoints, route registration |
| 2 | `internal/shift/domain.go` | CashMovement type, CashMovementSummary, Shift fields |
| 2 | `internal/permissions/permissions.go` | Add `ShiftCashMovement` constant |
| 2 | `web/src/modules/shifts/services/shift-service.ts` | Cash movement API functions |
| 2 | `web/src/modules/shifts/types/index.ts` | CashMovement type |
| 2 | `web/src/modules/shifts/stores/shift-store.svelte.ts` | Cash movement store methods |
| 2 | `web/src/modules/shifts/components/ShiftsPage.svelte` | Movement button, close modal summary |
| 2 | `web/src/modules/shifts/components/ShiftDetailDrawer.svelte` | Movement list section |
| 3 | `internal/shift/service.go` | SettingsProvider port, threshold helper |
| 3 | `internal/shift/repository.go` | Accept threshold parameter |
| 3 | `internal/wiring/wiring.go` | Wire SettingsProvider |
| 3 | `web/src/modules/admin/components/SettingsPage.svelte` | Shift settings section |
| 3 | `web/src/shared/stores/settings.svelte.ts` | New setting keys |
| 4 | `internal/shift/repository.go` | GetShiftReportData |
| 4 | `internal/shift/domain.go` | ShiftReportData, PaymentMethodTotal |
| 4 | `internal/shift/handler.go` | GetShiftReport endpoint |
| 4 | `internal/sale/shift_summary.go` | PaymentMethodBreakdown |
| 4 | `web/src/modules/shifts/services/shift-service.ts` | getShiftReport |
| 4 | `web/src/modules/shifts/components/ShiftDetailDrawer.svelte` | Print Report button |
| 5 | `internal/shift/domain.go` | DurationMinutes field |
| 5 | `internal/shift/repository.go` | Duration computation |
| 5 | `web/src/modules/shifts/components/ShiftsPage.svelte` | Store selector, duration display, polling |
| 5 | `cmd/server/main.go` | Auto-close goroutine |

---

## Testing Strategy

| Phase | Unit | Integration | E2E |
|-------|------|-------------|-----|
| 1 | `repository_test.go` (~~CloseAll carts~~, ReviewShift guard), `handler_extra_test.go` (audit flag) | — | — |
| 2 | `cash_movement_test.go` (CRUD, validation, ownership) | `repository_test.go` (close formula with movements) | `tests/e2e/shifts.spec.ts` (record movement) |
| 3 | Service threshold test (mock settings) | Verify setting read at close time | Settings page save/load |
| 4 | — | Report endpoint response shape | Print report flow |
| 5 | `auto_close_test.go` (timeout, disabled) | Duration computation | Active shift polling |

---

## Deployment Order

Migrations must be applied **before** deploying the corresponding binary (per AGENTS.md §Migration Ordering):

```
1. Apply migration 040_shift_cash_movements.sql
2. Apply migration 041_shift_settings.sql
3. Deploy binary with Phase 1-5 changes
4. Verify permissions: cashier can record cash movements, manager can review
```
