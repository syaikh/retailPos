# Konsinyasi Supplier (Consignment) — Implementation Plan

Source of truth: `docs/prd/PRD-Konsinyasi-Supplier-v7-Consolidated-Full.md` (v7, amended v7.1 2026-08-14: BR-05b / AC-C37 / EC-12 / EC-BR-11 / ownership definition / §15.9 consequence).
Status: Approved for implementation — design decisions locked 2026-08-14.

## Goal

Deliver the consignment supplier feature per the PRD: flag suppliers, manage arrangements + terms, record receipts (after inspection), track consignment stock ownership per supplier, sell through existing POS, pending returns, formal returns, full settlement, and payout records.

## Feasibility verdict

**Yes — implementable.** Every building block exists as a proven pattern in the codebase:

- **Workflow module template** — `internal/stockopname` (document-number sequences `so_seq`/`ia_seq`, status lifecycle, multi-table repository, permission seeds, wiring).
- **Cross-module ports (structural typing)** — `sale.StockDeducer` ← `inventory.StockDeducer`, `stockopname.MovementWriter` ← `inventory.MovementWriter` (ADR Modular_Monolith_Module_Boundaries §2.8). Consignment reuses the exact same seam.
- **Transaction Unit-of-Work** — checkout is a single tx (reserve stock → create sale → create payment, `cart_service.go:493-514`; `service.go:239`). Consignment sale tagging slots into this tx.
- **Price-at-sale snapshotting** — `sale_items` already stores `unit_price`, `original_price`, `cost`, `snapshot_created_at` (`repository.go:142-149`) → supports BR-19/BR-43 (terms at sale) with the existing snapshot pattern.
- **Supplier entity** — `internal/supplier` + `suppliers` table; adding an `is_consignment` flag is additive.
- **RBAC/audit** — dot-notation permission codes seeded via migration (AGENTS.md ordering), enforced by `middleware.RequirePermission`; audit via `audit.AuditCreator`.
- **Frontend** — Svelte 5 modules (`web/src/modules/*`), permission-gated routes, existing UI kit, Jakarta-time utils.

## Confirmed design decisions

| Decision | Choice |
| --- | --- |
| Supplier flag | `suppliers.is_consignment boolean DEFAULT false` (BR-01). No new supplier entity. |
| Stock model | **Separate consignment stock tables**; POS availability is ownership-aware (a SKU is either store-owned or consignment-owned, never both — BR-02/AC-C36 guarantees no merge needed). |
| Ownership conflict | Ownership = available + pending return (BR-05b). A SKU can only move to another supplier when available **and** pending return are both 0. Pending return **blocks** a different supplier (AC-C37 / EC-12). |
| Settlement | Full-only, all unsettled sales of a supplier in one settlement (BR-41/AC-C27/28). No partial. |
| Payment to supplier | **Settlement creates the liability**; a `consignment_payouts` record reusing existing `payment_methods` marks money-out (decoupled from sale payments). |
| Arrangement Ended (>2 weeks no visit) | **Lazy**: derived on read and on `RecordSupplierVisit`. No scheduler for MVP. |
| Terms snapshot | Store-share type/value + agreed price snapshotted onto each `consignment_sale_items` row at sale time (mirrors `sale_items` pricing snapshot; BR-19/43, AC-C11/12). |
| Settlement formula (AC-C29) | `supplier_payable = Σ sale_item.subtotal − store_share`; `%` share = round(subtotal × pct), `fixed` share = fixed_amount × qty. |
| Receipt document | `CR-` numbered; print doc deferred (PRD §15.4) but document id/number stored for later rendering (BR-11). |
| Locked-SKU consequence | **MVP = Option A**: accepted as ownership-first consequence; release-outside-visit (forced return + permission) deferred to PRD §15.9. |

## Gaps that shaped the design

