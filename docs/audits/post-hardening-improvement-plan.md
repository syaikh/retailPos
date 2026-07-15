# Post-Security Hardening Improvement Plan

**Date:** July 15, 2026
**Triggered by:** Code quality review after Phase 1 + Phase 5 execution
**Baseline after hardening:** Security 89/100, Architecture 7.0/10, Performance 6.5/10

---

## Scope

Phase 1 (Quick Wins) and Phase 5 (Security Hardening) dari `improvement-plan-2026-07-15.md` sudah dieksekusi. Plan ini mencakup **sisa temuan** yang teridentifikasi selalu proses hardening, diorganisir berdasarkan prioritas dan dampak.

---

## Phase A: Performance — Dashboard & Report (2-3 days, HIGH impact)

### A.1 — Consolidate `GetLiveDashboardStats` into single CTE query

**File:** `internal/report/repository.go:209-255`
**Problem:** 3 DB round-trips sequential queries (today's sales, product count, low stock). `GetDashboardStats` does the same work in 1 CTE query.
**Fix:** Rewrite `GetLiveDashboardStats` to use the same CTE pattern as `GetDashboardStats`. 3 queries → 1.
**Impact:** 66% reduction in DB round-trips for the most frequently-called dashboard endpoint.

### A.2 — Cache dashboard endpoints with short TTL

**Files:** `internal/report/repository.go`, `cmd/server/main.go`
**Problem:** `report.Repository` has zero caching (unlike product, user, category repos). Dashboard stats (product count, low stock count) change infrequently.
**Fix:** Add `cache *cache.Cache` field + `SetCache()` to `report.Repository`. Cache `GetDashboardStats` and `GetLiveDashboardStats` with 10-15s TTL + jitter. WebSocket `sale_created` events bypass cache and trigger invalidation.
**Impact:** Reduces 3-5 DB queries per dashboard page load to 0 on cache hit.

### A.3 — Replace correlated subquery in `GetSalesForExport`

**File:** `internal/sale/repository.go:309-314`
**Problem:** `(SELECT COUNT(*) FROM sale_items si WHERE si.sale_id = s.id)` executes once per sales row — O(N) subqueries for N sales.
**Fix:** Replace with a `LEFT JOIN (SELECT sale_id, COUNT(*) as cnt FROM sale_items GROUP BY sale_id) items ON items.sale_id = s.id`.
**Impact:** Export query goes from O(N) subqueries to O(1) join. Critical for large exports.

### A.4 — Batch stock deduction in `CreateSale`

**File:** `internal/sale/service.go:82-87`
**Problem:** N individual `UPDATE product_stock SET quantity = quantity - $1 WHERE product_id = $2` statements in a loop. Each is a separate DB round-trip inside a `FOR UPDATE` transaction.
**Fix:** Single batch UPDATE using `unnest()`:
```sql
UPDATE product_stock ps
SET quantity = ps.quantity - v.qty
FROM (SELECT unnest($1::int[]) AS product_id, unnest($2::int[]) AS qty) v
WHERE ps.product_id = v.product_id
  AND ps.warehouse_id IS NULL AND ps.store_id IS NULL
```
**Impact:** N round-trips → 1. Reduces lock hold time on `product_stock` rows from O(N) to O(1).

### A.5 — Batch sale items insert in `CreateSale`

**File:** `internal/sale/repository.go:28-53`
**Problem:** N individual `INSERT INTO sale_items` statements in a loop.
**Fix:** Use `pgx.CopyFrom` with `pgx.CopyFromRows` to batch-insert all items in a single round-trip.
**Impact:** N round-trips → 1 for sale item creation.

### A.6 — Add `pg_trgm` GIN indexes for ILIKE searches

**File:** `database/migrations/` (new migration)
**Problem:** ILIKE searches on `audit_logs(username, role, action)`, `customers(name, phone, email)`, `users(username, email)` use sequential scans.
**Fix:** Create migration with `CREATE INDEX` using `gin_trgm_ops`:
```sql
CREATE INDEX idx_audit_logs_username_trgm ON audit_logs USING gin (username gin_trgm_ops);
CREATE INDEX idx_customers_name_trgm ON customers USING gin (name gin_trgm_ops);
CREATE INDEX idx_customers_phone_trgm ON customers USING gin (phone gin_trgm_ops);
CREATE INDEX idx_users_username_trgm ON users USING gin (username gin_trgm_ops);
```
**Impact:** ILIKE searches go from sequential scan → index scan. Critical for search-as-you-type features.

---

## Phase B: Architecture Refactoring (2-3 days, MEDIUM impact)

### B.1 — Extract date-range logic from report handler

**File:** `internal/report/handler.go`
**Problem:** 184 lines of date-range calculation functions (lines 69-260) embedded in a 697-line handler. Same date-parsing boilerplate copy-pasted 4 times across handler methods.
**Fix:**
- Extract all `get*Ranges` functions + `isPeriodIncomplete` into `internal/report/ranges.go` (new file, ~184 lines).
- Create helper `parseDateRange(c *gin.Context, defaultOffsetDays int) (start, end time.Time, ok bool)` to eliminate duplicated date parsing boilerplate (~80 lines removed from handler).
**Impact:** `handler.go` shrinks from 697 → ~440 lines. Date logic becomes independently testable.

### B.2 — Extract product scan-to-struct helper

**File:** `internal/product/repository.go:82-246`
**Problem:** `GetProductByID` (lines 82-163) and `GetProductBySKU` (lines 165-246) have ~50 lines of identical scan-to-struct logic. Only the WHERE clause differs.
**Fix:** Extract shared scan logic into `scanProduct(row pgx.Row) (*Product, error)` helper. Both functions call the helper with their respective query.
**Impact:** Eliminates 50-line duplication. Future schema changes only need 1 update.

### B.3 — Consolidate EventBus interface

**Files:** `internal/product/service.go:10-12`, `internal/sale/service.go:13-15`, `internal/report/service.go:10-12`, `internal/inventory/service.go:11-13`
**Problem:** 4 identical `EventBus` interface definitions across 4 packages.
**Fix:** Define the interface once in `internal/eventbus/bus.go`:
```go
type Publisher interface {
    Publish(ctx context.Context, topic string, event interface{}) error
}
```
All consumer packages import `eventbus.Publisher` instead of defining their own.
**Impact:** DRY violation eliminated. Single source of truth for the event publishing contract.

### B.4 — Remove dead code in product/service.go

**File:** `internal/product/service.go:116-126`
**Problem:** `resolveCategoryID`, `resolveBrandID`, `resolveUnitOfMeasureID` — private methods only called from test code. `strPtr` duplicated in `adapter.go`.
**Fix:** Delete the 3 `resolve*` methods. Move `strPtr`/`intPtr` to a shared `ptr` utility or inline in `adapter.go`.
**Impact:** ~30 lines of dead code removed. Reduces cognitive load.

### B.5 — Consolidate timezone initialization

**Files:** `pkg/websocket/hub.go:19-25`, `internal/shared/timezone.go`, `internal/report/handler.go` (uses `config.Load().Timezone`)
**Problem:** 3 separate timezone initialization mechanisms:
- `websocket/hub.go`: uncached `time.LoadLocation` per call
- `shared/timezone.go`: cached singleton via `init()`
- `report/handler.go`: via `config.Load().Timezone`

**Fix:** Replace `getJakartaLoc()` in `websocket/hub.go` with `shared.JakartaLocation()`. Replace `config.Load().Timezone` references in report handler with `shared.JakartaLocation()`. Single source of truth.
**Impact:** Eliminates timezone duplication. Reduces risk of timezone inconsistencies.

### B.6 — Add `storeIDPtr` helper to shared package

**File:** `internal/shared/context.go`
**Problem:** Report handler has 7 instances of `storeID := shared.GetStoreID(c); sid := 0; if storeID != nil { sid = *storeID }` — the nil-to-zero coercion pattern.
**Fix:** Add `GetStoreIDInt(c *gin.Context) int` helper that returns 0 when nil.
**Impact:** ~20 lines of boilerplate removed from report handler.

---

## Phase C: Performance — Deep Optimizations (3-4 days, MEDIUM impact)

### C.1 — Optimize `GetPeriodComparison` from 6 CTEs to 2

**File:** `internal/report/repository.go:22-135`
**Problem:** 6 independent CTEs scan the `sales` table 6 times. `AT TIME ZONE` per-row prevents index-only scans for peak-hour/month aggregations. `(store_id = $5 OR $5 IS NULL)` defeats index filtering.
**Fix:**
- Replace with 2 base CTEs (current_period, previous_period) that aggregate revenue, orders, AND hourly/monthly breakdowns in a single scan each using `GROUP BY ROLLUP` or conditional aggregation.
- Replace `OR $5 IS NULL` with dynamic query construction (inject `AND store_id = $N` only when storeID is non-nil).
**Impact:** 6 table scans → 2. Peak-hour/month computed from the same scan as revenue/orders.

### C.2 — Streaming exports with per-row flush

**Files:** `internal/sale/repository.go` (`GetSalesForExport`), `internal/sale/export.go`
**Problem:** All rows loaded into `[]SaleExportRow` in memory before writing. XLSX uses `excelize` which builds entire workbook in RAM — double-buffering.
**Fix:**
- Change `GetSalesForExport` to return `pgx.Rows` cursor instead of `[]SaleExportRow`.
- Stream CSV rows directly from cursor to `io.Writer` with per-row flush.
- For XLSX, use `excelize` streaming writer mode or switch to a streaming XLSX library.
- Increase `WriteTimeout` to 300s for large exports (or implement background export with job queue).
**Impact:** Memory usage drops from O(N) to O(1) for exports. Prevents OOM on large datasets.

### C.3 — Cache report queries with short TTL

**Files:** `internal/report/repository.go`
**Problem:** `GetPeriodComparison` and `GetDualChartData` are expensive CTE queries executed on every page load.
**Fix:** Cache results with 30s TTL + jitter. Invalidate on `sale.created` events.
**Impact:** Reduces expensive CTE execution frequency for report page loads.

### C.4 — Add composite index for sales aggregation

**File:** `database/migrations/` (new migration)
**Problem:** Dashboard and report queries filter on `(status, created_at, store_id)` and aggregate `total_amount`. Current indexes may not cover this pattern.
**Fix:**
```sql
CREATE INDEX idx_sales_dashboard ON sales (status, created_at, store_id)
    INCLUDE (total_amount, customer_id, payment_method);
```
**Impact:** Covering index enables index-only scans for dashboard aggregation queries.

---

## Phase D: Config & Testability (1 day, LOW-MEDIUM impact)

### D.1 — Replace `config.Load()` singleton with dependency injection

**File:** `internal/user/auth_service.go`, `internal/config/config.go`
**Problem:** `config.Load()` is a global singleton. `auth_service.go` calls it directly, making it impossible to inject different config per test without `os.Setenv`.
**Fix:** Add `cfg *config.Config` parameter to `NewAuthService(repo, cfg)`. Store in `AuthService` struct. Same pattern for any future service that needs config.
**Impact:** Better testability. Eliminates hidden global state dependency.

### D.2 — Add comprehensive integration tests for auth flows

**File:** `internal/user/auth_service_test.go`
**Problem:** Existing tests use mock repos. No tests verify the full login → refresh → change-password → logout flow with real DB.
**Fix:** Add integration test that exercises the complete auth lifecycle against the test database.
**Impact:** Catches regressions in auth security (e.g., token rotation, refresh token invalidation).

---

## Execution Order

| Order | Phase | Effort | Impact | Risk |
|-------|-------|--------|--------|------|
| 1 | A.1 (consolidate live dashboard) | 1 hr | High | Low |
| 2 | A.2 (cache dashboard) | 2 hrs | High | Low |
| 3 | A.3 (fix correlated subquery) | 1 hr | High | Low |
| 4 | A.4 (batch stock deduction) | 2 hrs | High | Medium |
| 5 | A.5 (batch sale items insert) | 1 hr | Medium | Low |
| 6 | B.1 (extract date ranges) | 2 hrs | Medium | Low |
| 7 | B.2 (extract product scan) | 1 hr | Medium | Low |
| 8 | B.3 (consolidate EventBus) | 30 min | Medium | Low |
| 9 | B.5 (consolidate timezone) | 15 min | Low | Low |
| 10 | B.6 (storeID helper) | 15 min | Low | Low |
| 11 | A.6 (pg_trgm indexes) | 30 min | High | Low |
| 12 | C.1 (optimize period comparison) | 3 hrs | High | Medium |
| 13 | C.4 (composite index) | 15 min | Medium | Low |
| 14 | B.4 (dead code removal) | 15 min | Low | Low |
| 15 | C.3 (cache report queries) | 2 hrs | Medium | Low |
| 16 | D.1 (config DI) | 1 hr | Low | Low |
| 17 | D.2 (auth integration tests) | 2 hrs | Medium | Low |
| 18 | C.2 (streaming exports) | 3-4 hrs | High | High |

**Total estimated:** 18-22 hours (~3 days)

---

## Projected Scores After Execution

| Domain | After Hardening | After This Plan |
|--------|-----------------|-----------------|
| Security | 89/100 | 89/100 (no change) |
| Architecture | 7.0/10 | **7.5-8.0/10** |
| Performance | 6.5/10 | **7.5/10** |
| Test Coverage | 90.6% | **91.5%+** |

---

## Dependency Graph

```
A.1 (consolidate live dashboard)
 └─→ A.2 (cache dashboard) — depends on A.1 for single-query pattern

A.3 (fix correlated subquery)
 └─→ C.2 (streaming exports) — depends on A.3 for clean cursor

A.4 (batch stock deduction)
 └─→ independent

A.5 (batch sale items insert)
 └─→ independent

B.1 (extract date ranges)
 └─→ independent

B.2 (extract product scan)
 └─→ independent

B.3 (consolidate EventBus)
 └─→ independent

C.1 (optimize period comparison)
 └─→ C.3 (cache report queries) — cache after query is optimized

C.4 (composite index)
 └─→ independent, can run anytime
```

## Risk Assessment

| Change | Risk | Mitigation |
|--------|------|------------|
| A.4 (batch stock deduction) | Medium — changes concurrency behavior | Keep FOR UPDATE lock, test with concurrent sale scenarios |
| C.1 (optimize period comparison) | Medium — complex SQL rewrite | Verify with existing period comparison tests, compare results |
| C.2 (streaming exports) | High — changes API contract | Maintain backward compatibility, add streaming as opt-in |
| D.1 (config DI) | Low — constructor signature change | Update all callers (only `main.go`) |
