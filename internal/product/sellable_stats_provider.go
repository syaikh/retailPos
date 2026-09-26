package product

import (
	"context"
	"fmt"

	"retail-pos-system/internal/shared"
)

// SellableStats is the product-owned implementation of the store module's
// consumer-side SellableStockProvider port (structural typing — no import of
// internal/store needed). internal/product is the canonical owner of
// products/v_products_full (ADR Modular_Monolith_Module_Boundaries §2.8
// Katalog), so the store onboarding readiness check reports catalog health
// through this read instead of querying the catalog from internal/store.
type SellableStats struct{}

// SellableStockStats reports how many active products the shared catalog
// offers and how many of them have no sellable stock left (v_products_full
// surfaces the global bucket first, the bucket checkout deducts from).
// Soft-deleted products are excluded. The catalog is global, so the numbers
// are not store-scoped: a brand new store inherits whatever stock the
// catalogue already holds.
func (SellableStats) SellableStockStats(ctx context.Context, db shared.DBPool) (int, int, error) {
	var activeProducts, zeroStockProducts int
	err := db.QueryRow(ctx, `
		SELECT COUNT(*)::int,
		       COUNT(*) FILTER (WHERE COALESCE(v.stock, 0) <= 0)::int
		FROM v_products_full v
		WHERE v.status = 'active'
		  AND EXISTS (SELECT 1 FROM products p WHERE p.id = v.id AND p.deleted_at IS NULL)
	`).Scan(&activeProducts, &zeroStockProducts)
	if err != nil {
		return 0, 0, fmt.Errorf("count sellable catalog stats: %w", err)
	}
	return activeProducts, zeroStockProducts, nil
}
