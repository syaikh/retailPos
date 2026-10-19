package consignment

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// SaleConsignmentGuard implements salereturn.ConsignmentGuard (structural
// typing — no import of internal/salereturn; internal/wiring passes it to the
// salereturn service). internal/consignment single-owns
// consignment_sale_items (ADR_Modular_Monolith_Module_Boundaries), so only it
// may read which lines of a sale are consignment-owned goods.
//
// Sales Return Phase 1 rejects any return touching such a line: the sold unit
// belongs to the supplier, not the store, and no consignment restock path
// exists yet (review decision; see docs/design/sales-return-plan.md
// "Review amendments").
type SaleConsignmentGuard struct{}

// ConsignmentProductIDs returns the distinct product ids of a sale's
// consignment lines; empty means the sale has none.
func (SaleConsignmentGuard) ConsignmentProductIDs(ctx context.Context, tx pgx.Tx, saleID int) ([]int, error) {
	rows, err := tx.Query(ctx, `
		SELECT DISTINCT product_id
		FROM consignment_sale_items
		WHERE sale_id = $1
		ORDER BY product_id
	`, saleID)
	if err != nil {
		return nil, fmt.Errorf("failed to load consignment items of sale %d: %w", saleID, err)
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan consignment product of sale %d: %w", saleID, err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate consignment products of sale %d: %w", saleID, err)
	}
	return ids, nil
}
