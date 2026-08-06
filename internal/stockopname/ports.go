package stockopname

import (
	"context"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

// StockApplier is the consumer-side port for the inventory subsystem's
// product_stock writes performed when a stock opname posts its adjustment.
// Posting is a Unit of Work (ADR_Cross_Module_Transaction_Strategy), so the
// implementation MUST run against the caller's tx to preserve atomicity — a
// session must never post while its stock write fails. internal/inventory is the
// canonical single-writer of product_stock and provides the production
// implementation; the composition root MUST wire it via SetStockApplier before
// any posting path runs — an unwired service fails fast at runtime.
type StockApplier interface {
	SetProductStock(ctx context.Context, tx pgx.Tx, item shared.StockSetItem) error
	ReconcileLocationStock(ctx context.Context, tx pgx.Tx, reconcile shared.LocationStockReconcile) error
}
