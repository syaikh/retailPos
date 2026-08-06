# Development Commands

## Codebase Index Priority

This project has a semantic codebase index (`.opencode/index`). Prioritize the index-based tools for understanding code before falling back to plain grep/read scans:

- `codebase_search` / `codebase_peek` — semantic search by meaning (not just keyword), the default first step when asked about behavior or features
- `call_graph` / `call_graph_path` — trace caller/callee relationships between functions
- `implementation_lookup` — jump to symbol definitions
- `find_similar` — detect duplicate or similar patterns
- `pr_impact` — assess blast radius of a branch/PR

If the index is out of date or missing results, re-index with `index_codebase` before falling back to raw search.

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

For analytical/reporting purposes, materialized views or summary tables are used for:
- Period comparisons (reads `mv_hourly_sales`, refreshed via `refresh_sales_mv()`)
- Daily/hourly revenue aggregation (still computed on-the-fly from the raw `sales` table via `GetSalesChartData`)

`GetPeriodComparison` was migrated to the `mv_hourly_sales` materialized view; `GetSalesChartData` (daily/hourly revenue chart) still uses real-time aggregation. For production with large datasets (>1M records), a nightly-refreshed daily/hourly summary table would improve chart query performance.

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
- `-products=N` - Number of products (4500-5000, random if 0)
- `-days=N` - Days to generate (0 = interactive prompt)
- `-categories=N` - Number of categories (65-80, random if 0)
- `-truncate=false` - Skip truncating existing data

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
- `024_add_product_history_cost_permissions.sql` — adds `product.history.view` (Superadmin/Admin) and `product.cost.view` (Superadmin/Admin/Manager); must be applied before the binary that omits `cost` from `GET /products` / `GET /products/:id` for non-holders, otherwise nobody holds `product.cost.view` and cost is hidden for everyone (degraded, non-breaking)

## Filesystem Convention

Non-code files follow this organization:

- `docs/` — All documentation and planning documents (.md)
  - `docs/archive/` — Outdated/archived implementation plans
  - `docs/archived-plans/` — AI agent planning documents (copied from `.kilo/plans/`)
- Root-level kept: `README.md`, `CONTRIBUTING.md`, `AGENTS.md`, `LICENSE`
- Build artifacts and auto-generated files are gitignored (see `.gitignore`)
- SQL schema: `database/migrations/` and `database/seeds/`
