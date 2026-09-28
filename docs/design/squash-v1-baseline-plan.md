# Migration Squash → Version 1 Baseline (Execution Plan)

Execution plan for `docs/design/Database-Migration-Squashing-Optimization-and-Version-1-Baselining.md`.
That file is the spec; this file is the repo-specific plan derived from inspecting the
actual migration system.

## Decisions (confirmed)

| Decision | Choice |
|---|---|
| Squash scope | **Current working tree**, including uncommitted `052`/`053` and the modified `000_squash`/`040`/`042`/`050` |
| V1 filename | `database/migrations/000_baseline.sql` |
| Temporary databases | New DBs on the existing `localhost:5433` (`postgres-dev`) server, dropped afterwards. The `retail_pos` dev database is never read, written, or reset. |
| Archive format | Single `database/migrations/archive/pre-squash-migrations.tar.gz` |
| Committing | Never auto-commit. User reviews the diff. |

## Findings (Phase 1)

| | |
|---|---|
| Engine | PostgreSQL 18.6 (`postgres:18-alpine`, container `postgres-dev`, `localhost:5433`) |
| Framework | None. Plain `database/migrations/*.sql` applied in **lexical filename order** |
| Active migrations | **32 files**: `000_squash.sql`, `001`–`007`, `031`–`053` (incl. `033b`) |
| Archived (inert) | 47 files in `database/migrations/archive/` — matched by no runner or test glob |
| Scale | 62 tables · 124 index statements · 3 matviews · 4 functions · 4 triggers · 2 extensions · 7 standalone sequences |

### The four runners

All four must keep working after the squash.

| Runner | Reads `schema_migrations` | Writes it | Implication for V1 |
|---|---|---|---|
| `deploy/podman-deploy.sh:369-385` | no | yes, per file (`:381`) | Re-applies **every** file on **every** deploy → V1 must be fully re-runnable |
| `.github/workflows/ci.yml:376-379` | no | no | Fresh DB only |
| `.github/workflows/e2e.yml:63-66` | no | no | Fresh DB only, then runs the seeder |
| `internal/shared/testdb.go:37-117` | **yes** (`:92`) | yes (`:112`) | `if appliedCount == len(files) { return }` — count-sensitive |

### Facts that shape the plan

1. **`000_squash.sql` is already a regenerated baseline** for the 000–030 history. `001`–`053`
   were layered on top. This squash is the second and final pass.
2. **The deploy bootstrap is redundant.** `podman-deploy.sh:351-359` pre-creates `pgcrypto`,
   `invoice_seq` and `schema_migrations`, but `000_squash.sql:10,67` already creates two of the
   three. V1 will be self-sufficient and self-registering.
3. **`app.shift_migration_mode` is dead config.** `podman-deploy.sh:362-368` sets a GUC that the
   only consumer, `archive/026_shift_open_unique.sql`, read. That file is archived. Left alone;
   flagged in the report.
4. **`000_squash.sql:251` claims it "clears stale `00*.sql` tracking rows on each run"** — there is
   no such `DELETE` in the file. The claim is already false and must be corrected, not carried over.
5. **`pg_dump` emits plain `CREATE INDEX`**, losing `CONCURRENTLY` from `051`. Harmless on an empty
   DB; it also lets V1 drop out of the one-statement-per-`Exec` split that `testdb.go:105` needs for
   transaction-safe running.
6. **`crypt('admin123', gen_salt('bf', 14))` is salted per run.** The six seed users' `password_hash`
   will *never* be byte-identical between the legacy and baseline databases. Reference-data
   equivalence must compare `password_hash = crypt('admin123', password_hash)`, never the literal
   hash. This is the one comparison that yields a false positive if done naively.
7. **Stable IDs the app depends on:** `customers.id = 1` (walk-in), `stores.id = 1` (default store),
   plus the `role_permissions` FK pairs. `cmd/dummy/main.go:589` resolves the walk-in by
   `is_walk_in = true` rather than by ID, but `customers.store_id DEFAULT 1 NOT NULL` ties the two.
