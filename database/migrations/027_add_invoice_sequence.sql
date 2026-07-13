-- Replace global advisory lock with PostgreSQL SEQUENCE for invoice number generation.
-- This eliminates the serialization bottleneck on concurrent sale creation.

CREATE SEQUENCE IF NOT EXISTS invoice_seq START 1;
