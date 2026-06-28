package sale

type Sale struct {
	ID            int        `json:"id"`
	InvoiceNumber string     `json:"invoice_number"`
	CashierID     int        `json:"cashier_id"`
	CustomerID    *int       `json:"customer_id,omitempty"`
	CustomerName  string     `json:"customer_name,omitempty"`
	StoreID       *int       `json:"store_id,omitempty"`
	Subtotal      int        `json:"subtotal"`
	Discount      int        `json:"discount"`
	Tax           int        `json:"tax"`
	TotalAmount   int        `json:"total_amount"`
	PaymentMethod string     `json:"payment_method"`
	Status        string     `json:"status"`
	Items         []SaleItem `json:"items,omitempty"`
	CreatedAt     string     `json:"created_at,omitempty"`
	UpdatedAt     string     `json:"updated_at,omitempty"`
}

type SaleItem struct {
	ID        int    `json:"id"`
	SaleID    int    `json:"sale_id"`
	ProductID int    `json:"product_id"`
	Name      string `json:"name"`
	Quantity  int    `json:"quantity"`
	UnitPrice int    `json:"unit_price"`
	Subtotal  int    `json:"subtotal"`
	DPPAmount int    `json:"dpp_amount"`
	TaxAmount int    `json:"tax_amount"`
}

type SaleExportRow struct {
	InvoiceNumber string `json:"invoice_number"`
	CreatedAt     string `json:"created_at"`
	CustomerName  string `json:"customer_name"`
	ItemCount     int    `json:"items_count"`
	PaymentMethod string `json:"payment_method"`
	TotalAmount   int    `json:"total_amount"`
}

type SaleCreateRequest struct {
	InvoiceNumber string     `json:"invoice_number"`
	CashierID     int        `json:"cashier_id"`
	StoreID       *int       `json:"store_id,omitempty"`
	Subtotal      int        `json:"subtotal"`
	Discount      int        `json:"discount"`
	Tax           int        `json:"tax"`
	TotalAmount   int        `json:"total_amount"`
	PaymentMethod string     `json:"payment_method"`
	CustomerID    *int       `json:"customer_id,omitempty"`
	Items         []SaleItem `json:"items"`
}

type PaymentMethod struct {
	ID                int    `json:"id"`
	Code              string `json:"code"`
	Name              string `json:"name"`
	IsActive          bool   `json:"is_active"`
	RequiresReference bool   `json:"requires_reference"`
	SortOrder         int    `json:"sort_order"`
	CreatedAt         string `json:"created_at,omitempty"`
}