1. **`product_stock` has no ownership dimension.** It is `(product_id, warehouse_id, store_id, location_id)` + quantity, `quantity >= 0`, and `internal/inventory` is the canonical single-writer. Consignment ownership lives in new tables; any `product_stock` write (receipt add, return subtract, pending-return removal from sellable) must flow through an **inventory-owned port** inside the caller's tx.
2. **Checkout assumes all sellable qty is in `product_stock`.** `finalizeSaleItems` (`cart_service.go:527`) calls `StockDeducer.DeductStock` unconditionally. Under the separate-tables model, consignment items are not in `product_stock` — checkout must branch: resolve ownership, deduct `consignment_stock.available`, and record the consignment sale item in the same tx.
3. **No supplier payout flow exists.** `sale_payments` is customer-facing only. Settlement and payout must be decoupled; payout is a new record type (decision above).
4. **No sale→consignment link.** `sale_items` carries no supplier/terms snapshot; a new `consignment_sale_items` table carries sale_id + product_id + qty + unit_price + store-share snapshot (avoids needing `sale_items.id` back from `CopyFrom`).
5. **Migration baseline is squashed.** Only `000_squash.sql` remains. New feature = new `001_consignment.sql` applied after it (AGENTS.md ordering: deploy migration before the binary that references its tables/sequences/permissions).

## Phase 1 — Database (`database/migrations/001_consignment.sql`)

Sequences: `consignment_receipt_seq`, `consignment_return_seq`, `consignment_settlement_seq`, `consignment_payout_seq` → document numbers `CR-`, `RT-`, `CS-`, `CP-`.

Tables (all `CREATE TABLE IF NOT EXISTS`, FKs + indexes + CHECKs):

- `suppliers.is_consignment boolean DEFAULT false` (additive).
- `consignment_arrangements` — `id`, `supplier_id` FK, `store_id` FK, `status` (`active`/`ended`), `last_visit_at timestamptz`, `ended_at`, `created_by`, `created_at`, `updated_at`. UNIQUE `(supplier_id, store_id, status)` is **not** used (one arrangement may outlive an `ended` one → next arrangement same pair) — instead UNIQUE `(supplier_id, store_id)` on the single *active* row is enforced by a **partial unique index** `ON (supplier_id, store_id) WHERE status='active'` (one active arrangement per supplier+store; a new active arrangement after `ended` is allowed).
- `consignment_terms` — `id`, `arrangement_id` FK, `product_id` FK, `price int` (agreed), `store_share_type` (`percentage`/`fixed_amount`), `store_share_value numeric`, `effective_from timestamptz`, `created_by`, `created_at`. **CHECK exactly one share type semantics**: `CHECK (store_share_type IN ('percentage','fixed_amount'))` + `CHECK (store_share_value > 0)`. UNIQUE `(arrangement_id, product_id)` — one current term per product; history via future effective-dated rows (out of MVP).
- `consignment_receipts` — `id`, `receipt_number` UNIQUE (`CR-`), `supplier_id`, `store_id`, `arrangement_id`, `received_by`, `received_at`, `notes`, `created_at`.
- `consignment_receipt_items` — `id`, `consignment_receipt_id` FK, `product_id`, `accepted_qty` (BR-10, rejected never recorded), `price` + `store_share_type` + `store_share_value` (terms snapshot), `notes`.
- `consignment_stock` — `id`, `product_id`, `supplier_id`, `arrangement_id`, `store_id`, `available_qty int`, `pending_return_qty int`, `updated_at`. **UNIQUE `(product_id)`** — ownership invariant: one active owner per SKU (BR-03/AC-C35). CHECKs: `available_qty >= 0`, `pending_return_qty >= 0`.
- `consignment_pending_returns` — `id`, `supplier_id`, `product_id`, `arrangement_id`, `store_id`, `qty`, `reason` (`damaged`/`expired`/`customer_return`/`other`), `notes`, `status` (`open`/`returned`), `returned_at`, `created_by`, `created_at` (simple record, BR-29).
- `consignment_returns` — `id`, `return_number` UNIQUE (`RT-`), `supplier_id`, `store_id`, `arrangement_id`, `returned_by`, `returned_at`, `notes`, `created_at`.
- `consignment_return_items` — `id`, `consignment_return_id` FK, `product_id`, `qty`, `reason`, `pending_return_id` FK NULL (optional link, §6.7).
- `consignment_sale_items` — `id`, `sale_id` FK, `product_id`, `supplier_id`, `arrangement_id`, `store_id`, `quantity`, `unit_price` (snapshot at sale, BR-43), `subtotal`, `store_share_type`, `store_share_value` (snapshot, AC-C12), `settlement_id` FK NULL. **Unsettled = `settlement_id IS NULL`** (BR-24/AC-C18). Index on `(supplier_id, settlement_id)`.
- `consignment_settlements` — `id`, `settlement_number` UNIQUE (`CS-`), `supplier_id`, `store_id`, `total_sale_value int`, `total_store_share int`, `total_payable int`, `status` (`pending_payment`/`paid`), `created_by`, `created_at`, `paid_at`.
- `consignment_settlement_items` — `id`, `consignment_settlement_id` FK, `consignment_sale_item_id` FK, `quantity`, `unit_price`, `subtotal`, `store_share`.
- `consignment_payouts` — `id`, `payout_number` UNIQUE (`CP-`), `settlement_id` FK, `payment_method_id` FK (reuse `payment_methods`), `amount`, `reference_number`, `paid_by`, `paid_at`, `notes`, `created_at`.

