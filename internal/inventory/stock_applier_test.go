package inventory

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/product"
	"retail-pos-system/internal/shared"
)

// insertTestRack inserts a warehouse + active storage location and returns the
// location id.
func insertTestRack(ctx context.Context, t *testing.T, warehouseID int, code string) int {
	t.Helper()
	_, err := dbPool.Exec(ctx,
		`INSERT INTO warehouses (id, name, code, store_id, is_active) VALUES ($1, $2, $3, NULL, true) ON CONFLICT (id) DO NOTHING`,
		warehouseID, "Test WH", fmt.Sprintf("SA-WH-%d", warehouseID),
	)
	require.NoError(t, err)
	var locID int
	err = dbPool.QueryRow(ctx,
		`INSERT INTO storage_locations (code, name, warehouse_id, is_active) VALUES ($1, $2, $3, true) RETURNING id`,
		code, "Test Rack "+code, warehouseID).Scan(&locID)
	require.NoError(t, err)
	return locID
}

func insertRackRow(ctx context.Context, t *testing.T, productID, warehouseID, locationID, quantity int) {
	t.Helper()
	_, err := dbPool.Exec(ctx,
		`INSERT INTO product_stock (product_id, warehouse_id, store_id, location_id, quantity)
		 VALUES ($1, $2, NULL, $3, $4)
		 ON CONFLICT ON CONSTRAINT uq_product_stock DO UPDATE SET quantity = EXCLUDED.quantity`,
		productID, warehouseID, locationID, quantity,
	)
	require.NoError(t, err)
}

func setProductsStock(ctx context.Context, t *testing.T, productID, quantity int) {
	t.Helper()
	_, err := dbPool.Exec(ctx, `UPDATE products SET stock = $1 WHERE id = $2`, quantity, productID)
	require.NoError(t, err)
}

func rackQty(ctx context.Context, t *testing.T, productID, locationID int) int {
	t.Helper()
	var qty int
	err := dbPool.QueryRow(ctx,
		`SELECT quantity FROM product_stock WHERE product_id = $1 AND location_id = $2`,
		productID, locationID).Scan(&qty)
	require.NoError(t, err)
	return qty
}

func globalQty(ctx context.Context, t *testing.T, productID int) int {
	t.Helper()
	var qty int
	err := dbPool.QueryRow(ctx,
		`SELECT quantity FROM product_stock WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL AND location_id IS NULL`,
		productID).Scan(&qty)
	require.NoError(t, err)
	return qty
}

func productsStock(ctx context.Context, t *testing.T, productID int) int {
	t.Helper()
	var qty int
	err := dbPool.QueryRow(ctx, `SELECT stock FROM products WHERE id = $1`, productID).Scan(&qty)
	require.NoError(t, err)
	return qty
}

// TestStockApplier_SetProductStock covers the canonical single-writer of
// product_stock (ADR_Modular_Monolith_Module_Boundaries §2.8) that the stock
// opname module uses when posting a non-location adjustment. It must run against
// the caller's transaction.
func TestStockApplier_SetProductStock(t *testing.T) {
	ctx := context.Background()
	a := StockApplier{StockSyncer: product.StockSyncer{}}

	t.Run("upserts existing global row and syncs products.stock", func(t *testing.T) {
		prodID := insertTestProduct(ctx, t, "SAS-UPSERT")
		insertTestStock(ctx, t, prodID, 10)
		setProductsStock(ctx, t, prodID, 10)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, a.SetProductStock(ctx, tx, shared.StockSetItem{ProductID: prodID, Quantity: 12}))
		require.NoError(t, tx.Commit(ctx))

		require.Equal(t, 12, stockQuantity(ctx, t, prodID))
		require.Equal(t, 12, productsStock(ctx, t, prodID))
	})

	t.Run("inserts global row lazily and syncs products.stock", func(t *testing.T) {
		prodID := insertTestProduct(ctx, t, "SAS-INSERT")
		setProductsStock(ctx, t, prodID, 0)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, a.SetProductStock(ctx, tx, shared.StockSetItem{ProductID: prodID, Quantity: 7}))
		require.NoError(t, tx.Commit(ctx))

		require.Equal(t, 7, stockQuantity(ctx, t, prodID))
		require.Equal(t, 7, productsStock(ctx, t, prodID))
	})

	t.Run("rollback undoes set", func(t *testing.T) {
		prodID := insertTestProduct(ctx, t, "SAS-ROLLBACK")
		insertTestStock(ctx, t, prodID, 10)
		setProductsStock(ctx, t, prodID, 10)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, a.SetProductStock(ctx, tx, shared.StockSetItem{ProductID: prodID, Quantity: 5}))
		require.NoError(t, tx.Rollback(ctx))

		require.Equal(t, 10, stockQuantity(ctx, t, prodID))
		require.Equal(t, 10, productsStock(ctx, t, prodID))
	})
}

