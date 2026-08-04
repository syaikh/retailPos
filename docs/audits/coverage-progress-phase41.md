# Test Coverage Progress Report — Phase 49

**Date:** 2026-08-04  
**Total Coverage:** 85.7% (excl. cmd/tools, this measurement; see Phase 49 note)  
**Target:** 80% ✓ EXCEEDED  
**Baseline (incl. cmd/tools):** 74.5%

## Coverage by Package

| Package | Coverage | Change | Notes |
|---------|----------|--------|-------|
| `cache` | 100.0% | — | |
| `shared/importexport` | 100.0% | — | |
| `platform/importexport` | 100.0% | — | |
| `platform/importexport/schema` | 100.0% | — | |
| `platform/importexport/validation` | 100.0% | — | |
| `inventory` | **97.7%** | +14.0pp | pgxmock AdjustStock error paths (Phase 47) |
| `uom` | **97.1%** | **+22.9pp** | pgxmock repo+adapter tests (Phase 44) |
| `brand` | **97.0%** | **+23.4pp** | pgxmock repo+adapter tests (Phase 44) |
| `eventbus` | 96.7% | — | |
| `validation/validators` | **96.3%** | **+11.2pp** | Name() methods, empty rows, label fallback, type branches (Phase 48) |
| `customer` | **95.0%** | **+11.4pp** | scanCustomerRow refactor + pgxmock tests (Phase 42) |
| `platform/importexport/export` | 94.3% | — | |
| `template` | **92.8%** | **+9.5pp** | headerStyleFor, buildValidationHint, joinStrings, colWidth, columnIndex (Phase 48) |
| `user` | **92.1%** | **+13.1pp** | Repository mock tests (Phase 45) |
| `platform/importexport/progress` | 92.1% | +38.8pp | pgxmock PgRepository tests (Phase 41) |
| `platform/importexport/history` | **91.5%** | **+9.6pp** | pgxmock store error paths (Phase 47) |
| `category` | **90.8%** | **+24.0pp** | pgxmock repo+adapter tests + bug fix (Phase 44) |
| `handler` | **90.1%** | **+8.9pp** | HistoryReader interface + mock GetSnapshot/GetRows tests (Phase 48) |
| `config` | 89.7% | — | |
| `product` | **89.6%** | **+26.2pp** | pgxmock repo tests (Phase 43) |
| `middleware` | 89.5% | +2.2pp | |
| `report` | 88.2% | +11.4pp | |
| `websocket` | 87.8% | — | Flaky TestHub_BroadcastStoreFiltering |
| `sale` | **90.1%** | **+2.7pp** | cart repo/service branches, handler auth/error branches (Phase 49) |
| `audit` | **90.4%** | **+2.6pp** | export id-filter parse branches (Phase 49) |
| `platform/importexport/import` | 83.9% | — | |
| `shared` | **80.2%** | **+16.7pp** | timezone.go loadLocation extraction + init fallback test (Phase 48) |

## Phases Completed

| Phase | Description | Key Gains |
|-------|-------------|-----------|
| 1-25 | Initial coverage push (various packages) | 53.2% → 71.7% |
| 29-38 | Interface extractions + handler tests | Various package gains |
| 40 | jwt/v4 dead code fix + tests | user 78.2%→79.0% |
| 41 | DBPool interface + pgxmock repo tests | 63.8%→65.0% |
| 42 | customer scanCustomerRow refactor + full pgxmock tests | 65.0%→65.3%, customer 73.6%→80.2% |
| 43 | product pgxmock repo tests (46 tests) | 65.3%→67.4%, product 63.4%→78.3% |
| 44 | brand/uom/category pgxmock repo+adapter tests | 67.4%→69.8%, brand 73.6%→97.0%, uom 74.2%→97.1%, category 66.8%→90.8% |
| 45 | user repository mock tests (40+ tests) | 69.8%→70.2%, user 79.0%→83.2% |
| 46 | shared scanner/context/logger + inventory EventBus + handler error branches | 70.2%→89.5% (excl. cmd/tools) |
| 47 | history pgxmock + inventory AdjustStock DB error paths | 89.5%→89.8% |
| 48 | timezone.go init fallback, template engine branches, validators edge cases, handler HistoryReader interface | 89.8%→**90.6%** |
| 49 | sale + audit close to ≥90%: cart repo/service branches, handler auth/error branches, export id-filter parses | sale 87.4%→**90.1%**, audit 87.8%→**90.4%** |

