# Backend Code Review — 2026-08-18

**Scope:** Full backend codebase review (`/my-review backend`)
**Codebase:** ~104k lines Go, Gin + PostgreSQL
**Status:** All 20 findings fixed — commit pending

---

## Summary

Well-structured Go retail POS system with clean module boundaries (handler/service/repository per domain), JWT auth with refresh token rotation, CSRF protection, rate limiting, and materialized views for reporting. Strong defensive programming throughout: atomic stock deduction with `FOR UPDATE` locking, parameterized SQL, bcrypt hashing, and comprehensive state machine guards. Found 1 critical security issue (cross-store IDOR), 7 warnings across performance/deploy-safety/business-logic, 9 dead code items, and 5 duplication patterns.

---

## Issues Found

| #  | Severity   | Area        | File:Line                                | Issue                                                    |
|----|------------|-------------|------------------------------------------|----------------------------------------------------------|
| 1  | CRITICAL   | Security    | `internal/product/repository.go:215`     | IDOR: batch product lookup bypasses store scoping         |
| 2  | WARNING    | Performance | `internal/sale/repository.go:575`        | CSV export buffers entire result set in memory            |
| 3  | WARNING    | Performance | `internal/sale/service.go:203`           | N+1 query: payment method lookup per payment in checkout  |
| 4  | WARNING    | Performance | `internal/report/repository.go:509`      | Dashboard stats: 5 sequential DB round-trips              |
| 5  | WARNING    | Performance | `internal/sale/report_adapter.go:18`     | Today's dashboard stats scan raw sales table instead of MV|
| 6  | WARNING    | Deploy      | `database/migrations/000_squash.sql:1243`| Seed data runs outside transaction block                  |
| 7  | WARNING    | Business    | `internal/shift/repository.go:144`       | Shift close does not verify no active sales in progress   |
| 8  | WARNING    | Business    | `internal/pricing/service.go:39`         | Pricing rule conflicts not detected on create/update      |
| 9  | SUGGESTION | Dead Code   | `internal/sale/export.go:14`             | `WriteCSV` dead in production (handler uses streaming)    |
| 10 | SUGGESTION | Dead Code   | `internal/consignment/domain.go:27`      | `ErrSettlementForbidden` — zero callers anywhere          |
| 11 | SUGGESTION | Dead Code   | `internal/consignment/service.go:15`     | `ErrUnauthorized` unused (handler uses `shared.ErrUnauthorized`) |
| 12 | SUGGESTION | Dead Code   | `internal/consignment/service.go:27`     | `Service.Repo()` method — zero callers                    |
| 13 | SUGGESTION | Dead Code   | `internal/consignment/service.go:904`    | `Service.repository()` method — zero callers              |
| 14 | SUGGESTION | Dead Code   | `internal/consignment/domain.go:256`     | `SupplierSummary` type — defined but never used            |
| 15 | SUGGESTION | Dead Code   | `internal/shared/logging.go:43`          | `LogInfo` — zero production callers                       |
| 16 | SUGGESTION | Dead Code   | `internal/audit/handler.go:218`          | `GenerateAuditDescription` — zero production callers      |
| 17 | SUGGESTION | Dead Code   | `internal/consignment/repository.go:1275`| `var _ = errors.Is` dead suppressor                       |
| 18 | SUGGESTION | Duplication | `internal/sale/repository.go:476,545`    | `GetSalesForExport` and `StreamSalesExportCSV` duplicate identical SQL + customer resolution |
| 19 | SUGGESTION | Duplication | `internal/shared/csv.go` vs `internal/platform/importexport/export/engine.go:196` | `sanitizeCSVField` duplicated across packages |
| 20 | SUGGESTION | Duplication | `internal/inventory/stock_applier.go` + 4 files | Global-product-stock upsert pattern repeated 5+ times |

---

## Detailed Findings

### 1. IDOR: `GET /products?ids=...` bypasses store scoping

