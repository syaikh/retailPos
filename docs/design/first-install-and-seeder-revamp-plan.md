# First-Install Bootstrap & Seeder Revamp Plan

Status: implemented (deliverables A–F complete)
Depends on: migration `050_store_onboarding.sql` (shipped), `StoreOnboardingWizard.svelte` (shipped)

## Context

Migration `050` delivered two prerequisites of the store onboarding wizard
(`docs/design/store-onboarding-wizard.md`):

1. `users.must_change_password` — enforced as HTTP 428 on every protected route
   except the change-password/logout/validate allowlist
   (`internal/middleware/auth.go`).
2. `store.create` revoked from `manager` — store provisioning is HQ-only
   (superadmin retains it).

What is still missing is anything that bootstraps a **first-time install**, and
the dummy seeder predates store scoping entirely.

### First-run state today

`deploy/podman-deploy.sh migrate` bootstraps SQL prerequisites only
(`pgcrypto`, `invoice_seq`, `schema_migrations`). It creates no first user and
no first store. After migrate:

| Aspect      | State |
|-------------|-------|
| Accounts    | 6 seeded users (`000_squash.sql:1637-1667`), all password `admin123`, all `must_change_password = false` |
| Store       | one `'Default Store'` (`044_store_first_and_finance_role.sql:15-17`), address + phone `'0000000000'` (both non-blank → no `store.address` / `store.phone` blockers) |
| Staff       | 5 of 5 `RequiredRoles` backfilled to Default Store by `044:101-105` |
| Locations   | none → `storage_location` blocker |
| Catalog     | empty → `catalog` blocker |

So Default Store is *almost* ready: only `storage_location` and `catalog`
remain. Store onboarding is a **post-login** superadmin modal — it provisions a
*second* store, not the first install.

### The install hole

The six seed accounts are not flagged `must_change_password`, and the forced
rotation modal is the **only** in-UI rotation entry point. Therefore no account
reachable right after install has a supported rotation path — including the
seeded `superadmin`. `README.md:730` claims a self-service screen exists;
`README.md:1988` admits it does not.

### The seeder gap

`cmd/dummy/` (5,906 lines, zero `internal/` imports, coupled to the schema only
through hardcoded column lists and role-name strings) predates store scoping.

**Break 1 — stores 2..N are permanently "not ready" (biggest).**
`000_squash` seeds 5 roles on store 1; `main.go:469-476` (bulk
`UPDATE users SET store_id … WHERE store_id IS NULL`) and `ensureCashierUsers`
(`main.go:2766`) pin everything to the first active store. Locations *are*
per-store (`main.go:1112-1125`) and the catalog is global, so staff is the only
gap — stores 2-40 each show 5 `staff.*` blockers in the Stores page.

**Break 2 — bulk sales write `store_id = NULL`** (`main.go:2125`, `2271`). Every
MV groups by `store_id` and every report filters
`store_id IS NULL OR store_id = $N` (`internal/report/repository.go:140, 173,
255, 399, 448, 479`), so all 10-20 seeded sales/day count in *every* store
simultaneously. `daily.go:248-256` already assigns a store correctly.

**Break 3 — the `users` restore silently strips new columns.** The snapshot
SELECT (`main.go:675`) and restore INSERT (`main.go:766`) enumerate 8 columns
and omit `must_change_password`. Harmless today; the moment `052` flags the seed
accounts, a dev seed wipes the flag and the forced-rotation flow disappears.

**Break 4 — `supplier_seq` never resynced.** The seeder emits `SUP-%03d`
(`main.go:1335`) while the app emits `SUP-%06d`
(`internal/supplier/repository.go:30-37`). Unlike the other 10 sequences the
seeder resyncs, this one is skipped, so a post-seed UI supplier gets a stale
high-water mark.

## Decisions

