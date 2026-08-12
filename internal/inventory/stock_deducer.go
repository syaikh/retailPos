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
//
// P2-1 D2: deduction is an atomic conditional decrement — each item issues a
// single `UPDATE ... SET quantity = quantity - n WHERE quantity >= n` and the
// sale aborts with ErrInsufficientStock when 0 rows are affected. Combined with
// the quantity pre-check (which detects missing stock rows under a row lock),
// a duplicate or concurrent deduction can never drive product_stock negative.
// No global CHECK constraint is added; stock-opname absolute writes via
// inventory_adjustments keep their semantics.
//
// On any error the caller MUST roll back (or otherwise discard) the transaction:
// items are decremented as they are processed, so a multi-item deduction that
// fails midway leaves earlier items already subtracted inside the tx. All current
// callers run this inside a deferred Rollback, so the fail-closed guarantee holds.
func (StockDeducer) DeductStock(ctx context.Context, tx pgx.Tx, items []shared.StockDeductItem) error {
	productIDs := make([]int, len(items))
	for i, item := range items {
		productIDs[i] = item.ProductID
	}

	// Lock the target rows and detect missing stock records. The FOR UPDATE
	// serializes concurrent deductions on the same rows, and lets us distinguish
	// "no stock row" from "insufficient stock" before the conditional UPDATE.
	rows, err := tx.Query(ctx, `SELECT product_id FROM product_stock WHERE product_id = ANY($1) AND warehouse_id IS NULL AND store_id IS NULL FOR UPDATE`, productIDs)
	if err != nil {
		return fmt.Errorf("batch check stock: %w", err)
	}
	found := make(map[int]bool, len(items))
	for rows.Next() {
		var pid int
		if err := rows.Scan(&pid); err != nil {
			rows.Close()
			return fmt.Errorf("scan stock: %w", err)
		}
		found[pid] = true
	}
	rows.Close()

	for _, item := range items {
		if !found[item.ProductID] {
			return fmt.Errorf("stock record not found for product %d", item.ProductID)
		}
	}

	// Atomic check-and-decrement per item. The WHERE clause re-checks the
	// available quantity, so even a stale pre-read (or a duplicate line item
	// that slipped past dedupe) fails closed instead of overselling.
	for _, item := range items {
		tag, err := tx.Exec(ctx, `UPDATE product_stock SET quantity = quantity - $1
			WHERE product_id = $2 AND warehouse_id IS NULL AND store_id IS NULL AND quantity >= $1`,
			item.Quantity, item.ProductID)
		if err != nil {
			return fmt.Errorf("deduct stock: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return shared.ErrInsufficientStock
		}
	}

	return nil
}
