package purchase

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/eventbus"
)

func setupHandlerTest(t *testing.T) (*gin.Engine, *Handler, int) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := NewService(repo, bus)
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
	perm := func(code string) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	}

	api := r.Group("/api")
	handler.RegisterRoutes(api, auth, perm)

	return r, handler, 1
}

func intPtr(i int) *int {
	return &i
}

func TestHandler_CreateDraft(t *testing.T) {
	r, _, _ := setupHandlerTest(t)
	ctx := context.Background()
	supplierID := insertTestSupplier(t, ctx, "Handler Supplier")
	prodID := insertTestProduct(t, ctx, "HANDLER-PROD", "Handler Product", 10000, 100)

	req := map[string]interface{}{
		"supplier_id": supplierID,
		"items": []map[string]interface{}{
			{"product_id": prodID, "qty_ordered": 5, "unit_cost": 8000},
		},
	}
	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/api/purchase-orders", bytes.NewReader(body))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "draft", data["status"])
	assert.Equal(t, float64(40000), data["grand_total"])
}

func TestHandler_ConfirmPO(t *testing.T) {
	r, _, _ := setupHandlerTest(t)
	ctx := context.Background()
	supplierID := insertTestSupplier(t, ctx, "Handler Confirm Supplier")
	prodID := insertTestProduct(t, ctx, "HANDLER-CONF", "Handler Confirm", 10000, 100)

	repo := NewRepository(dbPool)
	userID := 1
	po := &PurchaseOrder{SupplierID: supplierID, StoreID: 1, Status: StatusDraft, CreatedBy: userID, UpdatedBy: userID}
	items := []PurchaseOrderItem{{ProductID: prodID, QtyOrdered: 5, UnitCost: 8000, ProductName: "Handler Confirm", SKU: "HANDLER-CONF"}}
	tx, _ := repo.BeginTx(ctx)
	po.PONumber, _ = repo.GetNextPONumber(ctx)
	repo.CreatePurchaseOrder(ctx, tx, po, items)
	tx.Commit(ctx)

	fetched, _ := repo.GetPurchaseOrderByID(ctx, po.ID, nil)
	require.Len(t, fetched.Items, 1)
	po.Items = fetched.Items

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/api/purchase-orders/"+strconv.Itoa(po.ID)+"/confirm", nil)
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "confirmed", data["status"])
}

func TestHandler_CancelPOWithReceiptsFails(t *testing.T) {
	r, _, _ := setupHandlerTest(t)
	ctx := context.Background()
	supplierID := insertTestSupplier(t, ctx, "Handler Cancel Supplier")
	prodID := insertTestProduct(t, ctx, "HANDLER-CANCEL", "Handler Cancel", 10000, 100)

	repo := NewRepository(dbPool)
	userID := 1
	po := &PurchaseOrder{SupplierID: supplierID, StoreID: 1, Status: StatusDraft, CreatedBy: userID, UpdatedBy: userID}
	items := []PurchaseOrderItem{{ProductID: prodID, QtyOrdered: 10, UnitCost: 8000, ProductName: "Handler Cancel", SKU: "HANDLER-CANCEL"}}
	tx, _ := repo.BeginTx(ctx)
	po.PONumber, _ = repo.GetNextPONumber(ctx)
	repo.CreatePurchaseOrder(ctx, tx, po, items)
	tx.Commit(ctx)

	fetched, _ := repo.GetPurchaseOrderByID(ctx, po.ID, nil)
	po.Items = fetched.Items

	tx, _ = repo.BeginTx(ctx)
	repo.LockPurchaseOrderForUpdate(ctx, tx, po.ID)
	repo.ConfirmPurchaseOrder(ctx, tx, po.ID, userID, "2026-07-27T10:00:00+07:00")
	tx.Commit(ctx)

	grItems := []CreateGRItemInput{{PurchaseOrderItemID: po.Items[0].ID, QtyGood: 5, QtyDamaged: 1}}
	svc := NewService(repo, eventbus.New())
	svc.CreateGoodsReceipt(ctx, po.ID, userID, 1, grItems)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/api/purchase-orders/"+strconv.Itoa(po.ID)+"/cancel", nil)
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusConflict, w.Code)
}