Permissions (INSERT ... ON CONFLICT DO NOTHING + role grants, pattern `031_supplier_permissions.sql`):
- `consignment.view` (superadmin, admin, manager)
- `consignment.create` (superadmin, admin, manager — receipt/arrangement/pending return/return)
- `consignment.update` (superadmin, admin, manager — terms change)
- `consignment.settle` (superadmin, admin, manager)
- `consignment.pay` (superadmin, admin)
- `supplier.update` already exists for the consignment flag toggle.

Self-register: `INSERT INTO schema_migrations (filename) VALUES ('001_consignment.sql');`

**Deployment ordering:** apply `001_consignment.sql` before deploying the binary that reads `consignment_*` tables and validates `consignment.*` permission codes at startup.

## Phase 2 — Backend (`internal/consignment/`, new package)

Standard repo/service/handler split, wired in `internal/wiring/wiring.go`, routes in `cmd/server/main.go` (`protected` group).

**Domain (`domain.go`)** — errors (`ErrConsignmentNotFound`, `ErrConflictStoreStock`, `ErrConflictOtherSupplier`, `ErrPendingReturnBlocksTransfer`, `ErrInsufficientConsignmentStock`, `ErrPartialSettlementForbidden`, `ErrShareType`, `ErrNotConsignmentSupplier`, …), status consts, `storeShareType` type.

**Service methods (business rules enforced in service + SQL guards):**

- `CreateArrangement(supplier, store)` — reject non-consignment supplier; reject if active arrangement exists (409). (BR-01)
- `ListArrangements / GetArrangement` — pagination/filter; **lazy Ended**: `status = COALESCE(status,'ended')` when `status='active' AND last_visit_at < now() − 14 days` (BR-48), surfaced in response and persisted on next write. (BR-47)
- `SetTerms(arrangement, product, price, shareType, shareValue)` — validate one share type (BR-14/AC-C09); arrangement-owner scoped; upsert current term (BR-17/AC-C13 — applies to unsold stock, never retroactive to past sales). (BR-16)
- `RecordSupplierVisit(arrangement)` — update `last_visit_at`; runs lazy Ended check; groups Receipt/Return/Settlement/Payout for the visit (BR-46/AC-C31).
- `CreateReceipt` — **conflict check per item** (BR-02/03/05b/AC-C04/05/06/07/08):
  - product has `product_stock` store row qty > 0 → reject (BR-02).
  - `consignment_stock` row exists owned by another supplier → reject (BR-03).
  - `consignment_stock` row owned by same supplier with `available=0` but `pending_return>0` → another supplier is the requester → reject (BR-05b/EC-12); same supplier → allowed (BR-04/BR-30).
  - accepted_qty recorded only (BR-07/10), rejected qty dropped (BR-08).
  - tx: insert receipt + items → inventory `ConsignmentStockAdjuster` (+ available, + product_stock) → snapshot terms onto items → audit. (BR-09/AC-C02/C03)
