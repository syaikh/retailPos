# Test Coverage Audit — 70.6% Milestone

**Date:** 2026-07-14
**Baseline:** 53.2% (starting point)
**Current:** 70.6% (+17.4pp)
**Target:** 80%

---

## Coverage Snapshot by Package

| Package | Coverage | Status |
|---------|----------|--------|
| `cache` | 100.0% | — |
| `shared/importexport` | 100.0% | — |
| `platform/importexport` | 100.0% | — |
| `platform/importexport/schema` | 100.0% | — |
| `platform/importexport/validation` | 100.0% | — |
| `platform/importexport/export` | 94.3% | — |
| `config` | 89.7% | — |
| `websocket` | 88.2% | — |
| `platform/importexport/validation/validators` | 85.1% | — |
| `platform/importexport/import` | 83.9% | — |
| `audit` | 83.5% | — |
| `platform/importexport/template` | 83.3% | — |
| `inventory` | 81.4% | — |
| `sale` | 78.7% | — |
| `platform/importexport/handler` | 76.2% | — |
| `report` | 75.4% | — |
| `brand` | 71.9% | — |
| `user` | 71.0% | — |
| `middleware` | 70.8% | — |
| `shared` | 69.9% | Below target |
| `customer` | 68.9% | Below target |
| `uom` | 67.6% | Below target |
| `eventbus` | 61.1% | Below target |
| `product` | 59.4% | Below target |
| `category` | 59.3% | Below target |
| `progress` | 53.3% | Below target |

---

## Per-Package Gap Analysis

### `internal/eventbus` (61.1%)

**Uncovered functions:**
| Function | Line | Uncovered Lines | Testable? |
|----------|------|-----------------|-----------|
| `Metrics()` | bus.go:217 | ~4 stmts | Pure |
| `DroppedCount()` | bus.go:245 | ~2 stmts | Pure |
| `SetDeadLetterStore()` | bus.go:78 | ~1 stmt | Pure |
| `deadLetter()` | bus.go:190 | ~8 stmts | Mock DeadLetterStore |
| `dispatch` retry/error | bus.go:149-184 | ~10 stmts | Pure (mock listener) |
| `Shutdown` double-call | bus.go:233 | ~2 stmts | Pure |
| `PgDeadLetterStore.*` | bus.go:38-49 | ~10 stmts | DB-dependent |

**Estimated recoverable:** ~27 stmts (pure/mock), ~10 stmts (DB)

### `internal/middleware` (70.8%)

**Uncovered functions:**
| Function | Line | Uncovered Lines | Testable? |
|----------|------|-----------------|-----------|
| `Stop()` | rate_limit.go:47 | ~7 stmts | Pure |
| `evictOldestLocked()` | rate_limit.go:98 | ~4 stmts | Pure |
| `LoginRateLimitMiddleware()` | rate_limit.go:156 | ~10 stmts | Gin test context |
| `RefreshRateLimitMiddleware()` | rate_limit.go:177 | ~10 stmts | Gin test context |
| `setCtxValue()` | auth.go:14 | ~3 stmts | Pure |
| `NewModularAuthMiddleware()` | auth.go:21 | ~13 stmts | Mock AuthService |
| `cleanupLoop()` | rate_limit.go:56 | ~5 stmts | Pure (timer-based) |

**Estimated recoverable:** ~52 stmts

### `internal/product` (59.4%)

**Uncovered functions (mock-testable):**
| Function | Line | Uncovered Lines | Testable? |
|----------|------|-----------------|-----------|
| `service.GetAllProducts()` | service.go:47 | ~25 stmts | Mock CategoryRepo |
| `handler.DeleteProduct()` | handler.go:290 | ~5 stmts | Mock Service |
| `handler.UpdateProduct()` | handler.go:226 | ~10 stmts | Mock Service |
| `handler.CreateProduct()` | handler.go:180 | ~5 stmts | Mock Service |

**DB-dependent (not testable with mocks):**
| Function | Line | Uncovered Lines |
|----------|------|-----------------|
| `adapter.*` | adapter.go | ~62 stmts |
| `query.GetAllProductsForExport` | query.go:231 | ~30 stmts |
| `BulkUpdateProducts` | bulk.go:239 | ~56 stmts |
| `Repository.*` (various) | repository.go | ~34 stmts |

**Estimated recoverable:** ~45 stmts (mock), ~180+ stmts (DB)

### `internal/category` (59.3%)

**Mock-testable:**
| Function | Line | Uncovered Lines |
|----------|------|-----------------|
| `handler.DeleteCategoryHandler` | handler.go:150 | ~8 stmts |
| `handler.CreateCategoryHandler` | handler.go:76 | ~2 stmts |
| `handler.UpdateCategoryHandler` | handler.go:107 | ~4 stmts |
| `service.CreateCategory` | service.go:27 | ~2 stmts |
| `service.UpdateCategory` | service.go:40 | ~3 stmts |

**DB-dependent:**
- `adapter.*`, `Repository.*` — ~85 stmts

**Estimated recoverable:** ~19 stmts (mock), ~85 stmts (DB)

### `internal/uom` (67.6%)

**Mock-testable:**
| Function | Line | Uncovered Lines |
|----------|------|-----------------|
| `handler.DeleteUnitOfMeasure` | handler.go:129 | ~8 stmts |
| `handler.UpdateUnitOfMeasure` | handler.go:86 | ~2 stmts |
| `handler.CreateUnitOfMeasure` | handler.go:55 | ~3 stmts |
| `adapter.LoadReferences` | adapter.go:101 | ~1 stmt (pure) |

**Estimated recoverable:** ~14 stmts (mock), ~17 stmts (DB)

### `internal/brand` (71.9%)

