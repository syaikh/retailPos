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

// PricingType is a classification label for the applied pricing.
type PricingType string

// ResolveItem is the minimal input for resolving a price snapshot.
type ResolveItem struct {
	ProductID       int
	Quantity        int
	CustomerGroupID *int
	StoreID         *int
}

// PricingRule is the subset of a pricing rule carried by a cart snapshot.
type PricingRule struct {
	ID          int
	Name        string
	PricingType PricingType
}

// PriceSnapshot is the immutable result of a price resolution for a single item.
// It carries cost & tax information captured at snapshot time.
type PriceSnapshot struct {
	ProductID     int
	ProductName   string
	UnitPrice     int
	OriginalPrice int
	Discount      int
	PricingType   PricingType
	Rule          *PricingRule
	Cost          int
	TaxClassID    *int
	TaxRate       *float64
	SnapshotAt    time.Time
}
