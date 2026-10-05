# First-Time Installation Runbook

Bootstraps a fresh deployment from an empty database to a ready store. There
is no scripted setup endpoint — the install flow is **migrate → log in →
rotate the seeded password → finish the Default Store → verify readiness**.

Prerequisites: the stack is deployed per
[`deploy/PRODUCTION-DEPLOYMENT.md`](../../deploy/PRODUCTION-DEPLOYMENT.md)
(PostgreSQL, backend, frontend, print agent) and `JWT_SECRET` is set — the
server refuses to start without it.

## 1. Apply migrations

```bash
./deploy/podman-deploy.sh migrate
```

On a fresh database this bootstraps `pgcrypto`, `invoice_seq`, and the
`schema_migrations` table, then applies every file in `database/migrations/`
sequentially with `ON_ERROR_STOP=1` (currently just `000_baseline.sql`, the
squashed Version 1 baseline). Migrations are **not** run automatically on
server start, and must be applied **before** deploying a new server binary.

Result: full schema + reference data — 6 roles, 86 permissions, grants, the 6
default users (below), payment methods, customer groups, and the placeholder
**Default Store**.

## 2. Log in

Open the frontend and sign in with a seeded account:

| Username | Password |
|----------|----------|
| `superadmin` | `admin123` |
| `manager` | `admin123` |
| `supervisor` | `admin123` |
| `cashier` | `admin123` |
| `inventory_staff` | `admin123` |
| `finance` | `admin123` |

All six are development defaults. Migration `052` flags every one of them
`must_change_password = true`, so treat the whole list as a bootstrap credential
that dies in the next step.

## 3. Rotate the forced password (428 gate)

Because the accounts are flagged, the first protected request after login is
answered with **HTTP 428** and the SPA blocks behind a *Change Your Password*
dialog (min. 8 characters, confirm must match). Rotating re-issues the access
token with the claim cleared — the session continues without re-login.

Wrong current password → 401 and the gate stays up.

> On a legacy database that predates `052`, no gate fires. Rotate the seeded
> passwords manually in step 6 instead of skipping it.

## 4. Finish the Default Store

Migration `044` seeded **Default Store** with placeholder address/phone and
assigned all default users to it, so staff is already complete (5/5 required
roles). Two readiness blockers remain:

1. **Storage location** — go to *Storage Locations* and add at least one
   location for Default Store (code + name).
2. **Catalog** — import the product list with stock via
   *Inventory → Products → Import CSV* (templates under `docs/examples/`).
   The blocker clears once at least one active product has stock.

Also replace the placeholder store address and phone (Stores → Edit). They are
*not* what blocks readiness: the check only fires `store.address` / `store.phone`
when the field is **empty**, and the baseline seeds both with non-empty
placeholders. Readiness is gated by `storage_location` and `catalog` alone.

Stores added later should go through the **Stores → Tambah Toko** onboarding
wizard instead: Details → Staff → Location → Stock → Readiness, one step at a
time.

## 5. Verify readiness

`GET /api/stores/:id/readiness` (permission `store.view`) — store `1` is the
Default Store on a fresh database:

```bash
BACKEND=http://localhost:9095
TOKEN=$(curl -s -X POST "$BACKEND/api/login" \
  -H 'Content-Type: application/json' \
  -d '{"username":"superadmin","password":"<rotated-password>"}' \
  | jq -r .access_token)
curl -s "$BACKEND/api/stores/1/readiness" -H "Authorization: Bearer $TOKEN"
```

Expected: `"ready": true` with an empty `"blockers"` array. Possible blockers:

| Blocker | Meaning |
|---------|---------|
| `store.inactive` | Store is disabled |
| `store.address` / `store.phone` | Still placeholder/empty |
| `staff.<role>` | No active user for one of: manager, supervisor, cashier, inventory_staff, finance |
| `storage_location` | No storage location exists for the store |
| `catalog` | No active products, or all products are at zero stock |

## 6. Rotate the remaining seeded accounts

Step 3 only rotated the account you logged in with. Change the other default
accounts (or disable the ones this installation does not use) via
`/account/password` — the lock icon in the sidebar — or by an administrator in
*Administration → Users*.

## Development note

`./seed-dev.sh` is for development databases only. The seeder **preserves**
`must_change_password` on the six system accounts (the e2e workflow unflags its
own test users instead). See `AGENTS.md` for seeder flags.

## Resetting to first-install state

Re-running `migrate` is **not** a reset. `000_baseline.sql` is purely additive —
123 `ON CONFLICT` guards, forward-only `setval`, and a single `DELETE` that
touches only `schema_migrations` — so replaying it over a database that has been
used leaves every extra store, user, role, grant, and sale in place. A true
first-install state requires dropping and recreating the database.

```bash
./scripts/reset-dev-db.sh        # or: make db-fresh
```

This reproduces this runbook end to end: drop, recreate, bootstrap, apply
`database/migrations/*.sql`, then verify. It is **development only** — it refuses
to run against `ENV=production`, a non-`localhost` `DB_HOST`, a `DB_NAME` other
than `retail_pos`, or a port served by a container other than `postgres-dev`, and
it never names the production volumes. The container-identity check is skipped
with a warning if `podman` is not on `PATH`; the other three guards still apply.

The script verifies the result and then prints the three steps it cannot perform
itself:

1. **Restart the backend** — the Ristretto (10 min) and branding (60 s) caches
   survive a database swap.
2. **Clear browser site data for `http://localhost:8000`** — the `refresh_token`
   cookie and `sessionStorage.access_token` point at the old database.
   `localStorage.clear(); sessionStorage.clear();` from the console is enough.
3. **Delete the Playwright token cache** — `rm -f /tmp/retail-pos-e2e-tokens.v1.json`.

After the reset, all six accounts are back behind the 428 gate with `admin123`.
Rotate the password first (step 3): the gate answers 428 for every protected
route except `/api/change-password`, `/api/logout` and `/api/validate`, so
`GET /api/stores/1/readiness` also returns 428 until the rotation lands. Once you
are past the gate, readiness reports `ready: false` with exactly the two blockers
described in step 4 — `storage_location` and `catalog`. Design rationale:
`docs/design/dev-db-fresh-install-reset-plan.md`.
