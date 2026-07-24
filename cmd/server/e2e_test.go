package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/brand"
	"retail-pos-system/internal/category"
	"retail-pos-system/internal/config"
	"retail-pos-system/internal/customer"
	"retail-pos-system/internal/customergroup"
	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/inventory"
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/pricing"
	"retail-pos-system/internal/product"
	"retail-pos-system/internal/report"
	"retail-pos-system/internal/sale"
	"retail-pos-system/internal/shared"
	"retail-pos-system/internal/store"
	"retail-pos-system/internal/uom"
	"retail-pos-system/internal/user"
	"retail-pos-system/pkg/websocket"
)

type authAdapter struct {
	svc *user.AuthService
}

func (a *authAdapter) ValidateToken(tokenString string) (*websocket.Claims, error) {
	claims, err := a.svc.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}
	return &websocket.Claims{
		ID:       claims.ID,
		Role:     claims.Role,
		StoreID:  claims.StoreID,
		Username: claims.Username,
	}, nil
}

var e2ePool *pgxpool.Pool

func TestMain(m *testing.M) {
	pool, err := shared.NewTestDB()
	if err != nil {
		os.Exit(0)
	}
	e2ePool = pool

	if err := shared.RunMigrations(pool, "../../database/migrations"); err != nil {
		e2ePool.Close()
		os.Exit(0)
	}

	// Seed the superadmin user in case migrations ran previously and data was truncated
	_, _ = pool.Exec(context.Background(),
		`INSERT INTO users (username, email, password_hash, role_id, is_active)
		 VALUES ('superadmin', 'superadmin@retailpos.local', crypt('admin123', gen_salt('bf', 14)), (SELECT id FROM roles WHERE name='superadmin'), true)
		 ON CONFLICT (username) DO NOTHING`)

	// Re-assign all permissions to superadmin (migration 012 removes dot-notation permissions)
	_, _ = pool.Exec(context.Background(),
		`INSERT INTO role_permissions (role_id, permission_id)
		 SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
		 WHERE r.name = 'superadmin'
		 ON CONFLICT DO NOTHING`)

	// Add colon-notation permissions used by handlers but missing from migrations
	for _, code := range []string{
		"user:read", "user:create", "user:update", "user:delete",
		"role:read", "role:create", "role:update", "role:delete",
		"product:create", "product:update", "product:delete",
		"category:read", "category:create",
		"customer:read", "customer:create", "customer:update", "customer:delete",
		"audit:read",
		"dashboard:read",
		"report:read",
		"inventory:adjust",
		"sale:create", "sale:read",
		"customer_group:read", "customer_group:create", "customer_group:update", "customer_group:delete",
		"store:read", "store:create", "store:update", "store:delete",
		"pricing:read", "pricing:create", "pricing:update", "pricing:delete",
	} {
		_, _ = pool.Exec(context.Background(),
			`INSERT INTO permissions (code, name, description) VALUES ($1, $2, $3) ON CONFLICT (code) DO NOTHING`, code, code, code)
	}
	_, _ = pool.Exec(context.Background(),
		`INSERT INTO role_permissions (role_id, permission_id)
		 SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
		 WHERE r.name = 'superadmin'
		 ON CONFLICT DO NOTHING`)

	code := m.Run()
	e2ePool.Close()
	os.Exit(code)
}

