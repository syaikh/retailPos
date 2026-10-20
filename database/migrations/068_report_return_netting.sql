-- 068_report_return_netting.sql
-- Sales Return Phase 4.3 — net completed refunds out of the reporting read model.
--
-- A completed Sales Return pays money back to the customer, so the revenue the
-- reports show must exclude it. Before this migration the three reporting
-- materialized views (mv_daily_sales, mv_hourly_sales, mv_dashboard_totals)
-- aggregated public.sales only, so a refund kept inflating daily/hourly/
-- dashboard revenue. Only the SHIFT close-time netting accounted for it.
--
-- Semantics (must match the shift netting in
-- internal/salereturn/repository.go:ReturnTotalsForShift and summariseRefund):
--   * Only status = 'completed' returns count. 'pending_approval' has not paid
--     out, and 'cancelled'/'expired' never will.
--   * Only the money actually paid out counts: sale_returns.total_refund_amount.
--     A pure replacement return has total_refund_amount = 0 (applyLineFigures
--     accumulates refund-outcome lines only), so replacements subtract nothing
--     — the customer received goods, not cash.
--   * transaction_count / active_cashiers are intentionally NOT reduced: a
--     refund is not a sale, and the "number of sales" KPI stays a gross count.
--     Only the money columns are netted.
--
-- Date bucketing uses the RETURN's own created_at, in Jakarta time, exactly as
-- the sales side does. This is the day/hour the money left the drawer, which is
-- the bucket the operator reconciles against.
--
-- FULL OUTER JOIN, not LEFT JOIN: a refund is frequently the first transaction
-- of a day (the customer returns an item bought earlier, before the store has
-- made a sale). Keying the join off the sales side alone would drop that refund
-- whenever the store has no completed sale in the same day/hour bucket, so the
-- daily and hourly charts would overstate revenue and disagree with the
-- all-time dashboard (which has no date key and always nets). The outer join
-- keeps a return-only bucket as a row with negative net revenue; consumers
-- already fill missing dates from a generated series (GetDailySales) and
-- COALESCE the counts, so a return-only bucket renders correctly. Count columns
-- are COALESCEd to 0 so the row shape matches a sale bucket.
--
-- Note on store_id: sales.store_id is nullable (FK ON DELETE SET NULL) while
-- sale_returns.store_id is NOT NULL, so a legacy no-store sale can never be
-- matched to a refund and is always counted gross. This is unchanged from
-- before and deliberate: a return is always bound to a real store.
--
-- Because the views are the single read model for every consumer (hourly chart,
-- daily chart, dual-period comparison, all-time dashboard totals), netting here
-- fixes all of them without touching a single Go query.
--
-- Permanently re-runnable per the migration replay contract: DROP ... IF EXISTS
-- then CREATE. The unique indexes the REFRESH ... CONCURRENTLY in
-- refresh_sales_mv() depends on are recreated immediately after each view.
-- Dropping and recreating also discards any concurrently-refreshing session's
-- snapshot, which is why WITH DATA is used (a populated view, matching the
-- baseline definition's WITH DATA).

-- ---------------------------------------------------------------------------
-- mv_daily_sales
-- ---------------------------------------------------------------------------

DROP MATERIALIZED VIEW IF EXISTS public.mv_daily_sales;

CREATE MATERIALIZED VIEW public.mv_daily_sales AS
 SELECT COALESCE(s.sale_date, r.return_date) AS sale_date,
    COALESCE(s.store_id, r.store_id) AS store_id,
    COALESCE(s.transaction_count, (0)::bigint) AS transaction_count,
    COALESCE(s.active_cashiers, (0)::bigint) AS active_cashiers,
    COALESCE(s.total_revenue, (0)::bigint) - COALESCE(r.refund_amount, (0)::bigint) AS total_revenue,
    COALESCE(s.total_subtotal, (0)::bigint) - COALESCE(r.refund_subtotal, (0)::bigint) AS total_subtotal,
    COALESCE(s.total_discount, (0)::bigint) AS total_discount,
    COALESCE(s.total_tax, (0)::bigint) - COALESCE(r.refund_tax, (0)::bigint) AS total_tax
   FROM (
     SELECT date((created_at AT TIME ZONE 'Asia/Jakarta'::text)) AS sale_date,
        store_id,
        count(*) AS transaction_count,
        count(DISTINCT cashier_id) AS active_cashiers,
        COALESCE(sum(total_amount), (0)::bigint) AS total_revenue,
        COALESCE(sum(subtotal), (0)::bigint) AS total_subtotal,
        COALESCE(sum(discount), (0)::bigint) AS total_discount,
        COALESCE(sum(tax), (0)::bigint) AS total_tax
       FROM public.sales
      WHERE ((status)::text = 'completed'::text)
      GROUP BY (date((created_at AT TIME ZONE 'Asia/Jakarta'::text))), store_id
   ) s
   FULL JOIN (
     SELECT date((created_at AT TIME ZONE 'Asia/Jakarta'::text)) AS return_date,
        store_id,
        COALESCE(sum(total_refund_amount), (0)::bigint) AS refund_amount,
        COALESCE(sum(subtotal), (0)::bigint) AS refund_subtotal,
        COALESCE(sum(tax_reversal), (0)::bigint) AS refund_tax
       FROM public.sale_returns
      WHERE ((status)::text = 'completed'::text)
      GROUP BY (date((created_at AT TIME ZONE 'Asia/Jakarta'::text))), store_id
   ) r ON r.return_date = s.sale_date AND r.store_id = s.store_id
  WITH DATA;

CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_daily_sales_date_store
    ON public.mv_daily_sales USING btree (sale_date, store_id);

-- ---------------------------------------------------------------------------
-- mv_dashboard_totals
-- ---------------------------------------------------------------------------

DROP MATERIALIZED VIEW IF EXISTS public.mv_dashboard_totals;

CREATE MATERIALIZED VIEW public.mv_dashboard_totals AS
 SELECT COALESCE(s.store_id, r.store_id) AS store_id,
    COALESCE(s.transaction_count, (0)::bigint) AS transaction_count,
    COALESCE(s.total_revenue, (0)::bigint) - COALESCE(r.refund_amount, (0)::bigint) AS total_revenue
   FROM (
     SELECT store_id,
        count(*) AS transaction_count,
        COALESCE(sum(total_amount), (0)::bigint) AS total_revenue
       FROM public.sales
      WHERE ((status)::text = 'completed'::text)
      GROUP BY store_id
   ) s
   FULL JOIN (
     SELECT store_id,
        COALESCE(sum(total_refund_amount), (0)::bigint) AS refund_amount
       FROM public.sale_returns
      WHERE ((status)::text = 'completed'::text)
      GROUP BY store_id
   ) r ON r.store_id = s.store_id
  WITH DATA;

-- The store-level view groups by store alone, so a store with returns but no
-- sales still needs its own row (present in the FULL JOIN).

CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_dashboard_totals_store
    ON public.mv_dashboard_totals USING btree (store_id);

-- ---------------------------------------------------------------------------
-- mv_hourly_sales
-- ---------------------------------------------------------------------------

DROP MATERIALIZED VIEW IF EXISTS public.mv_hourly_sales;

CREATE MATERIALIZED VIEW public.mv_hourly_sales AS
 SELECT COALESCE(s.sale_hour, r.return_hour) AS sale_hour,
    COALESCE(s.store_id, r.store_id) AS store_id,
    COALESCE(s.transaction_count, (0)::bigint) AS transaction_count,
    COALESCE(s.total_revenue, (0)::bigint) - COALESCE(r.refund_amount, (0)::bigint) AS total_revenue
   FROM (
     SELECT date_trunc('hour'::text, (created_at AT TIME ZONE 'Asia/Jakarta'::text)) AS sale_hour,
        store_id,
        count(*) AS transaction_count,
        COALESCE(sum(total_amount), (0)::bigint) AS total_revenue
       FROM public.sales
      WHERE ((status)::text = 'completed'::text)
      GROUP BY (date_trunc('hour'::text, (created_at AT TIME ZONE 'Asia/Jakarta'::text))), store_id
   ) s
   FULL JOIN (
     SELECT date_trunc('hour'::text, (created_at AT TIME ZONE 'Asia/Jakarta'::text)) AS return_hour,
        store_id,
        COALESCE(sum(total_refund_amount), (0)::bigint) AS refund_amount
       FROM public.sale_returns
      WHERE ((status)::text = 'completed'::text)
      GROUP BY (date_trunc('hour'::text, (created_at AT TIME ZONE 'Asia/Jakarta'::text))), store_id
   ) r ON r.return_hour = s.sale_hour AND r.store_id = s.store_id
  WITH DATA;

CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_hourly_sales_hour_store
    ON public.mv_hourly_sales USING btree (sale_hour, store_id);
