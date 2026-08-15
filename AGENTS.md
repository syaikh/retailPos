# Development Commands

## Codebase Index Priority

This project has a semantic codebase index (`.opencode/index`). Prioritize the index-based tools for understanding code before falling back to plain grep/read scans:

- `codebase_context` — the default first step for repository questions (routes to definitions, call-graph paths, or conceptual evidence packs)
- `codebase_peek` — locate likely files/symbols by meaning without full source bodies
- `codebase_search` — retrieve full matching source content
- `codebase_edit_context` — bounded, edit-oriented evidence before modifying a known symbol
- `implementation_lookup` — jump to symbol definitions
- `call_graph` / `call_graph_path` — trace caller/callee relationships and dependency paths
- `find_similar` — detect duplicate or similar patterns
- `code_communities` — discover module boundaries and hub symbols
- `pr_impact` — assess blast radius of a branch/PR

If the index is out of date or missing results, check `index_status` / `index_health_check`, then re-index with `index_codebase` before falling back to raw search.

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

Re-seeding (`-truncate=false`) is supported: the seeder continues document sequences (`invoice_seq`, `po_seq`, `gr_seq`, `so_seq`) and reuses existing products/suppliers/pricing rules, so it adds new transactions without colliding on unique keys (SKUs, invoice numbers, PO/GR numbers, stock opname sessions, shift-per-cashier-date, customer phones).

## Deployment

### Migration Ordering

Migrations must be applied **before** deploying a new server binary. The server validates permission codes at startup, so if the binary expects dot-notation permissions (`.view`, `.create`) but the DB still has colon-notation (`:read`, `:create`), permission checks will fail for all non-superadmin users.

Key migrations with deployment ordering constraints:
- `006_consolidate_permissions.sql` — must be applied before the binary that removed `normalizePermissionCode`
- `009_add_do_sequence.sql` — creates the `do_seq` sequence required by `GetNextDONumber`; if the binary that auto-generates DO numbers on goods receipt is deployed first, every `POST /api/goods-receipts` will fail with a missing `do_seq` relation
- `012_stock_opname.sql` — creates the `so_seq` sequence, `stock_opnames` family of tables, and seeds `stock_opname.*` permissions/role grants; must be applied before the binary that added the Stock Opname endpoints, otherwise `POST /api/stock-opnames` fails with a missing `so_seq` relation and permission checks fail for `stock_opname.*` codes
- `016_stock_opname_scope_workflow.sql` — reworks the Stock Opname workflow to 9 states (draft/open/counting/verification/needs_recount/approved/posted/closed/cancelled), adds multi-scope sessions + recount requests, seeds `stock_opname.verify/post/close/report` and removes legacy `approve`/`reject`; must be applied before the binary that uses the new status codes and permissions, otherwise approval flows fail with unknown permission codes
- `017_stock_opname_adjustment_ledger.sql` — creates the `ia_seq` sequence and `inventory_adjustments`/`inventory_adjustment_items` ledger; must be applied before the binary that auto-generates adjustment numbers (IA-) on stock opname posting, otherwise `POST /api/stock-opnames/:id/post-adjustment` fails with a missing `ia_seq` relation
- `018_storage_locations.sql` — creates the `storage_locations` table and seeds `storage_location.*` permissions/role grants; must be applied before the binary that added the Storage Locations endpoints, otherwise `POST /api/storage-locations` fails with a missing `storage_locations` relation and permission checks fail for `storage_location.*` codes
- `013_remove_dead_permissions.sql` — deletes the unused permission codes `sale.print`, `sale.void`, `inventory.view`, `supplier_cost.view`, `supplier_cost.update` and their role grants; must be applied before the binary/UI that assumes those codes no longer exist
- `014_remove_orphaned_role_grants.sql` — least-privilege cleanup (Manager: `store.view`, `sale.create`, `sale.park`; Staff: `dashboard.view`, `shift.view`, `category.view`; Cashier: `dashboard.view`, `pricing.view`, `customer_group.view`, `store.view`); must be applied before the binary that hides those routes from those roles, otherwise the roles keep over-granted access via direct API calls
- `019_remove_remaining_orphaned_role_grants.sql` — completes the Manager cleanup by revoking `store.create/update/delete` and `customer_group.create/update/delete` (Manager has no Stores route and only views customer groups); same ordering constraint as `014`
- `020_per_rack_stock.sql` — adds `product_stock.location_id` (FK to `storage_locations`), re-creates `uq_product_stock` as `UNIQUE NULLS NOT DISTINCT (product_id, warehouse_id, store_id, location_id)`, and adds `stock_opnames.location_id` with `'location'` scope; must be applied before the binary that added the rack-stock endpoints and the `location` stock-opname scope, otherwise `POST /api/inventory/locations` fails with a missing `location_id` column and scope validation rejects `'location'`
- `021_grant_storage_location_view.sql` — grants `storage_location.view` to Manager/Staff/Cashier (the roles that render the product-detail rack panel and stock-opname location scopes); must be applied before the binary that exposes `GET /api/storage-locations` to those roles, otherwise the rack panel and location scope picker 403 for non-admin roles
- `022_admin_least_privilege.sql` — revokes `audit.view`, `role.update/delete`, `user.delete` from Admin (enforces the least-privilege split); must be applied before the binary/UI that hides those routes from Admin, otherwise Admin keeps over-granted access via direct API calls
- `023_sprint0_finalize_permissions.sql` — revokes `staff.product.update` and `staff.inventory.adjust` (final RBAC); must be applied **last** (PALING AKHIR), after all frontend Fase 1-5 changes, otherwise permission checks fail for non-superadmin users
- `024_add_product_history_cost_permissions.sql` — adds `product.history.view` (Superadmin/Admin) and `product.cost.view` (Superadmin/Admin/Manager); must be applied before the binary that omits `cost` from `GET /products` / `GET /products/:id` for non-holders, otherwise nobody holds `product.cost.view` and cost is hidden for everyone (degraded, non-breaking)
- `025_add_supplier_to_products_full_view.sql` — recreates `v_products_full` to add `supplier_id`/`supplier_name` (preferred supplier) columns; must be applied before the binary whose `productSelectCols` reads those columns from the view (otherwise `GET /products` and `GET /products/:id` fail with a missing `supplier_id` column)
- `028_mv_dashboard_totals.sql` — creates `mv_dashboard_totals` (per-store all-time totals) and extends `refresh_sales_mv()` to refresh it; must be applied before the binary whose `GetAllCompletedSalesStats` reads the view instead of `sales` (otherwise the dashboard all-time total card fails with a missing `mv_dashboard_totals` relation)
- `001_consignment.sql` — creates the `consignment_*` tables/sequences (`cr_seq`, `pr_seq`, `rt_seq`, `sl_seq`, `po_seq`) and seeds `consignment.*` permissions/role grants; must be applied before the binary that added the Konsinyasi Supplier module, otherwise `POST /api/consignment/*` fails with missing relations and permission checks fail for `consignment.*` codes
- `002_settlement_items_product_id.sql` — adds `consignment_settlement_items.product_id` (FK to `products`, NULL-able for empty previews); must be applied before the binary whose settlement reads scan that column (otherwise `GET /api/consignment/settlements` and settlement previews fail)
- `003_settlement_updated_at.sql` — adds `consignment_settlements.updated_at`; must be applied before the binary whose settlement queries scan that column (otherwise settlement reads fail)

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
