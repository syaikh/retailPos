package inventory

import (
	"context"
	"fmt"
	"sort"
	"strings"

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
// P2-1 D2: deduction is an atomic conditional decrement issued as ONE statement
// for the whole checkout — a single `UPDATE ... SET quantity = quantity - <net
// per product> CASE ... WHERE quantity >= <net>` — so a cart-sized checkout
// does not run one UPDATE per line (backend review P2). Combined with the
// quantity pre-check (which detects missing stock rows under a row lock), a
// duplicate or concurrent deduction can never drive product_stock negative. No
// global CHECK constraint is added; stock-opname absolute writes via
// inventory_adjustments keep their semantics.
//
// On any error the caller MUST roll back (or otherwise discard) the transaction:
// the single UPDATE is all-or-nothing, and callers run this inside a deferred
// Rollback, so the fail-closed guarantee holds.
func (StockDeducer) DeductStock(ctx context.Context, tx pgx.Tx, items []shared.StockDeductItem) error {
	if len(items) == 0 {
		return nil
	}

	// Net quantity per product so duplicate lines still subtract fully (the
	// per-row semantics of the old one-UPDATE-per-line loop).
	net := make(map[int]int, len(items))
	for _, item := range items {
		net[item.ProductID] += item.Quantity
	}
	productIDs := make([]int, 0, len(net))
	for p := range net {
		productIDs = append(productIDs, p)
	}
	sort.Ints(productIDs)

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

	// One conditional check-and-decrement for the entire checkout. Per product
	// the CASE subtracts its net quantity and the WHERE guard re-checks the
	// available quantity, so even a stale pre-read (or a duplicate line item
	// that slipped past dedupe) fails closed instead of overselling; the
	// affected-row count must equal the number of distinct products.
	var query strings.Builder
	query.WriteString(`UPDATE product_stock SET quantity = quantity - CASE product_id`)
	args := make([]any, 0, len(productIDs)*2+1)
	for i, p := range productIDs {
		fmt.Fprintf(&query, " WHEN $%d THEN $%d", i*2+1, i*2+2)
		args = append(args, p, net[p])
	}
	args = append(args, productIDs) // $len(productIDs)*2+1
	query.WriteString(` ELSE 0 END`)
	query.WriteString(` WHERE warehouse_id IS NULL AND store_id IS NULL`)
	fmt.Fprintf(&query, ` AND product_id = ANY($%d)`, len(productIDs)*2+1)
	query.WriteString(` AND quantity >= CASE product_id`)
	for i := range productIDs {
		fmt.Fprintf(&query, " WHEN $%d THEN $%d", i*2+1, i*2+2)
	}
	query.WriteString(` ELSE 0 END`)

	tag, err := tx.Exec(ctx, query.String(), args...)
	if err != nil {
		return fmt.Errorf("deduct stock: %w", err)
	}
	if int(tag.RowsAffected()) != len(productIDs) {
		return shared.ErrInsufficientStock
	}

	return nil
}
