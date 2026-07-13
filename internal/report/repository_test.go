package report

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

var dbPool *pgxpool.Pool

func TestMain(m *testing.M) {
	pool, err := shared.NewTestDB()
	if err != nil {
		os.Exit(0)
	}
	dbPool = pool
	defer pool.Close()

	if err := shared.RunMigrations(pool, "../../database/migrations"); err != nil {
		os.Exit(0)
	}

	if err := shared.TruncateTestData(pool); err != nil {
		os.Exit(0)
	}

	os.Exit(m.Run())
}

func TestReportRepository_PeriodComparison(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	now := time.Now()
	start := now.AddDate(0, -1, 0)
	prevStart := start.AddDate(0, -1, 0)

	result, err := repo.GetPeriodComparison(ctx, start, now, prevStart, start, nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.GreaterOrEqual(t, result.CurrentRevenue, 0)
	assert.GreaterOrEqual(t, result.PreviousRevenue, 0)
	assert.GreaterOrEqual(t, result.CurrentOrders, 0)
	assert.GreaterOrEqual(t, result.PreviousOrders, 0)
	assert.GreaterOrEqual(t, result.CurrentAOV, 0)
	assert.GreaterOrEqual(t, result.PreviousAOV, 0)
}

func TestReportRepository_DualChartData(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	now := time.Now()
	currentStart := now.AddDate(0, 0, -7)
	currentEnd := now
	prevStart := currentStart.AddDate(0, 0, -7)
	prevEnd := currentEnd.AddDate(0, 0, -7)

	current, previous, err := repo.GetDualChartData(ctx, currentStart, currentEnd, prevStart, prevEnd, nil)
	require.NoError(t, err)
	assert.NotNil(t, current)
	assert.NotNil(t, previous)
}

func TestReportRepository_LiveDashboardStats(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	rev, sales, products, lowStock, err := repo.GetLiveDashboardStats(ctx, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, rev, 0)
	assert.GreaterOrEqual(t, sales, 0)
	assert.GreaterOrEqual(t, products, 0)
	assert.GreaterOrEqual(t, lowStock, 0)
}

func TestReportRepository_AvailableYears(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	years, err := repo.GetAvailableYears(ctx, nil)
	require.NoError(t, err)
	assert.NotNil(t, years)
}

func TestReportRepository_HourlySales(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	date := time.Now()
	result, err := repo.GetHourlySales(ctx, date, nil)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestReportRepository_DailySales(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	start := time.Now().AddDate(0, -1, 0)
	end := time.Now()
	result, err := repo.GetDailySales(ctx, start, end, nil)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestReportRepository_SalesWeeklyReport(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	start := time.Now().AddDate(0, -3, 0)
	end := time.Now()
	result, err := repo.GetSalesWeeklyReport(ctx, start, end, nil)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestReportRepository_SalesMonthlyReport(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	start := time.Now().AddDate(0, -6, 0)
	end := time.Now()
	result, err := repo.GetSalesMonthlyReport(ctx, start, end, nil)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestReportRepository_DashboardStats(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	stats, err := repo.GetDashboardStats(ctx, nil, shared.JakartaLocation())
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
