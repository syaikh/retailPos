# Dev Database Fresh-Install Reset — Plan

**Status:** Implemented and verified against `postgres-dev` on 2026-09-29.
**Date:** 2026-09-29
**Related docs:**
- `docs/design/first-install-and-seeder-revamp-plan.md` — implemented (A–F). Owns the
  first-install *runbook* and the seeder revamp. This plan owns the *reset path* and
  does not overlap it.
- `docs/design/squash-v1-baseline-plan.md` — the squash that produced `000_baseline.sql`.
- `docs/guides/first-time-installation.md` — the runbook this reset reproduces.

## Context

There is no way to return the development database to first-install state.

`make clean` (`Makefile:128-133`) is the only existing full reset, and it stops and
removes the **deploy** stack's volume (`retail-pos-postgres-data`) — not the
`postgres-dev` container the dev workflow actually uses. `./seed-dev.sh` is not a
reset either: `truncateAllData` (`cmd/dummy/main.go:652-750`) truncates 40 tables but
deliberately preserves `users` (ids, `store_id`, `must_change_password`) and never
touches `roles`, `permissions`, `role_permissions`, or `app_settings`. Twelve tables —
`cart_sessions`, `cart_items`, `cash_movements`, all four `import_*` tables,
`dead_letter_events`, `stock_opname_recount_requests`, `stock_opname_session_scopes`,
`app_settings` — survive it entirely.

The live `postgres-dev` database has drifted far enough that it is no longer on the
baseline at all:

| | Current | First install |
|---|---|---|
| `schema_migrations` | **24 legacy rows**, `000_squash.sql` → `052_first_install_hardening.sql` (applied 2026-08-14 → 09-27). **No `000_baseline.sql` row** | 1 row: `000_baseline.sql` |
| `stores` | 8 — *Jadi Baru Ngadirejo* … *Jadi Baru Sumowono* | 1: *Default Store* |
| `users` | 46 (ids 7–46 are test/dummy residue) | 6, ids 1–6 |
| `roles` | 50 | 6 |
| `role_permissions` | 297 | 264 |
| `customer_groups` | 8 | 3 |
| `app_settings` | 8 (incl. `logo_path`, `store_name = 'Sahabat Mart'`) | 7 |
| `finance.role_id` | **209** | 6 |
| `superadmin.must_change_password` | `false` | `true` |
| `sales` / `sale_items` | 166 / 410 | 0 / 0 |
| `audit_logs` / `import_jobs` / `dead_letter_events` | 379 / 97 / 24 | 0 / 0 / 0 |
| `mv_daily_sales`, `mv_hourly_sales`, `mv_dashboard_totals` | populated | empty |

`internal/shared/testdb.go:70-79` explains how the ledger got there: it adopts a
pre-migrated database that has no `000_baseline.sql` entry by bulk-inserting every
filename it finds on disk. The 24 rows are a legacy chain that predates the squash.

### Why re-migrating is not enough

`000_baseline.sql` is **purely additive**:

- 123 `ON CONFLICT … DO NOTHING` guards — the seed inserts never overwrite an
  existing row.
- Forward-only sequence resync (`:5069-5134`) — `setval` never moves a sequence back.
- Exactly one `DELETE` in the whole file, at `:5242`, and it touches **only**
  `schema_migrations` (the ledger reconciliation block, `:5213-5277`).
- No `DROP TABLE`, no `DROP SCHEMA`, no `TRUNCATE`.

So replaying it against the current database would clean the ledger and change
nothing else. All 8 stores, 46 users, 50 roles, 297 grants, 8 customer groups, 166
sales, `logo_path`, and the renamed *Jadi Baru* stores would survive, and the result
would still not match a fresh install.

**A true fresh install requires dropping and recreating the database.** The
reproduction path is identical to the documented install
(`docs/guides/first-time-installation.md:12-26`): drop → create → bootstrap → apply
`000_baseline.sql` → verify.

## Decisions

