package report

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"retail-pos-system/internal/config"
	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/sale"
	"retail-pos-system/internal/shared"
	"retail-pos-system/pkg/cache"
)

type Repository struct {
	db    shared.DBPool
	cache *cache.Cache
}

func NewRepository(db shared.DBPool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) SetCache(c *cache.Cache) {
	r.cache = c
}

func (r *Repository) InvalidateDashboardCache(storeID *int) {
	if r.cache == nil {
		return
	}
	r.cache.Delete("dashboard:stats")
	r.cache.Delete("dashboard:live")
	if storeID != nil {
		r.cache.Delete(fmt.Sprintf("dashboard:stats:store:%d", *storeID))
		r.cache.Delete(fmt.Sprintf("dashboard:live:store:%d", *storeID))
	}
}

func (r *Repository) NewSaleCreatedListener() *saleCreatedListener {
	return &saleCreatedListener{repo: r}
}

type saleCreatedListener struct {
	repo *Repository
}

func (l *saleCreatedListener) EventTypes() []eventbus.EventType {
	return []eventbus.EventType{eventbus.SaleCreated}
}

func (l *saleCreatedListener) HandleEvent(ctx context.Context, event eventbus.Event) error {
	s, ok := event.Payload.(*sale.Sale)
	if !ok {
		return nil
	}
	l.repo.InvalidateDashboardCache(s.StoreID)
	return nil
}

func (r *Repository) GetPeriodComparison(
	ctx context.Context,
	currentStart, currentEnd time.Time,
	previousStart, previousEnd time.Time,
	storeID *int,
) (*PeriodComparison, error) {

	key := fmt.Sprintf("report:comparison:%s:%s:%s:%s", currentStart.Format("20060102"), currentEnd.Format("20060102"), previousStart.Format("20060102"), previousEnd.Format("20060102"))
	if storeID != nil {
		key += fmt.Sprintf(":store:%d", *storeID)
	}
	if r.cache != nil {
		if cached, found := r.cache.Get(key); found {
			return cached.(*PeriodComparison), nil
		}
	}

	query := `
		WITH period_stats AS (
			SELECT
				COALESCE(SUM(CASE WHEN created_at >= $1 AND created_at < $2 THEN total_amount ELSE 0 END), 0) as current_revenue,
				COUNT(CASE WHEN created_at >= $1 AND created_at < $2 THEN 1 END) as current_orders,
				COALESCE(SUM(CASE WHEN created_at >= $3 AND created_at < $4 THEN total_amount ELSE 0 END), 0) as previous_revenue,
				COUNT(CASE WHEN created_at >= $3 AND created_at < $4 THEN 1 END) as previous_orders
			FROM sales
			WHERE ((created_at >= $1 AND created_at < $2) OR (created_at >= $3 AND created_at < $4))
				AND status = 'completed'
		),
		peak_hours AS (
			SELECT
				MAX(CASE WHEN period = 'current' THEN hourly_total ELSE 0 END) as current_peak,
				MAX(CASE WHEN period = 'previous' THEN hourly_total ELSE 0 END) as previous_peak
			FROM (
				SELECT
					SUM(total_amount) as hourly_total,
					period
				FROM (
					SELECT total_amount,
						CASE WHEN created_at >= $1 AND created_at < $2 THEN 'current' ELSE 'previous' END as period,
						EXTRACT(HOUR FROM (created_at AT TIME ZONE 'Asia/Jakarta')) as hour
					FROM sales
					WHERE ((created_at >= $1 AND created_at < $2) OR (created_at >= $3 AND created_at < $4))
						AND status = 'completed'
				) tagged
				GROUP BY hour, period
			) hourly
		),
		peak_months AS (
			SELECT
				MAX(CASE WHEN period = 'current' THEN monthly_total ELSE 0 END) as current_peak,
				MAX(CASE WHEN period = 'previous' THEN monthly_total ELSE 0 END) as previous_peak
			FROM (
				SELECT
					SUM(total_amount) as monthly_total,
					period
				FROM (
					SELECT total_amount,
						CASE WHEN created_at >= $1 AND created_at < $2 THEN 'current' ELSE 'previous' END as period,
						EXTRACT(YEAR FROM (created_at AT TIME ZONE 'Asia/Jakarta')) as yr,
						EXTRACT(MONTH FROM (created_at AT TIME ZONE 'Asia/Jakarta')) as mo
					FROM sales
					WHERE ((created_at >= $1 AND created_at < $2) OR (created_at >= $3 AND created_at < $4))
						AND status = 'completed'
				) tagged
				GROUP BY yr, mo, period
			) monthly
		)
		SELECT
			ps.current_revenue, ps.current_orders,
			ps.previous_revenue, ps.previous_orders,
			COALESCE(ph.current_peak, 0), COALESCE(ph.previous_peak, 0),
			COALESCE(pm.current_peak, 0), COALESCE(pm.previous_peak, 0)
		FROM period_stats ps, peak_hours ph, peak_months pm`

	args := []interface{}{currentStart, currentEnd, previousStart, previousEnd, storeID}
	storeFilter := ` AND (store_id = $5 OR $5 IS NULL)`

	query = strings.ReplaceAll(query, "AND status = 'completed'", storeFilter+" AND status = 'completed'")

	var result PeriodComparison
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&result.CurrentRevenue,
		&result.CurrentOrders,
		&result.PreviousRevenue,
		&result.PreviousOrders,
		&result.PeakRevenueHour,
		&result.PreviousPeakRevenue,
		&result.PeakRevenueMonth,
		&result.PreviousPeakRevenueMonth,
	)

	if err != nil {
		return nil, err
	}

	days := int(currentEnd.Sub(currentStart).Hours() / 24)
	if days == 0 {
		days = 1
	}

	if result.CurrentOrders > 0 {
		result.CurrentAOV = result.CurrentRevenue / result.CurrentOrders
	}
	if result.PreviousOrders > 0 {
		result.PreviousAOV = result.PreviousRevenue / result.PreviousOrders
	}

	result.RevenuePerDay = int(math.Round(float64(result.CurrentRevenue) / float64(days)))
	result.PreviousRevenuePerDay = int(math.Round(float64(result.PreviousRevenue) / float64(days)))

	if r.cache != nil {
		ttl := 5*time.Second + time.Duration(rand.Intn(5))*time.Second
		r.cache.SetWithTTL(key, &result, ttl)
	}

	return &result, nil
}

