package inventory

import (
	"context"

	"github.com/jackc/pgx/v5"
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
