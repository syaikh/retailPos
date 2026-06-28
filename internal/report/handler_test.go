package report

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/eventbus"
)

func skipIfNoDB(t *testing.T) {
	t.Helper()
	if dbPool == nil {
		t.Skip("no database connection")
	}
}

func testAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "testuser")
		c.Set("roleID", 1)
		c.Set("role", "superadmin")
		c.Set("permissions", []string{"dashboard:read", "report:read"})
		c.Set("storeID", nil)
		c.Next()
	}
}

func testPermMiddleware(perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func setupReportRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()

	svc := NewService(repo, bus)
	h := NewHandler(svc)

	r := gin.New()
	h.RegisterRoutes(r.Group("/"), testAuthMiddleware(), testPermMiddleware)
	return r
}

func TestHandler_DashboardStats(t *testing.T) {
	skipIfNoDB(t)
	r := setupReportRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/dashboard/stats", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data struct {
			TodaysRevenue   int `json:"todays_revenue"`
			YesterdayRevenue int `json:"yesterday_revenue"`
			TodaysSales     int `json:"todays_sales"`
			TotalProducts   int `json:"total_products"`
			LowStockCount   int `json:"low_stock_count"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, resp.Data.TodaysRevenue, 0)
}

func TestHandler_LiveDashboardStats(t *testing.T) {
	skipIfNoDB(t)
	r := setupReportRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/dashboard/live", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data struct {
			TodaysRevenue  int `json:"todays_revenue"`
			TodaysSales    int `json:"todays_sales"`
			TotalProducts  int `json:"total_products"`
			LowStockCount  int `json:"low_stock_count"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, resp.Data.TodaysRevenue, 0)
}

func TestHandler_SalesChartData(t *testing.T) {
	skipIfNoDB(t)
	r := setupReportRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/dashboard/chart?startDate=2023-01-01&endDate=2023-01-07", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data []ChartDataPoint `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotNil(t, resp.Data)
}

func TestHandler_SalesWeeklyReport(t *testing.T) {
	skipIfNoDB(t)
	r := setupReportRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/dashboard/chart/weekly?startDate=2023-01-01&endDate=2023-03-01", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data []WeeklyReportItem `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotNil(t, resp.Data)
}

func TestHandler_SalesMonthlyReport(t *testing.T) {
	skipIfNoDB(t)
	r := setupReportRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/dashboard/chart/monthly?startDate=2023-01-01&endDate=2023-06-01", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data []MonthlyReportItem `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotNil(t, resp.Data)
}

func TestHandler_PeriodComparison(t *testing.T) {
	skipIfNoDB(t)
	r := setupReportRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/dashboard/comparison?period=daily&mode=realtime", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data *PeriodComparison `json:"data"`
		Meta interface{}      `json:"meta"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.NotNil(t, resp.Data)
}

func TestHandler_AvailableYears(t *testing.T) {
	skipIfNoDB(t)
	r := setupReportRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/dashboard/years", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data []int `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotNil(t, resp.Data)
}

func TestHandler_ExportDashboard(t *testing.T) {
	skipIfNoDB(t)
	r := setupReportRouter()

	body := "period=daily&mode=realtime"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/dashboard/export", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "spreadsheetml")
}
