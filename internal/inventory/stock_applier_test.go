package inventory

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

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

// TestStockApplier_SetProductStock covers the canonical single-writer of
// product_stock (ADR_Modular_Monolith_Module_Boundaries §2.8) that the stock
// opname module uses when posting a non-location adjustment. It must run against
// the caller's transaction.
func TestStockApplier_SetProductStock(t *testing.T) {
	ctx := context.Background()
	a := StockApplier{}

	t.Run("upserts existing global row", func(t *testing.T) {
		prodID := insertTestProduct(ctx, t, "SAS-UPSERT")
		insertTestStock(ctx, t, prodID, 10)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, a.SetProductStock(ctx, tx, shared.StockSetItem{ProductID: prodID, Quantity: 12}))
		require.NoError(t, tx.Commit(ctx))

		require.Equal(t, 12, stockQuantity(ctx, t, prodID))
	})

	t.Run("inserts global row lazily", func(t *testing.T) {
		prodID := insertTestProduct(ctx, t, "SAS-INSERT")

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, a.SetProductStock(ctx, tx, shared.StockSetItem{ProductID: prodID, Quantity: 7}))
		require.NoError(t, tx.Commit(ctx))

		require.Equal(t, 7, stockQuantity(ctx, t, prodID))
	})

	t.Run("rollback undoes set", func(t *testing.T) {
		prodID := insertTestProduct(ctx, t, "SAS-ROLLBACK")
		insertTestStock(ctx, t, prodID, 10)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, a.SetProductStock(ctx, tx, shared.StockSetItem{ProductID: prodID, Quantity: 5}))
		require.NoError(t, tx.Rollback(ctx))

		require.Equal(t, 10, stockQuantity(ctx, t, prodID))
	})
}

// TestStockApplier_ReconcileLocationStock covers the location-scoped reconcile
// (rack delta applied + global recomputed from the rack share) used by
// location-scoped stock opname posting.
func TestStockApplier_ReconcileLocationStock(t *testing.T) {
	ctx := context.Background()
	a := StockApplier{}
	reconcile := func(prodID, locID int, wh *int, delta int) shared.LocationStockReconcile {
		return shared.LocationStockReconcile{ProductID: prodID, LocationID: locID, WarehouseID: wh, Delta: delta}
	}

	t.Run("creates rows lazily", func(t *testing.T) {
		prodID := insertTestProduct(ctx, t, "SAR-LAZY")
		insertTestStock(ctx, t, prodID, 10)
		wh := 9701
		locID := insertTestRack(ctx, t, wh, "SAR-LAZY-LOC")

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, a.ReconcileLocationStock(ctx, tx, reconcile(prodID, locID, &wh, 3)))
		require.NoError(t, tx.Commit(ctx))

		require.Equal(t, 3, rackQty(ctx, t, prodID, locID))
		require.Equal(t, 13, globalQty(ctx, t, prodID))
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

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, a.ReconcileLocationStock(ctx, tx, reconcile(prodID, locID, &wh, -3)))
		require.NoError(t, tx.Commit(ctx))

		require.Equal(t, 7, rackQty(ctx, t, prodID, locID))
		require.Equal(t, 7, globalQty(ctx, t, prodID))
	})

	t.Run("clamps global at zero", func(t *testing.T) {
		// An over-set rack (50) with a count of 0 must not drive the global row
		// negative (CHECK constraint would wedge the posting session).
		prodID := insertTestProduct(ctx, t, "SAR-CLAMP")
		insertTestStock(ctx, t, prodID, 10)
		wh := 9703
		locID := insertTestRack(ctx, t, wh, "SAR-CLAMP-LOC")
		insertRackRow(ctx, t, prodID, wh, locID, 50)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, a.ReconcileLocationStock(ctx, tx, reconcile(prodID, locID, &wh, -50)))
		require.NoError(t, tx.Commit(ctx))

		require.Equal(t, 0, rackQty(ctx, t, prodID, locID))
		require.Equal(t, 0, globalQty(ctx, t, prodID))
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
	})

	t.Run("rollback undoes reconcile", func(t *testing.T) {
		prodID := insertTestProduct(ctx, t, "SAR-ROLLBACK")
		insertTestStock(ctx, t, prodID, 10)
		wh := 9705
		locID := insertTestRack(ctx, t, wh, "SAR-ROLLBACK-LOC")

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, a.ReconcileLocationStock(ctx, tx, reconcile(prodID, locID, &wh, 4)))
		require.NoError(t, tx.Rollback(ctx))

		require.Equal(t, 10, globalQty(ctx, t, prodID))
	})
}

