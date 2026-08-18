package report

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"sync"
	"time"

	"retail-pos-system/internal/config"
	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/events"
	"retail-pos-system/internal/shared"
	"retail-pos-system/pkg/cache"
)

type Repository struct {
	db           shared.DBPool
	cache        *cache.Cache
	saleStats    SaleStatsProvider
	productStats ProductStatsProvider
	stockStats   StockStatsProvider
}

func NewRepository(db shared.DBPool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) SetCache(c *cache.Cache) {
	r.cache = c
}

func (r *Repository) SetSaleStatsProvider(p SaleStatsProvider) {
	if r.saleStats != nil {
		panic("report: SaleStatsProvider already set")
	}
	r.saleStats = p
}

func (r *Repository) SetProductStatsProvider(p ProductStatsProvider) {
	if r.productStats != nil {
		panic("report: ProductStatsProvider already set")
	}
	r.productStats = p
}

func (r *Repository) SetStockStatsProvider(p StockStatsProvider) {
	if r.stockStats != nil {
		panic("report: StockStatsProvider already set")
	}
	r.stockStats = p
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
	r.cache.FlushByPrefix("report:")
}

// RefreshSalesMV executes a single full refresh of the reporting materialized
// views. Used by the RefreshCoordinator worker and by the startup/seed paths;
// SaleCreated event listeners never call it directly. After a successful
// refresh it invalidates the dashboard/report caches so the freshly rebuilt
// totals (including mv_dashboard_totals) are not masked by stale entries.
func (r *Repository) RefreshSalesMV(ctx context.Context) error {
	_, err := r.db.Exec(ctx, "SELECT refresh_sales_mv()")
	if err != nil {
		return fmt.Errorf("refresh sales mv: %w", err)
	}
	if r.cache != nil {
		r.cache.FlushByPrefix("dashboard:")
		r.cache.FlushByPrefix("report:")
	}
	return nil
}

func (r *Repository) NewSaleCreatedListener() eventbus.Listener {
	return &saleCreatedListener{repo: r}
}

type saleCreatedListener struct {
	repo *Repository
}

func (l *saleCreatedListener) EventTypes() []eventbus.EventType {
	return []eventbus.EventType{events.TopicSaleCreated}
}

