package report

type DashboardStats struct {
	TotalSales      int64 `json:"total_sales"`
	TotalRevenue    int64 `json:"total_revenue"`
	TotalProducts   int64 `json:"total_products"`
	LowStockCount   int64 `json:"low_stock_count"`
	TodaysSales     int64 `json:"todays_sales"`
	TodaysRevenue   int64 `json:"todays_revenue"`
	ActiveCustomers int64 `json:"active_customers"`
}

type ChartDataPoint struct {
	Date  string `json:"date"`
	Total int    `json:"total"`
}

type PeriodComparison struct {
	CurrentRevenue        int `json:"current_revenue"`
	PreviousRevenue       int `json:"previous_revenue"`
	CurrentOrders         int `json:"current_orders"`
	PreviousOrders        int `json:"previous_orders"`
	CurrentAOV            int `json:"current_aov"`
	PreviousAOV           int `json:"previous_aov"`
	RevenuePerDay         int `json:"revenue_per_day"`
	PreviousRevenuePerDay int `json:"previous_revenue_per_day"`
	PeakRevenueHour       int `json:"peak_revenue_hour"`
	PreviousPeakRevenue   int `json:"previous_peak_revenue"`
	PeakRevenueMonth      int `json:"peak_revenue_month"`
	PreviousPeakRevenueMonth int `json:"previous_peak_revenue_month"`
}

type WeeklyReportItem struct {
	WeekStart  string `json:"week_start"`
	WeekEnd    string `json:"week_end"`
	Total      int    `json:"total"`
	OrderCount int    `json:"order_count"`
}

type MonthlyReportItem struct {
	Month      string `json:"month"`
	MonthStart string `json:"month_start"`
	Total      int    `json:"total"`
	OrderCount int    `json:"order_count"`
}
