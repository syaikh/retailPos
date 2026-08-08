package category

import (
	"context"

	"retail-pos-system/internal/shared"
)

// CategoryNamesProvider is the category-owned implementation of the stockopname
// module's consumer-side scope-name read (stockopname.ScopeNameResolver,
// structural typing — no import of internal/stockopname needed). internal/category
// owns the categories table (ADR §2.8 Katalog), so category scope names are
// resolved here rather than via a correlated subquery inside internal/stockopname.
type CategoryNamesProvider struct{}

// CategoryNamesByIDs returns a map of category id -> name for the given ids.
// IDs without a category row are absent from the result map.
func (CategoryNamesProvider) CategoryNamesByIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]string, error) {
	if len(ids) == 0 {
		return map[int]string{}, nil
	}
	names := make(map[int]string, len(ids))
	rows, err := db.Query(ctx, `
		SELECT id, name
		FROM categories
		WHERE id = ANY($1)
	`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		names[id] = name
	}
	return names, rows.Err()
}

// CategoryIDsByName returns the IDs of categories whose name ILIKE-matches the
// given search pattern (caller supplies the '%' pattern). Used by
// internal/pricing to resolve the pricing rule listing search filter without a
// categories EXISTS clause (see pricing.CategoryNameSearchProvider).
func (CategoryNamesProvider) CategoryIDsByName(ctx context.Context, db shared.DBPool, search string) ([]int, error) {
	rows, err := db.Query(ctx, `
		SELECT id
		FROM categories
		WHERE name ILIKE $1
	`, search)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []int{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
