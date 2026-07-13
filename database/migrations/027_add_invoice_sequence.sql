-- Replace global advisory lock with PostgreSQL SEQUENCE for invoice number generation.
-- This eliminates the serialization bottleneck on concurrent sale creation.

CREATE SEQUENCE IF NOT EXISTS invoice_seq START 1;

-- Sync sequence to avoid conflicts with existing auto-generated invoice numbers.
-- Extracts the numeric suffix from INV-YYYY-NNNNNN format and sets the sequence
-- to the maximum value found plus a safety margin.
DO $$
DECLARE
    max_seq bigint;
BEGIN
    SELECT COALESCE(MAX(
        CAST(REGEXP_REPLACE(invoice_number, '^INV-\d+-0*', '') AS bigint)
    ), 0) + 1000 INTO max_seq
    FROM sales
    WHERE invoice_number ~ '^INV-\d+-\d+$';

    IF max_seq > 1 THEN
        PERFORM setval('invoice_seq', max_seq);
    END IF;
END $$;