| Decision | Choice |
|---|---|
| Reset mechanism | Drop + recreate the database, then replay `database/migrations/*.sql`. Not re-migrate; not truncate. |
| Scope | **Dev only** — `postgres-dev` container, `retail_pos` database. The `retail-pos-postgres-data` and `pgdata_retail` volumes are never named. |
| Deliverable | A committed, repeatable script (`scripts/reset-dev-db.sh`) plus a `make db-fresh` target — not a one-off command. |
| Backup before reset | None. The dev data is disposable by decision. |
| Post-reset depth | **Empty baseline only.** No `./seed-dev.sh` re-run. Readiness should report exactly the two blockers a real install hits. |
| UI / browser state | Reset is out of the script's reach. The script prints the manual browser + backend steps as a runbook. |
| Guard strictness | Hard refusal, no override flag: `ENV=production`, non-`localhost` host, wrong database name, or a non-`postgres-dev` server all exit 1. |

## Deliverables

### A. `scripts/reset-dev-db.sh` (new)

Style-matched to `scripts/kill-port.sh` and `scripts/lint-rbac.sh`: `#!/bin/bash`,
`set -euo pipefail`, flag parsing, sourced `.env`.

**Flags**

| Flag | Default | Meaning |
|---|---|---|
| `--yes` | off | Skip the interactive `RESET` confirmation |
| `--keep-uploads` | off | Do not clear `uploads/logos/*` |
| `--reseed` | off | Run `./seed-dev.sh` after verification. **Not a fresh install** — the seeder recreates `stores`, `payment_methods`, and `customer_groups` with new ids, so `stores(id=1)` and `customers(id=1)` drift off the invariant the baseline self-check asserts. |

**Safety guards (hard refuse, no override)**

- `ENV=production` → exit 1
- `DB_HOST` ≠ `localhost` → exit 1
- `DB_NAME` ≠ `retail_pos` → exit 1
- Resolved port ≠ `DATABASE_PORT` (5433), or the container serving that port is not
  `postgres-dev` → exit 1
- Prints the drift table above and requires typing `RESET` unless `--yes`

**Steps**

1. **Preflight** — `pg_isready`; confirm `retail_pos` exists and is owned by `pos`;
   dump current row counts to stdout for the record.
2. **Drop** — `psql -d postgres -c "DROP DATABASE retail_pos WITH (FORCE)"`.
   `WITH (FORCE)` (PostgreSQL 13+; server is 18.3) evicts the running backend's pooled
   connections. `retail_pos_test`, `retail_pos_seeder_test`, and `pos_squash_verify`
   are never named.
3. **Recreate** — `CREATE DATABASE retail_pos OWNER pos ENCODING 'UTF8'
   LC_COLLATE 'en_US.utf8' LC_CTYPE 'en_US.utf8' TEMPLATE template0`. Read the
   current encoding and collation from `pg_database` rather than hard-coding.
4. **Bootstrap** — mirror `deploy/podman-deploy.sh:350-359`: `pgcrypto`,
   `invoice_seq`, `schema_migrations`. The baseline creates all three itself
   (`:46`, `:50`, `:907`, `:1320`), so this is belt-and-braces parity with the
   production runner rather than a requirement.
5. **Migrate** — lexical loop over `database/migrations/*.sql` (non-recursive, so
   `archive/` is never touched) with `psql -v ON_ERROR_STOP=1 -f`, plus
   `INSERT INTO schema_migrations ON CONFLICT DO NOTHING` per file — exactly
   `deploy/podman-deploy.sh:369-383`. Replaying rather than ledger-checking matches
   all three non-test runners and the replay contract in `AGENTS.md`.
