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

	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/permissions"
)

type handlerOpts struct {
	withUser  bool
	withStore bool
	denyPerm  bool
}

func setupHandlerOpt(t *testing.T, opts handlerOpts) *gin.Engine {
	t.Helper()
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	t.Cleanup(bus.Shutdown)

	svc := newWiredService(repo, bus)
	auditSvc := &mockAuditCreator{}
	handler := NewHandler(svc, auditSvc)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	auth := func(c *gin.Context) {
		if opts.withUser {
			c.Set("userID", 1)
			c.Set("username", "testuser")
			c.Set("role", "admin")
		}
		if opts.withStore {
			c.Set("storeID", intPtr(1))
		}
		c.Next()
	}
	perm := func(code permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) {
			if opts.denyPerm {
				sharedJSONError403(c)
				return
			}
			c.Next()
		}
	}

	api := r.Group("/api")
	handler.RegisterRoutes(api, auth, perm)
	return r
}

func sharedJSONError403(c *gin.Context) {
	c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "forbidden", "message": "forbidden"}})
}

func postJSON(t *testing.T, r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		buf.Write(b)
	}
	req, _ := http.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestHandler_UnauthorizedBranches(t *testing.T) {
	r := setupHandlerOpt(t, handlerOpts{withUser: false, withStore: true})

	poBody := map[string]interface{}{
		"supplier_id": 1,
		"items":       []map[string]interface{}{{"product_id": 1, "qty_ordered": 1, "unit_cost": 100}},
	}
	grBody := map[string]interface{}{
		"purchase_order_id": 1,
		"items":             []map[string]interface{}{{"purchase_order_item_id": 1, "qty_good": 1}},
	}
	cases := []struct {
		method, path string
		body         interface{}
	}{
		{"POST", "/api/purchase-orders", poBody},
		{"PUT", "/api/purchase-orders/1", poBody},
		{"DELETE", "/api/purchase-orders/1", nil},
		{"POST", "/api/purchase-orders/1/confirm", nil},
		{"POST", "/api/purchase-orders/1/cancel", nil},
		{"POST", "/api/goods-receipts", grBody},
	}
	for _, c := range cases {
		w := postJSON(t, r, c.method, c.path, c.body)
		assert.Equal(t, http.StatusUnauthorized, w.Code, c.method+" "+c.path)
	}
}

func TestHandler_PermissionDeniedBranches(t *testing.T) {
	r := setupHandlerOpt(t, handlerOpts{withUser: true, withStore: true, denyPerm: true})

	cases := []struct {
		method, path string
	}{
		{"POST", "/api/purchase-orders"},
		{"GET", "/api/purchase-orders"},
		{"PUT", "/api/purchase-orders/1"},
		{"DELETE", "/api/purchase-orders/1"},
		{"POST", "/api/purchase-orders/1/confirm"},
		{"POST", "/api/purchase-orders/1/cancel"},
		{"POST", "/api/goods-receipts"},
	}
	for _, c := range cases {
		w := postJSON(t, r, c.method, c.path, map[string]interface{}{})
		assert.Equal(t, http.StatusForbidden, w.Code, c.method+" "+c.path)
	}
}

