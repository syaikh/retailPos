# Development Commands

## Codebase Investigation & Semantic Index Workflow

This project uses a semantic codebase index located at `.opencode/index`.

The semantic index should be the primary tool for understanding non-trivial repository structure, relationships, and existing implementation patterns. However, the current working tree is always the source of truth.

The goal is to use semantic tools efficiently, avoid unnecessary full-codebase scans, and keep the index sufficiently up to date through incremental indexing.

### Core principles

1. Prefer semantic index tools for non-trivial codebase investigation.
2. Use the smallest tool that can answer the current question.
3. Do not run every semantic tool for every task.
4. Do not perform a full re-index unless there is evidence that the index is stale or incomplete.
5. Use incremental indexing when only a limited part of the repository has changed.
6. Use raw `read`/`grep` when the task is inherently exact, textual, or concerns a known file.
7. Always verify important conclusions against the current working tree before editing.
8. Never treat stale semantic-index results as authoritative over the current source.

### Tool selection

Use the following decision rules:

#### `codebase_context`

Use as the default starting point when:

- the question is feature-level, architectural, conceptual, or repository-wide;
- the relevant implementation area is not yet known;
- multiple modules may be involved;
- the task requires understanding how several components work together.

Examples:

- "How does checkout work?"
- "How does Stock Opname approval affect inventory?"
- "Where should consignment logic live?"
- "What happens when a cashier completes a split payment?"

Do not use it for a simple exact lookup when the target is already known.

#### `codebase_peek`

Use when the conceptual area is known but the exact files or symbols are not.

Purpose:

- locate likely files;
- locate likely symbols;
- narrow the investigation scope;
- avoid retrieving large source bodies unnecessarily.

Example:

"Find the files and symbols involved in Stock Opname approval."

#### `implementation_lookup`

Use when the symbol is already known and the goal is to locate its authoritative definition.

Examples:

- "Where is `CompleteSale` implemented?"
- "Find the implementation of `PricingService.ResolvePrice`."
- "Where is `StockAdjustmentService` defined?"

Prefer this over broad search when the symbol name is known.

#### `codebase_search`

Use when relevant files, symbols, or concepts are known and actual source content is required.

Examples:

- retrieve the implementation of `ResolvePrice`;
- inspect the validation around `ApproveStockOpname`;
- find all relevant source implementations of a known concept.

Prefer targeted semantic searches over broad repository-wide grep.

#### `call_graph`

Use when understanding callers or callees of a known symbol.

Examples:

- "Who calls `CompleteSale`?"
- "What does `ProcessPayment` call?"
- "Which code paths depend on `AdjustStock`?"

#### `call_graph_path`

Use when the specific relationship between two points matters.

Examples:

- "How does `POST /checkout` reach `PaymentRepository.Save`?"
- "How does Stock Opname approval eventually update inventory?"

Prefer this when investigating an end-to-end execution path.

#### `find_similar`

Use before creating a new implementation when an existing pattern may already exist.

Examples:

- document-number generation;
- lifecycle/state machines;
- approval workflows;
- repository/service patterns;
- event handlers;
- validation patterns;
- import/export jobs.

The goal is to follow existing repository conventions rather than inventing a parallel pattern.

#### `code_communities`

Use when the question concerns module boundaries, architecture, coupling, or important hub symbols.

Examples:

- "Which modules are most closely related to consignment?"
- "What depends heavily on the pricing module?"
- "Where are the architectural boundaries around inventory?"

#### `codebase_edit_context`

Use immediately before modifying a known implementation when additional bounded context is needed.

It should help establish:

- the target implementation;
- relevant interfaces;
- nearby domain/application logic;
- important callers/callees;
- related tests;
- relevant invariants and side effects.

Do not retrieve the entire repository merely to edit one known symbol.

#### `pr_impact`

Use when assessing the consequences of a branch or planned change.

Examples:

- "What could be affected by changing pricing resolution?"
- "What is the blast radius of this PR?"
- "Which modules and tests may need updates?"

Use it before finalizing significant cross-module changes when impact is uncertain.

---

## Recommended investigation workflow

For a non-trivial repository question, follow this general progression:

1. Understand the problem.
2. Identify the relevant area with `codebase_context`.
3. Narrow to likely files/symbols with `codebase_peek`.
4. Locate authoritative definitions with `implementation_lookup`.
5. Retrieve relevant source with `codebase_search`.
6. Trace dependencies with `call_graph` or `call_graph_path` when necessary.
7. Search for existing patterns with `find_similar` before designing new behavior.
8. Use `code_communities` when module boundaries or coupling matter.
9. Use `codebase_edit_context` before modifying a known implementation.
10. Use `pr_impact` when the change may affect multiple modules.
11. Verify important findings against the current working tree.
12. Only then implement the change.

Do not blindly execute every step. Stop once sufficient evidence has been obtained.

### Example: broad architectural question

For:

"How does payment work in this POS?"

Prefer:

`codebase_context`
→ `codebase_peek`
→ `implementation_lookup` / `codebase_search`
→ `call_graph` if necessary

Do not immediately run `grep` across the entire repository.

### Example: known symbol

For:

"Where is `ResolvePrice` implemented?"

Prefer:

`implementation_lookup`

Do not start with `codebase_context`.

### Example: exact text

For:

"Find every occurrence of `ErrPaymentTotalMismatch`."

Use:

`grep` or another exact textual search.

Semantic investigation is unnecessary unless the user asks for behavioral/contextual analysis.

### Example: implementing a new feature

For:

"Add consignment support."

Prefer:

`codebase_context`
→ `codebase_peek`
→ `find_similar`
→ `code_communities`
→ `implementation_lookup` / `codebase_search`
→ `codebase_edit_context`
→ implement
→ `pr_impact`

---

## Incremental indexing policy

The semantic index should be kept reasonably synchronized with the working tree without unnecessary full re-indexing.

### First check

Before performing expensive indexing operations, determine whether the index is:

- missing;
- unhealthy;
- clearly stale;
- missing recently changed files;
- returning incomplete or contradictory results.

Use:

- `index_status`
- `index_health_check`

when available.

### Prefer incremental indexing

When only a small portion of the repository has changed, prefer incremental indexing of the affected files/directories if the indexing tool supports it.

Examples:

- one edited Go package;
- a new application service;
- several files in one module;
- recently added tests.

Do not rebuild the entire index merely because a few files changed.

### When to use full re-indexing

Perform a full `index_codebase` only when appropriate, such as:

- the index is missing;
- index health indicates corruption or an unrecoverable problem;
- a large portion of the repository changed;
- major refactoring changed package/module structure;
- many files were added, deleted, or moved;
- the index schema/indexer version requires rebuilding;
- incremental indexing cannot reliably reconcile the changes.

### After normal edits

Do not automatically run a full re-index after every code change.

For small changes:

1. Make the code change.
2. If supported, incrementally index the affected files.
3. Continue working.
4. Re-check index health only when semantic results appear stale or inconsistent.

### After large refactors

For changes involving many files, packages, symbols, or module boundaries:

1. Complete the refactor.
2. Check index status/health.
3. Prefer incremental indexing if it can reliably cover all affected files.
4. Otherwise run a full `index_codebase`.
5. Re-run important semantic queries after indexing.

### Stale-index detection

Treat the index as potentially stale when:

- a recently created symbol cannot be found;
- a recently deleted symbol is still returned;
- file paths no longer exist;
- semantic results contradict the current source;
- call-graph relationships appear impossible;
- search results stop at an older implementation;
- newly changed code is consistently absent.

When this happens:

1. Check `index_status` / `index_health_check`.
2. Determine the smallest affected scope.
3. Incrementally index that scope when possible.
4. Retry the semantic query.
5. Escalate to full re-indexing only if necessary.
6. Fall back to `read`/`grep` when the index cannot provide reliable evidence.

### Important

Do not repeatedly re-index just because a semantic query returned no result.

First consider:

- whether the query is too vague;
- whether the wrong semantic tool was selected;
- whether the target is outside the indexed source;
- whether the symbol has a different name;
- whether the index is actually stale.

---

## Source-of-truth rule

The semantic index is an optimization and navigation layer.

The current working tree is authoritative.

When semantic-index results conflict with the current source:

1. Trust the current source.
2. Investigate why the index is stale or inconsistent.
3. Update the index when appropriate.
4. Do not make implementation decisions based solely on stale index results.

---

## Raw search/read exceptions

Use `read`, `grep`, or equivalent raw tools directly when:

- the exact file is already known;
- the exact symbol location is already known;
- an exact literal/string search is requested;
- inspecting a migration;
- inspecting configuration;
- inspecting generated files;
- inspecting fixtures or snapshots;
- inspecting documentation;
- inspecting a small known source range;
- semantic indexing does not cover the relevant content.

Semantic tools are intended to improve repository understanding, not to replace every raw file operation.

---

## Efficiency rule

Optimize for the minimum evidence required to answer the question correctly.

Do not:

- run every semantic tool for every question;
- perform repository-wide searches when the target is known;
- retrieve full source files when only symbol locations are needed;
- perform a full re-index for a small change;
- repeatedly re-index without evidence that the index is stale.

Use the smallest appropriate tool and expand the investigation only when the evidence is insufficient.

## Environment Configuration

### Database Connection Parameters
All database connection parameters are defined in `.env.example` for the development environment:
- `DB_HOST=localhost`
- `DB_PORT=5433` (development), `5432` (default)
- `DB_USER=pos`
- `DB_PASSWORD=admin123`
- `DB_NAME=retail_pos`

### Required Environment Variables
The following environment variables are **required** — the server will panic at startup if missing:
- `JWT_SECRET` — 256-bit random secret for JWT signing. Generate with: `openssl rand -hex 32`

### Optional Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET_REFRESH` | (derived from `JWT_SECRET`) | Separate secret for refresh tokens (recommended different in production) |
| `FRONTEND_PORT` | `5173` | Frontend dev server port (Vite) |
| `BACKEND_PORT` | `9095` | Backend dev server port (Go) |
| `DATABASE_PORT` | `5433` | Development database port (postgres-dev container) |

Copy `.env.example` to `.env` and adjust values as needed for your local setup.

## Timezone Handling

**All queries must use Asia/Jakarta timezone** as data is stored in UTC. The backend uses Jakarta timezone for date calculations, and the frontend calculates Jakarta dates in UTC before sending to the API.

Key points:
- Jakarta midnight = UTC 07:00 (7-hour offset)
- Date filters should use `time.ParseInLocation("2006-01-02", dateStr, jakartaLoc)` on backend
- Frontend uses `getTodayInJakarta()` and `getDateNDaysAgoInJakarta()` utilities

## Analytical Data Consideration

Analytical queries are served from materialized views pre-aggregated in Jakarta time and refreshed via `refresh_sales_mv()`:

- `mv_hourly_sales` — period comparisons (`GetPeriodComparison`) and the hourly chart (`GetHourlySales`)
- `mv_daily_sales` — the daily chart (`GetDailySales`), the dual-period chart (`GetDualChartData`), and available years (`GetAvailableYears`)

`refresh_sales_mv()` is owned by `report.RefreshCoordinator` (`internal/report/refresh_coordinator.go`), which refreshes once at startup and then at each Jakarta hour (`:00`) boundary; the `sale.created` listener (`report.Repository.NewSaleCreatedListener`) only invalidates dashboard caches and never triggers a refresh. A single worker goroutine runs at most one refresh at a time. Startup (`cmd/server/main.go`) and seed (`cmd/dummy/main.go`) still refresh directly. Refresh failures are retried by the coordinator with exponential backoff (`REPORT_REFRESH_DEBOUNCE` is the base retry delay, not a debounce window) and never fail the `SaleCreated` event or trigger eventbus retries.

**Only completed hours/days are displayed.** Realtime and single-day reports show data through the *last completed hour* — e.g., at 11:20 Jakarta the last bucket is 10:00 (10:00–<11:00). The in-progress hour is never surfaced, even if a mid-hour refresh (startup at :20, retry backoff, manual `refresh_sales_mv()`) already wrote a partial bucket for it. This invariant is enforced in three coordinated places that must stay in sync:

