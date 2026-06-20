package domain

// PeriodComparison holds comparison data between two periods
type PeriodComparison struct {
	CurrentRevenue        int `json:"current_revenue"`
	PreviousRevenue       int `json:"previous_revenue"`
	CurrentOrders         int `json:"current_orders"`
	PreviousOrders        int `json:"previous_orders"`
	CurrentAOV            int `json:"current_aov"`  // Average Order Value
	PreviousAOV           int `json:"previous_aov"`
	RevenuePerDay         int `json:"revenue_per_day"`
	PreviousRevenuePerDay int `json:"previous_revenue_per_day"`
	PeakRevenueHour          int `json:"peak_revenue_hour"`           // Peak hour revenue for current period
	PreviousPeakRevenue      int `json:"previous_peak_revenue"`       // Peak hour revenue for previous period
	PeakRevenueMonth         int `json:"peak_revenue_month"`          // Peak month revenue for current period
	PreviousPeakRevenueMonth int `json:"previous_peak_revenue_month"` // Peak month revenue for previous period
}