func setupE2ERouter(t *testing.T) *gin.Engine {
	t.Helper()
	if e2ePool == nil {
		t.Skip("no database connection")
	}
	gin.SetMode(gin.TestMode)

	bus := eventbus.New()
	go bus.Run()
	t.Cleanup(bus.Shutdown)

	userRepo := user.NewRepository(e2ePool)
	productRepo := product.NewRepository(e2ePool)
	saleRepo := sale.NewRepository(e2ePool)
	inventoryRepo := inventory.NewRepository(e2ePool)
	customerRepo := customer.NewRepository(e2ePool)
	categoryRepo := category.NewRepository(e2ePool)
	brandRepo := brand.NewRepository(e2ePool)
	uomRepo := uom.NewRepository(e2ePool)
	auditRepo := audit.NewRepository(e2ePool)
	reportRepo := report.NewRepository(e2ePool)
	cgRepo := customergroup.NewRepository(e2ePool)
	storeRepo := store.NewRepository(e2ePool)
	pricingRepo := pricing.NewRepository(e2ePool)

	userSvc := user.NewService(userRepo)
	authSvc := user.NewAuthService(userRepo, nil, config.Load())
	productSvc := product.NewService(productRepo, categoryRepo, brandRepo, uomRepo, bus)
	brandSvc := brand.NewService(brandRepo)
	uomSvc := uom.NewService(uomRepo)
	saleSvc := sale.NewService(saleRepo, bus)
	inventorySvc := inventory.NewService(inventoryRepo, bus)
	customerSvc := customer.NewService(customerRepo)
	categorySvc := category.NewService(categoryRepo)
	auditSvc := audit.NewService(auditRepo)
	reportSvc := report.NewService(reportRepo, bus)
	cgSvc := customergroup.NewService(cgRepo)
	storeSvc := store.NewService(storeRepo)
	resolver := pricing.NewResolver(pricingRepo)
	pricingSvc := pricing.NewService(pricingRepo)

	userH := user.NewHandler(userSvc, nil)
	authH := user.NewAuthHandler(authSvc, nil)
	productH := product.NewHandler(productSvc, nil)
	saleH := sale.NewHandler(saleSvc, nil)
	inventoryH := inventory.NewHandler(inventorySvc, nil)
	customerH := customer.NewHandler(customerSvc, nil)
	categoryH := category.NewHandler(categorySvc, nil)
	brandH := brand.NewHandler(brandSvc, nil)
	uomH := uom.NewHandler(uomSvc, nil)
	auditH := audit.NewHandler(auditSvc)
	reportH := report.NewHandler(reportSvc)
	cgH := customergroup.NewHandler(cgSvc, nil)
	storeH := store.NewHandler(storeSvc, nil)
	pricingH := pricing.NewHandler(pricingSvc, resolver, nil)
	pricingH.SetProductSearcher(pricingRepo)

	hub := websocket.NewHub(&authAdapter{authSvc})
	go hub.Run()
	t.Cleanup(hub.Shutdown)

	r := gin.New()
	r.Use(gin.Recovery())

	authMiddleware := middleware.NewModularAuthMiddleware(authSvc)
	permMiddleware := middleware.RequirePermission

	noopCSRF := func(c *gin.Context) { c.Next() }
	noopRateLimit := func(c *gin.Context) { c.Next() }
	authH.RegisterLoginRoute(r.Group("/api"), noopRateLimit)
	authH.RegisterRoutes(r.Group("/api"), authMiddleware, noopCSRF, permMiddleware)

	protected := r.Group("/api")
	protected.Use(authMiddleware)
	{
		productH.RegisterRoutes(protected, authMiddleware, permMiddleware)
		saleH.RegisterRoutes(protected, authMiddleware, permMiddleware)
		inventoryH.RegisterRoutes(protected, authMiddleware, permMiddleware)
		customerH.RegisterRoutes(protected, authMiddleware, permMiddleware)
		categoryH.RegisterRoutes(protected, authMiddleware, permMiddleware)
		userH.RegisterRoutes(protected, authMiddleware, permMiddleware)
		auditH.RegisterRoutes(protected, authMiddleware, permMiddleware)
		reportH.RegisterRoutes(protected, authMiddleware, permMiddleware)
		brandH.RegisterRoutes(protected, authMiddleware, permMiddleware)
		uomH.RegisterRoutes(protected, authMiddleware, permMiddleware)
		cgH.RegisterRoutes(protected, authMiddleware, permMiddleware)
		storeH.RegisterRoutes(protected, authMiddleware, permMiddleware)
		pricingH.RegisterRoutes(protected, authMiddleware, permMiddleware)
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return r
}

func loginAs(t *testing.T, r *gin.Engine, username, password string) string {
	t.Helper()
	body := `{"username":"` + username + `","password":"` + password + `"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("login failed: %d %s", w.Code, w.Body.String())
	}

	var resp struct {
		AccessToken string `json:"access_token"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.NotEmpty(t, resp.AccessToken)
	return resp.AccessToken
}

func TestE2E_Health(t *testing.T) {
	router := setupE2ERouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "ok", resp["status"])
}

