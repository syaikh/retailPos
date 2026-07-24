# Post-Security Hardening Improvement Plan (Updated)

**Date:** July 15, 2026 (updated July 24, 2026)
**Triggered by:** Code quality review after Phase 1 + Phase 5 execution
**Baseline after hardening:** Security 89/100, Architecture 7.0/10, Performance 6.5/10
**Overall status:** 18/18 ✅ completed, 0/18 ⚠️ partial, 0/18 ❌ remaining

---

## Scope

Phase 1 (Quick Wins) and Phase 5 (Security Hardening) dari `improvement-plan-2026-07-15.md` sudah dieksekusi. Plan ini mencakup **sisa temuan** yang teridentifikasi selalu proses hardening, diorganisir berdasarkan prioritas dan dampak.

---

## Phase A: Performance — Dashboard & Report (2-3 days, HIGH impact)

### A.1 — Consolidate `GetLiveDashboardStats` into single CTE query ✅

**File:** `internal/report/repository.go:259-321`
**Status:** SUDAH — single query with 3 CTEs (today_sales, product_count, stock_count)
**Impact:** 66% reduction in DB round-trips for the most frequently-called dashboard endpoint.

### A.2 — Cache dashboard endpoints with short TTL ✅

**Files:** `internal/report/repository.go`, `internal/wiring/wiring.go`
**Status:** SUDAH — ristretto cache with TTL + jitter, event-based invalidation on `sale.created`
**Impact:** Reduces 3-5 DB queries per dashboard page load to 0 on cache hit.

### A.3 — Replace correlated subquery in `GetSalesForExport` ✅

**File:** `internal/sale/repository.go:284-314`
**Status:** SUDAH — uses LEFT JOIN with derived GROUP BY table, not correlated subquery
**Impact:** Export query goes from O(N) subqueries to O(1) join. Critical for large exports.

### A.4 — Batch stock deduction in `CreateSale` ✅

**File:** `internal/sale/service.go:84-95`
**Status:** SUDAH — single `unnest()` batch UPDATE with FOR UPDATE lock
**Impact:** N round-trips → 1. Reduces lock hold time on `product_stock` rows from O(N) to O(1).

### A.5 — Batch sale items insert in `CreateSale` ✅

**File:** `internal/sale/repository.go:43-78`
**Status:** SUDAH — uses `pgx.CopyFrom` with `pgx.CopyFromRows` for protocol-level bulk insert
**Impact:** N round-trips → 1 for sale item creation.

### A.6 — Add `pg_trgm` GIN indexes for ILIKE searches ✅

**File:** `database/migrations/000_squash.sql`
**Status:** SUDAH — pg_trgm extension enabled, 3 trigram indexes (products.name, sales.invoice_number, customers.name)
**Impact:** ILIKE searches go from sequential scan → index scan. Critical for search-as-you-type features.

---

## Phase B: Architecture Refactoring (2-3 days, MEDIUM impact)

### B.1 — Extract date-range logic from report handler ✅

**Files:** `internal/report/ranges.go` (new), `internal/report/handler.go`
**Status:** SUDAH — all range functions + `PeriodType`/`PeriodRange`/`dateRange` types + `parseDateRange`/`parseDateParam` helpers extracted to `ranges.go`
**Impact:** handler.go shrinks from 713 → 469 lines. Date logic independently testable via existing `handler_helpers_test.go`.

### B.2 — Extract product scan-to-struct helper ✅

**File:** `internal/product/repository.go:48`
**Status:** SUDAH — `scanProduct(row rowScanner)` helper reused by all query methods
**Impact:** Eliminates 50-line duplication. Future schema changes only need 1 update.

### B.3 — Consolidate EventBus interface ✅

**File:** `internal/shared/eventbus.go`
**Status:** SUDAH — single `shared.EventBus` interface, all 4 services import from shared package
**Impact:** DRY violation eliminated. Single source of truth for the event publishing contract.

### B.4 — Remove dead code in product/service.go ✅

**File:** `internal/product/service.go:116-126`
**Problem:** `resolveCategoryID`, `resolveBrandID`, `resolveUnitOfMeasureID` — private methods only called from test code. `strPtr` duplicated in `adapter.go`.
**Fix:** Delete the 3 `resolve*` methods. Move `strPtr`/`intPtr` to a shared `ptr` utility or inline in `adapter.go`.
**Impact:** ~30 lines of dead code removed. Reduces cognitive load.

