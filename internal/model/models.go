package models

import (
	"time"
)

type User struct {
	ID           int       `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	PasswordHash string    `json:"-" db:"password_hash"`
	RoleID       int       `json:"role_id" db:"role_id"`
	Role         string    `json:"role" db:"role"` // legacy + populated via join
	Permissions  []string  `json:"permissions,omitempty" db:"-"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type ProductGroup struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	ProductCount int       `json:"product_count"`
	CreatedAt    time.Time `json:"created_at"`
}

type Product struct {
	ID         int        `json:"id"`
	Name       string     `json:"name"`
	SKU        string     `json:"sku"`
	Barcode    *string    `json:"barcode"` // Using *string to handle NULL values
	Price      int        `json:"price"`
	Stock      int        `json:"stock"`
	GroupID    *int       `json:"group_id"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
	RestoredAt *time.Time `json:"restored_at,omitempty"`
}

type Sale struct {
	ID            int        `json:"id"`
	TotalAmount   int        `json:"total_amount"`
	PaymentMethod string     `json:"payment_method"`
	CashierID     int        `json:"cashier_id"`
	CreatedAt     time.Time  `json:"created_at"`
	Items         []SaleItem `json:"items,omitempty"`
}

type SaleItem struct {
	ID          int    `json:"id"`
	SaleID      int    `json:"sale_id"`
	ProductID   int    `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
	PriceAtSale int    `json:"price_at_sale"`
}
