-- Migration: 003_settlement_updated_at.sql
-- Description: Adds the missing updated_at column to consignment_settlements.
-- MarkSettlementPaid touches updated_at when a payout completes, but the column
-- was omitted from the 001 table definition. Idempotent (IF NOT EXISTS).
-- Deployment ordering: apply BEFORE deploying any binary that records a payout,
-- otherwise marking a settlement paid fails with a missing updated_at column.

ALTER TABLE consignment_settlements
    ADD COLUMN IF NOT EXISTS updated_at timestamp with time zone DEFAULT now() NOT NULL;