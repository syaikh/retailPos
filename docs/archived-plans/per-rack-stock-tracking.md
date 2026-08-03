# Per-Rack Stock Tracking & Stock Opname per Location — Implementation Plan

## Goal

Phase 2 of Storage Locations (see `docs/upcoming-features.md` §7): track how much stock physically sits in each storage location (rack/shelf) and allow Stock Opname sessions to count a single rack. Rack stock is a **sub-account of the global stock** (Model A).

### Model A invariants

- The existing `product_stock` **global row** (`warehouse_id`, `store_id`, `location_id` all NULL) stays the authoritative stock that sales, PO receiving, and inventory adjustments consume. **No changes to the sale/POS/purchase flows.**
- Every rack may have its own `product_stock` row with `location_id` set. Rack rows **always mirror the rack's `warehouse_id`/`store_id`** (copied from `storage_locations`), so existing queries filtering `warehouse_id IS NULL AND store_id IS NULL` keep selecting exactly the global row.
- Rack rows are bookkeeping: set/transfer operations record where stock sits without touching the global total. Rack-level **stock opname posting is the reconciliation point** — it updates the rack row *and* applies the same delta to the global row so `sum(racks) == global` stays true after posting.
- A product with no rack row has expected rack stock `0` for that rack.

---

## 1. Database Migration

### File to Create

`database/migrations/020_per_rack_stock.sql`

```sql
-- 1. location_id dimension on product_stock
ALTER TABLE product_stock ADD COLUMN location_id INTEGER REFERENCES storage_locations(id) ON DELETE SET NULL;
ALTER TABLE product_stock DROP CONSTRAINT IF EXISTS uq_product_stock;
ALTER TABLE product_stock ADD CONSTRAINT uq_product_stock
  UNIQUE NULLS NOT DISTINCT (product_id, warehouse_id, store_id, location_id);
CREATE INDEX IF NOT EXISTS idx_product_stock_location_id ON product_stock(location_id);

-- 2. location_id on stock opname sessions (for location-scoped sessions)
ALTER TABLE stock_opnames ADD COLUMN location_id INTEGER REFERENCES storage_locations(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_stock_opnames_location_id ON stock_opnames(location_id);

INSERT INTO schema_migrations (filename) VALUES ('020_per_rack_stock.sql');
```

**Notes**
- `NULLS NOT DISTINCT` keeps the "single global row per product" guarantee while allowing many location rows per product.
- Rack rows must never be inserted with both `warehouse_id`/`store_id` NULL — a service-level invariant (validated on write), because all existing global-read queries use `warehouse_id IS NULL AND store_id IS NULL`.

---

## 2. Backend — `internal/inventory` (rack stock management)

New endpoints (all gated by existing `inventory.adjust` permission — no new permission codes):

