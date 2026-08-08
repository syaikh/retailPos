package inventory

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

// ProductStockWriter is the inventory-owned implementation of the product
// module's ProductStockWriter port (structural typing — no import of
// internal/product needed). internal/inventory is the canonical single-writer
// of product_stock (ADR_Modular_Monolith_Module_Boundaries §2.8), so the
// product_stock row writes performed when products are created, updated,
// restored or bulk-imported live here rather than inside internal/product.
type ProductStockWriter struct{}

// SetStoreStock upserts a single product_stock row to an absolute quantity
// within the caller's transaction: the store-scoped row when StoreID is set,
// otherwise the global row. Product writes are a Unit of Work
// (ADR_Cross_Module_Transaction_Strategy), so the caller's tx must be used to
// preserve atomicity.
func (w ProductStockWriter) SetStoreStock(ctx context.Context, tx pgx.Tx, item shared.StockRowSet) error {
	if item.StoreID != nil {
		_, err := tx.Exec(ctx, `
			INSERT INTO product_stock (product_id, store_id, quantity) VALUES ($1, $2, $3)
			ON CONFLICT ON CONSTRAINT uq_product_stock DO UPDATE SET quantity = EXCLUDED.quantity
		`, item.ProductID, *item.StoreID, item.Quantity)
		if err != nil {
			return fmt.Errorf("failed to set store stock: %w", err)
		}
		return nil
	}
	tag, err := tx.Exec(ctx, `
		UPDATE product_stock SET quantity = $1, updated_at = NOW()
		WHERE product_id = $2 AND warehouse_id IS NULL AND store_id IS NULL AND location_id IS NULL
	`, item.Quantity, item.ProductID)
	if err != nil {
		return fmt.Errorf("failed to update product stock: %w", err)
	}
	if tag.RowsAffected() == 0 {
		_, err = tx.Exec(ctx, `
			INSERT INTO product_stock (product_id, quantity, updated_at) VALUES ($1, $2, NOW())
		`, item.ProductID, item.Quantity)
		if err != nil {
			return fmt.Errorf("failed to insert product stock: %w", err)
		}
	}
	return nil
}

// SetStoreStockBatch upserts many product_stock rows in a single statement,
// mixing store-scoped and global rows as given. It is used by bulk imports,
// where per-row round trips would be prohibitive.
func (w ProductStockWriter) SetStoreStockBatch(ctx context.Context, tx pgx.Tx, items []shared.StockRowSet) error {
	if len(items) == 0 {
		return nil
	}
	values := make([]string, 0, len(items))
	args := make([]interface{}, 0, len(items)*3)
	for _, item := range items {
		var storeID interface{}
		if item.StoreID != nil {
			storeID = *item.StoreID
		}
		offset := len(args)
		values = append(values, fmt.Sprintf("($%d, $%d, $%d)", offset+1, offset+2, offset+3))
		args = append(args, item.ProductID, storeID, item.Quantity)
	}
	query := fmt.Sprintf(`
		INSERT INTO product_stock (product_id, store_id, quantity)
		VALUES %s
		ON CONFLICT ON CONSTRAINT uq_product_stock DO UPDATE SET quantity = EXCLUDED.quantity
	`, strings.Join(values, ", "))
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to set product stock batch: %w", err)
	}
	return nil
}
