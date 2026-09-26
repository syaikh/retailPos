# Pagination Deep-Paging Performance Plan

Status: approved for implementation (not started)
Scope: P0 hardening + keyset pagination for audit/sales + consignment server-side pagination
Related audits: `docs/audits/project-audit-report.md` (P‑02 count duplication), `docs/reviews/backend-review-2026-08-18.md` (export memory)

## Background: review findings

### Current pagination patterns

No cursor/keyset pagination exists anywhere in the backend. Four coexisting patterns:

1. **SQL `LIMIT/OFFSET` + separate `COUNT(*)`** — dominant pattern (18 queries). Handlers parse
   `limit`/`offset` via `shared.ParsePaginationParams` (`internal/shared/paging.go:10`) and return
   `PaginatedResponse{data,total,limit,offset,total_pages}` (`paging.go:31`, emitted at
   `internal/shared/response.go:173`). Repositories build the WHERE clause twice (COUNT + data):
   - `internal/sale/repository.go:345-374`
   - `internal/audit/repository.go:161-240`
   - `internal/product/query.go:13-137`
   - `internal/customer/repository.go:106-161`
   - `internal/user/repository.go:179-227`
   - `internal/shift/repository.go:390-419`
   - `internal/stockopname/repository.go:526-561`, `repository_workflow.go:475`
   - `internal/purchase/repository.go:467-486`
   - `internal/consignment/repository.go:211-247`
   - `internal/pricing/repository.go:568-597`
   - `category:144`, `brand:109`, `uom:109`, `supplier:229`, `store:136`,
     `storagelocation:83`, `customergroup:87`
2. **Fetch everything, paginate in the browser** — all five consignment detail lists render the
   shared `<Pagination>` component but slice arrays client-side:
   `SettlementPage.svelte:79-80`, `StockPage.svelte:36`, `ReturnPage.svelte:67`,
   `ReceiptEntry.svelte:96`, `PendingReturnPage.svelte:58`, backed by unbounded backend
   `ListSettlements/ListStock/ListReturns/ListReceipts/ListPendingReturns`
   (`internal/consignment/service.go:745, 989, 1106, 1296, 1469` — no LIMIT).
3. **Load-all → hydrate → filter in Go → slice** — consignment supplier-name search
   (`internal/consignment/service.go:159-194`).
4. **Unpaginated exports** — `GetSalesForExport` / `StreamSalesExportCSV`
   (`internal/sale/repository.go:518-593, 595+`).

Client-side slicing in `RolesPage`, `StockOpnameDetailPage`, `TermsEditor` is acceptable
(small, bounded datasets).

### Issues identified

- **Unbounded `offset`** — `ParsePaginationParams` caps `limit` but not `offset`; any authenticated
  client can force O(offset) scans (rate limit 50 rps is not protection). Page cost is
  O(offset + limit) whenever the ORDER BY cannot be index-backed.
- **`limit` clamp bug** — `limit > 100` resets to **20**, not 100 (`paging.go:12`, asserted by
  `paging_test.go:21`). Frontend callers asking for 500/1000/200 silently get 20 rows:
  `product-service.ts:133,166`, `UserFormModal.svelte:115`, `StockOpnamesPage.svelte:82,96`,
  `RackStockPanel.svelte:84`, `PosPage.svelte:175`. Live silent-truncation bug.
- **Unstable ordering** — nearly every paginated `ORDER BY` lacks a unique tie-breaker, so tied
  rows can repeat/skip across pages and concurrent inserts shift offsets. Keyset cannot be built
  on top until ordering is deterministic.
- **Index gaps** — `audit_logs` has `(created_at)` and `(store_id)` separately but no composite
  `(store_id, created_at, id)`; store scoping uses `(store_id IS NULL OR store_id = $1)`
  (`sale/repository.go:317`, `user/repository.go:196`, `audit/repository.go:168,207`) which defeats
  a plain `store_id` index for the COUNT pass.
- **Duplicated COUNT per page-view** — identical WHERE built twice per request. Largest
  *present-day* per-page cost (deferred; see Follow-ups).
- **Consignment** — four list endpoints return unbounded arrays while the UI shows a page footer;
  payload grows linearly with history. Client-side filters (status tab, product-name search)
  currently run over the full fetched array, so naïve server pagination would only filter within a
  page — filters must move into SQL.
- **Deep offsets are UI-reachable** — `Pagination.svelte:121-193` has first/last/jump-to-page;
  ~25 pages use it. No infinite scroll anywhere.

### Scale context

Dev seed: 10–20 sales/day × 180 days (~2–4k sales), 4,500–5,000 products. Deep offsets today are
~thousands, not millions. `audit_logs` (written on every mutating request) and `sales` are the two
append-only tables where this degrades fastest.

