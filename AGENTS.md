# Development Commands

## Codebase Investigation & Semantic Index Workflow

This project uses a semantic codebase index at `.opencode/index`. The current working tree is always the source of truth.

### Core principles

1. Prefer semantic index tools for non-trivial codebase investigation.
2. Use the smallest tool that can answer the current question.
3. Do not run every semantic tool for every task.
4. Do not perform a full re-index unless the index is stale or incomplete.
5. Use incremental indexing when only a limited part of the repository has changed.
6. Use raw `read`/`grep` when the task is inherently exact or textual.
7. Always verify important conclusions against the current working tree before editing.
8. Never treat stale semantic-index results as authoritative over the current source.

### Tool selection

| Tool | When to use |
|------|-------------|
| `codebase_context` | Default starting point for feature-level, architectural, or cross-module questions |
| `codebase_peek` | Conceptual area is known, but exact files/symbols are not |
| `implementation_lookup` | Symbol name is known, locate its authoritative definition |
| `codebase_search` | Need actual source content for known files/symbols/concepts |
| `call_graph` | Understand callers or callees of a known symbol |
| `call_graph_path` | Investigate end-to-end execution path between two points |
| `find_similar` | Before creating new implementation, check for existing patterns |
| `code_communities` | Module boundaries, architecture, coupling, hub symbols |
| `codebase_edit_context` | Before modifying a known implementation, get bounded context |
| `pr_impact` | Assess blast radius of a branch or planned change |

### Recommended investigation workflow

For non-trivial questions, follow this progression (stop when sufficient evidence is obtained):

1. Identify the relevant area with `codebase_context`
2. Narrow to likely files/symbols with `codebase_peek`
3. Locate definitions with `implementation_lookup`
4. Retrieve source with `codebase_search`
5. Trace dependencies with `call_graph` or `call_graph_path`
6. Search for existing patterns with `find_similar` before designing new behavior
7. Use `codebase_edit_context` before modifying
8. Verify findings against the current working tree, then implement

**Quick rules:** For known symbols, start with `implementation_lookup`. For exact text, use `grep`. For new features, add `find_similar` → `code_communities` before implementing.

### Incremental indexing policy

- **Prefer incremental indexing** for small changes (edited package, new service, added tests).
- **Full re-index** only when: index is missing/corrupt, large portion changed, major refactoring, or incremental cannot reconcile changes.
- **After normal edits:** Make change → incrementally index if supported → continue. Re-check health only when results appear stale.
- **After large refactors:** Complete refactor → check status/health → prefer incremental, otherwise full `index_codebase`.
- **Stale-index detection:** If a symbol cannot be found, is still returned after deletion, or results contradict source → check `index_status`/`index_health_check`, determine smallest scope, incrementally index, retry, escalate to full re-index only if necessary.
- **Do not repeatedly re-index** because a query returned no result — first consider whether the query is vague, wrong tool selected, target outside index, or symbol has a different name.

### Source-of-truth rule

The current working tree is authoritative. When semantic-index results conflict: trust the source, investigate staleness, update index when appropriate.

### Raw search/read exceptions

Use `read`/`grep` directly when: exact file/symbol is known, exact literal search requested, inspecting migrations/config/generated files/fixtures/docs, or semantic indexing does not cover the content.

### Efficiency rule

Optimize for minimum evidence. Do not: run every tool for every question, search the whole repo when target is known, retrieve full source when only symbol locations needed, full re-index for small changes, or repeatedly re-index without evidence of staleness.

## Environment Configuration

All database connection parameters are in `.env.example`:
- `DB_HOST=localhost`, `DB_PORT=5433` (dev) / `5432` (default), `DB_USER=pos`, `DB_PASSWORD=admin123`, `DB_NAME=retail_pos`

