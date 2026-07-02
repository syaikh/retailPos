# Test Summary — Retail POS System

> Generated: 2026-06-17

---

## Overview

| Metric | Value |
|--------|-------|
| **Total test files** | 24 (17 Go + 7 frontend) |
| **Total test functions / tests** | ~264 (103 Go + 161 frontend) |
| **Go unit tests** | 49 (pure logic, no DB) |
| **Go DB-dependent tests** | 54 (~56 functions across 8 files) |
| **Frontend tests** | 161 (Vitest + @testing-library/svelte) |
| **Go benchmarks** | 1 (`BenchmarkChartDual`) |

---

## Go Backend Tests

### config — `internal/config/config_test.go`
**Package:** `config` · **Type:** Unit · **Tests:** 5

| Test | What it verifies |
|------|-----------------|
| `TestLoadDefaults` | Default env=development, CORS origin, JWT secret, stock thresholds, **Timezone=Asia/Jakarta** |
| `TestLoadFromEnv` | Env overrides for all config fields |
| `TestGetEnvInt_Defaults` | Default fallback for missing env vars |
| `TestGetEnvInt_InvalidFallsBack` | Invalid env values fall back to default |
| `TestGetEnvInt_ZeroAllowed` | Zero is a valid getEnvInt return |

### auth — `internal/auth/auth_test.go`
**Package:** `auth` · **Type:** DB · **Tests:** 11

| Test | What it verifies |
|------|-----------------|
| `TestAuthService_NewAuthService` | JWT secret defaults and TTLs |
| `TestAuthService_NewAuthService_WithEnv` | Env override for JWT secret |
| `TestAuthService_Login_Success` | Login with superadmin/admin123 → tokens, user fields, permissions |
| `TestAuthService_Login_InvalidCredentials` | Wrong password → ErrInvalidCredentials |
| `TestAuthService_Login_InactiveUser` | Inactive user → "account is inactive" |
| `TestAuthService_Login_UserNotFound` | Non-existent user → ErrInvalidCredentials |
| `TestAuthService_GenerateToken` | JWT claims (user ID, username, role, permissions, expiry) |
| `TestAuthService_ParseToken_Expired` | TTL=0 token → parse fails |
| `TestAuthService_ValidateToken` | Valid token validates successfully |
| `TestAuthService_ValidateToken_Invalid` | Invalid JWT string → error |
| `TestAuthService_RefreshToken_Success` | Refresh token flow |

### service — `internal/service/sales_test.go`
**Package:** `service` · **Type:** Unit · **Tests:** 6

| Test | What it verifies |
|------|-----------------|
| `TestErrInsufficientStock` | Error message formatting |
| `TestSalesService_ValidateStock` | Stock validation logic |
| `TestSalesService_StockCheckLogic` | Sufficient/insufficient stock |
| `TestSalesService_CreateSaleItemCalculation` | Sale item price × qty math |
| `TestSaleStruct` | Domain struct fields |
| `TestSaleItemSubtotal` | Subtotal calculation |

### repository — `internal/repository/` (7 test files)

#### `helpers_test.go` · **Unit** · 3 test functions (20+ cases)

| Test | What it verifies |
|------|-----------------|
| `TestGenerateSlug_Basic` | 20 cases: simple, special chars, consecutive hyphens, truncation, empty string |
| `TestGenerateSlug_Truncation` | Slug >120 chars truncated to 120 |
| `TestMustLoadJakarta` | Returns Asia/Jakarta (+7h), singleton instance |

#### `timezone_test.go` · **Unit** · 7 test functions

| Test | What it verifies |
|------|-----------------|
| `TestJakartaLocation_Load` | Asia/Jakarta loads with +07:00 offset |
| `TestParseDateInJakarta` | `time.ParseInLocation(..., Asia/Jakarta)` produces midnight Jakarta |
| `TestGetAllSalesDateBoundaries` | Closed-open interval `[start, end+24h)` in Jakarta |
| `TestGetDashboardStatsTodayString` | `now.In(jkt).Format("2006-01-02")` round-trips |
| `TestParseDateInJakarta_Invalid` | 7 cases: empty, garbage, wrong format, partial, valid, leap, non-leap Feb 29 |
| `TestAuditLogBoundaryExclusive` | `< endDate+24h` boundary math (regression for `<=` → `<` fix) |
| `TestPeriodComparisonRanges` | DateRange symmetry in Jakarta |

#### `timezone_integration_test.go` · **DB** · 4 test functions

