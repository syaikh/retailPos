package inventory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

// stockQuantity reads the default (non-rack, non-warehouse) stock row.
func stockQuantity(ctx context.Context, t *testing.T, productID int) int {
	t.Helper()
	var qty int
	err := dbPool.QueryRow(ctx, `SELECT quantity FROM product_stock WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL`, productID).Scan(&qty)
	require.NoError(t, err)
	return qty
}

// TestStockDeducer_DeductStock covers the canonical single-writer of
// product_stock (ADR_Modular_Monolith_Module_Boundaries §2.8) that the sale
// module uses in production. It must run against the caller's transaction.
func TestStockDeducer_DeductStock(t *testing.T) {
	ctx := context.Background()
	d := StockDeducer{}

	t.Run("deducts within caller tx and commits", func(t *testing.T) {
		prodID := insertTestProduct(ctx, t, "SD-CMT")
		insertTestStock(ctx, t, prodID, 10)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, d.DeductStock(ctx, tx, []shared.StockDeductItem{{ProductID: prodID, Quantity: 3}}))
		require.NoError(t, tx.Commit(ctx))

		require.Equal(t, 7, stockQuantity(ctx, t, prodID))
	})

	t.Run("rollback undoes deduction", func(t *testing.T) {
		prodID := insertTestProduct(ctx, t, "SD-RB")
		insertTestStock(ctx, t, prodID, 10)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, d.DeductStock(ctx, tx, []shared.StockDeductItem{{ProductID: prodID, Quantity: 3}}))
		require.NoError(t, tx.Rollback(ctx))

		require.Equal(t, 10, stockQuantity(ctx, t, prodID))
	})

	t.Run("deducts multiple products in one batch", func(t *testing.T) {
		p1 := insertTestProduct(ctx, t, "SD-M1")
		insertTestStock(ctx, t, p1, 10)
		p2 := insertTestProduct(ctx, t, "SD-M2")
		insertTestStock(ctx, t, p2, 20)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, d.DeductStock(ctx, tx, []shared.StockDeductItem{
			{ProductID: p1, Quantity: 1},
			{ProductID: p2, Quantity: 2},
		}))
		require.NoError(t, tx.Commit(ctx))

		require.Equal(t, 9, stockQuantity(ctx, t, p1))
		require.Equal(t, 18, stockQuantity(ctx, t, p2))
	})

	t.Run("insufficient stock", func(t *testing.T) {
		prodID := insertTestProduct(ctx, t, "SD-INS")
		insertTestStock(ctx, t, prodID, 2)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		err = d.DeductStock(ctx, tx, []shared.StockDeductItem{{ProductID: prodID, Quantity: 5}})
		require.ErrorIs(t, err, shared.ErrInsufficientStock)
	})

	t.Run("record not found", func(t *testing.T) {
		var prodID int
		err := dbPool.QueryRow(ctx,
			`INSERT INTO products (sku, name, price) VALUES ($1, $2, 10000) RETURNING id`,
			"SD-NF", "No Stock Row",
		).Scan(&prodID)
		require.NoError(t, err)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		err = d.DeductStock(ctx, tx, []shared.StockDeductItem{{ProductID: prodID, Quantity: 1}})
		require.ErrorContains(t, err, "stock record not found")
	})
}
