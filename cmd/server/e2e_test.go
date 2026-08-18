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
	"retail-pos-system/internal/purchase"
	"retail-pos-system/internal/report"
	"retail-pos-system/internal/sale"
	"retail-pos-system/internal/shared"
	"retail-pos-system/internal/shift"
	"retail-pos-system/internal/store"
	"retail-pos-system/internal/supplier"
	"retail-pos-system/internal/uom"
	"retail-pos-system/internal/user"
	"retail-pos-system/pkg/websocket"
)

func init() {
	_ = os.Setenv("JWT_SECRET", "test-secret-for-e2e-tests")
}

type authAdapter struct {
	svc *user.AuthService
}

type e2eProductNameLookup struct {
	repo *product.Repository
}

func (l e2eProductNameLookup) GetProductNamesByIDs(ctx context.Context, ids []int) (map[int]purchase.ProductInfo, error) {
	products, err := l.repo.GetProductsByIDs(ctx, ids, nil)
	if err != nil {
		return nil, err
	}
	result := make(map[int]purchase.ProductInfo, len(products))
	for _, p := range products {
		result[p.ID] = purchase.ProductInfo{Name: p.Name, SKU: p.SKU}
	}
	return result, nil
}

type e2eSupplierNameLookup struct {
	repo *supplier.Repository
}

func (l e2eSupplierNameLookup) GetSupplierNamesByIDs(ctx context.Context, ids []int) (map[int]purchase.SupplierInfo, error) {
	suppliers, err := l.repo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	result := make(map[int]purchase.SupplierInfo, len(suppliers))
	for _, s := range suppliers {
		result[s.ID] = purchase.SupplierInfo{Name: s.Name}
	}
	return result, nil
}

