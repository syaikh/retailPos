package product

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
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

	// The fixtures are inserted with raw SQL, so the product write paths did
	// not get a chance to drop the memoised snapshot.
	InvalidateSellableStats()

	afterActive, afterZero, err := provider.SellableStockStats(ctx, dbPool)
	require.NoError(t, err)

	assert.Equal(t, beforeActive+2, afterActive,
		"only the two non-deleted active fixtures are new sellable products")
	assert.Equal(t, beforeZero+1, afterZero,
		"only the fixture without stock adds to the zero-stock count")
}

// TestSellableStats_MemoisesTheGlobalAggregate pins the reason the snapshot
// exists: readiness is computed per visible store row, and the catalogue half
// is identical for every row, so the v_products_full scan must happen once per
// TTL window instead of once per row.
func TestSellableStats_MemoisesTheGlobalAggregate(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	InvalidateSellableStats()
	t.Cleanup(InvalidateSellableStats)

	query := "FROM v_products_full"
	mock.ExpectQuery(query).WillReturnRows(
		pgxmock.NewRows([]string{"count", "zero"}).AddRow(120, 7))
	mock.ExpectQuery(query).WillReturnRows(
		pgxmock.NewRows([]string{"count", "zero"}).AddRow(130, 9))

	provider := SellableStats{}

	active, zero, err := provider.SellableStockStats(context.Background(), mock)
	require.NoError(t, err)
	assert.Equal(t, 120, active)
	assert.Equal(t, 7, zero)

	// Second call must be served from the snapshot: no extra expectation is
	// registered, so a query here would fail the mock.
	active, zero, err = provider.SellableStockStats(context.Background(), mock)
	require.NoError(t, err)
	assert.Equal(t, 120, active, "cached read must not observe a later catalogue")
	assert.Equal(t, 7, zero)

	InvalidateSellableStats()
	active, zero, err = provider.SellableStockStats(context.Background(), mock)
	require.NoError(t, err)
	assert.Equal(t, 130, active, "a product write must make the next read fresh")
	assert.Equal(t, 9, zero)

	assert.NoError(t, mock.ExpectationsWereMet())
}
