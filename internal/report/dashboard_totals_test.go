package report

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

var parityCounter atomic.Int32

type rawStats struct {
	revenue int
	orders  int
}

func rawAllTimeStats(ctx context.Context, t *testing.T, storeID *int) rawStats {
	t.Helper()
	query := `SELECT COALESCE(SUM(total_amount), 0), COUNT(*) FROM sales WHERE status = 'completed'`
	args := []interface{}{}
	if storeID != nil {
		query += ` AND (store_id IS NULL OR store_id = $1)`
		args = append(args, *storeID)
	}
	var s rawStats
	err := dbPool.QueryRow(ctx, query, args...).Scan(&s.revenue, &s.orders)
	require.NoError(t, err)
	return s
}

func insertParityCashier(ctx context.Context, t *testing.T) int {
	t.Helper()
	username := fmt.Sprintf("parity_cashier_%d", time.Now().UnixNano())
	var id int
	err := dbPool.QueryRow(ctx,
		`INSERT INTO users (username, email, password_hash, role_id)
		 VALUES ($1, $2, 'hash', 1) RETURNING id`,
		username, username+"@test.com",
	).Scan(&id)
	require.NoError(t, err)
	return id
}

func insertParityCustomer(ctx context.Context, t *testing.T) int {
	t.Helper()
	phone := fmt.Sprintf("08%09d", time.Now().UnixNano()%1_000_000_000)
	var id int
	err := dbPool.QueryRow(ctx,
		`INSERT INTO customers (name, phone, email, is_walk_in, is_active, store_id)
		 VALUES ('Parity Customer', $1, $2, true, true, 1) RETURNING id`,
		phone, phone+"@test.com",
	).Scan(&id)
	require.NoError(t, err)
	return id
}

func insertParitySale(ctx context.Context, t *testing.T, cashierID, customerID int, storeID *int, totalAmount int, status string) int {
	t.Helper()
	invoice := fmt.Sprintf("PARITY-%d-%d", time.Now().UnixNano(), parityCounter.Add(1))
	var storeArg any
	if storeID != nil {
		storeArg = *storeID
	}
	var id int
	err := dbPool.QueryRow(ctx,
		`INSERT INTO sales (invoice_number, cashier_id, customer_id, store_id, subtotal, total_amount, payment_method, status)
		 VALUES ($1, $2, $3, $4, $5, $6, 'CASH', $7) RETURNING id`,
		invoice, cashierID, customerID, storeArg, totalAmount, totalAmount, status,
	).Scan(&id)
	require.NoError(t, err)
	return id
}

// TestRepository_GetAllCompletedSalesStats_MVParity pins the all-time
// dashboard totals to the mv_dashboard_totals read model and asserts they
// match the equivalent raw sales query (global and per store).
func TestRepository_GetAllCompletedSalesStats_MVParity(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, shared.TruncateTestData(dbPool))

	var storeA, storeB int
	require.NoError(t, dbPool.QueryRow(ctx, `INSERT INTO stores (name) VALUES ('Parity Store A') RETURNING id`).Scan(&storeA))
	require.NoError(t, dbPool.QueryRow(ctx, `INSERT INTO stores (name) VALUES ('Parity Store B') RETURNING id`).Scan(&storeB))

	cashierID := insertParityCashier(ctx, t)
	customerID := insertParityCustomer(ctx, t)

	// Completed sales across stores, including a legacy sale without a store.
	insertParitySale(ctx, t, cashierID, customerID, &storeA, 100_000, "completed")
	insertParitySale(ctx, t, cashierID, customerID, &storeB, 200_000, "completed")
	insertParitySale(ctx, t, cashierID, customerID, nil, 50_000, "completed")
	// Split-payment sale: two payment rows, one consolidated total.
	splitSaleID := insertParitySale(ctx, t, cashierID, customerID, &storeA, 300_000, "completed")
	// Non-completed sale must be excluded.
	insertParitySale(ctx, t, cashierID, customerID, &storeA, 999_999, "cancelled")

	var pmID int
	require.NoError(t, dbPool.QueryRow(ctx, `SELECT id FROM payment_methods WHERE is_active ORDER BY id LIMIT 1`).Scan(&pmID))
	for i := 0; i < 2; i++ {
		_, err := dbPool.Exec(ctx,
			`INSERT INTO sale_payments (sale_id, payment_method_id, payment_method_code, amount)
			 VALUES ($1, $2, 'CASH', $3)`,
			splitSaleID, pmID, 150_000)
		require.NoError(t, err)
	}

	refreshMaterializedViews(ctx, t)

	repo := NewRepository(dbPool)

	globRev, globOrders, err := repo.getAllCompletedSalesStats(ctx, nil)
	require.NoError(t, err)

	// Global: 4 completed sales across stores and the legacy no-store sale
	// (cancelled excluded, split payment counted once via its consolidated
	// total). Assert parity against the former raw query.
	rawGlobal := rawAllTimeStats(ctx, t, nil)
	assert.Equal(t, rawGlobal.revenue, globRev, "global revenue parity")
	assert.Equal(t, rawGlobal.orders, globOrders, "global orders parity")
	assert.Equal(t, 650_000, globRev, "global revenue total")
	assert.Equal(t, 4, globOrders, "global order count")

	rawStoreA := rawAllTimeStats(ctx, t, &storeA)
	gotRev, gotOrders, err := repo.getAllCompletedSalesStats(ctx, &storeA)
	require.NoError(t, err)
	assert.Equal(t, rawStoreA.revenue, gotRev, "store A revenue parity")
	assert.Equal(t, rawStoreA.orders, gotOrders, "store A orders parity")

	rawStoreB := rawAllTimeStats(ctx, t, &storeB)
	gotRev, gotOrders, err = repo.getAllCompletedSalesStats(ctx, &storeB)
	require.NoError(t, err)
	assert.Equal(t, rawStoreB.revenue, gotRev, "store B revenue parity")
	assert.Equal(t, rawStoreB.orders, gotOrders, "store B orders parity")
}
