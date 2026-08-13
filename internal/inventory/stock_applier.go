package inventory

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

// StockApplier is the inventory-owned implementation of the stock opname
// module's StockApplier port (structural typing — no import of internal/
// stockopname needed). internal/inventory is the canonical single-writer of
// product_stock (ADR_Modular_Monolith_Module_Boundaries §2.8), so the product
// stock writes performed on stock opname posting live here rather than inside
// internal/stockopname.
type StockApplier struct{}

// SetProductStock sets a product's global stock to an absolute value, upserting
// the global product_stock row, within the caller's transaction. Stock opname
// posting is a Unit of Work (ADR_Cross_Module_Transaction_Strategy), so the
// caller's tx must be used to preserve atomicity.
func (a StockApplier) SetProductStock(ctx context.Context, tx pgx.Tx, item shared.StockSetItem) error {
	tag, err := tx.Exec(ctx, `
		UPDATE product_stock SET quantity = $1, updated_at = NOW()
		WHERE product_id = $2 AND warehouse_id IS NULL AND store_id IS NULL
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

// ReconcileLocationStock applies a signed delta to a product's rack row and
// reconciles the global row from the rack share, within the caller's
// transaction. Both rows are locked first so concurrent reconciles stay
// race-free. The global row is clamped at zero (max(global-rack, 0) + newRack)
// so an over-set rack can never drive the global row negative.
func (a StockApplier) ReconcileLocationStock(ctx context.Context, tx pgx.Tx, reconcile shared.LocationStockReconcile) error {
	var rackQty, globalQty int
	row := tx.QueryRow(ctx, `
		SELECT quantity FROM product_stock
		WHERE product_id = $1 AND location_id = $2
		FOR UPDATE
	`, reconcile.ProductID, reconcile.LocationID)
	if err := row.Scan(&rackQty); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to lock location stock: %w", err)
	}
	row = tx.QueryRow(ctx, `
		SELECT quantity FROM product_stock
		WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL AND location_id IS NULL
		FOR UPDATE
	`, reconcile.ProductID)
	if err := row.Scan(&globalQty); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to lock global stock: %w", err)
	}

	newRack := rackQty + reconcile.Delta
	newGlobal := max(globalQty-rackQty, 0) + newRack

	tag, err := tx.Exec(ctx, `
		UPDATE product_stock SET quantity = $1, updated_at = NOW()
		WHERE product_id = $2 AND location_id = $3
	`, newRack, reconcile.ProductID, reconcile.LocationID)
	if err != nil {
		return fmt.Errorf("failed to update location stock: %w", err)
	}
	if tag.RowsAffected() == 0 {
		_, err = tx.Exec(ctx, `
			INSERT INTO product_stock (product_id, quantity, warehouse_id, store_id, location_id, updated_at)
			VALUES ($1, $2, $3, $4, $5, NOW())
		`, reconcile.ProductID, newRack, reconcile.WarehouseID, reconcile.StoreID, reconcile.LocationID)
		if err != nil {
			return fmt.Errorf("failed to insert location stock: %w", err)
		}
	}

	tag, err = tx.Exec(ctx, `
		UPDATE product_stock SET quantity = $1, updated_at = NOW()
		WHERE product_id = $2 AND warehouse_id IS NULL AND store_id IS NULL AND location_id IS NULL
	`, newGlobal, reconcile.ProductID)
	if err != nil {
		return fmt.Errorf("failed to update global stock: %w", err)
	}
	if tag.RowsAffected() == 0 {
		_, err = tx.Exec(ctx, `
			INSERT INTO product_stock (product_id, quantity, updated_at) VALUES ($1, $2, NOW())
		`, reconcile.ProductID, newGlobal)
		if err != nil {
			return fmt.Errorf("failed to insert global stock: %w", err)
		}
	}

	return nil
}