func TestE2E_Login(t *testing.T) {
	router := setupE2ERouter(t)

	t.Run("valid credentials", func(t *testing.T) {
		body := `{"username":"superadmin","password":"admin123"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			AccessToken string `json:"access_token"`
			User        struct {
				ID       int    `json:"id"`
				Username string `json:"username"`
				Role     struct {
					ID   int    `json:"id"`
					Name string `json:"name"`
				} `json:"role"`
			} `json:"user"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.AccessToken)
		assert.Equal(t, "superadmin", resp.User.Username)
		assert.Equal(t, "superadmin", resp.User.Role.Name)

		cookies := w.Result().Cookies()
		var refreshTokenFound bool
		for _, c := range cookies {
			if c.Name == "refresh_token" {
				refreshTokenFound = true
				assert.NotEmpty(t, c.Value)
			}
		}
		assert.True(t, refreshTokenFound, "refresh_token cookie should be set")
	})

	t.Run("invalid password", func(t *testing.T) {
		body := `{"username":"superadmin","password":"wrongpassword"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("nonexistent user", func(t *testing.T) {
		body := `{"username":"nobody","password":"admin123"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestE2E_UnauthenticatedAccess(t *testing.T) {
	router := setupE2ERouter(t)

	endpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/api/admin/users"},
		{"GET", "/api/products?id=1"},
		{"POST", "/api/products"},
		{"GET", "/api/sales"},
		{"GET", "/api/dashboard/stats"},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(ep.method, ep.path, nil)
			if ep.method == "POST" {
				req.Body = http.NoBody
			}
			router.ServeHTTP(w, req)

			if ep.method == "GET" && (ep.path == "/api/products" || ep.path == "/api/dashboard/stats") {
				return
			}
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

func TestE2E_AuthenticatedCRUD(t *testing.T) {
	router := setupE2ERouter(t)
	token := loginAs(t, router, "superadmin", "admin123")

	t.Run("list users", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/admin/users", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data  []user.User `json:"data"`
			Total int         `json:"total"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, resp.Total, 1)
	})

	t.Run("create product", func(t *testing.T) {
		sku := fmt.Sprintf("E2E-SKU-%d", time.Now().UnixNano())
		body := fmt.Sprintf(`{"sku":"%s","name":"E2E Product","price":25000,"cost":15000,"stock":100,"status":"active"}`, sku)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/products", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp struct {
			Data product.Product `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "E2E Product", resp.Data.Name)
		assert.Greater(t, resp.Data.ID, 0)
	})

	t.Run("list products", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/products?limit=5", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data  []product.Product `json:"data"`
			Total int               `json:"total"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotNil(t, resp.Data)
	})
}

func TestE2E_ValidateSession(t *testing.T) {
	router := setupE2ERouter(t)
	token := loginAs(t, router, "superadmin", "admin123")

	w := httptest.NewRecorder()
	body := `{}`
	req, _ := http.NewRequest("POST", "/api/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		User struct {
			ID       int    `json:"id"`
			Username string `json:"username"`
			Role     string `json:"role"`
		} `json:"user"`
		Permissions []string `json:"permissions"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Greater(t, resp.User.ID, 0)
	assert.Equal(t, "superadmin", resp.User.Username)
	assert.Equal(t, "superadmin", resp.User.Role)
	assert.NotEmpty(t, resp.Permissions)
}

func TestE2E_BrandCRUD(t *testing.T) {
	router := setupE2ERouter(t)
	token := loginAs(t, router, "superadmin", "admin123")

	var brandID int

	t.Run("create brand", func(t *testing.T) {
		body := `{"name":"E2E Brand","description":"e2e test","is_active":true}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/brands", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp struct {
			Data brand.Brand `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "E2E Brand", resp.Data.Name)
		assert.Greater(t, resp.Data.ID, 0)
		brandID = resp.Data.ID
	})

	t.Run("update brand", func(t *testing.T) {
		body := `{"name":"E2E Brand Updated","description":"updated","is_active":true}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/api/brands/"+strconv.Itoa(brandID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("delete brand", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/brands/"+strconv.Itoa(brandID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestE2E_UnitOfMeasureCRUD(t *testing.T) {
	router := setupE2ERouter(t)
	token := loginAs(t, router, "superadmin", "admin123")

	var uomID int

	t.Run("create unit of measure", func(t *testing.T) {
		body := `{"code":"E2EUM","name":"E2E UOM","is_active":true}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/units-of-measure", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp struct {
			Data uom.UnitOfMeasure `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "E2E UOM", resp.Data.Name)
		assert.Greater(t, resp.Data.ID, 0)
		uomID = resp.Data.ID
	})

	t.Run("update unit of measure", func(t *testing.T) {
		body := `{"code":"E2EUM","name":"E2E UOM Updated","is_active":true}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/api/units-of-measure/"+strconv.Itoa(uomID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("delete unit of measure", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/units-of-measure/"+strconv.Itoa(uomID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestE2E_PublicEndpoints(t *testing.T) {
	router := setupE2ERouter(t)

	t.Run("available years with auth", func(t *testing.T) {
		token := loginAs(t, router, "superadmin", "admin123")
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/dashboard/years", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestE2E_TokenRefresh(t *testing.T) {
	router := setupE2ERouter(t)
	_ = loginAs(t, router, "superadmin", "admin123")
}

func TestE2E_Logout(t *testing.T) {
	router := setupE2ERouter(t)
	token := loginAs(t, router, "superadmin", "admin123")

	w := httptest.NewRecorder()
	body := `{}`
	req, _ := http.NewRequest("POST", "/api/logout", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestE2E_CustomerGroupCRUD(t *testing.T) {
	router := setupE2ERouter(t)
	token := loginAs(t, router, "superadmin", "admin123")

	var cgID int

	t.Run("create customer group", func(t *testing.T) {
		body := `{"name":"VIP Customers","description":"High-value customers"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/customer-groups", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp struct {
			Data customergroup.CustomerGroup `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "VIP Customers", resp.Data.Name)
		assert.True(t, resp.Data.IsActive)
		cgID = resp.Data.ID
	})

	t.Run("list customer groups", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/customer-groups", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("get customer group by id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/customer-groups/"+strconv.Itoa(cgID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data customergroup.CustomerGroup `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "VIP Customers", resp.Data.Name)
	})

	t.Run("update customer group", func(t *testing.T) {
		body := `{"name":"VIP Customers Updated","description":"Updated description"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/api/customer-groups/"+strconv.Itoa(cgID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("delete customer group", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/customer-groups/"+strconv.Itoa(cgID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestE2E_StoreCRUD(t *testing.T) {
	router := setupE2ERouter(t)
	token := loginAs(t, router, "superadmin", "admin123")

	var storeID int

	t.Run("create store", func(t *testing.T) {
		body := `{"name":"E2E Store","address":"123 Test St","phone":"08123456789"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/stores", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp struct {
			Data store.Store `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "E2E Store", resp.Data.Name)
		assert.True(t, resp.Data.IsActive)
		storeID = resp.Data.ID
	})

	t.Run("list stores", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/stores", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("list active stores", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/stores/active", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("get store by id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/stores/"+strconv.Itoa(storeID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data store.Store `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "E2E Store", resp.Data.Name)
	})

	t.Run("update store", func(t *testing.T) {
		body := `{"name":"E2E Store Updated","address":"456 Updated Ave"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/api/stores/"+strconv.Itoa(storeID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("delete store", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/stores/"+strconv.Itoa(storeID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestE2E_PricingRulesCRUD(t *testing.T) {
	router := setupE2ERouter(t)
	token := loginAs(t, router, "superadmin", "admin123")

	var ruleID int
	var productID int

	// Create a product to target with pricing rules
	sku := fmt.Sprintf("RULE-%d", time.Now().UnixNano())
	prodBody := fmt.Sprintf(`{"sku":"%s","name":"Rule Test Product","price":50000,"cost":30000,"stock":100,"status":"active"}`, sku)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/products", strings.NewReader(prodBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	var prodResp struct {
		Data product.Product `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &prodResp))
	productID = prodResp.Data.ID

	t.Run("create pricing rule with discount_percent", func(t *testing.T) {
		body := fmt.Sprintf(`{"name":"E2E Discount Rule","pricing_type":"promotion","pricing_method":"discount_percent","pricing_value":10,"priority":100,"minimum_quantity":1,"product_id":%d,"is_active":true}`, productID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/pricing-rules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp struct {
			Data pricing.PricingRule `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "E2E Discount Rule", resp.Data.Name)
		assert.Equal(t, pricing.PricingTypePromotion, resp.Data.PricingType)
		assert.Equal(t, pricing.PricingMethodDiscountPct, resp.Data.PricingMethod)
		assert.Equal(t, 10.0, resp.Data.PricingValue)
		ruleID = resp.Data.ID
	})

	t.Run("list pricing rules", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/pricing-rules", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("get pricing rule by id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/pricing-rules/"+strconv.Itoa(ruleID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data pricing.PricingRule `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, ruleID, resp.Data.ID)
	})

	t.Run("update pricing rule", func(t *testing.T) {
		body := fmt.Sprintf(`{"name":"E2E Discount Rule Updated","pricing_type":"promotion","pricing_method":"discount_percent","pricing_value":15,"priority":100,"minimum_quantity":1,"product_id":%d,"is_active":true}`, productID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/api/pricing-rules/"+strconv.Itoa(ruleID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("delete pricing rule", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/pricing-rules/"+strconv.Itoa(ruleID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestE2E_PricingResolver(t *testing.T) {
	router := setupE2ERouter(t)
	token := loginAs(t, router, "superadmin", "admin123")

	// Create a product to use in resolution
	var productID int
	sku := fmt.Sprintf("RES-%d", time.Now().UnixNano())
	prodBody := fmt.Sprintf(`{"sku":"%s","name":"Resolver Test Product","price":50000,"cost":30000,"stock":100,"status":"active"}`, sku)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/products", strings.NewReader(prodBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	var prodResp struct {
		Data product.Product `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &prodResp))
	productID = prodResp.Data.ID

	t.Run("resolve with no rules returns base price", func(t *testing.T) {
		body := fmt.Sprintf(`{"items":[{"product_id":%d,"quantity":1}]}`, productID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/pricing/resolve", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data []pricing.ResolvedPrice `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Len(t, resp.Data, 1)
		assert.Equal(t, 50000, resp.Data[0].UnitPrice)
		assert.Equal(t, 50000, resp.Data[0].OriginalPrice)
	})

	t.Run("resolve with discount_percent rule", func(t *testing.T) {
		// Create a promotion rule
		ruleBody := fmt.Sprintf(`{"name":"Resolve Test Discount","pricing_type":"promotion","pricing_method":"discount_percent","pricing_value":20,"priority":100,"minimum_quantity":1,"product_id":%d,"is_active":true}`, productID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/pricing-rules", strings.NewReader(ruleBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		// Now resolve
		body := fmt.Sprintf(`{"items":[{"product_id":%d,"quantity":1}]}`, productID)
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/api/pricing/resolve", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data []pricing.ResolvedPrice `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Len(t, resp.Data, 1)
		assert.Equal(t, 40000, resp.Data[0].UnitPrice) // 50000 - 20% = 40000
		assert.Equal(t, 50000, resp.Data[0].OriginalPrice)
		assert.Equal(t, 10000, resp.Data[0].Discount)
	})

	t.Run("resolve with empty items returns 400", func(t *testing.T) {
		body := `{"items":[]}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/pricing/resolve", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestE2E_ProductSearch(t *testing.T) {
	router := setupE2ERouter(t)
	token := loginAs(t, router, "superadmin", "admin123")

	// Create a product with a known name
	sku := fmt.Sprintf("SEARCH-%d", time.Now().UnixNano())
	prodBody := fmt.Sprintf(`{"sku":"%s","name":"Searchable Widget","price":10000,"cost":5000,"stock":50,"status":"active"}`, sku)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/products", strings.NewReader(prodBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	t.Run("search by name returns matching products", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/products/search?q=Searchable&limit=10", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data []pricing.ProductSearchResult `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(resp.Data), 1)
		assert.Equal(t, "Searchable Widget", resp.Data[0].Name)
	})

	t.Run("search with no results returns empty array", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/products/search?q=ZZZNONEXISTENT&limit=10", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data []pricing.ProductSearchResult `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Empty(t, resp.Data)
	})
}

func init() {
	gin.SetMode(gin.TestMode)
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.UTC
	}
	time.Local = loc
}
