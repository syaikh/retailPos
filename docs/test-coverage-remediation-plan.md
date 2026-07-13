# Test Coverage Remediation Plan

**Date:** 2026-07-13
**Status:** Phase 1 + Phase 2 complete

## Problem

Critical test coverage gaps identified after audit remediation. Several security-sensitive
packages and new features have zero or minimal test coverage.

## Current State

| Area | Coverage | Status |
|------|----------|--------|
| Go packages (integration) | ~75% of public functions | Good baseline |
| E2E API endpoints | ~83% (either E2E or Go handler test) | Acceptable |
| Frontend components | ~67% | Moderate |
| Middleware (auth, CSRF, rate limit) | **0%** | Critical gap |
| Cache package | **0%** | Critical gap |
| Shared utilities (SQL/CSV injection prevention) | **0%** | Security risk |
| Auth service + handler | **0%** | Blocked by concrete types |

---

## Phase 1 — Pure Unit Tests (no DB, no mocks)

All functions in this phase are pure or self-contained in-memory logic. No database,
no interface mocking, no external services. Fast to write, fast to run.

### 1a. `pkg/cache/cache.go` → `pkg/cache/cache_test.go`

**Functions to test:**
- `New(defaultTTL, cleanupInterval)` — returns non-nil cache
- `Set(key, value)` / `Get(key)` — basic round-trip
- `Get` on missing key — returns (nil, false)
- `Delete(key)` — removes entry, subsequent Get returns miss
- `SetWithTTL(key, value, ttl)` — value expires after TTL (+/-10% jitter tolerance)
- `FlushByPrefix(prefix)` — removes all matching keys, preserves non-matching
- `FlushByPrefix` with empty prefix — removes nothing
- `Stats()` — returns correct item count
- `jitter(ttl)` — returns value within +/-10% of input, handles zero/negative input

**Approach:** Table-driven tests. Use short TTLs (50-100ms) for expiry tests.

### 1b. `internal/shared/paging.go` → `internal/shared/paging_test.go`

**Functions to test:**
- `ParsePaginationParams(limitStr, offsetStr)` — normal values, empty strings, zero, negative, >1000, non-numeric
- `SanitizeSortBy(sortBy, context)` — valid column, invalid column (SQL injection), unknown context
- `SanitizeSortDir(sortDir)` — "ASC", "DESC", "", "asc", "bogus"
- `NewPaginatedResponse(data, total, limit, offset)` — correct TotalPages calculation, edge cases (total=0, limit=0)

**Approach:** Table-driven tests. Focus on SQL injection prevention (e.g. `"id; DROP TABLE"`).

### 1c. `internal/shared/csv.go` → `internal/shared/csv_test.go`

**Functions to test:**
- `SanitizeCSVField(s)` — each dangerous prefix (`=`, `+`, `-`, `@`, `\t`, `\r`), normal strings, empty string
- `WriteCSVRow(w, record)` — writes sanitized CSV to buffer, verify output via csv.NewReader

**Approach:** Table-driven tests. Verify prefix characters are escaped/neutralized.

### 1d. `internal/middleware/security_headers.go` → `internal/middleware/security_headers_test.go`

**Functions to test:**
- `buildCSP(allowedOrigins)` — single origin, multiple origins, empty list
- `SecurityHeadersMiddleware(allowedOrigins)` — verify X-Frame-Options, X-Content-Type-Options, CSP, etc. are set
- `CSRFMiddleware` — GET passes, POST without Authorization blocked (403), POST with Authorization passes, POST with `X-Requested-With` passes

**Approach:** Gin test context with `httptest.NewRecorder`. No DB needed.

### 1e. `internal/middleware/rate_limit.go` → `internal/middleware/rate_limit_test.go`

**Functions to test:**
- `NewIPRateLimiter(rate, burst)` — creates limiter, Stop() cleans up
- `GetLimiter(ip)` — returns same limiter for same IP, different for different
- `GetLimiter` rate limiting — exhaust burst, verify Allow() returns false
- `getClientIP(c)` — from RemoteAddr, with X-Forwarded-For
- `RateLimitMiddleware` — request under limit returns 200, over limit returns 429

**Approach:** Unit tests with gin test context. Must call `Stop()` to clean up goroutines.

### 1f. `internal/middleware/auth.go` → `internal/middleware/auth_test.go`