| Decision | Choice |
|----------|--------|
| First-run approach | Migration + runbook + self-service password screen. No new `POST /api/setup` surface. |
| Seeder scope | Breaks 1-4 only; secondary debt stays documented, not fixed. |
| Adjacent defects in scope | `050` rollback, `manager` + NULL `store_id`, `finance` `is_system`, stale emails, `stores.name` uniqueness, stop truncating `users`. |
| Out of scope | Role-name indirection refactor, `audit_logs.store_id` backfill, `store_id=1` pricing-rule hardcode, `stockopname.go` scope id hardcodes, `-daily.no-audit` flag fix, `-daily` shift creation, `./dummy` binary removal, `getIDs` `LIMIT 200`. |

## Deliverables

### A. `database/migrations/052_first_install_hardening.sql`

Single migration, `BEGIN`/`COMMIT`, with the required `schema_migrations` insert
and a rollback block (rollback must be the true inverse, not a repeat of the
action — the `050` rollback got this wrong, see B).

Ordered so each step is independently safe.

**A1. Force rotation on accounts still holding the seed password.**

```sql
UPDATE users SET must_change_password = true
WHERE username IN ('superadmin','manager','supervisor','cashier','inventory_staff','finance')
  AND password_hash = crypt('admin123', password_hash);
```

`pgcrypto`'s `crypt(password, existing_bcrypt_hash)` reuses the existing salt,
so the comparison re-verifies against `admin123` exactly. Accounts that already
rotated fail the comparison and are untouched; re-runs are no-ops. This targets
the seed hash rather than the username list alone, so an account named `manager`
that a user already re-secured is not re-flagged.

**A2. `finance` role → `is_system = true`.**

`044:38-40` inserts `finance` without `is_system`, so legacy-upgraded
deployments can edit/delete the role while fresh installs (which get it from
`000_squash:1250-1257` with `is_system = true`) cannot.

**A3. Realign stale emails on upgraded DBs.**

`045_rename_usernames.sql` renames `username` but never `email`, leaving
`manager` → `admin@retailpos.local`, `supervisor` → `manager@retailpos.local`,
`inventory_staff` → `staff@retailpos.local`. `users_email_key UNIQUE (email)`
(`000_squash.sql:223`) makes this collision-sensitive — each update needs a
`NOT EXISTS` guard so the migration aborts only on real conflicts rather than
assumes the target address is free:

```sql
UPDATE users SET email = 'manager@retailpos.local'
WHERE username = 'manager' AND email = 'admin@retailpos.local'
  AND NOT EXISTS (SELECT 1 FROM users x WHERE x.email = 'manager@retailpos.local');
```

Repeat for `supervisor` → `manager@retailpos.local` and `inventory_staff` →
`staff@retailpos.local`.

**A4. Backfill store-less managers.** Data fix for accounts already stranded by
the `OperationalRoles` gap (Deliverable C). Same shape as `044:101-105`, scoped
to `manager` only, with an `EXISTS (SELECT 1 FROM stores …)` guard so an empty
`stores` table is a no-op rather than an error:

```sql
UPDATE users u SET store_id =
  (SELECT s.id FROM stores s WHERE s.is_active ORDER BY s.id LIMIT 1)
WHERE u.store_id IS NULL
  AND u.role_id = (SELECT id FROM roles WHERE name = 'manager')
  AND EXISTS (SELECT 1 FROM stores s WHERE s.is_active);
```

**A5. `stores.name` uniqueness.** Use a **unique expression index on
`LOWER(name)`**, not a plain `UNIQUE` constraint, to match the existing lookup
semantics at `internal/store/repository.go:248-260`
(`WHERE LOWER(name) = LOWER($1)`); a case-sensitive constraint would allow
`Toko A` / `toko a` while `GetByName` treats them as one.

```sql
CREATE UNIQUE INDEX IF NOT EXISTS stores_name_lower_key ON stores (LOWER(name));
```

Pre-check for duplicates first and `RAISE EXCEPTION` listing the offending
names — a silent rename would corrupt operational data. Consider `'Default
Store'` from `044:15-17` in that pre-check. `CONCURRENTLY` is unavailable
(migrations run in a transaction block); the table is small so a plain index is
fine.

### B. Fix `050`'s rollback block