// TestStockApplier_ReconcileLocationStock covers the location-scoped reconcile
// (rack delta applied + global recomputed from the rack share) used by
// location-scoped stock opname posting.
func TestStockApplier_ReconcileLocationStock(t *testing.T) {
	ctx := context.Background()
	a := StockApplier{StockSyncer: product.StockSyncer{}}
	reconcile := func(prodID, locID int, wh *int, delta int) shared.LocationStockReconcile {
		return shared.LocationStockReconcile{ProductID: prodID, LocationID: locID, WarehouseID: wh, Delta: delta}
	}

	t.Run("creates rows lazily", func(t *testing.T) {
		prodID := insertTestProduct(ctx, t, "SAR-LAZY")
		insertTestStock(ctx, t, prodID, 10)
		setProductsStock(ctx, t, prodID, 10)
		wh := 9701
		locID := insertTestRack(ctx, t, wh, "SAR-LAZY-LOC")

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, a.ReconcileLocationStock(ctx, tx, reconcile(prodID, locID, &wh, 3)))
		require.NoError(t, tx.Commit(ctx))

		require.Equal(t, 3, rackQty(ctx, t, prodID, locID))
		require.Equal(t, 13, globalQty(ctx, t, prodID))
		require.Equal(t, 13, productsStock(ctx, t, prodID))
	})

	t.Run("reconciles global from count", func(t *testing.T) {
		// After a sale the global row moved but the rack row did not (global 7 /
		// rack 10). A physical count of 7 gives delta -3; the global must be
		// recomputed as max(global-rack, 0) + newRack = 7, not 4.
		prodID := insertTestProduct(ctx, t, "SAR-DESYNC")
		insertTestStock(ctx, t, prodID, 7)
		wh := 9702
		locID := insertTestRack(ctx, t, wh, "SAR-DESYNC-LOC")
		insertRackRow(ctx, t, prodID, wh, locID, 10)
		setProductsStock(ctx, t, prodID, 7)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, a.ReconcileLocationStock(ctx, tx, reconcile(prodID, locID, &wh, -3)))
		require.NoError(t, tx.Commit(ctx))

		require.Equal(t, 7, rackQty(ctx, t, prodID, locID))
		require.Equal(t, 7, globalQty(ctx, t, prodID))
		require.Equal(t, 7, productsStock(ctx, t, prodID))
	})

	t.Run("clamps global at zero", func(t *testing.T) {
		// An over-set rack (50) with a count of 0 must not drive the global row
		// negative (CHECK constraint would wedge the posting session).
		prodID := insertTestProduct(ctx, t, "SAR-CLAMP")
		insertTestStock(ctx, t, prodID, 10)
		wh := 9703
		locID := insertTestRack(ctx, t, wh, "SAR-CLAMP-LOC")
		insertRackRow(ctx, t, prodID, wh, locID, 50)
		setProductsStock(ctx, t, prodID, 10)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, a.ReconcileLocationStock(ctx, tx, reconcile(prodID, locID, &wh, -50)))
		require.NoError(t, tx.Commit(ctx))

		require.Equal(t, 0, rackQty(ctx, t, prodID, locID))
		require.Equal(t, 0, globalQty(ctx, t, prodID))
		require.Equal(t, 0, productsStock(ctx, t, prodID))
	})

	t.Run("inserts global row when missing", func(t *testing.T) {
		// A rack-only product (no global row) gets its global row created from
		// the reconciled rack share: max(0-10, 0) + 7 = 7.
		prodID := insertTestProduct(ctx, t, "SAR-NOGLOBAL")
		wh := 9704
		locID := insertTestRack(ctx, t, wh, "SAR-NOGLOBAL-LOC")
		insertRackRow(ctx, t, prodID, wh, locID, 10)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, a.ReconcileLocationStock(ctx, tx, reconcile(prodID, locID, &wh, -3)))
		require.NoError(t, tx.Commit(ctx))

		require.Equal(t, 7, rackQty(ctx, t, prodID, locID))
		require.Equal(t, 7, globalQty(ctx, t, prodID))
		require.Equal(t, 7, productsStock(ctx, t, prodID))
	})

	t.Run("rollback undoes reconcile", func(t *testing.T) {
		prodID := insertTestProduct(ctx, t, "SAR-ROLLBACK")
		insertTestStock(ctx, t, prodID, 10)
		wh := 9705
		locID := insertTestRack(ctx, t, wh, "SAR-ROLLBACK-LOC")
		setProductsStock(ctx, t, prodID, 10)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, a.ReconcileLocationStock(ctx, tx, reconcile(prodID, locID, &wh, 4)))
		require.NoError(t, tx.Rollback(ctx))

		require.Equal(t, 10, globalQty(ctx, t, prodID))
		require.Equal(t, 10, productsStock(ctx, t, prodID))
	})
}