// TestStockApplier_SetProductStocks_Batch covers the batched global-stock
// writer that stock opname posting now uses (backend review P3): many products
// in one pair of statements, mixed existing/missing global rows, deterministic
// last-wins on a duplicated product, and an empty batch as a no-op.
func TestStockApplier_SetProductStocks_Batch(t *testing.T) {
	ctx := context.Background()
	a := StockApplier{}

	t.Run("updates existing and inserts missing rows in one batch", func(t *testing.T) {
		existing := insertTestProduct(ctx, t, "SASB-EXIST")
		missing := insertTestProduct(ctx, t, "SASB-MISS")
		insertTestStock(ctx, t, existing, 10)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, a.SetProductStocks(ctx, tx, []shared.StockSetItem{
			{ProductID: existing, Quantity: 12},
			{ProductID: missing, Quantity: 7},
		}))
		require.NoError(t, tx.Commit(ctx))

		require.Equal(t, 12, stockQuantity(ctx, t, existing))
		require.Equal(t, 7, stockQuantity(ctx, t, missing))
	})

	t.Run("duplicate product keeps the last value", func(t *testing.T) {
		prodID := insertTestProduct(ctx, t, "SASB-DUP")
		insertTestStock(ctx, t, prodID, 10)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, a.SetProductStocks(ctx, tx, []shared.StockSetItem{
			{ProductID: prodID, Quantity: 3},
			{ProductID: prodID, Quantity: 9},
		}))
		require.NoError(t, tx.Commit(ctx))

		require.Equal(t, 9, stockQuantity(ctx, t, prodID))
	})

	t.Run("empty batch is a no-op", func(t *testing.T) {
		prodID := insertTestProduct(ctx, t, "SASB-EMPTY")
		insertTestStock(ctx, t, prodID, 4)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, a.SetProductStocks(ctx, tx, nil))
		require.NoError(t, tx.Commit(ctx))

		require.Equal(t, 4, stockQuantity(ctx, t, prodID))
	})
}

// TestStockApplier_ReconcileLocationStocks_Batch covers the batched reconcile
// that location-scoped stock opname posting now uses (backend review P3): many
// products in one pass, the update path (existing rack row) and the insert path
// (missing rack row) inside the same batch, grouping across locations, and an
// empty batch as a no-op. The per-product arithmetic must match the
// single-product path exactly.
func TestStockApplier_ReconcileLocationStocks_Batch(t *testing.T) {
	ctx := context.Background()
	a := StockApplier{}
	reconcile := func(prodID, locID int, wh *int, delta int) shared.LocationStockReconcile {
		return shared.LocationStockReconcile{ProductID: prodID, LocationID: locID, WarehouseID: wh, Delta: delta}
	}

	t.Run("mixed existing and missing rack rows in one batch", func(t *testing.T) {
		// Product A has a rack row (10) and a desynced global (7): delta -3 must
		// reconcile the global to max(7-10, 0) + 7 = 7, not 4. Product B has no
		// rack row: its rack row is created and the global grows by the delta.
		prodA := insertTestProduct(ctx, t, "SARB-MIX-A")
		insertTestStock(ctx, t, prodA, 7)
		prodB := insertTestProduct(ctx, t, "SARB-MIX-B")
		insertTestStock(ctx, t, prodB, 5)
		wh := 9711
		locID := insertTestRack(ctx, t, wh, "SARB-MIX-LOC")
		insertRackRow(ctx, t, prodA, wh, locID, 10)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, a.ReconcileLocationStocks(ctx, tx, []shared.LocationStockReconcile{
			reconcile(prodA, locID, &wh, -3),
			reconcile(prodB, locID, &wh, 4),
		}))
		require.NoError(t, tx.Commit(ctx))

		require.Equal(t, 7, rackQty(ctx, t, prodA, locID))
		require.Equal(t, 7, globalQty(ctx, t, prodA))
		require.Equal(t, 4, rackQty(ctx, t, prodB, locID))
		require.Equal(t, 9, globalQty(ctx, t, prodB))
	})

	t.Run("groups products across locations", func(t *testing.T) {
		prodA := insertTestProduct(ctx, t, "SARB-LOC-A")
		insertTestStock(ctx, t, prodA, 5)
		prodB := insertTestProduct(ctx, t, "SARB-LOC-B")
		insertTestStock(ctx, t, prodB, 5)
		wh := 9712
		loc1 := insertTestRack(ctx, t, wh, "SARB-LOC-1")
		loc2 := insertTestRack(ctx, t, wh, "SARB-LOC-2")

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, a.ReconcileLocationStocks(ctx, tx, []shared.LocationStockReconcile{
			reconcile(prodA, loc1, &wh, 2),
			reconcile(prodB, loc2, &wh, 3),
		}))
		require.NoError(t, tx.Commit(ctx))

		require.Equal(t, 2, rackQty(ctx, t, prodA, loc1))
		require.Equal(t, 7, globalQty(ctx, t, prodA))
		require.Equal(t, 3, rackQty(ctx, t, prodB, loc2))
		require.Equal(t, 8, globalQty(ctx, t, prodB))
	})

	t.Run("empty batch is a no-op", func(t *testing.T) {
		prodID := insertTestProduct(ctx, t, "SARB-EMPTY")
		insertTestStock(ctx, t, prodID, 6)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, a.ReconcileLocationStocks(ctx, tx, nil))
		require.NoError(t, tx.Commit(ctx))

		require.Equal(t, 6, globalQty(ctx, t, prodID))
	})
}

