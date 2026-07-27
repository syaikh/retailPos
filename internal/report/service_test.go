package report

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/shared"
	"retail-pos-system/internal/user"
)

func TestReportService_DashboardStats(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	stats, err := svc.GetDashboardStats(ctx, 0)
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.GreaterOrEqual(t, stats.TotalSales, int64(0))
	assert.GreaterOrEqual(t, stats.TotalRevenue, int64(0))
	assert.GreaterOrEqual(t, stats.TotalProducts, int64(0))
	assert.GreaterOrEqual(t, stats.LowStockCount, int64(0))
	assert.GreaterOrEqual(t, stats.TodaysSales, int64(0))
	assert.GreaterOrEqual(t, stats.TodaysRevenue, int64(0))
	assert.GreaterOrEqual(t, stats.ActiveCustomers, int64(0))
}

func TestReportService_LiveDashboardStats(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	rev, sales, products, lowStock, err := svc.GetLiveDashboardStats(ctx, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, rev, 0)
	assert.GreaterOrEqual(t, sales, 0)
	assert.GreaterOrEqual(t, products, 0)
	assert.GreaterOrEqual(t, lowStock, 0)
}

func TestReportService_PeriodComparison(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	now := time.Now()
	start := now.AddDate(0, -1, 0)
	prevStart := start.AddDate(0, -1, 0)

	result, err := svc.GetPeriodComparison(ctx, start, now, prevStart, start, nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.GreaterOrEqual(t, result.CurrentRevenue, 0)
	assert.GreaterOrEqual(t, result.PreviousRevenue, 0)
	assert.GreaterOrEqual(t, result.CurrentOrders, 0)
	assert.GreaterOrEqual(t, result.PreviousOrders, 0)
	assert.GreaterOrEqual(t, result.CurrentAOV, 0)
	assert.GreaterOrEqual(t, result.PreviousAOV, 0)
	assert.NotNil(t, result.PreviousHasAnyData)
}

func TestReportService_DualChartData(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	now := time.Now()
	currentStart := now.AddDate(0, 0, -7)
	currentEnd := now
	prevStart := currentStart.AddDate(0, 0, -7)
	prevEnd := currentEnd.AddDate(0, 0, -7)

	current, previous, err := svc.GetDualChartData(ctx, currentStart, currentEnd, prevStart, prevEnd, nil)
	require.NoError(t, err)
	assert.NotNil(t, current)
	assert.NotNil(t, previous)
}

func TestReportService_AvailableYears(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	years, err := svc.GetAvailableYears(ctx, 0)
	require.NoError(t, err)
	assert.NotNil(t, years)
}

func TestReportService_HourlySales(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	result, err := svc.GetHourlySales(ctx, 0, time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestReportService_DailySales(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	start := time.Now().AddDate(0, -1, 0)
	result, err := svc.GetDailySales(ctx, 0, start, time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestReportService_SalesWeeklyReport(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	start := time.Now().AddDate(0, -3, 0)
	result, err := svc.GetSalesWeeklyReport(ctx, 0, start, time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestReportService_SalesMonthlyReport(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	start := time.Now().AddDate(0, -6, 0)
	result, err := svc.GetSalesMonthlyReport(ctx, 0, start, time.Now())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestReportService_GetPricingBreakdown(t *testing.T) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	now := time.Now()
	start := now.AddDate(0, -1, 0)

	t.Run("nil storeID", func(t *testing.T) {
		items, err := svc.GetPricingBreakdown(ctx, start, now, nil)
		require.NoError(t, err)
		_ = items
	})

	t.Run("with storeID", func(t *testing.T) {
		sid := 1
		items, err := svc.GetPricingBreakdown(ctx, start, now, &sid)
		require.NoError(t, err)
		_ = items
	})
}

func TestReportService_GetDualMonthlyReport(t *testing.T) {
	require.NoError(t, shared.TruncateTestData(dbPool))
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
	ctx := context.Background()

	now := time.Now()
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	previousMonth := currentMonth.AddDate(0, -1, 0)

	hash, _ := bcrypt.GenerateFromPassword([]byte("cashier123"), bcrypt.MinCost)
	cashier := &user.User{
		Username: "report_cashier",
		Email:    "report_cashier@test.com",
		Password: string(hash),
		RoleID:   1,
		IsActive: true,
	}
	userRepo := user.NewRepository(dbPool)
	err := userRepo.CreateUser(ctx, cashier)
	require.NoError(t, err)

	var custID int
	err = dbPool.QueryRow(ctx, `
		INSERT INTO customers (name, phone, email, is_walk_in, is_active)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (phone) DO UPDATE SET phone = customers.phone
		RETURNING id
	`, "Report Customer", "0819999999", "report@test.com", false, true).Scan(&custID)
	require.NoError(t, err)

	_, err = dbPool.Exec(ctx, `
		INSERT INTO sales (invoice_number, cashier_id, customer_id, store_id, subtotal, discount, tax, total_amount, payment_method, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, "INV-CURR-001", cashier.ID, custID, nil, 100000, 0, 11000, 111000, "cash", "completed", currentMonth.Add(2*time.Hour))
	require.NoError(t, err)

	_, err = dbPool.Exec(ctx, `
		INSERT INTO sales (invoice_number, cashier_id, customer_id, store_id, subtotal, discount, tax, total_amount, payment_method, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, "INV-PREV-001", cashier.ID, custID, nil, 200000, 0, 22000, 222000, "cash", "completed", previousMonth.Add(2*time.Hour))
	require.NoError(t, err)

	currentStart := currentMonth
	currentEnd := currentMonth.AddDate(0, 1, 0)
	previousStart := previousMonth
	previousEnd := currentMonth

	current, previous, err := svc.GetDualMonthlyReport(ctx, 0, currentStart, currentEnd, previousStart, previousEnd)
	require.NoError(t, err)
	require.NotNil(t, current)
	require.NotNil(t, previous)
	assert.NotEmpty(t, current)
	assert.NotEmpty(t, previous)
	assert.Equal(t, 111000, current[0].Total)
	assert.Equal(t, 222000, previous[0].Total)
}
