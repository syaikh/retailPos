package inventory

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

// StockDeducer is the inventory-owned implementation of the sale module's
// StockDeducer port (structural typing — no import of internal/sale needed).
// internal/inventory is the canonical single-writer of product_stock
// (ADR_Modular_Monolith_Module_Boundaries §2.8, ADR_Cross_Module_Transaction_Strategy
// use case #2), so the product_stock write logic for completed-sale items lives
// here rather than inside internal/sale.
type StockDeducer struct{}

// DeductStock checks and deducts stock within the caller's transaction. Checkout
// is a Unit of Work (ADR_Cross_Module_Transaction_Strategy §2.2: Reserve Stock →
// Create Sale → Create Payment in one transaction), so the caller's tx must be
// used to preserve atomicity.
func (StockDeducer) DeductStock(ctx context.Context, tx pgx.Tx, items []shared.StockDeductItem) error {
	productIDs := make([]int, len(items))
	for i, item := range items {
		productIDs[i] = item.ProductID
	}

	rows, err := tx.Query(ctx, `SELECT product_id, COALESCE(quantity, 0) FROM product_stock WHERE product_id = ANY($1) AND warehouse_id IS NULL AND store_id IS NULL FOR UPDATE`, productIDs)
	if err != nil {
		return fmt.Errorf("batch check stock: %w", err)
	}
	stockMap := make(map[int]int, len(items))
	for rows.Next() {
		var pid, qty int
		if err := rows.Scan(&pid, &qty); err != nil {
			rows.Close()
			return fmt.Errorf("scan stock: %w", err)
		}
		stockMap[pid] = qty
	}
	rows.Close()

	for _, item := range items {
		stock, ok := stockMap[item.ProductID]
		if !ok {
			return fmt.Errorf("stock record not found for product %d", item.ProductID)
		}
		if stock < item.Quantity {
			return shared.ErrInsufficientStock
		}
	}

	stockPIDs := make([]int, len(items))
	stockQtys := make([]int, len(items))
	for i, item := range items {
		stockPIDs[i] = item.ProductID
		stockQtys[i] = item.Quantity
	}
	_, err = tx.Exec(ctx, `UPDATE product_stock SET quantity = quantity - v.qty
		FROM (SELECT unnest($1::int[]) AS product_id, unnest($2::int[]) AS qty) v
		WHERE product_stock.product_id = v.product_id AND warehouse_id IS NULL AND store_id IS NULL`, stockPIDs, stockQtys)
	if err != nil {
		return fmt.Errorf("batch deduct stock: %w", err)
	}

	return nil
}
