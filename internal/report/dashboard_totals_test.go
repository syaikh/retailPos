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
	return insertParitySaleAt(ctx, t, cashierID, customerID, storeID, totalAmount, status, time.Time{})
}

// insertParitySaleAt is insertParitySale with an explicit created_at, used to
// place a sale in a different Jakarta day/hour bucket from a return. A zero
// createdAt falls back to the column default (NOW()).
func insertParitySaleAt(ctx context.Context, t *testing.T, cashierID, customerID int, storeID *int, totalAmount int, status string, createdAt time.Time) int {
	t.Helper()
	invoice := fmt.Sprintf("PARITY-%d-%d", time.Now().UnixNano(), parityCounter.Add(1))
	var storeArg any
	if storeID != nil {
		storeArg = *storeID
	}
	createdArg := any(nil)
	if !createdAt.IsZero() {
		createdArg = createdAt
	}
	var id int
	err := dbPool.QueryRow(ctx,
		`INSERT INTO sales (invoice_number, cashier_id, customer_id, store_id, subtotal, total_amount, payment_method, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, 'CASH', $7, COALESCE($8, NOW())) RETURNING id`,
		invoice, cashierID, customerID, storeArg, totalAmount, totalAmount, status, createdArg,
	).Scan(&id)
	require.NoError(t, err)
	return id
}

// insertParityReturn inserts a sale_return with an explicit status and
// total_refund_amount, reusing the given open shift (the schema allows only one
// open shift per user). storeID may be nil for a legacy no-store return.
func insertParityReturn(ctx context.Context, t *testing.T, cashierID, shiftID, saleID int, storeID *int, totalRefund int, status string) {
	t.Helper()
	insertParityReturnAt(ctx, t, cashierID, shiftID, saleID, storeID, totalRefund, status, time.Time{})
}

// insertParityReturnAt is insertParityReturn with an explicit created_at, used
// to place the refund in a different Jakarta day/hour bucket from its sale. A
// zero createdAt falls back to the column default (NOW()).
func insertParityReturnAt(ctx context.Context, t *testing.T, cashierID, shiftID, saleID int, storeID *int, totalRefund int, status string, createdAt time.Time) {
	t.Helper()
	var storeArg any
	if storeID != nil {
		storeArg = *storeID
	}
	createdArg := any(nil)
	if !createdAt.IsZero() {
		createdArg = createdAt
	}
	// return_number is varchar(30); keep it short.
	invoice := fmt.Sprintf("PR-%d-%d", time.Now().UnixNano()%1_000_000_000, parityCounter.Add(1))
	_, err := dbPool.Exec(ctx,
		`INSERT INTO sale_returns (return_number, original_sale_id, cashier_id, store_id,
		                           shift_id, return_type, reason, condition, status, total_refund_amount, created_at)
		 VALUES ($1, $2, $3, $4, $5, 'refund', 'damaged', 'damaged', $6, $7, COALESCE($8, NOW()))`,
		invoice, saleID, cashierID, storeArg, shiftID, status, totalRefund, createdArg)
	require.NoError(t, err)
}

// rawAllTimeNettedStats is the netted store-level raw query the parity test
// compares the materialized view against: gross completed-sale revenue minus
// completed refunds. It is deliberately a single-level store aggregate (no date
// key), which is why it cannot detect a dropped refund in the day/hour buckets
// — see TestRepository_DashboardTotals_ReturnOnlyBucket for that case.
func rawAllTimeNettedStats(ctx context.Context, t *testing.T, storeID *int) rawStats {
	t.Helper()
	storeFilter := ""
	args := []interface{}{}
	if storeID != nil {
		storeFilter = ` AND (store_id IS NULL OR store_id = $1)`
		args = append(args, *storeID)
	}
	var s rawStats
	require.NoError(t, dbPool.QueryRow(ctx,
		`SELECT COALESCE(SUM(s.total_amount), 0) - COALESCE((
		            SELECT SUM(total_refund_amount) FROM sale_returns
		            WHERE status = 'completed'`+storeFilter+`), 0),
		        COUNT(*)
		   FROM sales s
		  WHERE s.status = 'completed'`+storeFilter, args...).Scan(&s.revenue, &s.orders))
	return s
}