func (l *saleCreatedListener) HandleEvent(ctx context.Context, event eventbus.Event) error {
	s, ok := event.Payload.(*events.SaleCreated)
	if !ok {
		return nil
	}
	l.repo.InvalidateDashboardCache(s.StoreID)
	// Reporting is allowed to be eventually consistent: the handler returns
	// without triggering a refresh. The RefreshCoordinator worker owns the
	// materialized view refresh — once at startup and then at each Jakarta
	// hour boundary — so a refresh failure can never fail the SaleCreated
	// event or trigger eventbus retries.
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

	// Aggregations read from mv_hourly_sales (refreshed by the coordinator at
	// each Jakarta hour boundary) instead of scanning the raw sales table,
	// keeping period comparisons cheap on large datasets. Hour-granularity rows
	// still preserve the exact instant boundaries used by realtime mode and
	// collapse to identical results for whole-day periods (peak month groups
	// hours by Jakarta calendar month).
	query := `
		WITH base AS (
			SELECT sale_hour, total_revenue, transaction_count
			FROM mv_hourly_sales
			WHERE ((sale_hour >= $1 AND sale_hour < $2) OR (sale_hour >= $3 AND sale_hour < $4))
				AND ($5::int IS NULL OR store_id = $5)
		),
		base_metrics AS (
			SELECT
				COALESCE(SUM(CASE WHEN sale_hour >= $1 AND sale_hour < $2 THEN total_revenue ELSE 0 END), 0) as current_revenue,
				COALESCE(SUM(CASE WHEN sale_hour >= $1 AND sale_hour < $2 THEN transaction_count ELSE 0 END), 0) as current_orders,
				COALESCE(SUM(CASE WHEN sale_hour >= $3 AND sale_hour < $4 THEN total_revenue ELSE 0 END), 0) as previous_revenue,
				COALESCE(SUM(CASE WHEN sale_hour >= $3 AND sale_hour < $4 THEN transaction_count ELSE 0 END), 0) as previous_orders
			FROM base
		),
		peak_hours AS (
			SELECT
				COALESCE(MAX(CASE WHEN sale_hour >= $1 AND sale_hour < $2 THEN total_revenue ELSE 0 END), 0) as current_peak_hour,
				COALESCE(MAX(CASE WHEN sale_hour >= $3 AND sale_hour < $4 THEN total_revenue ELSE 0 END), 0) as previous_peak_hour
			FROM base
		),
		monthly AS (
			SELECT
				CASE WHEN sale_hour >= $1 AND sale_hour < $2 THEN 'current' ELSE 'previous' END as period,
				SUM(total_revenue) as monthly_total
			FROM base
			GROUP BY to_char(sale_hour AT TIME ZONE 'Asia/Jakarta', 'YYYY-MM'), period
		),
		peak_months AS (
			SELECT
				COALESCE(MAX(CASE WHEN period = 'current' THEN monthly_total ELSE 0 END), 0) as current_peak_month,
				COALESCE(MAX(CASE WHEN period = 'previous' THEN monthly_total ELSE 0 END), 0) as previous_peak_month
			FROM monthly
		),
		previous_any AS (
			SELECT EXISTS(
				SELECT 1 FROM mv_hourly_sales
				WHERE sale_hour >= $3 AND sale_hour < $3 + interval '24 hours'
					AND ($5::int IS NULL OR store_id = $5)
			) as has_any
		)
		SELECT
			bm.current_revenue, bm.current_orders,
			bm.previous_revenue, bm.previous_orders,
			ph.current_peak_hour, ph.previous_peak_hour,
			pm.current_peak_month, pm.previous_peak_month,
			pa.has_any
		FROM base_metrics bm, peak_hours ph, peak_months pm, previous_any pa`

	args := []interface{}{currentStart, currentEnd, previousStart, previousEnd, storeID}

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
		&result.PreviousHasAnyData,
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
		ttl := 25*time.Second + time.Duration(rand.Intn(10))*time.Second
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

	key := fmt.Sprintf("report:dualchart:%s:%s:%s:%s", cs, ce, ps, pe)
	if storeID != nil {
		key += fmt.Sprintf(":store:%d", *storeID)
	}
	type dualResult struct {
		current  []ChartDataPoint
		previous []ChartDataPoint
	}
	if r.cache != nil {
		if cached, found := r.cache.Get(key); found {
			res := cached.(*dualResult)
			return res.current, res.previous, nil
		}
	}

	storeFilter := ""
	args := []interface{}{cs, ce, ps, pe}
	if storeID != nil {
		storeFilter = " AND store_id = $5"
		args = append(args, storeID)
	}

	query := `
		WITH date_series AS (
			SELECT generate_series($1::date, $2::date, '1 day') AS dt
		),
		current_agg AS (
			SELECT sale_date AS dt,
				   SUM(total_revenue) AS revenue
			FROM mv_daily_sales
			WHERE sale_date >= $1::date AND sale_date < ($2::date + 1)` + storeFilter + `
			GROUP BY sale_date
		),
		previous_agg AS (
			SELECT sale_date AS dt,
				   SUM(total_revenue) AS revenue
			FROM mv_daily_sales
			WHERE sale_date >= $3::date AND sale_date < ($4::date + 1)` + storeFilter + `
			GROUP BY sale_date
		)
		SELECT ds.dt,
			   COALESCE(c.revenue, 0),
			   COALESCE(p.revenue, 0)
		FROM date_series ds
		LEFT JOIN current_agg c ON c.dt = ds.dt
		LEFT JOIN previous_agg p ON p.dt = ds.dt::date - ($1::date - $3::date)
		ORDER BY ds.dt`

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

	if r.cache != nil && len(current) > 0 {
		ttl := 25*time.Second + time.Duration(rand.Intn(10))*time.Second
		r.cache.SetWithTTL(key, &dualResult{current, previous}, ttl)
	}

	return current, previous, rows.Err()
}