8. **`customers.store_id` has no FK** (verified — one of the FKs `053` did not restore). That is why
   `000_squash.sql:1278` can insert the walk-in customer before `044` creates the store. Ordering is
   not load-bearing today, but V1 will order store-then-customer for readability.
9. **Reference data inventory** (29 `role_permissions` grant blocks, 7 `permissions`, 6 `users`,
   2 `roles`, 2 `app_settings`, 1 `stores`, 1 `payment_methods`, 1 `customers`, 1 `customer_groups`),
   net of the `DELETE`/`UPDATE` operations in `006`, `031`, `043`, `044`, `045`, `049`, `050`, `052`.
10. **`internal/shared/testdb_test.go:154-172` hard-codes `051_pagination_indexes.sql`** and asserts
    exactly 4 statements with `CREATE INDEX CONCURRENTLY`. Archiving 051 breaks this test.

## Execution steps

### Step 1 — Build the legacy database

Three new empty databases on the `5433` server, all dropped at the end:

```
pos_squash_legacy   ← all 32 migrations, lexical order      (source of truth)
pos_squash_baseline ← V1 only                               (equivalence target)
pos_squash_verify   ← V1 only, from scratch                 (Phase 10 fresh-init test)
```

Replay loop mirrors `ci.yml:376-379` (bootstrap `schema_migrations`, then
`psql -v ON_ERROR_STOP=1 -f` each file in lexical order) with the deploy runner's `PGOPTIONS` for
parity. Any failure stops and is investigated — no skips, no rewrites.

### Step 2 — Generate the baseline

1. `pg_dump --schema-only` (no owner/ACL/subscription noise) from `pos_squash_legacy` → final schema.
2. Classify reference data by **intent**, not by dumping rows:

   **Preserved (required initial/reference data):**
   - 6 roles (incl. `finance.is_system = true` from 052)
   - all permission codes (`000_squash` block + 005/007/032/036/039/040 additions)
   - `role_permissions` grants, net of every `DELETE` in 031/044/046/047/049/050
   - 5 `payment_methods`, 3 `customer_groups`
   - walk-in `customers` id 1, default `stores` id 1
   - 8 `app_settings` keys (5 from 005 + 3 from 041), minus the `default_language` row 006 deletes
   - 6 system users (net of 044/045 renames, 052 email realignment and hardening)
   - the 13 FKs restored by 053, the 051 keyset indexes, the `stores_name_lower_key` index from 052

   **Excluded:**
   - `schema_migrations` rows (bookkeeping; runners record the file)
   - `NOW()`-derived timestamps on seeded rows
   - matview contents (`WITH DATA` over empty `sales` is zero rows)
   - transactional/development data — none exists in a fresh DB, nothing to exclude

3. **Preserve 052's guard semantics verbatim:** `must_change_password = true` only where
   `password_hash = crypt('admin123', password_hash)`. An unconditional update would re-flag
   rotated production accounts on every deploy, because the deploy runner re-applies V1 each run.
4. Use `000_squash.sql`'s existing **natural-key** style
   (`SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = ...`) rather than literal-ID
   dumps. Established convention here, readable, self-heals on re-run, same final state.
5. Structure: `extensions → functions → sequences → tables (constraints inline) → indexes → views →
   matviews → triggers → reference data → ledger cleanup`. Everything `IF NOT EXISTS` /
   `ON CONFLICT DO NOTHING` / `DROP ... IF EXISTS` + `CREATE` (triggers) / `CREATE OR REPLACE`
   (functions). Wrapped in `BEGIN`/`COMMIT` blocks like the current file.
6. **Ledger cleanup** inside V1: delete stale `000_squash.sql` … `053_*.sql` rows from
   `schema_migrations`, then self-register `000_baseline.sql`. Without this, `testdb.go:66`'s
   `appliedCount == len(files)` desyncs on databases that already ran the old chain.
7. **No secrets beyond the existing published seed credential** (`admin123` for the six system
   accounts) — already in `000_squash.sql`, is the documented first-install contract, and is gated
   by `must_change_password`. No hostnames, paths, or temporary database names.

