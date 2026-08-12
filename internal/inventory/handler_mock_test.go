package inventory

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"retail-pos-system/internal/permissions"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type mockService struct {
	adjustStockFn           func(ctx context.Context, productID int, quantityChange int, storeID *int, userID int, notes string) error
	adjustStockBatchFn      func(ctx context.Context, adjustments []StockAdjustment, userID int, notes string) error
	getStockByProductIDFn   func(ctx context.Context, productID int) (*ProductStock, error)
	listLocationStockFn     func(ctx context.Context, productID, locationID int, storeID *int) ([]LocationStockItem, error)
	setLocationStockFn      func(ctx context.Context, productID, locationID, quantity, userID int, storeID *int) error
	transferLocationStockFn func(ctx context.Context, productID, fromLocationID, toLocationID, quantity, userID int, storeID *int) error
}

func (m *mockService) AdjustStock(ctx context.Context, productID int, quantityChange int, storeID *int, userID int, notes string) error {
	return m.adjustStockFn(ctx, productID, quantityChange, storeID, userID, notes)
}

func (m *mockService) AdjustStockBatch(ctx context.Context, adjustments []StockAdjustment, userID int, notes string) error {
	if m.adjustStockBatchFn == nil {
		return nil
	}
	return m.adjustStockBatchFn(ctx, adjustments, userID, notes)
}

func (m *mockService) GetStockByProductID(ctx context.Context, productID int) (*ProductStock, error) {
	if m.getStockByProductIDFn == nil {
		return nil, nil
	}
	return m.getStockByProductIDFn(ctx, productID)
}

func (m *mockService) ListLocationStock(ctx context.Context, productID, locationID int, storeID *int) ([]LocationStockItem, error) {
	if m.listLocationStockFn == nil {
		return nil, nil
	}
	return m.listLocationStockFn(ctx, productID, locationID, storeID)
}

func (m *mockService) SetLocationStock(ctx context.Context, productID, locationID, quantity, userID int, storeID *int) error {
	if m.setLocationStockFn == nil {
		return nil
	}
	return m.setLocationStockFn(ctx, productID, locationID, quantity, userID, storeID)
}

func (m *mockService) TransferLocationStock(ctx context.Context, productID, fromLocationID, toLocationID, quantity, userID int, storeID *int) error {
	if m.transferLocationStockFn == nil {
		return nil
	}
	return m.transferLocationStockFn(ctx, productID, fromLocationID, toLocationID, quantity, userID, storeID)
}

var _ Service = (*mockService)(nil)

func setupMockInventoryRouter(svc Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "admin")
		c.Set("role", "admin")
		c.Next()
	})
	h := NewHandler(svc, nil)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	return r
}