type liveDashboardResult struct {
	todaysRevenue int
	todaysSales   int
	totalProducts int
	lowStockCount int
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

	if r.saleStats == nil || r.productStats == nil || r.stockStats == nil {
		return 0, 0, 0, 0, fmt.Errorf("report: stats providers not wired; call SetSaleStatsProvider/SetProductStatsProvider/SetStockStatsProvider")
	}

	jakartaNow := time.Now().In(shared.JakartaLocation())
	todayStart := time.Date(jakartaNow.Year(), jakartaNow.Month(), jakartaNow.Day(), 0, 0, 0, 0, shared.JakartaLocation())
	todayEnd := todayStart.Add(24 * time.Hour)

	cfg := config.Load()

	todaysRevenue, todaysSales, err = r.saleStats.GetCompletedSalesStats(ctx, r.db, todayStart, todayEnd, storeID)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to get today's sales stats: %w", err)
	}

	var totalProductsInt64 int64
	totalProductsInt64, err = r.productStats.GetActiveProductCount(ctx, r.db, storeID)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to get product count: %w", err)
	}
	totalProducts = int(totalProductsInt64)

	var lowStockCountInt64 int64
	lowStockCountInt64, err = r.stockStats.GetLowStockCount(ctx, r.db, cfg.StockCriticalThreshold, storeID)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to get low stock count: %w", err)
	}
	lowStockCount = int(lowStockCountInt64)

	if r.cache != nil {
		ttl := 10*time.Second + time.Duration(rand.Intn(5))*time.Second
		r.cache.SetWithTTL(key, &liveDashboardResult{todaysRevenue, todaysSales, totalProducts, lowStockCount}, ttl)
	}

	return
}

