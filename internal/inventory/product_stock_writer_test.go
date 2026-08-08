package inventory

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

func pswSKU(base string) string {
	return base + "-" + time.Now().Format("20060102150405") + "-" + fmt.Sprintf("%09d", time.Now().Nanosecond())
}

func TestProductStockWriter_SetStoreStock(t *testing.T) {
	ctx := context.Background()
	productID := insertTestProduct(ctx, t, pswSKU("PSW"))

	var storeID int
	require.NoError(t, dbPool.QueryRow(ctx, `INSERT INTO stores (name) VALUES ($1) RETURNING id`, pswSKU("PSW-Store")).Scan(&storeID))

	tx, err := dbPool.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	w := ProductStockWriter{}

	// global row (nil store): first write inserts, second write updates in place
	require.NoError(t, w.SetStoreStock(ctx, tx, shared.StockRowSet{ProductID: productID, Quantity: 100}))
	require.NoError(t, w.SetStoreStock(ctx, tx, shared.StockRowSet{ProductID: productID, Quantity: 75}))

	// store-scoped row: first write inserts, second write upserts
	require.NoError(t, w.SetStoreStock(ctx, tx, shared.StockRowSet{ProductID: productID, StoreID: &storeID, Quantity: 25}))
	require.NoError(t, w.SetStoreStock(ctx, tx, shared.StockRowSet{ProductID: productID, StoreID: &storeID, Quantity: 40}))

	require.NoError(t, tx.Commit(ctx))

	var globalQty int
	require.NoError(t, dbPool.QueryRow(ctx,
		`SELECT quantity FROM product_stock
		 WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL AND location_id IS NULL`,
		productID).Scan(&globalQty))
	require.Equal(t, 75, globalQty)

	var storeQty int
	require.NoError(t, dbPool.QueryRow(ctx,
		`SELECT quantity FROM product_stock WHERE product_id = $1 AND store_id = $2`,
		productID, storeID).Scan(&storeQty))
	require.Equal(t, 40, storeQty)
}

func TestProductStockWriter_SetStoreStockBatch(t *testing.T) {
	ctx := context.Background()
	id1 := insertTestProduct(ctx, t, pswSKU("PSW-B1"))
	id2 := insertTestProduct(ctx, t, pswSKU("PSW-B2"))

	var storeID int
	require.NoError(t, dbPool.QueryRow(ctx, `INSERT INTO stores (name) VALUES ($1) RETURNING id`, pswSKU("PSW-BatchStore")).Scan(&storeID))

	tx, err := dbPool.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	err = ProductStockWriter{}.SetStoreStockBatch(ctx, tx, []shared.StockRowSet{
		{ProductID: id1, Quantity: 10},
		{ProductID: id2, Quantity: 20},
		{ProductID: id1, StoreID: &storeID, Quantity: 5},
	})
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))

	var qty int
	require.NoError(t, dbPool.QueryRow(ctx,
		`SELECT quantity FROM product_stock WHERE product_id = $1 AND store_id IS NULL`,
		id1).Scan(&qty))
	require.Equal(t, 10, qty)
	require.NoError(t, dbPool.QueryRow(ctx,
		`SELECT quantity FROM product_stock WHERE product_id = $1 AND store_id = $2`,
		id1, storeID).Scan(&qty))
	require.Equal(t, 5, qty)
	require.NoError(t, dbPool.QueryRow(ctx,
		`SELECT quantity FROM product_stock WHERE product_id = $1 AND store_id IS NULL`,
		id2).Scan(&qty))
	require.Equal(t, 20, qty)
}