- `CreatePendingReturn(product, qty, reason)` — tx: move `available → pending_return` in `consignment_stock` (BR-25/26/27) → inventory `ConsignmentStockAdjuster` (subtract product_stock so not sellable, BR-26/AC-C20) → insert record (BR-29). Not part of any settlement (BR-28/AC-C22).
- `CreateReturn` — formal, only when goods truly handed to supplier (BR-31); reduce ownership in `consignment_stock` (available or pending_return) + `product_stock` via inventory port (AC-C23); no settlement effect (BR-32/AC-C24); optional `pending_return_id` link resolves open pending returns (AC-C25). Return + new receipt in one visit supported (BR-34).
- `CreateSettlement(supplier)` — tx: load **all** `consignment_sale_items WHERE settlement_id IS NULL` (BR-40), compute totals per AC-C29 using each row's snapshotted terms (BR-42/43), create settlement + items, backfill `settlement_id`, status `pending_payment`. Empty → reject. Partial never allowed (BR-41/AC-C27/28/EC-03/04). (BR-45)
- `CreatePayout(settlement, paymentMethodID, amount, ref)` — mark settlement `paid`, insert payout row (BR-44/AC-C30). Reuse `payment_methods` validation.
- `SettlementPreview` (read-only) — show all unsettled rows + computed totals before confirm.
- Reports/export: list settlements, stock summary per supplier (available / pending return / unsettled value) — reuse `internal/platform/importexport/export` (PRD §15.5).

**Audit events**: `consignment.create_arrangement`, `consignment.set_terms`, `consignment.receipt`, `consignment.pending_return`, `consignment.return`, `consignment.settle`, `consignment.payout` via `audit.AuditCreator`.

## Phase 3 — Cross-module integration (single-writer & checkout)

### 3.1 Inventory-owned port (consignment module defines, inventory implements)

New port in `internal/consignment/ports.go`:

```go
type ConsignmentStockAdjuster interface {
    // ApplyConsignmentDelta applies a signed delta to a product's global
    // product_stock row and appends inventory_movements, within the caller's tx.
    ApplyConsignmentDelta(ctx context.Context, tx pgx.Tx, rows []shared.ConsignmentStockDelta) error
}
```

- `internal/shared/stock.go`: add `ConsignmentStockDelta{ ProductID int; Delta int; MovementType string; ReferenceID int; ReferenceTable string; UserID int; Notes string }`.
- `internal/inventory/consignment_adjuster.go`: implement — per row `UPDATE product_stock SET quantity = quantity + delta WHERE product_id=$1 AND warehouse_id IS NULL AND store_id IS NULL AND location_id IS NULL`, upsert when no row (delta must be ≥ 0 then), never negative (SQL guard); `CopyFrom` into `inventory_movements` (types `consignment_receipt` / `consignment_return` / `consignment_pending_return` / `consignment_customer_return`). Keeps inventory as canonical single-writer of `product_stock` (ADR §2.8).

### 3.2 Sale-side checkout port (sale defines, consignment implements)

New port in `internal/sale/ports.go`:

```go
type ConsignmentCheckout interface {
    // ResolveAndDeductConsignment deducts consignment_stock.available for the
    // products that are consignment-owned and returns the rows to record as
    // consignment_sale_items, within the caller's tx. Products not
    // consignment-owned are skipped (normal product_stock path handles them).
    ResolveAndDeductConsignment(ctx context.Context, tx pgx.Tx, items []Item) ([]ConsignmentSaleRecord, error)
}
```

