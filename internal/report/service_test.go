package report

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/eventbus"
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