### Step 3 — Prove equivalence (A vs B)

Diff across every category in the spec using normalized `pg_dump` output plus targeted
`information_schema` / `pg_catalog` queries:

schemas · tables · columns (type / nullable / default) · PK/FK/UNIQUE/CHECK · indexes
(`pg_get_indexdef`) · sequences + identity · enums/custom types · extensions · views ·
matviews (`pg_get_viewdef`) · triggers (`pg_get_triggerdef`) · functions
(`pg_get_functiondef`) · reference data per table.

Reference-data comparison normalizes `password_hash` → the `crypt()` boolean and compares seeded
rows by business key rather than timestamp.

Every diff is investigated and classified. Target: **zero unintended differences**.

### Step 4 — Test

- V1 alone on the third virgin DB (`pos_squash_verify`) — fresh-init test
- Backend bootstrap: server starts against a V1-only DB (`/health`)
- `go test ./internal/...` — ~25 packages rebuild from V1 through `shared.RunMigrations`
- `internal/shared/testdb_test.go:154-172` updated (051 no longer exists)
- `internal/archtest/archtest_test.go:290` globs `database/migrations/*.sql` for `knownTables` —
  re-verify it still resolves every table the Go code references
- Per `AGENTS.md`, full suites run in CI; targeted per-package checks locally

### Step 5 — Replace the old files (only after equivalence + tests pass)

- Add `database/migrations/000_baseline.sql`
- Move `000_squash.sql` + `001`–`053` into `database/migrations/archive/pre-squash-migrations.tar.gz`
- Update references: `AGENTS.md` migration table, `deploy/PRODUCTION-DEPLOYMENT.md:239,245,251`
  (correcting the already-false line 251 claim), `deploy/podman-deploy.sh:346,377` comments,
  `cmd/dummy/main.go:468` comment
- Point-in-time audit reports under `docs/audits/` are left alone (they cite line numbers in
  historical files); noted as drift in the report

## Risks

1. **`TestSplitSQLStatements_PaginationMigration`** breaks on 051's removal — must be fixed, not skipped.
2. **Production blast radius:** deploy re-applies V1 to already-migrated DBs on every run. Every
   statement must be re-runnable and no seed may clobber operator-modified data. Highest-severity
   design constraint on the file.
3. **Ledger divergence in prod:** 20 of 32 files never self-register, so prod's ledger is
   incomplete-and-unread while `testdb.go` treats it as authoritative. The squash fixes this (1
   self-registered file), but stale rows must be deleted on first V1 deploy — hence the cleanup step.
4. **WIP baked in:** 052/053 and the first-install feature are uncommitted. V1 is only as reviewable
   as that diff.
5. **Doc drift:** `docs/audits/*` cite `000_squash.sql` line numbers that will no longer resolve.

## Definition of done

A fresh database initialized with only `000_baseline.sql` is structurally and functionally
equivalent to one initialized by the complete 32-file chain, including all required
initial/reference data, with zero unintended differences and passing tests.

## Outcome

**Status: complete.** Everything below ran against PostgreSQL 18.6 on the dev
server (`localhost:5433`). The generated artefacts live in `/tmp/opencode/squash/`
(`transform.py`, `gen_refdata.py`, `assemble.py` are the reproducible toolchain).

### What shipped

- **`database/migrations/000_baseline.sql`** — 5210 lines, one transaction
  (`BEGIN` … `COMMIT`), assembled in three parts:
  1. **Schema** — `pg_dump --schema-only` of the legacy database, rewritten by
     `transform.py` into idempotent form: 61 `CREATE TABLE IF NOT EXISTS`,
     70 `CREATE SEQUENCE IF NOT EXISTS`, 132 `CREATE INDEX IF NOT EXISTS`,
     225 `DO`-guarded `ADD CONSTRAINT`, 5 `CREATE OR REPLACE FUNCTION`,
     3 drop/create trigger pairs, 3 `MATERIALIZED VIEW … WITH DATA`.
  2. **Reference data** — `gen_refdata.py` emits 121 statements covering the 9
     seed tables plus `044`'s `store_id` backfill, all `ON CONFLICT DO NOTHING`.
  3. **Self-check + ledger reconciliation**, deliberately last so a failure
     leaves the old ledger rows intact.
