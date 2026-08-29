package pricing

import (
	"context"

	"retail-pos-system/internal/shared"
)

// ProductPricingProvider is the product-owned read side of the pricing module's
// price-resolution inputs. products, tax_classes are owned by the katalog
// bounded context (internal/product); pricing no longer queries those tables
// directly and instead routes base-price, scope, cost/tax, and autocomplete
// reads through this port. internal/product provides the production
// implementation (product.PricingLookup); the composition root MUST wire
// it via SetProductPricingProvider before any price resolution or product
// search — an unwired repository fails fast at runtime.
type ProductPricingProvider interface {
	// BasePricesByIDs returns product base prices keyed by product ID. IDs with
	// no matching non-deleted product are absent from the map.
	BasePricesByIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]int, error)
	// ProductScopesByIDs returns the category/brand scope keyed by product ID.
	ProductScopesByIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]shared.ProductScope, error)
	// ProductCostTaxesByIDs returns cost/tax/name rows keyed by product ID.
	ProductCostTaxesByIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]shared.ProductCostTax, error)
	// ProductIDsByName returns the IDs of non-deleted products whose name
	// ILIKE-matches the given search pattern (caller supplies the '%' pattern).
	// Used to resolve the pricing rule listing search filter.
	ProductIDsByName(ctx context.Context, db shared.DBPool, search string) ([]int, error)
	// SearchPricingProducts returns product autocomplete rows matching
	// name/sku/barcode (active, non-deleted products only).
	SearchPricingProducts(ctx context.Context, db shared.DBPool, query string, limit int) ([]shared.ProductSearchResult, error)
}

// CategoryNameSearchProvider resolves category-name search matches for the
// pricing rule listing filter. categories is owned by the katalog bounded
// context (internal/category); pricing routes the category-name EXISTS clause
// through this port instead of querying categories directly.
type CategoryNameSearchProvider interface {
	// CategoryIDsByName returns the IDs of categories whose name ILIKE-matches
	// the given search pattern (caller supplies the '%' pattern).
	CategoryIDsByName(ctx context.Context, db shared.DBPool, search string) ([]int, error)
}

// BrandNameSearchProvider resolves brand-name search matches for the pricing
// rule listing filter. brands is owned by the katalog bounded context
// (internal/brand); pricing routes the brand-name EXISTS clause through this
// port instead of querying brands directly.
type BrandNameSearchProvider interface {
	// BrandIDsByName returns the IDs of brands whose name ILIKE-matches the
	// given search pattern (caller supplies the '%' pattern).
	BrandIDsByName(ctx context.Context, db shared.DBPool, search string) ([]int, error)
}