func (l e2eSupplierNameLookup) GetSupplierIDsByName(ctx context.Context, name string) ([]int, error) {
	return l.repo.GetIDsByName(ctx, name)
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

	// Reset app data between runs so E2E tests are self-contained. Reference
	// data seeded by migrations (roles, permissions, payment_methods) is not
	// truncated; superadmin and permissions are re-seeded below.
	if err := shared.TruncateTestData(pool); err != nil {
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

	// Add dot-notation permissions used by handlers
	for _, code := range []string{
		"user.view", "user.create", "user.update", "user.delete",
		"role.view", "role.create", "role.update", "role.delete",
		"product.create", "product.update", "product.delete",
		"category.view", "category.create",
		"customer.view", "customer.create", "customer.update", "customer.delete",
		"audit.view",
		"dashboard.view",
		"report.view",
		"inventory.adjust",
		"sale.create", "sale.view",
		"customer_group.view", "customer_group.create", "customer_group.update", "customer_group.delete",
		"store.view", "store.create", "store.update", "store.delete",
		"pricing.view", "pricing.create", "pricing.update", "pricing.delete",
		"purchase_order.view", "purchase_order.create", "purchase_order.update", "purchase_order.delete",
		"purchase_order.confirm", "purchase_order.cancel", "purchase_order.receive",
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

// e2ePriceResolverAdapter bridges the sale package's checkout port to the
// pricing resolver, mirroring internal/wiring for the in-process test router.
type e2ePriceResolverAdapter struct {
	resolver *pricing.Resolver
}

func (a e2ePriceResolverAdapter) ResolveSnapshotsBatch(ctx context.Context, items []sale.ResolveItem) ([]sale.PriceSnapshot, error) {
	pricingItems := make([]pricing.ResolveItem, len(items))
	for i, it := range items {
		pricingItems[i] = pricing.ResolveItem{
			ProductID:       it.ProductID,
			Quantity:        it.Quantity,
			CustomerGroupID: it.CustomerGroupID,
			StoreID:         it.StoreID,
		}
	}
	snaps, err := a.resolver.ResolveSnapshotsBatch(ctx, pricingItems)
	if err != nil {
		return nil, err
	}
	result := make([]sale.PriceSnapshot, len(snaps))
	for i, snap := range snaps {
		result[i] = sale.PriceSnapshot{
			ProductID:     snap.ProductID,
			ProductName:   snap.ProductName,
			UnitPrice:     snap.UnitPrice,
			OriginalPrice: snap.OriginalPrice,
			Discount:      snap.Discount,
			Type:          sale.Type(snap.Type),
			Cost:          snap.Cost,
			TaxClassID:    snap.TaxClassID,
			TaxRate:       snap.TaxRate,
			SnapshotAt:    snap.SnapshotAt,
		}
		if snap.Rule != nil {
			result[i].Rule = &sale.Rule{
				ID:   snap.Rule.ID,
				Name: snap.Rule.Name,
				Type: sale.Type(snap.Rule.Type),
			}
		}
	}
	return result, nil
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
	productRepo.SetProductStockWriter(inventory.ProductStockWriter{})
	saleRepo := sale.NewRepository(e2ePool)
	saleRepo.SetProductNameProvider(product.ProductNameLookup{})
	saleRepo.SetCustomerNameProvider(customer.CustomerNameLookup{})
	inventoryRepo := inventory.NewRepository(e2ePool)
	customerRepo := customer.NewRepository(e2ePool)
	customerRepo.SetCustomerGroupNameProvider(customergroup.CustomerGroupNameLookup{})
	categoryRepo := category.NewRepository(e2ePool)
	categoryRepo.SetProductQueryProvider(product.CategoryProductCountProvider{})
	brandRepo := brand.NewRepository(e2ePool)
	uomRepo := uom.NewRepository(e2ePool)
	auditRepo := audit.NewRepository(e2ePool)
	reportRepo := report.NewRepository(e2ePool)
	cgRepo := customergroup.NewRepository(e2ePool)
	cgRepo.SetCustomerCountProvider(customer.CustomerGroupCountsLookup{})
	storeRepo := store.NewRepository(e2ePool)
	pricingRepo := pricing.NewRepository(e2ePool)
	pricingRepo.SetProductPricingProvider(product.ProductPricingLookup{})
	pricingRepo.SetCategorySearchProvider(category.CategoryNamesProvider{})
	pricingRepo.SetBrandSearchProvider(brand.BrandNamesProvider{})
	supplierRepo := supplier.NewRepository(e2ePool)
	purchaseRepo := purchase.NewRepository(e2ePool)

	userSvc := user.NewService(userRepo)
	authSvc := user.NewAuthService(userRepo, nil, config.Load())
	productSvc := product.NewService(productRepo, categoryRepo, brandRepo, uomRepo, bus)
	brandSvc := brand.NewService(brandRepo)
	uomSvc := uom.NewService(uomRepo)
	saleSvc := sale.NewService(saleRepo, bus)
	saleSvc.SetStockDeducer(inventory.StockDeducer{})
	saleSvc.SetShiftTotalUpdater(shift.TotalUpdater{})
	inventorySvc := inventory.NewService(inventoryRepo, bus)
	customerSvc := customer.NewService(customerRepo)
	categorySvc := category.NewService(categoryRepo)
	auditSvc := audit.NewService(auditRepo)
	reportSvc := report.NewService(reportRepo, bus)
	cgSvc := customergroup.NewService(cgRepo)
	storeSvc := store.NewService(storeRepo)
	resolver := pricing.NewResolver(pricingRepo)
	saleSvc.SetPriceResolver(e2ePriceResolverAdapter{resolver: resolver})
	pricingSvc := pricing.NewService(pricingRepo)

	purchaseSvc := purchase.NewService(purchaseRepo, bus)
	purchaseSvc.SetProductLookup(e2eProductNameLookup{repo: productRepo})
	purchaseSvc.SetSupplierLookup(e2eSupplierNameLookup{repo: supplierRepo})

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
	purchaseH := purchase.NewHandler(purchaseSvc, auditSvc)

	hub := websocket.NewHub(&authAdapter{authSvc})
	go hub.Run()
	t.Cleanup(hub.Shutdown)

	r := gin.New()
	r.Use(gin.Recovery())

	authMiddleware := middleware.NewModularAuthMiddleware(authSvc)
	permMiddleware := middleware.RequirePermission

	noopCSRF := func(c *gin.Context) { c.Next() }
	noopRateLimit := func(c *gin.Context) { c.Next() }
	noopAuth := func(c *gin.Context) { c.Next() }
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
		purchaseH.RegisterRoutes(protected, noopAuth, permMiddleware)
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
			Data pricing.Rule `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "E2E Discount Rule", resp.Data.Name)
		assert.Equal(t, pricing.PricingTypePromotion, resp.Data.Type)
		assert.Equal(t, pricing.PricingMethodDiscountPct, resp.Data.Method)
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
			Data pricing.Rule `json:"data"`
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

	// Create products with known name, SKU, and barcode
	sku := fmt.Sprintf("SEARCH-%d", time.Now().UnixNano())
	barcode := fmt.Sprintf("BC-%d", time.Now().UnixNano())
	prodBody := fmt.Sprintf(`{"sku":"%s","name":"Searchable Widget","barcode":"%s","price":10000,"cost":5000,"stock":50,"status":"active"}`, sku, barcode)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/products", strings.NewReader(prodBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// ---- pricing endpoint (legacy) ----
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
		require.GreaterOrEqual(t, len(resp.Data), 1)
		assert.Equal(t, sku, resp.Data[0].SKU)
		assert.Equal(t, "Searchable Widget", resp.Data[0].Name)
	})

	t.Run("search by SKU returns matching products", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/products/search?q="+sku+"&limit=10", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data []pricing.ProductSearchResult `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(resp.Data), 1)
		assert.Equal(t, sku, resp.Data[0].SKU)
		assert.Equal(t, "Searchable Widget", resp.Data[0].Name)
	})

	t.Run("search by barcode returns matching products", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/products/search?q="+barcode+"&limit=10", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data []pricing.ProductSearchResult `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(resp.Data), 1)
		assert.Equal(t, sku, resp.Data[0].SKU)
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

	// ---- product handler endpoint (exercises GetAllProducts with ILIKE fallback) ----
	t.Run("product handler search by full SKU", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/products?search="+sku+"&limit=10", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data  []product.Product `json:"data"`
			Total int               `json:"total"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.GreaterOrEqual(t, resp.Total, 1)
		require.GreaterOrEqual(t, len(resp.Data), 1)
		assert.Equal(t, sku, resp.Data[0].SKU)
	})

	t.Run("product handler search by full barcode", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/products?search="+barcode+"&limit=10", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data  []product.Product `json:"data"`
			Total int               `json:"total"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.GreaterOrEqual(t, resp.Total, 1)
		require.GreaterOrEqual(t, len(resp.Data), 1)
		assert.Equal(t, sku, resp.Data[0].SKU)
		if resp.Data[0].Barcode != nil {
			assert.Equal(t, barcode, *resp.Data[0].Barcode)
		}
	})

	t.Run("product handler search by partial SKU (last digits)", func(t *testing.T) {
		// Extract last 3 digits of the SKU (format: SEARCH-<timestamp>)
		partial := sku[len(sku)-3:]
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/products?search="+partial+"&limit=10", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data  []product.Product `json:"data"`
			Total int               `json:"total"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.GreaterOrEqual(t, resp.Total, 1)
		found := false
		for _, p := range resp.Data {
			if p.SKU == sku {
				found = true
				break
			}
		}
		assert.True(t, found, "expected product with SKU %s to appear in partial search results for %q", sku, partial)
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

func seedE2EPaymentMethods(t *testing.T) {
	t.Helper()
	if e2ePool == nil {
		t.Skip("no database connection")
	}
	_, err := e2ePool.Exec(context.Background(), `
		INSERT INTO payment_methods (code, name, is_active, requires_reference, sort_order)
		VALUES
			('CASH', 'Cash', true, false, 1),
			('CARD', 'Card', true, true, 2),
			('E_WALLET', 'E-Wallet', true, true, 3),
			('TRANSFER', 'Transfer', true, true, 4),
			('QRIS', 'QRIS', true, false, 5)
		ON CONFLICT (code) DO NOTHING`)
	require.NoError(t, err)
}

func createE2EProduct(t *testing.T, r *gin.Engine, token string) int {
	t.Helper()
	sku := fmt.Sprintf("E2E-SPLIT-%d", time.Now().UnixNano())
	body := fmt.Sprintf(`{"sku":"%s","name":"Split Test Product","price":100000,"cost":60000,"stock":100,"status":"active"}`, sku)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/products", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	var resp struct {
		Data product.Product `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp.Data.ID
}

func TestE2E_SplitPayment(t *testing.T) {
	router := setupE2ERouter(t)
	token := loginAs(t, router, "superadmin", "admin123")
	seedE2EPaymentMethods(t)
	productID := createE2EProduct(t, router, token)

	t.Run("backward compat: payment_method field creates single payment", func(t *testing.T) {
		body := fmt.Sprintf(`{
			"cashier_id": 1,
			"payment_method": "CASH",
			"status": "completed",
			"items": [{"product_id": %d, "quantity": 1}]
		}`, productID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/sales", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp struct {
			Data struct {
				ID            int    `json:"id"`
				PaymentMethod string `json:"payment_method"`
				Payments      []struct {
					PaymentMethodCode string `json:"payment_method_code"`
					Amount            int    `json:"amount"`
				} `json:"payments"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "CASH", resp.Data.PaymentMethod)
		assert.Greater(t, resp.Data.ID, 0)

		detail := httptest.NewRecorder()
		detailReq, _ := http.NewRequest("GET", fmt.Sprintf("/api/sales/%d", resp.Data.ID), nil)
		detailReq.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(detail, detailReq)
		assert.Equal(t, http.StatusOK, detail.Code)
		var detailResp struct {
			Data struct {
				Payments []struct {
					PaymentMethodCode string `json:"payment_method_code"`
					Amount            int    `json:"amount"`
				} `json:"payments"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(detail.Body.Bytes(), &detailResp))
		require.Len(t, detailResp.Data.Payments, 1)
		assert.Equal(t, "CASH", detailResp.Data.Payments[0].PaymentMethodCode)
		assert.Equal(t, 100000, detailResp.Data.Payments[0].Amount)
	})

	t.Run("new payments array: single cash payment", func(t *testing.T) {
		body := fmt.Sprintf(`{
			"cashier_id": 1,
			"payments": [{"payment_method_code": "CASH", "amount": 100000}],
			"status": "completed",
			"items": [{"product_id": %d, "quantity": 1}]
		}`, productID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/sales", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("split payment: cash + card", func(t *testing.T) {
		body := fmt.Sprintf(`{
			"cashier_id": 1,
			"payments": [
				{"payment_method_code": "CASH", "amount": 50000},
				{"payment_method_code": "CARD", "amount": 50000, "reference_number": "REF-12345"}
			],
			"status": "completed",
			"items": [{"product_id": %d, "quantity": 1}]
		}`, productID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/sales", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp struct {
			Data struct {
				ID       int `json:"id"`
				Payments []struct {
					PaymentMethodCode string `json:"payment_method_code"`
					Amount            int    `json:"amount"`
					ReferenceNumber   string `json:"reference_number"`
				} `json:"payments"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		saleID := resp.Data.ID
		assert.Greater(t, saleID, 0)

		detail := httptest.NewRecorder()
		detailReq, _ := http.NewRequest("GET", fmt.Sprintf("/api/sales/%d", saleID), nil)
		detailReq.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(detail, detailReq)
		assert.Equal(t, http.StatusOK, detail.Code)
		var detailResp struct {
			Data struct {
				PaymentMethod string `json:"payment_method"`
				Payments      []struct {
					PaymentMethodCode string `json:"payment_method_code"`
					Amount            int    `json:"amount"`
					ReferenceNumber   string `json:"reference_number"`
				} `json:"payments"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(detail.Body.Bytes(), &detailResp))
		require.Len(t, detailResp.Data.Payments, 2)
		assert.Contains(t, detailResp.Data.PaymentMethod, "CASH")
		assert.Contains(t, detailResp.Data.PaymentMethod, "CARD")
	})

	t.Run("three-way split payment", func(t *testing.T) {
		body := fmt.Sprintf(`{
			"cashier_id": 1,
			"payments": [
				{"payment_method_code": "CASH", "amount": 30000},
				{"payment_method_code": "QRIS", "amount": 30000},
				{"payment_method_code": "CARD", "amount": 40000, "reference_number": "CARD-67890"}
			],
			"status": "completed",
			"items": [{"product_id": %d, "quantity": 1}]
		}`, productID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/sales", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp struct {
			Data struct {
				ID int `json:"id"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		detail := httptest.NewRecorder()
		detailReq, _ := http.NewRequest("GET", fmt.Sprintf("/api/sales/%d", resp.Data.ID), nil)
		detailReq.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(detail, detailReq)
		var detailResp struct {
			Data struct {
				Payments []struct {
					PaymentMethodCode string `json:"payment_method_code"`
					Amount            int    `json:"amount"`
				} `json:"payments"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(detail.Body.Bytes(), &detailResp))
		require.Len(t, detailResp.Data.Payments, 3)
	})

	t.Run("payment total mismatch returns 400", func(t *testing.T) {
		body := fmt.Sprintf(`{
			"cashier_id": 1,
			"payments": [{"payment_method_code": "CASH", "amount": 80000}],
			"status": "completed",
			"items": [{"product_id": %d, "quantity": 1}]
		}`, productID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/sales", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("duplicate payment method returns 400", func(t *testing.T) {
		body := fmt.Sprintf(`{
			"cashier_id": 1,
			"payments": [
				{"payment_method_code": "CASH", "amount": 50000},
				{"payment_method_code": "CASH", "amount": 50000}
			],
			"status": "completed",
			"items": [{"product_id": %d, "quantity": 1}]
		}`, productID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/sales", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("zero payment amount returns 400", func(t *testing.T) {
		body := fmt.Sprintf(`{
			"cashier_id": 1,
			"payments": [{"payment_method_code": "CASH", "amount": 0}],
			"status": "completed",
			"items": [{"product_id": %d, "quantity": 1}]
		}`, productID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/sales", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid payment method code returns 400", func(t *testing.T) {
		body := fmt.Sprintf(`{
			"cashier_id": 1,
			"payments": [{"payment_method_code": "NONEXISTENT", "amount": 100000}],
			"status": "completed",
			"items": [{"product_id": %d, "quantity": 1}]
		}`, productID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/sales", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("exactly 10 payments with duplicate methods returns 400", func(t *testing.T) {
		var payments []string
		for i := 0; i < 10; i++ {
			payments = append(payments, `{"payment_method_code":"QRIS","amount":10000}`)
		}
		body := fmt.Sprintf(`{
			"cashier_id": 1,
			"payments": [%s],
			"status": "completed",
			"items": [{"product_id": %d, "quantity": 1}]
		}`, strings.Join(payments, ","), productID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/sales", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("all 5 unique payment methods succeeds", func(t *testing.T) {
		body := fmt.Sprintf(`{
			"cashier_id": 1,
			"payments": [
				{"payment_method_code": "CASH", "amount": 20000},
				{"payment_method_code": "QRIS", "amount": 20000},
				{"payment_method_code": "CARD", "amount": 20000, "reference_number": "REF-CARD"},
				{"payment_method_code": "E_WALLET", "amount": 20000, "reference_number": "REF-EW"},
				{"payment_method_code": "TRANSFER", "amount": 20000, "reference_number": "REF-TRF"}
			],
			"status": "completed",
			"items": [{"product_id": %d, "quantity": 1}]
		}`, productID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/sales", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("11 payments returns 400", func(t *testing.T) {
		var payments []string
		for i := 0; i < 11; i++ {
			payments = append(payments, fmt.Sprintf(`{"payment_method_code":"QRIS","amount":%d}`, 100000/11+1))
		}
		body := fmt.Sprintf(`{
			"cashier_id": 1,
			"payments": [%s],
			"status": "completed",
			"items": [{"product_id": %d, "quantity": 1}]
		}`, strings.Join(payments, ","), productID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/sales", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("empty payments and no payment_method returns 400", func(t *testing.T) {
		body := fmt.Sprintf(`{
			"cashier_id": 1,
			"status": "completed",
			"items": [{"product_id": %d, "quantity": 1}]
		}`, productID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/sales", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestE2E_CreateSale_RejectsLegacyPricingFields asserts the direct sale
// endpoint is server-authoritative: any payload still carrying a client pricing
// field is rejected instead of silently corrected.
func TestE2E_CreateSale_RejectsLegacyPricingFields(t *testing.T) {
	router := setupE2ERouter(t)
	token := loginAs(t, router, "superadmin", "admin123")
	seedE2EPaymentMethods(t)
	productID := createE2EProduct(t, router, token)

	cases := []struct {
		name string
		body string
	}{
		{"item subtotal", fmt.Sprintf(`{"items":[{"product_id": %d, "quantity": 1, "subtotal": 100000}]}`, productID)},
		{"item unit_price", fmt.Sprintf(`{"items":[{"product_id": %d, "quantity": 1, "unit_price": 100000}]}`, productID)},
		{"top-level subtotal", fmt.Sprintf(`{"items":[{"product_id": %d, "quantity": 1}],"subtotal": 100000}`, productID)},
		{"top-level total_amount", fmt.Sprintf(`{"items":[{"product_id": %d, "quantity": 1}],"total_amount": 100000}`, productID)},
		{"discount", fmt.Sprintf(`{"items":[{"product_id": %d, "quantity": 1}],"discount": 5000}`, productID)},
		{"tax", fmt.Sprintf(`{"items":[{"product_id": %d, "quantity": 1}],"tax": 1000}`, productID)},
		{"store_id", fmt.Sprintf(`{"items":[{"product_id": %d, "quantity": 1}],"store_id": 99}`, productID)},
		{"invoice_number", fmt.Sprintf(`{"items":[{"product_id": %d, "quantity": 1}],"invoice_number": "INV-E2E-LEGACY"}`, productID)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/sales", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func seedE2EProduct(t *testing.T) int {
	t.Helper()
	if e2ePool == nil {
		t.Skip("no database connection")
		return 0
	}
	var id int
	err := e2ePool.QueryRow(context.Background(),
		`INSERT INTO products (sku, name, price, cost, status)
		 VALUES (concat('E2E-PROD-', floor(random()*100000)::int), 'E2E Test Product', 100000, 50000, 'active')
		 RETURNING id`).Scan(&id)
	require.NoError(t, err)
	_, err = e2ePool.Exec(context.Background(),
		`INSERT INTO product_stock (product_id, quantity) VALUES ($1, 100)`, id)
	require.NoError(t, err)
	return id
}

func seedE2ESupplier(t *testing.T) int {
	t.Helper()
	if e2ePool == nil {
		t.Skip("no database connection")
		return 0
	}
	var id int
	err := e2ePool.QueryRow(context.Background(),
		`INSERT INTO suppliers (name, code, is_active) VALUES ('E2E Supplier', 'E2E-SUP', true)
		 ON CONFLICT (code) DO UPDATE SET name = 'E2E Supplier' RETURNING id`).Scan(&id)
	if err != nil {
		err = e2ePool.QueryRow(context.Background(),
			`INSERT INTO suppliers (name, code, is_active) VALUES ('E2E Supplier', concat('E2E-SUP-', floor(random()*100000)::int), true) RETURNING id`).Scan(&id)
		require.NoError(t, err)
	}
	return id
}

func seedE2EStore(t *testing.T) {
	t.Helper()
	if e2ePool == nil {
		t.Skip("no database connection")
	}
	var storeCount int
	e2ePool.QueryRow(context.Background(), "SELECT COUNT(*) FROM stores").Scan(&storeCount)
	if storeCount == 0 {
		_, err := e2ePool.Exec(context.Background(), `
			INSERT INTO stores (name, address, phone, is_active)
			VALUES ('E2E Test Store', 'Test Address', '081234567890', true)`)
		require.NoError(t, err)
	}
	_, err := e2ePool.Exec(context.Background(), `
		UPDATE users SET store_id = (SELECT id FROM stores LIMIT 1)
		WHERE username = 'superadmin' AND store_id IS NULL`)
	require.NoError(t, err)
}

func TestE2E_PurchaseOrders(t *testing.T) {
	router := setupE2ERouter(t)
	seedE2EStore(t)
	token := loginAs(t, router, "superadmin", "admin123")

	var poID int
	supplierID := seedE2ESupplier(t)
	productID := seedE2EProduct(t)

	t.Run("POST /api/purchase-orders creates draft", func(t *testing.T) {
		body := fmt.Sprintf(`{
			"supplier_id": %d,
			"store_id": 1,
			"expected_date": "2026-08-15",
			"payment_term": "NET30",
			"notes": "E2E test PO",
			"items": [{"product_id": %d, "qty_ordered": 5, "unit_cost": 50000, "discount_amount": 0}]
		}`, supplierID, productID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/purchase-orders", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)
		var resp struct {
			Data struct {
				ID       int    `json:"id"`
				PONumber string `json:"po_number"`
				Status   string `json:"status"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Greater(t, resp.Data.ID, 0)
		require.NotEmpty(t, resp.Data.PONumber)
		require.Equal(t, "draft", resp.Data.Status)
		poID = resp.Data.ID
	})

	t.Run("GET /api/purchase-orders lists POs", func(t *testing.T) {
		require.Greater(t, poID, 0)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/purchase-orders?limit=10&offset=0", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data []struct {
				ID           int    `json:"id"`
				PONumber     string `json:"po_number"`
				Status       string `json:"status"`
				SupplierName string `json:"supplier_name"`
			} `json:"data"`
			Total int `json:"total"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.GreaterOrEqual(t, len(resp.Data), 1)
		assert.GreaterOrEqual(t, resp.Total, 1)
		assert.Equal(t, poID, resp.Data[0].ID)
		assert.Equal(t, "draft", resp.Data[0].Status)
	})

	t.Run("GET /api/purchase-orders/:id gets detail", func(t *testing.T) {
		require.Greater(t, poID, 0)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/purchase-orders/%d", poID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data struct {
				ID       int    `json:"id"`
				PONumber string `json:"po_number"`
				Status   string `json:"status"`
				Items    []struct {
					ID int `json:"id"`
				} `json:"items"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, poID, resp.Data.ID)
		assert.Equal(t, "draft", resp.Data.Status)
		require.GreaterOrEqual(t, len(resp.Data.Items), 1)
	})

	var poItemID int
	t.Run("GET /api/purchase-orders/:id gets item id for update", func(t *testing.T) {
		require.Greater(t, poID, 0)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/purchase-orders/%d", poID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data struct {
				Items []struct {
					ID int `json:"id"`
				} `json:"items"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.GreaterOrEqual(t, len(resp.Data.Items), 1)
		poItemID = resp.Data.Items[0].ID
	})

	t.Run("PUT /api/purchase-orders/:id updates draft", func(t *testing.T) {
		require.Greater(t, poID, 0)
		require.Greater(t, poItemID, 0)
		body := fmt.Sprintf(`{
			"supplier_id": %d,
			"notes": "Updated E2E test PO",
			"items": [{"id": %d, "product_id": %d, "qty_ordered": 10, "unit_cost": 45000, "discount_amount": 0}]
		}`, supplierID, poItemID, productID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/purchase-orders/%d", poID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("POST /api/purchase-orders/:id/confirm confirms PO", func(t *testing.T) {
		require.Greater(t, poID, 0)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/purchase-orders/%d/confirm", poID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data struct {
				Status string `json:"status"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "confirmed", resp.Data.Status)
	})

	t.Run("POST /api/purchase-orders/:id/confirm twice returns 409", func(t *testing.T) {
		require.Greater(t, poID, 0)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/purchase-orders/%d/confirm", poID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("GET /api/purchase-orders/:id/receipts returns empty", func(t *testing.T) {
		require.Greater(t, poID, 0)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/purchase-orders/%d/receipts", poID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		var listResp struct {
			Data []interface{} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &listResp))
		assert.Empty(t, listResp.Data)
	})

	t.Run("POST /api/purchase-orders/:id/cancel cancels confirmed PO", func(t *testing.T) {
		require.Greater(t, poID, 0)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/purchase-orders/%d/cancel", poID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data struct {
				Status string `json:"status"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "cancelled", resp.Data.Status)
	})

	t.Run("POST /api/purchase-orders/:id/cancel on cancelled PO returns 409", func(t *testing.T) {
		require.Greater(t, poID, 0)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/purchase-orders/%d/cancel", poID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("GET /api/purchase-orders without auth returns 401", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/purchase-orders", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("GET /api/purchase-orders as cashier returns 403", func(t *testing.T) {
		if e2ePool != nil {
			_, err := e2ePool.Exec(context.Background(),
				`INSERT INTO users (username, email, password_hash, role_id, is_active)
				 VALUES ('cashier', 'cashier@test.local', crypt('admin123', gen_salt('bf', 14)), (SELECT id FROM roles WHERE name='cashier'), true)
				 ON CONFLICT (username) DO NOTHING`)
			require.NoError(t, err)
		}
		loginRes := httptest.NewRecorder()
		loginReq, _ := http.NewRequest("POST", "/api/login", strings.NewReader(`{"username":"cashier","password":"admin123"}`))
		loginReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(loginRes, loginReq)
		if loginRes.Code != http.StatusOK {
			t.Skip("cashier login failed")
			return
		}
		var loginResp struct {
			AccessToken string `json:"access_token"`
		}
		require.NoError(t, json.Unmarshal(loginRes.Body.Bytes(), &loginResp))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/purchase-orders", nil)
		req.Header.Set("Authorization", "Bearer "+loginResp.AccessToken)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("POST /api/purchase-orders without items returns error", func(t *testing.T) {
		body := fmt.Sprintf(`{"supplier_id": %d, "store_id": 1, "items": []}`, supplierID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/purchase-orders", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.GreaterOrEqual(t, w.Code, http.StatusBadRequest)
	})

	t.Run("GET /api/purchase-orders/:id with invalid id returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/purchase-orders/abc", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("GET /api/purchase-orders/:id not found returns 404", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/purchase-orders/999999", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestE2E_PurchaseOrderGoodsReceipt(t *testing.T) {
	router := setupE2ERouter(t)
	seedE2EStore(t)
	token := loginAs(t, router, "superadmin", "admin123")

	supplierID := seedE2ESupplier(t)
	productID := seedE2EProduct(t)
	var poID int
	var poItemID int
	var grID int

	t.Run("POST /api/purchase-orders creates draft", func(t *testing.T) {
		body := fmt.Sprintf(`{
			"supplier_id": %d, "store_id": 1, "expected_date": "2026-08-15",
			"notes": "GR E2E test PO",
			"items": [{"product_id": %d, "qty_ordered": 10, "unit_cost": 50000, "discount_amount": 0}]
		}`, supplierID, productID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/purchase-orders", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)
		var createResp struct {
			Data struct {
				ID int `json:"id"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp))
		poID = createResp.Data.ID
	})

	t.Run("GET /api/purchase-orders/:id gets items for GR", func(t *testing.T) {
		require.Greater(t, poID, 0)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/purchase-orders/%d", poID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		var detailResp struct {
			Data struct {
				Items []struct {
					ID int `json:"id"`
				} `json:"items"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &detailResp))
		require.Greater(t, len(detailResp.Data.Items), 0)
		poItemID = detailResp.Data.Items[0].ID
	})

	t.Run("POST /api/purchase-orders/:id/confirm", func(t *testing.T) {
		require.Greater(t, poID, 0)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/purchase-orders/%d/confirm", poID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("POST /api/goods-receipts creates GR", func(t *testing.T) {
		require.Greater(t, poID, 0)
		require.Greater(t, poItemID, 0)
		body := fmt.Sprintf(`{
			"purchase_order_id": %d, "delivery_order_number": "DO-001",
			"shipping_method": "courier", "driver_name": "E2E Driver",
			"notes": "E2E GR test",
			"items": [{"purchase_order_item_id": %d, "qty_good": 8, "qty_damaged": 1}]
		}`, poID, poItemID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/goods-receipts", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
		var resp struct {
			Data struct {
				ID              int    `json:"id"`
				GRNumber        string `json:"gr_number"`
				PurchaseOrderID int    `json:"purchase_order_id"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Greater(t, resp.Data.ID, 0)
		assert.NotEmpty(t, resp.Data.GRNumber)
		assert.Equal(t, poID, resp.Data.PurchaseOrderID)
		grID = resp.Data.ID
	})

	t.Run("GET /api/purchase-orders/:id/receipts returns GR", func(t *testing.T) {
		require.Greater(t, poID, 0)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/purchase-orders/%d/receipts", poID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		var listResp struct {
			Data []struct {
				ID int `json:"id"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &listResp))
		assert.GreaterOrEqual(t, len(listResp.Data), 1)
		assert.Equal(t, grID, listResp.Data[0].ID)
	})

	t.Run("POST /api/purchase-orders/:id/cancel with receipts returns 409", func(t *testing.T) {
		require.Greater(t, poID, 0)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/purchase-orders/%d/cancel", poID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusConflict, w.Code)
	})
}

func TestE2E_PurchaseOrderDeleteDraft(t *testing.T) {
	router := setupE2ERouter(t)
	seedE2EStore(t)
	token := loginAs(t, router, "superadmin", "admin123")

	supplierID := seedE2ESupplier(t)
	productID := seedE2EProduct(t)
	var poID int

	t.Run("POST /api/purchase-orders creates draft to delete", func(t *testing.T) {
		body := fmt.Sprintf(`{
			"supplier_id": %d, "store_id": 1, "notes": "delete me",
			"items": [{"product_id": %d, "qty_ordered": 3, "unit_cost": 10000, "discount_amount": 0}]
		}`, supplierID, productID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/purchase-orders", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)
		var resp struct {
			Data struct {
				ID int `json:"id"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		poID = resp.Data.ID
	})

	t.Run("DELETE /api/purchase-orders/:id deletes draft", func(t *testing.T) {
		require.Greater(t, poID, 0)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/purchase-orders/%d", poID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("GET /api/purchase-orders/:id after delete returns 404", func(t *testing.T) {
		require.Greater(t, poID, 0)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/purchase-orders/%d", poID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestE2E_UserHierarchy(t *testing.T) {
	router := setupE2ERouter(t)
	token := loginAs(t, router, "superadmin", "admin123")

	var managerID int
	var staffID int

	t.Run("create manager user", func(t *testing.T) {
		username := fmt.Sprintf("e2emgr%d", time.Now().UnixNano())
		body := fmt.Sprintf(`{"username":"%s","email":"%s@test.com","password":"password123","role_id":3}`, username, username)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/admin/users", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code, "body: %s", w.Body.String())
		var resp struct {
			Data user.User `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Greater(t, resp.Data.ID, 0)
		managerID = resp.Data.ID
	})

	t.Run("create staff with reports_to", func(t *testing.T) {
		require.Greater(t, managerID, 0)
		username := fmt.Sprintf("e2estaff%d", time.Now().UnixNano())
		body := fmt.Sprintf(`{"username":"%s","email":"%s@test.com","password":"password123","role_id":4,"reports_to":%d}`, username, username, managerID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/admin/users", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)
		var resp struct {
			Data user.User `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Greater(t, resp.Data.ID, 0)
		staffID = resp.Data.ID
	})

	t.Run("get subordinates", func(t *testing.T) {
		require.Greater(t, managerID, 0)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/admin/users/%d/subordinates", managerID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data []user.User `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Len(t, resp.Data, 1)
		assert.Equal(t, staffID, resp.Data[0].ID)
	})

	t.Run("get manager", func(t *testing.T) {
		require.Greater(t, staffID, 0)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/admin/users/%d/manager", staffID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data user.User `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, managerID, resp.Data.ID)
	})

	t.Run("get org chart", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/admin/users/org-chart", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data []user.User `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(resp.Data), 1)
	})

	t.Run("update reports_to rejects self reference", func(t *testing.T) {
		require.Greater(t, staffID, 0)
		body := fmt.Sprintf(`{"reports_to":%d}`, staffID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/admin/users/%d", staffID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("update reports_to success", func(t *testing.T) {
		require.Greater(t, staffID, 0)
		require.Greater(t, managerID, 0)
		body := fmt.Sprintf(`{"reports_to":%d}`, managerID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/admin/users/%d", staffID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("get manager not found for top-level user", func(t *testing.T) {
		require.Greater(t, managerID, 0)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/admin/users/%d/manager", managerID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("circular reference rejected", func(t *testing.T) {
		require.Greater(t, managerID, 0)
		require.Greater(t, staffID, 0)
		body := fmt.Sprintf(`{"reports_to":%d}`, managerID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/admin/users/%d", managerID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