- `report.getRealtimeRanges` (`internal/report/ranges.go`) — realtime period end is the start of the in-progress hour (exclusive), so `sale_hour < hourStart` excludes any partial bucket and both periods cover the same completed hours.
- `Repository.GetHourlySales` (`internal/report/repository.go`) — a chart for *today* caps its upper bound at the start of the current hour; past days span the full 24 hours.
- Frontend (`web/src/modules/reporting/utils/chart-config.ts` builds the axis with `length: currentHour`, and `data-fetching.ts` filters realtime points with `hour < currentHour`) — the chart never renders the in-progress hour slot.

Charts read the MVs and are eventually consistent: completed hours/days are always up to date, and the in-progress hour/day appears at the next `:00` boundary.

Remaining real-time `sales` reads are intentional: the live dashboard stats (`GetLiveDashboardStats`, `GetDashboardStats`) are today-only with a short cache TTL, and the weekly/monthly sales reports are date-bounded scans.

## Git Commit Policy
Never auto-commit on each change. User will request commits explicitly when ready.

## Running Tests

Tests require PostgreSQL connection and `JWT_SECRET`. Use env vars to point to dev DB:

```bash
TEST_DB_PORT=5433 DB_PORT=5433 TEST_DB_USER=pos TEST_DB_PASSWORD=admin123 DB_USER=pos DB_PASSWORD=admin123 JWT_SECRET=test-secret-for-testing-only go test -p 1 -count=1 ./...
```

