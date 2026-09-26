package product

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSellableStats_CountsActiveAndZeroStockProducts covers the catalogue half
// of the store onboarding readiness check: only active, non-deleted products
// count, and the zero-stock figure only counts products whose sellable bucket
// is empty. The catalogue is global (not store-scoped), so the assertion is a
// delta taken around the fixtures instead of an absolute total that other
// package tests may change.
func TestSellableStats_CountsActiveAndZeroStockProducts(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	provider := SellableStats{}

	beforeActive, beforeZero, err := provider.SellableStockStats(ctx, dbPool)
	require.NoError(t, err)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	insertProduct := func(tag, status string, softDeleted bool, quantity *int) {
		t.Helper()
		var id int
		require.NoError(t, dbPool.QueryRow(ctx,
			`INSERT INTO products (sku, name, price, cost, status)
			 VALUES ($1, $2, 1000, 500, $3) RETURNING id`,
			uniqueSKU("SELLSTAT_"+tag),
			"Sellable Stats "+tag+" "+suffix,
			status,
		).Scan(&id))
		if softDeleted {
			_, err := dbPool.Exec(ctx, `UPDATE products SET deleted_at = NOW() WHERE id = $1`, id)
			require.NoError(t, err)
		}
		if quantity != nil {
			_, err := dbPool.Exec(ctx,
				`INSERT INTO product_stock (product_id, quantity) VALUES ($1, $2)`, id, *quantity)
			require.NoError(t, err)
		}
	}

	stockedQty := 5
	insertProduct("ZEROSTOCK", "active", false, nil)       // counted, zero stock
	insertProduct("INSTOCK", "active", false, &stockedQty) // counted, stocked
	insertProduct("INACTIVE", "inactive", false, nil)
	insertProduct("DELETED", "active", true, nil)

	afterActive, afterZero, err := provider.SellableStockStats(ctx, dbPool)
	require.NoError(t, err)

	assert.Equal(t, beforeActive+2, afterActive,
		"only the two non-deleted active fixtures are new sellable products")
	assert.Equal(t, beforeZero+1, afterZero,
		"only the fixture without stock adds to the zero-stock count")
}