| Test | What it verifies |
|------|-----------------|
| `TestGetAllSales_JakartaDateBoundary` | Sale at 23:59:59 UTC → included in next Jakarta day, excluded from previous |
| `TestGetAvailableYears_JakartaBoundary` | Cross-year sale `2024-12-31T18:00Z` counted as 2025 (Jakarta), not 2024 |
| `TestGetDualChartData_JakartaCrossMidnight` | Revenue aggregated by Jakarta date even across 17:00 UTC midnight |
| `TestAuditLogBoundary_Integration` | `GetAll` with `endDate` boundary: late UTC log included/excluded correctly |

#### `repository_test.go` · **DB** · 17 test functions

| Test | What it verifies |
|------|-----------------|
| `TestPostgresRepository_GetByUsername` | Fetch existing user |
| `TestPostgresRepository_GetByUsername_NotFound` | Non-existent user → error |
| `TestPostgresRepository_GetAllUsers` | Pagination, ordering |
| `TestPostgresRepository_GetAllUsers_WithSearch` | Search by keyword |
| `TestPostgresRepository_GetAllUsers_FilterByRole` | Filter by role ID |
| `TestPostgresRepository_GetAllUsers_FilterByActiveStatus` | `is_active = true` |
| `TestPostgresRepository_GetAllUsers_FilterByInactiveStatus` | `is_active = false` |
| `TestPostgresRepository_GetAllUsers_SortByUsernameAsc` | Username ascending |
| `TestPostgresRepository_GetAllUsers_SortByEmailDesc` | Email descending |
| `TestPostgresRepository_GetRolePermissions` | Superadmin permissions |
| `TestPostgresRepository_GetRolePermissions_Cashier` | Cashier permissions |
| `TestPostgresRepository_GetAllRoles` | All roles exist |
| `TestPostgresRepository_GetAllPermissions` | All permission codes |
| `TestPostgresRepository_GetAllProducts` | Pagination, fields |
| `TestPostgresRepository_GetProductBySKU` | Fetch by SKU |
| `TestPostgresRepository_GetProductBySKU_NotFound` | Non-existent SKU |
| `TestPostgresRepository_GetAllProducts_WithMultipleCategoryFilter` | Multi-category filter |

#### `live_dashboard_test.go` · **DB** · 2 test functions

| Test | What it verifies |
|------|-----------------|
| `TestGetLiveDashboardStats_Accuracy` | Inserted sale revenue matches dashboard exactly |
| `TestGetLiveDashboardStats_StoreScoped_SingleStoreRequirement` | Skipped (single store in seed data) |

#### `sales_performance_test.go` · **DB** · 1 test function

| Test | What it verifies |
|------|-----------------|
| `TestGetAllSalesPerformance` | N+1 query fix: performance measurement |

#### `chart_dual_benchmark_test.go` · **DB** · 1 benchmark + 1 test

| Test | What it verifies |
|------|-----------------|
| `BenchmarkChartDual` | Go loop vs SQL CTE performance across 7d/30d/90d/365d |
| `TestDualApproachesMatch` | Go loop and SQL CTE produce matching totals |

### handler — `internal/delivery/http/handler/` (5 test files)

#### `helpers_test.go` · **Unit** · 10 test functions (50+ cases)

| Test | What it verifies |
|------|-----------------|
| `TestNormalizeUsername` | 9 cases: lowercase, trim, alphanumeric-only, empty, mixed |
| `TestParseIntPtr` | 6 cases: positive, zero, negative, empty, invalid, mixed |
| `TestValidateCustomerRequest` | 12 cases: valid, empty/whitespace name, name >200, nil/empty/invalid phone, nil/empty/invalid email |
| `TestNormalizeProductBarcode` | 4 cases: nil stays nil, trimmed, empty→nil, whitespace→nil |
| `TestValidateProductPayload` | 9 cases: valid, valid w/ category_id, empty/whitespace name, empty sku, no category, price 0/negative, stock negative |
| `TestUserRole` | 4 cases: role exists, lowercase/trimmed, not exists, wrong type |
| `TestHasPermission` | 4 cases: permission exists, not exists, no key, wrong type |
| `TestCanManageProduct` | 5 cases: superadmin/admin/staff bypass, cashier with/without perm |
| `TestCanManageCategory` | 6 cases: superadmin/admin bypass, staff with/without, cashier with/without |
| `TestExportFilenameTimezone` | Valid YYYY-MM-DD format, +07:00 offset |

#### `period_test.go` · **Unit** · 14 test functions

