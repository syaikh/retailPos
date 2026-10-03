# Supplier Master-Data Governance Plan

> **Status:** Proposed — not started
> **Created:** 2026-10-03
> **Scope:** Retain manager `supplier.create/update/delete` on the global `suppliers`
> master and add safety rails around the shared-row effects (concurrency, referential
> delete/deactivate guards, provenance, code reuse).
> **Related:** [Master Data Store Boundary Audit & Fix Plan](./master-data-store-boundary-audit.md) (D3) · [Storage-Location Store Boundary Plan](./storage-location-store-boundary-plan.md) · [Store Scoping: All Non-Superadmin Roles](./store-scoping-all-roles.md) · [Supplier Terms Store Scope (migration 058)](../../database/migrations/058_supplier_terms_store_scope.sql)

---

## 1. Context

`suppliers` is a **global master table** (no `store_id`; migration
`058_supplier_terms_store_scope.sql` dropped the column and moved the store boundary onto
`product_suppliers`). `supplier.create/update/delete` are granted to **manager + superadmin**;
supervisor is view-only; cashier/inventory_staff/finance have none
(`database/migrations/000_baseline.sql:4913-4922`, role grants `:5038-5067`).

Supplier CRUD routes are permission-only, with **no store check**:
`internal/supplier/handler.go:121-128` (`POST`→`supplier.create`, `PUT`/bulk `PUT`→`supplier.update`,
`DELETE`/bulk `DELETE`→`supplier.delete`). By contrast the store-scoped `product_suppliers` link
writes already enforce `linkScope` → `authorizeLinkWrite` → `authorizeLink`
(`internal/supplier/handler.go:62/99/81`; superadmin `isAdmin` bypass at `:52`).

The audit doc deliberately kept supplier identity global (D3) and floated an optional
`supplier.edit_global` permission. **This plan retains manager CRUD** and instead hardens the
consequences of editing a row every store shares.

### 1.1 Failure modes being mitigated

| # | Failure mode | Evidence |
|---|---|---|
| F1 | **Silent lost update.** `UpdateSupplier` loads the row only for audit + empty `name`/`code` preservation, then `repo.Update` overwrites every column by id. Two stores editing concurrently → last write wins. | `handler.go:260-288`, `repository.go:166-174` |
| F2 | **Unguarded cross-store delete.** Soft delete (`UPDATE … SET deleted_at`) with no check that links, POs, or consignments reference the row; the supplier disappears from every store's picker while references dangle. | `handler.go:321`, `service.go:75-77`, `repository.go:181-185` |
| F3 | **No provenance on the row.** Audit logs capture old/new, but the supplier has no `created_by`/`updated_by`, so the screen gives no "changed by X" signal. | `handler.go:290-306`, `:340+` |
| F4 | **Global UNIQUE code traps soft-deleted rows.** Delete-then-recreate with the same code fails because `suppliers_code_key UNIQUE (code)` counts deleted rows. | `000_baseline.sql:2719/2721` |
| F5 | **Deactivation is equally global and unguarded** (single and bulk), with no impact warning. | `handler.go:659/702` |

## 2. Decisions (confirmed)

1. **Delete** of a referenced supplier → always **409**. No superadmin bypass.
2. **Deactivate** of a transactionally-dependent supplier → **409** (guard both).
3. **Optimistic concurrency** via a new `version` column → **409** + UI reload/re-apply prompt.
4. Graded blocking sets:

   | Reference | Owner | Blocks delete | Blocks deactivate |
   |---|---|---|---|
   | `product_suppliers.supplier_id` (non-deleted) | `internal/product` | **yes** | **no** (reversible; only hides from new selection) |
   | `purchase_orders.supplier_id` status ∈ `draft,confirmed,partial_received` | `internal/purchase` | yes | yes |
   | `consignment_arrangements.supplier_id` status = `active` (+ live consignment stock) | `internal/consignment` | yes | yes |

   `goods_receipts` has **no** `supplier_id` (links via `purchase_order_id`,
   `000_baseline.sql:711-724`), so historical GRs never block; open POs cover them transitively.
   PO statuses: `internal/purchase/domain.go:21-25`. Consignment statuses:
   `internal/consignment/domain.go:45` (`active`/`ended`).

## 3. Binding constraint: archtest

