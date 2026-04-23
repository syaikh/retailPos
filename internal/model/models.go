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
	StoreID      *int      `json:"store_id,omitempty" db:"store_id"`
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
	ID         int        `json:"id" db:"id"`
	Name       string     `json:"name" db:"name"`
	SKU        string     `json:"sku" db:"sku"`
	Barcode    *string    `json:"barcode" db:"barcode"`
	Price      int        `json:"price" db:"price"`
	Stock      int        `json:"stock" db:"stock"`
	GroupID    *int       `json:"group_id" db:"group_id"`
	StoreID    *int       `json:"store_id,omitempty" db:"store_id"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
	RestoredAt *time.Time `json:"restored_at,omitempty" db:"restored_at"`
}

type Sale struct {
	ID            int        `json:"id" db:"id"`
	TotalAmount   int        `json:"total_amount" db:"total_amount"`
	PaymentMethod string     `json:"payment_method" db:"payment_method"`
	CashierID       int        `json:"cashier_id" db:"cashier_id"`
	StoreID         *int       `json:"store_id,omitempty" db:"store_id"`
	TransactionCode string     `json:"transaction_code" db:"-"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
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