| Test | What it verifies |
|------|-----------------|
| `TestDailyRanges_H7Comparison` | Yesterday vs H-7 (not H-2) |
| `TestWeeklyRanges_PartialDetection` | Mon-Sat partial, Sunday complete |
| `TestWeeklyRanges_LikeForLikeComparison` | Partial week = same-day comparison |
| `TestMonthlyRanges_LikeForLikeComparison` | Month-to-date comparison |
| `TestMonthlyRanges_CompletedMode` | Full last month comparison |
| `TestYearlyRanges_WithDec31ComparisonDate` | Full year vs full previous year |
| `TestYearlyRanges_CompletedMode` | Last completed year comparison |
| `TestRealtimeRanges_IncludesCurrentHour` | Current (partial) hour included |
| `TestRealtimeRanges_MidnightBoundary` | Correct at 00:15 Jakarta |
| `TestRealtimeRanges_EndOfDay` | Coverage up to hour 23 |
| `TestRealtimeRanges_TimezonePreservation` | All ranges in Asia/Jakarta |
| `TestMonthlyRanges_IsPeriodIncomplete` | 7 cases: Jan 1/30/31, Feb 28 leap/non-leap, Feb 29, Dec 31 |
| `TestWeeklyRanges_IsPeriodIncomplete` | 7 cases: Mon–Sat incomplete, Sunday complete |
| `TestYearlyRanges_YearIncomplete` | Before Dec 31 incomplete, Dec 31 complete |

#### `handler_audit_description_test.go` · **Unit** · 1 test function (7 sub-tests)

| Sub-test | What it verifies |
|----------|-----------------|
| `auth_login_with_username_should_include_username` | Login with username |
| `auth_logout_with_username_should_include_username` | Logout with username |
| `auth_login_without_username_falls_back_to_action_only` | Login without username |
| `create_product_with_identifier` | Create product description |
| `update_sale_with_invoice_identifier` | Update sale description |
| `delete_user_with_entity_id_fallback` | Delete user description |
| `create_role_with_identifier` | Create role description |

#### `handler_test.go` · **DB** · 7 test functions

| Test | What it verifies |
|------|-----------------|
| `TestAPI_Login_Success` | POST `/api/login` 200 |
| `TestAPI_Login_InvalidCredentials` | Wrong password → 401 |
| `TestAPI_GetProducts_Unauthorized` | No token → 401 |
| `TestAPI_GetProducts_Authorized` | GET `/api/products` 200 |
| `TestAPI_GetStats_Authorized` | GET `/api/stats` 200 |
| `TestAPI_GetProducts_WithCategoryFilter` | Category filter |
| `TestAPI_GetProducts_WithMultipleCategoryFilter` | Multi-category filter |

#### `handler_threshold_test.go` · **DB** · 9 test functions

| Test | What it verifies |
|------|-----------------|
| `TestConfig_Defaults` | Stock threshold defaults |
| `TestConfig_EnvOverride` | Env overrides |
| `TestConfig_InvalidEnvFallsBack` | Invalid env → defaults |
| `TestHandler_GetStockThresholds_Defaults` | GET `/api/stock-thresholds` |
| `TestHandler_GetStockThresholds_EnvOverrides` | Thresholds with env |
| `TestHandler_GetDashboardStats_LowStockCount_Default` | Low stock count default |
| `TestHandler_GetDashboardStats_LowStockCount_CustomThreshold` | Low stock count custom |
| `TestRepository_GetAllProducts_LowStockFilter` | Low stock filter DB query |
| `TestIntegration_StockClassificationBoundaries` | Boundary values classification |

### websocket — `pkg/websocket/` (2 test files)

#### `hub_test.go` · **DB** · 10 test functions

| Test | What it verifies |
|------|-----------------|
| `TestHub_Shutdown` | Clean shutdown, connections removed |
| `TestHub_NewHub` | Channels and rate limiter initialized |
| `TestHub_Run` | No premature broadcast on startup |
| `TestCheckOrigin` | Origin allowlist: localhost, 127.0.0.1, 192.168.x.x, 10.x.x.x |
| `TestServeWebSocket_Unauthorized` | No token → 401 |
| `TestServeWebSocket_InvalidToken` | Invalid token → 401 |
| `TestClient_Management` | Register/unregister via hub channels |
| `TestBroadcastProductUpdate` | Scoped broadcast to correct stores/roles |
| `TestRateLimiting` | IP-based rate limiter |
| `TestConnectionLimits` | Max connections per user enforced |

#### `broadcast_test.go` · **Unit** · 5 test functions

| Test | What it verifies |
|------|-----------------|
| `TestBroadcastStockUpdate` | Stock update event payload |
| `TestBroadcastSaleCreated` | Sale created event payload |
| `TestEventTypes` | Event type constants |
| `TestEventJSONMarshaling` | JSON serialization |
| `TestShouldReceiveEvent` | Event filtering logic |

---

## Frontend Tests (Svelte / Vitest)

### `web/src/lib/utils/jakartaTime.test.ts`
**Tests:** 15

