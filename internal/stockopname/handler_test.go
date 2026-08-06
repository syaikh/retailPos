package stockopname

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/inventory"
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/permissions"
	"retail-pos-system/internal/product"
)

func skipIfNoDB(t *testing.T) {
	t.Helper()
	if dbPool == nil {
		t.Skip("no database connection")
	}
}

func setupStockOpnameRouter() *gin.Engine {
	return setupStockOpnameRouterAs(1, "superadmin")
}

func setupStockOpnameRouterAs(userID int, role string) *gin.Engine {
	gin.SetMode(gin.TestMode)

	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()

	svc := NewService(repo, bus)
	svc.SetStockApplier(inventory.StockApplier{StockSyncer: product.StockSyncer{}})
	auditSvc := audit.NewService(audit.NewRepository(dbPool))
	h := NewHandler(svc, auditSvc)

	r := gin.New()
	auth := func(c *gin.Context) {
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, middleware.CtxKeyUserID, userID)
		ctx = context.WithValue(ctx, middleware.CtxKeyUsername, "testuser")
		ctx = context.WithValue(ctx, middleware.CtxKeyRole, role)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
	perm := func(code permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	}
	h.RegisterRoutes(r.Group("/"), auth, perm)
	return r
}

func postJSON(t *testing.T, r *gin.Engine, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	return requestJSON(t, r, http.MethodPost, path, body)
}

func putJSON(t *testing.T, r *gin.Engine, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	return requestJSON(t, r, http.MethodPut, path, body)
}