**Mock-testable:**
| Function | Line | Uncovered Lines |
|----------|------|-----------------|
| `handler.UpdateBrand` | handler.go:86 | ~2 stmts |
| `adapter.LoadReferences` | adapter.go:92 | ~1 stmt (pure) |

**Estimated recoverable:** ~3 stmts (mock), ~17 stmts (DB)

### `internal/report` (75.4%)

**Mock-testable:**
| Function | Line | Uncovered Lines |
|----------|------|-----------------|
| `handler.GetSalesChartData` | handler.go:324 | ~18 stmts |
| `handler.GetDashboardStats` | handler.go:257 | ~8 stmts |
| `handler.GetLiveDashboardStats` | handler.go:287 | ~6 stmts |
| `handler.GetAvailableYears` | handler.go:673 | ~6 stmts |
| `handler.ExportDashboard` | handler.go:586 | ~5 stmts |
| `service.GetDualMonthlyReport` | service.go:69 | ~2 stmts |

**Estimated recoverable:** ~45 stmts (mock), ~50 stmts (DB)

### `internal/shared` (69.9%)

**Uncovered:**
| Function | Line | Uncovered Lines | Testable? |
|----------|------|-----------------|-----------|
| `NewTestDB` | testdb.go:24 | ~10 stmts | DB-dependent |
| `RunMigrations` | testdb.go:37 | ~13 stmts | DB-dependent |
| `TruncateAll` | testdb.go:74 | ~1 stmt | DB-dependent |
| `TruncateTestData` | testdb.go:83 | ~1 stmt | DB-dependent |

**Estimated recoverable:** ~25 stmts (all DB-dependent)

### `internal/platform/importexport/progress` (53.3%)

All uncovered functions are in `pg_repo.go` — DB-dependent.
**Estimated recoverable:** ~55 stmts (pgxmock or integration tests)

### `internal/customer` (68.9%)

**Mock-testable:**
| Function | Line | Uncovered Lines |
|----------|------|-----------------|
| `handler.DeleteCustomer` | handler.go:301 | ~7 stmts |

**DB-dependent:**
- `adapter.*`, `Repository.GetAllCustomersForExport`, `Repository.BulkUpsertCustomers` — ~95 stmts

**Estimated recoverable:** ~7 stmts (mock), ~95 stmts (DB)

---

## Commit History (53.2% → 70.6%)

| Commit | Description | Coverage |
|--------|-------------|----------|
| `4d13cd9` | Fix 2 flaky E2E tests (roles, transactions) | 53.2% |
| `6827ffe` | E2E remediation Phase 1+2 + fix 14 pre-existing failures | 53.2% |
| `7c60468` | Fix 19 Vitest source-structure guard tests | 53.2% |
| `43baf53` | Unit tests for shared + middleware packages | 54.8% |
| `3264750` | Unit tests for user, product, category, customer packages | 55.4% |
| `c683ebd` | Extract service interfaces + XLSX roundtrip fix | 55.4% |
| `e05d04f` | E2E config: fullyParallel=false + self-contained product delete test | 55.4% |
| `c046112` | test(websocket): pure function, broadcast, listener tests | 57.2% |
| `2870b2e` | test(user): AuthHandler mock tests (61.5% → 71.0%) | 58.1% |
| `049a313` | Expand coverage phases 4-14,16,20 + fix flaky websocket tests | 67.0% |
| `337f95e` | Phase 17+19: import execute mock tests + websocket httptest tests | 70.3% |
| `2aafee8` | Fix E2E: reduce token cache TTL to 10 min | 70.3% |
| `c626fa5` | Phase 18+21: shared/testdb tests + report helper coverage | 70.6% |

---

## E2E Test Status

- **363 total tests** across 30 spec files
- **361 passing**, 2 flaky-in-isolation (pre-existing)
- **1 critical fix**: Roles API Update Role 401 — token cache TTL (55 min) exceeded JWT expiry (15 min)
- **Fix applied**: Reduced `TOKEN_TTL_MS` to 10 min in `tests/e2e/fixtures.ts:116`

---

## Remaining Phases (70.6% → 80%)

| Phase | Package | Action | Est. Gain |
|---|---|---|---|
| 22 | `eventbus` | Pure tests: Metrics, DroppedCount, deadLetter, retry+error, double-shutdown | ~27 stmts |
| 23 | `middleware` | Rate limiter: Stop, evictOldestLocked, LoginRateLimit, RefreshRateLimit, setCtxValue | ~52 stmts |
| 24 | `product` | Mock tests: GetAllProducts, DeleteProduct, UpdateProduct error paths | ~40 stmts |
| 25 | `category`+`uom`+`brand` | Handler error paths for all three | ~30 stmts |
| 26 | `report` | GetSalesChartData, GetDashboardStats, GetAvailableYears handler mock tests | ~40 stmts |
| 27 | `shared` | NewTestDB/RunMigrations (live DB) | ~20 stmts |
| 28 | `progress` | PgRepository pgxmock tests | ~55 stmts |
| 29 | `customer` | Handler Delete error paths | ~12 stmts |

**Total estimated gain from phases 22-29: ~276 stmts**

---

## Test Commands

```bash
# Full test suite
TEST_DB_PORT=5433 DB_PORT=5433 TEST_DB_USER=pos DB_PASSWORD=admin123 JWT_SECRET=test-secret-for-testing-only go test -p 1 -count=1 ./...

# Coverage report
TEST_DB_PORT=5433 DB_PORT=5433 TEST_DB_USER=pos DB_PASSWORD=admin123 JWT_SECRET=test-secret-for-testing-only go test -p 1 -count=1 -coverprofile=/tmp/cover.out $(go list ./... | grep -v '/cmd/')

# View total
go tool cover -func=/tmp/cover.out | tail -1
```