`database/migrations/050_store_onboarding.sql:38-42` re-`DELETE`s the `manager`
`store.create` grant it is supposed to restore. Replace with an
`INSERT … SELECT` joining `roles`/`permissions`.

### C. `manager` + NULL `store_id` (code)

Add `RoleManager: true` to `permissions.OperationalRoles`
(`internal/permissions/permissions.go:143-148`). `internal/user/handler.go:199-202`
then rejects a store-less manager at creation instead of minting an account that
403s from `RequireStoreID` (`internal/middleware/auth.go:181-185`) on every
protected route — including `/api/stores`, so it cannot self-repair.

While editing: confirm whether `UpdateUser` has the same gap as `CreateUser`,
and confirm nothing else reads `OperationalRoles` in a way that adding `manager`
would break.

### D. Self-service change-password screen (frontend)

Closes the install hole: the seeded `superadmin` is not flagged, so
`ForceChangePasswordModal` never appears for it.

- New route `/account/password` in `main.svelte`'s `pageModules` map.
  **No `routePermissions` entry** — `main.svelte:175-179` returns `true` for
  unmapped paths, so any authenticated user reaches it.
- New component `web/src/modules/user/components/ChangePasswordPage.svelte`,
  posting to the existing `POST /api/change-password` (registered at
  `cmd/server/main.go:173`, on the 428 allowlist).
- Extract the submit logic and validation from
  `ForceChangePasswordModal.svelte:18-54` (8-char minimum, matching confirm)
  into a shared form component both consume, rather than duplicating it.
- Entry point in `Sidebar.svelte`. **Verify** that the
  `routePermissions[href]` branch at `Sidebar.svelte:99` renders items absent
  from the map, following the `?? routePermissions[basePath(path)]` pattern in
  `NotificationBell.svelte:77`.
- i18n labels through the existing `labels` object.

### E. Seeder revamp — `cmd/dummy/` (breaks 1-4)

**E1. Stop truncating `users`.** Replace the snapshot/restore block
(`main.go:662-698`, `756-772`) with a delete of non-system users:

```go
DELETE FROM users
WHERE username NOT IN ('superadmin','manager','supervisor','cashier','inventory_staff','finance')
```

This eliminates the whole class of bug Break 3 is an instance of (any future
`NOT NULL`/flagged column added to `users` no longer needs a matching change
here), and stops the seeder from stripping the `052` flag — which the current
code would silently revert. Remove the now-dead `sysUser` struct and the
`users_id_seq` resync at `main.go:775` (the sequence is untouched when rows are
not inserted with explicit ids). Keep the store-id backfill at `main.go:782-789`
only if it still serves a purpose after E2, otherwise it becomes redundant.

**e2e consequence (plan gap, resolved):** migration `052` flags `superadmin`,
the seeder now preserves the flag, and the 428 gate blocks every protected API
call for a flagged token — so `.github/workflows/e2e.yml` clears the gate in a
step after seeding (the workflow has no in-UI rotation path). The gate's own
behaviour stays covered by `force-password-change.spec.ts`, which flags its own
user. A local e2e run needs the dev superadmin rotated once (the intended
first-login flow) before the suite can authenticate.

**E2. Per-store staff distribution.** Today `main.go:469-476` and
`ensureCashierUsers` (`main.go:2766`) pin every account to the first active
store. Distribute seeded system users and dummy cashiers across stores, and top
up each store to ≥1 of all five `RequiredRoles`
(`internal/store/service.go:17-23`: `manager`, `supervisor`, `cashier`,
`inventory_staff`, `finance`) so readiness is demonstrable on every store. This
is the single biggest realism gap now that readiness is a visible UI signal.

**E3. `store_id` on bulk sales.** `main.go:2125` and `2271` write literal
`NULL`. Assign the cashier's store, falling back to a random active store —
`daily.go:248-256` already implements this shape (`storeIDForSale`).

**E4. Resync `supplier_seq`.** Add the resync next to the existing sequence
resyncs (`sku_seq` `main.go:502-506`, `invoice_seq` `main.go:1925`,
`stores_id_seq` `main.go:937`/`982`, …) so a post-seed UI supplier does not draw
a stale high-water mark.

