package pricing

import (
	"context"
	"errors"
	"time"
)

var (
	ErrRuleNotFound      = errors.New("pricing rule not found")
	ErrProductNotFound   = errors.New("product not found")
	ErrInvalidRule       = errors.New("invalid pricing rule")
	ErrDuplicateRuleType = errors.New("duplicate pricing type for this product")
)

// PricingType is a classification label — it describes what kind of pricing applies.
// It carries no behavior. See ADR-001 §Domain Model.
type PricingType string

const (
	PricingTypeNormal    PricingType = "normal"
	PricingTypeDiscount  PricingType = "discount"
	PricingTypeWholesale PricingType = "wholesale"
	PricingTypeMember    PricingType = "member"    // future
	PricingTypePromotion PricingType = "promotion" // future
)

// PricingRule is a business rule entity — it defines eligibility, priority,
// validity, and the price to charge. See ADR-006.
type PricingRule struct {
	ID              int         `json:"id"`
	ProductID       int         `json:"product_id"`
	PricingType     PricingType `json:"pricing_type"`
	Name            string      `json:"name"`
	Price           int         `json:"price"`
	MinimumQuantity int         `json:"minimum_quantity"`
	Priority        int         `json:"priority"`
	IsActive        bool        `json:"is_active"`
	EffectiveFrom   *time.Time  `json:"effective_from,omitempty"`
	EffectiveUntil  *time.Time  `json:"effective_until,omitempty"`
	CreatedAt       string      `json:"created_at,omitempty"`
	UpdatedAt       string      `json:"updated_at,omitempty"`
}

// ResolvedPrice is the output of the pricing resolution algorithm.
// See ADR-005 §Output.
type ResolvedPrice struct {
	UnitPrice     int          `json:"unit_price"`
	OriginalPrice int          `json:"original_price"`
	Discount      int          `json:"discount"`
	PricingType   PricingType  `json:"pricing_type"`
	Rule          *PricingRule `json:"rule,omitempty"`
}

// PriceResolver is the public interface for the pricing subsystem.
// Consumers (POS, Sales, Reports) depend only on this interface.
// See INV-P8.
type PriceResolver interface {
	// Resolve returns the effective selling price for a product at a given quantity.
	Resolve(ctx context.Context, productID int, quantity int) (*ResolvedPrice, error)

	// ResolveBatch returns effective selling prices for multiple products in a single operation.
	// This avoids N+1 queries for POS cart checkout. See INV-P11.
	ResolveBatch(ctx context.Context, items []ResolveItem) ([]ResolvedPrice, error)
}

// ResolveItem is the input for batch resolution.
type ResolveItem struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}
