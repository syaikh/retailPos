package report

import (
	"context"
	"time"

	"retail-pos-system/internal/shared"
)

type Repo interface {
	GetAvailableYears(ctx context.Context, storeID *int) ([]int, error)
	GetDailySales(ctx context.Context, start, end time.Time, storeID *int) ([]ChartDataPoint, error)
	GetDashboardStats(ctx context.Context, storeID *int, jakartaLoc *time.Location) (*DashboardStats, error)
	GetDualChartData(ctx context.Context, currentStart, currentEnd, previousStart, previousEnd time.Time, storeID *int) (current, previous []ChartDataPoint, err error)
	GetHourlySales(ctx context.Context, date time.Time, storeID *int) ([]ChartDataPoint, error)
	GetLiveDashboardStats(ctx context.Context, storeID *int) (todaysRevenue, todaysSales, totalProducts, lowStockCount int, err error)
	GetPeriodComparison(ctx context.Context, currentStart, currentEnd, previousStart, previousEnd time.Time, storeID *int) (*PeriodComparison, error)
	GetPricingBreakdown(ctx context.Context, start, end time.Time, storeID *int) ([]PricingBreakdownItem, error)
	GetSalesMonthlyReport(ctx context.Context, start, end time.Time, storeID *int) ([]MonthlyReportItem, error)
	GetSalesWeeklyReport(ctx context.Context, start, end time.Time, storeID *int) ([]WeeklyReportItem, error)
}

type service struct {
	repo     Repo
	eventBus shared.EventBus
}

func NewService(repo Repo, eventBus shared.EventBus) Service {
	return &service{
		repo:     repo,
		eventBus: eventBus,
	}
}

func storeIDPtr(storeID int) *int {
	if storeID == 0 {
		return nil
	}
	return &storeID
}

func (s *service) GetDashboardStats(ctx context.Context, storeID int) (*DashboardStats, error) {
	return s.repo.GetDashboardStats(ctx, storeIDPtr(storeID), shared.JakartaLocation())
}

func (s *service) GetLiveDashboardStats(ctx context.Context, storeID int) (todaysRevenue, todaysSales, totalProducts, lowStockCount int, err error) {
	return s.repo.GetLiveDashboardStats(ctx, storeIDPtr(storeID))
}

func (s *service) GetPeriodComparison(ctx context.Context, currentStart, currentEnd, previousStart, previousEnd time.Time, storeID *int) (*PeriodComparison, error) {
	return s.repo.GetPeriodComparison(ctx, currentStart, currentEnd, previousStart, previousEnd, storeID)
}

func (s *service) GetDualChartData(ctx context.Context, currentStart, currentEnd, previousStart, previousEnd time.Time, storeID *int) (current, previous []ChartDataPoint, err error) {
	return s.repo.GetDualChartData(ctx, currentStart, currentEnd, previousStart, previousEnd, storeID)
}

func (s *service) GetAvailableYears(ctx context.Context, storeID int) ([]int, error) {
	return s.repo.GetAvailableYears(ctx, storeIDPtr(storeID))
}

func (s *service) GetHourlySales(ctx context.Context, storeID int, date time.Time) ([]ChartDataPoint, error) {
	return s.repo.GetHourlySales(ctx, date, storeIDPtr(storeID))
}

func (s *service) GetDailySales(ctx context.Context, storeID int, start, end time.Time) ([]ChartDataPoint, error) {
	return s.repo.GetDailySales(ctx, start, end, storeIDPtr(storeID))
}

func (s *service) GetSalesWeeklyReport(ctx context.Context, storeID int, start, end time.Time) ([]WeeklyReportItem, error) {
	return s.repo.GetSalesWeeklyReport(ctx, start, end, storeIDPtr(storeID))
}

func (s *service) GetSalesMonthlyReport(ctx context.Context, storeID int, start, end time.Time) ([]MonthlyReportItem, error) {
	return s.repo.GetSalesMonthlyReport(ctx, start, end, storeIDPtr(storeID))
}

func (s *service) GetDualMonthlyReport(ctx context.Context, storeID int, currentStart, currentEnd, previousStart, previousEnd time.Time) (current, previous []MonthlyReportItem, err error) {
	current, err = s.repo.GetSalesMonthlyReport(ctx, currentStart, currentEnd, storeIDPtr(storeID))
	if err != nil {
		return nil, nil, err
	}
	previous, err = s.repo.GetSalesMonthlyReport(ctx, previousStart, previousEnd, storeIDPtr(storeID))
	if err != nil {
		return nil, nil, err
	}
	return current, previous, nil
}

func (s *service) GetPricingBreakdown(ctx context.Context, start, end time.Time, storeID *int) ([]PricingBreakdownItem, error) {
	return s.repo.GetPricingBreakdown(ctx, start, end, storeID)
}
