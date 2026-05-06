package domain

// PeriodComparison holds comparison data between two periods
type PeriodComparison struct {
	CurrentRevenue       int `json:"current_revenue"`
	PreviousRevenue      int `json:"previous_revenue"`
	CurrentOrders        int `json:"current_orders"`
	PreviousOrders       int `json:"previous_orders"`
	CurrentAOV           int `json:"current_aov"`  // Average Order Value
	PreviousAOV          int `json:"previous_aov"`
	RevenuePerDay        int `json:"revenue_per_day"`
	PreviousRevenuePerDay int `json:"previous_revenue_per_day"`
}