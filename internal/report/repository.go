package report

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"retail-pos-system/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

var jakartaLoc *time.Location

func init() {
	var err error
	jakartaLoc, err = time.LoadLocation("Asia/Jakarta")
	if err != nil {
		jakartaLoc = time.UTC
	}
}

func mustLoadJakarta() *time.Location {
	return jakartaLoc
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetPeriodComparison(
	ctx context.Context,
	currentStart, currentEnd time.Time,
	previousStart, previousEnd time.Time,
	storeID *int,
) (*PeriodComparison, error) {

	query := `
		WITH current_period AS (
			SELECT
				COALESCE(SUM(total_amount), 0) as revenue,
				COUNT(*) as orders
			FROM sales
			WHERE created_at >= $1 AND created_at < $2
				AND status = 'completed'
		),
		previous_period AS (
			SELECT
				COALESCE(SUM(total_amount), 0) as revenue,
				COUNT(*) as orders
			FROM sales
			WHERE created_at >= $3 AND created_at < $4
				AND status = 'completed'
		),
		current_peak_hour AS (
			SELECT COALESCE(MAX(hourly_total), 0) as peak_revenue
			FROM (
				SELECT SUM(total_amount) as hourly_total
				FROM sales
				WHERE created_at >= $1 AND created_at < $2
					AND status = 'completed'
				GROUP BY EXTRACT(HOUR FROM (created_at AT TIME ZONE 'Asia/Jakarta'))
			) hourly
		),
		previous_peak_hour AS (
			SELECT COALESCE(MAX(hourly_total), 0) as peak_revenue
			FROM (
				SELECT SUM(total_amount) as hourly_total
				FROM sales
				WHERE created_at >= $3 AND created_at < $4
					AND status = 'completed'
				GROUP BY EXTRACT(HOUR FROM (created_at AT TIME ZONE 'Asia/Jakarta'))
			) hourly
		),
		current_peak_month AS (
			SELECT COALESCE(MAX(monthly_total), 0) as peak_revenue
			FROM (
				SELECT SUM(total_amount) as monthly_total
				FROM sales
				WHERE created_at >= $1 AND created_at < $2
					AND status = 'completed'
				GROUP BY EXTRACT(YEAR FROM (created_at AT TIME ZONE 'Asia/Jakarta')),
				         EXTRACT(MONTH FROM (created_at AT TIME ZONE 'Asia/Jakarta'))
			) monthly
		),
		previous_peak_month AS (
			SELECT COALESCE(MAX(monthly_total), 0) as peak_revenue
			FROM (
				SELECT SUM(total_amount) as monthly_total
				FROM sales
				WHERE created_at >= $3 AND created_at < $4
					AND status = 'completed'
				GROUP BY EXTRACT(YEAR FROM (created_at AT TIME ZONE 'Asia/Jakarta')),
				         EXTRACT(MONTH FROM (created_at AT TIME ZONE 'Asia/Jakarta'))
			) monthly
		)
		SELECT
			cp.revenue, cp.orders,
			pp.revenue, pp.orders,
			cpeak_hour.peak_revenue, ppeak_hour.peak_revenue,
			cpeak_month.peak_revenue, ppeak_month.peak_revenue
		FROM current_period cp, previous_period pp,
		     current_peak_hour cpeak_hour, previous_peak_hour ppeak_hour,
		     current_peak_month cpeak_month, previous_peak_month ppeak_month`

	args := []interface{}{currentStart, currentEnd, previousStart, previousEnd, storeID}
	storeFilter := ` AND (store_id = $5 OR $5 IS NULL)`

	// Inject storeFilter into every subquery that filters on created_at
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

func (r *Repository) GetLiveDashboardStats(ctx context.Context, storeID *int) (todaysRevenue, todaysSales, totalProducts, lowStockCount int, err error) {
	jakartaNow := time.Now().In(mustLoadJakarta())
	todayStart := time.Date(jakartaNow.Year(), jakartaNow.Month(), jakartaNow.Day(), 0, 0, 0, 0, mustLoadJakarta())
	todayEnd := todayStart.Add(24 * time.Hour)

	todayQuery := `
		SELECT COALESCE(SUM(total_amount), 0), COUNT(*)
		FROM sales
		WHERE created_at >= $1 AND created_at < $2
		  AND status = 'completed'`

	args := []interface{}{todayStart, todayEnd}
	argIdx := 3
	if storeID != nil {
		todayQuery += fmt.Sprintf(" AND store_id = $%d", argIdx)
		args = append(args, *storeID)
	}

	if err := r.db.QueryRow(ctx, todayQuery, args...).Scan(&todaysRevenue, &todaysSales); err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to query live dashboard sales: %w", err)
	}

	productsQuery := `SELECT COUNT(*) FROM products WHERE deleted_at IS NULL`
	args2 := []interface{}{}
	argIdx2 := 1
	if storeID != nil {
		productsQuery += fmt.Sprintf(" AND store_id = $%d", argIdx2)
		args2 = append(args2, *storeID)
	}
	if err := r.db.QueryRow(ctx, productsQuery, args2...).Scan(&totalProducts); err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to query live dashboard products: %w", err)
	}

	cfg := config.Load()
	stockQuery := `SELECT COUNT(*) FROM product_stock WHERE quantity <= $1`
	stockArgs := []interface{}{cfg.StockCriticalThreshold}
	stockIdx := 2
	if storeID != nil {
		stockQuery += fmt.Sprintf(" AND store_id = $%d", stockIdx)
		stockArgs = append(stockArgs, *storeID)
	}
	if err := r.db.QueryRow(ctx, stockQuery, stockArgs...).Scan(&lowStockCount); err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to query live dashboard low stock: %w", err)
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
	now := time.Now().In(jakartaLoc)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, jakartaLoc)
	todayEnd := todayStart.Add(24 * time.Hour)

	var stats DashboardStats

	todayQuery := `
		SELECT COALESCE(SUM(total_amount), 0), COUNT(*)
		FROM sales
		WHERE created_at >= $1 AND created_at < $2
		  AND status = 'completed'`
	args := []interface{}{todayStart, todayEnd}
	argIdx := 3
	if storeID != nil {
		todayQuery += fmt.Sprintf(" AND store_id = $%d", argIdx)
		args = append(args, *storeID)
	}
	var todaysRevInt, todaysSalesInt int
	if err := r.db.QueryRow(ctx, todayQuery, args...).Scan(&todaysRevInt, &todaysSalesInt); err != nil {
		return nil, fmt.Errorf("failed to query today stats: %w", err)
	}
	stats.TodaysRevenue = int64(todaysRevInt)
	stats.TodaysSales = int64(todaysSalesInt)

	totalQuery := `
		SELECT COALESCE(SUM(total_amount), 0), COUNT(*)
		FROM sales
		WHERE status = 'completed'`
	totalArgs := []interface{}{}
	totalArgIdx := 1
	if storeID != nil {
		totalQuery += fmt.Sprintf(" AND store_id = $%d", totalArgIdx)
		totalArgs = append(totalArgs, *storeID)
	}
	var totalRevInt, totalSalesInt int
	if err := r.db.QueryRow(ctx, totalQuery, totalArgs...).Scan(&totalRevInt, &totalSalesInt); err != nil {
		return nil, fmt.Errorf("failed to query total stats: %w", err)
	}
	stats.TotalRevenue = int64(totalRevInt)
	stats.TotalSales = int64(totalSalesInt)

	productQuery := `SELECT COUNT(*) FROM products WHERE deleted_at IS NULL`
	productArgs := []interface{}{}
	productArgIdx := 1
	if storeID != nil {
		productQuery += fmt.Sprintf(" AND store_id = $%d", productArgIdx)
		productArgs = append(productArgs, *storeID)
	}
	var totalProducts int64
	if err := r.db.QueryRow(ctx, productQuery, productArgs...).Scan(&totalProducts); err != nil {
		return nil, fmt.Errorf("failed to query total products: %w", err)
	}
	stats.TotalProducts = totalProducts

	cfg := config.Load()
	stockQuery := `SELECT COUNT(*) FROM product_stock WHERE quantity <= $1`
	stockArgs := []interface{}{cfg.StockCriticalThreshold}
	if storeID != nil {
		stockQuery += fmt.Sprintf(" AND store_id = $%d", len(stockArgs)+1)
		stockArgs = append(stockArgs, *storeID)
	}
	var lowStockCount int64
	if err := r.db.QueryRow(ctx, stockQuery, stockArgs...).Scan(&lowStockCount); err != nil {
		return nil, fmt.Errorf("failed to query low stock count: %w", err)
	}
	stats.LowStockCount = lowStockCount

	customerArgs := []interface{}{}
	customerQuery := `SELECT COUNT(DISTINCT customer_id) FROM sales WHERE status = 'completed' AND customer_id IS NOT NULL`
	if storeID != nil {
		customerQuery += fmt.Sprintf(" AND store_id = $%d", len(customerArgs)+1)
		customerArgs = append(customerArgs, *storeID)
	}
	if err := r.db.QueryRow(ctx, customerQuery, customerArgs...).Scan(&stats.ActiveCustomers); err != nil {
		stats.ActiveCustomers = 0
	}

	return &stats, nil
}
