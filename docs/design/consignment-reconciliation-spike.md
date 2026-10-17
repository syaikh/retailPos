# Consignment Physical Reconciliation — Design Spike (Phase 3, PRD §15.10)

> **Status:** Implemented (D1–D8); see `consignment-extras-plan.md` Phase 3

## Problem

PRD §15.10 is a single future-consideration line: *"Detail rekonsiliasi jika terdapat
perbedaan fisik antara catatan dan kondisi barang."* System `consignment_stock`
(`available_qty` + `pending_return_qty`) can drift from what is physically on the shelf
(damage, miscount, theft, unrecorded receipt). There is no scoped way to detect or correct
consignment-owned drift today. This spike resolves the open questions the plan flagged
before any code is written.

## Spike findings (verified against the working tree)

1. **No consignment ledger exists.** `consignment_stock` (`000_baseline.sql:563`) has one row
   per `(product_id, supplier_id, arrangement_id, store_id)` with `available_qty` /
   `pending_return_qty` (`CHECK >= 0`). There is **no** movement/history table for it.

2. **Every consignment stock change is paired across two modules.** `internal/consignment`
   updates `consignment_stock` (`repository.go:575` upsert, `:598` MoveToPendingReturn,
   `:618` ReduceAvailable, `:653` bulk reduce, `:678` ResolvePendingReturn) **and**, for the
   same transaction, calls `internal/inventory.ConsignmentAdjuster.ApplyConsignmentDelta`
   (`internal/inventory/consignment_adjuster.go:37`), which applies a signed delta to the
   global `product_stock` row **and** appends an `inventory_movements` row. The consignment
   module is the owner of the ownership ledger; `internal/inventory` is the canonical
   single-writer of `product_stock` (ADR §2.8). This is why POS can sell consignment goods:
   `product_stock` tracks sellable qty (BR-18), `consignment_stock` tracks supplier ownership.

3. **`inventory_movements.type` is `varchar(50)` with no CHECK constraint** (`:885`), so a new
   `consignment_reconciliation` movement type is safe. Existing consignment types are
   `consignment_receipt`, `consignment_return`, `consignment_pending_return`,
   `consignment_customer_return` (`internal/consignment/domain.go:74-77`).

4. **Stock opname must not be reused.** `internal/stockopname` runs a 9-state machine
   (`draft → open → counting → verification → needs_recount → approved → posted → closed →
   cancelled`, `domain.go:39-49`) with `stock_opname_*` tables, and posts **absolute** writes
   to `product_stock` via `inventory_adjustments`. Routing consignment drift through it would
   misattribute ownership and drag in assignment/recount machinery the case does not need.

5. **Settlement is orthogonal to on-hand stock.** Settlement is driven entirely by
   `consignment_sale_items` (sold, unsettled) — `ListUnsettledSaleItems`, `InsertSettlementItem`
   (`repository.go:1330`, `:1400`). It never reads `consignment_stock`. So reconciling
   *on-hand* qty does not change what is owed for already-sold goods.

## Corrected design (vs. the plan's candidate approach)

- **Correction to "adjusts `consignment_stock` only":** a posted reconciliation must move
  **both** ledgers by the same signed delta — `consignment_stock.available_qty` **and**
  `product_stock` (via `ApplyConsignmentDelta` with the new movement type). Adjusting only
  `consignment_stock` would silently desynchronise the store's sellable qty from supplier
  ownership and break BR-18. "Only" in the plan meant *not via stock-opname's
  `inventory_adjustments` absolute-write path* — that part stands.
- **A durable reconciliation record needs new tables.** None exists. Confirmed design: a
  header + items pair mirroring `consignment_returns` / `consignment_return_items` — header
  carries arrangement, document number (`RECON-YYYY-xxxxxx`), status, counted-by/posted-by,
  notes; items carry `product_id`, `system_available`, `physical_available`, `variance`,
  `reason`.
- **State machine:** `draft → counted → posted`.
- **Permission:** new `consignment.reconcile` (follows the `stock_opname.*` granularity
  precedent and the §15.6/§15.7 access direction), granted to `superadmin`, `manager`,
  `inventory_staff`.
