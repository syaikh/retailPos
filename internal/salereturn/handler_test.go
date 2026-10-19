package salereturn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/events"
	"retail-pos-system/internal/permissions"
	"retail-pos-system/internal/shared"
)

// ==================== RESPONSE DTOs ====================
// The handlers emit map[string]interface{} through presentReturn; decoding
// into typed mirrors keeps the assertions on the JSON wire contract (field
// names, nesting, the null approval) rather than on the presenter internals.

type returnItemDTO struct {
	ID       int `json:"id"`
	Quantity int `json:"quantity"`
	Subtotal int `json:"subtotal"`
}

type returnPaymentDTO struct {
	Amount int `json:"amount"`
}

type returnApprovalDTO struct {
	Status         string  `json:"status"`
	RejectedReason *string `json:"rejected_reason"`
}

type returnDTO struct {
	ID                int                `json:"id"`
	ReturnNumber      string             `json:"return_number"`
	CashierID         int                `json:"cashier_id"`
	ReturnType        string             `json:"return_type"`
	Status            string             `json:"status"`
	TotalRefundAmount int                `json:"total_refund_amount"`
	Items             []returnItemDTO    `json:"items"`
	Payments          []returnPaymentDTO `json:"payments"`
	Approval          *returnApprovalDTO `json:"approval"`
}

type returnDataResponse struct {
	Data returnDTO `json:"data"`
}

type returnListResponse struct {
	Data       []returnDTO `json:"data"`
	Total      int         `json:"total"`
	Limit      int         `json:"limit"`
	Offset     int         `json:"offset"`
	TotalPages int         `json:"total_pages"`
	HasMore    bool        `json:"has_more"`
}

type returnErrorBody struct {
	Error string `json:"error"`
}

// ==================== ROUTER HELPERS ====================

// returnAuth stands in for the production auth chain (cmd/server wires
// noopAuth + middleware.RequirePermission). It sets exactly the gin keys the
// handlers read: "userID" as an int for middleware.GetUserID (a present but
// non-int value panics there, matching production) and "storeID" as a *int
// for shared.GetStoreID; username/role live in the request context in
// production and are absent here, so actorFrom leaves them empty.
type returnAuth struct {
	userID  int
	storeID *int
}

func (a *returnAuth) middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userID", a.userID)
		c.Set("username", "salereturn_test")
		c.Set("role", "cashier")
		c.Set("storeID", a.storeID)
		c.Next()
	}
}

func returnPermPassthrough(_ permissions.Code) gin.HandlerFunc {
	return func(c *gin.Context) { c.Next() }
}

// setupReturnRoutes builds a fresh service + router per (sub)test so port
// overrides made through the returned *Service cannot leak into a sibling
// test. A nil auth becomes the plain "no store scope" default.
func setupReturnRoutes(t *testing.T, f *seeded, stock map[int]int, threshold string, auth *returnAuth) (*gin.Engine, *Service, *stubBus, *stubAdjuster) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	svc, bus, adjuster := newFlowService(f, stock, &stubDeduct{}, threshold)
	if auth == nil {
		auth = &returnAuth{userID: f.userID}
	}
	r := gin.New()
	NewHandler(svc).RegisterRoutes(r.Group("/"), auth.middleware(), returnPermPassthrough)
	return r, svc, bus, adjuster
}

func seedReturnScenario(t *testing.T) *seeded {
	t.Helper()
	_ = shared.TruncateTestData(dbPool)
	return seedSaleFixture(t)
}

func doRequest(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	r.ServeHTTP(w, req)
	return w
}

func decodeJSONBody(t *testing.T, w *httptest.ResponseRecorder, dst interface{}) {
	t.Helper()
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), dst), "unexpected response body: %s", w.Body.String())
}

func decodeReturnData(t *testing.T, w *httptest.ResponseRecorder) returnDTO {
	t.Helper()
	var resp returnDataResponse
	decodeJSONBody(t, w, &resp)
	return resp.Data
}

func decodeReturnList(t *testing.T, w *httptest.ResponseRecorder) returnListResponse {
	t.Helper()
	var resp returnListResponse
	decodeJSONBody(t, w, &resp)
	return resp
}

func errorBody(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var body returnErrorBody
	decodeJSONBody(t, w, &body)
	return body.Error
}

// ==================== DB WITNESSES ====================

