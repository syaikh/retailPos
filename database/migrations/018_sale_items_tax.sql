-- Migration: 018_sale_items_tax.sql
-- Description: Add DPP (tax base) and tax amount columns to sale_items
-- for proper Indonesian PPN accounting

ALTER TABLE sale_items
    ADD COLUMN IF NOT EXISTS dpp_amount INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS tax_amount INTEGER NOT NULL DEFAULT 0;

-- Backfill existing records: old records had no tax, so DPP = subtotal (shelf price)
UPDATE sale_items SET dpp_amount = subtotal, tax_amount = 0 WHERE dpp_amount = 0 AND tax_amount = 0;
