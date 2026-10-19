package inventory

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

// StockChecker implements salereturn.StockChecker (structural typing — no
// import of internal/salereturn). It reads the product's global sellable row,
// the SAME row StockDeducer.DeductStock conditionally decrements
// (warehouse_id IS NULL AND store_id IS NULL), so "stock > 0 means a
// replacement is offered" and the deduction itself can never disagree about
// which row they mean.
type StockChecker struct{}

// AvailableStock returns 0 when the product has no global stock row (a
// missing row is simply zero stock, not an error).
func (StockChecker) AvailableStock(ctx context.Context, db shared.DBPool, productID int) (int, error) {
	var qty int
	err := db.QueryRow(ctx, `
		SELECT COALESCE(quantity, 0)
		FROM product_stock
		WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL
	`, productID).Scan(&qty)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to read stock for product %d: %w", productID, err)
	}
	return qty, nil
}