func TestHandler_CreateDraft_InvalidJSON(t *testing.T) {
	r := setupHandlerOpt(t, handlerOpts{withUser: true, withStore: true})
	w := postJSON(t, r, "POST", "/api/purchase-orders", "{not-json")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateDraft_MissingStoreID(t *testing.T) {
	r := setupHandlerOpt(t, handlerOpts{withUser: true, withStore: false})
	ctx := context.Background()
	supplierID := insertTestSupplier(ctx, t, "H NoStore Supplier")
	prodID := insertTestProduct(ctx, t, "H-NOSTORE-001", "NoStore Product", 10000, 100)

	req := map[string]interface{}{
		"supplier_id": supplierID,
		"items":       []map[string]interface{}{{"product_id": prodID, "qty_ordered": 1, "unit_cost": 1000}},
	}
	w := postJSON(t, r, "POST", "/api/purchase-orders", req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateDraft_ValidationError(t *testing.T) {
	r := setupHandlerOpt(t, handlerOpts{withUser: true, withStore: true})
	ctx := context.Background()
	supplierID := insertTestSupplier(ctx, t, "H SvcErr Supplier")
	prodID := insertTestProduct(ctx, t, "H-SVCERR-001", "SvcErr Product", 10000, 100)

	req := map[string]interface{}{
		"supplier_id": supplierID,
		"items":       []map[string]interface{}{{"product_id": prodID, "qty_ordered": 0, "unit_cost": 1000}},
	}
	w := postJSON(t, r, "POST", "/api/purchase-orders", req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateDraft_InvalidJSON(t *testing.T) {
	r := setupHandlerOpt(t, handlerOpts{withUser: true, withStore: true})
	w := postJSON(t, r, "PUT", "/api/purchase-orders/1", "{not-json")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateDraft_NotFound(t *testing.T) {
	r := setupHandlerOpt(t, handlerOpts{withUser: true, withStore: true})
	req := map[string]interface{}{
		"supplier_id": 1,
		"items":       []map[string]interface{}{{"product_id": 1, "qty_ordered": 1, "unit_cost": 1000}},
	}
	w := postJSON(t, r, "PUT", "/api/purchase-orders/999999", req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_ConfirmPO_NotFound(t *testing.T) {
	r := setupHandlerOpt(t, handlerOpts{withUser: true, withStore: true})
	w := postJSON(t, r, "POST", "/api/purchase-orders/999999/confirm", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_ConfirmPO_AlreadyConfirmed(t *testing.T) {
	r, _, _ := setupHandlerTest(t)
	ctx := context.Background()
	supplierID := insertTestSupplier(ctx, t, "H ConfirmX2 Supplier")
	prodID := insertTestProduct(ctx, t, "H-CONF2-001", "ConfirmX2 Product", 10000, 100)
	userID := insertTestUser(ctx, t, "po_conf2_user")

	req := map[string]interface{}{
		"supplier_id": supplierID,
		"items":       []map[string]interface{}{{"product_id": prodID, "qty_ordered": 5, "unit_cost": 8000}},
	}
	w := postJSON(t, r, "POST", "/api/purchase-orders", req)
	require.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	poID := int(resp["data"].(map[string]interface{})["id"].(float64))
	_ = userID

	w = postJSON(t, r, "POST", "/api/purchase-orders/"+strconv.Itoa(poID)+"/confirm", nil)
	require.Equal(t, http.StatusOK, w.Code)
	w = postJSON(t, r, "POST", "/api/purchase-orders/"+strconv.Itoa(poID)+"/confirm", nil)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHandler_CancelPO_NotFound(t *testing.T) {
	r := setupHandlerOpt(t, handlerOpts{withUser: true, withStore: true})
	w := postJSON(t, r, "POST", "/api/purchase-orders/999999/cancel", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_CancelPO_AlreadyCancelled(t *testing.T) {
	r, _, _ := setupHandlerTest(t)
	ctx := context.Background()
	supplierID := insertTestSupplier(ctx, t, "H CancelX2 Supplier")
	prodID := insertTestProduct(ctx, t, "H-CANC2-001", "CancelX2 Product", 10000, 100)

	req := map[string]interface{}{
		"supplier_id": supplierID,
		"items":       []map[string]interface{}{{"product_id": prodID, "qty_ordered": 5, "unit_cost": 8000}},
	}
	w := postJSON(t, r, "POST", "/api/purchase-orders", req)
	require.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	poID := int(resp["data"].(map[string]interface{})["id"].(float64))

	w = postJSON(t, r, "POST", "/api/purchase-orders/"+strconv.Itoa(poID)+"/cancel", nil)
	require.Equal(t, http.StatusOK, w.Code)
	w = postJSON(t, r, "POST", "/api/purchase-orders/"+strconv.Itoa(poID)+"/cancel", nil)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHandler_CreateGoodsReceipt_MissingStoreID(t *testing.T) {
	r := setupHandlerOpt(t, handlerOpts{withUser: true, withStore: false})
	req := map[string]interface{}{
		"purchase_order_id": 1,
		"items":             []map[string]interface{}{},
	}
	w := postJSON(t, r, "POST", "/api/goods-receipts", req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
