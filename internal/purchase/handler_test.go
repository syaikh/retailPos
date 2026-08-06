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
	"retail-pos-system/internal/permissions"
)

func setupHandlerTest(t *testing.T) (*gin.Engine, *Handler, int) {
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	svc := newWiredService(repo, bus)
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

	return r, handler, 1
}

func intPtr(i int) *int {
	return &i
}

func TestHandler_CreateDraft(t *testing.T) {
	r, _, _ := setupHandlerTest(t)
	ctx := context.Background()
	supplierID := insertTestSupplier(ctx, t, "Handler Supplier")
	prodID := insertTestProduct(ctx, t, "HANDLER-PROD", "Handler Product", 10000, 100)

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
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "draft", data["status"])
	assert.Equal(t, float64(40000), data["grand_total"])
}

func TestHandler_ConfirmPO(t *testing.T) {
	r, _, _ := setupHandlerTest(t)
	ctx := context.Background()
	supplierID := insertTestSupplier(ctx, t, "Handler Confirm Supplier")
	prodID := insertTestProduct(ctx, t, "HANDLER-CONF", "Handler Confirm", 10000, 100)

	repo := NewRepository(dbPool)
	userID := 1
	po := &Order{SupplierID: supplierID, StoreID: 1, Status: StatusDraft, CreatedBy: userID, UpdatedBy: userID}
	items := []OrderItem{{ProductID: prodID, QtyOrdered: 5, UnitCost: 8000, ProductName: "Handler Confirm", SKU: "HANDLER-CONF"}}
	tx, _ := repo.BeginTx(ctx)
	po.PONumber, _ = repo.GetNextPONumber(ctx)
	_ = repo.CreatePurchaseOrder(ctx, tx, po, items)
	_ = tx.Commit(ctx)

	fetched, _ := repo.GetPurchaseOrderByID(ctx, po.ID, nil)
	require.Len(t, fetched.Items, 1)
	po.Items = fetched.Items

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/api/purchase-orders/"+strconv.Itoa(po.ID)+"/confirm", nil)
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "confirmed", data["status"])
}

func TestHandler_CancelPOWithReceiptsFails(t *testing.T) {
	r, _, _ := setupHandlerTest(t)
	ctx := context.Background()
	supplierID := insertTestSupplier(ctx, t, "Handler Cancel Supplier")
	prodID := insertTestProduct(ctx, t, "HANDLER-CANCEL", "Handler Cancel", 10000, 100)

	repo := NewRepository(dbPool)
	userID := 1
	po := &Order{SupplierID: supplierID, StoreID: 1, Status: StatusDraft, CreatedBy: userID, UpdatedBy: userID}
	items := []OrderItem{{ProductID: prodID, QtyOrdered: 10, UnitCost: 8000, ProductName: "Handler Cancel", SKU: "HANDLER-CANCEL"}}
	tx, _ := repo.BeginTx(ctx)
	po.PONumber, _ = repo.GetNextPONumber(ctx)
	_ = repo.CreatePurchaseOrder(ctx, tx, po, items)
	_ = tx.Commit(ctx)

	fetched, _ := repo.GetPurchaseOrderByID(ctx, po.ID, nil)
	po.Items = fetched.Items

	tx, _ = repo.BeginTx(ctx)
	_ = repo.LockPurchaseOrderForUpdate(ctx, tx, po.ID)
	_ = repo.ConfirmPurchaseOrder(ctx, tx, po.ID, userID, "2026-07-27T10:00:00+07:00")
	_ = tx.Commit(ctx)

	grItems := []CreateGRItemInput{{PurchaseOrderItemID: po.Items[0].ID, QtyGood: 5, QtyDamaged: 1}}
	svc := newWiredService(repo, eventbus.New())
	_, _ = svc.CreateGoodsReceipt(ctx, po.ID, userID, 1, grItems)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/api/purchase-orders/"+strconv.Itoa(po.ID)+"/cancel", nil)
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHandler_GetPO(t *testing.T) {
	r, _, _ := setupHandlerTest(t)
	ctx := context.Background()
	supplierID := insertTestSupplier(ctx, t, "Handler GetPO Supplier")
	prodID := insertTestProduct(ctx, t, "HANDLER-GETPO", "Handler GetPO", 15000, 50)

	repo := NewRepository(dbPool)
	userID := 1
	po := &Order{SupplierID: supplierID, StoreID: 1, Status: StatusDraft, CreatedBy: userID, UpdatedBy: userID}
	items := []OrderItem{{ProductID: prodID, QtyOrdered: 3, UnitCost: 12000, ProductName: "Handler GetPO", SKU: "HANDLER-GETPO"}}
	tx, _ := repo.BeginTx(ctx)
	po.PONumber, _ = repo.GetNextPONumber(ctx)
	_ = repo.CreatePurchaseOrder(ctx, tx, po, items)
	_ = tx.Commit(ctx)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("GET", "/api/purchase-orders/"+strconv.Itoa(po.ID), nil)
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(po.ID), data["id"])
	assert.Equal(t, "draft", data["status"])
}

