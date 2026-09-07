package consignment

import (
	"context"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

// StockAdjuster is the consumer-side port for product_stock writes,
// implemented by internal/inventory (structural typing — no import of
// internal/inventory needed). internal/inventory is the single-writer of
// product_stock and the inventory_movements ledger, so consignment receipts,
// pending returns, and returns write through this port inside the caller's
// Unit of Work.
type StockAdjuster interface {
	ApplyConsignmentDelta(ctx context.Context, tx pgx.Tx, delta shared.ConsignmentStockDelta) error
}

// StockReader is the consumer-side port for product_stock reads, implemented
// by internal/inventory. It answers ownership questions about the global
// product_stock table without a direct cross-context query.
type StockReader interface {
	GetStoreOwnedQuantity(ctx context.Context, productID int) (int, error)
}

// SupplierStore is the supplier-side read port, implemented by
// internal/supplier. It answers ownership questions about the suppliers table
// (is_consignment flag) and supplier display names.
type SupplierStore interface {
	IsConsignmentSupplier(ctx context.Context, db shared.DBPool, supplierID int) (bool, error)
	ListConsignmentSuppliers(ctx context.Context, db shared.DBPool) ([]shared.SupplierRef, error)
	SupplierNamesByIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]string, error)
}

// StoreNameProvider is the store-side read port, implemented by
// internal/store. It resolves store display names for consignment documents.
type StoreNameProvider interface {
	StoreNamesByIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]string, error)
}

// ProductMetaProvider is the product-side read port, implemented by
// internal/product. It resolves product sku/name for consignment documents.
type ProductMetaProvider interface {
	ProductMetasByIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]shared.ProductMeta, error)
}

// UsernameProvider is the user-side read port, implemented by internal/user.
// It resolves the usernames of staff who create consignment documents.
type UsernameProvider interface {
	UsernamesByIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]string, error)
}

// PaymentMethodProvider is the payment-method read port, implemented by
// internal/sale via a wiring adapter (structural typing with a local mirror of
// sale.PaymentMethod). Payouts reuse the existing payment_methods table but are
// decoupled from sale payments.
type PaymentMethodProvider interface {
	ActivePaymentMethods(ctx context.Context) ([]PaymentMethod, error)
	PaymentMethodByID(ctx context.Context, id int) (*PaymentMethod, error)
	PaymentMethodsByIDs(ctx context.Context, ids []int) (map[int]PaymentMethod, error)
}

// PaymentMethod mirrors the shape of sale.PaymentMethod so the payout flow can
// display the chosen method without importing internal/sale.
type PaymentMethod struct {
	ID   int    `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}
