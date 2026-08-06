package product

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// StockSyncer is the Katalog-owned implementation of the inventory module's
// StockSyncer port (structural typing — no import of internal/inventory
// needed). internal/product owns the products table
// (ADR_Modular_Monolith_Module_Boundaries §2.8), so the products.stock mirror
// that the inventory subsystem keeps current lives here.
type StockSyncer struct{}

// SyncStock sets a product's stock column to the given value within the
// caller's transaction. It MUST run against the caller's tx so the stock
// mirror stays atomic with the product_stock write that drives it.
func (StockSyncer) SyncStock(ctx context.Context, tx pgx.Tx, productID int, stock int) error {
	_, err := tx.Exec(ctx, `UPDATE products SET stock = $1, updated_at = NOW() WHERE id = $2`, stock, productID)
	if err != nil {
		return fmt.Errorf("failed to sync product stock: %w", err)
	}
	return nil
}
