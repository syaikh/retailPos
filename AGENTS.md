# Development Commands

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

For analytical/reporting purposes, consider using materialized views or summary tables for:
- Daily/hourly revenue aggregation (currently computed on-the-fly)
- Period comparisons (can be expensive for large datasets)
- Year-over-year and month-over-month comparisons

Currently, the system uses real-time aggregation via the `GetSalesChartData` and `GetPeriodComparison` endpoints, which query the raw `sales` table. For production with large datasets (>1M records), materialized views refreshed nightly would improve query performance.

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

## Filesystem Convention

Non-code files follow this organization:

- `docs/` — All documentation and planning documents (.md)
  - `docs/archive/` — Outdated/archived implementation plans
  - `docs/archived-plans/` — AI agent planning documents (copied from `.kilo/plans/`)
- Root-level kept: `README.md`, `CONTRIBUTING.md`, `AGENTS.md`, `LICENSE`
- Build artifacts and auto-generated files are gitignored (see `.gitignore`)
- SQL schema: `database/migrations/` and `database/seeds/`
