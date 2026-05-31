package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"retail-pos-system/internal/auth"
	"retail-pos-system/internal/delivery/http/handler"
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestServer(t *testing.T) (*gin.Engine, *repository.TestDB) {
	t.Helper()

	// Set up test database
	testDB := repository.NewTestDB(t)

	// Create repositories
	userRepo := repository.NewPostgresRepository(testDB.Pool())
	roleRepo := repository.NewPostgresRepository(testDB.Pool())
	productRepo := repository.NewPostgresRepository(testDB.Pool())
	saleRepo := repository.NewPostgresRepository(testDB.Pool())
	auditRepo := repository.NewPostgresRepository(testDB.Pool())

	// Create services
	authService := auth.NewAuthService(userRepo, testDB.Pool())

	// Create handler
	h := handler.NewHandler(
		userRepo,
		roleRepo,
		productRepo,
		saleRepo,
		authService,
		nil, // wsHub not needed for API tests
		auditRepo,
	)

	// Set up Gin router
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Public routes
	r.POST("/api/login", h.Login)
	r.POST("/api/refresh", h.RefreshToken)

	// Protected routes
	protected := r.Group("/api")
	protected.Use(func(c *gin.Context) {
		c.Set("authService", authService)
		c.Next()
	})
	protected.Use(middleware.AuthMiddleware())

	// Products
	protected.GET("/products", h.GetProducts)
	protected.POST("/products", h.CreateProduct)
	protected.GET("/products/:id", h.GetProductByID)
	protected.PUT("/products/:id", h.UpdateProduct)
	protected.DELETE("/products/:id", h.DeleteProduct)

	// Sales
	protected.POST("/sales", h.CreateSale)
	protected.GET("/sales", h.GetSalesHistory)
	protected.GET("/sales/:id", h.GetSaleByID)

// Stats
 	protected.GET("/stats", h.GetDashboardStats)
 	protected.GET("/dashboard/chart", h.GetSalesChartData)
 	protected.GET("/dashboard/chart/weekly", h.GetSalesWeeklyReport)
 	protected.GET("/dashboard/chart/monthly", h.GetSalesMonthlyReport)
 	protected.GET("/dashboard/comparison", h.GetPeriodComparison)
 	protected.GET("/dashboard/years", h.GetAvailableYears)

	return r, testDB
}

func TestAPI_Login_Success(t *testing.T) {
	r, testDB := setupTestServer(t)
	defer testDB.Close(t)

	loginReq := map[string]string{
		"username": "superadmin",
		"password": "admin123",
	}
	reqBody, _ := json.Marshal(loginReq)

	req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "access_token")
	assert.Contains(t, response, "refresh_token")
	assert.Contains(t, response, "user")

	user := response["user"].(map[string]interface{})
	assert.Equal(t, "superadmin", user["username"])
	assert.Equal(t, float64(1), user["id"])
}

func TestAPI_Login_InvalidCredentials(t *testing.T) {
	r, testDB := setupTestServer(t)
	defer testDB.Close(t)

	loginReq := map[string]string{
		"username": "superadmin",
		"password": "wrongpass",
	}
	reqBody, _ := json.Marshal(loginReq)

	req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "error")
	assert.Contains(t, response["error"].(string), "invalid username or password")
}

func TestAPI_GetProducts_Unauthorized(t *testing.T) {
	r, testDB := setupTestServer(t)
	defer testDB.Close(t)

	req, _ := http.NewRequest("GET", "/api/products", nil)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "error")
	assert.Contains(t, response["error"].(string), "authorization token required")
}

func TestAPI_GetProducts_Authorized(t *testing.T) {
	r, testDB := setupTestServer(t)
	defer testDB.Close(t)

	// First login to get token
	loginReq := map[string]string{
		"username": "superadmin",
		"password": "admin123",
	}
	reqBody, _ := json.Marshal(loginReq)

	req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var loginResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &loginResponse)
	require.NoError(t, err)

	token := loginResponse["access_token"].(string)

	// Now test protected endpoint
	req, _ = http.NewRequest("GET", "/api/products", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "data")
	assert.Contains(t, response, "total")
}

func TestAPI_GetStats_Authorized(t *testing.T) {
	r, testDB := setupTestServer(t)
	defer testDB.Close(t)

	// Login first
	loginReq := map[string]string{
		"username": "superadmin",
		"password": "admin123",
	}
	reqBody, _ := json.Marshal(loginReq)

	req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var loginResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &loginResponse)
	require.NoError(t, err)

	token := loginResponse["access_token"].(string)

	// Test stats endpoint
	req, _ = http.NewRequest("GET", "/api/stats", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].(map[string]interface{})
	assert.Contains(t, data, "todays_sales")
	assert.Contains(t, data, "todays_revenue")
	assert.Contains(t, data, "total_products")
	assert.Contains(t, data, "low_stock_count")
 }

func TestAPI_GetProducts_WithCategoryFilter(t *testing.T) {
	r, testDB := setupTestServer(t)
	defer testDB.Close(t)

	// First login to get token
	loginReq := map[string]string{
		"username": "superadmin",
		"password": "admin123",
	}
	reqBody, _ := json.Marshal(loginReq)

	req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var loginResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &loginResponse)
	require.NoError(t, err)

	token := loginResponse["access_token"].(string)

	// Test products endpoint with category filter (using Indonesian category name from seed)
	req, _ = http.NewRequest("GET", "/api/products?category=Makanan", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "data")
	assert.Contains(t, response, "total")

	data := response["data"].([]interface{})
	for _, item := range data {
		product := item.(map[string]interface{})
		if categoryName, ok := product["category_name"].(string); ok {
			assert.Equal(t, "Makanan", categoryName)
		}
	}
}

func TestAPI_GetProducts_WithMultipleCategoryFilter(t *testing.T) {
	r, testDB := setupTestServer(t)
	defer testDB.Close(t)

	// First login to get token
	loginReq := map[string]string{
		"username": "superadmin",
		"password": "admin123",
	}
	reqBody, _ := json.Marshal(loginReq)

	req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var loginResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &loginResponse)
	require.NoError(t, err)

	token := loginResponse["access_token"].(string)

	// Test products endpoint with multiple categories (comma-separated, using Indonesian names)
	req, _ = http.NewRequest("GET", "/api/products?category=Makanan,Minuman", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "data")
	assert.Contains(t, response, "total")
}