- `ConsignmentSaleRecord` (sale-side DTO): product_id, supplier_id, arrangement_id, quantity, unit_price, subtotal, store_share_type, store_share_value.
- `internal/consignment/checkout_provider.go` implements it: `SELECT ... FROM consignment_stock WHERE product_id = ANY(...) FOR UPDATE`; for owned rows check `available >= qty` else `ErrInsufficientConsignmentStock`; deduct; snapshot the current term (agreed price + share) into the record (BR-19/43).
- `internal/sale` service: `finalizeSaleItems` (or each CreateSale path) — after `CreateSale`, call the wired `ConsignmentCheckout` and insert `consignment_sale_items` rows (sale_id link) in the **same tx**; if a consignment record is produced for a product, it must NOT also be deducted from `product_stock`. Add `SetConsignmentCheckout` fail-fast wiring like `SetStockDeducer`.
- Checkout entry points to cover (share the helper): `CheckoutCart` (`cart_service.go:388`), `CreateSale` (`service.go:239`), `CreateSaleWithParkedSale` (`service.go:380`).

### 3.3 Wiring (`internal/wiring/wiring.go`)

- `ConsignmentRepo = consignment.NewRepository(p.DB)`; set `SupplierStore`/`ProductMetaProvider` (name enrichment) + `StockSnapshotProvider` (inventory) + `ConsignmentStockAdjuster(inventory.ConsignmentStockAdjuster{})`.
- `ConsignmentSvc = consignment.NewService(ConsignmentRepo, d.Bus)`.
- `SaleSvc.SetConsignmentCheckout(consignment.CheckoutProvider{...})`.
- `ConsignmentH = consignment.NewHandler(ConsignmentSvc, d.AuditSvc)`; register routes (`deps.ConsignmentH.RegisterRoutes(protected, noopAuth, permMiddleware)` in `cmd/server/main.go`).
- Eventbus: no new business logic in listeners (AGENTS `plan-remove-eventbus-business-logic`); settlement/payout do not publish sale-affecting events.

### 3.4 API surface (`/api/consignment/*`)

| Endpoint | Permission |
|---|---|
| `GET/POST /api/consignment/arrangements`, `GET/PATCH /api/consignment/arrangements/:id` | `consignment.view` / `consignment.create` |
| `POST /api/consignment/arrangements/:id/visit` | `consignment.create` (record supplier visit; lazy Ended) |
| `PUT /api/consignment/arrangements/:id/terms` | `consignment.update` |
| `POST /api/consignment/arrangements/:id/receipts` (+ `GET` list) | `consignment.create` |
| `POST /api/consignment/arrangements/:id/pending-returns`, `POST /api/consignment/arrangements/:id/returns` | `consignment.create` |
| `GET/POST /api/consignment/settlements`, `POST /api/consignment/settlements/:id/payout` | `consignment.view` / `consignment.settle` / `consignment.pay` |
| `GET /api/consignment/suppliers` (list consignment-flagged suppliers) | `consignment.view` |
| `GET /api/consignment/stock` (per-supplier ownership summary) | `consignment.view` |

## Phase 4 — Frontend (`web/src/modules/consignment/`)

Svelte 5 module mirroring `stock-opname`/`supplier` conventions (`services/`, `types/`, `components/`, `index.ts`), registered in `web/src/app/main.svelte` + `permissions.ts`.

- **Supplier list/detail** (`web/src/modules/supplier/`) — add "Supplier Konsinyasi" flag toggle (`is_consignment`) on create/edit, gated by `supplier.update`.
- **ArrangementsPage** — list/filter (status incl. lazy-derived `ended`), create arrangement modal (consignment suppliers only), terms editor (price + share type/value with type switch, exactly-one validation), supplier visit button (runs visit → surfaces receipt/return/settlement/payout entry points).
- **ReceiptEntry** — multi-SKU line editor with inspection columns: brought qty / accepted qty / rejected qty (only accepted recorded, BR-10), per-line conflict feedback (store-stock / other-supplier / pending-return-block), terms shown read-only from arrangement.
- **PendingReturnPage** — simple record form (product, qty, reason), list with status.
- **ReturnPage** — formal return form, optional link to open pending returns.
- **SettlementPage** — preview of ALL unsettled sales (value, store share, payable per supplier), confirm creates full settlement; **no partial UI**; payout modal (payment method + reference) → marks paid.
- **StockPage** — per-supplier ownership summary (available / pending return / unsettled value).

POS is **unchanged visually** (customer unaware, AC-C16); consignment SKUs simply resolve through the ownership-aware checkout.