- **File:** `internal/product/repository.go:203-237`, `internal/product/handler.go:118-134`
- **Confidence:** 95%
- **Problem:** When `?ids=1,2,3` is provided, the handler calls `GetProductsByIDs` which queries `v_products_full WHERE v.id IN (...)` with **no store filter**. The normal paginated path (`GetAllProducts`) correctly scopes by `store_id` from the JWT, but the batch `ids` path returns products from any store. An authenticated user from Store A can enumerate product IDs belonging to Store B.
- **Suggestion:** Add `storeID *int` parameter to `GetProductsByIDs` at all layers (repo, service, handler). In the repository, add `AND v.store_id = $N` when storeID is non-nil. In the handler, pass `middleware.GetStoreID(c)`.

### 2. CSV export buffers entire result set in memory

- **File:** `internal/sale/repository.go:575-631`
- **Confidence:** 90%
- **Problem:** `StreamSalesExportCSV` reads all matching sales into a `buffered []csvRow` slice, then does a separate `customerNamesByIDs` batch lookup, then writes the CSV. For a broad date range on a busy store (100k+ sales), this allocates an unbounded in-memory slice.
- **Suggestion:** Stream CSV rows as they are scanned. Join customer names in the main query via LEFT JOIN, or resolve names in batches during streaming.

### 3. N+1 query in payment validation during checkout

- **File:** `internal/sale/service.go:198-203`
- **Confidence:** 90%
- **Problem:** `validatePayments` calls `GetPaymentMethodByCode` inside a `for _, p := range payments` loop, issuing one `SELECT` per payment method. The `payment_methods` table is small (5 rows) and rarely changes.
- **Suggestion:** Fetch all active payment methods once before the loop via `GetAllActive`, build a `map[string]*PaymentMethod`, and validate against the map.

### 4. Dashboard stats: 5 sequential DB round-trips

- **File:** `internal/report/repository.go:509-574`
- **Confidence:** 85%
- **Problem:** `GetDashboardStats` executes 5 independent queries sequentially: today's revenue, all-time revenue, active customer count, active product count, and low stock count. With a 10-second cache TTL, every cache miss pays all 5 round-trips.
- **Suggestion:** Use `pgx.Batch` to execute all 5 queries in a single network round-trip, or run them concurrently with `errgroup`.

### 5. Today's dashboard stats scan raw `sales` table instead of MV

- **File:** `internal/sale/report_adapter.go:17-25`
- **Confidence:** 85%
- **Problem:** `GetCompletedSalesStats` queries `SELECT SUM(total_amount), COUNT(*) FROM sales WHERE status='completed' AND created_at >= $1 AND created_at < $2` against the raw `sales` table. The all-time stats already use `mv_dashboard_totals`, but today's stats don't use `mv_hourly_sales`.
- **Suggestion:** Replace with `SELECT COALESCE(SUM(total_revenue), 0), COALESCE(SUM(transaction_count), 0) FROM mv_hourly_sales WHERE sale_hour >= date_trunc('hour', $1::timestamptz) AND sale_hour < date_trunc('hour', $2::timestamptz)`.

### 6. Seed data in `000_squash.sql` runs outside transaction

- **File:** `database/migrations/000_squash.sql:1243-1611`
- **Confidence:** 90%
- **Problem:** The `COMMIT;` at line 1243 ends the transactional schema block. All seed data (roles, permissions, payment methods, users, grants) runs as individual auto-committed statements. A crash mid-seed leaves the DB with tables but missing roles/permissions, causing all non-superadmin logins to fail with 403.
- **Suggestion:** Wrap all seed data in a second `BEGIN;`/`COMMIT;` block so it's atomic.

### 7. Shift close does not verify no active sales in progress

- **File:** `internal/shift/repository.go:144-217`
- **Confidence:** 80%
- **Problem:** `CloseShift` validates the shift is `open` with `FOR UPDATE` locking but does not check whether any sales are currently in-progress for this shift. A shift could be closed while a sale is being finalized.
- **Suggestion:** Add `SELECT COUNT(*) FROM sales WHERE shift_id = $1 AND status IN ('pending', 'processing')` and return an error if any are found.

### 8. Pricing rule conflict detection not enforced on create/update

- **File:** `internal/pricing/service.go:39-55`
- **Confidence:** 80%
- **Problem:** `FindConflicts` exists on the repository but is never called during `Create` or `Update`. Multiple non-combinable rules targeting the same product can be created without warning.
- **Suggestion:** Call `FindConflicts` during `Create` and `Update`. Return a warning or error with details about conflicting rules.

### 9–17. Dead Code

