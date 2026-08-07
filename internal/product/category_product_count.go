package product

import (
	"context"

	"retail-pos-system/internal/shared"
)

// CategoryProductCountProvider is the product-owned implementation of the
// category module's consumer-side port (category.ProductQueryProvider,
// structural typing — no import of internal/category needed). internal/product
// is the canonical owner of the products table (ADR
// Modular_Monolith_Module_Boundaries §2.8 Katalog), so the per-category
// active-product aggregate that internal/category lists is computed here
// rather than via direct SQL inside internal/category.
type CategoryProductCountProvider struct{}

// CountActiveByCategoryIDs returns the number of active (deleted_at IS NULL)
// products grouped by category for the given category IDs. IDs with no active
// products are absent from the result map.
func (CategoryProductCountProvider) CountActiveByCategoryIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]int, error) {
	if len(ids) == 0 {
		return map[int]int{}, nil
	}
	counts := make(map[int]int, len(ids))
	rows, err := db.Query(ctx, `
		SELECT category_id, COUNT(*) AS n
		FROM products
		WHERE category_id = ANY($1) AND deleted_at IS NULL
		GROUP BY category_id
	`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var categoryID, n int
		if err := rows.Scan(&categoryID, &n); err != nil {
			return nil, err
		}
		counts[categoryID] = n
	}
	return counts, rows.Err()
}
