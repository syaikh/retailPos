-- Store-scope supplier commercial terms (audit D3, Option C).
--
-- A supplier is a trading partner, not a per-store record: one distributor
-- supplying thirty stores is one row, and splitting it into thirty rows would
-- be a data migration with no security benefit. What actually varies per store
-- is the *terms* -- supplier_sku, unit_cost, lead_time_days and which supplier is
-- preferred -- and those live in product_suppliers. So the store boundary moves
-- from suppliers (identity, correctly global) down to product_suppliers
-- (commercial terms, correctly per store).
--
-- This mirrors the pattern consignment_arrangements already uses: a global
-- supplier joined to a store-scoped deal.
--
-- store_id is therefore nullable, and NULL means *global terms that apply to
-- every store*. That is the value every existing row already has, so this is
-- additive: no row is split, no FK is rewritten, and the seeder needs no
-- redesign. (Option A, wiring suppliers.store_id up instead, needed all three.)
--
-- Both uniqueness rules gain the store dimension and must treat NULL as a
-- comparable value. Plain UNIQUE would not: PostgreSQL treats NULLs as
-- distinct, so two global links for the same product/supplier pair, or two
-- global preferred suppliers for one product, would both be accepted. That is
-- the exact ambiguity this migration has to close, hence NULLS NOT DISTINCT on
-- the constraint and on the partial index.
--
-- Re-runnable by construction, per the replay contract: the column add is
-- guarded, and every constraint and index that changes shape is dropped before
-- being recreated, because CREATE ... IF NOT EXISTS cannot express "same name,
-- new definition" and CREATE UNIQUE INDEX has no IF NOT EXISTS form at all.
ALTER TABLE product_suppliers
  ADD COLUMN IF NOT EXISTS store_id integer REFERENCES stores(id);

COMMENT ON COLUMN product_suppliers.store_id IS
  'Store whose commercial terms this row carries; NULL means global terms applying to every store.';

-- One link per (product, supplier, store). Both names are dropped before the new
-- one is added, so a second run is a no-op. ADD CONSTRAINT has no IF NOT EXISTS
-- form, so the re-runnable shape is drop-then-create -- the same reason 057
-- recreates chk_pricing_status instead of altering it.
ALTER TABLE product_suppliers
  DROP CONSTRAINT IF EXISTS product_suppliers_product_id_supplier_id_key;

ALTER TABLE product_suppliers
  DROP CONSTRAINT IF EXISTS product_suppliers_product_id_supplier_id_store_id_key;

ALTER TABLE product_suppliers
  ADD CONSTRAINT product_suppliers_product_id_supplier_id_store_id_key
  UNIQUE NULLS NOT DISTINCT (product_id, supplier_id, store_id);

-- is_preferred becomes store-relative: each store picks its own preferred
-- supplier for a product, instead of the whole estate sharing one.
DROP INDEX IF EXISTS idx_product_suppliers_one_preferred;
DROP INDEX IF EXISTS idx_product_suppliers_one_preferred_per_store;

CREATE UNIQUE INDEX idx_product_suppliers_one_preferred_per_store
  ON product_suppliers (product_id, store_id) NULLS NOT DISTINCT
  WHERE is_preferred = true;

-- Listing a supplier's products now filters by store, which no existing index
-- serves: idx_product_suppliers_supplier leads with supplier_id alone.
CREATE INDEX IF NOT EXISTS idx_product_suppliers_supplier_store
  ON product_suppliers (supplier_id, store_id);

-- Drop the dead column Option C retires. suppliers.store_id was never read,
-- written or enforced by any code path -- it only looked like a security
-- boundary, which the audit called the worst of both worlds.
--
-- The emptiness guard makes dropping it honest instead of lossy. Every supplier
-- row is expected to carry NULL here (that is what "never wired up" means), but
-- an install that somehow populated it would otherwise lose those values
-- silently at the next migration replay. Refusing to run is the loud, correct
-- failure: an operator has to decide where the values belong, which is a
-- judgement call, not something a schema change should make for them.
DO $$
DECLARE
  populated integer;
BEGIN
  -- Guarded on the column's existence because a replay of this migration finds
  -- it already gone: without this the second run would fail to parse the query
  -- below, turning a completed migration into a permanent error.
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = current_schema() AND table_name = 'suppliers' AND column_name = 'store_id'
  ) THEN
    SELECT count(*) INTO populated FROM suppliers WHERE store_id IS NOT NULL;
    IF populated > 0 THEN
      RAISE EXCEPTION
        'suppliers.store_id holds % row(s) and D3 Option C retires the column. Re-point those values at product_suppliers.store_id (per-store terms) before re-running this migration.', populated;
    END IF;
  END IF;
END
$$;

ALTER TABLE suppliers DROP COLUMN IF EXISTS store_id;