// TestRepository_DashboardTotals_NetRefunds proves the Phase 4.3 contract on
// both reporting paths: a completed refund return reduces the revenue the read
// model reports (mv_dashboard_totals and the live adapter), while a
// pending/cancelled return and a zero-amount replacement return do not, and
// the order count stays a gross sale count.
func TestRepository_DashboardTotals_NetRefunds(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, shared.TruncateTestData(dbPool))

	var storeA int
	require.NoError(t, dbPool.QueryRow(ctx, `INSERT INTO stores (name) VALUES ('Net Refund Store') RETURNING id`).Scan(&storeA))

	cashierID := insertParityCashier(ctx, t)
	customerID := insertParityCustomer(ctx, t)

	saleID := insertParitySale(ctx, t, cashierID, customerID, &storeA, 100_000, "completed")

	var shiftID int
	require.NoError(t, dbPool.QueryRow(ctx,
		`INSERT INTO shifts (user_id, status, opening_balance, opened_at)
		 VALUES ($1, 'open', 0, NOW()) RETURNING id`, cashierID).Scan(&shiftID))

	// Completed refund of 30_000: must be netted out.
	insertParityReturn(ctx, t, cashierID, shiftID, saleID, &storeA, 30_000, "completed")
	// A completed replacement pays nothing out (total_refund_amount = 0).
	insertParityReturn(ctx, t, cashierID, shiftID, saleID, &storeA, 0, "completed")
	// Neither pending nor cancelled money has left the drawer.
	insertParityReturn(ctx, t, cashierID, shiftID, saleID, &storeA, 999_999, "pending_approval")
	insertParityReturn(ctx, t, cashierID, shiftID, saleID, &storeA, 888_888, "cancelled")

	refreshMaterializedViews(ctx, t)

	repo := NewRepository(dbPool)

	gotRev, gotOrders, err := repo.getAllCompletedSalesStats(ctx, &storeA)
	require.NoError(t, err)
	assert.Equal(t, 70_000, gotRev, "mv_dashboard_totals nets only completed refunds")
	assert.Equal(t, 1, gotOrders, "order count stays a gross sale count")

	raw := rawAllTimeNettedStats(ctx, t, &storeA)
	assert.Equal(t, raw.revenue, gotRev, "view parity with the netted raw query")

	// Live dashboard path (SaleStatsProvider.GetCompletedSalesStats), same
	// netting, evaluated over the sale's window.
	wireReportAdapters(repo)
	liveRev, liveOrders, err := repo.saleStats.GetCompletedSalesStats(
		ctx, dbPool, time.Now().Add(-time.Hour), time.Now().Add(time.Hour), &storeA)
	require.NoError(t, err)
	assert.Equal(t, 70_000, liveRev, "live adapter nets only completed refunds")
	assert.Equal(t, 1, liveOrders, "live adapter order count stays gross")
}

// TestRepository_DailySales_ReturnOnlyBucket covers the case that the
// store-level parity query cannot: a completed refund processed on a Jakarta
// day that has no completed sale in the same store. A refund nets against its
// own created_at bucket, so without a FULL JOIN the day's row would not exist
// and the refund would be silently dropped from mv_daily_sales, overstating the
// daily chart while mv_dashboard_totals netted correctly. Covers the same drop
// for mv_hourly_sales.
func TestRepository_DailySales_ReturnOnlyBucket(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, shared.TruncateTestData(dbPool))

	var storeID int
	require.NoError(t, dbPool.QueryRow(ctx, `INSERT INTO stores (name) VALUES ('Return Only Store') RETURNING id`).Scan(&storeID))

	cashierID := insertParityCashier(ctx, t)
	customerID := insertParityCustomer(ctx, t)

	var shiftID int
	require.NoError(t, dbPool.QueryRow(ctx,
		`INSERT INTO shifts (user_id, status, opening_balance, opened_at)
		 VALUES ($1, 'open', 0, NOW()) RETURNING id`, cashierID).Scan(&shiftID))

	jakarta := shared.JakartaLocation()
	nowJakarta := time.Now().In(jakarta)
	// Sale and refund land in different Jakarta days: the refund is the first
	// activity of the later day, before the store has made any sale.
	saleDay := time.Date(nowJakarta.Year(), nowJakarta.Month(), nowJakarta.Day(), 10, 0, 0, 0, jakarta).AddDate(0, 0, -1)
	returnDay := time.Date(nowJakarta.Year(), nowJakarta.Month(), nowJakarta.Day(), 9, 0, 0, 0, jakarta)

	saleID := insertParitySaleAt(ctx, t, cashierID, customerID, &storeID, 100_000, "completed", saleDay)
	insertParityReturnAt(ctx, t, cashierID, shiftID, saleID, &storeID, 30_000, "completed", returnDay)

	refreshMaterializedViews(ctx, t)

	var dailyRevenue int
	var dailyOrders int
	require.NoError(t, dbPool.QueryRow(ctx,
		`SELECT total_revenue, transaction_count FROM mv_daily_sales
		  WHERE store_id = $1 AND sale_date = $2::date`,
		storeID, returnDay.Format("2006-01-02")).Scan(&dailyRevenue, &dailyOrders))
	assert.Equal(t, -30_000, dailyRevenue, "refund-only day keeps its own row with negative net revenue")
	assert.Equal(t, 0, dailyOrders, "a refund-only bucket contributes no sale count")

	// The prior day still carries the untouched sale.
	var priorRevenue int
	require.NoError(t, dbPool.QueryRow(ctx,
		`SELECT total_revenue FROM mv_daily_sales
		  WHERE store_id = $1 AND sale_date = $2::date`,
		storeID, saleDay.Format("2006-01-02")).Scan(&priorRevenue))
	assert.Equal(t, 100_000, priorRevenue, "sale day is unaffected by the later refund")

	// The hourly view must keep the refund's hour too, not drop it.
	var hourlyRows int
	require.NoError(t, dbPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM mv_hourly_sales
		  WHERE store_id = $1 AND date(sale_hour) = $2::date`,
		storeID, returnDay.Format("2006-01-02")).Scan(&hourlyRows))
	assert.Equal(t, 1, hourlyRows, "refund-only hour keeps its own row in mv_hourly_sales")

	// The store-level dashboard total nets across all time, independently of
	// the day bucketing, so it must agree with the sum of the daily rows.
	repo := NewRepository(dbPool)
	totalRev, _, err := repo.getAllCompletedSalesStats(ctx, &storeID)
	require.NoError(t, err)
	assert.Equal(t, 70_000, totalRev)

	var dailySum int
	require.NoError(t, dbPool.QueryRow(ctx,
		`SELECT COALESCE(SUM(total_revenue), 0) FROM mv_daily_sales WHERE store_id = $1`,
		storeID).Scan(&dailySum))
	assert.Equal(t, totalRev, dailySum, "daily rows must reconcile with the store-level total")
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
