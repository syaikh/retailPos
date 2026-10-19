package inventory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

func TestStockChecker_AvailableStock(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	checker := StockChecker{}

	t.Run("missing product returns 0 no error", func(t *testing.T) {
		qty, err := checker.AvailableStock(ctx, dbPool, 999999)
		require.NoError(t, err)
		require.Equal(t, 0, qty)
	})

	t.Run("missing stock row returns 0 no error", func(t *testing.T) {
		p := insertTestProduct(ctx, t, "CHK-NOROW")
		// Do not insert product_stock
		qty, err := checker.AvailableStock(ctx, dbPool, p)
		require.NoError(t, err)
		require.Equal(t, 0, qty)
	})

	t.Run("global stock only", func(t *testing.T) {
		p := insertTestProduct(ctx, t, "CHK-GLOBAL")
		insertTestStock(ctx, t, p, 7)
		qty, err := checker.AvailableStock(ctx, dbPool, p)
		require.NoError(t, err)
		require.Equal(t, 7, qty)
		require.Equal(t, 7, stockQuantity(ctx, t, p)) // sanity
	})

	t.Run("store-scoped stock is not counted", func(t *testing.T) {
		p := insertTestProduct(ctx, t, "CHK-STORE")
		// global
		insertTestStock(ctx, t, p, 10)
		// store-scoped row (warehouse_id NULL, store_id set)
		_, err := dbPool.Exec(ctx, `
			INSERT INTO product_stock (product_id, warehouse_id, store_id, location_id, quantity)
			VALUES ($1, NULL, $2, NULL, 5)
			ON CONFLICT ON CONSTRAINT uq_product_stock DO UPDATE SET quantity = EXCLUDED.quantity
		`, p, 1)
		require.NoError(t, err)
		qty, err := checker.AvailableStock(ctx, dbPool, p)
		require.NoError(t, err)
		require.Equal(t, 10, qty) // global only
	})

	t.Run("warehouse-scoped stock is not counted", func(t *testing.T) {
		p := insertTestProduct(ctx, t, "CHK-WH")
		insertTestStock(ctx, t, p, 3)
		loc := insertTestRack(ctx, t, 901, "CHK-R1")
		insertRackRow(ctx, t, p, 901, loc, 25)
		qty, err := checker.AvailableStock(ctx, dbPool, p)
		require.NoError(t, err)
		require.Equal(t, 3, qty) // global only
	})

	t.Run("global 7 and store 5 returns 7", func(t *testing.T) {
		p := insertTestProduct(ctx, t, "CHK-MIX")
		insertTestStock(ctx, t, p, 7)
		// store 1 is the baseline store TruncateTestData restores; any other
		// id would violate product_stock_store_id_fkey.
		_, err := dbPool.Exec(ctx, `
			INSERT INTO product_stock (product_id, warehouse_id, store_id, location_id, quantity)
			VALUES ($1, NULL, $2, NULL, 5)
		`, p, 1)
		require.NoError(t, err)
		qty, err := checker.AvailableStock(ctx, dbPool, p)
		require.NoError(t, err)
		require.Equal(t, 7, qty)
	})
}