func countSaleReturns(t *testing.T) int {
	t.Helper()
	var count int
	require.NoError(t, dbPool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM sale_returns`).Scan(&count))
	return count
}

func returnRowStatus(t *testing.T, returnID int) string {
	t.Helper()
	var status string
	require.NoError(t, dbPool.QueryRow(context.Background(),
		`SELECT status FROM sale_returns WHERE id = $1`, returnID).Scan(&status))
	return status
}

// approvalRowStatus reads the decision row directly: the approve/reject
// responses carry the Approval snapshot loaded BEFORE the status flip, so
// the database is the only trustworthy witness of the recorded decision.
func approvalRowStatus(t *testing.T, returnID int) (status, rejectedReason string) {
	t.Helper()
	require.NoError(t, dbPool.QueryRow(context.Background(), `
		SELECT status, COALESCE(rejected_reason, '') FROM return_approvals
		WHERE sale_return_id = $1
	`, returnID).Scan(&status, &rejectedReason))
	return status, rejectedReason
}

// ==================== PURE UNIT TESTS ====================

func TestPathID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name  string
		param string
		want  int
		ok    bool
	}{
		{name: "positive integer", param: "12", want: 12, ok: true},
		{name: "zero is rejected", param: "0"},
		{name: "negative is rejected", param: "-3"},
		{name: "non-numeric", param: "abc"},
		{name: "empty", param: ""},
		{name: "float", param: "1.5"},
		{name: "overflow", param: "99999999999999999999"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Params = gin.Params{{Key: "id", Value: tc.param}}
			id, ok := pathID(c, "id")
			assert.Equal(t, tc.ok, ok)
			if tc.ok {
				assert.Equal(t, tc.want, id)
			} else {
				assert.Zero(t, id)
			}
		})
	}
}

func TestOptionalStatus(t *testing.T) {
	assert.Nil(t, optionalStatus(""), "an absent filter must not narrow the query")
	got := optionalStatus("completed")
	require.NotNil(t, got)
	assert.Equal(t, "completed", *got)
}

func TestRespondError_HTTPStatusMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// The handler under test reads the error from a closure variable so every
	// mapping case reuses the same engine.
	var current error
	r := gin.New()
	r.GET("/boom", func(c *gin.Context) { respondError(c, current) })

	tests := []struct {
		name string
		err  error
		want int
	}{
		// 404 — unknown or hidden by store scope.
		{"sale not found", ErrSaleNotFound, http.StatusNotFound},
		{"return not found", ErrReturnNotFound, http.StatusNotFound},
		{"store mismatch", ErrStoreMismatch, http.StatusNotFound},
		{"wrapped return not found", fmt.Errorf("%w: sale 9", ErrReturnNotFound), http.StatusNotFound},
		// 409 — the request was well formed but conflicts with state.
		{"not pending", ErrReturnNotPending, http.StatusConflict},
		{"approval expired", ErrApprovalExpired, http.StatusConflict},
		{"shift not open", ErrShiftNotOpen, http.StatusConflict},
		{"stock changed", ErrStockChanged, http.StatusConflict},
		// 400 — everything else, verbatim domain message.
		{"invalid return", ErrInvalidReturn, http.StatusBadRequest},
		{"wrapped invalid return", fmt.Errorf("%w: reason is required", ErrInvalidReturn), http.StatusBadRequest},
		{"sale item not found", ErrSaleItemNotFound, http.StatusBadRequest},
		{"qty exceeds sold", ErrQtyExceedsSold, http.StatusBadRequest},
		{"consignment unsupported", ErrConsignmentNotSupported, http.StatusBadRequest},
		{"sale not completed", ErrSaleNotCompleted, http.StatusBadRequest},
		{"return window expired", ErrReturnWindowExpired, http.StatusBadRequest},
		{"condition unsupported", ErrConditionUnsupported, http.StatusBadRequest},
		{"unknown error", errors.New("boom"), http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			current = tc.err
			w := doRequest(r, http.MethodGet, "/boom", "")
			assert.Equal(t, tc.want, w.Code)
			assert.Equal(t, tc.err.Error(), errorBody(t, w), "the domain message must reach the client verbatim")
		})
	}
}

// gin must register exactly the six PRD endpoints — a static/wildcard
// collision on /sales/returns/... would panic here, and a route silently
// dropped or renamed would fail the set comparison.
func TestRegisterRoutes_AllEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(NewService(NewRepository(dbPool), &stubBus{}))
	r := gin.New()
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, returnPermPassthrough)

	want := []string{
		"POST /sales/:id/return",
		"GET /sales/:id/returns",
		"GET /sales/returns/pending",
		"GET /sales/returns/:returnId",
		"POST /sales/returns/:returnId/approve",
		"POST /sales/returns/:returnId/reject",
	}
	got := make(map[string]bool)
	for _, route := range r.Routes() {
		got[route.Method+" "+route.Path] = true
	}
	assert.Len(t, got, len(want), "exactly six endpoints must be registered, got %v", got)
	for _, route := range want {
		assert.True(t, got[route], "missing route %s", route)
	}
	for route := range got {
		assert.Contains(t, want, route, "unexpected route %s", route)
	}
}

func TestRegisterRoutes_AuthPermGuards(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(NewService(NewRepository(dbPool), &stubBus{}))
	endpoints := []struct {
		method string
		path   string
		code   permissions.Code
	}{
		{http.MethodPost, "/sales/1/return", permissions.SaleReturn},
		{http.MethodGet, "/sales/1/returns", permissions.SaleView},
		{http.MethodGet, "/sales/returns/pending", permissions.SaleReturnApprove},
		{http.MethodGet, "/sales/returns/1", permissions.SaleView},
		{http.MethodPost, "/sales/returns/1/approve", permissions.SaleReturnApprove},
		{http.MethodPost, "/sales/returns/1/reject", permissions.SaleReturnApprove},
	}

	t.Run("every endpoint requires authentication", func(t *testing.T) {
		r := gin.New()
		denied := func(c *gin.Context) { c.AbortWithStatus(http.StatusUnauthorized) }
		h.RegisterRoutes(r.Group("/"), denied, returnPermPassthrough)
		for _, tc := range endpoints {
			w := doRequest(r, tc.method, tc.path, "")
			assert.Equal(t, http.StatusUnauthorized, w.Code, "%s %s", tc.method, tc.path)
		}
	})

	t.Run("every endpoint requires its own permission code", func(t *testing.T) {
		var recorded []permissions.Code
		perm := func(code permissions.Code) gin.HandlerFunc {
			return func(c *gin.Context) {
				recorded = append(recorded, code)
				c.AbortWithStatus(http.StatusForbidden)
			}
		}
		r := gin.New()
		h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, perm)
		for _, tc := range endpoints {
			recorded = nil
			w := doRequest(r, tc.method, tc.path, "")
			assert.Equal(t, http.StatusForbidden, w.Code, "%s %s", tc.method, tc.path)
			require.Len(t, recorded, 1, "%s %s", tc.method, tc.path)
			assert.Equal(t, tc.code, recorded[0], "%s %s", tc.method, tc.path)
		}
	})
}

// ==================== POST /sales/:id/return ====================

func TestProcessReturnHandler_HappyPath(t *testing.T) {
	f := seedReturnScenario(t)
	// Stock is gone, so the line refunds its full tax-inclusive value (5000),
	// well under the 500000 threshold: the return completes immediately.
	r, _, bus, adjuster := setupReturnRoutes(t, f, map[int]int{f.productID: 0}, "500000", nil)

	body := fmt.Sprintf(`{"reason":"barang rusak","condition":"damaged","items":[{"sale_item_id":%d,"quantity":1}]}`, f.itemID)
	w := doRequest(r, http.MethodPost, fmt.Sprintf("/sales/%d/return", f.saleID), body)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	data := decodeReturnData(t, w)
	assert.Positive(t, data.ID)
	assert.NotEmpty(t, data.ReturnNumber)
	assert.Equal(t, ReturnTypeRefund, data.ReturnType)
	assert.Equal(t, ReturnStatusCompleted, data.Status)
	assert.Equal(t, f.userID, data.CashierID, "the return must hang off the processing cashier")
	assert.Equal(t, 5000, data.TotalRefundAmount)
	require.Len(t, data.Items, 1)
	assert.Equal(t, 1, data.Items[0].Quantity)
	assert.Equal(t, 5000, data.Items[0].Subtotal)
	require.Len(t, data.Payments, 1, "the response must carry the refund split")
	assert.Equal(t, 5000, data.Payments[0].Amount)
	assert.Nil(t, data.Approval, "an immediate refund needs no approval record")

	require.Equal(t, 1, adjuster.calls, "the cashier's shift totals must absorb the refund")
	require.Len(t, bus.topics, 1, "sale.returned.v1 published once after commit")
	assert.Equal(t, events.TopicSaleReturned, bus.topics[0])
	assert.Equal(t, 1, countSaleReturns(t))
}

func TestProcessReturnHandler_ValidationFailures(t *testing.T) {
	f := seedReturnScenario(t)
	r, _, _, _ := setupReturnRoutes(t, f, map[int]int{f.productID: 0}, "500000", nil)

	saleURL := fmt.Sprintf("/sales/%d/return", f.saleID)
	items := fmt.Sprintf(`"items":[{"sale_item_id":%d,"quantity":1}]`, f.itemID)
	valid := fmt.Sprintf(`{"reason":"barang rusak","condition":"damaged",%s}`, items)

	tests := []struct {
		name    string
		path    string
		body    string
		wantErr string
	}{
		{"non-numeric sale id", "/sales/abc/return", valid, ErrInvalidReturn.Error()},
		{"zero sale id", "/sales/0/return", valid, ErrInvalidReturn.Error()},
		{"empty body", saleURL, "", "invalid request body"},
		{"malformed json", saleURL, `{"reason":`, "invalid request body"},
		{"missing reason", saleURL, fmt.Sprintf(`{"condition":"damaged",%s}`, items),
			"invalid return request: reason is required"},
		{"no items", saleURL, `{"reason":"barang rusak","condition":"damaged","items":[]}`,
			"invalid return request: no items"},
		{"functional condition", saleURL, fmt.Sprintf(`{"reason":"x","condition":"functional",%s}`, items),
			ErrConditionUnsupported.Error()},
		{"missing condition", saleURL, fmt.Sprintf(`{"reason":"x",%s}`, items),
			ErrConditionUnsupported.Error()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := doRequest(r, http.MethodPost, tc.path, tc.body)
			assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			assert.Equal(t, tc.wantErr, errorBody(t, w))
		})
	}

	assert.Zero(t, countSaleReturns(t), "a rejected request must not persist a return")
}

func TestProcessReturnHandler_SaleAndScopeErrors(t *testing.T) {
	// Each subtest seeds its own scenario and router: they override ports on
	// the service, which must not leak into a sibling subtest.
	t.Run("missing sale is not found", func(t *testing.T) {
		f := seedReturnScenario(t)
		r, svc, _, _ := setupReturnRoutes(t, f, map[int]int{f.productID: 0}, "500000", nil)
		svc.SetSaleReader(stubSaleReader{sale: nil})

		w := doRequest(r, http.MethodPost, fmt.Sprintf("/sales/%d/return", f.saleID),
			fmt.Sprintf(`{"reason":"x","condition":"damaged","items":[{"sale_item_id":%d,"quantity":1}]}`, f.itemID))
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, ErrSaleNotFound.Error(), errorBody(t, w))
	})

	t.Run("another store's sale is not found", func(t *testing.T) {
		f := seedReturnScenario(t)
		// The seeded sale belongs to store 1; the caller scopes to store 2.
		r, _, _, _ := setupReturnRoutes(t, f, map[int]int{f.productID: 0}, "500000",
			&returnAuth{userID: f.userID, storeID: ptrInt(2)})

		w := doRequest(r, http.MethodPost, fmt.Sprintf("/sales/%d/return", f.saleID),
			fmt.Sprintf(`{"reason":"x","condition":"damaged","items":[{"sale_item_id":%d,"quantity":1}]}`, f.itemID))
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, ErrStoreMismatch.Error(), errorBody(t, w))
	})

	t.Run("no open shift is a conflict", func(t *testing.T) {
		f := seedReturnScenario(t)
		r, svc, _, _ := setupReturnRoutes(t, f, map[int]int{f.productID: 0}, "500000", nil)
		svc.SetActiveShiftFinder(stubShiftFinder{shiftID: 0})

		w := doRequest(r, http.MethodPost, fmt.Sprintf("/sales/%d/return", f.saleID),
			fmt.Sprintf(`{"reason":"x","condition":"damaged","items":[{"sale_item_id":%d,"quantity":1}]}`, f.itemID))
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Equal(t, ErrShiftNotOpen.Error(), errorBody(t, w))
	})

	t.Run("sale outside the return window is rejected", func(t *testing.T) {
		f := seedReturnScenario(t)
		r, svc, _, _ := setupReturnRoutes(t, f, map[int]int{f.productID: 0}, "500000", nil)
		stale := returnSaleFromSeed(f)
		stale.CreatedAt = time.Now().Add(-48 * time.Hour) // window: 1 day
		svc.SetSaleReader(stubSaleReader{sale: stale})

		w := doRequest(r, http.MethodPost, fmt.Sprintf("/sales/%d/return", f.saleID),
			fmt.Sprintf(`{"reason":"x","condition":"damaged","items":[{"sale_item_id":%d,"quantity":1}]}`, f.itemID))
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, ErrReturnWindowExpired.Error(), errorBody(t, w))
	})

	t.Run("unfinished sale is rejected", func(t *testing.T) {
		f := seedReturnScenario(t)
		r, svc, _, _ := setupReturnRoutes(t, f, map[int]int{f.productID: 0}, "500000", nil)
		held := returnSaleFromSeed(f)
		held.Status = "held"
		svc.SetSaleReader(stubSaleReader{sale: held})

		w := doRequest(r, http.MethodPost, fmt.Sprintf("/sales/%d/return", f.saleID),
			fmt.Sprintf(`{"reason":"x","condition":"damaged","items":[{"sale_item_id":%d,"quantity":1}]}`, f.itemID))
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, ErrSaleNotCompleted.Error(), errorBody(t, w))
	})

	t.Run("quantity above the sold amount is rejected", func(t *testing.T) {
		f := seedReturnScenario(t)
		r, _, _, _ := setupReturnRoutes(t, f, map[int]int{f.productID: 0}, "500000", nil)

		// The sale line holds 3 units; 5 exceeds the over-return guard.
		w := doRequest(r, http.MethodPost, fmt.Sprintf("/sales/%d/return", f.saleID),
			fmt.Sprintf(`{"reason":"x","condition":"damaged","items":[{"sale_item_id":%d,"quantity":5}]}`, f.itemID))
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, errorBody(t, w), "qty retur")
	})

	assert.Zero(t, countSaleReturns(t), "no failed attempt may persist a return")
}

// ==================== GET /sales/:id/returns ====================

func TestListSaleReturnsHandler(t *testing.T) {
	f := seedReturnScenario(t)
	repo := NewRepository(dbPool)
	_ = insertReturnWithItem(context.Background(), t, repo, f, ReturnStatusCompleted, nil)

	r, _, _, _ := setupReturnRoutes(t, f, nil, "500000", nil)
	saleURL := fmt.Sprintf("/sales/%d/returns", f.saleID)

	t.Run("lists the returns of the sale", func(t *testing.T) {
		w := doRequest(r, http.MethodGet, saleURL, "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		resp := decodeReturnList(t, w)
		assert.Equal(t, 1, resp.Total)
		require.Len(t, resp.Data, 1)
		assert.Positive(t, resp.Data[0].ID)
		assert.Equal(t, ReturnStatusCompleted, resp.Data[0].Status)
		assert.Equal(t, f.totalRefund, resp.Data[0].TotalRefundAmount)
	})

	t.Run("status filter narrows the list", func(t *testing.T) {
		w := doRequest(r, http.MethodGet, saleURL+"?status=completed", "")
		require.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 1, decodeReturnList(t, w).Total)

		w = doRequest(r, http.MethodGet, saleURL+"?status=cancelled", "")
		require.Equal(t, http.StatusOK, w.Code)
		resp := decodeReturnList(t, w)
		assert.Zero(t, resp.Total)
		assert.Empty(t, resp.Data)
	})

	t.Run("non-numeric sale id is a validation error", func(t *testing.T) {
		w := doRequest(r, http.MethodGet, "/sales/abc/returns", "")
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, ErrInvalidReturn.Error(), errorBody(t, w))
	})

	t.Run("offset beyond the cap is rejected", func(t *testing.T) {
		w := doRequest(r, http.MethodGet, saleURL+"?offset=10001", "")
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, errorBody(t, w), "offset must not exceed")
	})

	t.Run("limit and offset are clamped", func(t *testing.T) {
		w := doRequest(r, http.MethodGet, saleURL+"?limit=abc", "")
		require.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 20, decodeReturnList(t, w).Limit)

		w = doRequest(r, http.MethodGet, saleURL+"?limit=500", "")
		require.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, 100, decodeReturnList(t, w).Limit)

		w = doRequest(r, http.MethodGet, saleURL+"?offset=-5", "")
		require.Equal(t, http.StatusOK, w.Code)
		assert.Zero(t, decodeReturnList(t, w).Offset)
	})
}

// ==================== GET /sales/returns/pending ====================

func TestListPendingReturnsHandler(t *testing.T) {
	f := seedReturnScenario(t)
	repo := NewRepository(dbPool)

	past := time.Now().Add(-time.Minute)
	active := seedPendingReturn(t, f, time.Now().Add(5*time.Minute))
	_ = mustInsertReturn(context.Background(), t, repo, f, ReturnStatusPendingApproval, &past)
	_ = mustInsertReturn(context.Background(), t, repo, f, ReturnStatusCompleted, nil)

	r, _, _, _ := setupReturnRoutes(t, f, nil, "500000", nil)

	t.Run("only actionable pendings are listed", func(t *testing.T) {
		w := doRequest(r, http.MethodGet, "/sales/returns/pending", "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		resp := decodeReturnList(t, w)
		assert.Equal(t, 1, resp.Total, "elapsed and decided returns must be hidden")
		require.Len(t, resp.Data, 1)
		assert.Equal(t, active.ID, resp.Data[0].ID)
		assert.Equal(t, ReturnStatusPendingApproval, resp.Data[0].Status)
		assert.Equal(t, 20, resp.Limit, "default page size")
	})

	t.Run("offset beyond the cap is rejected", func(t *testing.T) {
		w := doRequest(r, http.MethodGet, "/sales/returns/pending?offset=10001", "")
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, errorBody(t, w), "offset must not exceed")
	})
}

// ==================== GET /sales/returns/:returnId ====================

func TestGetReturnHandler(t *testing.T) {
	f := seedReturnScenario(t)
	pending := seedPendingReturn(t, f, time.Now().Add(5*time.Minute))
	r, _, _, _ := setupReturnRoutes(t, f, nil, "500000", nil)

	t.Run("returns the detail with its approval record", func(t *testing.T) {
		w := doRequest(r, http.MethodGet, fmt.Sprintf("/sales/returns/%d", pending.ID), "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		data := decodeReturnData(t, w)
		assert.Equal(t, pending.ID, data.ID)
		assert.Equal(t, ReturnStatusPendingApproval, data.Status)
		require.Len(t, data.Items, 1, "the detail must carry the proposal lines")
		require.NotNil(t, data.Approval)
		assert.Equal(t, "pending", data.Approval.Status)
	})

	t.Run("non-numeric id is not found", func(t *testing.T) {
		w := doRequest(r, http.MethodGet, "/sales/returns/abc", "")
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, ErrReturnNotFound.Error(), errorBody(t, w))
	})

	t.Run("unknown id is not found", func(t *testing.T) {
		w := doRequest(r, http.MethodGet, "/sales/returns/99999999", "")
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, ErrReturnNotFound.Error(), errorBody(t, w))
	})

	t.Run("another store's return is not found", func(t *testing.T) {
		r2, _, _, _ := setupReturnRoutes(t, f, nil, "500000",
			&returnAuth{userID: f.userID, storeID: ptrInt(2)})
		w := doRequest(r2, http.MethodGet, fmt.Sprintf("/sales/returns/%d", pending.ID), "")
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, ErrReturnNotFound.Error(), errorBody(t, w))
	})
}

// ==================== POST /sales/returns/:returnId/approve ====================

func TestApproveReturnHandler(t *testing.T) {
	// Every subtest seeds its own pending return: approving mutates it.
	body := `{"note":"ok"}`

	t.Run("approves and completes a pending return", func(t *testing.T) {
		f := seedReturnScenario(t)
		pending := seedPendingReturn(t, f, time.Now().Add(5*time.Minute))
		r, _, bus, adjuster := setupReturnRoutes(t, f, map[int]int{}, "500000", nil)

		w := doRequest(r, http.MethodPost, fmt.Sprintf("/sales/returns/%d/approve", pending.ID), body)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())

		data := decodeReturnData(t, w)
		assert.Equal(t, pending.ID, data.ID)
		assert.Equal(t, ReturnStatusCompleted, data.Status)
		assert.Equal(t, ReturnTypeRefund, data.ReturnType)
		assert.Equal(t, 5000, data.TotalRefundAmount, "re-planned from the sale, not the 10000 proposal")
		require.Len(t, data.Items, 1)
		assert.Equal(t, 5000, data.Items[0].Subtotal)
		// data.Approval is deliberately NOT asserted: the response carries
		// the pre-flip snapshot (still "pending") — a known production
		// quirk. The database is the witness of the recorded decision.
		require.Equal(t, 1, adjuster.calls)
		require.Len(t, bus.topics, 1)
		assert.Equal(t, events.TopicSaleReturned, bus.topics[0])

		status, _ := approvalRowStatus(t, pending.ID)
		assert.Equal(t, "approved", status)
		assert.Equal(t, ReturnStatusCompleted, returnRowStatus(t, pending.ID))
	})

	t.Run("unparsable note body is tolerated", func(t *testing.T) {
		f := seedReturnScenario(t)
		pending := seedPendingReturn(t, f, time.Now().Add(5*time.Minute))
		r, _, _, _ := setupReturnRoutes(t, f, map[int]int{}, "500000", nil)

		// Production quirk: ApproveReturn zeroes the body on a bind error
		// instead of failing, so a garbage payload still approves without a
		// note. Asserted here to keep the quirk visible if it ever changes.
		w := doRequest(r, http.MethodPost, fmt.Sprintf("/sales/returns/%d/approve", pending.ID), "not-json")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.Equal(t, ReturnStatusCompleted, decodeReturnData(t, w).Status)

		status, _ := approvalRowStatus(t, pending.ID)
		assert.Equal(t, "approved", status)
	})

	t.Run("a second approval is a conflict", func(t *testing.T) {
		f := seedReturnScenario(t)
		pending := seedPendingReturn(t, f, time.Now().Add(5*time.Minute))
		r, _, _, _ := setupReturnRoutes(t, f, map[int]int{}, "500000", nil)

		w := doRequest(r, http.MethodPost, fmt.Sprintf("/sales/returns/%d/approve", pending.ID), body)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())

		w = doRequest(r, http.MethodPost, fmt.Sprintf("/sales/returns/%d/approve", pending.ID), body)
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Equal(t, ErrReturnNotPending.Error(), errorBody(t, w))
	})

	t.Run("elapsed window flips to expired and conflicts", func(t *testing.T) {
		f := seedReturnScenario(t)
		pending := seedPendingReturn(t, f, time.Now().Add(-time.Minute))
		r, _, bus, adjuster := setupReturnRoutes(t, f, map[int]int{}, "500000", nil)

		w := doRequest(r, http.MethodPost, fmt.Sprintf("/sales/returns/%d/approve", pending.ID), body)
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Equal(t, ErrApprovalExpired.Error(), errorBody(t, w))

		// The flip commits while the caller still sees the conflict: the
		// sale's returnable quantity is released.
		assert.Equal(t, ReturnStatusExpired, returnRowStatus(t, pending.ID))
		status, _ := approvalRowStatus(t, pending.ID)
		assert.Equal(t, "expired", status)
		assert.Zero(t, adjuster.calls, "no money moves on an expired approval")
		assert.Empty(t, bus.topics)
	})

	t.Run("non-numeric id is not found", func(t *testing.T) {
		f := seedReturnScenario(t)
		r, _, _, _ := setupReturnRoutes(t, f, map[int]int{}, "500000", nil)
		w := doRequest(r, http.MethodPost, "/sales/returns/abc/approve", body)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, ErrReturnNotFound.Error(), errorBody(t, w))
	})

	t.Run("another store's return is not found", func(t *testing.T) {
		f := seedReturnScenario(t)
		pending := seedPendingReturn(t, f, time.Now().Add(5*time.Minute))
		r, _, _, _ := setupReturnRoutes(t, f, map[int]int{}, "500000",
			&returnAuth{userID: f.userID, storeID: ptrInt(2)})

		w := doRequest(r, http.MethodPost, fmt.Sprintf("/sales/returns/%d/approve", pending.ID), body)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, ErrReturnNotFound.Error(), errorBody(t, w))

		status, _ := approvalRowStatus(t, pending.ID)
		assert.Equal(t, "pending", status, "a scoped-out approval must stay untouched")
	})
}

// ==================== POST /sales/returns/:returnId/reject ====================

func TestRejectReturnHandler(t *testing.T) {
	t.Run("rejects and cancels a pending return", func(t *testing.T) {
		f := seedReturnScenario(t)
		pending := seedPendingReturn(t, f, time.Now().Add(5*time.Minute))
		r, _, bus, adjuster := setupReturnRoutes(t, f, nil, "500000", nil)

		w := doRequest(r, http.MethodPost, fmt.Sprintf("/sales/returns/%d/reject", pending.ID),
			`{"reason":"di luar kebijakan"}`)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.Equal(t, ReturnStatusCancelled, decodeReturnData(t, w).Status)

		status, reason := approvalRowStatus(t, pending.ID)
		assert.Equal(t, "rejected", status)
		assert.Equal(t, "di luar kebijakan", reason)
		assert.Equal(t, ReturnStatusCancelled, returnRowStatus(t, pending.ID))
		assert.Empty(t, bus.topics, "a rejection moves no goods or money")
		assert.Zero(t, adjuster.calls)
	})

	t.Run("malformed json is a validation error", func(t *testing.T) {
		f := seedReturnScenario(t)
		pending := seedPendingReturn(t, f, time.Now().Add(5*time.Minute))
		r, _, _, _ := setupReturnRoutes(t, f, nil, "500000", nil)

		w := doRequest(r, http.MethodPost, fmt.Sprintf("/sales/returns/%d/reject", pending.ID), `{"reason":`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "invalid request body", errorBody(t, w))

		status, _ := approvalRowStatus(t, pending.ID)
		assert.Equal(t, "pending", status, "an unparseable rejection must not decide anything")
	})

	t.Run("empty reason is a validation error", func(t *testing.T) {
		f := seedReturnScenario(t)
		pending := seedPendingReturn(t, f, time.Now().Add(5*time.Minute))
		r, _, _, _ := setupReturnRoutes(t, f, nil, "500000", nil)

		w := doRequest(r, http.MethodPost, fmt.Sprintf("/sales/returns/%d/reject", pending.ID), `{"reason":""}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "invalid return request: rejection reason is required", errorBody(t, w))
	})

	t.Run("non-numeric id is not found", func(t *testing.T) {
		f := seedReturnScenario(t)
		r, _, _, _ := setupReturnRoutes(t, f, nil, "500000", nil)
		w := doRequest(r, http.MethodPost, "/sales/returns/abc/reject", `{"reason":"x"}`)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, ErrReturnNotFound.Error(), errorBody(t, w))
	})

	t.Run("unknown id is not found", func(t *testing.T) {
		f := seedReturnScenario(t)
		r, _, _, _ := setupReturnRoutes(t, f, nil, "500000", nil)
		w := doRequest(r, http.MethodPost, "/sales/returns/99999999/reject", `{"reason":"x"}`)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, ErrReturnNotFound.Error(), errorBody(t, w))
	})

	t.Run("a completed return cannot be rejected", func(t *testing.T) {
		f := seedReturnScenario(t)
		repo := NewRepository(dbPool)
		done := insertReturnWithItem(context.Background(), t, repo, f, ReturnStatusCompleted, nil)
		r, _, _, _ := setupReturnRoutes(t, f, nil, "500000", nil)

		w := doRequest(r, http.MethodPost, fmt.Sprintf("/sales/returns/%d/reject", done.ID), `{"reason":"x"}`)
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Equal(t, ErrReturnNotPending.Error(), errorBody(t, w))
		assert.Equal(t, ReturnStatusCompleted, returnRowStatus(t, done.ID), "the completed row must stay untouched")
	})

	t.Run("another store's return is not found", func(t *testing.T) {
		f := seedReturnScenario(t)
		pending := seedPendingReturn(t, f, time.Now().Add(5*time.Minute))
		r, _, _, _ := setupReturnRoutes(t, f, nil, "500000",
			&returnAuth{userID: f.userID, storeID: ptrInt(2)})

		w := doRequest(r, http.MethodPost, fmt.Sprintf("/sales/returns/%d/reject", pending.ID), `{"reason":"x"}`)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, ErrReturnNotFound.Error(), errorBody(t, w))

		assert.Equal(t, ReturnStatusPendingApproval, returnRowStatus(t, pending.ID))
		status, _ := approvalRowStatus(t, pending.ID)
		assert.Equal(t, "pending", status, "a scoped-out rejection must stay untouched")
	})
}