## Key Technical Findings

### Coverage Measurement Note (Phase 49)
Full-repo re-measurement on 2026-08-04 reports 85.7% (excl. `cmd/`/`tools/`). The drop from the Phase 48 headline of 90.6% is a measurement difference, not a regression: `internal/wiring` (untested, 0.0%) is now included in the package list and no longer excluded, lowering the aggregate. Sale (90.1%) and audit (90.4%) both exceed the 90% bar set for this phase; the doc's Phase 48 total was measured against a narrower package set.

### HistoryReader Interface Extraction (Phase 48)
The `handler.HistoryStore` field was a concrete `*history.Store`, preventing mock testing. Extracted a `HistoryReader` interface with `GetSnapshot` and `GetRows` methods. The concrete `*history.Store` already satisfies this interface — no changes needed to production wiring.

### sync.Once Reset for Logger Testing (Phase 46)
Testing `InitLogger("production")` requires resetting the package-level `once` variable since `sync.Once` only fires once per process. Accessible from tests because they're in the same package.

### loadLocation Extraction for Timezone Testing (Phase 48)
`time.LoadLocation` is a stdlib function called in `init()`, which runs once at package load. Extracted it into a package-level function variable `loadLocation` to allow tests to simulate failure. Refactored init body into `loadJakartaLocation()` so tests can call it directly.

### pgxmock WithArgs Requirement (Phases 43-45)
Every `ExpectQuery()`/`ExpectExec()` MUST have `.WithArgs(...)` when the actual code passes arguments. Use `pgxmock.AnyArg()` for argument positions where exact matching isn't needed.

### Coverage Calculation Note (Phase 46)
`cmd/server` E2E tests hit a live server and are not instrumented by Go's coverage profiler. `tools/` has no tests. Both are excluded from coverage measurement via:
```bash
go test -coverprofile=coverage.out $(go list ./... | grep -v -E '(cmd/|tools/)')
```

## Files Created/Modified (Phase 49)

### Created
- `internal/sale/cart_service_extra_test.go` — DB-driven cart tests: CreateOrGetOpenCart, GetCartByID ownership, Hold/Resume/expire/not-owned/not-open, quantity validation, checkout-with-shift, held-cart store/shift/customer slide-through, default TTL, item update/remove branches, `TestSaleRepository_CreateCartSession`
- `internal/sale/cart_handler_mock_test.go` — mock tests for all 11 cart handlers + `TestSaleCartHandler_NonIntUserID`

### Modified
- `internal/sale/handler_mock_test.go` — `setupSaleHandlerUser` helper + `TestSaleHandler_UserContextBranches` (missing/non-int userID), `TestSaleHandler_CreateSale_ParkedSaleNotRecalled` (409 conflict), `TestSaleHandler_ParkSale_BindJSONError` (400)
- `internal/sale/service_test.go` — `TestSaleService_CreateSaleWithShift` (shift totals), `TestSaleService_CreateSaleNegativeUnitPrice` (invalid unit price branch)
- `internal/sale/repository_test.go` — fixed `TestSaleRepository_StreamSalesExportCSV`, added `TestSaleRepository_StoreScopedReads` (store-id filter + non-null store_id scan branches)
- `internal/audit/handler_test.go` — export id-filter subtest (user_id/entity_id parse branches)
- `docs/audits/coverage-progress-phase41.md` — this report

## Remaining Gaps Below 90%

| Package | Coverage | Difficulty |
|---------|----------|------------|
| `shared` | 80.2% | testdb.go needs real DB (0% coverage) |
| `platform/importexport/import` | 83.9% | engine.go complex file parsing |
| `websocket` | 87.8% | flaky broadcast test |
| `report` | 88.2% | remaining query edge cases |
| `wiring` | 0.0% | pure dependency-assembly package, no unit-testable logic |