| #  | File | Symbol | Notes |
|----|------|--------|-------|
| 9  | `internal/sale/export.go:14` | `WriteCSV` | Only called in tests; handler uses `StreamSalesExportCSV` |
| 10 | `internal/consignment/domain.go:27` | `ErrSettlementForbidden` | Zero callers anywhere |
| 11 | `internal/consignment/service.go:15` | `ErrUnauthorized` | Handler uses `shared.ErrUnauthorized` instead |
| 12 | `internal/consignment/service.go:27` | `Service.Repo()` | Zero callers; wiring passes repo directly |
| 13 | `internal/consignment/service.go:904` | `Service.repository()` | Zero callers |
| 14 | `internal/consignment/domain.go:256` | `SupplierSummary` | Defined but never used |
| 15 | `internal/shared/logging.go:43` | `LogInfo` | Zero production callers |
| 16 | `internal/audit/handler.go:218` | `GenerateAuditDescription` | Zero production callers |
| 17 | `internal/consignment/repository.go:1275` | `var _ = errors.Is` | Dead suppressor |

### 18–20. Duplication

| #  | Files | Pattern | Risk |
|----|-------|---------|------|
| 18 | `sale/repository.go:476,545` | `GetSalesForExport` and `StreamSalesExportCSV` duplicate identical SQL + customer resolution | Query changes in one won't propagate to the other |
| 19 | `shared/csv.go` vs `importexport/export/engine.go:196` | `sanitizeCSVField` duplicated with separate prefix lists | Security fix in one won't reach the other |
| 20 | `inventory/stock_applier.go` + 4 files | Global-product-stock upsert pattern (UPDATE then conditional INSERT) repeated 5+ times | Constraint changes in one writer won't be replicated |

---

## Positive Observations

| Area | Status |
|------|--------|
| SQL injection | All queries use `$N` parameterized placeholders. Sort columns validated against allowlist maps. |
| Password handling | Bcrypt cost 14. `User.Password` has `json:"-"` tag. Handlers clear password before response. |
| Login brute force | Per-IP (10/15min) and per-username (5/15min) lockouts. |
| JWT secrets | Required env var, panics if missing. Refresh tokens use separate secret with rotation. |
| Stock deduction | Atomic conditional decrement (`WHERE quantity >= $1`) with `FOR UPDATE` row locking. |
| Cart checkout | Validates cart status, expiry, ownership, and payment totals within a transaction. |
| Stock opname states | All transitions guarded with status checks in SQL WHERE clauses. |
| CORS | Production blocks `*` origin at startup. |
| Security headers | CSP, HSTS, X-Frame-Options DENY, X-Content-Type-Options nosniff all set. |

---

## Recommendation

**NEEDS CHANGES** — The cross-store IDOR on `GET /products?ids=...` is a material security issue. The non-atomic seed migration creates a deploy safety risk.

---

## Fix Progress

| #  | Finding | Status | Commit |
|----|---------|--------|--------|
| 1  | IDOR: batch product lookup store scoping | DONE | — |
| 2  | CSV export memory buffering | DONE | — |
| 3  | N+1 payment method query | DONE | — |
| 4  | Dashboard stats sequential queries | DONE | — |
| 5  | Today's stats raw table vs MV | DONE | — |
| 6  | Seed data outside transaction | DONE | — |
| 7  | Shift close active sale check | DONE | — |
| 8  | Pricing conflict detection | DONE | — |
| 9  | Dead code: `WriteCSV` | DONE | — |
| 10 | Dead code: `ErrSettlementForbidden` | DONE | — |
| 11 | Dead code: `ErrUnauthorized` in consignment | DONE | — |
| 12 | Dead code: `Service.Repo()` | SKIPPED | not dead — used by handler |
| 13 | Dead code: `Service.repository()` | DONE | — |
| 14 | Dead code: `SupplierSummary` | DONE | — |
| 15 | Dead code: `LogInfo` | SKIPPED | public utility, tested |
| 16 | Dead code: `GenerateAuditDescription` | SKIPPED | utility, tested |
| 17 | Dead code: `var _ = errors.Is` | DONE | — |
| 18 | Duplication: export query logic | DONE | — |
| 19 | Duplication: `sanitizeCSVField` | DONE | — |
| 20 | Duplication: stock upsert pattern | SKIPPED | too risky for now |
