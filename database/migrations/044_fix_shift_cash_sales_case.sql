-- Migration: 044_fix_shift_cash_sales_case
-- Fix: cash_sales was 0 for closed shifts due to case mismatch
--      payment_method stored as 'Cash' but query compared with 'cash'
-- Now recalculates all closed shifts with LOWER() for case-insensitive comparison

UPDATE shifts s
SET
    cash_sales = COALESCE((
        SELECT SUM(CASE WHEN LOWER(sa.payment_method) = 'cash' THEN sa.total_amount ELSE 0 END)
        FROM sales sa
        WHERE sa.cashier_id = s.user_id
          AND sa.created_at >= s.opened_at
          AND sa.status = 'completed'
    ), 0),
    non_cash_sales = COALESCE((
        SELECT SUM(CASE WHEN LOWER(sa.payment_method) != 'cash' THEN sa.total_amount ELSE 0 END)
        FROM sales sa
        WHERE sa.cashier_id = s.user_id
          AND sa.created_at >= s.opened_at
          AND sa.status = 'completed'
    ), 0),
    total_sales = COALESCE((
        SELECT SUM(sa.total_amount)
        FROM sales sa
        WHERE sa.cashier_id = s.user_id
          AND sa.created_at >= s.opened_at
          AND sa.status = 'completed'
    ), 0),
    transaction_count = COALESCE((
        SELECT COUNT(*)
        FROM sales sa
        WHERE sa.cashier_id = s.user_id
          AND sa.created_at >= s.opened_at
          AND sa.status = 'completed'
    ), 0),
    discrepancy = s.closing_balance - s.opening_balance - COALESCE((
        SELECT SUM(CASE WHEN LOWER(sa.payment_method) = 'cash' THEN sa.total_amount ELSE 0 END)
        FROM sales sa
        WHERE sa.cashier_id = s.user_id
          AND sa.created_at >= s.opened_at
          AND sa.status = 'completed'
    ), 0)
WHERE s.status = 'closed';
