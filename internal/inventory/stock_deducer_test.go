package inventory

import (
	"context"
	"sync"
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

	// P2-1 D2: duplicate line items for the same product must fail closed via
	// the atomic conditional decrement instead of subtracting twice and driving
	// stock negative. The deducer is the last line of defense; the sale module
	// aggregates duplicates before calling it, but the guard must hold regardless.
	t.Run("duplicate line items cannot oversell", func(t *testing.T) {
		prodID := insertTestProduct(ctx, t, "SD-DUP")
		insertTestStock(ctx, t, prodID, 8)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		err = d.DeductStock(ctx, tx, []shared.StockDeductItem{
			{ProductID: prodID, Quantity: 5},
			{ProductID: prodID, Quantity: 5},
		})
		require.ErrorIs(t, err, shared.ErrInsufficientStock)
		require.Equal(t, 8, stockQuantity(ctx, t, prodID), "stock must be untouched after abort")
	})

	// P2-1 D2: when the combined quantity fits, a duplicate-friendly deduction
	// still lands exactly once — the second conditional UPDATE sees the reduced
	// row and fails closed, so stock cannot drop below zero.
	t.Run("duplicate line items within capacity deduct total once", func(t *testing.T) {
		prodID := insertTestProduct(ctx, t, "SD-DUP2")
		insertTestStock(ctx, t, prodID, 10)

		tx, err := dbPool.Begin(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		err = d.DeductStock(ctx, tx, []shared.StockDeductItem{
			{ProductID: prodID, Quantity: 5},
			{ProductID: prodID, Quantity: 5},
		})
		require.NoError(t, err)
		require.NoError(t, tx.Commit(ctx))
		require.Equal(t, 0, stockQuantity(ctx, t, prodID), "10 requested from 10 available")
	})

	// P2-1 D2: two concurrent transactions must not both be able to deduct more
	// than the available stock; exactly one wins and the other aborts.
	t.Run("concurrent deductions serialize and one fails", func(t *testing.T) {
		prodID := insertTestProduct(ctx, t, "SD-CONC")
		insertTestStock(ctx, t, prodID, 10)

		const workers = 2
		start := make(chan struct{})
		errs := make([]error, workers)
		committed := make([]bool, workers)
		var wg sync.WaitGroup
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				tx, err := dbPool.Begin(ctx)
				if err != nil {
					errs[i] = err
					return
				}
				defer func() { _ = tx.Rollback(ctx) }()
				errs[i] = d.DeductStock(ctx, tx, []shared.StockDeductItem{{ProductID: prodID, Quantity: 6}})
				if errs[i] == nil {
					if cerr := tx.Commit(ctx); cerr != nil {
						errs[i] = cerr
						return
					}
					committed[i] = true
				}
			}(i)
		}
		close(start)
		wg.Wait()

		successCount := 0
		for i := 0; i < workers; i++ {
			if committed[i] {
				successCount++
				require.NoError(t, errs[i])
			} else {
				require.ErrorIs(t, errs[i], shared.ErrInsufficientStock)
			}
		}
		require.Equal(t, 1, successCount, "exactly one of two concurrent deductions succeeds")
		require.Equal(t, 4, stockQuantity(ctx, t, prodID), "10 - 6 = 4, never negative")
	})
}