| Endpoint | Purpose |
|---|---|
| `GET /api/inventory/locations` | List rack stock. Filters: `product_id` (breakdown for one product) or `location_id` (products at a rack). Response: `[{ product_id, sku, name, location_id, location_code, location_name, quantity }]`. |
| `POST /api/inventory/locations` | **Record/set** rack stock: `{ product_id, location_id, quantity }`. Upsert the `product_stock` row for `(product, location)` (mirroring the rack's `warehouse_id`/`store_id`). **Does not change global stock.** Records `inventory_movements` (`type='location_set'`, `reference_id=location_id`). Rejects if the rack is inactive. |
| `POST /api/inventory/locations/transfer` | **Move stock between racks**: `{ product_id, from_location_id, to_location_id, quantity }`. Decrement source row (must have enough), increment destination row (upsert). Global unchanged. Records two `inventory_movements` (`type='location_transfer'`). |

**New files**
```
internal/inventory/location.go          // handler(s) for the 3 endpoints
internal/inventory/location_repository.go
internal/inventory/location_service.go
internal/inventory/location_*.test.go
```

**Reuse** existing `ProductStock` domain + `shared` helpers; keep `AdjustStock` (global) untouched.

---

## 3. Backend — `internal/stockopname` (rack-scoped sessions)

### 3.1 Scope plumbing (mirrors the existing `warehouse` scope)

- `domain.go` `validScopes`: add `"location": true`.
- `repository_workflow.go`:
  - `ResolveScopeName`: add `case "location": table = "storage_locations"`.
  - `ScopeProductIDs`: add
    `case "location": query = SELECT DISTINCT product_id FROM product_stock WHERE location_id = $1`.
- `domain.go` `Session`: add `LocationID *int \`json:"location_id,omitempty"\``.
- `service.go` `CreateSession`:
  - When a scope has `ScopeType == "location"`, set `session.LocationID` (mirror `warehouse` handling). Also derive `warehouse_id`/`store_id` for the session from the rack (`storage_locations.warehouse_id`/`store_id`) so websocket broadcast scope and existing session-level columns stay populated — add `GetLocationScope(ctx, tx, id)` returning `(warehouseID *int, storeID *int)`.

### 3.2 Location-aware snapshot (expected qty = rack row qty)

- `LoadSnapshotProductsByIDs` currently reads the **global** row. Add a location-aware variant used when `session.LocationID` is set:
  - Product universe still comes from `ScopeProductIDs` (location scope).
  - Expected/opening qty comes from `product_stock WHERE location_id = $loc` (`COALESCE 0` when the rack has no row for that product).
  - Products already present in a location row whose product is inactive are excluded (same filter as existing snapshot).

### 3.3 Location-aware verify & post

The verify and post flows (`VerifySession`, `PostAdjustment`) use `LockStockForProducts` (locks global rows) and `UpdateProductStock` (upserts global). Add location variants and switch on `session.LocationID`:

- `LockStockForLocation(ctx, tx, productIDs, locationID) (map[int]int, error)` — locks the rack rows (`location_id = $2 FOR UPDATE`).
- `UpdateLocationStock(ctx, tx, productID, locationID, newRackQty, delta int)` — single tx unit:
  1. Upsert rack row to `newRackQty`.
  2. Apply the **same delta** to the global row (upsert global = `COALESCE(global,0) + delta`), keeping `sum(racks) == global` after posting.
- Expected quantity at verify/post for location sessions = locked rack qty (`0` if no row).

`MovementTypeStockOpname` movements keep being inserted as today; no change to the IA- adjustment ledger.

### 3.4 Adjustments report / exports

No structural change — `stock_opname_items` records expected/physical/diff already; the session JSON now carries `location_id` for display.

---

## 4. Frontend

### 4.1 Stock Opname — scope picker (`StockOpnamesPage.svelte`)

- `web/src/modules/stock-opname/types/index.ts`: extend `StockOpnameScopeType` with `'location'`.
- `StockOpnamesPage.svelte` `loadOptions`: add `location` branch → `getStorageLocations({ is_active: true, limit: 500 })`, label `code ? name (code) : name`.
- Scope label/options list in the create modal gets the new entry; detail page scope name resolution already generic.

### 4.2 Inventory — rack stock management

New page/panel under Inventory (or a dedicated `/inventory/locations` route, permission `inventory.adjust`):
- Per-product rack breakdown (from `GET /api/inventory/locations?product_id=`).
- **Set rack stock** modal (product, location, quantity).
- **Transfer between racks** modal (product, from, to, qty).
- Service + types in `web/src/modules/inventory/`, matching existing module conventions (Svelte 5, `$state`, existing UI kit).

---

## 5. Tests

- **Repository/service (Go)**
  - Inventory: set (upsert), transfer (decrement/increment, insufficient source, inactive rack), list filters, global untouched by set/transfer.
  - Stock opname: `location` scope validation, scope-name resolution, product universe from rack rows, create → snapshot expected from rack qty (and `0` when no rack row), verify expected from rack qty, post updates rack row **and** reconciles global, separation of duties unaffected.
- **Migration** auto-applied to `retail_pos_test` by existing `internal/shared` runner (add `020` expectations where relevant).
- **Frontend**: component tests for the new inventory rack modals + scope option loading.
- **E2E** (optional): `tests/e2e/` rack opname happy path — create location-scoped session, count, verify, post, assert global reconciled.

Run: `go build ./...`, then the full `-p 1` test suite (AGENTS.md command), plus `npm run test` in `web/` if present.

---

## 6. Documentation

- `docs/user-manual.md`: extend Stock Opname §13 with a "Count per Storage Location" note + scope picker mention; add a "Per-Location Stock" note in §12 Storage Locations and §7 Inventory.
- `docs/upcoming-features.md`: mark Storage Locations §7 as "fase 2 selesai" once shipped.
- `AGENTS.md`: add migration-ordering entry for `020_per_rack_stock.sql`.
- Copy this plan to `docs/archived-plans/` on completion.