6. **Verify** — the baseline's own self-check (`:5138-5211`) runs inside the migration
   transaction and `RAISE EXCEPTION`s on incomplete seed, so a clean `psql` exit
   already proves the seed landed. The script then asserts the *fresh-install* shape
   the self-check deliberately does not test (it uses `<` not `=` to tolerate a
   database that was already in use):

   - `schema_migrations` = 1 row, named `000_baseline.sql`
     (required by `internal/shared/testdb.go:90-114`, which asserts
     `len(schema_migrations) == len(files)`)
   - `roles` = 6, `permissions` = 86, `role_permissions` = 264, `payment_methods` = 5,
     `customer_groups` = 3, `app_settings` = 7
   - `stores(id=1).name = 'Default Store'`; `customers(id=1).is_walk_in`
   - all six accounts: `must_change_password = true` **and**
     `password_hash = crypt('admin123', password_hash)`
   - zero rows in `sales`, `sale_items`, `shifts`, `audit_logs`, `import_jobs`,
     `import_rows`, `import_errors`, `import_snapshots`, `dead_letter_events`,
     `cart_sessions`, `cart_items`, `cash_movements`, `inventory_movements`,
     `inventory_adjustments`, `purchase_orders`, `goods_receipts`, `stock_opnames`,
     and the twelve `consignment_*` tables
   - `count(*) < 2` for `stores` and `customers`

7. **Clear uploads** — `rm -f uploads/logos/*`. `app_settings.logo_path` dies with
   the database, leaving the on-disk file orphaned. The directory is created solely
   by `internal/appsettings/handler.go:55-66` and is gitignored (`.gitignore:112`).
8. **Print the runbook** — see below.

### B. `Makefile`: `db-fresh`

Added next to the existing `db-backup` / `db-restore` / `db-shell` targets
(`Makefile:114-125`) and listed in `make help` (`:9-21`):

```make
db-fresh: ## Reset the dev database to first-install state (DESTROYS ALL DATA)
	@chmod +x scripts/reset-dev-db.sh
	./scripts/reset-dev-db.sh
```

### C. Documentation

| File | Change |
|---|---|
| `docs/guides/first-time-installation.md` | New *Resetting to first-install state* section. The runbook currently implies re-migrating is the install path (`:12-26`); it needs the explicit warning that the baseline is additive and that a reset requires a drop. |
| `AGENTS.md` | One line beside the seeder table pointing at `./scripts/reset-dev-db.sh` / `make db-fresh`, plus the note that `seed-dev.sh` is **not** a reset. |
| `README.md:665` | Add `make db-fresh` to the Makefile command list. |
| `docs/design/first-install-and-seeder-revamp-plan.md` | Cross-reference this plan from the header, so the two first-install documents are discoverable from each other. |

## Execution (this reset)

1. `make db-fresh` → drop, create, bootstrap, migrate, verify, clear uploads.
2. **Restart the backend.** The 10-minute Ristretto caches
   (`internal/wiring/wiring.go:419-448` — `user:*`, `product:*`, `categories:list`,
   `dashboard:stats`, `dashboard:live`, `report:*`) and the 60-second branding cache
   (`internal/appsettings/handler.go:105-133`) survive a database swap. In an existing
   `run-dev.sh` session: press `r` + Enter.
3. **Clear browser state for `localhost:5173`.** The reset cannot reach browser
   storage. The `refresh_token` HttpOnly cookie
   (`internal/user/auth_handler.go:91-95`) points at a deleted `refresh_tokens` row, and
   `sessionStorage.access_token` was minted against the old user set. DevTools →
   Application → Storage → *Clear site data*, or in the console:
   ```js
   localStorage.clear(); sessionStorage.clear();
   ```
   This also resets `pos.theme` → `dark` and `pos.locale` → `en` (frontend defaults at
   `web/src/shared/utils/theme.svelte.ts:9-15` and
   `web/src/shared/i18n/index.svelte.ts:16-22`), and `pos.printConfig` back to
   `VITE_PRINT_AGENT_URL`. Note the frontend defaults disagree with the seeded account
   preferences (`light` / `id`, `000_baseline.sql:4984+`) — first paint follows the
   browser, not the account. That is pre-existing behaviour, not a reset artifact.
4. **Delete the e2e token cache** — `rm -f /tmp/retail-pos-e2e-tokens.v1.json`
   (`tests/e2e/fixtures.ts:176-190`), or the next Playwright run reuses dead tokens.