`internal/archtest/archtest_test.go:143-200` enforces strict module→table allowlists.
`internal/supplier` may touch **only `suppliers`**, so every reference check must go through a
consumer-side **port** implemented by the owning module — the established pattern
(`internal/supplier/ports.go:23` `ProductSupplierStore`, `consignment_provider.go:15`,
`name_provider.go:14`). Provenance display must likewise avoid a `users` join inside `supplier`
(resolve client-side from existing user data, or add a `UserNameProvider` port).

---

## 4. Phase 1 — Safety rails (no schema change)

### 4.1 Ports + providers

- Extend `ProductSupplierStore` (`internal/supplier/ports.go:23`) with
  `CountLinksBySupplier(ctx, db, supplierID) (int, error)` (estate-wide, includes global + all
  store rows); implement in `internal/product/product_supplier_link_store.go`.
- Extend `ConsignmentSupplierProvider` (`internal/supplier/consignment_provider.go:15`) with
  `CountActiveBySupplier(ctx, db, supplierID) (int, error)` (active arrangements + live stock);
  implement in `internal/consignment`.
- New `PurchaseSupplierProvider` port + implementation in `internal/purchase` counting open POs
  (`status IN ('draft','confirmed','partial_received')`).
- Aggregate behind a `supplier.UsageChecker` composite wired in the composition root
  (`cmd/server/main.go`) and injected into `supplier.Service`. The composite returns a structured
  `UsageCounts{ ProductLinks, OpenPurchaseOrders, ActiveConsignments int }`.

### 4.2 Service guard

`internal/supplier/service.go`: before `Delete` (`:75-77`) reject if **any** reference exists;
before an active→inactive transition reject if **in-flight** references exist. Return a typed
`ErrSupplierInUse{Counts}`.

### 4.3 Handler mapping

`internal/supplier/handler.go`: map `ErrSupplierInUse` → **409** with JSON breakdown
(`product_links`, `open_purchase_orders`, `active_consignments`) so the UI can explain *why*.
Apply to:

- `DeleteSupplier` (`:321`),
- the deactivation branch of `UpdateSupplier` (`:260-288`),
- bulk `BulkUpdate` (`:659`) / `BulkDelete` (`:702`) — **all-or-nothing**: any blocked id → 409
  listing the blocked ids, no partial write.

### 4.4 UI blast-radius

`web/src/modules/supplier/components/SuppliersPage.svelte`: delete/deactivate confirm modal fetches
and shows usage counts; destructive actions are disabled on referenced rows with an explanatory
tooltip; **Deactivate** is promoted as the primary reversible action (the existing status filter at
`:76-77` supports active/inactive).

---

## 5. Phase 2 — Migration `060_supplier_governance.sql` + concurrency/provenance

### 5.1 Migration (must be permanently re-runnable — `IF NOT EXISTS` / `IF EXISTS`)

```sql
ALTER TABLE suppliers ADD COLUMN IF NOT EXISTS created_by integer REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE suppliers ADD COLUMN IF NOT EXISTS updated_by integer REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE suppliers ADD COLUMN IF NOT EXISTS version integer NOT NULL DEFAULT 1;
CREATE INDEX IF NOT EXISTS idx_suppliers_updated_by ON suppliers(updated_by);

ALTER TABLE suppliers DROP CONSTRAINT IF EXISTS suppliers_code_key;
CREATE UNIQUE INDEX IF NOT EXISTS suppliers_code_active_key
    ON suppliers (code) WHERE deleted_at IS NULL;
```

- Old global constraint guarantees the non-deleted subset is already unique → partial index
  creation cannot fail on existing data.
- `ADD COLUMN … NOT NULL DEFAULT 1` is metadata-only on PG 11+ (dev is 18.3).
- `created_by`/`updated_by` stay NULL for pre-existing rows.

### 5.2 Backend

- `internal/supplier/domain.go`: add `CreatedBy`, `UpdatedBy *int`, `Version int`; optionally drop
  the now-dead `StoreID` field (column removed by migration 058).
- Extend `repo.Create` / `repo.Update` signatures to carry `updatedBy *int` and
  `expectedVersion *int`. `Update` sets `updated_by`, `version = version + 1`,
  `WHERE id = $ AND deleted_at IS NULL [AND version = $]`, `RETURNING version`. `RowsAffected == 0`
  with an expected version → `ErrSupplierVersionConflict` (409); without one → `ErrSupplierNotFound`
  (404 — currently `Update` ignores `RowsAffected`).
