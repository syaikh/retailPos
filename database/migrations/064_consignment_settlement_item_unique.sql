-- 064_consignment_settlement_item_unique.sql
--
-- Backend review B5: a sale item could end up covered by more than one
-- settlement. CreateSettlement serializes on the sale rows (FOR UPDATE on
-- ListUnsettledSaleItems) and the settlement row (FOR UPDATE on
-- GetSettlementByIDQuery), but a UNIQUE on the settlement-line's linking
-- column is the last line of defence: it turns any residue of a double-cover
-- race into a hard duplicate-key failure instead of two payouts for the same
-- goods.
--
-- Permanently re-runnable per the migration replay contract: DROP CONSTRAINT
-- IF EXISTS + ADD CONSTRAINT makes a second runner pass a no-op. The column is
-- NOT NULL and INSERTed on every row, so a full UNIQUE (not partial) is
-- correct. Adding the constraint validates existing rows; databases that have
-- already double-covered a sale item fail here and must be cleaned up first.
ALTER TABLE consignment_settlement_items
    DROP CONSTRAINT IF EXISTS consignment_settlement_items_consignment_sale_item_id_key;

ALTER TABLE consignment_settlement_items
    ADD CONSTRAINT consignment_settlement_items_consignment_sale_item_id_key
    UNIQUE (consignment_sale_item_id);