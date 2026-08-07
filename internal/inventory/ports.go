package inventory

import (
	"context"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

// StockSyncer is the consumer-side port for the products.stock column, which
// lives on the Katalog-owned products table
// (ADR_Modular_Monolith_Module_Boundaries §2.8). internal/inventory owns
// product_stock and computes the authoritative quantity, but the products.stock
// mirror is written by internal/product, the canonical single-writer of the
// products table. Stock adjustment is a Unit of Work
// (ADR_Cross_Module_Transaction_Strategy), so the implementation MUST run
// against the caller's tx to preserve atomicity. The composition root MUST wire
// the port via SetStockSyncer before any adjustment path runs — an unwired
// repository fails fast at the sync point.
type StockSyncer interface {
	SyncStock(ctx context.Context, tx pgx.Tx, productID int, stock int) error
}

// LocationRackProvider is the consumer-side port for the storage_locations
// table, which lives on the Referensi-owned storage_locations table
// (ADR_Modular_Monolith_Module_Boundaries §2.8). internal/inventory
// (transaksional) must not query it directly; rack metadata is loaded through
// internal/storagelocation, the canonical single-writer. The composition root
// MUST wire the port via SetLocationRackProvider before any rack-stock path
// runs — an unwired repository fails fast at the load point.
type LocationRackProvider interface {
	GetRack(ctx context.Context, db shared.DBPool, locationID int) (*shared.LocationRack, error)
	RacksByIDs(ctx context.Context, db shared.DBPool, ids []int) ([]shared.LocationRack, error)
}

// ProductMetaProvider is the consumer-side port for the products table
// (Katalog-owned by internal/product). internal/inventory uses it to enrich
// rack-stock listings with sku/name without joining products directly. The
// composition root MUST wire the port via SetProductMetaProvider before
// ListLocationStock runs — an unwired repository fails fast at the enrichment
// point.
type ProductMetaProvider interface {
	ProductMetasByIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]shared.ProductMeta, error)
}