func (r *Repository) GetDualChartData(
	ctx context.Context,
	currentStart, currentEnd, previousStart, previousEnd time.Time,
	storeID *int,
) (current, previous []ChartDataPoint, err error) {

	cs := currentStart.Format("2006-01-02")
	ce := currentEnd.Format("2006-01-02")
	ps := previousStart.Format("2006-01-02")
	pe := previousEnd.Format("2006-01-02")

	storeFilter := ""
	args := []interface{}{cs, ce, ps, pe}
	if storeID != nil {
		storeFilter = " AND store_id = $5"
		args = append(args, storeID)
	}

	query := `
		WITH date_series AS (
			SELECT generate_series(($1::date AT TIME ZONE 'Asia/Jakarta'), ($2::date AT TIME ZONE 'Asia/Jakarta'), '1 day') AS dt
		),
		current_agg AS (
			SELECT (created_at AT TIME ZONE 'Asia/Jakarta')::date AS dt,
				   COALESCE(SUM(total_amount), 0) AS revenue
			FROM sales
			WHERE created_at >= ($1::date AT TIME ZONE 'Asia/Jakarta') AND created_at < (($2::date + 1) AT TIME ZONE 'Asia/Jakarta')
				AND status = 'completed'` + storeFilter + `
			GROUP BY 1
		),
		previous_agg AS (
			SELECT (created_at AT TIME ZONE 'Asia/Jakarta')::date AS dt,
				   COALESCE(SUM(total_amount), 0) AS revenue
			FROM sales
			WHERE created_at >= ($3::date AT TIME ZONE 'Asia/Jakarta') AND created_at < (($4::date + 1) AT TIME ZONE 'Asia/Jakarta')
				AND status = 'completed'` + storeFilter + `
			GROUP BY 1
		)
		SELECT (ds.dt AT TIME ZONE 'Asia/Jakarta')::date,
			   COALESCE(c.revenue, 0),
			   COALESCE(p.revenue, 0)
		FROM date_series ds
		LEFT JOIN current_agg c ON c.dt = (ds.dt AT TIME ZONE 'Asia/Jakarta')::date
		LEFT JOIN previous_agg p ON p.dt = (ds.dt AT TIME ZONE 'Asia/Jakarta')::date - ($1::date - $3::date)
		ORDER BY 1`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var c ChartDataPoint
		var p ChartDataPoint
		var prevTotal int
		var currentDate time.Time
		if err := rows.Scan(&currentDate, &c.Total, &prevTotal); err != nil {
			return nil, nil, fmt.Errorf("scan row: %w", err)
		}
		c.Date = currentDate.Format("2006-01-02")
		offsetDays := int(currentStart.Sub(previousStart).Hours() / 24)
		prevTime := currentDate.AddDate(0, 0, -offsetDays)
		p.Date = prevTime.Format("2006-01-02")
		p.Total = prevTotal
		current = append(current, c)
		previous = append(previous, p)
	}

	return current, previous, rows.Err()
}

