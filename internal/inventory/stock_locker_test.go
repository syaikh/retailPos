package inventory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestStockLocker_LockProductStock covers the global product_stock lock the
// stock opname module takes before posting a non-location adjustment. The lock
// must run against the caller's transaction so it is held until commit/rollback.
func TestStockLocker_LockProductStock(t *testing.T) {
	ctx := context.Background()
	l := StockLocker{}

	t.Run("returns global quantities", func(t *testing.T) {
		p1 := insertTestProduct(ctx, t, "SLK-GLOBAL-1")
		p2 := insertTestProduct(ctx, t, "SLK-GLOBAL-2")
		insertTestStock(ctx, t, p1, 10)
		insertTestStock(ctx, t, p2, 7)
		setProductsStock(ctx, t, p1, 10)
		setProductsStock(ctx, t, p2, 7)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		stock, err := l.LockProductStock(ctx, tx, []int{p1, p2})
		require.NoError(t, err)
		require.Equal(t, 10, stock[p1])
		require.Equal(t, 7, stock[p2])
	})

	t.Run("empty ids returns empty map", func(t *testing.T) {
		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		stock, err := l.LockProductStock(ctx, tx, nil)
		require.NoError(t, err)
		require.Empty(t, stock)
	})

	t.Run("lock does not mutate stock", func(t *testing.T) {
		prodID := insertTestProduct(ctx, t, "SLK-GLOBAL-NOMUT")
		insertTestStock(ctx, t, prodID, 10)
		setProductsStock(ctx, t, prodID, 10)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		stock, err := l.LockProductStock(ctx, tx, []int{prodID})
		require.NoError(t, err)
		require.Equal(t, 10, stock[prodID])
		require.Equal(t, 10, stockQuantity(ctx, t, prodID))
	})
}

// TestStockLocker_LockLocationStock covers the rack product_stock lock the
// stock opname module takes before posting a location-scoped adjustment.
func TestStockLocker_LockLocationStock(t *testing.T) {
	ctx := context.Background()
	l := StockLocker{}

	t.Run("returns rack quantities with 0 default", func(t *testing.T) {
		withRack := insertTestProduct(ctx, t, "SLK-RACK-1")
		withoutRack := insertTestProduct(ctx, t, "SLK-RACK-2")
		wh := 9801
		locID := insertTestRack(ctx, t, wh, "SLK-RACK-LOC")
		insertRackRow(ctx, t, withRack, wh, locID, 5)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		stock, err := l.LockLocationStock(ctx, tx, []int{withRack, withoutRack}, locID)
		require.NoError(t, err)
		require.Equal(t, 5, stock[withRack])
		require.Equal(t, 0, stock[withoutRack])
	})

	t.Run("empty ids returns empty map", func(t *testing.T) {
		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		stock, err := l.LockLocationStock(ctx, tx, nil, 1)
		require.NoError(t, err)
		require.Empty(t, stock)
	})
}
