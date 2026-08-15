-- Migration: 002_settlement_items_product_id.sql
-- Description: Adds the missing product_id column to consignment_settlement_items.
-- The settlement item rows snapshot the sold product for display/join purposes
-- (the repo's insert and read queries already reference it), but the column was
-- omitted from the 001 table definition. Idempotent (IF NOT EXISTS).
-- Deployment ordering: apply BEFORE deploying any binary that creates a
-- consignment settlement, otherwise CREATE SETTLEMENT fails with a missing
-- product_id column on consignment_settlement_items.

ALTER TABLE consignment_settlement_items
    ADD COLUMN IF NOT EXISTS product_id integer;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'consignment_settlement_items_product_id_fkey'
    ) THEN
        ALTER TABLE consignment_settlement_items
            ADD CONSTRAINT consignment_settlement_items_product_id_fkey
            FOREIGN KEY (product_id) REFERENCES products(id);
    END IF;
END $$;