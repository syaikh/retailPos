# Plan: Enforce store boundary in storage locations

## Problem

Storage locations only partially respect per-store assignment. `GET /storage-locations` leaks warehouse-scoped rows (`sl.store_id IS NULL OR sl.store_id = $1` matches every store's warehouses); `GET /:id`, `POST`, `PUT /:id`, `DELETE /:id`, and both `/bulk` endpoints have **no ownership check** — any store-scoped role with the permission can read/write across stores. `List` is the sole scoped endpoint (`internal/storagelocation/handler.go:59`).

The correct pattern already exists: `checkRackStore` (`internal/inventory/location_repository.go:274`) — nil caller store (superadmin) bypasses; otherwise the row's store must match. Warehouse→store resolution exists: `store.WarehouseStoreIDProvider` (`internal/store/name_provider.go:90`).

## Decisions (confirmed)

1. Cross-store access → **403** (`ErrStoreForbidden` style, like inventory).
2. Warehouses with `store_id IS NULL` → **superadmin-only** (hidden + non-manageable by store roles).
3. `GetAllActive` → **delete** (dead code, unscoped trap — only tests reference it).
4. Strict bulk: any foreign ID → 403, **no partial write**.
5. E2E simulation included, with a UI list-scoping test.

## Key constraint

`archtest` limits `storagelocation` SQL to only `storage_locations` (`internal/archtest/archtest_test.go:207-209`). The List filter **cannot** subquery `warehouses`; all warehouse→store resolution goes through the `ExistenceProvider` port owned by `internal/store`.

---

## Step 1 — Ports (store-owned provider)

- `internal/storagelocation/ports.go` — extend `ExistenceProvider` with:
  - `WarehouseStoreID(ctx, db, warehouseID int) (*int, error)`
  - `WarehouseIDsByStoreID(ctx, db, storeID int) ([]int, error)`
- `internal/store/existence_provider.go` — implement both (SQL stays in `internal/store`, owner of `warehouses`).
- Wiring unchanged (`SetStoreExistenceProvider(store.ExistenceProvider{})`); keep unwired fail-fast guards.

## Step 2 — Domain error

- `internal/storagelocation/domain.go`: `ErrStoreForbidden = errors.New("storage location is not in your store")`.

## Step 3 — Repository (`internal/storagelocation/repository.go`)

1. `GetAll`: for `storeID != nil` → `(sl.store_id = $n OR sl.warehouse_id = ANY($n+1))` with warehouse IDs from `WarehouseIDsByStoreID` (excludes other stores + NULL-store warehouses); fail fast if provider unwired.
2. Add `WarehouseStoreID(ctx, id)` (provider delegate with unwired guard).
3. Add `GetByIDs(ctx, ids)` (`WHERE id = ANY($1)`) for bulk pre-check.
4. Delete `GetAllActive`.

## Step 4 — Service (`internal/storagelocation/service.go`)

1. Repo interface: remove `GetAllActive`; add `WarehouseStoreID`, `GetByIDs`.
2. `ensureStoreScope(row, callerStoreID)`:
   - nil caller (superadmin) → pass
   - `row.StoreID` set → must equal caller, else `ErrStoreForbidden`
   - else resolve warehouse → must be non-nil and equal caller (NULL warehouse-store → forbidden)
   - else forbidden (defensive)
3. Add `storeID *int` param to `GetByID / Create / Update / Delete / BulkUpdate / BulkDelete`:
   - **GetByID/Delete**: load → `ensureStoreScope`.
   - **Create**: after existence check, body `store_id` must equal caller; `warehouse_id` must resolve to caller → else 403.
   - **Update**: scope-check existing row **and** final merged `warehouse_id`/`store_id`.
   - **Bulk (strict)**: `GetByIDs` → scope-check each existing row → any foreign → `ErrStoreForbidden`; nonexistent IDs remain no-ops.
4. Delete `Service.GetAllActive`.

## Step 5 — Handler (`internal/storagelocation/handler.go`)

- Pass `shared.GetStoreID(c)` to all six service calls.
- `errors.Is(err, ErrStoreForbidden)` → `403` in every error path (add branch **before** GetByID's 404 and before Create/Update/Delete/Bulk's 400).

## Step 6 — Go tests (`internal/storagelocation/`)

- **Update:** add `, nil` to all service calls in `service_test.go`; delete `TestService_GetAllActive`; delete `GetAllActive` subtest (`repository_test.go:106`); update `repository_mock_test.go` (remove `GetAllActive` usage, add new interface methods); add helper `createTestWarehouseForStore(t, code, storeID)` (existing helper creates NULL-store warehouses — reused for the superadmin-only case).
- **New:** `testAuthMiddlewareWithStore(storeID int)`; 403 assertions for foreign GET/PUT/DELETE/bulk/POST (`store_id`, foreign warehouse, NULL warehouse); List scoping (store A sees own + own-warehouse only; superadmin sees all); `assert.ErrorIs(..., ErrStoreForbidden)` service cases; `WarehouseIDsByStoreID` repository test.

## Step 7 — E2E simulation (Playwright)

### 7a. `tests/e2e/db-helper.ts`

Extend `TestDataTracker` with `locationIds`/`warehouseIds` + `trackLocation`/`trackWarehouse`; cleanup order: locations → warehouses → stores (FK-safe).

### 7b. New `tests/e2e/storage-locations-api.spec.ts` (behavior, per AGENTS.md)

`beforeAll` (superadmin): create Store B (`POST /api/stores`); `managerA`/`managerB` (role 2, store A/B via `POST /api/admin/users`); `warehouseA`/`warehouseB`/`warehouseNULL` via `execSQL` (no warehouse POST API exists — only `GET /warehouses`); locations `locA`, `locB`, `locWhA`, `locWhNull` via `POST /api/storage-locations`; track everything. `loginDriver` per manager; fresh driver per test (`beforeEach`); unique `Date.now()` suffixes (shard-safe).

Tests (as `managerA`):

1. List contains `locA`+`locWhA`; excludes `locB`+`locWhNull`
2. `GET locB` → 403; `GET locA` → 200; `GET locWhNull` → 403
3. `PUT locB` → 403 (unchanged); `PUT locA` → 200
4. `DELETE locB` → 403 (survives)
5. `DELETE /bulk [locA, locB]` → 403, neither deleted
6. `PUT /bulk` with `locB` → 403
7. `POST store_id:B` → 403; `warehouse_id:warehouseB` → 403; `warehouse_id:warehouseNULL` → 403
8. `POST store_id:A` → 201; `warehouse_id:warehouseA` → 201
9. Superadmin control: List shows all 4; `GET locB` → 200

### 7c. UI test

`loginUI(managerA)` → `goto ${FRONTEND_BASE}/storage-locations` → `waitForAppReady` → table shows `locA` code, does **not** show `locB` code (genuine UI list-scoping behavior).

---

## Out of scope

- Frontend code (scoping is server-side; 403 flows through existing `getApiErrorMessage` toasts).
- Permissions/roles/migrations (unchanged).
- Wiring (unchanged).
- README permission tables (permission codes unchanged).

## Verification

No local full lint/test runs (AGENTS.md) — CI runs gofmt/vet/golangci-lint/`go test`/archtest + e2e (4 shards, full stack). Targeted local runs only on request:

- `go test ./internal/storagelocation/... ./internal/store/... ./internal/archtest/...`
- `npx playwright test storage-locations` from repo root

Note: the E2E spec goes red until the Go fix lands — expected regression simulation. Never auto-commit.
