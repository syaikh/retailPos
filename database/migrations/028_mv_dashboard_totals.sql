-- Materialized view for all-time dashboard totals
-- ============================================================
-- Pre-aggregates completed sales per store so the dashboard's
-- all-time total card reads a few rows instead of scanning the raw
-- sales table. The single NULL group covers sales without a store
-- (pre-multi-store data). Refreshed inside refresh_sales_mv().
-- ============================================================

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_dashboard_totals AS
SELECT
    store_id,
    COUNT(*) AS transaction_count,
    COALESCE(SUM(total_amount), 0) AS total_revenue
FROM sales
WHERE status = 'completed'
GROUP BY store_id
WITH DATA;

CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_dashboard_totals_store ON mv_dashboard_totals(store_id);

CREATE OR REPLACE FUNCTION refresh_sales_mv()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_daily_sales;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_hourly_sales;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_dashboard_totals;
END;
$$ LANGUAGE plpgsql;