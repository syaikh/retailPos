# Development Commands

## Codebase Investigation & Semantic Index

This project uses a semantic codebase index at `.opencode/index`, use semantic tools provided as much as possible. The current working tree is always the source of truth.

### Tool selection

| Tool | When to use |
|------|-------------|
| `codebase_context` | Default starting point for feature-level, architectural, or cross-module questions |
| `codebase_peek` | Conceptual area is known, but exact files/symbols are not |
| `implementation_lookup` | Symbol name is known, locate its authoritative definition |
| `codebase_search` | Need actual source content for known files/symbols/concepts |
| `semble_search` | Second opinion from the independent Semble index when `codebase_search` misses or returns doc-heavy noise; strongest on frontend/Svelte, where the Go-weighted index underperforms. Discards paths under `docs/`, `*.md`, and `database/migrations/` when judging whether a hit is real code |
| `call_graph` | Understand callers or callees of a known symbol |
| `call_graph_path` | Investigate end-to-end execution path between two points |
| `find_similar` | Before creating new implementation, check for existing patterns; matches against a pasted snippet |
| `semble_find_related` | Similarity from the Semble index, keyed off a `file:line` you already have — use when there is no snippet to match against, e.g. locating near-duplicates of a known implementation for the duplication review track |
| `code_communities` | Module boundaries, architecture, coupling, hub symbols |
| `codebase_edit_context` | Before modifying a known implementation, get bounded context |
| `pr_impact` | Assess blast radius of a branch or planned change |

**Quick rules:** For known symbols, start with `implementation_lookup`. For exact text, use `grep`. For new features, add `find_similar` → `code_communities` before implementing. When a semantic search comes back thin, cross-check with `semble_search` before falling back to `grep`. When you have a location but no snippet, use `semble_find_related` instead of `find_similar`.

### Incremental indexing

- Prefer incremental indexing for small changes. Full re-index only when index is missing/corrupt, large portion changed, or major refactoring.
- Do not repeatedly re-index because a query returned no result — first consider whether the query is vague, wrong tool selected, or target outside index.
- **Never run `index_codebase` with `force=true`, or `/index force`, on your own initiative.** A full re-index is expensive and the user runs it themselves. If the index is stale or a symbol you just wrote is not found, say so and let the user re-index; fall back to `grep`/`Read` in the meantime rather than triggering a rebuild.

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
| `LOGIN_RATE_LIMIT_RPM` | `5` | Login rate limit requests per minute |
| `LOGIN_RATE_LIMIT_BURST` | `10` | Login rate limit burst |
| `RATE_LIMIT_RPS` | `50` | General API rate limit requests per second |
| `RATE_LIMIT_BURST` | `100` | General API rate limit burst |
| `REFRESH_RATE_LIMIT_RPM` | `10` | Token refresh rate limit RPM |
| `REFRESH_RATE_LIMIT_BURST` | `20` | Token refresh rate limit burst |
| `STOCK_WARNING_THRESHOLD` | `10` | Stock warning level |
| `STOCK_CRITICAL_THRESHOLD` | `5` | Stock critical level |
| `CART_HOLD_TTL_HOURS` | `24` | Cart hold TTL in hours |
| `REPORT_REFRESH_DEBOUNCE` | `30` | Report refresh retry delay (seconds, exponential backoff) |
| `ENV` | `development` | Log format: development/production |
| `LOG_LEVEL` | `info` | Log level: debug/info/warn/error |
| `VITE_PRINT_MODE` | | Receipt printing mode (frontend) |
| `VITE_PRINT_AGENT_URL` | | Print agent URL (frontend). **Leave unset for a multi-register shop** — see below |

`VITE_PRINT_AGENT_URL` is a build-time override that names **one** print agent
for **every** register. The frontend falls back to `http://localhost:9123`, so
each register finds the agent on its own PC and one build serves the whole shop.
Set this only for development (frontend and agent on different machines) or a
kiosk where printers sit on a server. In a shop with one printer PC per till,
setting it sends every till's receipts to a single machine. The per-register
setting in the print settings screen outranks it, so it stays correctable, and the
frontend warns when the active address is not localhost in silent mode.
`PRINT_TOKEN` was removed from the print agent: the only client is a browser, so
a token shipped to a browser is not a secret. See
`docs/guides/print-agent-production.md`.

Copy `.env.example` to `.env` and adjust as needed.

## Timezone Handling

**All queries must use Asia/Jakarta timezone** (data stored in UTC, Jakarta midnight = UTC 07:00).

- Backend: `time.ParseInLocation("2006-01-02", dateStr, jakartaLoc)`
- Frontend: `getTodayInJakarta()` and `getDateNDaysAgoInJakarta()` utilities

