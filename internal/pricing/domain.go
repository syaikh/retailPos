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
type PricingType string

const (
	PricingTypeDefault    PricingType = "default"     // fallback only — not creatable
	PricingTypeSpecialPrice PricingType = "special_price"
	PricingTypePromotion  PricingType = "promotion"
)

// PricingMethod defines how the pricing value is applied to the base price.
type PricingMethod string

const (
	PricingMethodFixedPrice  PricingMethod = "fixed_price"
	PricingMethodDiscountPct PricingMethod = "discount_percent"
	PricingMethodDiscountAmt PricingMethod = "discount_amount"
	PricingMethodMarkupPct   PricingMethod = "markup_percent"
)

// PricingRule is a business rule entity — it defines eligibility, priority,
// validity, and the pricing method/value to apply.
type PricingRule struct {
	ID              int           `json:"id"`
	ProductID       *int          `json:"product_id,omitempty"`
	CategoryID      *int          `json:"category_id,omitempty"`
	BrandID         *int          `json:"brand_id,omitempty"`
	PricingType     PricingType   `json:"pricing_type"`
	PricingMethod   PricingMethod `json:"pricing_method"`
	PricingValue    float64       `json:"pricing_value"`
	Name            string        `json:"name"`
	MinimumQuantity int           `json:"minimum_quantity"`
	MaximumQuantity *int          `json:"maximum_quantity,omitempty"`
	Priority        int           `json:"priority"`
	CustomerGroupID *int          `json:"customer_group_id,omitempty"`
	StoreID         *int          `json:"store_id,omitempty"`
	RecurrenceDays  []string      `json:"recurrence_days,omitempty"`
	TimeFrom        *string       `json:"time_from,omitempty"`
	TimeTo          *string       `json:"time_to,omitempty"`
	AllowCombine    bool          `json:"allow_combine"`
	IsActive        bool          `json:"is_active"`
	EffectiveFrom   *time.Time    `json:"effective_from,omitempty"`
	EffectiveUntil  *time.Time    `json:"effective_until,omitempty"`
	CreatedAt       string        `json:"created_at,omitempty"`
	UpdatedAt       string        `json:"updated_at,omitempty"`

	// Scope fields populated on read (not stored in pricing_rules)
	ScopeType string `json:"scope_type,omitempty"` // "product", "category", "brand"
}

// ResolvedPrice is the output of the pricing resolution algorithm.
type ResolvedPrice struct {
	UnitPrice     int           `json:"unit_price"`
	OriginalPrice int           `json:"original_price"`
	Discount      int           `json:"discount"`
	PricingType   PricingType   `json:"pricing_type"`
	PricingMethod PricingMethod `json:"pricing_method"`
	Rule          *PricingRule  `json:"rule,omitempty"`
}

// PriceResolver is the public interface for the pricing subsystem.
type PriceResolver interface {
	Resolve(ctx context.Context, rc ResolveContext) (*ResolvedPrice, error)
	ResolveBatch(ctx context.Context, items []ResolveItem) ([]ResolvedPrice, error)
}

// ResolveContext carries the full context for price resolution.
type ResolveContext struct {
	ProductID       int
	Quantity        int
	CustomerGroupID *int
	StoreID         *int
}

// ResolveItem is the input for batch resolution.
type ResolveItem struct {
	ProductID       int  `json:"product_id"`
	Quantity        int  `json:"quantity"`
	CustomerGroupID *int `json:"customer_group_id,omitempty"`
	StoreID         *int `json:"store_id,omitempty"`
}

// ValidPricingMethods returns all valid pricing methods.
func ValidPricingMethods() []PricingMethod {
	return []PricingMethod{
		PricingMethodFixedPrice,
		PricingMethodDiscountPct,
		PricingMethodDiscountAmt,
		PricingMethodMarkupPct,
	}
}

// ValidPricingTypes returns all valid pricing types.
func ValidPricingTypes() []PricingType {
	return []PricingType{
		PricingTypeSpecialPrice,
		PricingTypePromotion,
	}
}
