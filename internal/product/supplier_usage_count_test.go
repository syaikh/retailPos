package product

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func productUsageSuffix() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func insertUsageProduct(ctx context.Context, t *testing.T, sku string) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(ctx, `
		INSERT INTO products (sku, name, price, cost, status)
		VALUES ($1, $1, 1000, 500, 'active')
		RETURNING id
	`, sku).Scan(&id)
	require.NoError(t, err)
	return id
}

func insertUsageSupplier(ctx context.Context, t *testing.T, code string) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(ctx, `
		INSERT INTO suppliers (name, code, is_active)
		VALUES ($1, $2, true)
		RETURNING id
	`, code, code).Scan(&id)
	require.NoError(t, err)
	return id
}

// TestSupplierLinkStore_CountLinksBySupplier pins the contract the supplier
// delete guard relies on: the count is store-blind. A link row carries a store
// (nil = shared global terms), but a link in ANY store is a live reference, so
// the count must not filter by store the way the read paths do.
func TestSupplierLinkStore_CountLinksBySupplier(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	ctx := context.Background()
	suffix := productUsageSuffix()
	storeA := insertTestStore(ctx, t, "Usage Store A "+suffix)
	storeB := insertTestStore(ctx, t, "Usage Store B "+suffix)
	supplierID := insertUsageSupplier(ctx, t, "SUP-LINKC-"+suffix)

	global := insertUsageProduct(ctx, t, "SKU-LINKC-G-"+suffix)
	storeScopedA := insertUsageProduct(ctx, t, "SKU-LINKC-A-"+suffix)
	storeScopedB := insertUsageProduct(ctx, t, "SKU-LINKC-B-"+suffix)

	_, err := dbPool.Exec(ctx, `
		INSERT INTO product_suppliers (product_id, supplier_id, store_id, unit_cost)
		VALUES ($1, $2, NULL, 1000), ($3, $2, $4, 1000), ($5, $2, $6, 1000)
	`, global, supplierID, storeScopedA, storeA, storeScopedB, storeB)
	require.NoError(t, err)

	count, err := (SupplierLinkStore{}).CountLinksBySupplier(ctx, dbPool, supplierID)
	require.NoError(t, err)
	assert.Equal(t, 3, count, "global and both store-scoped links must be counted")

	// A supplier with no links counts zero rather than erroring.
	empty, err := (SupplierLinkStore{}).CountLinksBySupplier(ctx, dbPool, -1)
	require.NoError(t, err)
	assert.Equal(t, 0, empty)
}
