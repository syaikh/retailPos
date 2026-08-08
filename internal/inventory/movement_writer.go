package inventory

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

// MovementWriter is the inventory-owned implementation of the stock opname
// module's MovementWriter port (structural typing — no import of internal/
// stockopname needed). internal/inventory is the canonical owner of
// inventory_movements (ADR_Modular_Monolith_Module_Boundaries §2.8
// transaksional), so stock-opname adjustment movements are appended to the
// ledger here rather than via a direct CopyFrom inside internal/stockopname.
type MovementWriter struct{}

// InsertMovements appends rows to the inventory_movements ledger within the
// caller's transaction. Posting is a Unit of Work
// (ADR_Cross_Module_Transaction_Strategy), so the caller's tx must be used —
// the ledger write commits/rolls back atomically with the stock writes.
func (w MovementWriter) InsertMovements(ctx context.Context, tx pgx.Tx, rows []shared.InventoryMovement) error {
	if len(rows) == 0 {
		return nil
	}
	copyRows := make([][]interface{}, len(rows))
	for i, m := range rows {
		copyRows[i] = []interface{}{m.ProductID, m.QuantityChange, m.Type, m.ReferenceID, m.ReferenceTable, m.UserID, m.Notes}
	}
	_, err := tx.CopyFrom(ctx, pgx.Identifier{"inventory_movements"},
		[]string{"product_id", "quantity_change", "type", "reference_id", "reference_table", "user_id", "notes"},
		pgx.CopyFromRows(copyRows),
	)
	if err != nil {
		return fmt.Errorf("batch insert inventory movements: %w", err)
	}
	return nil
}