---

## Phase 1 — P0 hardening (backend, no API shape change)

### 1.1 Unique tie-breakers on every paginated ORDER BY

| File:line | Change |
|---|---|
| `internal/sale/repository.go:366,371` | `ORDER BY <sortBy> <dir>, s.id <dir>` / `ORDER BY s.created_at DESC, s.id DESC` |
| `internal/sale/repository.go:534,611,786` | append `, s.id DESC` (exports + lookup) |
| `internal/audit/repository.go:239` | `, al.id DESC` |
| `internal/shift/repository.go:417` | `ORDER BY s.%s %s, s.id %s` |
| `internal/purchase/repository.go:481` | `, po.id %s` |
| `internal/product/query.go:130` | `, v.id <dir>` |
| `internal/user/repository.go:226` | `, u.id <dir>` |
| `internal/stockopname/repository.go:560` | `, id DESC` |
| `internal/stockopname/repository_workflow.go:475` | `, a.id DESC` |
| `internal/consignment/repository.go:243,798,953,1088,1347` | `, a/r/pr/rt/st.id` matching direction |
| `internal/category/repository.go:144` | `, c.id ASC` |
| `internal/brand/repository.go:109` | `, id ASC` |
| `internal/uom/repository.go:109` | `, id ASC` |
| `internal/supplier/repository.go:229` | `, id ASC` |
| `internal/store/repository.go:136` | `, id ASC` |
| `internal/storagelocation/repository.go:83` | `, sl.id ASC` |

Already unique (no change): `customer:160` (`c.id`), `customergroup:87` (`cg.id`),
`pricing:573` (`id`).

Mock-regex tests (`adapter_test.go`, `repository_mock_test.go`) use substring matching
(`ORDER BY name`), so appended columns still match — verify with targeted `go test` per package.

### 1.2 Bound offset — `internal/shared/paging.go:10`

- Add `const MaxPageOffset = 10000`; clamp `offset` to it.
- Update `internal/shared/paging_test.go` with clamp cases.

### 1.3 Fix the limit clamp

- `paging.go:12`: `limit > DefaultMaxPageLimit` → clamp to **100** (not reset to 20).
- Update `paging_test.go:21` (`150 → 100`).
- Align frontend callers that exceed 100 → 100:
  `web/src/modules/product/services/product-service.ts:133,166`,
  `web/src/modules/admin/components/UserFormModal.svelte:115`,
  `web/src/modules/stock-opname/components/StockOpnamesPage.svelte:82,96`,
  `web/src/modules/inventory/components/RackStockPanel.svelte:84`,
  `web/src/modules/pos/components/PosPage.svelte:175`.
- Update source-asserting tests:
  `StockOpnamesPage.svelte.test.ts:25`, `RackStockPanel.svelte.test.ts:78`.
- Documented residual limitation: "get all rows" lookups now receive up to 100 rows
  (`DefaultMaxPageLimit` is the intended ceiling).

## Phase 2 — Keyset for `/audit-logs` and `/sales`

### 2.1 Contract (additive, back-compatible)

- New optional params: `after_created_at` (RFC3339) + `after_id` (int).
  Seek predicate `(created_at, id) < ($1, $2)`, `ORDER BY created_at DESC, id DESC`,
  fetch `limit+1` rows to compute `has_more`.
- Rules: cursor requires default sort (`created_at DESC`) → otherwise **400** with clear message;
  cursor takes precedence over `offset` (documented); cursor ignored on non-default sorts only via
  the 400 path (no silent fallback).
- Response: extend `PaginatedResponse` (`paging.go:31`) with `next_cursor` (string, omitempty) and
  `has_more` (bool); add `shared.JSONPaginatedCursor` beside `JSONPaginated` (`response.go:173`).
  `total` still returned (COUNT unchanged in this phase).
- Swagger `@Param` annotations: `GetSalesHistory` / `GetSalesLookup`
  (`internal/sale/handler.go:550+`, `705+`), `ListAuditLogs` (`internal/audit/handler.go:37`).

### 2.2 Repositories

- `internal/audit/repository.go:161` — cursor branch; COUNT path untouched.
- `internal/sale/repository.go:345` — cursor branch for default sort only; other sorts keep offset.

### 2.3 Indexes — new migration `database/migrations/051_pagination_indexes.sql`

```sql
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_id ON audit_logs (created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_store_created_id
  ON audit_logs (store_id, created_at DESC, id DESC) WHERE store_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_sales_created_id ON sales (created_at DESC, id DESC);
```

