package report

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/events"
	"retail-pos-system/internal/shared"
	"retail-pos-system/pkg/cache"
)

var dbPool *pgxpool.Pool

var productCounter atomic.Int32

func refreshMaterializedViews(t *testing.T, ctx context.Context) {
	t.Helper()
	_, err := dbPool.Exec(ctx, "SELECT refresh_sales_mv()")
	require.NoError(t, err)
}

func uniqueSKU(prefix string) string {
	n := productCounter.Add(1)
	return fmt.Sprintf("%s-RPT-%d-%d", prefix, time.Now().UnixNano(), n)
}

func seedSale(t *testing.T, ctx context.Context) (saleID int, productID int, saleAmount int, saleQty int) {
	t.Helper()
	userSKU := uniqueSKU("USR")
	var cashierID int
	err := dbPool.QueryRow(ctx,
		`INSERT INTO users (username, email, password_hash, role_id) VALUES ($1, $2, 'hash', 1) RETURNING id`,
		userSKU, userSKU+"@test.com",
	).Scan(&cashierID)
	require.NoError(t, err)

	custSKU := uniqueSKU("CUST")
	var customerID int
	err = dbPool.QueryRow(ctx,
		`INSERT INTO customers (name, phone, email, is_walk_in, is_active) VALUES ($1, $2, $3, true, true) RETURNING id`,
		"Test "+custSKU, fmt.Sprintf("%010d", productCounter.Add(1)), custSKU+"@test.com",
	).Scan(&customerID)
	require.NoError(t, err)

	sku := uniqueSKU("RPT-SEED")
	productName := "Test Seed Product"
	price := 50000
	qty := 2
	total := price * qty

	// Insert product
	err = dbPool.QueryRow(ctx,
		`INSERT INTO products (sku, name, price, stock, status) VALUES ($1, $2, $3, $4, 'active') RETURNING id`,
		sku, productName, price, qty,
	).Scan(&productID)
	require.NoError(t, err)

	// Insert product_stock
	_, err = dbPool.Exec(ctx,
		`INSERT INTO product_stock (product_id, quantity) VALUES ($1, $2)`,
		productID, qty,
	)
	require.NoError(t, err)

	// Insert sale
	invSKU := uniqueSKU("INV")
	err = dbPool.QueryRow(ctx,
		`INSERT INTO sales (invoice_number, cashier_id, customer_id, subtotal, total_amount, payment_method, status)
		 VALUES ($1, $2, $3, $4, $5, 'CASH', 'completed') RETURNING id`,
		invSKU, cashierID, customerID, total, total,
	).Scan(&saleID)
	require.NoError(t, err)

	// Insert sale_item
	_, err = dbPool.Exec(ctx,
		`INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal, dpp_amount, tax_amount)
		 VALUES ($1, $2, $3, $4, $5, $5, 0)`,
		saleID, productID, qty, price, total,
	)
	require.NoError(t, err)

	return saleID, productID, total, qty
}

