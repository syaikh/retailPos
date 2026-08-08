package inventory

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// StockLocker is the inventory-owned implementation of the stock opname
// module's StockLocker port (structural typing — no import of internal/
// stockopname needed). internal/inventory is the canonical owner of
// product_stock (ADR_Modular_Monolith_Module_Boundaries §2.8), so the FOR
// UPDATE locks that serialize stock opname posting against concurrent stock
// changes live here rather than inside internal/stockopname.
type StockLocker struct{}

// LockProductStock locks the global product_stock rows of the given products
// within the caller's transaction and returns their current quantities.
// Posting is a Unit of Work (ADR_Cross_Module_Transaction_Strategy), so the
// caller's tx must be used — the locks are held until commit/rollback.
func (l StockLocker) LockProductStock(ctx context.Context, tx pgx.Tx, productIDs []int) (map[int]int, error) {
	stock := make(map[int]int, len(productIDs))
	if len(productIDs) == 0 {
		return stock, nil
	}
	rows, err := tx.Query(ctx, `
		SELECT product_id, quantity FROM product_stock
		WHERE product_id = ANY($1::int[]) AND warehouse_id IS NULL AND store_id IS NULL
		FOR UPDATE
	`, productIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to lock product stock: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var pid, qty int
		if err := rows.Scan(&pid, &qty); err != nil {
			return nil, fmt.Errorf("failed to scan product stock: %w", err)
		}
		stock[pid] = qty
	}
	return stock, rows.Err()
}

// LockLocationStock locks the rack product_stock rows of the given products on
// a location within the caller's transaction and returns the current rack
// quantities (0 when no rack row exists).
func (l StockLocker) LockLocationStock(ctx context.Context, tx pgx.Tx, productIDs []int, locationID int) (map[int]int, error) {
	stock := make(map[int]int, len(productIDs))
	if len(productIDs) == 0 {
		return stock, nil
	}
	for _, pid := range productIDs {
		stock[pid] = 0
	}
	rows, err := tx.Query(ctx, `
		SELECT product_id, quantity FROM product_stock
		WHERE product_id = ANY($1::int[]) AND location_id = $2
		FOR UPDATE
	`, productIDs, locationID)
	if err != nil {
		return nil, fmt.Errorf("failed to lock location stock: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var pid, qty int
		if err := rows.Scan(&pid, &qty); err != nil {
			return nil, fmt.Errorf("failed to scan location stock: %w", err)
		}
		stock[pid] = qty
	}
	return stock, rows.Err()
}
