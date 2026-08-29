package inventory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

func movementCount(ctx context.Context, t *testing.T, productID, refID int) int {
	t.Helper()
	var n int
	err := dbPool.QueryRow(ctx, `
		SELECT COUNT(*) FROM inventory_movements WHERE product_id = $1 AND reference_id = $2`,
		productID, refID).Scan(&n)
	require.NoError(t, err)
	return n
}

// TestConsignmentAdjuster_ApplyConsignmentDelta covers the consignment module's
// product_stock + inventory_movements writes (single-writer of product_stock and
// the movement ledger, ADR_Modular_Monolith_Module_Boundaries §2.8). The delta
// must run against the caller's transaction.
func TestConsignmentAdjuster_ApplyConsignmentDelta(t *testing.T) {
	ctx := context.Background()
	a := ConsignmentAdjuster{}

	t.Run("adds delta to existing global row + ledger", func(t *testing.T) {
		insertTestUser(ctx, t, 1)
		prodID := insertTestProduct(ctx, t, "CA-ADD")
		insertTestStock(ctx, t, prodID, 10)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, a.ApplyConsignmentDelta(ctx, tx, shared.ConsignmentStockDelta{
			ProductID: prodID, Delta: 5, MovementType: "consignment_receipt",
			ReferenceID: 1, ReferenceTable: "consignment_receipts", UserID: 1, Notes: "receipt",
		}))
		require.NoError(t, tx.Commit(ctx))

		require.Equal(t, 15, stockQuantity(ctx, t, prodID))
		require.Equal(t, 1, movementCount(ctx, t, prodID, 1))
	})

	t.Run("upserts global row when missing", func(t *testing.T) {
		insertTestUser(ctx, t, 1)
		prodID := insertTestProduct(ctx, t, "CA-INSERT")

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, a.ApplyConsignmentDelta(ctx, tx, shared.ConsignmentStockDelta{
			ProductID: prodID, Delta: 7, MovementType: "consignment_receipt",
			ReferenceID: 2, ReferenceTable: "consignment_receipts", UserID: 1, Notes: "receipt",
		}))
		require.NoError(t, tx.Commit(ctx))

		require.Equal(t, 7, stockQuantity(ctx, t, prodID))
		require.Equal(t, 1, movementCount(ctx, t, prodID, 2))
	})

	t.Run("clamps global at zero", func(t *testing.T) {
		insertTestUser(ctx, t, 1)
		prodID := insertTestProduct(ctx, t, "CA-CLAMP")
		insertTestStock(ctx, t, prodID, 3)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, a.ApplyConsignmentDelta(ctx, tx, shared.ConsignmentStockDelta{
			ProductID: prodID, Delta: -10, MovementType: "consignment_pending_return",
			ReferenceID: 3, ReferenceTable: "consignment_pending_returns", UserID: 1, Notes: "pull",
		}))
		require.NoError(t, tx.Commit(ctx))

		require.Equal(t, 0, stockQuantity(ctx, t, prodID))
		require.Equal(t, 1, movementCount(ctx, t, prodID, 3))
	})

	t.Run("writes negative delta row", func(t *testing.T) {
		insertTestUser(ctx, t, 1)
		prodID := insertTestProduct(ctx, t, "CA-NEG")
		insertTestStock(ctx, t, prodID, 20)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, a.ApplyConsignmentDelta(ctx, tx, shared.ConsignmentStockDelta{
			ProductID: prodID, Delta: -6, MovementType: "consignment_return",
			ReferenceID: 4, ReferenceTable: "consignment_returns", UserID: 1, Notes: "return",
		}))
		require.NoError(t, tx.Commit(ctx))

		require.Equal(t, 14, stockQuantity(ctx, t, prodID))
		require.Equal(t, 1, movementCount(ctx, t, prodID, 4))
	})

	t.Run("zero delta writes nothing", func(t *testing.T) {
		insertTestUser(ctx, t, 1)
		prodID := insertTestProduct(ctx, t, "CA-ZERO")
		insertTestStock(ctx, t, prodID, 5)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, a.ApplyConsignmentDelta(ctx, tx, shared.ConsignmentStockDelta{
			ProductID: prodID, Delta: 0, MovementType: "consignment_receipt",
			ReferenceID: 5, ReferenceTable: "consignment_receipts", UserID: 1, Notes: "",
		}))
		require.NoError(t, tx.Commit(ctx))

		require.Equal(t, 5, stockQuantity(ctx, t, prodID))
		require.Equal(t, 0, movementCount(ctx, t, prodID, 5))
	})

	t.Run("rollback undoes delta and ledger", func(t *testing.T) {
		insertTestUser(ctx, t, 1)
		prodID := insertTestProduct(ctx, t, "CA-ROLLBACK")
		insertTestStock(ctx, t, prodID, 10)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, a.ApplyConsignmentDelta(ctx, tx, shared.ConsignmentStockDelta{
			ProductID: prodID, Delta: 4, MovementType: "consignment_receipt",
			ReferenceID: 6, ReferenceTable: "consignment_receipts", UserID: 1, Notes: "receipt",
		}))
		require.NoError(t, tx.Rollback(ctx))

		require.Equal(t, 10, stockQuantity(ctx, t, prodID))
		require.Equal(t, 0, movementCount(ctx, t, prodID, 6))
	})
}
