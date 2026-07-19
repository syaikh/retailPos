package report

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("JWT_SECRET", "test-secret-for-report-mock-tests")
	os.Setenv("ENV", "development")
}

type mockReportService struct {
	getDashboardStatsFn     func(ctx context.Context, storeID int) (*DashboardStats, error)
	getLiveDashboardStatsFn func(ctx context.Context, storeID int) (int, int, int, int, error)
	getHourlySalesFn        func(ctx context.Context, storeID int, date time.Time) ([]ChartDataPoint, error)
	getDailySalesFn         func(ctx context.Context, storeID int, start, end time.Time) ([]ChartDataPoint, error)
	getDualChartDataFn      func(ctx context.Context, cs, ce, ps, pe time.Time, sid *int) ([]ChartDataPoint, []ChartDataPoint, error)
	getPeriodComparisonFn   func(ctx context.Context, cs, ce, ps, pe time.Time, sid *int) (*PeriodComparison, error)
	getSalesWeeklyReportFn  func(ctx context.Context, storeID int, start, end time.Time) ([]WeeklyReportItem, error)
	getSalesMonthlyReportFn func(ctx context.Context, storeID int, start, end time.Time) ([]MonthlyReportItem, error)
	getDualMonthlyReportFn  func(ctx context.Context, storeID int, cs, ce, ps, pe time.Time) ([]MonthlyReportItem, []MonthlyReportItem, error)
	getAvailableYearsFn     func(ctx context.Context, storeID int) ([]int, error)
	getPricingBreakdownFn   func(ctx context.Context, start, end time.Time, storeID *int) ([]PricingBreakdownItem, error)
}

func (m *mockReportService) GetDashboardStats(ctx context.Context, storeID int) (*DashboardStats, error) {
	return m.getDashboardStatsFn(ctx, storeID)
}
func (m *mockReportService) GetLiveDashboardStats(ctx context.Context, storeID int) (int, int, int, int, error) {
	return m.getLiveDashboardStatsFn(ctx, storeID)
}
func (m *mockReportService) GetHourlySales(ctx context.Context, storeID int, date time.Time) ([]ChartDataPoint, error) {
	return m.getHourlySalesFn(ctx, storeID, date)
}
func (m *mockReportService) GetDailySales(ctx context.Context, storeID int, start, end time.Time) ([]ChartDataPoint, error) {
	return m.getDailySalesFn(ctx, storeID, start, end)
}
func (m *mockReportService) GetDualChartData(ctx context.Context, cs, ce, ps, pe time.Time, sid *int) ([]ChartDataPoint, []ChartDataPoint, error) {
	return m.getDualChartDataFn(ctx, cs, ce, ps, pe, sid)
}
func (m *mockReportService) GetPeriodComparison(ctx context.Context, cs, ce, ps, pe time.Time, sid *int) (*PeriodComparison, error) {
	return m.getPeriodComparisonFn(ctx, cs, ce, ps, pe, sid)
}
func (m *mockReportService) GetSalesWeeklyReport(ctx context.Context, storeID int, start, end time.Time) ([]WeeklyReportItem, error) {
	return m.getSalesWeeklyReportFn(ctx, storeID, start, end)
}
func (m *mockReportService) GetSalesMonthlyReport(ctx context.Context, storeID int, start, end time.Time) ([]MonthlyReportItem, error) {
	return m.getSalesMonthlyReportFn(ctx, storeID, start, end)
}
func (m *mockReportService) GetDualMonthlyReport(ctx context.Context, storeID int, cs, ce, ps, pe time.Time) ([]MonthlyReportItem, []MonthlyReportItem, error) {
	return m.getDualMonthlyReportFn(ctx, storeID, cs, ce, ps, pe)
}
func (m *mockReportService) GetAvailableYears(ctx context.Context, storeID int) ([]int, error) {
	return m.getAvailableYearsFn(ctx, storeID)
}
func (m *mockReportService) GetPricingBreakdown(ctx context.Context, start, end time.Time, storeID *int) ([]PricingBreakdownItem, error) {
	if m.getPricingBreakdownFn != nil {
		return m.getPricingBreakdownFn(ctx, start, end, storeID)
	}
	return []PricingBreakdownItem{}, nil
}