type liveDashboardResult struct {
	todaysRevenue  int
	todaysSales    int
	totalProducts  int
	lowStockCount  int
}

func (r *Repository) GetLiveDashboardStats(ctx context.Context, storeID *int) (todaysRevenue, todaysSales, totalProducts, lowStockCount int, err error) {
	key := "dashboard:live"
	if storeID != nil {
		key = fmt.Sprintf("dashboard:live:store:%d", *storeID)
	}
	if r.cache != nil {
		if cached, found := r.cache.Get(key); found {
			res := cached.(*liveDashboardResult)
			return res.todaysRevenue, res.todaysSales, res.totalProducts, res.lowStockCount, nil
		}
	}

	jakartaNow := time.Now().In(shared.JakartaLocation())
	todayStart := time.Date(jakartaNow.Year(), jakartaNow.Month(), jakartaNow.Day(), 0, 0, 0, 0, shared.JakartaLocation())
	todayEnd := todayStart.Add(24 * time.Hour)

	cfg := config.Load()
	storeFilter := ""
	args := []interface{}{todayStart, todayEnd, cfg.StockCriticalThreshold}
	argIdx := 4
	if storeID != nil {
		storeFilter = fmt.Sprintf(" AND store_id = $%d", argIdx)
		args = append(args, *storeID)
	}

	query := `
		WITH today_sales AS (
			SELECT COALESCE(SUM(total_amount), 0) AS revenue, COUNT(*) AS orders
			FROM sales
			WHERE created_at >= $1 AND created_at < $2 AND status = 'completed'` + storeFilter + `
		),
		product_count AS (
			SELECT COUNT(*) AS total
			FROM products
			WHERE deleted_at IS NULL` + storeFilter + `
		),
		stock_count AS (
			SELECT COUNT(*) AS low
			FROM product_stock
			WHERE quantity <= $3` + storeFilter + `
		)
		SELECT ts.revenue, ts.orders, pc.total, sc.low
		FROM today_sales ts, product_count pc, stock_count sc`

	var todaysRevInt, todaysSalesInt int
	var totalProductsInt, lowStockCountInt int64
	if err := r.db.QueryRow(ctx, query, args...).Scan(
		&todaysRevInt, &todaysSalesInt, &totalProductsInt, &lowStockCountInt,
	); err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to query live dashboard stats: %w", err)
	}
	todaysRevenue = todaysRevInt
	todaysSales = todaysSalesInt
	totalProducts = int(totalProductsInt)
	lowStockCount = int(lowStockCountInt)

	if r.cache != nil {
		ttl := 10*time.Second + time.Duration(rand.Intn(5))*time.Second
		r.cache.SetWithTTL(key, &liveDashboardResult{todaysRevenue, todaysSales, totalProducts, lowStockCount}, ttl)
	}

	return
}

func (r *Repository) GetAvailableYears(ctx context.Context, storeID *int) ([]int, error) {
	query := `
		SELECT DISTINCT EXTRACT(YEAR FROM (created_at AT TIME ZONE 'Asia/Jakarta'))::integer as year
		FROM sales
		WHERE status = 'completed'
	`
	args := []interface{}{}
	argIdx := 1

	if storeID != nil {
		query += fmt.Sprintf(" AND store_id = $%d", argIdx)
		args = append(args, *storeID)
	}

	query += " ORDER BY year DESC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch available years: %w", err)
	}
	defer rows.Close()

	var years []int
	for rows.Next() {
		var year int
		if err := rows.Scan(&year); err != nil {
			continue
		}
		years = append(years, year)
	}
	if years == nil {
		years = []int{}
	}

	return years, nil
}

