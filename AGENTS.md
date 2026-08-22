# Development Commands

## Codebase Investigation — Semantic Index First

This project has a semantic codebase index at `.opencode/index`.

For repository investigation, prefer the semantic codebase tools over broad raw `grep`/`read` scans. The goal is to understand existing architecture, relationships, and implementation patterns before making changes.

### Required tool routing

Choose the semantic tool that best matches the investigation task:

- `codebase_context` — default for feature-level, architectural, conceptual, or repository questions; use this first when the relevant implementation area is not yet known.
- `codebase_peek` — locate likely files and symbols by meaning without retrieving full source bodies.
- `codebase_search` — retrieve matching source content after the relevant area/symbols are known.
- `codebase_edit_context` — gather bounded, edit-oriented evidence before modifying a known symbol or implementation.
- `implementation_lookup` — locate authoritative definitions of a known symbol.
- `call_graph` / `call_graph_path` — trace callers, callees, and dependency paths.
- `find_similar` — find analogous or duplicate implementations before creating new patterns.
- `code_communities` — understand module boundaries, clusters, and important hub symbols.
- `pr_impact` — assess the blast radius of a branch or PR.

### Investigation workflow

For non-trivial repository questions:

1. Start with the semantic index using the tool appropriate to the task.
2. Use `codebase_peek` to narrow the relevant files/symbols when necessary.
3. Use `codebase_search` or `implementation_lookup` to retrieve authoritative source.
4. Use `call_graph` / `call_graph_path` when behavior crosses package/module boundaries.
5. Use `find_similar` before introducing a new implementation when an existing pattern may already exist.
6. Before editing a known implementation, use `codebase_edit_context` to establish bounded context.
7. Only fall back to broad `grep`/raw `read` when semantic tools cannot answer the question or when an exact textual/file-level operation is more appropriate.

### Index health

If semantic tools return no result, suspiciously incomplete results, or appear inconsistent with the repository:

1. Check `index_status` and/or `index_health_check`.
2. If the index is stale and re-indexing is appropriate, run `index_codebase`.
3. Retry the semantic query after indexing.
4. If the relevant content is intentionally outside the index, or the task is inherently exact/textual, use `grep`/`read` directly.

Do not repeatedly retry semantic searches without changing the query or checking index health.

### Direct raw-search exceptions

Use `grep` or `read` directly when:

- the user requests an exact literal/string search;
- the exact file/path is already known;
- reading a specific configuration, migration, generated file, fixture, or documentation file;
- inspecting a small known source range;
- the semantic index does not contain the relevant content;
- re-indexing would provide no meaningful benefit.

### Important

Do not treat the semantic index as authoritative when its results conflict with the current working tree. The working tree is the source of truth for the actual code.

When semantic evidence and raw source disagree, verify against the current source before making conclusions or edits.

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

## Building

```bash
go build ./...
```

## Running Server

```bash
go run cmd/server/main.go
```

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
- SQL schema: `database/migrations/` (the `database/seeds/` directory was retired — all seed data is consolidated into migrations, see `030_consolidate_seed_permissions.sql`)
