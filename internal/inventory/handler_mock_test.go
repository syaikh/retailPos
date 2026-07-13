package inventory

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type mockInventoryService struct {
	adjustStockFn func(ctx context.Context, productID int, quantityChange int, userID int, notes string) error
}

func (m *mockInventoryService) AdjustStock(ctx context.Context, productID int, quantityChange int, userID int, notes string) error {
	return m.adjustStockFn(ctx, productID, quantityChange, userID, notes)
}

var _ InventoryService = (*mockInventoryService)(nil)

func setupMockInventoryRouter(svc InventoryService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "admin")
		c.Set("role", "admin")
		c.Next()
	})
	h := NewHandler(svc, nil)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm string) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	return r
}

func TestMockHandler_AdjustStock(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockInventoryService{
			adjustStockFn: func(ctx context.Context, productID int, quantityChange int, userID int, notes string) error {
				assert.Equal(t, 42, productID)
				assert.Equal(t, 10, quantityChange)
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
		r := setupMockInventoryRouter(&mockInventoryService{})
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/inventory/adjust", strings.NewReader("{bad"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("zero quantity change", func(t *testing.T) {
		r := setupMockInventoryRouter(&mockInventoryService{})
		body := `{"product_id":1,"quantity_change":0,"notes":"no-op"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/inventory/adjust", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "must not be zero")
	})

	t.Run("empty notes", func(t *testing.T) {
		r := setupMockInventoryRouter(&mockInventoryService{})
		body := `{"product_id":1,"quantity_change":5,"notes":"  "}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/inventory/adjust", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "notes are required")
	})

	t.Run("service error", func(t *testing.T) {
		svc := &mockInventoryService{
			adjustStockFn: func(ctx context.Context, productID int, quantityChange int, userID int, notes string) error {
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
		svc := &mockInventoryService{
			adjustStockFn: func(ctx context.Context, productID int, quantityChange int, userID int, notes string) error {
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
