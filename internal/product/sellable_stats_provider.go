package product

import (
	"context"
	"fmt"
	"sync"
	"time"

	"retail-pos-system/internal/shared"
)

// SellableStats is the product-owned implementation of the store module's
// consumer-side SellableStockProvider port (structural typing — no import of
// internal/store needed). internal/product is the canonical owner of
// products/v_products_full (ADR Modular_Monolith_Module_Boundaries §2.8
// Katalog), so the store onboarding readiness check reports catalog health
// through this read instead of querying the catalog from internal/store.
type SellableStats struct{}

// catalogStatsTTL bounds how stale the readiness catalog figures may be. The
// numbers are store-independent, so the Stores list recomputes the identical
// aggregate once per visible row; without memoisation that is one
// v_products_full scan (two LATERAL stock/supplier lookups per product) per
// row per page load. 30s keeps a freshly received PO out of the readiness
// checklist for at most half a minute while collapsing that fan-out to a
// single scan.
const catalogStatsTTL = 30 * time.Second

type sellableStatsSnapshot struct {
	active    int
	zeroStock int
}

var (
	sellableStatsMu     sync.Mutex
	sellableStatsCached sellableStatsSnapshot
	sellableStatsAt     time.Time
)

// InvalidateSellableStats drops the memoised catalog counts. Called by the
// product write paths; stock movements from other modules (sales, purchase
// orders, stock opnames) are covered by the TTL instead.
func InvalidateSellableStats() {
	sellableStatsMu.Lock()
	defer sellableStatsMu.Unlock()
	sellableStatsCached = sellableStatsSnapshot{}
	sellableStatsAt = time.Time{}
}

// SellableStockStats reports how many active products the shared catalog
// offers and how many of them have no sellable stock left (v_products_full
// surfaces the global bucket first, the bucket checkout deducts from).
// Soft-deleted products are excluded. The catalog is global, so the numbers
// are not store-scoped: a brand new store inherits whatever stock the
// catalogue already holds.
func (SellableStats) SellableStockStats(ctx context.Context, db shared.DBPool) (int, int, error) {
	if active, zero, ok := cachedSellableStats(); ok {
		return active, zero, nil
	}

	active, zero, err := querySellableStats(ctx, db)
	if err != nil {
		return 0, 0, err
	}

	sellableStatsMu.Lock()
	sellableStatsCached = sellableStatsSnapshot{active: active, zeroStock: zero}
	sellableStatsAt = time.Now()
	sellableStatsMu.Unlock()

	return active, zero, nil
}

func cachedSellableStats() (int, int, bool) {
	sellableStatsMu.Lock()
	defer sellableStatsMu.Unlock()
	if sellableStatsAt.IsZero() || time.Since(sellableStatsAt) > catalogStatsTTL {
		return 0, 0, false
	}
	return sellableStatsCached.active, sellableStatsCached.zeroStock, true
}

func querySellableStats(ctx context.Context, db shared.DBPool) (int, int, error) {
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
