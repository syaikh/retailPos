-- Migration: 009_fix_invoice_numbers.sql
-- Description: Fix malformed invoice numbers to follow INV-YYYY-XXXXXX format
-- Created: 2026-06-04

BEGIN;

-- Get max sequence for 2026 invoices
WITH stats AS (
  SELECT COALESCE(MAX(CAST(SUBSTRING(invoice_number FROM '\d+$') AS INTEGER)), 0) as next_seq
  FROM sales 
  WHERE invoice_number LIKE 'INV-2026-%'
),
malformed AS (
  SELECT id, created_at, ROW_NUMBER() OVER (ORDER BY created_at) as rn
  FROM sales 
  WHERE invoice_number IN ('INV-1780552651256', 'INV-1780541325463')
)
UPDATE sales s
SET invoice_number = 'INV-2026-' || LPAD(((stats.next_seq + malformed.rn)::INTEGER)::TEXT, 6, '0')
FROM stats, malformed
WHERE s.id = malformed.id;

COMMIT;