## Analytical Data Consideration

Analytical queries are served from materialized views pre-aggregated in Jakarta time, refreshed via `refresh_sales_mv()`:
- `mv_hourly_sales` — period comparisons and hourly chart
- `mv_daily_sales` — daily chart, dual-period chart, available years
- `mv_dashboard_totals` — live dashboard stats

`report.RefreshCoordinator` (`internal/report/refresh_coordinator.go`) refreshes once at startup and then at each Jakarta hour (`:00`) boundary. The `sale.created` listener only invalidates dashboard caches and never triggers a refresh. Refresh failures are retried with exponential backoff (`REPORT_REFRESH_DEBOUNCE` is base retry delay).

**Only completed hours/days are displayed.** The in-progress hour is never surfaced, even if a mid-hour refresh wrote a partial bucket. This invariant is enforced in: `report.getRealtimeRanges`, `Repository.GetHourlySales`, and frontend `chart-config.ts` / `data-fetching.ts`.

## Git Commit Policy
Never auto-commit. User will request commits explicitly.

## Linting Conventions

New/edited code must pass all linters on the next CI run. Load the `lint-code` skill for backend (golangci-lint, go vet, gofmt -s) and frontend (ESLint, Prettier, svelte-check) rules, depguard import boundaries, and the verified Svelte 5 `$effect` bare-read fix patterns. Do not run full suites locally — run targeted per-package/file checks only when the user asks.

## CI/CD (GitHub Actions)

**All validation — formatting, linting, type-checking, testing, building, and security scanning — runs in GitHub CI.** Do not run full suites locally. The CI workflow (`.github/workflows/ci.yml`) triggers on pushes to `main` and on pull requests.

**Do not run these locally — CI performs them:**
- `gofmt`, `go vet`, `go build`, `go test`, `go test -race`, `golangci-lint`, `govulncheck`
- `npm run lint`, `npm run check` (svelte-check), `npm run build`, `npm run test:run`, `prettier --check`
- Database migrations, integration health checks, CodeQL analysis

**Do not execute lint, type-check, build, or test commands without being asked.** If the user wants to verify something, provide the exact command and let them run it themselves.

For rapid iteration during development, only run the specific package or file you are changing.

### CI Jobs

| Job | What it checks |
|-----|----------------|
| Go: Format | `gofmt -s` formatting |
| Go: Vet | `go vet ./...` |
| Go: Build | `go build ./...` |
| Go: Tests | `go test -p 1 -count=1 ./...` + race detector (with PostgreSQL) |
| Go: Lint | `golangci-lint` (govet, staticcheck, errcheck, ineffassign, unused, revive, depguard) |
| Go: Vulnerability Check | `govulncheck ./...` |
| Frontend: Install | `npm ci` (dependency gate) |
| Frontend: Format | `prettier --check` |
| Frontend: Lint | `eslint .` (TypeScript + Svelte) |
| Frontend: Type Check | `svelte-check --tsconfig ./tsconfig.json` |
| Frontend: Tests | `vitest run` |
| Frontend: Build | `vite build` |
| Integration: Build & Migrate | Backend build + DB migrations + server health check + frontend build |
| Security: CodeQL | GitHub CodeQL analysis (main branch only) |

E2E tests run in a separate workflow (`.github/workflows/e2e.yml`) with sharded Playwright tests against a full stack (PostgreSQL, backend, frontend, print-agent).

### E2E Testing Conventions (Playwright, `tests/e2e/`)

- **Behavior tests** (`*-api.spec.ts`) assert against API via `api-driver.ts`. **UI tests** (`*.spec.ts`) keep only genuine UI behavior (labels, navigation, validation messages).
- **`apiAs(request, role)`** returns authenticated `ApiDriver` for seeded roles (`superadmin`/`admin`/`manager`/`cashier`). Returns `ApiResult = { status, ok, body, headers }`.
- **Token rule (critical):** Do not reuse `request` fixture from `beforeAll` inside `test` bodies. Recreate `ApiDriver` per test via `beforeEach`.
- **Login rate-limited** (5/min per IP). Use `getToken(request, role)` / `apiAs` — tokens cached on disk in `fixtures.ts`. Bump `TOKEN_CACHE_VERSION` after backend restart with new `JWT_SECRET`.
- **Browser navigation:** Use explicit `page.goto(\`${FRONTEND_BASE}/...\`)`. Use `waitForAppReady(page)` (not `networkidle`).
- **POS/cart scenarios:** Use `pos-api.ts` helpers for shift/cart flows; raw `apiAs` for entity/CRUD specs.
- **Run from repo root** (`npx playwright test` / `npm run test:e2e`), never from `tests/e2e/`.

