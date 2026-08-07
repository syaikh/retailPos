package category

import (
	"context"

	"retail-pos-system/internal/shared"
)

// ProductQueryProvider is the consumer-side port for the product-owned
// aggregate read the category module needs: the active-product count per
// category (management listing). internal/product is the canonical owner of
// the products table (ADR Modular_Monolith_Module_Boundaries §2.8 Katalog) and
// provides the production implementation; the composition root MUST wire it
// via SetProductQueryProvider before any listing runs — an unwired repository
// fails fast at the read point.
type ProductQueryProvider interface {
	// CountActiveByCategoryIDs returns the number of active (non-deleted)
	// products grouped by category ID for the given category IDs. IDs with no
	// active products are absent from the result map.
	CountActiveByCategoryIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]int, error)
}
