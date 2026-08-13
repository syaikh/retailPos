package sale

import (
	"context"
	"fmt"
	"time"

	"retail-pos-system/internal/shared"
)

type reportAdapter struct{}

func NewReportAdapter() *reportAdapter {
	return &reportAdapter{}
}

func (reportAdapter) GetCompletedSalesStats(ctx context.Context, db shared.DBPool, start, end time.Time, storeID *int) (revenue int, orders int, err error) {
	query := `SELECT COALESCE(SUM(total_amount), 0), COUNT(*) FROM sales WHERE status = 'completed' AND created_at >= $1 AND created_at < $2`
	args := []interface{}{start, end}
	if storeID != nil {
		query += ` AND store_id = $3`
		args = append(args, *storeID)
	}
	err = db.QueryRow(ctx, query, args...).Scan(&revenue, &orders)
	return
}

func (reportAdapter) GetAllCompletedSalesStats(ctx context.Context, db shared.DBPool, storeID *int) (revenue int, orders int, err error) {
	// Read from mv_dashboard_totals (refreshed by the coordinator at each
	// Jakarta hour boundary) instead of scanning the raw sales table, keeping
	// the all-time dashboard total as cheap as the charts. The view holds the
	// same completed-sale rows grouped per store, so the global/store filters
	// produce identical results to the former raw query.
	query := `SELECT COALESCE(SUM(total_revenue), 0), COALESCE(SUM(transaction_count), 0) FROM mv_dashboard_totals`
	args := []interface{}{}
	if storeID != nil {
		query += ` WHERE store_id = $1`
		args = append(args, *storeID)
	}
	err = db.QueryRow(ctx, query, args...).Scan(&revenue, &orders)
	return
}

func (reportAdapter) GetActiveCustomerCount(ctx context.Context, db shared.DBPool, storeID *int) (count int64, err error) {
	query := `SELECT COUNT(DISTINCT customer_id) FROM sales WHERE status = 'completed' AND customer_id IS NOT NULL`
	args := []interface{}{}
	if storeID != nil {
		query += ` AND store_id = $1`
		args = append(args, *storeID)
	}
	err = db.QueryRow(ctx, query, args...).Scan(&count)
	return
}

func (reportAdapter) GetWeeklySales(ctx context.Context, db shared.DBPool, start, end time.Time, storeID *int) ([]shared.WeeklyReportItem, error) {
	query := shared.WeeklySalesQueryTemplate
	args := []interface{}{start, end}
	if storeID != nil {
		query += ` AND store_id = $3`
		args = append(args, *storeID)
	}
	query += ` GROUP BY week_start, week_end ORDER BY week_start`

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query weekly sales: %w", err)
	}
	defer rows.Close()

	var result []shared.WeeklyReportItem
	for rows.Next() {
		var item shared.WeeklyReportItem
		if err := rows.Scan(&item.WeekStart, &item.WeekEnd, &item.Total, &item.OrderCount); err != nil {
			return nil, fmt.Errorf("scan weekly sales row: %w", err)
		}
		result = append(result, item)
	}
	if result == nil {
		result = []shared.WeeklyReportItem{}
	}
	return result, rows.Err()
}

func (reportAdapter) GetMonthlySales(ctx context.Context, db shared.DBPool, start, end time.Time, storeID *int) ([]shared.MonthlyReportItem, error) {
	query := shared.MonthlySalesQueryTemplate
	args := []interface{}{start, end}
	if storeID != nil {
		query += ` AND store_id = $3`
		args = append(args, *storeID)
	}
	query += ` GROUP BY month, month_start ORDER BY month`

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query monthly sales: %w", err)
	}
	defer rows.Close()

	var result []shared.MonthlyReportItem
	for rows.Next() {
		var item shared.MonthlyReportItem
		if err := rows.Scan(&item.Month, &item.MonthStart, &item.Total, &item.OrderCount); err != nil {
			return nil, fmt.Errorf("scan monthly sales row: %w", err)
		}
		result = append(result, item)
	}
	if result == nil {
		result = []shared.MonthlyReportItem{}
	}
	return result, rows.Err()
}

func (reportAdapter) GetPricingBreakdown(ctx context.Context, db shared.DBPool, start, end time.Time, storeID *int) ([]shared.PricingBreakdownItem, error) {
	query := shared.PricingBreakdownQueryTemplate
	args := []interface{}{start, end}
	if storeID != nil {
		query += ` AND s.store_id = $3`
		args = append(args, *storeID)
	}
	query += ` GROUP BY COALESCE(si.pricing_type, 'normal') ORDER BY revenue DESC`

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query pricing breakdown: %w", err)
	}
	defer rows.Close()

	var items []shared.PricingBreakdownItem
	for rows.Next() {
		var item shared.PricingBreakdownItem
		if err := rows.Scan(&item.Type, &item.Revenue, &item.OrderCount, &item.ItemCount); err != nil {
			return nil, fmt.Errorf("scan pricing breakdown: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
