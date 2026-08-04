package shift

type Shift struct {
	ID               int     `json:"id"`
	UserID           int     `json:"user_id"`
	Username         string  `json:"username,omitempty"`
	StoreID          *int    `json:"store_id,omitempty"`
	StoreName        string  `json:"store_name,omitempty"`
	Status           string  `json:"status"`
	OpeningBalance   int     `json:"opening_balance"`
	ClosingBalance   *int    `json:"closing_balance,omitempty"`
	CashSales        int     `json:"cash_sales"`
	NonCashSales     int     `json:"non_cash_sales"`
	TotalSales       int     `json:"total_sales"`
	TransactionCount int     `json:"transaction_count"`
	Discrepancy      *int    `json:"discrepancy,omitempty"`
	Notes            *string `json:"notes,omitempty"`
	NeedsReview      bool    `json:"needs_review"`
	ReviewedBy       *int    `json:"reviewed_by,omitempty"`
	ReviewedAt       string  `json:"reviewed_at,omitempty"`
	OpenedAt         string  `json:"opened_at"`
	ClosedAt         string  `json:"closed_at,omitempty"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

type ShiftSummary struct {
	TotalCashSales    int `json:"total_cash_sales"`
	TotalNonCashSales int `json:"total_non_cash_sales"`
	TotalSales        int `json:"total_sales"`
	TotalTransactions int `json:"total_transactions"`
}