## Phase 4.5 — Migration application (rollout)

Migrations are **not** applied automatically by the server — `shared.RunMigrations` (`internal/shared/testdb.go`) runs only in tests; the dev DB (`retail_pos`) schema is applied manually. This step is easy to miss; it is part of the plan.

1. **Dev DB (required before running the server / seeding):**
   ```bash
   # from repo root, after writing database/migrations/001_consignment.sql
   psql "postgres://pos:admin123@localhost:5433/retail_pos" -f database/migrations/001_consignment.sql
   ```
   The migration self-registers into `schema_migrations` (its final statement), so the dev `schema_migrations` table stays in sync with `000_squash.sql` + `001_consignment.sql`.
2. **Test DB (`retail_pos_test`):** applied automatically by the existing test runner on first `go test` run (`RunMigrations` reads the sorted `database/migrations/` dir). If the test DB was previously baselined, recreate it to pick up the new file:
   ```bash
   dropdb retail_pos_test && createdb retail_pos_test
   ```
3. **Deployment ordering (pre-commit/CI/prod):** apply `001_consignment.sql` **before** deploying the binary that references `consignment_*` tables/sequences and validates `consignment.*` permission codes at startup (AGENTS.md convention). A dry-run check: after applying, `SELECT filename FROM schema_migrations ORDER BY filename;` must list both `000_squash.sql` and `001_consignment.sql`.

## Phase 5 — Tests & verification

**Backend (Go):**
- `internal/consignment/`: arrangement lifecycle + lazy Ended (2-week rule), terms validation (one share type), receipt conflict matrix (BR-02/03/05b + EC-01..EC-12), pending-return ownership lock (BR-05b/AC-C37), return reduces ownership + no settlement, full settlement math (AC-C29, % and fixed, EC-03/04), payout marks paid, insufficient consignment stock.
- `internal/inventory/consignment_adjuster_test.go`: delta on product_stock, upsert when no row, never-negative, movements ledger rows.
- `internal/sale/`: checkout with mixed cart (store + consignment items) — consignment items not double-deducted, records written in same tx; insufficient consignment stock aborts the whole sale (rollback); parked-sale completion path.
- Scenario matrix from PRD §9 as service-level tests (01 initial receipt, 02 top-up, 03 conflict, 04 ownership release incl. pending-return block, 05 sale, 06/07 damage/expired → pending return, 09 eligible customer return → pending return, 10 ended-with-stock still sellable, 11/12 terms change at sale, 13 pending return + new supply, 14 supplier request return before settlement, 15 multi-SKU independence).
- Migration auto-applies to `retail_pos_test` via existing runner.

Run: `go build ./...` then `TEST_DB_PORT=5433 DB_PORT=5433 TEST_DB_USER=pos TEST_DB_PASSWORD=admin123 DB_USER=pos DB_PASSWORD=admin123 JWT_SECRET=test-secret-for-testing-only go test -p 1 -count=1 ./internal/consignment/... ./internal/inventory/... ./internal/sale/...`.

**Frontend:** `cd web && npx vitest run src/modules/consignment/...` + component tests for terms validation and settlement preview; `cd web && npm run build`.

**Full suite** (pre-commit/CI only, per AGENTS.md): `go test -p 1 -count=1 ./...` and `npm run test:run`, optional Playwright E2E happy path (receipt → sale → pending return → return → settlement → payout).

## Documentation

- `AGENTS.md`: add migration-ordering entry for `001_consignment.sql` (tables/sequences + `consignment.*` permission codes).
- `docs/guides/user-manual.md`: new "Konsinyasi Supplier" section.
- `docs/roadmap/upcoming-features.md`: mark consignment done when shipped.
- Copy this plan to `docs/archived-plans/` on completion.

## Out of scope (deferred — PRD §15)

Receipt print-document format (id/number stored), consignment analytics dashboard, expiry-by-shelf-life rule, refund/void beyond eligible customer return, audit-trail detail, settlement policy when supplier absent long-term (incl. forced-return release path, Option A consequence §15.9), physical-count reconciliation, payment flows beyond the payout record.