func (r *Repository) GetHourlySales(ctx context.Context, date time.Time, storeID *int) ([]ChartDataPoint, error) {
	end := date.Add(24 * time.Hour)
	query := `
		SELECT EXTRACT(HOUR FROM (created_at AT TIME ZONE 'Asia/Jakarta'))::int AS hour,
			   COALESCE(SUM(total_amount), 0) AS revenue
		FROM sales
		WHERE created_at >= $1 AND created_at < $2
		  AND status = 'completed'`
	args := []interface{}{date, end}
	argIdx := 3
	if storeID != nil {
		query += fmt.Sprintf(" AND store_id = $%d", argIdx)
		args = append(args, *storeID)
	}
	query += " GROUP BY 1 ORDER BY 1"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query hourly sales: %w", err)
	}
	defer rows.Close()

	var result []ChartDataPoint
	for rows.Next() {
		var hour int
		var dp ChartDataPoint
		if err := rows.Scan(&hour, &dp.Total); err != nil {
			return nil, fmt.Errorf("scan hourly sales row: %w", err)
		}
		dp.Date = fmt.Sprintf("%02d", hour)
		result = append(result, dp)
	}
	if result == nil {
		result = []ChartDataPoint{}
	}
	return result, rows.Err()
}

func (r *Repository) GetDailySales(ctx context.Context, start, end time.Time, storeID *int) ([]ChartDataPoint, error) {
	query := `
		SELECT (created_at AT TIME ZONE 'Asia/Jakarta')::date AS dt,
			   COALESCE(SUM(total_amount), 0) AS revenue
		FROM sales
		WHERE created_at >= $1 AND created_at < $2
		  AND status = 'completed'`
	args := []interface{}{start, end}
	argIdx := 3
	if storeID != nil {
		query += fmt.Sprintf(" AND store_id = $%d", argIdx)
		args = append(args, *storeID)
	}
	query += " GROUP BY 1 ORDER BY 1"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily sales: %w", err)
	}
	defer rows.Close()

	var result []ChartDataPoint
	for rows.Next() {
		var dp ChartDataPoint
		var t time.Time
		if err := rows.Scan(&t, &dp.Total); err != nil {
			return nil, fmt.Errorf("scan daily sales row: %w", err)
		}
		dp.Date = t.Format("2006-01-02")
		result = append(result, dp)
	}
	if result == nil {
		result = []ChartDataPoint{}
	}
	return result, rows.Err()
}

func (r *Repository) GetSalesWeeklyReport(ctx context.Context, start, end time.Time, storeID *int) ([]WeeklyReportItem, error) {
	query := `
		WITH weeks AS (
			SELECT
				date_trunc('week', (created_at AT TIME ZONE 'Asia/Jakarta')::date)::date AS week_start,
				(date_trunc('week', (created_at AT TIME ZONE 'Asia/Jakarta')::date) + interval '6 days')::date AS week_end,
				total_amount
			FROM sales
			WHERE created_at >= $1 AND created_at < $2
			  AND status = 'completed'`
	args := []interface{}{start, end}
	argIdx := 3
	if storeID != nil {
		query += fmt.Sprintf(" AND store_id = $%d", argIdx)
		args = append(args, *storeID)
	}
	query += `
		)
		SELECT week_start::text, week_end::text,
			   COALESCE(SUM(total_amount), 0)::bigint,
			   COUNT(*)::integer
		FROM weeks
		GROUP BY week_start, week_end
		ORDER BY week_start`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query weekly report: %w", err)
	}
	defer rows.Close()

	var result []WeeklyReportItem
	for rows.Next() {
		var item WeeklyReportItem
		if err := rows.Scan(&item.WeekStart, &item.WeekEnd, &item.Total, &item.OrderCount); err != nil {
			return nil, fmt.Errorf("scan weekly report row: %w", err)
		}
		result = append(result, item)
	}
	if result == nil {
		result = []WeeklyReportItem{}
	}
	return result, rows.Err()
}

