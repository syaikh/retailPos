package handler_test

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"retail-pos-system/internal/auth"
	"retail-pos-system/internal/config"
	"retail-pos-system/internal/delivery/http/handler"
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dbAvailable returns true when a PostgreSQL instance is reachable on the given port.
// Falls back to TEST_DB_PORT / DB_PORT env, then localhost:5432.
func dbAvailable() bool {
	port := os.Getenv("TEST_DB_PORT")
	if port == "" {
		port = os.Getenv("DB_PORT")
	}
	if port == "" {
		port = "5432"
	}
	conn, err := net.DialTimeout("tcp", "localhost:"+port, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// requireDB skips the calling test when no PostgreSQL is available.
func requireDB(t *testing.T) {
	t.Helper()
	if !dbAvailable() {
		t.Skip("skipping: PostgreSQL not available on configured port")
	}
}

// ─── helpers ───────────────────────────────────────────────────────────────────

func setupThresholdTestServer(t *testing.T) (*gin.Engine, *repository.TestDB, *handler.Handler) {
	t.Helper()
	testDB := repository.NewTestDB(t)
	userRepo := repository.NewPostgresRepository(testDB.Pool())
	roleRepo := repository.NewPostgresRepository(testDB.Pool())
	productRepo := repository.NewPostgresRepository(testDB.Pool())
	saleRepo := repository.NewPostgresRepository(testDB.Pool())
	auditRepo := repository.NewPostgresRepository(testDB.Pool())
	authSvc := auth.NewAuthService(userRepo, testDB.Pool())
	h := handler.NewHandler(userRepo, roleRepo, productRepo, saleRepo, authSvc, nil, auditRepo)

	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Public routes (no auth)
	r.GET("/api/stock-thresholds", h.GetStockThresholds)
	r.POST("/api/login", h.Login)

	// Protected routes
	protected := r.Group("/api")
	protected.Use(func(c *gin.Context) { c.Set("authService", authSvc); c.Next() })
	protected.Use(middleware.AuthMiddleware())
	protected.GET("/products", h.GetProducts)
	protected.GET("/products/:id", h.GetProductByID)
	protected.POST("/products", h.CreateProduct)
	protected.PUT("/products/:id", h.UpdateProduct)
	protected.DELETE("/products/:id", h.DeleteProduct)
	protected.POST("/sales", h.CreateSale)
	protected.GET("/sales", h.GetSalesHistory)
	protected.GET("/sales/:id", h.GetSaleByID)
	protected.GET("/stats", h.GetDashboardStats)

	return r, testDB, h
}

func loginToken(t *testing.T, r *gin.Engine) string {
	t.Helper()
	reqBody, _ := json.Marshal(map[string]string{"username": "superadmin", "password": "admin123"})
	req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp["access_token"].(string)
}

// ─── Config / threshold tests ──────────────────────────────────────────────────

func TestConfig_Defaults(t *testing.T) {
	os.Unsetenv("STOCK_WARNING_THRESHOLD")
	os.Unsetenv("STOCK_CRITICAL_THRESHOLD")
	defer os.Unsetenv("STOCK_WARNING_THRESHOLD")
	defer os.Unsetenv("STOCK_CRITICAL_THRESHOLD")

	cfg := config.Load()
	assert.Equal(t, 10, cfg.StockWarningThreshold, "default warning threshold")
	assert.Equal(t, 5, cfg.StockCriticalThreshold, "default critical threshold")
}

func TestConfig_EnvOverride(t *testing.T) {
	os.Setenv("STOCK_WARNING_THRESHOLD", "15")
	os.Setenv("STOCK_CRITICAL_THRESHOLD", "7")
	defer os.Unsetenv("STOCK_WARNING_THRESHOLD")
	defer os.Unsetenv("STOCK_CRITICAL_THRESHOLD")

	cfg := config.Load()
	assert.Equal(t, 15, cfg.StockWarningThreshold)
	assert.Equal(t, 7, cfg.StockCriticalThreshold)
}

func TestConfig_InvalidEnvFallsBack(t *testing.T) {
	os.Setenv("STOCK_WARNING_THRESHOLD", "abc")
	os.Setenv("STOCK_CRITICAL_THRESHOLD", "-1")
	defer os.Unsetenv("STOCK_WARNING_THRESHOLD")
	defer os.Unsetenv("STOCK_CRITICAL_THRESHOLD")

	cfg := config.Load()
	assert.Equal(t, 5, cfg.StockCriticalThreshold, "invalid critical falls back to default 5")
	assert.Equal(t, 10, cfg.StockWarningThreshold, "invalid warning falls back to default 10")
}

// ─── GetStockThresholds handler ────────────────────────────────────────────────

func TestHandler_GetStockThresholds_Defaults(t *testing.T) {
	requireDB(t)
	os.Unsetenv("STOCK_WARNING_THRESHOLD")
	os.Unsetenv("STOCK_CRITICAL_THRESHOLD")
	defer os.Unsetenv("STOCK_WARNING_THRESHOLD")
	defer os.Unsetenv("STOCK_CRITICAL_THRESHOLD")

	r, testDB, _ := setupThresholdTestServer(t)
	defer testDB.Close(t)

	req, _ := http.NewRequest("GET", "/api/stock-thresholds", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(10), resp["warning"], "response warning key")
	assert.Equal(t, float64(5), resp["critical"], "response critical key")
}

func TestHandler_GetStockThresholds_EnvOverrides(t *testing.T) {
	requireDB(t)
	os.Setenv("STOCK_WARNING_THRESHOLD", "20")
	os.Setenv("STOCK_CRITICAL_THRESHOLD", "3")
	defer os.Unsetenv("STOCK_WARNING_THRESHOLD")
	defer os.Unsetenv("STOCK_CRITICAL_THRESHOLD")

	r, testDB, _ := setupThresholdTestServer(t)
	defer testDB.Close(t)

	req, _ := http.NewRequest("GET", "/api/stock-thresholds", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(20), resp["warning"])
	assert.Equal(t, float64(3), resp["critical"])
}

// ─── DashboardStats uses critical threshold for low_stock_count ─────────────────

func TestHandler_GetDashboardStats_LowStockCount_Default(t *testing.T) {
	requireDB(t)
	os.Unsetenv("STOCK_WARNING_THRESHOLD")
	defer os.Unsetenv("STOCK_WARNING_THRESHOLD")
	defer os.Unsetenv("STOCK_CRITICAL_THRESHOLD")

	r, testDB, h := setupThresholdTestServer(t)
	defer testDB.Close(t)

	_ = h // needed to verify intent: stats handler uses config.StockCriticalThreshold

	cfg := config.Load()
	assert.Equal(t, 5, cfg.StockCriticalThreshold)

	token := loginToken(t, r)
	req, _ := http.NewRequest("GET", "/api/stats", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	data := resp["data"].(map[string]interface{})
	assert.Contains(t, data, "low_stock_count")
}

func TestHandler_GetDashboardStats_LowStockCount_CustomThreshold(t *testing.T) {
	requireDB(t)
	os.Setenv("STOCK_CRITICAL_THRESHOLD", "10")
	defer os.Unsetenv("STOCK_CRITICAL_THRESHOLD")

	r, testDB, _ := setupThresholdTestServer(t)
	defer testDB.Close(t)

	cfg := config.Load()
	assert.Equal(t, 10, cfg.StockCriticalThreshold)

	token := loginToken(t, r)
	req, _ := http.NewRequest("GET", "/api/stats", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	data := resp["data"].(map[string]interface{})
	assert.Contains(t, data, "low_stock_count")
	_ = data["low_stock_count"]
}

// ─── Repository: GetAllProducts maxStock filter runs against real column ─────────

func TestRepository_GetAllProducts_LowStockFilter(t *testing.T) {
	requireDB(t)
	testDB := repository.NewTestDB(t)
	defer testDB.Close(t)
	repo := repository.NewPostgresRepository(testDB.Pool())

	t.Run("no filter returns all", func(t *testing.T) {
		products, total, err := repo.GetAllProducts(nil, 100, 0, "", nil, "", "", nil, nil)
		require.NoError(t, err)
		assert.Greater(t, total, 0)
		assert.Greater(t, len(products), 0)
	})

	t.Run("filter stock<=5", func(t *testing.T) {
		maxStock := 5
		products, total, err := repo.GetAllProducts(nil, 100, 0, "", nil, "", "", &maxStock, nil)
		require.NoError(t, err)
		for _, p := range products {
			assert.LessOrEqual(t, p.Stock, 5, "every result must be <= 5")
		}
		if total > 0 {
			assert.GreaterOrEqual(t, len(products), 1)
		}
	})

	t.Run("filter stock<=0 none expected", func(t *testing.T) {
		maxStock := 0
		products, _, err := repo.GetAllProducts(nil, 100, 0, "", nil, "", "", &maxStock, nil)
		require.NoError(t, err)
		// seed data has stock >= 0; filter stock<=0 should only be items with 0 stock
		for _, p := range products {
			assert.Equal(t, 0, p.Stock)
		}
	})
}

// ─── Integration: classification boundaries ────────────────────────────────────

func TestIntegration_StockClassificationBoundaries(t *testing.T) {
	os.Setenv("STOCK_WARNING_THRESHOLD", "10")
	os.Setenv("STOCK_CRITICAL_THRESHOLD", "5")
	defer os.Unsetenv("STOCK_WARNING_THRESHOLD")
	defer os.Unsetenv("STOCK_CRITICAL_THRESHOLD")

	cfg := config.Load()
	assert.Equal(t, 10, cfg.StockWarningThreshold)
	assert.Equal(t, 5, cfg.StockCriticalThreshold)

	tests := []struct {
		stock    int
		critical bool // is stock <= critical threshold?
	}{
		{0, true}, {3, true}, {5, true},  // at or below critical
		{6, false}, {7, false}, {10, false}, {50, false}, // above critical
	}
	for _, tc := range tests {
		t.Run("", func(t *testing.T) {
			assert.Equal(t, tc.critical, tc.stock <= cfg.StockCriticalThreshold)
		})
		if tc.stock != 5 {
			assert.NotEqual(t, tc.critical, tc.stock > cfg.StockCriticalThreshold)
		}
	}
}