func TestMockHandler_AdjustStock(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockService{
			adjustStockFn: func(ctx context.Context, productID int, quantityChange int, storeID *int, userID int, notes string) error {
				assert.Equal(t, 42, productID)
				assert.Equal(t, 10, quantityChange)
				assert.Equal(t, (*int)(nil), storeID)
				assert.Equal(t, 1, userID)
				assert.Equal(t, "restock", notes)
				return nil
			},
		}
		r := setupMockInventoryRouter(svc)
		body := `{"product_id":42,"quantity_change":10,"notes":"restock"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/inventory/adjust", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ok")
	})

	t.Run("invalid json", func(t *testing.T) {
		r := setupMockInventoryRouter(&mockService{})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/inventory/adjust", strings.NewReader("{bad"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("zero quantity change", func(t *testing.T) {
		r := setupMockInventoryRouter(&mockService{})
		body := `{"product_id":1,"quantity_change":0,"notes":"no-op"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/inventory/adjust", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "must not be zero")
	})

	t.Run("empty notes", func(t *testing.T) {
		r := setupMockInventoryRouter(&mockService{})
		body := `{"product_id":1,"quantity_change":5,"notes":"  "}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/inventory/adjust", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "notes are required")
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockService{
			adjustStockFn: func(ctx context.Context, productID int, quantityChange int, storeID *int, userID int, notes string) error {
				return errors.New("product not found")
			},
		}
		r := setupMockInventoryRouter(svc)
		body := `{"product_id":999,"quantity_change":5,"notes":"test"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/inventory/adjust", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("negative quantity change (decrease)", func(t *testing.T) {
		svc := &mockService{
			adjustStockFn: func(ctx context.Context, productID int, quantityChange int, storeID *int, userID int, notes string) error {
				assert.Equal(t, -5, quantityChange)
				return nil
			},
		}
		r := setupMockInventoryRouter(svc)
		body := `{"product_id":10,"quantity_change":-5,"notes":"sale"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/inventory/adjust", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestMockHandler_ListLocationStock(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockService{
			listLocationStockFn: func(ctx context.Context, productID, locationID int, storeID *int) ([]LocationStockItem, error) {
				assert.Equal(t, 7, productID)
				assert.Equal(t, 3, locationID)
				assert.Equal(t, (*int)(nil), storeID)
				return []LocationStockItem{
					{ProductID: 7, LocationID: 3, LocationCode: "RACK-A", Quantity: 4},
				}, nil
			},
		}
		r := setupMockInventoryRouter(svc)
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/inventory/locations?product_id=7&location_id=3", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"data":`)
		assert.Contains(t, w.Body.String(), "RACK-A")
		assert.Contains(t, w.Body.String(), `"quantity":4`)
	})

	t.Run("no filters passes zero", func(t *testing.T) {
		svc := &mockService{
			listLocationStockFn: func(ctx context.Context, productID, locationID int, storeID *int) ([]LocationStockItem, error) {
				assert.Equal(t, 0, productID)
				assert.Equal(t, 0, locationID)
				assert.Equal(t, (*int)(nil), storeID)
				return nil, nil
			},
		}
		r := setupMockInventoryRouter(svc)
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/inventory/locations", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockService{
			listLocationStockFn: func(ctx context.Context, productID, locationID int, storeID *int) ([]LocationStockItem, error) {
				return nil, errors.New("boom")
			},
		}
		r := setupMockInventoryRouter(svc)
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/inventory/locations?product_id=1", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockHandler_SetLocationStock(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockService{
			setLocationStockFn: func(ctx context.Context, productID, locationID, quantity, userID int, storeID *int) error {
				assert.Equal(t, 42, productID)
				assert.Equal(t, 5, locationID)
				assert.Equal(t, 12, quantity)
				assert.Equal(t, 1, userID)
				assert.Equal(t, (*int)(nil), storeID)
				return nil
			},
		}
		r := setupMockInventoryRouter(svc)
		body := `{"product_id":42,"location_id":5,"quantity":12}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/inventory/locations", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ok")
	})

	t.Run("invalid json", func(t *testing.T) {
		r := setupMockInventoryRouter(&mockService{})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/inventory/locations", strings.NewReader("{bad"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing ids", func(t *testing.T) {
		r := setupMockInventoryRouter(&mockService{})
		body := `{"product_id":0,"location_id":5,"quantity":1}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/inventory/locations", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "required")
	})

	t.Run("invalid user context", func(t *testing.T) {
		svc := &mockService{
			setLocationStockFn: func(ctx context.Context, productID, locationID, quantity, userID int, storeID *int) error {
				t.Fatal("service must not be called when user is missing from context")
				return nil
			},
		}
		h := NewHandler(svc, nil)
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.Use(func(c *gin.Context) { c.Next() })
		h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
			return func(c *gin.Context) { c.Next() }
		})
		body := `{"product_id":42,"location_id":5,"quantity":1}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/inventory/locations", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("known domain error maps to 400", func(t *testing.T) {
		svc := &mockService{
			setLocationStockFn: func(ctx context.Context, productID, locationID, quantity, userID int, storeID *int) error {
				return ErrLocationInactive
			},
		}
		r := setupMockInventoryRouter(svc)
		body := `{"product_id":42,"location_id":5,"quantity":1}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/inventory/locations", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("negative quantity maps to 400", func(t *testing.T) {
		svc := &mockService{
			setLocationStockFn: func(ctx context.Context, productID, locationID, quantity, userID int, storeID *int) error {
				return ErrNegativeQuantity
			},
		}
		r := setupMockInventoryRouter(svc)
		body := `{"product_id":42,"location_id":5,"quantity":-1}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/inventory/locations", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "must not be negative")
	})

	t.Run("unknown error maps to 500", func(t *testing.T) {
		svc := &mockService{
			setLocationStockFn: func(ctx context.Context, productID, locationID, quantity, userID int, storeID *int) error {
				return errors.New("db down")
			},
		}
		r := setupMockInventoryRouter(svc)
		body := `{"product_id":42,"location_id":5,"quantity":1}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/inventory/locations", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockHandler_TransferLocationStock(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockService{
			transferLocationStockFn: func(ctx context.Context, productID, fromLocationID, toLocationID, quantity, userID int, storeID *int) error {
				assert.Equal(t, 42, productID)
				assert.Equal(t, 5, fromLocationID)
				assert.Equal(t, 6, toLocationID)
				assert.Equal(t, 3, quantity)
				assert.Equal(t, 1, userID)
				assert.Equal(t, (*int)(nil), storeID)
				return nil
			},
		}
		r := setupMockInventoryRouter(svc)
		body := `{"product_id":42,"from_location_id":5,"to_location_id":6,"quantity":3}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/inventory/locations/transfer", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ok")
	})

	t.Run("missing ids", func(t *testing.T) {
		r := setupMockInventoryRouter(&mockService{})
		body := `{"product_id":42,"from_location_id":5,"to_location_id":0,"quantity":3}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/inventory/locations/transfer", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "required")
	})

	t.Run("invalid json", func(t *testing.T) {
		r := setupMockInventoryRouter(&mockService{})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/inventory/locations/transfer", strings.NewReader("{bad"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("same location maps to 400", func(t *testing.T) {
		svc := &mockService{
			transferLocationStockFn: func(ctx context.Context, productID, fromLocationID, toLocationID, quantity, userID int, storeID *int) error {
				return ErrSameLocation
			},
		}
		r := setupMockInventoryRouter(svc)
		body := `{"product_id":42,"from_location_id":5,"to_location_id":5,"quantity":3}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/inventory/locations/transfer", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("non-positive quantity maps to 400", func(t *testing.T) {
		svc := &mockService{
			transferLocationStockFn: func(ctx context.Context, productID, fromLocationID, toLocationID, quantity, userID int, storeID *int) error {
				return ErrNonPositiveQuantity
			},
		}
		r := setupMockInventoryRouter(svc)
		body := `{"product_id":42,"from_location_id":5,"to_location_id":6,"quantity":0}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/inventory/locations/transfer", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "quantity must be positive")
	})

	t.Run("unknown error maps to 500", func(t *testing.T) {
		svc := &mockService{
			transferLocationStockFn: func(ctx context.Context, productID, fromLocationID, toLocationID, quantity, userID int, storeID *int) error {
				return errors.New("db down")
			},
		}
		r := setupMockInventoryRouter(svc)
		body := `{"product_id":42,"from_location_id":5,"to_location_id":6,"quantity":3}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/inventory/locations/transfer", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMockHandler_AdjustStock_NoUserContext(t *testing.T) {
	svc := &mockService{
		adjustStockFn: func(ctx context.Context, productID int, quantityChange int, storeID *int, userID int, notes string) error {
			t.Fatal("service must not be called when user is missing from context")
			return nil
		},
	}
	h := NewHandler(svc, nil)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Next() })
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	body := `{"product_id":42,"quantity_change":10,"notes":"restock"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/inventory/adjust", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid user")
}

func TestMockHandler_TransferLocationStock_NoUserContext(t *testing.T) {
	svc := &mockService{
		transferLocationStockFn: func(ctx context.Context, productID, fromLocationID, toLocationID, quantity, userID int, storeID *int) error {
			t.Fatal("service must not be called when user is missing from context")
			return nil
		},
	}
	h := NewHandler(svc, nil)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Next() })
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	body := `{"product_id":42,"from_location_id":5,"to_location_id":6,"quantity":3}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/inventory/locations/transfer", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid user")
}

func TestMockHandler_SetLocationStock_Audits(t *testing.T) {
	svc := &mockService{
		setLocationStockFn: func(ctx context.Context, productID, locationID, quantity, userID int, storeID *int) error {
			return nil
		},
	}
	r := setupMockInventoryRouterWithAudit(svc)
	body := `{"product_id":42,"location_id":5,"quantity":12}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/inventory/locations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}
