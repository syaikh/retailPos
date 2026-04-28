package domain

type User struct {
	ID          int      `json:"id"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	Password    string   `json:"-"`
	RoleID      int      `json:"role_id"`
	Role        Role     `json:"role"`
	StoreID     *int     `json:"store_id,omitempty"`
	IsActive    bool     `json:"is_active"`
	CreatedAt   string   `json:"created_at,omitempty"`
	UpdatedAt   string   `json:"updated_at,omitempty"`
}

type Role struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	IsSystem    bool     `json:"is_system"`
	CreatedAt   string   `json:"created_at,omitempty"`
}

type Permission struct {
	ID          int      `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
	CreatedAt   string   `json:"created_at,omitempty"`
}

type Product struct {
	ID          int      `json:"id"`
	SKU         string   `json:"sku"`
	Name        string   `json:"name"`
	Barcode     *string  `json:"barcode,omitempty"`
	CategoryID  *int     `json:"category_id,omitempty"`
	Price       int      `json:"price"`
	Cost        int      `json:"cost"`
	Stock       int      `json:"stock"`
	StockMin    int      `json:"stock_min"`
	StockMax    int      `json:"stock_max"`
	StoreID     *int     `json:"store_id,omitempty"`
	IsActive    bool     `json:"is_active"`
	CreatedAt   string   `json:"created_at,omitempty"`
	UpdatedAt   string   `json:"updated_at,omitempty"`
}

type Sale struct {
	ID          int      `json:"id"`
	InvoiceNumber string     `json:"invoice_number"`
	CashierID     int        `json:"cashier_id"`
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
	ID          int      `json:"id"`
	SaleID    int `json:"sale_id"`
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
	UnitPrice int `json:"unit_price"`
	Subtotal  int `json:"subtotal"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}

type UserWithPermissions struct {
	ID          int      `json:"id"`
	Username    string   `json:"username"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
	StoreID     *int     `json:"store_id,omitempty"`
}

type InventoryMovement struct {
	ID          int      `json:"id"`
	ProductID    int    `json:"product_id"`
	Quantity     int    `json:"quantity_change"`
	Type         string `json:"type"`
	ReferenceID  *int   `json:"reference_id,omitempty"`
	UserID       *int   `json:"user_id,omitempty"`
	Notes        string `json:"notes"`
}

type DashboardStats struct {
	TotalSales      int64 `json:"total_sales"`
	TotalRevenue    int64 `json:"total_revenue"`
	TotalProducts   int64 `json:"total_products"`
	LowStockCount   int64 `json:"low_stock_count"`
	TodaysSales     int64 `json:"todays_sales"`
	TodaysRevenue   int64 `json:"todays_revenue"`
	ActiveCustomers int64 `json:"active_customers"`
}

type AuditLog struct {
	ID          int      `json:"id"`
	UserID     *int   `json:"user_id,omitempty"`
	Username   string `json:"username"`
	Role       string `json:"role"`
	Action     string `json:"action"`
	EntityType string `json:"entity_type"`
	IPAddress   string  `json:"ip_address,omitempty"`
	EntityID   *int   `json:"entity_id,omitempty"`
	OldValues  any    `json:"old_values,omitempty"`
	NewValues  any    `json:"new_values,omitempty"`
	CreatedAt  string `json:"created_at"`
}

type Claims struct {
	ID          int      `json:"id"`
	Username    string   `json:"username"`
	RoleID      int      `json:"role_id"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
	StoreID     *int     `json:"store_id,omitempty"`
}

type SaleCreateRequest struct {
	InvoiceNumber string      `json:"invoice_number"`
	CashierID     int         `json:"cashier_id"`
	StoreID       *int        `json:"store_id,omitempty"`
	Subtotal      int         `json:"subtotal"`
	Discount      int         `json:"discount"`
	Tax           int         `json:"tax"`
	TotalAmount   int         `json:"total_amount"`
	PaymentMethod string      `json:"payment_method"`
	Items         []SaleItem  `json:"items"`
}