// TestStockApplier_ReconcileLocationStocks_DuplicateProducts pins the
// duplicate-product handling added after the backend review: repeated products
// are merged deterministically instead of erroring or applying an arbitrary
// value. Same-location duplicates sum their deltas (equivalent to sequential
// application); a product repeated across locations carries its computed
// global forward so the single global upsert row matches applying the
// single-product formula location group by location group.
func TestStockApplier_ReconcileLocationStocks_DuplicateProducts(t *testing.T) {
	ctx := context.Background()
	a := StockApplier{}
	reconcile := func(prodID, locID int, wh *int, delta int) shared.LocationStockReconcile {
		return shared.LocationStockReconcile{ProductID: prodID, LocationID: locID, WarehouseID: wh, Delta: delta}
	}

	t.Run("same product twice at one location sums the deltas", func(t *testing.T) {
		prodID := insertTestProduct(ctx, t, "SARD-SAMELOC")
		wh := 9721
		locID := insertTestRack(ctx, t, wh, "SARD-SAMELOC-LOC")
		insertRackRow(ctx, t, prodID, wh, locID, 10)
		insertTestStock(ctx, t, prodID, 10)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, a.ReconcileLocationStocks(ctx, tx, []shared.LocationStockReconcile{
			reconcile(prodID, locID, &wh, 2),
			reconcile(prodID, locID, &wh, 3),
		}))
		require.NoError(t, tx.Commit(ctx))

		// rack 10 + (2+3) = 15; global = max(10-10, 0) + 15 = 15.
		require.Equal(t, 15, rackQty(ctx, t, prodID, locID))
		require.Equal(t, 15, globalQty(ctx, t, prodID))
	})

	t.Run("same product at two locations carries the global forward", func(t *testing.T) {
		prodID := insertTestProduct(ctx, t, "SARD-TWOLOC")
		wh := 9722
		loc1 := insertTestRack(ctx, t, wh, "SARD-TWOLOC-1")
		loc2 := insertTestRack(ctx, t, wh, "SARD-TWOLOC-2")
		insertRackRow(ctx, t, prodID, wh, loc1, 3)
		insertRackRow(ctx, t, prodID, wh, loc2, 4)
		insertTestStock(ctx, t, prodID, 5)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		require.NoError(t, a.ReconcileLocationStocks(ctx, tx, []shared.LocationStockReconcile{
			reconcile(prodID, loc1, &wh, 1),
			reconcile(prodID, loc2, &wh, 2),
		}))
		require.NoError(t, tx.Commit(ctx))

		// Sequential application: loc1 -> rack 4, global max(5-3,0)+4 = 6;
		// loc2 -> rack 6, global max(6-4,0)+6 = 8. The upsert must contain a
		// single row (the final carried-forward value), not two.
		require.Equal(t, 4, rackQty(ctx, t, prodID, loc1))
		require.Equal(t, 6, rackQty(ctx, t, prodID, loc2))
		require.Equal(t, 8, globalQty(ctx, t, prodID))

		var globalRows int
		require.NoError(t, dbPool.QueryRow(ctx,
			`SELECT COUNT(*) FROM product_stock
			 WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL AND location_id IS NULL`,
			prodID).Scan(&globalRows))
		require.Equal(t, 1, globalRows, "must not create a duplicate global row")
	})
}