- App-level code-existence check must ignore soft-deleted rows so a code freed by delete is
  reusable.
- Expose `created_by`/`updated_by`/`version` in list/detail responses.

### 5.3 UI

- `web/src/modules/supplier/`: send `version` with updates; on **409** reload the supplier and
  prompt re-apply; show "last changed by/at" (resolve the user name from existing user data rather
  than adding a cross-module `users` join).
- Bulk update edits bypass the version check by design; they still stamp `updated_by`.

---

## 6. Phase 3 — Observability

Dashboard/notification entry when a shared supplier is deactivated or deleted. The audit log already
records old/new (`handler.go:290-306`), so this is additive. Design confirmed 2026-10-03:

- **Event (single topic, action field).** `internal/events/supplier.go`:
  `TopicSupplierChanged = "supplier.changed.v1"`; action constants
  `SupplierActionDeactivated = "deactivated"`, `SupplierActionDeleted = "deleted"`; payload
  `SupplierChanged{SupplierID, Name, Code, Action, Version}`. Only a real state change emits:
  `Update` active → inactive and soft `Delete`. Reactivation and ordinary field edits do not.
- **Publish.** The supplier service gains a nil-guarded `SetEventBus(shared.EventBus)` (mirrors the
  `SetProductSupplierStore` pattern, so `NewService(repo)` call sites and tests are untouched) and a
  best-effort `publishChanged` helper. Wired in `internal/wiring/wiring.go` next to
  `supplier.NewService`.
- **WebSocket.** `pkg/websocket` gains `EventSupplierChanged = "supplier_changed"` +
  `SupplierChangedEvent` + `BroadcastSupplierChanged` + `NewSupplierChangedListener`, registered in
  the `wiring.go` subscription block. Suppliers are global (no store scope), so the broadcast is
  **not** store-filtered and reaches every client.
- **Audience (gated on `supplier.view`).** The frontend gates display on the `supplier.view`
  permission, mirroring `canReceiveStockOpnameNotifications` for stock opname. The backend still
  broadcasts to all; the client decides whether to surface it.
- **Frontend surfaces (bell + live-refresh list).**
  - Add `supplier_changed` to `NotificationType` and `getNotificationIcon`.
  - `NotificationBell.svelte` `onMount` registers `ws.on("supplier_changed", ...)` and pushes a
    notification with `navigateTo: "/suppliers"`, guarded by a
    `canReceiveSupplierNotifications(permissions)` helper.
  - The suppliers list subscribes to the same event and reloads, the way
    `purchase-orders/stores/po-store.svelte.ts` reacts to PO events.
  - New `en.ts`/`id.ts` title/description keys.
- **Persistence.** None: notifications stay in the in-memory store, consistent with every existing
  `NotificationType`.

---

## 7. Tests

- **Handler/service (`internal/supplier`):** referenced delete → 409; deactivate with open PO /
  active consignment → 409; stale `version` → 409; success when unreferenced; version increments;
  provenance stamped; bulk all-or-nothing.
- **Repository:** partial unique index permits the same code after soft delete; version-guarded
  update; not-found vs conflict distinction.
- **Ports/providers:** unit tests in `internal/product`, `internal/purchase`, `internal/consignment`.
- **Archtest:** passes unchanged.
- **E2E** `tests/e2e/supplier-governance-api.spec.ts`, modeled on
  `tests/e2e/storage-locations-api.spec.ts`: managerA creates; managerB opens a PO; managerA delete
  → 409; cancel PO → delete 200; stale-version update → 409. Requires the E2E stack/DB, so it runs
  in CI per AGENTS' E2E conventions.
- No permission changes → `internal/permissions` baseline tests untouched.

## 8. Risks / rollout

- New migration must be idempotent across all three non-test runners (AGENTS: never make a runner
  skip by filename).
- Phase 1 introduces three port methods + composition-root wiring — the largest structural change.
- Migration 060 is additive/backward-compatible; no data backfill required.
- Verify with targeted `go test ./internal/supplier/... ./internal/product/... ./internal/purchase/...
  ./internal/consignment/...` + `gofmt -s` / `go vet` on changed packages (load the `lint-code`
  skill). Do not run full suites locally.

## 9. Open items

None — all decisions confirmed (delete/deactivate sets, `version` column, all phases).
