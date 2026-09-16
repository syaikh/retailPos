-- Migration: 048_add_termination_return_reason.sql
-- Description: Add 'termination' to consignment_pending_returns reason check constraint.
-- Required by bulk return feature (arrangement termination).

BEGIN;

-- Drop the existing check constraint
ALTER TABLE consignment_pending_returns
    DROP CONSTRAINT IF EXISTS chk_consignment_pending_return_reason;

-- Re-add with 'termination' included
ALTER TABLE consignment_pending_returns
    ADD CONSTRAINT chk_consignment_pending_return_reason
    CHECK ((reason)::text = ANY (ARRAY[
        'damaged'::character varying,
        'expired'::character varying,
        'customer_return'::character varying,
        'termination'::character varying,
        'other'::character varying
    ]::text[]));

COMMIT;
