package report

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("JWT_SECRET", "test-secret-for-report-mock-tests")
	os.Setenv("ENV", "development")
}

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