func TestMain(m *testing.M) {
	pool, err := shared.NewTestDB()
	if err != nil {
		os.Exit(1)
	}
	dbPool = pool
	defer pool.Close()

	if err := shared.RunMigrations(pool, "../../database/migrations"); err != nil {
		os.Exit(1)
	}

	if err := shared.TruncateTestData(pool); err != nil {
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestReportRepository_PeriodComparison_SeededData(t *testing.T) {
	require.NoError(t, shared.TruncateTestData(dbPool))
	repo := NewRepository(dbPool)
	ctx := context.Background()

	// Seed a sale in the current period first
	_, _, amount, _ := seedSale(t, ctx)

	now := time.Now()
	start := now.AddDate(0, -1, 0)
	prevStart := start.AddDate(0, -1, 0)

	result, err := repo.GetPeriodComparison(ctx, start, now, prevStart, start, nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.GreaterOrEqual(t, result.CurrentRevenue, amount, "should include seeded sale amount")
	assert.GreaterOrEqual(t, result.CurrentOrders, 1, "should have at least 1 sale in current period")
	assert.Equal(t, 0, result.PreviousRevenue, "no sales in previous period")
	assert.Equal(t, 0, result.PreviousOrders, "no sales in previous period")
	assert.False(t, result.PreviousHasAnyData, "no data in previous period")
}

func TestReportRepository_PeriodComparison_PreviousHasAnyData(t *testing.T) {
	require.NoError(t, shared.TruncateTestData(dbPool))
	repo := NewRepository(dbPool)
	ctx := context.Background()

	now := time.Now()
	start := now.AddDate(0, -1, 0)
	prevStart := start.AddDate(0, -1, 0)

	userSKU := uniqueSKU("PHD")
	var cashierID int
	err := dbPool.QueryRow(ctx,
		`INSERT INTO users (username, email, password_hash, role_id) VALUES ($1, $2, 'hash', 1) RETURNING id`,
		userSKU, userSKU+"@test.com",
	).Scan(&cashierID)
	require.NoError(t, err)

	custSKU := uniqueSKU("PHD")
	var customerID int
	err = dbPool.QueryRow(ctx,
		`INSERT INTO customers (name, phone, email, is_walk_in, is_active) VALUES ($1, $2, $3, true, true) RETURNING id`,
		"Test "+custSKU, fmt.Sprintf("%010d", productCounter.Add(1)), custSKU+"@test.com",
	).Scan(&customerID)
	require.NoError(t, err)

	sku := uniqueSKU("PHD-PROD")
	var productID int
	err = dbPool.QueryRow(ctx,
		`INSERT INTO products (sku, name, price, stock, status) VALUES ($1, $2, 50000, 10, 'active') RETURNING id`,
		sku, "Prev Period Product",
	).Scan(&productID)
	require.NoError(t, err)

	_, err = dbPool.Exec(ctx, `INSERT INTO product_stock (product_id, quantity) VALUES ($1, 10)`, productID)
	require.NoError(t, err)

	invSKU := uniqueSKU("PHD-INV")
	prevSaleDate := prevStart.Add(2 * time.Hour)
	var saleID int
	err = dbPool.QueryRow(ctx,
		`INSERT INTO sales (invoice_number, cashier_id, customer_id, subtotal, total_amount, payment_method, status, created_at)
		 VALUES ($1, $2, $3, 50000, 50000, 'CASH', 'completed', $4) RETURNING id`,
		invSKU, cashierID, customerID, prevSaleDate,
	).Scan(&saleID)
	require.NoError(t, err)

	_, err = dbPool.Exec(ctx,
		`INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal, dpp_amount, tax_amount)
		 VALUES ($1, $2, 1, 50000, 50000, 50000, 0)`,
		saleID, productID,
	)
	require.NoError(t, err)

	result, err := repo.GetPeriodComparison(ctx, start, now, prevStart, start, nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.PreviousHasAnyData, "should detect sales in previous period first 24h")
	assert.Greater(t, result.PreviousRevenue, 0)
}

func TestReportRepository_DualChartData_Seeded(t *testing.T) {
	require.NoError(t, shared.TruncateTestData(dbPool))
	repo := NewRepository(dbPool)
	ctx := context.Background()

	_, _, amount, _ := seedSale(t, ctx)
	refreshMaterializedViews(t, ctx)

	now := time.Now()
	currentStart := now.AddDate(0, 0, -7)
	currentEnd := now
	prevStart := currentStart.AddDate(0, 0, -7)
	prevEnd := currentEnd.AddDate(0, 0, -7)

	current, previous, err := repo.GetDualChartData(ctx, currentStart, currentEnd, prevStart, prevEnd, nil)
	require.NoError(t, err)
	require.NotNil(t, current)
	assert.NotNil(t, previous)

	// Verify the seeded sale appears in the current period
	found := false
	for _, dp := range current {
		if dp.Total >= amount {
			found = true
			break
		}
	}
	assert.True(t, found, "seeded sale amount should appear in current chart data")
}

func TestReportRepository_LiveDashboardStats_Seeded(t *testing.T) {
	require.NoError(t, shared.TruncateTestData(dbPool))
	repo := NewRepository(dbPool)
	ctx := context.Background()

	beforeRev, _, _, _, err := repo.GetLiveDashboardStats(ctx, nil)
	require.NoError(t, err)

	// Seed a sale
	_, _, amount, _ := seedSale(t, ctx)

	afterRev, sales, products, lowStock, err := repo.GetLiveDashboardStats(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, beforeRev+amount, afterRev, "revenue should increase by seeded sale amount")
	assert.GreaterOrEqual(t, sales, 1)
	assert.GreaterOrEqual(t, products, 1)
	assert.GreaterOrEqual(t, lowStock, 0)
}

func TestReportRepository_AvailableYears(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	years, err := repo.GetAvailableYears(ctx, nil)
	require.NoError(t, err)
	assert.NotNil(t, years)
}

func TestReportRepository_HourlySales_Seeded(t *testing.T) {
	require.NoError(t, shared.TruncateTestData(dbPool))
	repo := NewRepository(dbPool)
	ctx := context.Background()

	_, _, amount, _ := seedSale(t, ctx)
	refreshMaterializedViews(t, ctx)
	date := time.Now()

	result, err := repo.GetHourlySales(ctx, date, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	found := false
	for _, dp := range result {
		if dp.Total >= amount {
			found = true
			break
		}
	}
	assert.True(t, found, "seeded sale should appear in hourly sales data")
}

func TestReportRepository_DailySales_Seeded(t *testing.T) {
	require.NoError(t, shared.TruncateTestData(dbPool))
	repo := NewRepository(dbPool)
	ctx := context.Background()

	_, _, amount, _ := seedSale(t, ctx)
	refreshMaterializedViews(t, ctx)
	start := time.Now().AddDate(0, -1, 0)
	end := time.Now().Add(24 * time.Hour)

	result, err := repo.GetDailySales(ctx, start, end, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	found := false
	for _, dp := range result {
		if dp.Total >= amount {
			found = true
			break
		}
	}
	assert.True(t, found, "seeded sale should appear in daily sales data")
}

func TestReportRepository_SalesWeeklyReport_Seeded(t *testing.T) {
	require.NoError(t, shared.TruncateTestData(dbPool))
	repo := NewRepository(dbPool)
	ctx := context.Background()

	_, _, amount, _ := seedSale(t, ctx)
	start := time.Now().AddDate(0, -3, 0)
	end := time.Now()

	result, err := repo.GetSalesWeeklyReport(ctx, start, end, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	found := false
	for _, item := range result {
		if item.Total >= amount {
			found = true
			break
		}
	}
	assert.True(t, found, "seeded sale should appear in weekly report")
}

func TestReportRepository_SalesMonthlyReport_Seeded(t *testing.T) {
	require.NoError(t, shared.TruncateTestData(dbPool))
	repo := NewRepository(dbPool)
	ctx := context.Background()

	_, _, amount, _ := seedSale(t, ctx)
	start := time.Now().AddDate(0, -6, 0)
	end := time.Now()

	result, err := repo.GetSalesMonthlyReport(ctx, start, end, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	found := false
	for _, item := range result {
		if item.Total >= amount {
			found = true
			break
		}
	}
	assert.True(t, found, "seeded sale should appear in monthly report")
}

func TestReportRepository_SetCache(t *testing.T) {
	repo := NewRepository(dbPool)

	c := cache.New(5*time.Minute, 1*time.Minute)
	repo.SetCache(c)
	c.Set("test-key", "test-value")
	c.Wait()
	val, ok := repo.cache.Get("test-key")
	assert.True(t, ok)
	assert.Equal(t, "test-value", val)
}

func TestReportRepository_InvalidateDashboardCache(t *testing.T) {
	repo := NewRepository(dbPool)

	c := cache.New(5*time.Minute, 1*time.Minute)
	repo.SetCache(c)
	c.Set("dashboard:stats", "stale")
	c.Set("dashboard:live", "stale")
	c.Set("dashboard:stats:store:1", "stale")
	c.Set("report:some_key", "report-stale")
	c.Set("report:another", "report-stale")
	c.Wait()

	repo.InvalidateDashboardCache(nil)
	c.Wait()

	_, ok1 := repo.cache.Get("dashboard:stats")
	assert.False(t, ok1)
	_, ok2 := repo.cache.Get("dashboard:live")
	assert.False(t, ok2)
	_, ok3 := repo.cache.Get("dashboard:stats:store:1")
	assert.True(t, ok3)
	_, ok4 := repo.cache.Get("report:some_key")
	assert.False(t, ok4, "report: prefixed keys should also be flushed")
	_, ok5 := repo.cache.Get("report:another")
	assert.False(t, ok5, "report: prefixed keys should also be flushed")
}

func TestReportRepository_InvalidateDashboardCache_WithStoreID(t *testing.T) {
	repo := NewRepository(dbPool)

	c := cache.New(5*time.Minute, 1*time.Minute)
	repo.SetCache(c)
	c.Set("dashboard:stats", "stale")
	c.Set("dashboard:stats:store:5", "stale")

	sid := 5
	repo.InvalidateDashboardCache(&sid)

	_, ok1 := repo.cache.Get("dashboard:stats")
	assert.False(t, ok1)
	_, ok2 := repo.cache.Get("dashboard:stats:store:5")
	assert.False(t, ok2)
}

func TestReportRepository_WithCacheAndStoreID(t *testing.T) {
	require.NoError(t, shared.TruncateTestData(dbPool))
	repo := NewRepository(dbPool)
	ctx := context.Background()

	cacheObj := cache.New(5*time.Minute, 1*time.Minute)
	repo.SetCache(cacheObj)

	userSKU := uniqueSKU("CACHE")
	var cashierID int
	err := dbPool.QueryRow(ctx,
		`INSERT INTO users (username, email, password_hash, role_id) VALUES ($1, $2, 'hash', 1) RETURNING id`,
		userSKU, userSKU+"@test.com",
	).Scan(&cashierID)
	require.NoError(t, err)

	_, err = dbPool.Exec(ctx, `INSERT INTO stores (id, name, is_active) VALUES (1, 'Cache Store', true) ON CONFLICT (id) DO NOTHING`)
	require.NoError(t, err)

	custSKU := uniqueSKU("CACHE")
	var customerID int
	err = dbPool.QueryRow(ctx,
		`INSERT INTO customers (name, phone, email, is_walk_in, is_active) VALUES ($1, $2, $3, true, true) RETURNING id`,
		"Test "+custSKU, fmt.Sprintf("%010d", productCounter.Add(1)), custSKU+"@test.com",
	).Scan(&customerID)
	require.NoError(t, err)

	sku := uniqueSKU("CACHE-PROD")
	var productID int
	err = dbPool.QueryRow(ctx,
		`INSERT INTO products (sku, name, price, stock, status, store_id) VALUES ($1, $2, 50000, 10, 'active', 1) RETURNING id`,
		sku, "Cache Test Product",
	).Scan(&productID)
	require.NoError(t, err)

	_, err = dbPool.Exec(ctx, `INSERT INTO product_stock (product_id, store_id, quantity) VALUES ($1, 1, 10)`, productID)
	require.NoError(t, err)

	invSKU := uniqueSKU("CACHE-INV")
	var saleID int
	err = dbPool.QueryRow(ctx,
		`INSERT INTO sales (invoice_number, cashier_id, customer_id, subtotal, total_amount, payment_method, status, store_id)
		 VALUES ($1, $2, $3, 50000, 50000, 'CASH', 'completed', 1) RETURNING id`,
		invSKU, cashierID, customerID,
	).Scan(&saleID)
	require.NoError(t, err)

	_, err = dbPool.Exec(ctx,
		`INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal, dpp_amount, tax_amount)
		 VALUES ($1, $2, 1, 50000, 50000, 50000, 0)`,
		saleID, productID,
	)
	require.NoError(t, err)

	refreshMaterializedViews(t, ctx)

	now := time.Now()
	start := now.AddDate(0, -1, 0)
	end := now
	prevStart := start.AddDate(0, -1, 0)
	prevEnd := start
	sid := 1

	t.Run("cache miss then hit", func(t *testing.T) {
		pc, err := repo.GetPeriodComparison(ctx, start, end, prevStart, prevEnd, nil)
		require.NoError(t, err)
		require.NotNil(t, pc)
		_, _ = repo.GetPeriodComparison(ctx, start, end, prevStart, prevEnd, nil)
	})
	t.Run("storeID branch", func(t *testing.T) {
		pc, err := repo.GetPeriodComparison(ctx, start, end, prevStart, prevEnd, &sid)
		require.NoError(t, err)
		require.NotNil(t, pc)
	})
	t.Run("dual chart data cache+storeID", func(t *testing.T) {
		cs := now.AddDate(0, 0, -7)
		ce := now
		ps := cs.AddDate(0, 0, -7)
		pe := ce.AddDate(0, 0, -7)
		current, previous, err := repo.GetDualChartData(ctx, cs, ce, ps, pe, nil)
		require.NoError(t, err)
		require.NotNil(t, current)
		_ = previous
		current, previous, err = repo.GetDualChartData(ctx, cs, ce, ps, pe, &sid)
		require.NoError(t, err)
		require.NotNil(t, current)
		_ = previous
		_, _, err = repo.GetDualChartData(ctx, cs, ce, ps, pe, &sid)
		require.NoError(t, err)
	})
	t.Run("live dashboard cache+storeID", func(t *testing.T) {
		_, _, _, _, err := repo.GetLiveDashboardStats(ctx, nil)
		require.NoError(t, err)
		_, _, _, _, err = repo.GetLiveDashboardStats(ctx, nil)
		require.NoError(t, err)
		_, _, _, _, err = repo.GetLiveDashboardStats(ctx, &sid)
		require.NoError(t, err)
		_, _, _, _, err = repo.GetLiveDashboardStats(ctx, &sid)
		require.NoError(t, err)
	})
	t.Run("dashboard stats cache+storeID", func(t *testing.T) {
		stats, err := repo.GetDashboardStats(ctx, nil, shared.JakartaLocation())
		require.NoError(t, err)
		require.NotNil(t, stats)
		stats, err = repo.GetDashboardStats(ctx, nil, shared.JakartaLocation())
		require.NoError(t, err)
		require.NotNil(t, stats)
		stats, err = repo.GetDashboardStats(ctx, &sid, shared.JakartaLocation())
		require.NoError(t, err)
		require.NotNil(t, stats)
	})
	t.Run("available years storeID", func(t *testing.T) {
		_, err := repo.GetAvailableYears(ctx, nil)
		require.NoError(t, err)
		_, err = repo.GetAvailableYears(ctx, &sid)
		require.NoError(t, err)
	})
	t.Run("hourly sales storeID", func(t *testing.T) {
		_, err := repo.GetHourlySales(ctx, now, nil)
		require.NoError(t, err)
		_, err = repo.GetHourlySales(ctx, now, &sid)
		require.NoError(t, err)
	})
	t.Run("daily sales storeID", func(t *testing.T) {
		_, err := repo.GetDailySales(ctx, start, end, nil)
		require.NoError(t, err)
		_, err = repo.GetDailySales(ctx, start, end, &sid)
		require.NoError(t, err)
	})
	t.Run("weekly report storeID", func(t *testing.T) {
		_, err := repo.GetSalesWeeklyReport(ctx, start, end, nil)
		require.NoError(t, err)
		_, err = repo.GetSalesWeeklyReport(ctx, start, end, &sid)
		require.NoError(t, err)
	})
	t.Run("monthly report storeID", func(t *testing.T) {
		_, err := repo.GetSalesMonthlyReport(ctx, start, end, nil)
		require.NoError(t, err)
		_, err = repo.GetSalesMonthlyReport(ctx, start, end, &sid)
		require.NoError(t, err)
	})
}

func TestReportRepository_GetPricingBreakdown_NilStoreID_Seeded(t *testing.T) {
	require.NoError(t, shared.TruncateTestData(dbPool))
	repo := NewRepository(dbPool)
	ctx := context.Background()

	// Seed a sale first
	_, _, amount, _ := seedSale(t, ctx)

	start := time.Now().AddDate(0, -1, 0)
	end := time.Now()

	items, err := repo.GetPricingBreakdown(ctx, start, end, nil)
	require.NoError(t, err)
	require.NotEmpty(t, items)

	totalRevenue := 0
	for _, item := range items {
		totalRevenue += item.Revenue
	}
	assert.GreaterOrEqual(t, totalRevenue, amount, "total pricing breakdown should include seeded sale")
}

func TestReportRepository_GetPricingBreakdown_WithStoreID(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	start := time.Now().AddDate(0, -1, 0)
	end := time.Now()
	sid := 1

	items, err := repo.GetPricingBreakdown(ctx, start, end, &sid)
	require.NoError(t, err)
	_ = items
}

func TestReportRepository_NewSaleCreatedListener(t *testing.T) {
	repo := NewRepository(dbPool)
	listener := repo.NewSaleCreatedListener()
	assert.NotNil(t, listener)
}

func TestReportRepository_SaleCreatedListener_EventTypes(t *testing.T) {
	repo := NewRepository(dbPool)
	listener := repo.NewSaleCreatedListener()

	types := listener.EventTypes()
	assert.Contains(t, types, eventbus.SaleCreated)
}

func TestReportRepository_SaleCreatedListener_HandleEvent_InvalidPayload(t *testing.T) {
	repo := NewRepository(dbPool)
	listener := repo.NewSaleCreatedListener()

	err := listener.HandleEvent(context.Background(), eventbus.Event{
		Type:    eventbus.SaleCreated,
		Payload: "not-a-sale",
	})
	assert.NoError(t, err)
}

func TestReportRepository_SaleCreatedListener_HandleEvent_ValidSale(t *testing.T) {
	if dbPool == nil {
		t.Skip("no database connection")
	}
	require.NoError(t, shared.TruncateTestData(dbPool))
	repo := NewRepository(dbPool)
	listener := repo.NewSaleCreatedListener()
	ctx := context.Background()

	_, _, _, _ = seedSale(t, ctx)

	err := listener.HandleEvent(ctx, eventbus.Event{
		Type:    eventbus.SaleCreated,
		Payload: &events.SaleCreated{ID: 1},
	})
	assert.NoError(t, err)
}

func TestReportRepository_DashboardStats_Seeded(t *testing.T) {
	require.NoError(t, shared.TruncateTestData(dbPool))
	repo := NewRepository(dbPool)
	ctx := context.Background()

	// Seed a sale
	_, _, amount, _ := seedSale(t, ctx)

	stats, err := repo.GetDashboardStats(ctx, nil, shared.JakartaLocation())
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.GreaterOrEqual(t, stats.TotalRevenue, int64(amount))
	assert.GreaterOrEqual(t, stats.TotalSales, int64(1))
	assert.GreaterOrEqual(t, stats.TotalProducts, int64(1))
	assert.GreaterOrEqual(t, stats.LowStockCount, int64(0))
	assert.GreaterOrEqual(t, stats.ActiveCustomers, int64(0))
	// These could be 0 if the sale falls outside "today" in Jakarta time
	assert.GreaterOrEqual(t, stats.TodaysSales, int64(0))
	assert.GreaterOrEqual(t, stats.TodaysRevenue, int64(0))
}
