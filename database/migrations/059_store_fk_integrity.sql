-- Close the four store_id FK gaps (audit Wave 6).
--
-- customers, users, goods_receipts and purchase_orders are the only
-- store-scoped tables whose store_id has never carried a REFERENCES
-- stores(id). Nothing has stopped a row in any of them from pointing at a
-- store that does not exist, so a delete or a bad id could leave a dangling
-- reference that the application would then read as if it were valid. The
-- other 21 store-scoped tables already have the constraint; these four are
-- the gap.
--
-- Adding a foreign key validates every existing row at apply time, so this
-- migration hard-fails on the first orphan. That is deliberate: the
-- alternative is a constraint that is quietly unvalidated (NOT VALID),
-- which leaves the very rows it exists to catch still unchecked.
--
-- PREREQUISITE: scripts/audit-store-fk-orphans.sh must report clean against
-- the target database first. It is read-only and pins the session with
-- default_transaction_read_only=on, so it is safe to point at production.
-- Orphan cleanup is data migration and is deliberately out of scope here;
-- if the audit is non-empty, stop and remap the rows rather than weakening
-- this migration.
--
-- ON DELETE differs per table, and NOT per table by convenience but because
-- the column nullability forces it:
--
--   * customers, goods_receipts and purchase_orders have store_id NOT NULL.
--     ON DELETE SET NULL is therefore impossible without first dropping the
--     NOT NULL, which would let a customer or a financial document exist
--     outside any store -- a strictly worse state than a blocked delete. They
--     use ON DELETE RESTRICT: deleting a store that still holds customers or
--     documents is refused, and the operator resolves the rows first. This
--     matches the existing document-side convention in the schema
--     (consignment_receipts, consignment_settlements, import_jobs and seven
--     others are NO ACTION/RESTRICT).
--
--   * users.store_id is nullable, and a NULL store on a user is meaningful
--     rather than broken: it is an HQ identity that is deliberately visible
--     and editable across stores (see bindStoreScopedUser in wave 3).
--     SET NULL preserves that on delete instead of cascading the user away.
--     This matches the entity-side convention (products, sales, shifts,
--     warehouses and six others).
--
-- goods_receipts.store_id is also the only one of the four with no index, so
-- one is added here. A foreign key is not indexed automatically in
-- PostgreSQL, and without it every parent-side delete on stores has to
-- sequential-scan the child -- which is exactly the operation RESTRICT now
-- performs on every store delete.
--
-- Re-runnable by construction, per the replay contract: every constraint is
-- dropped before being recreated, because CREATE does not have an
-- "IF NOT EXISTS with this definition" form, and the index uses CREATE INDEX
-- IF NOT EXISTS since its shape never changes.
--
-- Paired application change: customers.store_id DEFAULT 1 is dropped in the
-- same file. The default masked a seeder bug -- cmd/dummy/main.go inserted
-- customers without a store_id, so every seeded customer silently landed in
-- store 1 regardless of the store it was being generated for. Both
-- application paths (CreateCustomer and the batch import) already pass the
-- store explicitly, so no request path is affected; the default was only
-- ever a silent fallback for a caller that forgot to pass one, on a column
-- that is NOT NULL and is about to carry a foreign key.

-- ---------------------------------------------------------------------------
-- 1. goods_receipts.store_id index (see note above on why this one is needed).
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_goods_receipts_store
    ON goods_receipts (store_id);

-- ---------------------------------------------------------------------------
-- 2. Drop customers.store_id's DEFAULT 1.
--
-- Guarded: only fires when the default is still present, so a re-run on a
-- database that already dropped it is a no-op.
-- ---------------------------------------------------------------------------
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'customers'
          AND column_name = 'store_id'
          AND column_default IS NOT NULL
    ) THEN
        -- The seeder must be fixed before this lands: it omits store_id, and
        -- with the default gone such an insert now fails on NOT NULL rather
        -- than silently writing store 1.
        EXECUTE 'ALTER TABLE customers ALTER COLUMN store_id DROP DEFAULT';
    END IF;
END
$$;

-- ---------------------------------------------------------------------------
-- 3. The four foreign keys.
--
-- Each is dropped before it is added so the definition below is the one that
-- survives a second runner pass. Dropping is safe even when the constraint
-- does not exist.
-- ---------------------------------------------------------------------------

ALTER TABLE customers DROP CONSTRAINT IF EXISTS customers_store_id_fkey;
ALTER TABLE customers
    ADD CONSTRAINT customers_store_id_fkey
    FOREIGN KEY (store_id) REFERENCES stores (id) ON DELETE RESTRICT;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_store_id_fkey;
ALTER TABLE users
    ADD CONSTRAINT users_store_id_fkey
    FOREIGN KEY (store_id) REFERENCES stores (id) ON DELETE SET NULL;

ALTER TABLE goods_receipts DROP CONSTRAINT IF EXISTS goods_receipts_store_id_fkey;
ALTER TABLE goods_receipts
    ADD CONSTRAINT goods_receipts_store_id_fkey
    FOREIGN KEY (store_id) REFERENCES stores (id) ON DELETE RESTRICT;

ALTER TABLE purchase_orders DROP CONSTRAINT IF EXISTS purchase_orders_store_id_fkey;
ALTER TABLE purchase_orders
    ADD CONSTRAINT purchase_orders_store_id_fkey
    FOREIGN KEY (store_id) REFERENCES stores (id) ON DELETE RESTRICT;