func setupReportHandler(svc ReportService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("storeID", nil)
		c.Next()
	})
	h := &Handler{svc: svc}
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm string) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	return r
}

func TestReportHandler_GetDashboardStats_Success(t *testing.T) {
	svc := &mockReportService{
		getDashboardStatsFn: func(ctx context.Context, storeID int) (*DashboardStats, error) {
			return &DashboardStats{
				TodaysRevenue: 500000,
				TotalRevenue:  1000000,
				TodaysSales:   25,
				TotalProducts: 150,
				LowStockCount: 3,
			}, nil
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/stats", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(500000), data["todays_revenue"])
	assert.Equal(t, float64(500000), data["yesterday_revenue"])
	assert.Equal(t, float64(25), data["todays_sales"])
	assert.Equal(t, float64(150), data["total_products"])
	assert.Equal(t, float64(3), data["low_stock_count"])
}

func TestReportHandler_GetDashboardStats_Error(t *testing.T) {
	svc := &mockReportService{
		getDashboardStatsFn: func(ctx context.Context, storeID int) (*DashboardStats, error) {
			return nil, assert.AnError
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/stats", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestReportHandler_GetDashboardStats_NegativeYesterdayRevenue(t *testing.T) {
	svc := &mockReportService{
		getDashboardStatsFn: func(ctx context.Context, storeID int) (*DashboardStats, error) {
			return &DashboardStats{
				TodaysRevenue: 1000000,
				TotalRevenue:  500000,
			}, nil
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/stats", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(0), data["yesterday_revenue"])
}

func TestReportHandler_GetDashboardStats_WithStoreID(t *testing.T) {
	var capturedStoreID int
	svc := &mockReportService{
		getDashboardStatsFn: func(ctx context.Context, storeID int) (*DashboardStats, error) {
			capturedStoreID = storeID
			return &DashboardStats{}, nil
		},
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		sid := 42
		c.Set("storeID", &sid)
		c.Next()
	})
	h := &Handler{svc: svc}
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm string) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/stats", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 42, capturedStoreID)
}

func TestReportHandler_GetLiveDashboardStats_Success(t *testing.T) {
	svc := &mockReportService{
		getLiveDashboardStatsFn: func(ctx context.Context, storeID int) (int, int, int, int, error) {
			return 250000, 15, 80, 5, nil
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/live", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(250000), data["todays_revenue"])
	assert.Equal(t, float64(15), data["todays_sales"])
	assert.Equal(t, float64(80), data["total_products"])
	assert.Equal(t, float64(5), data["low_stock_count"])
}

func TestReportHandler_GetLiveDashboardStats_Error(t *testing.T) {
	svc := &mockReportService{
		getLiveDashboardStatsFn: func(ctx context.Context, storeID int) (int, int, int, int, error) {
			return 0, 0, 0, 0, assert.AnError
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/live", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestReportHandler_GetSalesChartData_Hourly_Success(t *testing.T) {
	svc := &mockReportService{
		getHourlySalesFn: func(ctx context.Context, storeID int, date time.Time) ([]ChartDataPoint, error) {
			return []ChartDataPoint{{Date: "08:00", Total: 100}}, nil
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart?startDate=2024-01-15&endDate=2024-01-15", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestReportHandler_GetSalesChartData_Hourly_Error(t *testing.T) {
	svc := &mockReportService{
		getHourlySalesFn: func(ctx context.Context, storeID int, date time.Time) ([]ChartDataPoint, error) {
			return nil, assert.AnError
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart?startDate=2024-01-15&endDate=2024-01-15", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestReportHandler_GetSalesChartData_Daily_Success(t *testing.T) {
	svc := &mockReportService{
		getDailySalesFn: func(ctx context.Context, storeID int, start, end time.Time) ([]ChartDataPoint, error) {
			return []ChartDataPoint{{Date: "2024-01-15", Total: 500}}, nil
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart?startDate=2024-01-10&endDate=2024-01-15", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestReportHandler_GetSalesChartData_Daily_Error(t *testing.T) {
	svc := &mockReportService{
		getDailySalesFn: func(ctx context.Context, storeID int, start, end time.Time) ([]ChartDataPoint, error) {
			return nil, assert.AnError
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart?startDate=2024-01-10&endDate=2024-01-15", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestReportHandler_GetSalesChartData_DualChart_Success(t *testing.T) {
	svc := &mockReportService{
		getDualChartDataFn: func(ctx context.Context, cs, ce, ps, pe time.Time, sid *int) ([]ChartDataPoint, []ChartDataPoint, error) {
			return []ChartDataPoint{{Date: "2024-01-15", Total: 500}},
				[]ChartDataPoint{{Date: "2024-01-08", Total: 300}}, nil
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart?startDate=2024-01-10&endDate=2024-01-15&prevStart=2024-01-03&prevEnd=2024-01-08", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestReportHandler_GetSalesChartData_DualChart_Error(t *testing.T) {
	svc := &mockReportService{
		getDualChartDataFn: func(ctx context.Context, cs, ce, ps, pe time.Time, sid *int) ([]ChartDataPoint, []ChartDataPoint, error) {
			return nil, nil, assert.AnError
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart?startDate=2024-01-10&endDate=2024-01-15&prevStart=2024-01-03&prevEnd=2024-01-08", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestReportHandler_GetSalesChartData_Hourly_WithPrevStart(t *testing.T) {
	svc := &mockReportService{
		getHourlySalesFn: func(ctx context.Context, storeID int, date time.Time) ([]ChartDataPoint, error) {
			return []ChartDataPoint{{Date: "08:00", Total: 100}}, nil
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart?startDate=2024-01-15&endDate=2024-01-15&prevStart=2024-01-08&prevEnd=2024-01-08", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestReportHandler_GetSalesChartData_Hourly_InvalidPrevStart(t *testing.T) {
	svc := &mockReportService{
		getHourlySalesFn: func(ctx context.Context, storeID int, date time.Time) ([]ChartDataPoint, error) {
			return []ChartDataPoint{}, nil
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart?startDate=2024-01-15&endDate=2024-01-15&prevStart=bad&prevEnd=2024-01-08", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReportHandler_GetSalesChartData_Hourly_PrevStartError(t *testing.T) {
	callCount := 0
	svc := &mockReportService{
		getHourlySalesFn: func(ctx context.Context, storeID int, date time.Time) ([]ChartDataPoint, error) {
			callCount++
			if callCount == 1 {
				return []ChartDataPoint{}, nil
			}
			return nil, assert.AnError
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart?startDate=2024-01-15&endDate=2024-01-15&prevStart=2024-01-08&prevEnd=2024-01-08", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestReportHandler_GetSalesWeeklyReport_Success(t *testing.T) {
	svc := &mockReportService{
		getSalesWeeklyReportFn: func(ctx context.Context, storeID int, start, end time.Time) ([]WeeklyReportItem, error) {
			return []WeeklyReportItem{{WeekStart: "2024-01-08", WeekEnd: "2024-01-14", Total: 1000}}, nil
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart/weekly?startDate=2024-01-01&endDate=2024-01-15", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestReportHandler_GetSalesWeeklyReport_Error(t *testing.T) {
	svc := &mockReportService{
		getSalesWeeklyReportFn: func(ctx context.Context, storeID int, start, end time.Time) ([]WeeklyReportItem, error) {
			return nil, assert.AnError
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart/weekly?startDate=2024-01-01&endDate=2024-01-15", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestReportHandler_GetSalesMonthlyReport_Success(t *testing.T) {
	svc := &mockReportService{
		getSalesMonthlyReportFn: func(ctx context.Context, storeID int, start, end time.Time) ([]MonthlyReportItem, error) {
			return []MonthlyReportItem{{Month: "2024-01", Total: 5000}}, nil
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart/monthly?startDate=2024-01-01&endDate=2024-06-01", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestReportHandler_GetSalesMonthlyReport_Error(t *testing.T) {
	svc := &mockReportService{
		getSalesMonthlyReportFn: func(ctx context.Context, storeID int, start, end time.Time) ([]MonthlyReportItem, error) {
			return nil, assert.AnError
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart/monthly?startDate=2024-01-01&endDate=2024-06-01", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestReportHandler_GetSalesMonthlyReport_Dual_Success(t *testing.T) {
	svc := &mockReportService{
		getDualMonthlyReportFn: func(ctx context.Context, storeID int, cs, ce, ps, pe time.Time) ([]MonthlyReportItem, []MonthlyReportItem, error) {
			return []MonthlyReportItem{{Month: "2024-01", Total: 5000}},
				[]MonthlyReportItem{{Month: "2023-12", Total: 4000}}, nil
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart/monthly?startDate=2024-01-01&endDate=2024-06-01&prevStart=2023-06-01&prevEnd=2023-12-01", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestReportHandler_GetSalesMonthlyReport_Dual_Error(t *testing.T) {
	svc := &mockReportService{
		getDualMonthlyReportFn: func(ctx context.Context, storeID int, cs, ce, ps, pe time.Time) ([]MonthlyReportItem, []MonthlyReportItem, error) {
			return nil, nil, assert.AnError
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart/monthly?startDate=2024-01-01&endDate=2024-06-01&prevStart=2023-06-01&prevEnd=2023-12-01", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestReportHandler_GetSalesMonthlyReport_Dual_InvalidPrevEnd(t *testing.T) {
	svc := &mockReportService{
		getSalesMonthlyReportFn: func(ctx context.Context, storeID int, start, end time.Time) ([]MonthlyReportItem, error) {
			return []MonthlyReportItem{}, nil
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart/monthly?startDate=2024-01-01&endDate=2024-06-01&prevStart=2023-06-01&prevEnd=bad", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReportHandler_GetPeriodComparison_Realtime_Success(t *testing.T) {
	svc := &mockReportService{
		getPeriodComparisonFn: func(ctx context.Context, cs, ce, ps, pe time.Time, sid *int) (*PeriodComparison, error) {
			return &PeriodComparison{
				CurrentRevenue:  100000,
				PreviousRevenue: 80000,
				CurrentOrders:   10,
				PreviousOrders:  8,
			}, nil
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/comparison?period=daily&mode=realtime", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp, "data")
	assert.Contains(t, resp, "meta")
}

func TestReportHandler_GetPeriodComparison_Error(t *testing.T) {
	svc := &mockReportService{
		getPeriodComparisonFn: func(ctx context.Context, cs, ce, ps, pe time.Time, sid *int) (*PeriodComparison, error) {
			return nil, assert.AnError
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/comparison?period=daily&mode=realtime", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestReportHandler_GetPeriodComparison_Completed(t *testing.T) {
	svc := &mockReportService{
		getPeriodComparisonFn: func(ctx context.Context, cs, ce, ps, pe time.Time, sid *int) (*PeriodComparison, error) {
			return &PeriodComparison{}, nil
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/comparison?period=daily&mode=completed&date=2024-01-15", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestReportHandler_GetPeriodComparison_30Days(t *testing.T) {
	svc := &mockReportService{
		getPeriodComparisonFn: func(ctx context.Context, cs, ce, ps, pe time.Time, sid *int) (*PeriodComparison, error) {
			return &PeriodComparison{}, nil
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/comparison?period=daily&mode=30days&date=2024-01-15", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestReportHandler_GetPeriodComparison_DefaultMode(t *testing.T) {
	svc := &mockReportService{
		getPeriodComparisonFn: func(ctx context.Context, cs, ce, ps, pe time.Time, sid *int) (*PeriodComparison, error) {
			return &PeriodComparison{}, nil
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/comparison?period=weekly&mode=unknown&date=2024-01-15", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestReportHandler_GetAvailableYears_Success(t *testing.T) {
	svc := &mockReportService{
		getAvailableYearsFn: func(ctx context.Context, storeID int) ([]int, error) {
			return []int{2022, 2023, 2024}, nil
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/years", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].([]interface{})
	assert.Equal(t, 3, len(data))
}

func TestReportHandler_GetAvailableYears_Error(t *testing.T) {
	svc := &mockReportService{
		getAvailableYearsFn: func(ctx context.Context, storeID int) ([]int, error) {
			return nil, assert.AnError
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/years", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestReportHandler_ExportDashboard_Success(t *testing.T) {
	svc := &mockReportService{
		getPeriodComparisonFn: func(ctx context.Context, cs, ce, ps, pe time.Time, sid *int) (*PeriodComparison, error) {
			return &PeriodComparison{
				CurrentRevenue: 100000, PreviousRevenue: 80000,
				CurrentOrders: 10, PreviousOrders: 8,
				CurrentAOV: 10000, PreviousAOV: 10000,
				RevenuePerDay: 10000, PreviousRevenuePerDay: 8000,
			}, nil
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	body := "period=daily&mode=realtime&date=2024-01-15"
	req := httptest.NewRequest("POST", "/dashboard/export", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "spreadsheetml")
}

func TestReportHandler_ExportDashboard_WithChartData(t *testing.T) {
	svc := &mockReportService{
		getPeriodComparisonFn: func(ctx context.Context, cs, ce, ps, pe time.Time, sid *int) (*PeriodComparison, error) {
			return &PeriodComparison{}, nil
		},
	}
	r := setupReportHandler(svc)

	chartData := []ChartDataPoint{{Date: "2024-01-15", Total: 500}}
	encoded := base64.StdEncoding.EncodeToString([]byte(`[{"date":"2024-01-15","total":500}]`))

	w := httptest.NewRecorder()
	body := "period=daily&mode=realtime&date=2024-01-15&chartData=" + encoded
	req := httptest.NewRequest("POST", "/dashboard/export", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	_ = chartData
}

func TestReportHandler_ExportDashboard_Error(t *testing.T) {
	svc := &mockReportService{
		getPeriodComparisonFn: func(ctx context.Context, cs, ce, ps, pe time.Time, sid *int) (*PeriodComparison, error) {
			return nil, assert.AnError
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	body := "period=daily&mode=realtime&date=2024-01-15"
	req := httptest.NewRequest("POST", "/dashboard/export", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestReportHandler_ExportDashboard_ChartDataTooLarge(t *testing.T) {
	svc := &mockReportService{
		getPeriodComparisonFn: func(ctx context.Context, cs, ce, ps, pe time.Time, sid *int) (*PeriodComparison, error) {
			return &PeriodComparison{}, nil
		},
	}
	r := setupReportHandler(svc)

	// Create chartData string > 1MB
	bigData := strings.Repeat("x", 2<<20+1)
	w := httptest.NewRecorder()
	body := "period=daily&mode=realtime&date=2024-01-15&chartData=" + bigData
	req := httptest.NewRequest("POST", "/dashboard/export", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "chartData too large")
}

func TestReportHandler_ExportDashboard_CompletedMode(t *testing.T) {
	svc := &mockReportService{
		getPeriodComparisonFn: func(ctx context.Context, cs, ce, ps, pe time.Time, sid *int) (*PeriodComparison, error) {
			return &PeriodComparison{}, nil
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	body := "period=daily&mode=completed&date=2024-01-15"
	req := httptest.NewRequest("POST", "/dashboard/export", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestReportHandler_Weekly_InvalidDateRanges(t *testing.T) {
	svc := &mockReportService{
		getSalesWeeklyReportFn: func(ctx context.Context, storeID int, start, end time.Time) ([]WeeklyReportItem, error) {
			return []WeeklyReportItem{}, nil
		},
	}
	r := setupReportHandler(svc)

	t.Run("invalid startDate", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/dashboard/chart/weekly?startDate=bad&endDate=2024-01-10", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid endDate", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/dashboard/chart/weekly?startDate=2024-01-01&endDate=bad", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("endDate before startDate", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/dashboard/chart/weekly?startDate=2024-06-01&endDate=2024-01-01", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("range exceeds 366 days", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/dashboard/chart/weekly?startDate=2022-01-01&endDate=2024-12-31", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestReportHandler_Monthly_InvalidDateRanges(t *testing.T) {
	svc := &mockReportService{
		getSalesMonthlyReportFn: func(ctx context.Context, storeID int, start, end time.Time) ([]MonthlyReportItem, error) {
			return []MonthlyReportItem{}, nil
		},
	}
	r := setupReportHandler(svc)

	t.Run("invalid startDate", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/dashboard/chart/monthly?startDate=bad&endDate=2024-01-10", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid endDate", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/dashboard/chart/monthly?startDate=2024-01-01&endDate=bad", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("endDate before startDate", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/dashboard/chart/monthly?startDate=2024-06-01&endDate=2024-01-01", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("range exceeds 366 days", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/dashboard/chart/monthly?startDate=2022-01-01&endDate=2024-12-31", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestReportHandler_Chart_Daily_InvalidDateRanges(t *testing.T) {
	svc := &mockReportService{
		getDailySalesFn: func(ctx context.Context, storeID int, start, end time.Time) ([]ChartDataPoint, error) {
			return []ChartDataPoint{}, nil
		},
	}
	r := setupReportHandler(svc)

	t.Run("invalid startDate", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/dashboard/chart?startDate=bad&endDate=2024-01-10", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid endDate", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/dashboard/chart?startDate=2024-01-01&endDate=bad", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestReportHandler_Comparison_WithStoreID(t *testing.T) {
	var capturedStoreID *int
	svc := &mockReportService{
		getPeriodComparisonFn: func(ctx context.Context, cs, ce, ps, pe time.Time, sid *int) (*PeriodComparison, error) {
			capturedStoreID = sid
			return &PeriodComparison{}, nil
		},
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		sid := 7
		c.Set("storeID", &sid)
		c.Next()
	})
	h := &Handler{svc: svc}
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm string) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/comparison?period=daily&mode=realtime", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, capturedStoreID)
	assert.Equal(t, 7, *capturedStoreID)
}

func TestReportHandler_Live_WithStoreID(t *testing.T) {
	var capturedStoreID int
	svc := &mockReportService{
		getLiveDashboardStatsFn: func(ctx context.Context, storeID int) (int, int, int, int, error) {
			capturedStoreID = storeID
			return 0, 0, 0, 0, nil
		},
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		sid := 3
		c.Set("storeID", &sid)
		c.Next()
	})
	h := &Handler{svc: svc}
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm string) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/live", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 3, capturedStoreID)
}

func TestReportHandler_Years_WithStoreID(t *testing.T) {
	var capturedStoreID int
	svc := &mockReportService{
		getAvailableYearsFn: func(ctx context.Context, storeID int) ([]int, error) {
			capturedStoreID = storeID
			return []int{2024}, nil
		},
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		sid := 5
		c.Set("storeID", &sid)
		c.Next()
	})
	h := &Handler{svc: svc}
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm string) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/years", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 5, capturedStoreID)
}

// Keep the original validation tests
func setupReportValidationRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("storeID", nil)
		c.Next()
	})
	h := NewHandler(nil)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm string) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	return r
}

func TestMockReportHandler_Chart_InvalidStartDate(t *testing.T) {
	r := setupReportValidationRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart?startDate=not-a-date&endDate=2024-01-10", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid startDate")
}

func TestMockReportHandler_Chart_InvalidEndDate(t *testing.T) {
	r := setupReportValidationRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart?startDate=2024-01-01&endDate=bad", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid endDate")
}

func TestMockReportHandler_Chart_EndDateBeforeStart(t *testing.T) {
	r := setupReportValidationRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart?startDate=2024-01-10&endDate=2024-01-01", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "endDate must not be before startDate")
}

func TestMockReportHandler_Chart_RangeExceeds366Days(t *testing.T) {
	r := setupReportValidationRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart?startDate=2023-01-01&endDate=2024-12-31", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "date range must not exceed 366 days")
}

func TestMockReportHandler_Chart_InvalidPrevStart(t *testing.T) {
	r := setupReportValidationRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart?startDate=2024-01-01&endDate=2024-01-07&prevStart=bad&prevEnd=2023-12-25", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid prevStart")
}

func TestMockReportHandler_Chart_InvalidPrevEnd(t *testing.T) {
	r := setupReportValidationRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart?startDate=2024-01-01&endDate=2024-01-07&prevStart=2023-12-25&prevEnd=bad", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid prevEnd")
}

func TestMockReportHandler_Weekly_InvalidStartDate(t *testing.T) {
	r := setupReportValidationRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart/weekly?startDate=bad&endDate=2024-01-10", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid startDate")
}

func TestMockReportHandler_Weekly_InvalidEndDate(t *testing.T) {
	r := setupReportValidationRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart/weekly?startDate=2024-01-01&endDate=bad", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid endDate")
}

func TestMockReportHandler_Weekly_EndDateBeforeStart(t *testing.T) {
	r := setupReportValidationRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart/weekly?startDate=2024-06-01&endDate=2024-01-01", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "endDate must not be before startDate")
}

func TestMockReportHandler_Weekly_RangeExceeds366Days(t *testing.T) {
	r := setupReportValidationRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart/weekly?startDate=2022-01-01&endDate=2024-12-31", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "date range must not exceed 366 days")
}

func TestMockReportHandler_Monthly_InvalidStartDate(t *testing.T) {
	r := setupReportValidationRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart/monthly?startDate=bad&endDate=2024-01-10", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid startDate")
}

func TestMockReportHandler_Monthly_InvalidEndDate(t *testing.T) {
	r := setupReportValidationRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart/monthly?startDate=2024-01-01&endDate=bad", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid endDate")
}

func TestMockReportHandler_Monthly_EndDateBeforeStart(t *testing.T) {
	r := setupReportValidationRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart/monthly?startDate=2024-06-01&endDate=2024-01-01", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "endDate must not be before startDate")
}

func TestMockReportHandler_Monthly_InvalidPrevStart(t *testing.T) {
	r := setupReportValidationRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart/monthly?startDate=2024-01-01&endDate=2024-06-01&prevStart=bad&prevEnd=2023-12-01", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid prevStart")
}

func TestMockReportHandler_Monthly_InvalidPrevEnd(t *testing.T) {
	r := setupReportValidationRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart/monthly?startDate=2024-01-01&endDate=2024-06-01&prevStart=2023-06-01&prevEnd=bad", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid prevEnd")
}

func TestMockReportHandler_Monthly_RangeExceeds366Days(t *testing.T) {
	r := setupReportValidationRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart/monthly?startDate=2022-01-01&endDate=2024-12-31", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "date range must not exceed 366 days")
}

func TestMockReportHandler_Export_ValidDatePassesValidation(t *testing.T) {
	r := setupReportValidationRouter()
	w := httptest.NewRecorder()
	body := "period=daily&mode=realtime&date=not-a-date"
	req := httptest.NewRequest("POST", "/dashboard/export", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid date")
}

func TestMockReportHandler_RoutesRegistered(t *testing.T) {
	r := setupReportValidationRouter()

	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/dashboard/chart?startDate=bad"},
		{"GET", "/dashboard/chart/weekly?startDate=bad"},
		{"GET", "/dashboard/chart/monthly?startDate=bad"},
		{"GET", "/dashboard/comparison?date=bad"},
		{"POST", "/dashboard/export"},
	}

	for _, route := range routes {
		w := httptest.NewRecorder()
		var req *http.Request
		if route.method == "POST" {
			req = httptest.NewRequest(route.method, route.path, strings.NewReader("date=bad"))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		} else {
			req = httptest.NewRequest(route.method, route.path, nil)
		}
		r.ServeHTTP(w, req)
		assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest}, w.Code,
			"route %s %s should be registered", route.method, route.path)
	}
}

func TestMockReportHandler_Comparison_InvalidDate(t *testing.T) {
	r := setupReportValidationRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/comparison?period=daily&mode=realtime&date=bad", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid date")
}

func TestMockReportHandler_Export_InvalidDate(t *testing.T) {
	r := setupReportValidationRouter()
	w := httptest.NewRecorder()
	body := "period=daily&mode=realtime&date=bad"
	req := httptest.NewRequest("POST", "/dashboard/export", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid date")
}

func TestMockReportHandler_JSONResponseFormat(t *testing.T) {
	r := setupReportValidationRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart?startDate=bad", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp, "error")
}

func TestStoreIDPtr(t *testing.T) {
	t.Run("zero returns nil", func(t *testing.T) {
		result := storeIDPtr(0)
		assert.Nil(t, result)
	})

	t.Run("non-zero returns pointer", func(t *testing.T) {
		result := storeIDPtr(42)
		require.NotNil(t, result)
		assert.Equal(t, 42, *result)
	})

	t.Run("negative returns pointer", func(t *testing.T) {
		result := storeIDPtr(-1)
		require.NotNil(t, result)
		assert.Equal(t, -1, *result)
	})
}

func TestReportHandler_GetPricingBreakdown_Success(t *testing.T) {
	svc := &mockReportService{
		getPricingBreakdownFn: func(ctx context.Context, start, end time.Time, storeID *int) ([]PricingBreakdownItem, error) {
			return []PricingBreakdownItem{
				{PricingType: "normal", Revenue: 50000, OrderCount: 10, ItemCount: 25},
				{PricingType: "special_price", Revenue: 30000, OrderCount: 5, ItemCount: 12},
			}, nil
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/pricing-breakdown?start=2024-01-01&end=2024-01-31", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data []PricingBreakdownItem `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Len(t, resp.Data, 2)
	assert.Equal(t, "normal", resp.Data[0].PricingType)
}

func TestReportHandler_GetPricingBreakdown_NoDates(t *testing.T) {
	svc := &mockReportService{
		getPricingBreakdownFn: func(ctx context.Context, start, end time.Time, storeID *int) ([]PricingBreakdownItem, error) {
			return []PricingBreakdownItem{}, nil
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/pricing-breakdown", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestReportHandler_GetPricingBreakdown_InvalidStart(t *testing.T) {
	svc := &mockReportService{}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/pricing-breakdown?start=bad&end=2024-01-31", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid start date")
}

func TestReportHandler_GetPricingBreakdown_InvalidEnd(t *testing.T) {
	svc := &mockReportService{}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/pricing-breakdown?start=2024-01-01&end=bad", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid end date")
}

func TestReportHandler_GetPricingBreakdown_SvcError(t *testing.T) {
	svc := &mockReportService{
		getPricingBreakdownFn: func(ctx context.Context, start, end time.Time, storeID *int) ([]PricingBreakdownItem, error) {
			return nil, assert.AnError
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/pricing-breakdown?start=2024-01-01&end=2024-01-31", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestReportHandler_GetPricingBreakdown_WithStoreID(t *testing.T) {
	var capturedStoreID *int
	svc := &mockReportService{
		getPricingBreakdownFn: func(ctx context.Context, start, end time.Time, storeID *int) ([]PricingBreakdownItem, error) {
			capturedStoreID = storeID
			return []PricingBreakdownItem{}, nil
		},
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		sid := 9
		c.Set("storeID", &sid)
		c.Next()
	})
	h := &Handler{svc: svc}
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm string) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/pricing-breakdown?start=2024-01-01&end=2024-01-31", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, capturedStoreID)
	assert.Equal(t, 9, *capturedStoreID)
}

func TestReportHandler_GetSalesChartData_Daily_Dual_InvalidPrevEnd(t *testing.T) {
	svc := &mockReportService{
		getDailySalesFn: func(ctx context.Context, storeID int, start, end time.Time) ([]ChartDataPoint, error) {
			return []ChartDataPoint{}, nil
		},
	}
	r := setupReportHandler(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/dashboard/chart?startDate=2024-01-10&endDate=2024-01-15&prevStart=2024-01-03&prevEnd=bad", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid prevEnd")
}
