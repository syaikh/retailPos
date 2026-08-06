package sale

import (
	"context"
	"time"
)

// PriceResolver is the consumer-side port for the pricing subsystem. Only the
// snapshot batch operation used by cart flows is exposed; the rest of the
// pricing surface stays internal to internal/pricing.
type PriceResolver interface {
	ResolveSnapshotsBatch(ctx context.Context, items []ResolveItem) ([]PriceSnapshot, error)
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