func TestHandler_GetPO_NotFound(t *testing.T) {
	r, _, _ := setupHandlerTest(t)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("GET", "/api/purchase-orders/999999", nil)
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetPO_InvalidID(t *testing.T) {
	r, _, _ := setupHandlerTest(t)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("GET", "/api/purchase-orders/abc", nil)
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListPOs(t *testing.T) {
	r, _, _ := setupHandlerTest(t)
	ctx := context.Background()
	supplierID := insertTestSupplier(ctx, t, "Handler List Supplier")
	prodID := insertTestProduct(ctx, t, "HANDLER-LIST", "Handler List", 10000, 100)

	repo := NewRepository(dbPool)
	userID := 1
	for i := 0; i < 3; i++ {
		po := &Order{SupplierID: supplierID, StoreID: 1, Status: StatusDraft, CreatedBy: userID, UpdatedBy: userID}
		items := []OrderItem{{ProductID: prodID, QtyOrdered: 2, UnitCost: 5000, ProductName: "Handler List", SKU: "HANDLER-LIST"}}
		tx, _ := repo.BeginTx(ctx)
		po.PONumber, _ = repo.GetNextPONumber(ctx)
		_ = repo.CreatePurchaseOrder(ctx, tx, po, items)
		_ = tx.Commit(ctx)
	}

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("GET", "/api/purchase-orders?limit=10&offset=0", nil)
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	assert.GreaterOrEqual(t, len(data), 3)
	total := resp["total"].(float64)
	assert.GreaterOrEqual(t, total, float64(3))
}

func TestHandler_ListPOs_EmptyResult(t *testing.T) {
	r, _, _ := setupHandlerTest(t)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("GET", "/api/purchase-orders?limit=10&offset=0&search=ZZZZNONEXISTENT", nil)
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	assert.Empty(t, data)
}

func TestHandler_UpdateDraft(t *testing.T) {
	r, _, _ := setupHandlerTest(t)
	ctx := context.Background()
	supplierID := insertTestSupplier(ctx, t, "Handler Update Supplier")
	prodID := insertTestProduct(ctx, t, "HANDLER-UPD", "Handler Update", 10000, 100)

	repo := NewRepository(dbPool)
	userID := 1
	po := &Order{SupplierID: supplierID, StoreID: 1, Status: StatusDraft, CreatedBy: userID, UpdatedBy: userID}
	items := []OrderItem{{ProductID: prodID, QtyOrdered: 5, UnitCost: 8000, ProductName: "Handler Update", SKU: "HANDLER-UPD"}}
	tx, _ := repo.BeginTx(ctx)
	po.PONumber, _ = repo.GetNextPONumber(ctx)
	_ = repo.CreatePurchaseOrder(ctx, tx, po, items)
	_ = tx.Commit(ctx)

	req := map[string]interface{}{
		"supplier_id": supplierID,
		"items": []map[string]interface{}{
			{"id": 0, "product_id": prodID, "qty_ordered": 10, "unit_cost": 7500},
		},
	}
	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("PUT", "/api/purchase-orders/"+strconv.Itoa(po.ID), bytes.NewReader(body))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "draft", data["status"])
}

func TestHandler_UpdateDraft_InvalidID(t *testing.T) {
	r, _, _ := setupHandlerTest(t)
	ctx := context.Background()
	supplierID := insertTestSupplier(ctx, t, "Handler Upd Invalid Supplier")
	prodID := insertTestProduct(ctx, t, "HANDLER-UPDINV", "Handler Upd Invalid", 10000, 100)

	req := map[string]interface{}{
		"supplier_id": supplierID,
		"items": []map[string]interface{}{
			{"product_id": prodID, "qty_ordered": 5, "unit_cost": 8000},
		},
	}
	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("PUT", "/api/purchase-orders/abc", bytes.NewReader(body))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_DeleteDraft(t *testing.T) {
	r, _, _ := setupHandlerTest(t)
	ctx := context.Background()
	supplierID := insertTestSupplier(ctx, t, "Handler Delete Supplier")
	prodID := insertTestProduct(ctx, t, "HANDLER-DEL", "Handler Delete", 10000, 100)

	repo := NewRepository(dbPool)
	userID := 1
	po := &Order{SupplierID: supplierID, StoreID: 1, Status: StatusDraft, CreatedBy: userID, UpdatedBy: userID}
	items := []OrderItem{{ProductID: prodID, QtyOrdered: 2, UnitCost: 5000, ProductName: "Handler Delete", SKU: "HANDLER-DEL"}}
	tx, _ := repo.BeginTx(ctx)
	po.PONumber, _ = repo.GetNextPONumber(ctx)
	_ = repo.CreatePurchaseOrder(ctx, tx, po, items)
	_ = tx.Commit(ctx)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("DELETE", "/api/purchase-orders/"+strconv.Itoa(po.ID), nil)
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["data"].(map[string]interface{})["deleted"].(bool))
}

func TestHandler_DeleteDraft_NotDraftReturnsConflict(t *testing.T) {
	r, _, _ := setupHandlerTest(t)
	ctx := context.Background()
	supplierID := insertTestSupplier(ctx, t, "Handler Delete Conflict Supplier")
	prodID := insertTestProduct(ctx, t, "HANDLER-DELCF", "Handler Delete Conflict", 10000, 100)

	repo := NewRepository(dbPool)
	userID := 1
	po := &Order{SupplierID: supplierID, StoreID: 1, Status: StatusDraft, CreatedBy: userID, UpdatedBy: userID}
	items := []OrderItem{{ProductID: prodID, QtyOrdered: 2, UnitCost: 5000, ProductName: "Handler Delete Conflict", SKU: "HANDLER-DELCF"}}
	tx, _ := repo.BeginTx(ctx)
	po.PONumber, _ = repo.GetNextPONumber(ctx)
	_ = repo.CreatePurchaseOrder(ctx, tx, po, items)
	_ = tx.Commit(ctx)

	tx, _ = repo.BeginTx(ctx)
	_ = repo.LockPurchaseOrderForUpdate(ctx, tx, po.ID)
	_ = repo.ConfirmPurchaseOrder(ctx, tx, po.ID, userID, "2026-07-27T10:00:00+07:00")
	_ = tx.Commit(ctx)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("DELETE", "/api/purchase-orders/"+strconv.Itoa(po.ID), nil)
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHandler_DeleteDraft_InvalidID(t *testing.T) {
	r, _, _ := setupHandlerTest(t)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("DELETE", "/api/purchase-orders/abc", nil)
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetReceipts(t *testing.T) {
	r, _, _ := setupHandlerTest(t)
	ctx := context.Background()
	supplierID := insertTestSupplier(ctx, t, "Handler Receipts Supplier")
	prodID := insertTestProduct(ctx, t, "HANDLER-RCPT", "Handler Receipts", 10000, 100)

	repo := NewRepository(dbPool)
	userID := 1
	po := &Order{SupplierID: supplierID, StoreID: 1, Status: StatusDraft, CreatedBy: userID, UpdatedBy: userID}
	items := []OrderItem{{ProductID: prodID, QtyOrdered: 10, UnitCost: 8000, ProductName: "Handler Receipts", SKU: "HANDLER-RCPT"}}
	tx, _ := repo.BeginTx(ctx)
	po.PONumber, _ = repo.GetNextPONumber(ctx)
	_ = repo.CreatePurchaseOrder(ctx, tx, po, items)
	_ = tx.Commit(ctx)

	fetchedPO, _ := repo.GetPurchaseOrderByID(ctx, po.ID, nil)

	tx, _ = repo.BeginTx(ctx)
	_ = repo.LockPurchaseOrderForUpdate(ctx, tx, po.ID)
	_ = repo.ConfirmPurchaseOrder(ctx, tx, po.ID, userID, "2026-07-27T10:00:00+07:00")
	_ = tx.Commit(ctx)

	grNumber, _ := repo.GetNextGRNumber(ctx)
	gr := &GoodsReceipt{
		GRNumber:        grNumber,
		PurchaseOrderID: po.ID,
		StoreID:         1,
		ReceivedBy:      userID,
	}
	grItems := []GoodsReceiptItem{{
		PurchaseOrderItemID: fetchedPO.Items[0].ID,
		ProductID:           prodID,
		QtyGood:             5,
		QtyDamaged:          1,
		UnitCost:            8000,
		ProductName:         "Handler Receipts",
	}}
	tx, _ = repo.BeginTx(ctx)
	_ = repo.LockPurchaseOrderForUpdate(ctx, tx, po.ID)
	_ = repo.CreateGoodsReceipt(ctx, tx, gr, grItems)
	_ = repo.UpdatePOItemQtyReceived(ctx, tx, fetchedPO.Items[0].ID, 6)
	_ = repo.RecalculatePOStatus(ctx, tx, po.ID)
	_ = tx.Commit(ctx)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("GET", "/api/purchase-orders/"+strconv.Itoa(po.ID)+"/receipts", nil)
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	require.Len(t, data, 1)
	rcpt := data[0].(map[string]interface{})
	assert.Equal(t, float64(5), rcpt["items"].([]interface{})[0].(map[string]interface{})["qty_good"])
}

func TestHandler_GetReceipts_InvalidID(t *testing.T) {
	r, _, _ := setupHandlerTest(t)

	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("GET", "/api/purchase-orders/abc/receipts", nil)
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateGoodsReceipt(t *testing.T) {
	r, _, _ := setupHandlerTest(t)
	ctx := context.Background()
	supplierID := insertTestSupplier(ctx, t, "Handler GR Supplier")
	prodID := insertTestProduct(ctx, t, "HANDLER-GR", "Handler GR", 10000, 200)

	repo := NewRepository(dbPool)
	userID := 1
	po := &Order{SupplierID: supplierID, StoreID: 1, Status: StatusDraft, CreatedBy: userID, UpdatedBy: userID}
	items := []OrderItem{{ProductID: prodID, QtyOrdered: 10, UnitCost: 8000, ProductName: "Handler GR", SKU: "HANDLER-GR"}}
	tx, _ := repo.BeginTx(ctx)
	po.PONumber, _ = repo.GetNextPONumber(ctx)
	_ = repo.CreatePurchaseOrder(ctx, tx, po, items)
	_ = tx.Commit(ctx)

	fetchedPO, _ := repo.GetPurchaseOrderByID(ctx, po.ID, nil)

	tx, _ = repo.BeginTx(ctx)
	_ = repo.LockPurchaseOrderForUpdate(ctx, tx, po.ID)
	_ = repo.ConfirmPurchaseOrder(ctx, tx, po.ID, userID, "2026-07-27T10:00:00+07:00")
	_ = tx.Commit(ctx)

	req := map[string]interface{}{
		"purchase_order_id": po.ID,
		"items": []map[string]interface{}{
			{"purchase_order_item_id": fetchedPO.Items[0].ID, "qty_good": 7, "qty_damaged": 1},
		},
	}
	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/api/goods-receipts", bytes.NewReader(body))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.NotEmpty(t, data["gr_number"])
}

func TestHandler_CreateGoodsReceipt_InvalidJSON(t *testing.T) {
	r, _, _ := setupHandlerTest(t)

	body := []byte(`{invalid json}`)
	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/api/goods-receipts", bytes.NewReader(body))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateGoodsReceipt_OverReceiveFails(t *testing.T) {
	r, _, _ := setupHandlerTest(t)
	ctx := context.Background()
	supplierID := insertTestSupplier(ctx, t, "Handler GR Over Supplier")
	prodID := insertTestProduct(ctx, t, "HANDLER-GROV", "Handler GR Over", 10000, 200)

	repo := NewRepository(dbPool)
	userID := 1
	po := &Order{SupplierID: supplierID, StoreID: 1, Status: StatusDraft, CreatedBy: userID, UpdatedBy: userID}
	items := []OrderItem{{ProductID: prodID, QtyOrdered: 5, UnitCost: 8000, ProductName: "Handler GR Over", SKU: "HANDLER-GROV"}}
	tx, _ := repo.BeginTx(ctx)
	po.PONumber, _ = repo.GetNextPONumber(ctx)
	_ = repo.CreatePurchaseOrder(ctx, tx, po, items)
	_ = tx.Commit(ctx)

	fetchedPO, _ := repo.GetPurchaseOrderByID(ctx, po.ID, nil)

	tx, _ = repo.BeginTx(ctx)
	_ = repo.LockPurchaseOrderForUpdate(ctx, tx, po.ID)
	_ = repo.ConfirmPurchaseOrder(ctx, tx, po.ID, userID, "2026-07-27T10:00:00+07:00")
	_ = tx.Commit(ctx)

	req := map[string]interface{}{
		"purchase_order_id": po.ID,
		"items": []map[string]interface{}{
			{"purchase_order_item_id": fetchedPO.Items[0].ID, "qty_good": 10, "qty_damaged": 0},
		},
	}
	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/api/goods-receipts", bytes.NewReader(body))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateGoodsReceipt_POItemNotFound(t *testing.T) {
	r, _, _ := setupHandlerTest(t)
	ctx := context.Background()
	supplierID := insertTestSupplier(ctx, t, "Handler GR NotFound Supplier")
	prodID := insertTestProduct(ctx, t, "HANDLER-GRNF", "Handler GR NotFound", 10000, 200)

	repo := NewRepository(dbPool)
	userID := 1
	po := &Order{SupplierID: supplierID, StoreID: 1, Status: StatusDraft, CreatedBy: userID, UpdatedBy: userID}
	items := []OrderItem{{ProductID: prodID, QtyOrdered: 5, UnitCost: 8000, ProductName: "Handler GR NotFound", SKU: "HANDLER-GRNF"}}
	tx, _ := repo.BeginTx(ctx)
	po.PONumber, _ = repo.GetNextPONumber(ctx)
	_ = repo.CreatePurchaseOrder(ctx, tx, po, items)
	_ = tx.Commit(ctx)

	tx, _ = repo.BeginTx(ctx)
	_ = repo.LockPurchaseOrderForUpdate(ctx, tx, po.ID)
	_ = repo.ConfirmPurchaseOrder(ctx, tx, po.ID, userID, "2026-07-27T10:00:00+07:00")
	_ = tx.Commit(ctx)

	req := map[string]interface{}{
		"purchase_order_id": po.ID,
		"items": []map[string]interface{}{
			{"purchase_order_item_id": 99999, "qty_good": 1, "qty_damaged": 0},
		},
	}
	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	reqHTTP, _ := http.NewRequest("POST", "/api/goods-receipts", bytes.NewReader(body))
	reqHTTP.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, reqHTTP)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
