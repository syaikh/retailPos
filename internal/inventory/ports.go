package inventory

import (
	"context"

	"retail-pos-system/internal/shared"
)

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
