-- Materialized views for analytical queries
-- ============================================================
-- These views pre-aggregate sales data to speed up chart and
-- report queries. Refresh via refresh_sales_mv() function.
-- ============================================================

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_daily_sales AS
SELECT
    DATE(created_at AT TIME ZONE 'Asia/Jakarta') as sale_date,
    store_id,
    COUNT(*) as transaction_count,
    COUNT(DISTINCT cashier_id) as active_cashiers,
    COALESCE(SUM(total_amount), 0) as total_revenue,
    COALESCE(SUM(subtotal), 0) as total_subtotal,
    COALESCE(SUM(discount), 0) as total_discount,
    COALESCE(SUM(tax), 0) as total_tax
FROM sales
WHERE status = 'completed'
GROUP BY DATE(created_at AT TIME ZONE 'Asia/Jakarta'), store_id
WITH DATA;

CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_daily_sales_date_store ON mv_daily_sales(sale_date, store_id);

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_hourly_sales AS
SELECT
    DATE_TRUNC('hour', created_at AT TIME ZONE 'Asia/Jakarta') as sale_hour,
    store_id,
    COUNT(*) as transaction_count,
    COALESCE(SUM(total_amount), 0) as total_revenue
FROM sales
WHERE status = 'completed'
GROUP BY DATE_TRUNC('hour', created_at AT TIME ZONE 'Asia/Jakarta'), store_id
WITH DATA;

CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_hourly_sales_hour_store ON mv_hourly_sales(sale_hour, store_id);

CREATE OR REPLACE FUNCTION refresh_sales_mv()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_daily_sales;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_hourly_sales;
END;
$$ LANGUAGE plpgsql;