### B.5 — Consolidate timezone initialization ✅

**Files:** `pkg/websocket/hub.go:19-25`, `internal/shared/timezone.go`, `internal/report/handler.go` (uses `config.Load().Timezone`)
**Problem:** 3 separate timezone initialization mechanisms:
- `websocket/hub.go`: uncached `time.LoadLocation` per call
- `shared/timezone.go`: cached singleton via `init()`
- `report/handler.go`: via `config.Load().Timezone`

**Fix:** Replace `getJakartaLoc()` in `websocket/hub.go` with `shared.JakartaLocation()`. Replace `config.Load().Timezone` references in report handler with `shared.JakartaLocation()`. Single source of truth.
**Impact:** Eliminates timezone duplication. Reduces risk of timezone inconsistencies.

### B.6 — Add `storeIDPtr` helper to shared package ✅

**File:** `internal/shared/context.go`
**Problem:** Report handler has 7 instances of `storeID := shared.GetStoreID(c); sid := 0; if storeID != nil { sid = *storeID }` — the nil-to-zero coercion pattern.
**Fix:** Add `GetStoreIDInt(c *gin.Context) int` helper that returns 0 when nil.
**Impact:** ~20 lines of boilerplate removed from report handler.

---

## Phase C: Performance — Deep Optimizations (3-4 days, MEDIUM impact)

### C.1 — Optimize `GetPeriodComparison` from 6 CTEs to 2 ✅

**File:** `internal/report/repository.go:22-135`
**Problem:** 6 independent CTEs scan the `sales` table 6 times. `AT TIME ZONE` per-row prevents index-only scans for peak-hour/month aggregations. `(store_id = $5 OR $5 IS NULL)` defeats index filtering.
**Fix:**
- Replace with 2 base CTEs (current_period, previous_period) that aggregate revenue, orders, AND hourly/monthly breakdowns in a single scan each using `GROUP BY ROLLUP` or conditional aggregation.
- Replace `OR $5 IS NULL` with dynamic query construction (inject `AND store_id = $N` only when storeID is non-nil).
**Impact:** 6 table scans → 2. Peak-hour/month computed from the same scan as revenue/orders.

### C.2 — Streaming exports with per-row flush ✅

**Files:** `internal/sale/repository.go` (`GetSalesForExport`), `internal/sale/export.go`
**Problem:** All rows loaded into `[]SaleExportRow` in memory before writing. XLSX uses `excelize` which builds entire workbook in RAM — double-buffering.
**Fix:**
- Added `StreamSalesExportCSV` method to `Repository` — writes CSV directly to `io.Writer` with per-row flush, never buffering all rows in memory.
- Added matching `StreamSalesExportCSV` to `SaleService` interface and `Service` implementation.
- Handler CSV path now calls `StreamSalesExportCSV` instead of `GetSalesForExport` + `WriteCSV`.
- XLSX path still uses in-memory `GetSalesForExport` + `WriteXLSX` (excelize streaming writer would require v2 API redesign).
**Impact:** Memory usage drops from O(N) to O(1) for CSV exports. Prevents OOM on large datasets. XLSX still O(N).

### C.3 — Cache report queries with short TTL ✅

**Files:** `internal/report/repository.go`
**Problem:** `GetPeriodComparison` and `GetDualChartData` are expensive CTE queries executed on every page load.
**Fix:** Cache results with 30s TTL + jitter. Invalidate on `sale.created` events.
**Impact:** Reduces expensive CTE execution frequency for report page loads.

### C.4 — Add composite index for sales aggregation ✅

**File:** `database/migrations/` (new migration)
**Problem:** Dashboard and report queries filter on `(status, created_at, store_id)` and aggregate `total_amount`. Current indexes may not cover this pattern.
**Fix:**
- Already existed in `database/migrations/000_squash.sql:463` — `idx_sales_status_created_store ON sales (status, created_at, store_id) INCLUDE (total_amount)`
- No new migration needed; index is present and active.
**Impact:** Covering index enables index-only scans for dashboard aggregation queries.

---

## Phase D: Config & Testability (1 day, LOW-MEDIUM impact)

### D.1 — Replace `config.Load()` singleton with dependency injection ✅

