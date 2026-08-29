package product

import (
	"context"
	"fmt"
	"strings"

	"retail-pos-system/internal/shared"
)

// PricingLookup is the product-owned implementation of the pricing
// module's consumer-side port (pricing.ProductPricingProvider, structural
// typing — no import of internal/pricing needed). internal/product is the
// canonical owner of the products and tax_classes tables
// (ADR_Modular_Monolith_Module_Boundaries §2.8 Katalog), so the base-price,
// scope, cost/tax, and autocomplete reads that internal/pricing needs are
// computed here rather than via cross-context SQL inside internal/pricing.
type PricingLookup struct{}

// BasePricesByIDs returns product base prices keyed by product ID. IDs with no
// matching non-deleted product are absent from the map.
func (PricingLookup) BasePricesByIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]int, error) {
	if len(ids) == 0 {
		return map[int]int{}, nil
	}
	prices := make(map[int]int, len(ids))
	rows, err := db.Query(ctx, `
		SELECT id, price
		FROM products
		WHERE id = ANY($1) AND deleted_at IS NULL
	`, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to list base prices: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, price int
		if err := rows.Scan(&id, &price); err != nil {
			return nil, fmt.Errorf("failed to scan base price: %w", err)
		}
		prices[id] = price
	}
	return prices, rows.Err()
}

// ProductScopesByIDs returns the category/brand scope keyed by product ID. IDs
// with no matching non-deleted product are absent from the map.
func (PricingLookup) ProductScopesByIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]shared.ProductScope, error) {
	if len(ids) == 0 {
		return map[int]shared.ProductScope{}, nil
	}
	scopes := make(map[int]shared.ProductScope, len(ids))
	rows, err := db.Query(ctx, `
		SELECT id, category_id, brand_id
		FROM products
		WHERE id = ANY($1) AND deleted_at IS NULL
	`, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to list product scopes: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var scope shared.ProductScope
		if err := rows.Scan(&id, &scope.CategoryID, &scope.BrandID); err != nil {
			return nil, fmt.Errorf("failed to scan product scope: %w", err)
		}
		scopes[id] = scope
	}
	return scopes, rows.Err()
}

// ProductCostTaxesByIDs returns cost/tax/name rows keyed by product ID. IDs
// with no matching non-deleted product are absent from the map.
func (PricingLookup) ProductCostTaxesByIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]shared.ProductCostTax, error) {
	if len(ids) == 0 {
		return map[int]shared.ProductCostTax{}, nil
	}
	costs := make(map[int]shared.ProductCostTax, len(ids))
	rows, err := db.Query(ctx, `
		SELECT p.id, COALESCE(p.cost, 0), p.tax_class_id, tc.rate_percent, p.name
		FROM products p
		LEFT JOIN tax_classes tc ON tc.id = p.tax_class_id
		WHERE p.id = ANY($1) AND p.deleted_at IS NULL
	`, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to list product cost/tax: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var ct shared.ProductCostTax
		if err := rows.Scan(&id, &ct.Cost, &ct.TaxClassID, &ct.TaxRate, &ct.ProductName); err != nil {
			return nil, fmt.Errorf("failed to scan product cost/tax: %w", err)
		}
		costs[id] = ct
	}
	return costs, rows.Err()
}

// ProductIDsByName returns the IDs of non-deleted products whose name
// ILIKE-matches the given search pattern (caller supplies the '%' pattern).
func (PricingLookup) ProductIDsByName(ctx context.Context, db shared.DBPool, search string) ([]int, error) {
	rows, err := db.Query(ctx, `
		SELECT id
		FROM products
		WHERE name ILIKE $1 AND deleted_at IS NULL
	`, search)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve pricing product ids by name: %w", err)
	}
	defer rows.Close()
	ids := []int{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan pricing product id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// SearchPricingProducts returns product autocomplete rows (active, non-deleted)
// matching name, sku, or barcode, ordered by name.
func (PricingLookup) SearchPricingProducts(ctx context.Context, db shared.DBPool, query string, limit int) ([]shared.ProductSearchResult, error) {
	if query == "" {
		return []shared.ProductSearchResult{}, nil
	}
	like := "%" + strings.ToLower(query) + "%"
	rows, err := db.Query(ctx, `
		SELECT id, name, sku, price
		FROM products
		WHERE deleted_at IS NULL AND status = 'active'
		  AND (LOWER(name) LIKE $1 OR LOWER(sku) LIKE $1 OR LOWER(barcode) LIKE $1)
		ORDER BY name ASC
		LIMIT $2
	`, like, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search pricing products: %w", err)
	}
	defer rows.Close()
	results := []shared.ProductSearchResult{}
	for rows.Next() {
		var p shared.ProductSearchResult
		if err := rows.Scan(&p.ID, &p.Name, &p.SKU, &p.Price); err != nil {
			return nil, fmt.Errorf("failed to scan pricing product search row: %w", err)
		}
		results = append(results, p)
	}
	return results, rows.Err()
}
