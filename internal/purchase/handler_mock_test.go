package purchase

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/permissions"
)

type mockPurchaseService struct {
	createDraft        func(ctx context.Context, po *PurchaseOrder, items []PurchaseOrderItem) error
	updateDraft        func(ctx context.Context, id int, po *PurchaseOrder, items []PurchaseOrderItem) error
	deleteDraft        func(ctx context.Context, id int) error
	confirm            func(ctx context.Context, id, userID int) error
	cancel             func(ctx context.Context, id, userID int) error
	getDetail          func(ctx context.Context, id int, storeID *int) (*PurchaseOrder, error)
	list               func(ctx context.Context, limit, offset int, search, sortBy, sortDir, status, supplierID, startDate, endDate string, storeID *int) ([]PurchaseOrder, int, error)
	getReceipts        func(ctx context.Context, poID int, storeID *int) ([]GoodsReceipt, error)
	createGoodsReceipt func(ctx context.Context, poID, userID, storeID int, items []CreateGRItemInput) (*GoodsReceipt, error)
}

func (m *mockPurchaseService) CreateDraft(ctx context.Context, po *PurchaseOrder, items []PurchaseOrderItem) error {
	if m.createDraft == nil {
		return nil
	}
	return m.createDraft(ctx, po, items)
}

func (m *mockPurchaseService) UpdateDraft(ctx context.Context, id int, po *PurchaseOrder, items []PurchaseOrderItem) error {
	if m.updateDraft == nil {
		return nil
	}
	return m.updateDraft(ctx, id, po, items)
}

func (m *mockPurchaseService) DeleteDraft(ctx context.Context, id int) error {
	if m.deleteDraft == nil {
		return nil
	}
	return m.deleteDraft(ctx, id)
}

func (m *mockPurchaseService) Confirm(ctx context.Context, id, userID int) error {
	if m.confirm == nil {
		return nil
	}
	return m.confirm(ctx, id, userID)
}

func (m *mockPurchaseService) Cancel(ctx context.Context, id, userID int) error {
	if m.cancel == nil {
		return nil
	}
	return m.cancel(ctx, id, userID)
}

func (m *mockPurchaseService) GetDetail(ctx context.Context, id int, storeID *int) (*PurchaseOrder, error) {
	if m.getDetail == nil {
		return &PurchaseOrder{PONumber: "PO-MOCK-001"}, nil
	}
	return m.getDetail(ctx, id, storeID)
}

func (m *mockPurchaseService) List(ctx context.Context, limit, offset int, search, sortBy, sortDir, status, supplierID, startDate, endDate string, storeID *int) ([]PurchaseOrder, int, error) {
	if m.list == nil {
		return []PurchaseOrder{}, 0, nil
	}
	return m.list(ctx, limit, offset, search, sortBy, sortDir, status, supplierID, startDate, endDate, storeID)
}

func (m *mockPurchaseService) GetReceipts(ctx context.Context, poID int, storeID *int) ([]GoodsReceipt, error) {
	if m.getReceipts == nil {
		return []GoodsReceipt{}, nil
	}
	return m.getReceipts(ctx, poID, storeID)
}

func (m *mockPurchaseService) CreateGoodsReceipt(ctx context.Context, poID, userID, storeID int, items []CreateGRItemInput) (*GoodsReceipt, error) {
	if m.createGoodsReceipt == nil {
		return &GoodsReceipt{GRNumber: "GR-MOCK-001"}, nil
	}
	return m.createGoodsReceipt(ctx, poID, userID, storeID, items)
}

func setupHandlerMock(t *testing.T, svc PurchaseService) *gin.Engine {
	t.Helper()
	auditRepo := audit.NewRepository(dbPool)
	auditSvc := audit.NewService(auditRepo)
	handler := NewHandler(svc, auditSvc)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	auth := func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "testuser")
		c.Set("role", "admin")
		c.Set("storeID", intPtr(1))
		c.Next()
	}
	perm := func(code permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	}

	api := r.Group("/api")
	handler.RegisterRoutes(api, auth, perm)
	return r
}