func (r *Repository) GetAvailableYears(ctx context.Context, storeID *int) ([]int, error) {
	// Read from mv_daily_sales (Jakarta dates) instead of scanning the raw
	// sales table; the view holds the same completed-sale rows grouped by
	// Jakarta day, so the set of distinct years is equivalent.
	query := `
		SELECT DISTINCT EXTRACT(YEAR FROM sale_date)::integer as year
		FROM mv_daily_sales
	`
	args := []interface{}{}
	argIdx := 1

	if storeID != nil {
		query += fmt.Sprintf(" WHERE store_id = $%d", argIdx)
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
	// A chart for today must never surface the in-progress hour: cap the upper
	// bound at the start of the current hour so a partial bucket (e.g. created
	// by a mid-hour refresh on startup/retry/manual) is excluded. Completed
	// past days span the full 24 hours.
	nowJakarta := time.Now().In(shared.JakartaLocation())
	if date.Year() == nowJakarta.Year() && date.YearDay() == nowJakarta.YearDay() {
		hourStart := time.Date(nowJakarta.Year(), nowJakarta.Month(), nowJakarta.Day(), nowJakarta.Hour(), 0, 0, 0, nowJakarta.Location())
		if hourStart.Before(end) {
			end = hourStart
		}
	}
	query := `
		SELECT EXTRACT(HOUR FROM sale_hour)::int AS hour,
			   SUM(total_revenue) AS revenue
		FROM mv_hourly_sales
		WHERE sale_hour >= date_trunc('hour', $1::timestamptz AT TIME ZONE 'Asia/Jakarta')
		  AND sale_hour < date_trunc('hour', $2::timestamptz AT TIME ZONE 'Asia/Jakarta')`
	args := []interface{}{date, end}
	argIdx := 3
	if storeID != nil {
		query += fmt.Sprintf(" AND store_id = $%d", argIdx)
		args = append(args, *storeID)
	}
	query += " GROUP BY sale_hour ORDER BY sale_hour"

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
		SELECT sale_date AS dt,
			   SUM(total_revenue) AS revenue
		FROM mv_daily_sales
		WHERE sale_date >= ($1::timestamptz AT TIME ZONE 'Asia/Jakarta')::date
		  AND sale_date < ($2::timestamptz AT TIME ZONE 'Asia/Jakarta')::date`
	args := []interface{}{start, end}
	argIdx := 3
	if storeID != nil {
		query += fmt.Sprintf(" AND store_id = $%d", argIdx)
		args = append(args, *storeID)
	}
	query += " GROUP BY sale_date ORDER BY sale_date"

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

func (r *Repository) GetSalesWeeklyReport(ctx context.Context, start, end time.Time, storeID *int) ([]shared.WeeklyReportItem, error) {
	if r.saleStats == nil {
		return nil, fmt.Errorf("report: SaleStatsProvider not wired; call SetSaleStatsProvider")
	}
	return r.saleStats.GetWeeklySales(ctx, r.db, start, end, storeID)
}

func (r *Repository) GetSalesMonthlyReport(ctx context.Context, start, end time.Time, storeID *int) ([]shared.MonthlyReportItem, error) {
	if r.saleStats == nil {
		return nil, fmt.Errorf("report: SaleStatsProvider not wired; call SetSaleStatsProvider")
	}
	return r.saleStats.GetMonthlySales(ctx, r.db, start, end, storeID)
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

	if r.saleStats == nil || r.productStats == nil || r.stockStats == nil {
		return nil, fmt.Errorf("report: stats providers not wired; call SetSaleStatsProvider/SetProductStatsProvider/SetStockStatsProvider")
	}

	now := time.Now().In(jakartaLoc)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, jakartaLoc)
	todayEnd := todayStart.Add(24 * time.Hour)

	var stats DashboardStats
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(5)

	go func() {
		defer wg.Done()
		rev, sales, err := r.saleStats.GetCompletedSalesStats(ctx, r.db, todayStart, todayEnd, storeID)
		if err != nil {
			slog.Error("dashboard: today stats failed", "err", err)
			return
		}
		mu.Lock()
		stats.TodaysRevenue = int64(rev)
		stats.TodaysSales = int64(sales)
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		rev, sales, err := r.saleStats.GetAllCompletedSalesStats(ctx, r.db, storeID)
		if err != nil {
			slog.Error("dashboard: all-time stats failed", "err", err)
			return
		}
		mu.Lock()
		stats.TotalRevenue = int64(rev)
		stats.TotalSales = int64(sales)
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		count, err := r.saleStats.GetActiveCustomerCount(ctx, r.db, storeID)
		if err != nil {
			slog.Error("dashboard: active customer count failed", "err", err)
			return
		}
		mu.Lock()
		stats.ActiveCustomers = count
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		count, err := r.productStats.GetActiveProductCount(ctx, r.db, storeID)
		if err != nil {
			slog.Error("dashboard: product count failed", "err", err)
			return
		}
		mu.Lock()
		stats.TotalProducts = count
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		count, err := r.stockStats.GetLowStockCount(ctx, r.db, config.Load().StockCriticalThreshold, storeID)
		if err != nil {
			slog.Error("dashboard: low stock count failed", "err", err)
			return
		}
		mu.Lock()
		stats.LowStockCount = count
		mu.Unlock()
	}()

	wg.Wait()

	if r.cache != nil {
		ttl := 10*time.Second + time.Duration(rand.Intn(5))*time.Second
		r.cache.SetWithTTL(key, &stats, ttl)
	}

	return &stats, nil
}

func (r *Repository) GetPricingBreakdown(ctx context.Context, start, end time.Time, storeID *int) ([]shared.PricingBreakdownItem, error) {
	if r.saleStats == nil {
		return nil, fmt.Errorf("report: SaleStatsProvider not wired; call SetSaleStatsProvider")
	}
	return r.saleStats.GetPricingBreakdown(ctx, r.db, start, end, storeID)
}