| Test | What it verifies |
|------|-----------------|
| `JAKARTA_OFFSET_MS equals 7 hours` | 7h offset constant |
| `getTodayInJakarta returns a YYYY-MM-DD string` | Date format |
| `getTodayInJakarta: when UTC is 2025-05-31T23:30:00Z the Jakarta day is already 2025-06-01` | Cross-midnight boundary |
| `getDateNDaysAgoInJakarta(0) equals getTodayInJakarta` | Zero days ago |
| `getDateNDaysAgoInJakarta(1) is one calendar day before today` | 1 day ago |
| `getDateNDaysAgoInJakarta: getTodayInJakarta - getDateNDaysAgoInJakarta(6) = 7 distinct calendar days` | 7-day range width |
| `getFirstOfMonthNAgoInJakarta(0)` | Current month start |
| `getFirstOfMonthNAgoInJakarta(1)` | Previous month start |
| `getFirstOfMonthNAgoInJakarta(11)` | 11 months ago |
| `getMaxDateInJakarta` | Same as getTodayInJakarta |
| `all public functions return YYYY-MM-DD format` | Format consistency |
| `getCurrentJakartaHour returns 0-23 range` | Hour range |
| `getJakartaHourFromUTC converts correctly` | UTC→WIB conversion |
| `getJakartaHourFromUTC with RFC3339 offset format` | RFC3339 input |
| `getTodayInJakarta does not return a UTC date` | Not UTC |

### `web/src/lib/composables/useWebSocket.test.ts`
**Tests:** 3

| Test | What it verifies |
|------|-----------------|
| `should simulate the token refresh flow on reconnect` | Token refresh |
| `should use fallback to original token if refresh fails` | Fallback behavior |
| `should verify the reconnect delay increases with attempts` | Exponential backoff |

### `web/src/lib/components/Sidebar.svelte.test.ts`
**Tests:** 24

Role-based navigation structure: master data items, staff items, admin section, audit log superadmin guard, expanded/collapsed states, indentation, icons.

### `web/src/lib/pages/ProductsPage.svelte.test.ts`
**Tests:** 52 (4 describe blocks)

Stock classification thresholds, source-structure guards (modals, filters, stock adjustment, detail drawer, sort/search helpers), data-fetching roles, format helpers.

### `web/src/lib/pages/PosPage.svelte.test.ts`
**Tests:** 38 (2 top-level + 4 nested describe blocks)

Global keyboard shortcuts (F2/F4/Alt+Delete), checkout modal (payment, change calculation, quick cash presets), finalize sale & receipt print, thermal receipt print CSS, runtime logic (complete flow, change due, F4 guard, qty clamp).

### `web/src/lib/pages/Home.svelte.test.ts`
**Tests:** 10

Live dashboard: WebSocket connection status, sale_created listener, fetchLiveStats on mount, polling interval, cleanup on unmount, stat cards, quick access links.

### `web/src/lib/pages/admin/Users.svelte.test.ts`
**Tests:** 19

RBAC gating (canCreate/canEdit/canDelete for superadmin/admin/cashier), self-deletion guard, search/fetch/save/delete via apiClient, last_login display with "Never" fallback.

---

## Test Infrastructure

### `internal/repository/testdb.go`
- **Package:** `repository`
- Creates ephemeral PostgreSQL databases per test
- Runs all 16 migration files and 10 seed files
- Configurable via `TEST_DB_HOST`, `TEST_DB_PORT`, `TEST_DB_USER`, `TEST_DB_PASSWORD` env vars
- Default: `devuser:admin123@localhost:5433`
- Connection string includes `timezone=Asia/Jakarta`
- Cleans up by dropping the test database after each test

---

## Running Tests

```bash
# All pure-logic Go tests (no database required)
go test -count=1 -run '^(TestConfig_|TestGetEnvInt|TestNormalize|TestParseIntPtr
  |TestValidateCustomer|TestValidateProduct|TestUserRole|TestHasPermission
  |TestCanManage|TestExportFilename|TestGenerateSlug|TestMustLoadJakarta
  |TestJakarta|TestParseDate|TestGetAllSales|TestGetDashboard|TestPeriod
  |TestRealtime|TestMonthly|TestWeekly|TestYearly|TestDaily|TestDual
  |TestAuditLog|TestIntegration_Stock|TestGenerateAudit)' ./internal/...

# All Go tests (requires PostgreSQL)
TEST_DB_PORT=5433 DB_PORT=5433 \
  TEST_DB_USER=pos TEST_DB_PASSWORD=admin123 \
  DB_USER=pos DB_PASSWORD=admin123 \
  go test -v ./...

# Frontend tests
cd web && npx vitest run

# Build verification
go build ./...
cd web && npm run build
```
