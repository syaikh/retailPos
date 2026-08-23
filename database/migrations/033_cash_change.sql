-- Cash change (change_due) support for over-tender payments.
-- Permits a sale to be completed with cash exceeding the total amount and
-- records the returned change on the sale. Non-cash over-tender is rejected
-- by the backend (ErrPaymentOverTenderNonCash).

ALTER TABLE sales
  ADD COLUMN IF NOT EXISTS change_due integer NOT NULL DEFAULT 0;
