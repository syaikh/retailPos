# Store Onboarding Wizard + Forced First-Login Password Rotation

> **Status:** Planned → In implementation
> **Related:** [Store-First + Finance Role](./store-first-and-finance-role.md), [Store Scoping All Roles](./store-scoping-all-roles.md)

## Problem

Creating a store today is a bare `INSERT INTO stores`. After that, an operator must
manually discover and execute several unrelated tasks before the store can serve
a customer:

1. Create the operational staff accounts (manager, supervisor, cashier,
   inventory_staff, finance) — each needs `store_id` or the backend rejects them
   (`internal/user/handler.go` CreateUser/UpdateUser guards).
2. Create a default storage location — `chk_storage_locations_scope` requires a
   `store_id` **or** `warehouse_id`, so an empty store has nowhere to put
   store-scoped inventory.
3. Confirm stock is sellable — checkout deducts only from the global bucket
   (`internal/inventory/stock_deducer.go`), so a store with no global stock cannot
   complete a sale.
4. Nothing reports any of this back. There is no notion of "this store is ready".

Two related gaps make this worse:

- **A new store cannot identify itself.** The session/user payload returns only a
  numeric `store_id`; the sidebar and receipt header render the global
  `app_settings.store_name`. Out of scope here, tracked as a follow-up.
- **No forced password rotation.** Admin-created accounts get an initial password
  with no mechanism to require a change on first login.

## Decisions

| Decision | Choice |
|----------|--------|
| Scope | Full onboarding wizard (not a checklist-only page) |
| Staff creation | Inline in the wizard, one row per role |
| Stock intake | Deep links only (Import CSV / Purchase Order / Adjust Stock) — no automation |
| Readiness state | **Computed** by a new endpoint; no `stores.ready` column, no cached flag |
| Initial passwords | Backend-generated temp password + `users.must_change_password` column, forced rotation at first login (migration) |
| "Ready" definition | **All five roles** (manager, supervisor, cashier, inventory_staff, finance) have ≥1 active account, plus a storage location and address/phone present |
| Who provisions stores | **superadmin only** — `store.create` revoked from `manager` (migration). A store-scoped manager cannot read the new store's staff, so HQ provisions. |

## Migration 050 (`050_store_onboarding.sql`)

Applied **before** the new binary (AGENTS.md migration ordering):

1. `ALTER TABLE users ADD COLUMN IF NOT EXISTS must_change_password boolean NOT NULL DEFAULT false`
2. Revoke `store.create` from the `manager` role (same `DELETE FROM role_permissions` shape as `049`).
3. Register itself in `schema_migrations`.

No readiness column, no `stores.status` — readiness is always derived.

## Backend

### A. Forced first-login password rotation

- `internal/user/domain.go`: `User.MustChangePassword` (JSON `must_change_password`).
- `internal/user/auth_service.go`:
  - `AuthClaims.MustChangePassword` travels in the JWT.
  - Set on `Login` and re-read on `RefreshToken` (both already load the user).
  - `ChangePassword` clears the column and **re-issues the access token** so the
    caller does not need to log in again.
- `internal/user/repository.go`: column added to every user SELECT/scan site.
- `internal/user/handler.go`: `CreateUserRequest.must_change_password` (optional,
  default `false`). The wizard sends `true`.
- `internal/middleware/auth.go` `NewModularAuthMiddleware`: when
  `claims.MustChangePassword` is set and the request is not on the allowlist,
  abort with **428** and error code `PASSWORD_CHANGE_REQUIRED`.
  - Allowlist (by path): `POST /api/change-password`, `POST /api/logout`,
    `POST /api/validate` (so a reload can re-read the flag on the session).
  - Covers the whole `protected` group (`cmd/server/main.go`), which runs
    `authMiddleware` first; login/refresh routes registered outside that group
    are unaffected.

Why 428 (Precondition Required) rather than 403: the token is valid, one specific
precondition is unmet. Frontend maps it to a blocking change-password modal.

### B. Readiness endpoint