- Migration must be applied before deploying the new binary (migration-before-binary rule).
- Add a row to the AGENTS.md migration table.

### 2.4 Frontend adoption (shared `Pagination` component unchanged)

- `web/src/modules/admin/services/audit-logs-service.ts` + `AuditLogsTable` / audit page, and
  `web/src/modules/sales/stores/sales-store.svelte.ts` + `TransactionsPage` / `TransactionTable`:
  - Keep `cursorStack: string[]` + `nextCursor` per page. Every response (offset- or cursor-fetched)
    returns `next_cursor`.
  - Map `onPageChange` by offset arithmetic: `+limit` → cursor next; `−limit` → stack pop
    (fallback to offset when stack empty); anything else (jump/first/last) → offset request, then
    push the returned `next_cursor`.
  - Result: sequential browsing never pays an OFFSET scan; jump/first/last remain available and are
    bounded by `MaxPageOffset`.
- E2E: new specs asserting `after_created_at`/`after_id`/`next_cursor`/`has_more` round-trip and
  the 400 on cursor + non-default sort. Existing offset assertions must stay green:
  `tests/e2e/shifts.spec.ts:282`, `transactions.spec.ts`, `audit-logs-search.spec.ts`.

## Phase 3 — Consignment server-side pagination

### 3.1 Backend — `internal/consignment/{repository,service,handler}.go`

- Add `limit/offset` (+ filters) to `ListSettlements` (`service.go:1469`), `ListReturns` (`:1296`),
  `ListPendingReturns` (`:1106`), `ListReceipts` (`:745`), `ListStock` (`:989`); add `, id DESC`
  ordering + `COUNT(*)` sibling; always apply LIMIT.
- **Back-compat switch:** return paginated shape (`shared.JSONPaginated`) only when the request
  carries a `limit` param; when absent → current `{data:[...]}` full list, guarded by a hard SQL
  `LIMIT 1000`. Keeps `tests/e2e/consignment-flow.spec.ts:374`, `tests/e2e/rbac-api.spec.ts:62`,
  and full-list consumers (`ReturnPage` stock options, `ArrangementsPage:244`,
  `PendingSettlementsModal`) unchanged.
- Move client-side filters into SQL:
  - settlements: expose existing `status` as query param + `search` →
    `EXISTS (… consignment_settlement_items … product_name ILIKE …)`
  - receipts: `search` → `EXISTS` over receipt items (keep existing `product_id`)
  - stock: `search` over product name/sku
  - returns / pending-returns: no filters needed (pure slicing)
- Fix arrangements search: replace Go-side load-all (`service.go:159-194`) with SQL
  `JOIN suppliers … s.name ILIKE`; remove the `if limit > 0` unbounded guard
  (`repository.go:244`).

### 3.2 Frontend — five pages

- `SettlementPage`, `StockPage`, `ReturnPage`, `PendingReturnPage`, `ReceiptEntry`: fetch with
  `limit/offset`, bind `total` to the already-rendered `<Pagination>`, drop
  `.slice(pageOffset…)`, refetch (debounced) on filter change.
- `RolesPage` / `StockOpnameDetailPage` / `TermsEditor` keep client-side slicing.
- Update component tests asserting current fetch strings.

---

## Execution order & verification

1. Phase 1 → 2 → 3, each independently reviewable and shippable.
2. Load the `lint-code` skill before writing code. Per AGENTS.md: **no full local suites** — CI runs
   everything. Targeted local checks only when requested:
   - `gofmt -s -l` on touched Go files
   - `go test ./internal/shared/... ./internal/audit/... ./internal/sale/... ./internal/consignment/...`
   - `npx eslint <touched frontend files>`
   - `npx vitest run <touched test files>`
3. Migration `051` applied before the binary (CI Integration job covers this).

## Risks

- Phase 1 changes `limit=150 → 100` behavior — test update required; frontend callers expecting
  500/1000 will now get 100 (previously 20 — net improvement, but visible).
- Phase 2 adds params/fields only — no breakage expected.
- Phase 3 response shape depends on presence of `limit` param — frontend/backend must change in
  lockstep (single deploy).

## Follow-ups (explicitly deferred, measured later)

- `COUNT(*) OVER()` / count-once-per-filter-change — largest present-day per-page cost
  (P‑02 in `docs/audits/project-audit-report.md`).
- btree indexes on `ORDER BY name/code` columns (products, customers, brands, suppliers,
  categories, uom) — decide via `EXPLAIN ANALYZE` after Phase 1.
- Export streaming — `StreamSalesExportCSV` buffers all rows
  (`internal/sale/repository.go:595+`; noted in `docs/reviews/backend-review-2026-08-18.md`).
