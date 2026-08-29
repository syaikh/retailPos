package inventory

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

// ConsignmentAdjuster is the inventory-owned implementation of the consignment
// module's StockAdjuster port (structural typing — no import of
// internal/consignment needed). internal/inventory is the canonical single-writer
// of product_stock and the inventory_movements ledger
// (ADR_Modular_Monolith_Module_Boundaries §2.8), so the product_stock writes
// triggered by consignment receipts / pending-return holds / returns live here
// rather than inside internal/consignment. The consignment ownership ledger
// (consignment_stock) is separate and owned by internal/consignment.
type ConsignmentAdjuster struct{}

// ApplyConsignmentDelta applies a signed delta to a product's global
// product_stock row and appends a matching inventory_movements ledger entry,
// within the caller's transaction. Receipt/pending-return/return flows are Units
// of Work (ADR_Cross_Module_Transaction_Strategy), so the caller's tx must be
// used to preserve atomicity.
//
// The global row is clamped at zero: consignment quantities must never drive a
// store's product_stock negative (a consignment product is either store-owned or
// consignment-owned at any time — BR-18 — so the global row tracks the store's
// own quantity while the consignment_stock ledger tracks the supplier's). No
// global CHECK constraint is added; stock-opname absolute writes via
// inventory_adjustments keep their semantics.
func (a ConsignmentAdjuster) ApplyConsignmentDelta(ctx context.Context, tx pgx.Tx, delta shared.ConsignmentStockDelta) error {
	if delta.Delta == 0 {
		// Nothing moved: no stock change and no ledger row, so the two sides
		// never disagree (a movement without a stock change would be noise).
		return nil
	}
	if err := a.applyGlobalDelta(ctx, tx, delta.ProductID, delta.Delta); err != nil {
		return err
	}
	if err := a.writeMovement(ctx, tx, delta); err != nil {
		return err
	}
	return nil
}

func (a ConsignmentAdjuster) applyGlobalDelta(ctx context.Context, tx pgx.Tx, productID, delta int) error {
	if delta == 0 {
		return nil
	}

	var current int
	row := tx.QueryRow(ctx, `
		SELECT quantity FROM product_stock
		WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL AND location_id IS NULL
		FOR UPDATE
	`, productID)
	if err := row.Scan(&current); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to lock global stock: %w", err)
	}

	newQty := current + delta
	if newQty < 0 {
		newQty = 0
	}

	tag, err := tx.Exec(ctx, `
		UPDATE product_stock SET quantity = $1, updated_at = NOW()
		WHERE product_id = $2 AND warehouse_id IS NULL AND store_id IS NULL AND location_id IS NULL
	`, newQty, productID)
	if err != nil {
		return fmt.Errorf("failed to update global stock: %w", err)
	}
	if tag.RowsAffected() == 0 {
		_, err = tx.Exec(ctx, `
			INSERT INTO product_stock (product_id, quantity, updated_at) VALUES ($1, $2, NOW())
		`, productID, newQty)
		if err != nil {
			return fmt.Errorf("failed to insert global stock: %w", err)
		}
	}
	return nil
}

func (a ConsignmentAdjuster) writeMovement(ctx context.Context, tx pgx.Tx, delta shared.ConsignmentStockDelta) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO inventory_movements
			(product_id, quantity_change, type, reference_id, reference_table, user_id, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, delta.ProductID, delta.Delta, delta.MovementType, delta.ReferenceID, delta.ReferenceTable, delta.UserID, delta.Notes)
	if err != nil {
		return fmt.Errorf("failed to insert inventory movement: %w", err)
	}
	return nil
}