- **Missing stock creates a tracked liability (D3).** When the reason is `lost`/`stolen`
  (store-fault), the reconciliation computes what the store owes the supplier for that line
  = the supplier's normal entitlement `unitPrice × qty − computeStoreShare(unitPrice, qty,
  type, value)` (`repository.go:1863`). Other reasons (`miscount`, `damaged`, `expired`,
  `other`) and positive variances create **no** liability. The amount is **added to the
  supplier's next settlement** (see below), so it is paid with the normal payout.

### Settlement integration (D3b)

`CreateSettlement` (`service.go:1406`) currently requires ≥1 unsettled sale item and writes
one `consignment_settlement_items` row per sale item (FK `consignment_sale_item_id` is
`NOT NULL`). To fold liabilities in without touching that invariant:

- Add `total_liability integer NOT NULL DEFAULT 0` to `consignment_settlements`, and set
  `total_payable = total_sale_value − total_store_share + total_liability`.
- Settlement items stay sale-item-only; a liability is **not** a settlement item.
- Each posted reconciliation item with a liability carries `settled_settlement_id`
  (nullable FK). At settlement time the service pulls the supplier/store's **open** liability
  items (posted, `settled_settlement_id IS NULL`), sums them into `total_liability`, and
  stamps their `settled_settlement_id`.
- `CreateSettlement` now succeeds when there is ≥1 sale item **or** ≥1 open liability; the
  `ErrEmptySettlement` guard becomes "no sale items and no open liabilities".
- The settlement detail view gains a "Stock corrections owed" section listing the linked
  liabilities (number, product, qty, amount); the payout step is unchanged (it already pays
  `total_payable`).

## Decisions (signed off)

| # | Decision | Resolution |
|---|----------|-----------|
| D1 | Permission + roles | New `consignment.reconcile`; `superadmin`, `manager`, `inventory_staff` |
| D2 | Scope | Per-arrangement |
| D3 | Negative-variance loss | **Calculate/track amount the store owes the supplier** |
| D3a | Which reasons create a liability | **Only `lost`/`stolen`** (store-fault). Damage/expiry/miscount/other = none |
| D3b | How the amount is paid | **Added to the supplier's next settlement**, paid with the normal payout |
| D4 | State machine | `draft → counted → posted` |
| D5 | Positive variance | Allowed with a required reason (`unrecorded_receipt`); increases supplier ownership |
| D6 | Pending-return drift | Available-only |
| D7 | Settlement coupling | No block; D3b folds liabilities into settlement totals only |
| D8 | Count→post ownership drift | Posting re-validates each changed product against the locked ledger; a SKU released or re-owned since the count is rejected (`ErrReconciliationStale`, 409) instead of re-binding the owner columns (BR-03). A fully-released SKU may only be re-created by a surplus (D5) that is not store-owned. |

## Schema (migration `066`)

- `consignment_reconciliations` — id, `reconciliation_number` (unique, `RECON-YYYY-xxxxxx`),
  `arrangement_id` (FK), `store_id` (FK), `supplier_id` (FK), `status`
  (`draft`/`counted`/`posted`), `notes`, `counted_by`, `posted_by`, `created_at`,
  `posted_at`.
- `consignment_reconciliation_items` — id, `reconciliation_id` (FK, cascade), `product_id`
  (FK), `system_available`, `physical_available`, `variance`, `reason` (nullable, required
  when variance ≠ 0), `liability_amount integer NOT NULL DEFAULT 0`,
  `settled_settlement_id` (nullable FK to `consignment_settlements`).
- `ALTER TABLE consignment_settlements ADD COLUMN IF NOT EXISTS total_liability integer NOT
  NULL DEFAULT 0`.
- New permission `consignment.reconcile` + three `role_permissions` grants, guarded
  `WHERE NOT EXISTS`; mirrored into `000_baseline.sql` (fresh-install parity, per the
  convention `065` follows).
- Guarded DDL only; register the migration in the AGENTS.md ordering table.

## Test plan

- Repository/service: posted reconciliation applies the same delta to `consignment_stock`
  **and** `product_stock`, and appends one `consignment_reconciliation` movement per changed
  product; drawn-down `product_stock` clamps at 0; ownership invariants (BR-05b) hold;
  variance 0 writes no movement; reason required for non-zero variance; store-scope mismatch
  rejected.
- Liability: only `lost`/`stolen` produce `liability_amount`; amount = `unitPrice×qty −
  storeShare`; positive variance and non-store-fault reasons produce 0.
- Settlement integration: `CreateSettlement` includes open liabilities in `total_liability`
  and `total_payable`, stamps `settled_settlement_id`, succeeds with zero sale items but ≥1
  liability, and fails with neither; a second settlement does not re-consume settled
  liabilities.
- Handler: `consignment.reconcile` middleware wiring (denied for supervisor/cashier).
- Frontend: `web/src/modules/consignment` vitest for the reconcile modal (system / physical /
  variance columns, reason validation, liability preview).

## Out of scope

- Scheduled/automated counting, assignment and recount workflow (that is stock-opname's job).
- Pending-return reconciliation (D6).
- Auto-deducting a liability from an *existing* settlement; only future settlements absorb
  open liabilities (D3b).