5. **Verify the first-install experience** — sign in as `superadmin` / `admin123`, expect
   the HTTP **428** password dialog (`internal/middleware/auth.go:23-55`), then confirm
   `GET /api/stores/1/readiness` returns `ready: false` with exactly two blockers:
   `storage_location` and `catalog` (`internal/store/service.go:174-225`).

## Sequencing

`A` → `B` → run it → `C`.

Documentation last, so `C` can describe the tool as it actually behaves.

## Verification

Per `AGENTS.md`, do **not** run full lint/test/build suites locally — CI performs
them. Targeted checks only when explicitly requested.

Executed on 2026-09-29 against `postgres-dev` / `retail_pos`:

- [x] Guards refuse `ENV=production`, `DB_NAME=retail_pos_test`, a non-`localhost`
      `DB_HOST`, a port no container publishes, and an unknown flag — each exits 1
      without touching anything.
- [x] First run exits 0 and every assertion in `A6` passes.
- [x] Second run is idempotent: same verified state, exit 0.
- [x] All 51 non-seeded tables report 0 rows; `mv_daily_sales`, `mv_hourly_sales`,
      and `mv_dashboard_totals` are empty.
- [x] `schema_migrations` holds exactly one row, `000_baseline.sql`.
- [x] Six accounts, ids 1–6, `must_change_password = true`, `admin123` verifies
      against all six. `finance.role_id` is back to 6 (was 209).
- [x] One `Default Store`, one walk-in customer, seven `app_settings`
      (`logo_path` gone, `store_name` back to `RetailPOS`).
- [x] `retail_pos_test`, `retail_pos_seeder_test`, and `pos_squash_verify` still
      exist afterwards.
- [x] `uploads/logos/logo.png` cleared by default; the sibling database volumes
      were never named.
- [x] `POST /api/login` as `superadmin` / `admin123` returns 200, and
      `GET /api/stores` returns **428 `PASSWORD_CHANGE_REQUIRED`** — the
      first-install gate fires.
- [x] Readiness preconditions confirmed at the DB level (`storage_locations` = 0,
      `products` = 0, 5/5 required staff, store active, address/phone non-empty),
      which yields exactly the two expected blockers. The endpoint itself sits
      behind the 428 gate by design.
- [ ] `go test -p 1 -count=1 ./internal/shared/...` — not run; `testdb.go` requires
      `len(schema_migrations) == len(files)`, which the verification above
      confirms independently.

Manual steps still owed to the operator: restart the backend, clear browser site
data, delete the Playwright token cache. See *Execution* above.

## Incidental findings (all fixed in this changeset)

Found while scoping the reset, not part of its design:

- `uploads/logos` lived in the container's writable layer with no `VOLUME` and no
  mount — **logos were lost on every image rebuild**, and worse, `deploy/backend/Dockerfile`
  set no `WORKDIR` in the runtime stage, so the relative path in
  `internal/appsettings/handler.go:61` resolved to `/uploads/logos`, which uid 1000
  cannot create (`MkdirAll` failure is only a `slog.Warn`, so startup looked healthy
  and `POST /api/settings/logo` 500'd). Now: `WORKDIR /app` with a pre-created,
  correctly-owned `/app/uploads`, and the named volume `retail-pos-uploads` mounted in
  `deploy/podman-deploy.sh`, `deploy/docker-compose.yml`, the Quadlet unit, the manual
  runbook, the backup procedure, and `make clean`.
- `STOCK_MINIMUM` was in `.env` only (never `.env.example`) and read nowhere in the
  codebase. Removed. `STOCK_WARNING_THRESHOLD` / `STOCK_CRITICAL_THRESHOLD` are read
  at `internal/config/config.go:205-206` and were left alone.
- `backups/` was not in `.gitignore` even though `make db-backup` (`Makefile:116-119`)
  writes there. Added — those are plain SQL dumps and can contain customer data.

## Still out of scope (documented, not fixed)

- `database/migrations/archive/001_create_tables.sql` contains the repo's only
  `DROP TABLE … CASCADE` × 8 and is a standing footgun. A comment banner marking it
  never-executable would help.
