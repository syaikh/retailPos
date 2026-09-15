-- Cash change (change_due) support for over-tender payments.
-- Permits a sale to be completed with cash exceeding the total amount and
-- records the returned change on the sale. Non-cash over-tender is rejected
-- by the backend (ErrPaymentOverTenderNonCash).
--
-- NOTE: this file was renamed 033_cash_change.sql -> 033b_cash_change.sql so
-- the two-step "033" set (audit immutability, then cash change) has distinct,
-- unambiguously ordered filenames. Lexicographic order is preserved:
-- 033_audit_log_store_and_immutability.sql < 033b_cash_change.sql < 034_*.sql.
-- The content itself is unchanged and idempotent (IF NOT EXISTS).

ALTER TABLE sales
  ADD COLUMN IF NOT EXISTS change_due integer NOT NULL DEFAULT 0;
