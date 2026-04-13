package repo

import (
	"database/sql"
	model "retailPos/internal/model"
	"time"
)

type StatsRepo struct {
	db *sql.DB
}

func NewStatsRepo(db *sql.DB) *StatsRepo {
	return &StatsRepo{db: db}
}

type Activity struct {
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	User      string    `json:"user"`
	CreatedAt time.Time `json:"created_at"`
}

type DashboardStats struct {
	TodaySales        int             `json:"today_sales"`
	TodaySalesTrend   float64         `json:"today_sales_trend"`
	MonthSales        int             `json:"month_sales"`
	MonthSalesTrend   float64         `json:"month_sales_trend"`
	TodayTransactions int             `json:"today_transactions"`
	LowStockCount     int             `json:"low_stock_count"`
	LowStockProducts  []model.Product `json:"low_stock_products"`
	RecentActivities  []Activity      `json:"recent_activities"`
}

func (r *StatsRepo) GetDashboardStats() (*DashboardStats, error) {
	stats := &DashboardStats{
		LowStockProducts: []model.Product{},
		RecentActivities: []Activity{},
	}

	// 1. Sales & Transactions Today
	err := r.db.QueryRow(`SELECT COALESCE(SUM(total_amount), 0), COUNT(*) FROM sales WHERE created_at >= CURRENT_DATE`).Scan(&stats.TodaySales, &stats.TodayTransactions)
	if err != nil {
		return nil, err
	}

	// 2. Sales Yesterday (for Trend)
	var yesterdaySales int
	err = r.db.QueryRow(`SELECT COALESCE(SUM(total_amount), 0) FROM sales WHERE created_at >= CURRENT_DATE - INTERVAL '1 day' AND created_at < CURRENT_DATE`).Scan(&yesterdaySales)
	if err != nil {
		return nil, err
	}
	if yesterdaySales > 0 {
		stats.TodaySalesTrend = float64(stats.TodaySales-yesterdaySales) / float64(yesterdaySales) * 100
	} else if stats.TodaySales > 0 {
		stats.TodaySalesTrend = 100
	}

	// 3. Sales This Month
	err = r.db.QueryRow(`SELECT COALESCE(SUM(total_amount), 0) FROM sales WHERE created_at >= date_trunc('month', CURRENT_DATE)`).Scan(&stats.MonthSales)
	if err != nil {
		return nil, err
	}

	// 4. Sales Last Month (for Trend)
	var lastMonthSales int
	err = r.db.QueryRow(`SELECT COALESCE(SUM(total_amount), 0) FROM sales WHERE created_at >= date_trunc('month', CURRENT_DATE - INTERVAL '1 month') AND created_at < date_trunc('month', CURRENT_DATE)`).Scan(&lastMonthSales)
	if err != nil {
		return nil, err
	}
	if lastMonthSales > 0 {
		stats.MonthSalesTrend = float64(stats.MonthSales-lastMonthSales) / float64(lastMonthSales) * 100
	} else if stats.MonthSales > 0 {
		stats.MonthSalesTrend = 100
	}

	// 5. Low Stock
	err = r.db.QueryRow(`SELECT COUNT(*) FROM products WHERE stock < 10 AND deleted_at IS NULL`).Scan(&stats.LowStockCount)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(`SELECT id, name, sku, barcode, price, stock, group_id, created_at, updated_at FROM products WHERE stock < 10 AND deleted_at IS NULL ORDER BY stock ASC LIMIT 5`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p model.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.SKU, &p.Barcode, &p.Price, &p.Stock, &p.GroupID, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		stats.LowStockProducts = append(stats.LowStockProducts, p)
	}

	// 6. Recent Activities (Sales)
	rows, err = r.db.Query(`SELECT 'sale' as type, 'Memproses transaksi #TRX-' || s.id as message, u.username, s.created_at 
	                         FROM sales s JOIN users u ON s.cashier_id = u.id ORDER BY s.created_at DESC LIMIT 5`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var a Activity
		if err := rows.Scan(&a.Type, &a.Message, &a.User, &a.CreatedAt); err != nil {
			return nil, err
		}
		stats.RecentActivities = append(stats.RecentActivities, a)
	}

	return stats, nil
}
