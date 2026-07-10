package report

import (
	"context"
	"time"
)

type EventBus interface {
	Publish(ctx context.Context, topic string, event interface{}) error
}

type Service struct {
	repo     *Repository
	eventBus EventBus
}

func NewService(repo *Repository, eventBus EventBus) *Service {
	return &Service{
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

func (s *Service) GetDashboardStats(ctx context.Context, storeID int) (*DashboardStats, error) {
	return s.repo.GetDashboardStats(ctx, storeIDPtr(storeID), mustLoadJakarta())
}

func (s *Service) GetLiveDashboardStats(ctx context.Context, storeID int) (todaysRevenue, todaysSales, totalProducts, lowStockCount int, err error) {
	return s.repo.GetLiveDashboardStats(ctx, storeIDPtr(storeID))
}

func (s *Service) GetPeriodComparison(ctx context.Context, currentStart, currentEnd, previousStart, previousEnd time.Time, storeID *int) (*PeriodComparison, error) {
	return s.repo.GetPeriodComparison(ctx, currentStart, currentEnd, previousStart, previousEnd, storeID)
}

func (s *Service) GetDualChartData(ctx context.Context, currentStart, currentEnd, previousStart, previousEnd time.Time, storeID *int) (current, previous []ChartDataPoint, err error) {
	return s.repo.GetDualChartData(ctx, currentStart, currentEnd, previousStart, previousEnd, storeID)
}

func (s *Service) GetAvailableYears(ctx context.Context, storeID int) ([]int, error) {
	return s.repo.GetAvailableYears(ctx, storeIDPtr(storeID))
}

func (s *Service) GetHourlySales(ctx context.Context, storeID int, date time.Time) ([]ChartDataPoint, error) {
	return s.repo.GetHourlySales(ctx, date, storeIDPtr(storeID))
}

func (s *Service) GetDailySales(ctx context.Context, storeID int, start, end time.Time) ([]ChartDataPoint, error) {
	return s.repo.GetDailySales(ctx, start, end, storeIDPtr(storeID))
}

func (s *Service) GetSalesWeeklyReport(ctx context.Context, storeID int, start, end time.Time) ([]WeeklyReportItem, error) {
	return s.repo.GetSalesWeeklyReport(ctx, start, end, storeIDPtr(storeID))
}

func (s *Service) GetSalesMonthlyReport(ctx context.Context, storeID int, start, end time.Time) ([]MonthlyReportItem, error) {
	return s.repo.GetSalesMonthlyReport(ctx, start, end, storeIDPtr(storeID))
}

func (s *Service) GetDualMonthlyReport(ctx context.Context, storeID int, currentStart, currentEnd, previousStart, previousEnd time.Time) (current, previous []MonthlyReportItem, err error) {
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
