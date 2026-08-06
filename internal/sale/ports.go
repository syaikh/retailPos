package sale

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

// PriceResolver is the consumer-side port for the pricing subsystem. Only the
// snapshot batch operation used by cart flows is exposed; the rest of the
// pricing surface stays internal to internal/pricing.
type PriceResolver interface {
	ResolveSnapshotsBatch(ctx context.Context, items []ResolveItem) ([]PriceSnapshot, error)
}

// StockDeducer is the consumer-side port for the inventory subsystem's stock
// deduction. Checkout is a Unit of Work (ADR_Cross_Module_Transaction_Strategy
// §2.2: Reserve Stock → Create Sale → Create Payment in one transaction), so the
// implementation MUST run against the caller's tx to preserve atomicity — a
// sale must never commit while its stock deduction fails. internal/inventory is
// the canonical single-writer of product_stock and provides the production
// implementation; the composition root MUST wire it via SetStockDeducer before
// any checkout path runs — an unwired service fails fast at runtime.
type StockDeducer interface {
	DeductStock(ctx context.Context, tx pgx.Tx, items []shared.StockDeductItem) error
}

// ShiftTotalUpdater is the consumer-side port for the shift subsystem's shift
// totals. A completed sale contributes to its shift's running totals inside the
// same Unit of Work as the sale itself (ADR_Cross_Module_Transaction_Strategy
// §2.2), so the implementation MUST run against the caller's tx to preserve
// atomicity — a sale must never commit while its shift contribution fails.
// internal/shift is the canonical single-writer of the shifts running totals and
// provides the production implementation; the composition root MUST wire it via
// SetShiftTotalUpdater before any sale with a shift_id completes — an unwired
// service fails fast at runtime.
type ShiftTotalUpdater interface {
	UpdateShiftTotals(ctx context.Context, tx pgx.Tx, contribution shared.ShiftSaleContribution) error
}

// Type is a classification label for the applied pricing.
type Type string

// ResolveItem is the minimal input for resolving a price snapshot.
type ResolveItem struct {
	ProductID       int
	Quantity        int
	CustomerGroupID *int
	StoreID         *int
}

// Rule is the subset of a pricing rule carried by a cart snapshot.
type Rule struct {
	ID          int
	Name        string
	Type Type
}

// PriceSnapshot is the immutable result of a price resolution for a single item.
// It carries cost & tax information captured at snapshot time.
type PriceSnapshot struct {
	ProductID     int
	ProductName   string
	UnitPrice     int
	OriginalPrice int
	Discount      int
	Type   Type
	Rule          *Rule
	Cost          int
	TaxClassID    *int
	TaxRate       *float64
	SnapshotAt    time.Time
}