func (r *Repository) GetSalesMonthlyReport(ctx context.Context, start, end time.Time, storeID *int) ([]MonthlyReportItem, error) {
	query := `
		WITH months AS (
			SELECT
				to_char((created_at AT TIME ZONE 'Asia/Jakarta'), 'YYYY-MM') AS month,
				date_trunc('month', (created_at AT TIME ZONE 'Asia/Jakarta')::date)::date AS month_start,
				total_amount
			FROM sales
			WHERE created_at >= $1 AND created_at < $2
			  AND status = 'completed'`
	args := []interface{}{start, end}
	argIdx := 3
	if storeID != nil {
		query += fmt.Sprintf(" AND store_id = $%d", argIdx)
		args = append(args, *storeID)
	}
	query += `
		)
		SELECT month, month_start::text,
			   COALESCE(SUM(total_amount), 0)::bigint,
			   COUNT(*)::integer
		FROM months
		GROUP BY month, month_start
		ORDER BY month`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query monthly report: %w", err)
	}
	defer rows.Close()

	var result []MonthlyReportItem
	for rows.Next() {
		var item MonthlyReportItem
		if err := rows.Scan(&item.Month, &item.MonthStart, &item.Total, &item.OrderCount); err != nil {
			return nil, fmt.Errorf("scan monthly report row: %w", err)
		}
		result = append(result, item)
	}
	if result == nil {
		result = []MonthlyReportItem{}
	}
	return result, rows.Err()
}

func (r *Repository) GetDashboardStats(ctx context.Context, storeID *int, jakartaLoc *time.Location) (*DashboardStats, error) {
	key := "dashboard:stats"
	if storeID != nil {
		key = fmt.Sprintf("dashboard:stats:store:%d", *storeID)
	}
	if r.cache != nil {
		if cached, found := r.cache.Get(key); found {
			return cached.(*DashboardStats), nil
		}
	}

	now := time.Now().In(jakartaLoc)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, jakartaLoc)
	todayEnd := todayStart.Add(24 * time.Hour)

	cfg := config.Load()
	storeFilter := ""
	args := []interface{}{todayStart, todayEnd, cfg.StockCriticalThreshold}
	argIdx := 4
	if storeID != nil {
		storeFilter = fmt.Sprintf(" AND store_id = $%d", argIdx)
		args = append(args, *storeID)
		argIdx++
	}

	query := `
		WITH today_sales AS (
			SELECT COALESCE(SUM(total_amount), 0) AS revenue, COUNT(*) AS orders
			FROM sales
			WHERE created_at >= $1 AND created_at < $2 AND status = 'completed'` + storeFilter + `
		),
		total_sales AS (
			SELECT COALESCE(SUM(total_amount), 0) AS revenue, COUNT(*) AS orders
			FROM sales
			WHERE status = 'completed'` + storeFilter + `
		),
		product_count AS (
			SELECT COUNT(*) AS total
			FROM products
			WHERE deleted_at IS NULL` + storeFilter + `
		),
		stock_count AS (
			SELECT COUNT(*) AS low
			FROM product_stock
			WHERE quantity <= $3` + storeFilter + `
		),
		customer_count AS (
			SELECT COUNT(DISTINCT customer_id) AS active
			FROM sales
			WHERE status = 'completed' AND customer_id IS NOT NULL` + storeFilter + `
		)
		SELECT ts.revenue, ts.orders,
		       tots.revenue, tots.orders,
		       pc.total, sc.low, cc.active
		FROM today_sales ts, total_sales tots, product_count pc, stock_count sc, customer_count cc`

	var stats DashboardStats
	var todaysRevInt, todaysSalesInt int
	var totalRevInt, totalSalesInt int
	var totalProducts int64
	var lowStockCount int64
	var activeCustomers int64
	if err := r.db.QueryRow(ctx, query, args...).Scan(
		&todaysRevInt, &todaysSalesInt,
		&totalRevInt, &totalSalesInt,
		&totalProducts, &lowStockCount, &activeCustomers,
	); err != nil {
		return nil, fmt.Errorf("failed to query dashboard stats: %w", err)
	}
	stats.TodaysRevenue = int64(todaysRevInt)
	stats.TodaysSales = int64(todaysSalesInt)
	stats.TotalRevenue = int64(totalRevInt)
	stats.TotalSales = int64(totalSalesInt)
	stats.TotalProducts = totalProducts
	stats.LowStockCount = lowStockCount
	stats.ActiveCustomers = activeCustomers

	if r.cache != nil {
		ttl := 10*time.Second + time.Duration(rand.Intn(5))*time.Second
		r.cache.SetWithTTL(key, &stats, ttl)
	}

	return &stats, nil
}