- **`database/migrations/archive/pre-squash-migrations.tar.gz`** — the 32 files,
  byte-identical to their pre-squash content.

### Two defects found and fixed during generation

1. **`transform.py` dropped statement terminators.** The splitter consumed `;`
   and never restored it, so emitted statements ended without one. Fixed by
   re-appending `;` at emit time.
2. **Postgres' deparse of `x IN (…)` is not reparse-stable.** `pg_dump` prints
   `type IN ('a','b')` as `type::text = ANY ((ARRAY[...])::text[])`, but
   reparsing that output yields per-element casts instead of one array cast.
   The predicates are logically identical, yet the schema diff reported them
   as changed. Fixed by emitting the original migration text for
   `cash_movements_type_check` (040:13) and `chk_products_ownership_type`
   (043:18) — the only two constraints affected.

### Equivalence proofs

| Check | Result |
|-------|--------|
| Schema (normalised `pg_dump --schema-only`, legacy vs baseline) | **exact match** — 2119 lines, 0 differences |
| Reference data (379 rows across the 9 seed tables) | **exact match** |
| Row counts, all 60 tables | identical except `schema_migrations` 32 → 1 (the intended reconciliation) |
| Seed credential | all 6 accounts satisfy `password_hash = crypt('admin123', password_hash)`; 6 distinct salts, no literal hash in the file |
| Idempotency | 3 consecutive applies, rc=0, no errors, ledger stays at 1 row |
| Upgrade path | applied to the already-migrated legacy DB: 32 ledger rows → 1, schema unchanged (0 diffs), data unchanged |
| Bare `psql -f` (no bootstrap) | applies cleanly |
| `go test ./internal/shared ./internal/archtest` | pass |

Excluded from comparison **by design**: `created_at` / `updated_at` /
`last_login` (clock-derived), `password_hash` (per-install random salt),
materialised-view contents (empty in both).

### References updated

- `internal/shared/testdb_test.go` — `TestSplitSQLStatements_PaginationMigration`
  hard-coded `051_pagination_indexes.sql` (risk 1 above). Replaced with an
  inline fixture, `TestSplitSQLStatements_ConcurrentIndexAndRegistration`, so
  `splitSQLStatements` still has coverage for `CREATE INDEX CONCURRENTLY` (now
  including the `CREATE UNIQUE INDEX CONCURRENTLY` shape); added
  `TestSplitSQLStatements_BaselineMigration` for the new file.
- `deploy/podman-deploy.sh` — two comments, one of which already claimed
  `000_squash` cleared `00*.sql` tracking rows (false before this task).
- Comments in `cmd/dummy/main.go` and `internal/customer/repository_test.go`.
- `README.md`, `AGENTS.md`, `deploy/PRODUCTION-DEPLOYMENT.md`,
  `docs/guides/first-time-installation.md`.

Left alone as historical records: `docs/adr/*`, `docs/audits/*`, `.kilo/plans/*`,
and `docs/design/*` other than this file — they cite line numbers in the
archived migrations, which is noted as drift (risk 5) rather than rewritten.

### Known limitations

- `app.shift_migration_mode` in `deploy/podman-deploy.sh:362-368` remains dead
  configuration: its only consumer, `026_shift_open_unique.sql`, was already
  archived before this task. Untouched.
- `ci.yml:374` and `e2e.yml:64` bootstrap `schema_migrations` as
  `(filename TEXT PRIMARY KEY)` with no `applied_at`, while `deploy` and
  `testdb.go` create `(filename VARCHAR(255), applied_at TIMESTAMPTZ ...)`.
  This pre-existing inconsistency is unchanged: `CREATE TABLE IF NOT EXISTS`
  means whichever runner bootstraps first wins, and `000_baseline.sql` only
  ever inserts `filename`, so both shapes work.