func requestJSON(t *testing.T, r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func getPath(t *testing.T, r *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", path, nil)
	r.ServeHTTP(w, req)
	return w
}

func createHandlerSession(ctx context.Context, t *testing.T, r *gin.Engine, scopeType string, scopeID int64) int {
	t.Helper()
	body := fmt.Sprintf(`{"scope_type":%q,"scope_id":%d}`, scopeType, scopeID)
	w := postJSON(t, r, "/stock-opnames", body)
	require.Equal(t, http.StatusCreated, w.Code)
	var resp struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp.Data.ID
}

func TestHandler_CreateSession(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	resetStockOpname(ctx, t)
	insertTestUserWithRole(ctx, t, 9601, "so_hdl_u9601", 3)
	insertTestStore(ctx, t, 9601)
	p := insertTestProductStore(ctx, t, "SO-HDL-CREATE-001", 9601)
	insertTestStock(ctx, t, p, 10)

	t.Run("store scope creates session", func(t *testing.T) {
		r := setupStockOpnameRouter()
		w := postJSON(t, r, "/stock-opnames", `{"scope_type":"store","scope_id":9601}`)
		assert.Equal(t, http.StatusCreated, w.Code)
		var resp struct {
			Data Session `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "store", resp.Data.ScopeType)
		require.NotNil(t, resp.Data.StoreID)
		assert.Equal(t, 9601, *resp.Data.StoreID)
	})

	t.Run("unsupported scope returns 400", func(t *testing.T) {
		resetStockOpname(ctx, t)
		r := setupStockOpnameRouter()
		w := postJSON(t, r, "/stock-opnames", `{"scope_type":"bogus","scope_id":9601}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "SO-401")
	})

	t.Run("invalid json returns 400", func(t *testing.T) {
		resetStockOpname(ctx, t)
		r := setupStockOpnameRouter()
		w := postJSON(t, r, "/stock-opnames", "{invalid")
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing scope_id returns 400", func(t *testing.T) {
		resetStockOpname(ctx, t)
		r := setupStockOpnameRouter()
		w := postJSON(t, r, "/stock-opnames", `{"scope_type":"store"}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("overlapping active session returns 409", func(t *testing.T) {
		resetStockOpname(ctx, t)
		r := setupStockOpnameRouter()
		createHandlerSession(ctx, t, r, "store", 9601)
		w := postJSON(t, r, "/stock-opnames", `{"scope_type":"store","scope_id":9601}`)
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "SO-405")
	})

	t.Run("store scope for missing store returns error", func(t *testing.T) {
		resetStockOpname(ctx, t)
		r := setupStockOpnameRouter()
		w := postJSON(t, r, "/stock-opnames", `{"scope_type":"store","scope_id":989999}`)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestHandler_ListAndGetSession(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	resetStockOpname(ctx, t)
	insertTestUserWithRole(ctx, t, 9602, "so_hdl_u9602", 3)
	insertTestStore(ctx, t, 9602)
	p := insertTestProductStore(ctx, t, "SO-HDL-LIST-001", 9602)
	insertTestStock(ctx, t, p, 10)
	r := setupStockOpnameRouter()
	sessionID := createHandlerSession(ctx, t, r, "store", 9602)

	t.Run("list sessions returns the created session", func(t *testing.T) {
		w := getPath(t, r, "/stock-opnames")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "SO-")
	})

	t.Run("get session returns session", func(t *testing.T) {
		w := getPath(t, r, fmt.Sprintf("/stock-opnames/%d", sessionID))
		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data Session `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, sessionID, resp.Data.ID)
		require.NotNil(t, resp.Data.StoreID)
		assert.Equal(t, 9602, *resp.Data.StoreID)
	})

	t.Run("get session with invalid id returns 400", func(t *testing.T) {
		w := getPath(t, r, "/stock-opnames/not-a-number")
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("get session with missing id returns 404", func(t *testing.T) {
		w := getPath(t, r, "/stock-opnames/999999")
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandler_AssignAndCountFlow(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	resetStockOpname(ctx, t)
	managerID := 9603
	counterID := 9604
	insertTestUserWithRole(ctx, t, managerID, "so_hdl_u9603", 3)
	insertTestUserWithRole(ctx, t, counterID, "so_hdl_u9604", 5)
	insertTestStore(ctx, t, 9603)
	p := insertTestProductStore(ctx, t, "SO-HDL-FLOW-001", 9603)
	insertTestStock(ctx, t, p, 10)

	managerRouter := setupStockOpnameRouterAs(managerID, "manager")
	counterRouter := setupStockOpnameRouterAs(counterID, "cashier")
	sessionID := createHandlerSession(ctx, t, managerRouter, "store", 9603)

	// fetch a real item id so count operations target an existing item
	w := getPath(t, managerRouter, fmt.Sprintf("/stock-opnames/%d", sessionID))
	require.Equal(t, http.StatusOK, w.Code)
	var sessionResp struct {
		Data Session `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &sessionResp))
	require.NotEmpty(t, sessionResp.Data.Items)
	itemID := sessionResp.Data.Items[0].ID

	t.Run("assign counter", func(t *testing.T) {
		body := fmt.Sprintf(`{"user_id":%d,"role":"counter"}`, counterID)
		w := postJSON(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/assignments", sessionID), body)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("assign manager as counter returns 400", func(t *testing.T) {
		body := fmt.Sprintf(`{"user_id":%d,"role":"counter"}`, managerID)
		w := postJSON(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/assignments", sessionID), body)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "SO-104")
	})

	t.Run("list assignments", func(t *testing.T) {
		w := getPath(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/assignments", sessionID))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), strconv.Itoa(counterID))
	})

	t.Run("non-assigned user cannot start counting", func(t *testing.T) {
		w := postJSON(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/start", sessionID), "")
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("counter starts counting", func(t *testing.T) {
		w := postJSON(t, counterRouter, fmt.Sprintf("/stock-opnames/%d/start", sessionID), "")
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("submit before counting all items returns 409", func(t *testing.T) {
		w := postJSON(t, counterRouter, fmt.Sprintf("/stock-opnames/%d/submit", sessionID), "")
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "SO-201")
	})

	t.Run("save count as assigned counter", func(t *testing.T) {
		w := putJSON(t, counterRouter, fmt.Sprintf("/stock-opnames/items/%d/count", itemID), `{"physical_qty":5,"remarks":"ok"}`)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("count history for counted item", func(t *testing.T) {
		w := getPath(t, managerRouter, fmt.Sprintf("/stock-opnames/items/%d/counts", itemID))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "5")
	})

	t.Run("submit session to verification", func(t *testing.T) {
		w := postJSON(t, counterRouter, fmt.Sprintf("/stock-opnames/%d/submit", sessionID), "")
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("verify without comment returns 422", func(t *testing.T) {
		w := postJSON(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/verify", sessionID), `{"comment":""}`)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), "SO-402")
	})

	t.Run("manager verifies session", func(t *testing.T) {
		w := postJSON(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/verify", sessionID), `{"comment":"ok"}`)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("post adjustment writes ledger document", func(t *testing.T) {
		w := postJSON(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/post-adjustment", sessionID), `{"comment":"ok"}`)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "IA-")
	})

	t.Run("recount on non-pending session returns 409", func(t *testing.T) {
		w := postJSON(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/recount", sessionID), `{"comment":"recheck"}`)
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("summary returns counts", func(t *testing.T) {
		w := getPath(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/summary", sessionID))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("difference report returns session", func(t *testing.T) {
		w := getPath(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/difference", sessionID))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("export returns csv", func(t *testing.T) {
		w := getPath(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/export", sessionID))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/csv")
		assert.Contains(t, w.Body.String(), "session_number")
	})

	t.Run("close session", func(t *testing.T) {
		w := postJSON(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/close", sessionID), "")
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("assignable users returns users", func(t *testing.T) {
		w := getPath(t, managerRouter, "/stock-opnames/assignable-users")
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