**Functions to test (all context-based, no AuthService needed):**
- `GetUserID(c)` — returns correct int from context
- `GetUserRole(c)` — returns correct string from context
- `GetPermissions(c)` — returns correct []string from context
- `GetStoreID(c)` — returns correct *int from context
- `hasPermission(perms, required)` — true when present, false when absent
- `extractToken(c)` — from Authorization header, from cookie, missing returns ""
- `RoleMiddleware(role)` — matching role passes, non-matching returns 403
- `RequirePermission(perm)` — matching permission passes, missing returns 403
- `RequireAnyPermission(perms)` — any match passes, none match returns 403
- `AdminOnly()` — "superadmin" passes, "cashier" returns 403

**Approach:** Set gin context values directly, invoke middleware, check response.

### 1g. `internal/report/handler.go` helpers → `internal/report/handler_helpers_test.go`

**Functions to test (all pure time arithmetic):**
- `getDailyRanges(refDate, completedMode)` — rolling 7-day vs completed single-day
- `get7DaysRanges(refDate, completedMode)` — weekly comparison
- `get30DaysRanges(refDate)` — 30-day comparison
- `getWeeklyRanges(refDate, completedMode)` — Monday-aligned weeks
- `getRealtimeRanges(refDate)` — intraday comparison
- `getMonthlyRanges(refDate, completedMode)` — calendar month comparison
- `getYearlyRanges(refDate, completedMode)` — calendar year comparison
- `isPeriodIncomplete(periodType, refDate)` — weekly/monthly/yearly completeness checks

**Approach:** Table-driven with fixed Jakarta dates. Verify exact PeriodRange boundaries.
Note: These are unexported functions, tests go in `report` package. Must set `JWT_SECRET`
env var for `config.Load()` to succeed (used by `getComparisonRanges` which calls these helpers).

### 1h. `internal/audit/handler.go` helpers → `internal/audit/handler_helpers_test.go`

**Functions to test (pure):**
- `GenerateAuditDescription(log)` — various action/entity combinations, with/without name, with/without EntityID
- `parseDateParam(s)` — valid RFC3339, valid YYYY-MM-DD, invalid, empty

**Approach:** Table-driven tests. Tests in `audit` package for unexported `parseDateParam`.

### 1i. `internal/sale/export.go` → `internal/sale/export_test.go`

**Functions to test (pure, write to io.Writer):**
- `WriteCSV(rows, w)` — verify CSV header + rows, parse back with csv.NewReader
- `WriteXLSX(rows, w)` — verify no error, non-empty output, parse with excelize to validate

**Approach:** Write to `bytes.Buffer`, read back and validate structure.

---

## Phase 2 — Integration Tests (with test DB)

These require the existing `testdb` infrastructure (`shared.RunMigrations`, `shared.SetupTestDB`).

### 2a. `internal/user/handler_test.go` — Missing role endpoint tests

Added tests following existing `TestHandler_CreateRole` pattern:
- `TestHandler_UpdateRole` — happy path, not found (404), invalid ID (400)
- `TestHandler_DeleteRole` — happy path, role with users rejection (400), not found (idempotent 200), invalid ID (400)
- `TestHandler_ListPermissions` — returns all permissions

### 2b. `internal/sale/handler_test.go` — Missing payment method tests

- `TestHandler_GetPaymentMethodByCode` — found (CASH), not found (404)

### 2c. `internal/product/handler_test.go` — Missing handler tests

- `TestHandler_BulkUpdateProductStatus` — deactivate, activate, empty ids (400), invalid JSON (400)
- `TestHandler_PublicRoutes` — ListTaxClasses, GetStockThresholds (static response)

### 2d. `internal/audit/handler_test.go` — Missing endpoint tests

- `TestHandler_GetAuditLog` — found, not found (404), invalid ID (400)
- `TestHandler_ListEntityTypes` — returns distinct entity types

### 2e. `internal/sale/service_test.go` — Price validation integration

- `TestSaleService_CreateSalePriceValidation` — price mismatch returns `ErrPriceMismatch`, price match succeeds
- Added `mockPriceStore` implementing `ProductPriceGetter` and `ProductBatchPriceGetter`

---

## NOT in Scope (requires refactoring first)

### Auth service + handler (blocked)
- `auth_service.go` depends on concrete `*Repository` (no interface)
- `RefreshToken`, `Logout` use raw SQL via `dbPool` bypassing repo
- **Fix needed:** Extract `UserFinder`, `TokenStore`, `PasswordUpdater` interfaces

### Import/export handler (complex, lower ROI)
- Multi-step async import flow with progress tracking
- Would need extensive mocking or full integration setup

### WebSocket upgrade/pump logic (hard to unit test)
- Requires WebSocket client simulation
- Core broadcast logic already tested in `hub_test.go`

---

## Verification

After each phase:
1. `go build ./...` — must compile clean
2. `go test -p 1 -count=1 ./...` — all tests pass
3. New test files follow existing patterns (table-driven, testify assertions)