`GET /api/stores/:id/readiness` — `perm(store.view)` + `requireOwnStore`
(superadmin's nil store passes, a store-scoped caller may only read its own store).

Module boundaries (`.golangci.yaml` `clean-module-boundary` + `internal/archtest`)
forbid `internal/store` importing `user`/`product`/`storagelocation`, so readiness
reads go through **consumer-side ports wired in `internal/wiring`** (the same
structural-typing pattern as `internal/user/role_provider.go`,
`internal/customer/count_provider.go`, `SetProductMetaProvider`):

| Port (declared in `internal/store/ports.go`) | Implemented by | Question it answers |
|---|---|---|
| `StaffCountProvider.StaffCountsByStore` | `internal/user/store_staff_provider.go` | active users per role for this store |
| `SellableStockProvider.SellableStockStats` | `internal/product/sellable_stats_provider.go` | active catalog size + how many have zero sellable stock (mirrors `query.go` `stock <= 0`) |
| `StorageLocationCountProvider.CountByStore` | `internal/storagelocation/store_count_provider.go` | storage locations for this store |

Response:

```jsonc
{
  "store_id": 2,
  "name": "Store Bandung",
  "is_active": true,
  "address_set": true,
  "phone_set": false,
  "staff": { "manager": 1, "supervisor": 1, "cashier": 2, "inventory_staff": 1, "finance": 0 },
  "required_roles": ["manager", "supervisor", "cashier", "inventory_staff", "finance"],
  "catalog": { "active_products": 4500, "zero_stock_products": 12 },
  "storage_locations": 1,
  "ready": false,
  "blockers": ["staff.finance", "store.phone"]
}
```

`ready` is true when every required role has ≥1 active account, a storage
location exists, and address + phone are set. The catalog gates readiness too:
the `catalog` blocker is emitted when `active_products == 0` **or** every
active product is out of stock (`zero_stock_products == active_products`);
a partially out-of-stock catalog is informational (the catalog is shared).

## Frontend

- `stores-service.ts`: `getReadiness(id)`.
- `modules/stores/components/StoreOnboardingWizard.svelte` — `Modal` + `type Step`
  pattern from `shared/ui/ImportWizard.svelte` (local step header; no Stepper
  component exists yet):
  1. **Store details** — name/address/phone → `POST /stores`.
  2. **Staff** — one row per required role (defaults: 1 of each; rows skippable),
     password generated with `crypto.getRandomValues` (16 chars, satisfies the
     ≥8 rule) and shown once with a copy button → `POST /admin/users` with
     `store_id` + `must_change_password: true`.
  3. **Storage location** (optional) → `POST /storage-locations`.
  4. **Stock intake** — deep links only.
  5. **Readiness** — live checklist from `GET /stores/:id/readiness`, blockers in
     red, "Go to POS" / "Go to Users" actions.
- `StoresPage.svelte`: "Add store" opens the wizard; readiness badge per row.
- **Forced rotation UI** (no password-change screen exists in `web/src` today):
  `auth-store` holds `mustChangePassword`; `app/main.svelte` guard renders a
  blocking `ForceChangePasswordModal` → `POST /api/change-password` → store the
  re-issued token → continue.

## Security / privacy notes

- The readiness payload exposes **counts only** (no usernames, no emails), so it
  is safe to return to any caller that passes `store.view` + `requireOwnStore`.
- Generated passwords are displayed once in an admin session; they are never
  returned by the API (only the hash is stored).
- Revoking `store.create` from `manager` is a least-privilege change: provisioning
  is an HQ action, and a store-scoped manager could not verify the result anyway.

## Test plan

- **Go:** readiness service with mocked ports; `requireOwnStore` 403; middleware
  428 + allowlist pass; `CreateUser` stamping; `ChangePassword` clearing the flag
  and re-issuing the token.
- **Frontend (vitest):** wizard step gating; force-change guard behaviour.
- **E2E:** `store-onboarding.spec.ts` (superadmin → store → staff → readiness
  200), `force-password-change.spec.ts` (login as wizard-created user → 428 →
  change → 200).

## Out of scope / follow-ups

- Per-store display name in sidebar and receipt header (global
  `app_settings.store_name`).
- `store_name` added to `ValidateSession` so a cashier can see their own store.
- Automated starter-stock provisioning.