**File:** `internal/user/auth_service.go`, `internal/config/config.go`
**Problem:** `config.Load()` is a global singleton. `auth_service.go` calls it directly, making it impossible to inject different config per test without `os.Setenv`.
**Fix:** Added `cfg *config.Config` parameter to `NewAuthService(repo, auditSvc, cfg)`. Stored in `AuthService` struct. Updated wiring and test callers.
**Impact:** Better testability. Eliminates hidden global state dependency.

### D.2 — Add comprehensive integration tests for auth flows ✅

**File:** `internal/user/auth_service_test.go`
**Problem:** Existing tests use mock repos. No tests verify the full login → refresh → change-password → logout flow with real DB.
**Fix:** Added `TestAuthService_FullLifecycle` — exercises the complete auth lifecycle against the test database (login → validate → refresh → change password → logout).
**Impact:** Catches regressions in auth security (e.g., token rotation, refresh token invalidation).

---

## Execution Order

| Order | Phase | Effort | Impact | Risk | Status |
|-------|-------|--------|--------|------|--------|
| 1 | A.1 (consolidate live dashboard) | 1 hr | High | Low | ✅ |
| 2 | A.2 (cache dashboard) | 2 hrs | High | Low | ✅ |
| 3 | A.3 (fix correlated subquery) | 1 hr | High | Low | ✅ |
| 4 | A.4 (batch stock deduction) | 2 hrs | High | Medium | ✅ |
| 5 | A.5 (batch sale items insert) | 1 hr | Medium | Low | ✅ |
| 6 | B.1 (extract date ranges) | 2 hrs | Medium | Low | ✅ |
| 7 | B.2 (extract product scan) | 1 hr | Medium | Low | ✅ |
| 8 | B.3 (consolidate EventBus) | 30 min | Medium | Low | ✅ |
| 9 | B.5 (consolidate timezone) | 15 min | Low | Low | ✅ |
| 10 | B.6 (storeID helper) | 15 min | Low | Low | ✅ |
| 11 | A.6 (pg_trgm indexes) | 30 min | High | Low | ✅ |
| 12 | C.1 (optimize period comparison) | 3 hrs | High | Medium | ✅ |
| 13 | C.4 (composite index) | 15 min | Medium | Low | ✅ |
| 14 | B.4 (dead code removal) | 15 min | Low | Low | ✅ |
| 15 | C.3 (cache report queries) | 2 hrs | Medium | Low | ✅ |
| 16 | D.1 (config DI) | 1 hr | Low | Low | ✅ |
| 17 | D.2 (auth integration tests) | 2 hrs | Medium | Low | ✅ |
| 18 | C.2 (streaming exports) | 3-4 hrs | High | High | ✅ |

**Completed:** 18 ✅, 0 ⚠️ partial, 0 ❌ remaining
**Remaining estimated:** 0 hours — all items completed

---

## Current Scores (after Phase A + Phase B completed)

| Domain | After Hardening | Current |
|--------|-----------------|---------|
| Security | 89/100 | 89/100 (no change) |
| Architecture | 7.0/10 | **7.5/10** |
| Performance | 6.5/10 | **7.5/10** |
| Test Coverage | 90.6% | **90.6%** |

---

## Dependency Graph

```
✅ A.1 (consolidate live dashboard)
 └─→ ✅ A.2 (cache dashboard) — depends on A.1 for single-query pattern

✅ A.3 (fix correlated subquery)
 └─→ C.2 (streaming exports) — done, uses per-row csv.Writer flush

✅ A.4 (batch stock deduction)
 └─→ independent

✅ A.5 (batch sale items insert — CopyFrom)
 └─→ independent

✅ B.1 (extract date ranges)
 └─→ independent

✅ B.2 (extract product scan)
 └─→ independent

✅ B.3 (consolidate EventBus)
 └─→ independent

✅ C.1 (optimize period comparison)
 └─→ ✅ C.3 (cache report queries) — cached after query was optimized

✅ C.4 (composite index)
 └─→ independent — already existed in 000_squash.sql
```

## Risk Assessment (all items completed)

| Change | Risk | Mitigation | Outcome |
|--------|------|------------|---------|
| C.1 (optimize period comparison) | Medium — complex SQL rewrite | Verified with existing period comparison tests | ✅ All tests pass |
| C.2 (streaming exports) | High — changes API contract | Added new streaming method, kept old method for XLSX compat | ✅ CSV now O(1) memory, XLSX unchanged |
| D.1 (config DI) | Low — constructor signature change | Updated all callers | ✅ All callers updated |
