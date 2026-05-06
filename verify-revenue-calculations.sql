-- SQL Verification Queries for Revenue Calculations
-- Run these in PostgreSQL to verify the backend calculations

-- 1. Daily Revenue Check (for today)
SELECT
    DATE(created_at) as date,
    SUM(total_amount) as total_revenue,
    COUNT(*) as transaction_count
FROM sales
WHERE DATE(created_at) = CURRENT_DATE
    AND status = 'completed'
GROUP BY DATE(created_at);

-- 2. Weekly Revenue Check (last 12 weeks)
SELECT
    DATE_TRUNC('week', created_at) as week_start,
    (DATE_TRUNC('week', created_at) + INTERVAL '6 days') as week_end,
    SUM(total_amount) as total_revenue,
    COUNT(*) as transaction_count
FROM sales
WHERE created_at >= CURRENT_DATE - INTERVAL '84 days'
    AND status = 'completed'
GROUP BY DATE_TRUNC('week', created_at)
ORDER BY week_start DESC;

-- 3. Monthly Revenue Check (last 12 months)
SELECT
    TO_CHAR(created_at, 'YYYY-MM') as month,
    DATE_TRUNC('month', created_at) as month_start,
    SUM(total_amount) as total_revenue,
    COUNT(*) as transaction_count
FROM sales
WHERE created_at >= DATE_TRUNC('month', CURRENT_DATE - INTERVAL '11 months')
    AND status = 'completed'
GROUP BY TO_CHAR(created_at, 'YYYY-MM'), DATE_TRUNC('month', created_at)
ORDER BY month DESC;

-- 4. Total Products Count
SELECT COUNT(*) as total_products
FROM products
WHERE deleted_at IS NULL;

-- 5. Low Stock Count (stock <= 5)
SELECT COUNT(*) as low_stock_products
FROM products
WHERE deleted_at IS NULL
    AND stock <= 5;

-- 6. Previous Period Comparison (for % change verification)
-- Example: Compare this week vs last week
WITH current_week AS (
    SELECT SUM(total_amount) as revenue
    FROM sales
    WHERE created_at >= DATE_TRUNC('week', CURRENT_DATE)
        AND created_at < DATE_TRUNC('week', CURRENT_DATE) + INTERVAL '7 days'
        AND status = 'completed'
),
previous_week AS (
    SELECT SUM(total_amount) as revenue
    FROM sales
    WHERE created_at >= DATE_TRUNC('week', CURRENT_DATE - INTERVAL '7 days')
        AND created_at < DATE_TRUNC('week', CURRENT_DATE)
        AND status = 'completed'
)
SELECT
    cw.revenue as current_revenue,
    pw.revenue as previous_revenue,
    CASE
        WHEN pw.revenue > 0 THEN
            ROUND((((cw.revenue - pw.revenue)::numeric / pw.revenue) * 100)::numeric, 1)
        ELSE 0::numeric
    END as percent_change
FROM current_week cw, previous_week pw;