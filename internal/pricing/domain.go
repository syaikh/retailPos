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
	ErrRuleConflict      = errors.New("conflicting pricing rule")
	ErrDuplicateName     = errors.New("nama rule sudah digunakan")
)

// PricingType is a classification label — it describes what kind of pricing applies.
type PricingType string

const (
	PricingTypeDefault      PricingType = "default" // fallback only — not creatable
	PricingTypeSpecialPrice PricingType = "special_price"
	PricingTypePromotion    PricingType = "promotion"
)

// PricingMethod defines how the pricing value is applied to the base price.
type PricingMethod string

const (
	PricingMethodFixedPrice  PricingMethod = "fixed_price"
	PricingMethodDiscountPct PricingMethod = "discount_percent"
	PricingMethodDiscountAmt PricingMethod = "discount_amount"
	PricingMethodMarkupPct   PricingMethod = "markup_percent"
)

// PricingRuleStatus tracks the approval workflow state.
type PricingRuleStatus string

const (
	StatusDraft    PricingRuleStatus = "draft"
	StatusPending  PricingRuleStatus = "pending"
	StatusApproved PricingRuleStatus = "approved"
	StatusRejected PricingRuleStatus = "rejected"
)

// PricingRule is a business rule entity — it defines eligibility, priority,
// validity, and the pricing method/value to apply.
type PricingRule struct {
	ID              int               `json:"id"                         db:"id"`
	ProductID       *int              `json:"product_id,omitempty"       db:"product_id"`
	CategoryID      *int              `json:"category_id,omitempty"      db:"category_id"`
	BrandID         *int              `json:"brand_id,omitempty"         db:"brand_id"`
	PricingType     PricingType       `json:"pricing_type"               db:"pricing_type"     validate:"required,oneof=special_price promotion"`
	PricingMethod   PricingMethod     `json:"pricing_method"             db:"pricing_method"   validate:"required,oneof=fixed_price discount_percent discount_amount markup_percent"`
	PricingValue    float64           `json:"pricing_value"              db:"pricing_value"    validate:"gte=0"`
	Name            string            `json:"name"                       db:"name"             validate:"required"`
	MinimumQuantity int               `json:"minimum_quantity"           db:"minimum_quantity" validate:"gte=1"`
	MaximumQuantity *int              `json:"maximum_quantity,omitempty" db:"maximum_quantity"`
	Priority        int               `json:"priority"                   db:"priority"`
	CustomerGroupID *int              `json:"customer_group_id,omitempty" db:"customer_group_id"`
	StoreID         *int              `json:"store_id,omitempty"         db:"store_id"`
	RecurrenceDays  []string          `json:"recurrence_days,omitempty"  db:"recurrence_days"`
	TimeFrom        *string           `json:"time_from,omitempty"        db:"time_from"`
	TimeTo          *string           `json:"time_to,omitempty"          db:"time_to"`
	AllowCombine    bool              `json:"allow_combine"              db:"allow_combine"`
	IsActive        bool              `json:"is_active"                  db:"is_active"`
	Status          PricingRuleStatus `json:"status"                     db:"status"`
	EffectiveFrom   *time.Time        `json:"effective_from,omitempty"   db:"effective_from"`
	EffectiveUntil  *time.Time        `json:"effective_until,omitempty"  db:"effective_until"`
	CreatedAt       string            `json:"created_at,omitempty"       db:"created_at"`
	UpdatedAt       string            `json:"updated_at,omitempty"       db:"updated_at"`

	// Scope fields populated on read (not stored in pricing_rules)
	ScopeType string `json:"scope_type,omitempty" db:"scope_type"` // "product", "category", "brand"
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

// PriceSnapshot is the immutable result of a price resolution for a single item.
// It carries cost & tax information captured at snapshot time.
type PriceSnapshot struct {
	ProductID     int
	ProductName   string
	UnitPrice     int
	OriginalPrice int
	Discount      int
	PricingType   PricingType
	PricingMethod PricingMethod
	Rule          *PricingRule
	Cost          int
	TaxClassID    *int
	TaxRate       *float64
	SnapshotAt    time.Time
}

// ProductCostTax holds the cost and tax-class information of a product at snapshot time.
type ProductCostTax struct {
	Cost        int
	TaxClassID  *int
	TaxRate     *float64
	ProductName string
}

// PriceResolver is the public interface for the pricing subsystem.
type PriceResolver interface {
	Resolve(ctx context.Context, rc ResolveContext) (*ResolvedPrice, error)
	ResolveBatch(ctx context.Context, items []ResolveItem) ([]ResolvedPrice, error)
	ResolveSnapshot(ctx context.Context, rc ResolveContext) (*PriceSnapshot, error)
	ResolveSnapshotsBatch(ctx context.Context, items []ResolveItem) ([]PriceSnapshot, error)
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