### F. Documentation

**New `docs/guides/first-time-installation.md`** — the runbook:

1. `podman-deploy.sh migrate`
2. Log in as `superadmin` / `admin123`
3. The 428 forced-rotation gate fires on the next protected request; rotate
4. Complete Default Store: create a storage location, then load the catalog
   (address, phone and 5/5 staff already exist)
5. Verify via `GET /api/stores/1/readiness`
6. Change `superadmin` again through `/account/password` once provisioned
   (or note that step 3 covers it)

**Correct collateral:**

| File | Issue |
|------|-------|
| `README.md:696-704` | Migration list stops at `039`; omits `040`-`052` |
| `README.md:698, 723-728` | "5 default users" / "85 permissions" (actually 6 / 86); credential table omits `finance` |
| `README.md:730` vs `:1988` | Contradict each other on self-service change password — both become true after D |
| `README.md:1973-1980` | Pre-044 role names in the role dropdown and Administration section |
| `README.md:1963-1969` | Store section describes only Add/Edit/Delete; no wizard or readiness |
| `README.md:402-409` | Stores endpoint table omits `GET /api/stores/:id/readiness` and `GET /api/warehouses` |
| `deploy/PRODUCTION-DEPLOYMENT.md:151-155` | Publishes `superadmin`/`admin123` as the production login with no warning |
| `deploy/PRODUCTION-DEPLOYMENT.md:230-246` | Stale migration list; "They do not create stores" is wrong (`044` inserts one) |
| `AGENTS.md` Deployment | Claims the server validates permission codes at startup — no such validation exists. Correct the text (do not implement it). |
| `AGENTS.md` | Add `052` to the migration table; add `seed-dev.sh`'s undocumented flags if E touches them |
| `docs/design/store-onboarding-wizard.md:1-5` | Status still says "planned / in implementation" although the wizard ships |

## Discovered during implementation (documented, not fixed)

**Pre-existing: `-products=N` + `-truncate=false` crashes.** `generateSKU`
(`cmd/dummy/main.go`) is deterministic in the loop index, so a re-seed with a
non-zero product count regenerates colliding SKUs. The duplicate-sku skip
(`main.go:1715-1724`) issues a plain `INSERT`, and PostgreSQL aborts the
transaction on the first unique violation — the next statement then fails with
`25P02` and the worker hard-fails ("transaction poisoned"). The skip can only
work when the collision lands on a batch's final statement. Independent of this
plan's changes (nothing here touches product injection); the documented re-seed
path is `-products=0`, which reuses existing products via
`getExistingProducts`. A real fix needs either a savepoint per insert, an
`ON CONFLICT DO NOTHING` outside replica mode, or a SKU counter that starts
past the existing maximum (like `invoice_seq`).

## Sequencing

`B` → `A` → `C` → `E` → `D` → `F`.

- `A` before `C` so the stranded-manager backfill lands with the rest of the
  data corrections in one migration.
- `E` after `A` — otherwise a dev seed run between the two reverts the `052`
  flag (which is exactly Break 3).
- `D` before `F` so the runbook can reference the new screen as existing.

## Verification

Per `AGENTS.md`, do **not** run full lint/test/build suites locally — CI
performs them. Targeted checks only when explicitly requested:

- Migration: apply against a fresh DB and a legacy (pre-`044`) DB; confirm the
  `crypt('admin123', …)` predicate flags exactly the un-rotated seed accounts.
- `stores_name_lower_key`: seed duplicate names and confirm the `RAISE
  EXCEPTION` pre-check fires before the index does.
- `CreateUser` with role `manager` and `store_id = null` → 400.
- Seeder: `./seed-dev.sh` twice — first `-truncate=true`, then
  `-truncate=false` — and confirm stores 2..N report `ready: true`, seeded
  sales are partitioned by store in per-store reports, and
  `must_change_password` survives the truncating run.
- Frontend: log in as an unflagged account and rotate via
  `/account/password`; confirm the session stays valid (token re-issued by
  `ChangePassword`).
