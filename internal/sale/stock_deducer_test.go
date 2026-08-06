package sale

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/inventory"
	"retail-pos-system/internal/shared"
)

// paritySKUCounter keeps SKUs unique within a test run (tests run sequentially).
var paritySKUCounter int

// insertParityProduct inserts a product, optionally with a default stock row.
// A stock of -1 produces a product with no product_stock row (record-not-found).
func insertParityProduct(ctx context.Context, t *testing.T, impl, name string, stock int) int {
	t.Helper()
	paritySKUCounter++
	sku := fmt.Sprintf("PAR-%s-%d", impl, paritySKUCounter)
	var prodID int
	if stock >= 0 {
		prodID = insertTestProduct(ctx, t, sku, "Parity "+name, 10000, stock)
	} else {
		err := dbPool.QueryRow(ctx,
			`INSERT INTO products (sku, name, price, cost, status) VALUES ($1, $2, 10000, 5000, 'active') RETURNING id`,
			sku, "Parity "+name,
		).Scan(&prodID)
		require.NoError(t, err)
	}
	return prodID
}

// TestStockDeducer_Parity guards against drift between the sale-local default
// StockDeducer (exercised by unit tests) and the canonical inventory
// implementation (used in production via wiring). Both must produce identical
// outcomes for the same scenarios, otherwise tests would validate behavior that
// differs from production.
func TestStockDeducer_Parity(t *testing.T) {
	ctx := context.Background()
	impls := map[string]StockDeducer{
		"default":   stockDeducer{},
		"inventory": inventory.StockDeducer{},
	}

	cases := []struct {
		name       string
		stock      int // -1 = no product_stock row
		deduct     int
		wantErrSub string
	}{
		{name: "success", stock: 10, deduct: 3},
		{name: "insufficient", stock: 2, deduct: 5, wantErrSub: "insufficient stock"},
		{name: "record-not-found", stock: -1, deduct: 1, wantErrSub: "stock record not found"},
	}

	for implName, impl := range impls {
		impl := impl
		for _, tc := range cases {
			tc := tc
			t.Run(implName+"/"+tc.name, func(t *testing.T) {
				prodID := insertParityProduct(ctx, t, implName, tc.name, tc.stock)

				tx, err := dbPool.Begin(ctx)
				require.NoError(t, err)
				defer func() { _ = tx.Rollback(ctx) }()

				err = impl.DeductStock(ctx, tx, []shared.StockDeductItem{{ProductID: prodID, Quantity: tc.deduct}})
				if tc.wantErrSub == "" {
					require.NoError(t, err)
					var qty int
					require.NoError(t, tx.QueryRow(ctx, `SELECT quantity FROM product_stock WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL`, prodID).Scan(&qty))
					require.Equal(t, tc.stock-tc.deduct, qty, "deducted quantity must match across implementations")
				} else {
					require.ErrorContains(t, err, tc.wantErrSub)
				}
			})
		}
	}
}