**Required:** `JWT_SECRET` (256-bit, generate with `openssl rand -hex 32`) — server panics at startup if missing.

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET_REFRESH` | (derived) | Separate secret for refresh tokens |
| `FRONTEND_PORT` | `5173` | Frontend dev server (Vite) |
| `BACKEND_PORT` | `9095` | Backend dev server (Go) |
| `DATABASE_PORT` | `5433` | Development database port |

Copy `.env.example` to `.env` and adjust as needed.

## Timezone Handling

**All queries must use Asia/Jakarta timezone** (data stored in UTC, Jakarta midnight = UTC 07:00).

- Backend: `time.ParseInLocation("2006-01-02", dateStr, jakartaLoc)`
- Frontend: `getTodayInJakarta()` and `getDateNDaysAgoInJakarta()` utilities

## Analytical Data Consideration

Analytical queries are served from materialized views pre-aggregated in Jakarta time, refreshed via `refresh_sales_mv()`:
- `mv_hourly_sales` — period comparisons and hourly chart
- `mv_daily_sales` — daily chart, dual-period chart, available years

`report.RefreshCoordinator` (`internal/report/refresh_coordinator.go`) refreshes once at startup and then at each Jakarta hour (`:00`) boundary. The `sale.created` listener only invalidates dashboard caches and never triggers a refresh. Refresh failures are retried with exponential backoff (`REPORT_REFRESH_DEBOUNCE` is base retry delay).

**Only completed hours/days are displayed.** The in-progress hour is never surfaced, even if a mid-hour refresh wrote a partial bucket. This invariant is enforced in: `report.getRealtimeRanges`, `Repository.GetHourlySales`, and frontend `chart-config.ts` / `data-fetching.ts`.

## Git Commit Policy
Never auto-commit. User will request commits explicitly.

## Running Tests

Tests require PostgreSQL connection and `JWT_SECRET`. Use env vars to point to dev DB:

```bash
TEST_DB_PORT=5433 DB_PORT=5433 TEST_DB_USER=pos TEST_DB_PASSWORD=admin123 DB_USER=pos DB_PASSWORD=admin123 JWT_SECRET=test-secret-for-testing-only go test -p 1 -count=1 ./...
```

**Important:** Use `-p 1` to force sequential execution (prevents deadlocks between concurrent `TRUNCATE`/`INSERT` across packages sharing the same DB).

### Targeted Testing (default workflow)

For most changes, run only affected packages/files, then a fast build sanity check:

- **Backend:** `go test -p 1 -count=1 ./internal/<package>/...` (optionally `-run <TestName>`)
- **Frontend:** `cd web && npx vitest run <path/to/test.file>`
- **Sanity check:** `go build ./...` and `cd web && npm run build`

Reserve full suite (`go test -p 1 -count=1 ./...`, `cd web && npm run test:run`, Playwright E2E) for pre-commit/CI/release or when explicitly requested. **Never run full suite proactively.**

**Test database:** `retail_pos_test` DB (configurable via `TEST_DB_*`). Auto-applies pending migrations via `schema_migrations` table. To reset: `dropdb retail_pos_test && createdb retail_pos_test`.

### E2E Testing Conventions (Playwright, `tests/e2e/`)

- **Behavior tests** (`*-api.spec.ts`) assert against API via `api-driver.ts`. **UI tests** (`*.spec.ts`) keep only genuine UI behavior (labels, navigation, validation messages).
- **`apiAs(request, role)`** returns authenticated `ApiDriver` for seeded roles (`superadmin`/`admin`/`manager`/`cashier`). Returns `ApiResult = { status, ok, body, headers }`.
- **Token rule (critical):** Do not reuse `request` fixture from `beforeAll` inside `test` bodies. Recreate `ApiDriver` per test via `beforeEach`.
- **Login rate-limited** (5/min per IP). Use `getToken(request, role)` / `apiAs` — tokens cached on disk in `fixtures.ts`. Bump `TOKEN_CACHE_VERSION` after backend restart with new `JWT_SECRET`.
- **Browser navigation:** Use explicit `page.goto(\`${FRONTEND_BASE}/...\`)`. Use `waitForAppReady(page)` (not `networkidle`).
- **POS/cart scenarios:** Use `pos-api.ts` helpers for shift/cart flows; raw `apiAs` for entity/CRUD specs.
- **Run from repo root** (`npx playwright test` / `npm run test:e2e`), never from `tests/e2e/`.

## Building

```bash
go build ./...
```

## Running Server

```bash
go run cmd/server/main.go
```

## Utilities

- `scripts/kill-port.sh <port>` — force-kill process holding a TCP port. Useful when `go run` child keeps port occupied (killing the parent `go run` PID does NOT free the port).

## Seeding Dummy Data

Never auto-commit. Changes must be committed manually.

```bash
./seed-dev.sh [flags]
```

| Flag | Description |
|------|-------------|
| `-products=N` | Number of products (4500-5000, random if 0; if 0 and DB has products, reuses existing) |
| `-days=N` | Days to generate (0 = interactive prompt) |
| `-categories=N` | Number of categories (65-80, random if 0) |
| `-truncate=false` | Skip truncating existing data |

Re-seeding (`-truncate=false`) continues document sequences and reuses existing products/suppliers/pricing rules, adding new transactions without key collisions.

## Deployment

### Migration Ordering

Migrations must be applied **before** deploying a new server binary. The server validates permission codes at startup — mismatched notation (dot vs colon) causes permission failures for all non-superadmin users.

| Migration | Purpose |
|-----------|---------|
| `001_consignment.sql` | Creates `consignment_*` tables/sequences and `consignment.*` permissions |
| `002_settlement_items_product_id.sql` | Adds `consignment_settlement_items.product_id` FK |
| `003_settlement_updated_at.sql` | Adds `consignment_settlements.updated_at` |
| `004_supplier_code_sequence.sql` | Creates `supplier_seq` for auto-generating `SUP-%06d` codes |
| `005_app_settings.sql` | Creates `app_settings` key-value table, seeds defaults |
| `006_user_preferences.sql` | Adds per-user `language`/`theme` columns to `users` |
| `007_sale_lookup.sql` | Seeds `sale.lookup` permission for `cashier`/`manager` |
| `031_revoke_sale_lookup_manager.sql` | Revokes `sale.lookup` from `manager` (cashier-only) |
| `032_sale_detail_and_receipt_print.sql` | Seeds `sale.detail`/`receipt.print` permissions |

## Filesystem Convention

Non-code files follow this organization:

```
docs/
├── adr/          — Architecture/Business decision records
├── design/       — Technical design docs, specs, feature references
├── prd/          — Product requirement documents
├── guides/       — End-user documentation and manuals
├── reviews/      — Design/product reviews, feasibility analyses
├── reports/      — Implementation/status/progress summaries
├── roadmap/      — Upcoming features roadmap
├── audits/       — Security, architecture, UI/UX, RBAC audits
├── archive/      — Outdated implementation plans
├── archived-plans/ — AI agent planning documents
├── examples/     — Pre-filled import/export templates
└── exported-sample/ — Exported dashboard/report samples
```

- Root-level kept: `README.md`, `CONTRIBUTING.md`, `AGENTS.md`, `LICENSE`
- SQL schema: `database/migrations/` (seed data consolidated into `000_squash.sql`)
- `docs/docs.go`, `docs/swagger.*` — swag-generated OpenAPI artifacts (do not move)
