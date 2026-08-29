package product

import (
	"context"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

// StockWriter is the consumer-side port for the inventory subsystem's
// product_stock row writes performed when products are created, updated,
// restored or bulk-imported. Product writes are a Unit of Work
// (ADR_Cross_Module_Transaction_Strategy), so the implementation MUST run
// against the caller's tx to preserve atomicity. internal/inventory is the
// canonical single-writer of product_stock and provides the production
// implementation; the composition root MUST wire it via SetProductStockWriter
// before any stock write path runs — an unwired repository fails fast at
// runtime.
type StockWriter interface {
	SetStoreStock(ctx context.Context, tx pgx.Tx, item shared.StockRowSet) error
	SetStoreStockBatch(ctx context.Context, tx pgx.Tx, items []shared.StockRowSet) error
}
