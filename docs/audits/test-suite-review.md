# Test Suite Review

**Date:** 2026-07-21
**Coverage:** 81.2% statement coverage across 1,530 test functions
**Packages:** 21 packages with `t.Skip("no database connection")` — silently skipped, never enforced in CI

---

## Verified Test Failures (pre-existing)

| Test | Failure |
|------|---------|
| `TestE2E_ValidateSession` | handler returns `"user"` key, test expects `"data"` |
| `TestAuthService_RefreshToken_Success` | FK violation on `refresh_tokens.user_id_fkey` |
| `TestAuthService_ChangePassword_Success` | "invalid username or password" |

---

## Critical Issues

### 1. Duplicated test layers (DRY violation)
Every module has `handler_test.go` (DB-backed integration) + `handler_mock_test.go` (unit) testing the same handler paths with near-identical subtests. The DB-backed tests already cover the handler through the full stack; the mock tests add ~0 new coverage for handlers.

### 2. Dead mocks (no assertion)
`mockAuditCreator` in `brand/handler_audit_test.go` (and counterparts in 10+ modules) is configured but never asserted for calls. Tests like `TestAuditHandler_CreateBrand` create `mockAuditCreator{}` with no fields set, then never verify audit was actually created.

### 3. Pre-existing broken tests shipped to main
Three tests are failing but documented as "pre-existing" — flaky by design. `TestAuthService_RefreshToken_Success` depends on `Login` succeeding with a test user that has no corresponding FK in `refresh_tokens`.

---

## High Priority

### 4. Flaky async tests with fixed sleeps
- Eventbus: multiple `time.After(1s)` / `time.Sleep(5-200ms)` for async coordination
- `pkg/websocket/broadcast_test.go`: 8 instances of `time.Sleep(50ms)`
- `internal/product/service_test.go:48`: `time.After(2 * time.Second)`
- `internal/inventory/service_test.go:67`: `time.After(time.Second)`
- `internal/sale/service_test.go:57`: `time.After(time.Second)`

### 5. Weak assertions throughout
- `assert.GreaterOrEqual(t, total, 0)` — tautology (store handler)
- `assert.NotNil(t, list)` — doesn't check list is non-empty (multiple places)
- `assert.True(t, buf.Len() > 0)` — always true after any Write (CSV tests)
- `_ = json.Unmarshal(w.Body.Bytes(), &body)` — silently ignores parse failures (`response_test.go` lines 65, 86, 103, 121)
- `_ inserted` — unused variable in adapter test error case

### 6. Dead statement
- `sale/export_test.go:197`: `_, _ = io.ReadAll(&buf)` — pure side-effect-free read discarded into the void

---

## Medium Priority

### 7. Slow DB tests — 48.4s combined
```
store:       22.5s
user:        25.9s
uom:          3.6s
supplier:     2.4s
```
Each package runs CRUD tests against a real PostgreSQL. Password hashing uses bcrypt (cost 14 in migration). No parallel subtest execution. No `testing.Short()` support.

### 8. Duplicate setup boilerplate
`skipIfNoDB`, `testAuthMiddleware`, `testPermMiddleware`, `setupXxxRouter` copied verbatim into 15+ packages. ~300 lines of nearly identical code. A shared `testutil` package would eliminate this.

### 9. Missing edge cases
- No tests for `context.Canceled`/`DeadlineExceeded` in repositories
- No SQL injection tests beyond `SanitizeSortBy` (raw SQL queries exist in repositories)
- No concurrent access tests (race detector) for repositories
- No zero-value domain model tests
- No boundary tests for pagination (offset > INT_MAX, negative limits)

### 10. Inconsistent naming
- `TestXxx_Yyy` vs `TestXxxYyy` mixed freely
- `TestMockBrandHandler_ListBrands` vs `TestHandler_ListBrands` vs `TestAuditHandler_CreateBrand`
- `pure_test.go` vs `domain_test.go` vs `adapter_test.go` — same pattern, different naming

---

## Low Priority

### 11. Missing benchmark tests
Zero `testing.B` benchmarks exist. Performance-sensitive paths (pricing resolver, bulk upsert, CSV export) have no perf guardrails.

### 12. No test for `internal/config` package
Zero test files for the config package.

### 13. No Playwright tests executed in CI
`playwright.config.js` and `tests/e2e/*.spec.ts` exist (42 E2E spec files) but no Go test runner invokes them.

---

## Priority-Ranked Improvement Suggestions

| # | Change | Effort | Impact |
|---|--------|--------|--------|
| 1 | Fix 3 pre-existing test failures — fix FK seeding for refresh_tokens, fix ValidateSession response key | Low | Eliminates CI noise |
| 2 | Replace fixed sleeps with retry/backoff in eventbus and websocket tests | Medium | Stops CI flakes |
| 3 | Consolidate into shared `testutil` package — eliminate `skipIfNoDB`, `testAuthMiddleware`, `setupXxxRouter` duplication | Medium | -300 LOC, DRY |
| 4 | Migrate handler DB tests to mock-only — remove DB-backed handler_test.go files where handler_mock_test.go already covers same paths | Medium | -40% test time, eliminates duplication |
| 5 | Strengthen assertions — replace tautologies, verify response bodies, check error on unmarshal | Low | Catches real regressions |
| 6 | Assert mock expectations in handler_audit_test.go — verify audit.CreateAuditLog was called with expected payloads | Low | Mocks become meaningful |
| 7 | Add `testing.Short()` support — skip bcrypt-heavy and long DB tests under `-short` | Low | Faster dev iteration |
| 8 | Add parallel subtest execution with `t.Parallel()` where safe | Low | Faster suite |
| 9 | Add race-detector tests for concurrent repository access | Medium | Catches data races |
| 10 | Add context cancellation tests for DB repositories | Medium | Correctness for production paths |

---

**Verdict:** The test suite is **moderately sufficient** (81.2% coverage). Core business logic (`supplier/service.go` at 100%, scanner, paging, CSV sanitization) is well-tested. The weakest areas are **handler audit tests** (dead mocks), **async eventbus/websocket** (timing-flaky), and **handler/DB integration tests** (duplicated with mocks). The top 4 fixes would resolve ~60% of the test-quality debt.