**Coverage measurement** — exclude `cmd/` and `tools/` (E2E tests hit a live server, not instrumented by Go's coverage profiler):

```bash
TEST_DB_PORT=5433 DB_PORT=5433 TEST_DB_USER=pos DB_PASSWORD=admin123 JWT_SECRET=test-secret-for-testing-only go test -p 1 -count=1 -coverprofile=coverage.out $(go list ./... | grep -v -E '(retail-pos-system/cmd/|retail-pos-system/tools/)')
```

**Important:** All packages connect to the same PostgreSQL database. `go test` runs test binaries in parallel by default (`-p` defaults to `GOMAXPROCS`), causing deadlocks between concurrent `TRUNCATE` and `INSERT` ops across packages. Use `-p 1` to force sequential execution.

### Targeted Testing (default workflow)

For most changes, do **not** run the full suite. Run only the affected packages/files, then a fast build sanity check:

- **Backend:** `go test -p 1 -count=1 ./internal/<package>/...` (optionally append `-run <TestName>` to filter further)
- **Frontend:** `cd web && npx vitest run <path/to/test.file>` (e.g. `src/modules/admin/components/__tests__/RolesPage.svelte.test.ts`)
- **Sanity check:** `go build ./...` and `cd web && npm run build` catch compile/type errors without running any tests

Reserve the full suite (`go test -p 1 -count=1 ./...`, `cd web && npm run test:run`, Playwright E2E) for pre-commit/CI/release validation, or when explicitly requested. **Never run the full suite proactively — only when the user explicitly asks for it.**

All tests pass. Both previously documented pre-existing issues have been resolved:
- `TestE2E_ValidateSession` now passes — test expects `"user"` key matching handler response
- No `TestInteractors` exists in `internal/audit`; all audit tests pass consistently

**Test database setup:** Tests connect to `retail_pos_test` DB (configurable via `TEST_DB_*` env vars). The test framework auto-applies pending migrations on first run using a `schema_migrations` tracking table. If the test DB schema is out of sync, recreate it: `dropdb retail_pos_test && createdb retail_pos_test` and re-run tests.

### E2E Testing Conventions (Playwright, `tests/e2e/`)

The E2E suite distinguishes **behavior** tests (data, CRUD, RBAC contracts) from **UI** tests (rendering, navigation, client validation). Behavior is asserted against the API, not the DOM, so checks are fast and deterministic.

- **Behavior / CRUD / RBAC checks live in `*-api.spec.ts`** and use the API driver from `api-driver.ts`. **Browser-only `*.spec.ts` keep only genuine UI behavior** (visible labels, navigation, validation messages). Do not re-assert server-side data rules through the DOM.
- **`apiAs(request, role)`** returns an `ApiDriver` already authenticated as a seeded `TEST_USERS` role (`superadmin`/`admin`/`manager`/`cashier`). Methods: `get/post/put/patch/del/multipart(path, file, extra?)`. Every call returns `ApiResult = { status, ok, body, headers }` — treat `res.ok` as a boolean (not a method) and read `res.body` / `res.status`.
- **Anonymous / negative auth:** `new ApiDriver(request, '')` sends no token (expect 401); `new ApiDriver(request, 'invalid-token')` exercises rejected tokens.
- **RBAC is enforced at the endpoint, not by permission prefixes.** Assert the HTTP outcome: allowed roles get 2xx, forbidden roles get 403. Do not assert on permission-code strings.
- **Token rule (critical):** a `request` fixture captured in `beforeAll` must NOT be reused inside `test` bodies — Playwright forbids it. Store the returned token (or `apiAs` result) and **recreate the `ApiDriver` per test via `beforeEach`** using the test's own `request`.
- **Login is rate-limited** (5/min per IP). `fixtures.ts` caches tokens on disk (shared across workers, versioned file, deduped in-process, re-validated against the API). Use `getToken(request, role)` / `apiAs` rather than calling `/api/login` directly. Bump `TOKEN_CACHE_VERSION` in `fixtures.ts` if the backend restarts with a new `JWT_SECRET` or a fresh DB.
- **Browser navigation:** prefer explicit `page.goto(\`${FRONTEND_BASE}/...\`)` over `page.goto('/...')`. `waitForAppReady(page)` (in `fixtures.ts`) gates on a real nav button, not `networkidle` (the SPA holds a persistent WebSocket, so `networkidle` never fires).
- **POS/cart scenarios:** `pos-api.ts` helpers (`authAs`, `createCashier`, `ensureOpenShift`, `startFreshCart`, `addCartItem`, `holdCart`, `findProductWithStock`) accept `AuthCtx | ApiDriver` and layer on top of `apiAs`. Use them for shift/cart flows; use raw `apiAs` for entity/CRUD specs.
- **Run the suite from the repo root** (`npx playwright test` / `npm run test:e2e`), never from `tests/e2e/` — the root `playwright.config.js` supplies `baseURL` and the `api-driver`/`fixtures`/`pos-api` module graph.

## Building

```bash
go build ./...
```

## Running Server

```bash
go run cmd/server/main.go
```

## Utilities

- `scripts/kill-port.sh <port>` — force-kill whichever process is holding a TCP
  port (uses `lsof -ti :<port>` + `kill -9`). Useful when a leftover dev server,
  the Go print agent (`tools/print-agent`), or a Playwright-driven `go run`
  child binary keeps a port (e.g. `9123`/`9124`/`9095`) occupied after a crashed
  or backgrounded process. Note: `go run` spawns a temp-path binary, so killing
  the parent `go run` PID does NOT free the port — use this script instead.

## Seeding Dummy Data

Never auto-commit. Changes must be committed manually.

```bash
./seed-dev.sh [flags]
```

Flags:
- `-products=N` - Number of products (4500-5000, random if 0; if 0 and the DB already has products, reuses the existing ones — safe for re-seeding with `-truncate=false`)
- `-days=N` - Days to generate (0 = interactive prompt)
- `-categories=N` - Number of categories (65-80, random if 0)
- `-truncate=false` - Skip truncating existing data

Re-seeding (`-truncate=false`) is supported: the seeder continues document sequences (`invoice_seq`, `po_seq`, `gr_seq`, `so_seq`) and reuses existing products/suppliers/pricing rules, so it adds new transactions without colliding on unique keys (SKUs, invoice numbers, PO/GR numbers, stock opname sessions, shift-per-cashier-date, customer phones). Consignment data (arrangements, receipts, returns, settlements) is seeded when available.

## Deployment

### Migration Ordering

Migrations must be applied **before** deploying a new server binary. The server validates permission codes at startup, so if the binary expects dot-notation permissions (`.view`, `.create`) but the DB still has colon-notation (`:read`, `:create`), permission checks will fail for all non-superadmin users.

Key migrations with deployment ordering constraints:
- `001_consignment.sql` — creates the `consignment_*` tables/sequences (`consignment_receipt_seq`, `consignment_return_seq`, `consignment_settlement_seq`, `consignment_payout_seq`, `consignment_stock_seq`) and seeds `consignment.*` permissions/role grants; must be applied before the binary that added the Konsinyasi Supplier module, otherwise `POST /api/consignment/*` fails with missing relations and permission checks fail for `consignment.*` codes
- `002_settlement_items_product_id.sql` — adds `consignment_settlement_items.product_id` (FK to `products`, NULL-able for empty previews); must be applied before the binary whose settlement reads scan that column (otherwise `GET /api/consignment/settlements` and settlement previews fail)
- `003_settlement_updated_at.sql` — adds `consignment_settlements.updated_at`; must be applied before the binary whose settlement queries scan that column (otherwise settlement reads fail)
- `004_supplier_code_sequence.sql` — creates the `supplier_seq` sequence used to auto-generate supplier codes (`SUP-%06d`) when a create payload omits `code`; must be applied before the binary whose supplier `Create` auto-generates codes, otherwise `POST /api/suppliers` with a blank code fails with a missing `supplier_seq` relation
- `005_app_settings.sql` — creates the `app_settings` key-value table for global application configuration (store branding, receipt text), seeds defaults, and grants `app_settings.view`/`app_settings.update` to superadmin/admin; must be applied before the binary that reads/writes app settings, otherwise the server panics or returns 500 on `/api/settings`
- `006_user_preferences.sql` — adds per-user `language` and `theme` columns to `users`, removes dead `default_language` key from `app_settings`; must be applied before the binary that reads/writes user language/theme preferences, otherwise login responses omit those fields
- `007_sale_lookup.sql` — seeds the `sale.lookup` permission and grants it to the `cashier` and `manager` roles; must be applied before the binary that registers `permissions.SaleLookup` and gates the cross-cashier redacted `GET /api/sales/lookup` endpoint on it, otherwise cashiers/managers get 403 on the "Find Transaction" tab and its `sale.lookup` permission check fails
- `031_revoke_sale_lookup_manager.sql` — revokes the `sale.lookup` grant from the `manager` role (Find Transaction is cashier-only); must be applied before the binary that hides the "Find Transaction" tab for managers, otherwise managers see a redundant/weaker redacted subset
- `032_sale_detail_and_receipt_print.sql` — seeds `sale.detail` and `receipt.print` permissions and grants them to `cashier`, `manager`, `admin`, `superadmin`; must be applied before the binary that registers `permissions.SaleDetail`/`ReceiptPrint` and gates the cross-cashier drill-down (`/sales/lookup/:id`) and receipt reprint endpoints on them, otherwise cashiers get 403 on the Find Transaction detail/reprint actions

## Filesystem Convention

Non-code files follow this organization:

- `docs/` — All documentation and planning documents (.md), organized by content:
  - `docs/adr/` — Architecture/Business decision records (ADR, BDR, business process)
  - `docs/design/` — Technical design docs, specs, test specs, feature references
  - `docs/prd/` — Product requirement documents (PRD)
  - `docs/guides/` — End-user documentation and manuals
  - `docs/reviews/` — Design/product reviews and feasibility analyses
  - `docs/reports/` — Implementation/status/progress summaries
  - `docs/roadmap/` — Upcoming features roadmap
  - `docs/audits/` — Security, architecture, UI/UX, RBAC audits and improvement plans
  - `docs/archive/` — Outdated/archived implementation plans
  - `docs/archived-plans/` — AI agent planning documents (copied from `.kilo/plans/`)
  - `docs/examples/` — Pre-filled import/export example templates
  - `docs/exported-sample/` — Exported dashboard/report samples
  - `docs/docs.go`, `docs/swagger.go`, `docs/swagger.json`, `docs/swagger.yaml` — swag-generated OpenAPI artifacts (the `docs` Go package, imported by `cmd/server/main.go`; do not move)
- Root-level kept: `README.md`, `CONTRIBUTING.md`, `AGENTS.md`, `LICENSE`
- Build artifacts and auto-generated files are gitignored (see `.gitignore`)
- SQL schema: `database/migrations/` (the `database/seeds/` directory was retired — all seed data is consolidated into `000_squash.sql`)