## Running Server

```bash
go run cmd/server/main.go
```

A local `.env` is loaded automatically (dev only; production is skipped). The
loader uses `godotenv.Load`, so variables already exported in the shell always
win and `.env` only fills gaps. No `set -a; source .env` needed.

Two consequences worth knowing:

- `ENV` is read from the process environment **before** `.env` is loaded, so
  putting `ENV=production` in `.env` will not disable the loader. Export it in
  the shell for that.
- Settings are cached on startup (`config.Load` is `sync.Once`-guarded), so
  editing `.env` requires a restart.

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
| `-categories=N` | Number of categories (65-100, random if 0) |
| `-stores=N` | Number of stores to generate (random 20-40 if 0) |
| `-warehouses=N` | Warehouses per store (default 1) |
| `-storage-zones=N` | Storage zones per warehouse (default 4) |
| `-storage-racks=N` | Racks per storage zone (default 5) |
| `-stock-opnames=N` | Stock opname sessions to inject (0 = auto ~1/month) |
| `-suppliers=N` | Number of suppliers (random 10-15 if 0) |
| `-consignment=N` | Consignment suppliers (0 = auto, 10-20% of suppliers) |
| `-truncate=false` | Skip truncating existing data |

Re-seeding (`-truncate=false`) continues document sequences and reuses existing products/suppliers/pricing rules, adding new transactions without key collisions. The seeder preserves `must_change_password` on the six system accounts (the e2e workflow unflags its own test users instead).

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
| `007_sale_lookup.sql` | Grants `sale.lookup` to `cashier`/`manager` (code seeded in `000_squash`) |
| `031_revoke_sale_lookup_manager.sql` | Revokes `sale.lookup` from `manager` (cashier-only) |
| `032_sale_detail_and_receipt_print.sql` | Grants `sale.detail`/`receipt.print` to cashier/manager/superadmin |
| `033_audit_log_store_and_immutability.sql` | Adds `audit_logs.store_id` FK + append-only trigger |
| `033b_cash_change.sql` | Adds `sales.change_due` column (returned change on cash over-tender); part of the `033` pair — both apply in lexical order before `034` |
| `034_audit_immutable_bypass.sql` | GUC-aware bypass for audit immutability trigger |
| `035_audit_correlation_id.sql` | Adds `audit_logs.correlation_id` column |
| `036_audit_export_permission.sql` | Grants `audit.export` to superadmin (code seeded in `000_squash`) |
| `037_audit_immutable_fk_bypass.sql` | Allows FK-cascade updates through append-only trigger |
| `038_grant_audit_view_to_admin.sql` | No-op on fresh deploy (legacy `admin` role renamed; manager holds `audit.view`) |
| `039_business_permission_audit.sql` | Business-perspective audit: +12 manager, +4 cashier permissions |
| `040_shift_cash_movements.sql` | Creates `cash_movements` table, grants `shift.cash_movement` (code seeded in `000_squash`) |
| `041_shift_settings.sql` | Seeds `shift_*` keys into `app_settings` |
| `042_consignment_receipt_edit.sql` | Creates `consignment_receipt_edits` append-only audit table |
| `043_product_ownership_type.sql` | Adds `products.ownership_type` (store/consignment) + backfill + view rebuild |
| `044_store_first_and_finance_role.sql` | Seeds default store; renames roles (admin→manager, manager→supervisor, staff→inventory_staff); creates `finance` role; grants supervisor `sale.create`; replaces inventory_staff permissions; backfills `store_id` |
| `045_rename_usernames.sql` | Aligns default usernames to role names (admin→manager, manager→supervisor, staff→inventory_staff) |
| `046_manager_consignment_pay.sql` | Grants `consignment.pay` to `manager` (settle-without-pay bug fix) |
| `047_finance_consignment_view.sql` | Grants `consignment.view` to `finance` (fix: finance could pay settlements but not view them) |
| `048_add_termination_return_reason.sql` | Adds 'termination' to consignment_pending_returns reason check constraint |
| `049_revoke_store_view_finance_supervisor.sql` | Revokes `store.view` from `finance` and `supervisor` (both are store-scoped via JWT; permission was redundant) |
| `050_store_onboarding.sql` | Adds `users.must_change_password` (forced first-login rotation) and revokes `store.create` from `manager` (HQ-only store provisioning) |
| `051_pagination_indexes.sql` | Adds composite `(created_at, id)` indexes on `audit_logs` (incl. partial store-scoped) and `sales` backing keyset pagination |

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