func TestHandlerMock_ErrorBranches(t *testing.T) {
	boom := func(ctx context.Context, id int, storeID *int) (*PurchaseOrder, error) {
		return nil, assert.AnError
	}

	t.Run("create draft warehouse id", func(t *testing.T) {
		svc := &mockPurchaseService{}
		require.NotNil(t, svc)
		r := setupHandlerMock(t, svc)
		w := postJSON(t, r, "POST", "/api/purchase-orders", map[string]interface{}{
			"supplier_id":  1,
			"warehouse_id": 2,
			"items":        []map[string]interface{}{{"product_id": 1, "qty_ordered": 1, "unit_cost": 100, "notes": "rush"}},
		})
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("confirm invalid id", func(t *testing.T) {
		r := setupHandlerMock(t, &mockPurchaseService{})
		w := postJSON(t, r, "POST", "/api/purchase-orders/abc/confirm", nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("cancel invalid id", func(t *testing.T) {
		r := setupHandlerMock(t, &mockPurchaseService{})
		w := postJSON(t, r, "POST", "/api/purchase-orders/abc/cancel", nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("confirm get detail error", func(t *testing.T) {
		svc := &mockPurchaseService{getDetail: boom}
		r := setupHandlerMock(t, svc)
		w := postJSON(t, r, "POST", "/api/purchase-orders/1/confirm", nil)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("cancel get detail error", func(t *testing.T) {
		svc := &mockPurchaseService{getDetail: boom}
		r := setupHandlerMock(t, svc)
		w := postJSON(t, r, "POST", "/api/purchase-orders/1/cancel", nil)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("confirm already confirmed", func(t *testing.T) {
		svc := &mockPurchaseService{
			confirm: func(ctx context.Context, id, userID int) error { return ErrPurchaseOrderAlreadyConfirmed },
		}
		r := setupHandlerMock(t, svc)
		w := postJSON(t, r, "POST", "/api/purchase-orders/1/confirm", nil)
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("confirm generic error", func(t *testing.T) {
		svc := &mockPurchaseService{
			confirm: func(ctx context.Context, id, userID int) error { return assert.AnError },
		}
		r := setupHandlerMock(t, svc)
		w := postJSON(t, r, "POST", "/api/purchase-orders/1/confirm", nil)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("cancel generic error", func(t *testing.T) {
		svc := &mockPurchaseService{
			cancel: func(ctx context.Context, id, userID int) error { return assert.AnError },
		}
		r := setupHandlerMock(t, svc)
		w := postJSON(t, r, "POST", "/api/purchase-orders/1/cancel", nil)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("get po internal error", func(t *testing.T) {
		svc := &mockPurchaseService{getDetail: boom}
		r := setupHandlerMock(t, svc)
		w := postJSON(t, r, "GET", "/api/purchase-orders/1", nil)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("list internal error", func(t *testing.T) {
		svc := &mockPurchaseService{
			list: func(ctx context.Context, limit, offset int, search, sortBy, sortDir, status, supplierID, startDate, endDate string, storeID *int) ([]PurchaseOrder, int, error) {
				return nil, 0, assert.AnError
			},
		}
		r := setupHandlerMock(t, svc)
		w := postJSON(t, r, "GET", "/api/purchase-orders", nil)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("get receipts internal error", func(t *testing.T) {
		svc := &mockPurchaseService{
			getReceipts: func(ctx context.Context, poID int, storeID *int) ([]GoodsReceipt, error) {
				return nil, assert.AnError
			},
		}
		r := setupHandlerMock(t, svc)
		w := postJSON(t, r, "GET", "/api/purchase-orders/1/receipts", nil)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("get receipts nil slice", func(t *testing.T) {
		svc := &mockPurchaseService{
			getReceipts: func(ctx context.Context, poID int, storeID *int) ([]GoodsReceipt, error) {
				return nil, nil
			},
		}
		r := setupHandlerMock(t, svc)
		w := postJSON(t, r, "GET", "/api/purchase-orders/1/receipts", nil)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("goods receipt invalid status", func(t *testing.T) {
		svc := &mockPurchaseService{
			createGoodsReceipt: func(ctx context.Context, poID, userID, storeID int, items []CreateGRItemInput) (*GoodsReceipt, error) {
				return nil, ErrInvalidPOStatusForReceiving
			},
		}
		r := setupHandlerMock(t, svc)
		w := postJSON(t, r, "POST", "/api/goods-receipts", map[string]interface{}{
			"purchase_order_id": 1,
			"items":             []map[string]interface{}{{"purchase_order_item_id": 1, "qty_good": 1}},
		})
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("goods receipt invalid qty", func(t *testing.T) {
		svc := &mockPurchaseService{
			createGoodsReceipt: func(ctx context.Context, poID, userID, storeID int, items []CreateGRItemInput) (*GoodsReceipt, error) {
				return nil, ErrInvalidReceivingQty
			},
		}
		r := setupHandlerMock(t, svc)
		w := postJSON(t, r, "POST", "/api/goods-receipts", map[string]interface{}{
			"purchase_order_id": 1,
			"items":             []map[string]interface{}{{"purchase_order_item_id": 1, "qty_good": 1}},
		})
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("goods receipt generic error", func(t *testing.T) {
		svc := &mockPurchaseService{
			createGoodsReceipt: func(ctx context.Context, poID, userID, storeID int, items []CreateGRItemInput) (*GoodsReceipt, error) {
				return nil, assert.AnError
			},
		}
		r := setupHandlerMock(t, svc)
		w := postJSON(t, r, "POST", "/api/goods-receipts", map[string]interface{}{
			"purchase_order_id": 1,
			"items":             []map[string]interface{}{{"purchase_order_item_id": 1, "qty_good": 1}},
		})
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("create draft validation error maps to 400", func(t *testing.T) {
		svc := &mockPurchaseService{
			createDraft: func(ctx context.Context, po *PurchaseOrder, items []PurchaseOrderItem) error {
				return fmt.Errorf("%w: items cannot be empty", ErrInvalidInput)
			},
		}
		r := setupHandlerMock(t, svc)
		w := postJSON(t, r, "POST", "/api/purchase-orders", map[string]interface{}{
			"supplier_id": 1,
			"items":       []map[string]interface{}{{"product_id": 1, "qty_ordered": 1, "unit_cost": 100}},
		})
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("create draft duplicate item maps to 400", func(t *testing.T) {
		svc := &mockPurchaseService{
			createDraft: func(ctx context.Context, po *PurchaseOrder, items []PurchaseOrderItem) error {
				return ErrDuplicatePOItem
			},
		}
		r := setupHandlerMock(t, svc)
		w := postJSON(t, r, "POST", "/api/purchase-orders", map[string]interface{}{
			"supplier_id": 1,
			"items":       []map[string]interface{}{{"product_id": 1, "qty_ordered": 1, "unit_cost": 100}},
		})
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("create draft generic error stays 500", func(t *testing.T) {
		svc := &mockPurchaseService{
			createDraft: func(ctx context.Context, po *PurchaseOrder, items []PurchaseOrderItem) error {
				return assert.AnError
			},
		}
		r := setupHandlerMock(t, svc)
		w := postJSON(t, r, "POST", "/api/purchase-orders", map[string]interface{}{
			"supplier_id": 1,
			"items":       []map[string]interface{}{{"product_id": 1, "qty_ordered": 1, "unit_cost": 100}},
		})
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("update draft validation error maps to 400", func(t *testing.T) {
		svc := &mockPurchaseService{
			updateDraft: func(ctx context.Context, id int, po *PurchaseOrder, items []PurchaseOrderItem) error {
				return fmt.Errorf("%w: qty_ordered must be greater than 0 for product 1", ErrInvalidInput)
			},
		}
		r := setupHandlerMock(t, svc)
		w := postJSON(t, r, "PUT", "/api/purchase-orders/1", map[string]interface{}{
			"supplier_id": 1,
			"items":       []map[string]interface{}{{"product_id": 1, "qty_ordered": 0, "unit_cost": 100}},
		})
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("update draft duplicate item maps to 400", func(t *testing.T) {
		svc := &mockPurchaseService{
			updateDraft: func(ctx context.Context, id int, po *PurchaseOrder, items []PurchaseOrderItem) error {
				return ErrDuplicatePOItem
			},
		}
		r := setupHandlerMock(t, svc)
		w := postJSON(t, r, "PUT", "/api/purchase-orders/1", map[string]interface{}{
			"supplier_id": 1,
			"items":       []map[string]interface{}{{"product_id": 1, "qty_ordered": 1, "unit_cost": 100}},
		})
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("update draft not found maps to 404", func(t *testing.T) {
		svc := &mockPurchaseService{
			updateDraft: func(ctx context.Context, id int, po *PurchaseOrder, items []PurchaseOrderItem) error {
				return ErrPurchaseOrderNotFound
			},
		}
		r := setupHandlerMock(t, svc)
		w := postJSON(t, r, "PUT", "/api/purchase-orders/999999", map[string]interface{}{
			"supplier_id": 1,
			"items":       []map[string]interface{}{{"product_id": 1, "qty_ordered": 1, "unit_cost": 100}},
		})
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("update draft not draft maps to 409", func(t *testing.T) {
		svc := &mockPurchaseService{
			updateDraft: func(ctx context.Context, id int, po *PurchaseOrder, items []PurchaseOrderItem) error {
				return ErrPurchaseOrderNotDraft
			},
		}
		r := setupHandlerMock(t, svc)
		w := postJSON(t, r, "PUT", "/api/purchase-orders/1", map[string]interface{}{
			"supplier_id": 1,
			"items":       []map[string]interface{}{{"product_id": 1, "qty_ordered": 1, "unit_cost": 100}},
		})
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("update draft generic error stays 500", func(t *testing.T) {
		svc := &mockPurchaseService{
			updateDraft: func(ctx context.Context, id int, po *PurchaseOrder, items []PurchaseOrderItem) error {
				return assert.AnError
			},
		}
		r := setupHandlerMock(t, svc)
		w := postJSON(t, r, "PUT", "/api/purchase-orders/1", map[string]interface{}{
			"supplier_id": 1,
			"items":       []map[string]interface{}{{"product_id": 1, "qty_ordered": 1, "unit_cost": 100}},
		})
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
