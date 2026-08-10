package shared

// WeeklyReportItem is the cross-module contract for weekly sales aggregation.
type WeeklyReportItem struct {
	WeekStart  string `json:"week_start"`
	WeekEnd    string `json:"week_end"`
	Total      int    `json:"total"`
	OrderCount int    `json:"order_count"`
}

// MonthlyReportItem is the cross-module contract for monthly sales aggregation.
type MonthlyReportItem struct {
	Month      string `json:"month"`
	MonthStart string `json:"month_start"`
	Total      int    `json:"total"`
	OrderCount int    `json:"order_count"`
}

// PricingBreakdownItem is the cross-module contract for pricing type breakdown.
type PricingBreakdownItem struct {
	Type       string `json:"pricing_type"`
	Revenue    int    `json:"revenue"`
	OrderCount int    `json:"order_count"`
	ItemCount  int    `json:"item_count"`
}

// WeeklySalesQueryTemplate is the shared SQL for weekly sales aggregation.
// The caller appends the optional store_id filter and GROUP BY/ORDER BY clauses.
const WeeklySalesQueryTemplate = `
	SELECT
		date_trunc('week', (created_at AT TIME ZONE 'Asia/Jakarta')::date)::text AS week_start,
		(date_trunc('week', (created_at AT TIME ZONE 'Asia/Jakarta')::date) + interval '6 days')::text AS week_end,
		COALESCE(SUM(total_amount), 0)::bigint,
		COUNT(*)::integer
	FROM sales
	WHERE created_at >= $1 AND created_at < $2 AND status = 'completed'`

// MonthlySalesQueryTemplate is the shared SQL for monthly sales aggregation.
// The caller appends the optional store_id filter and GROUP BY/ORDER BY clauses.
const MonthlySalesQueryTemplate = `
	SELECT
		to_char((created_at AT TIME ZONE 'Asia/Jakarta'), 'YYYY-MM') AS month,
		date_trunc('month', (created_at AT TIME ZONE 'Asia/Jakarta')::date)::text AS month_start,
		COALESCE(SUM(total_amount), 0)::bigint,
		COUNT(*)::integer
	FROM sales
	WHERE created_at >= $1 AND created_at < $2 AND status = 'completed'`

// PricingBreakdownQueryTemplate is the shared SQL for pricing breakdown aggregation.
// The caller appends the optional store_id filter and GROUP BY/ORDER BY clauses.
const PricingBreakdownQueryTemplate = `
	SELECT COALESCE(si.pricing_type, 'normal') AS pricing_type,
	       SUM(si.unit_price * si.quantity) AS revenue,
	       COUNT(DISTINCT si.sale_id) AS order_count,
	       COUNT(*) AS item_count
	FROM sale_items si
	JOIN sales s ON si.sale_id = s.id
	WHERE s.status = 'completed'
	  AND s.created_at >= $1 AND s.created_at < $2`
