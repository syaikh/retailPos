package stockopname

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runHandlerToVerification drives a store-scoped session through
// create -> assign -> start -> count all items -> submit, ending in
// 'verification'. The counter counts the opening quantity for every item.
func runHandlerToVerification(ctx context.Context, t *testing.T, managerRouter, counterRouter *gin.Engine, counterID, storeID int) int {
	t.Helper()
	sessionID := createHandlerSession(ctx, t, managerRouter, "store", int64(storeID))

	w := postJSON(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/assignments", sessionID),
		fmt.Sprintf(`{"user_id":%d,"role":"counter"}`, counterID))
	require.Equal(t, http.StatusCreated, w.Code)

	w = postJSON(t, counterRouter, fmt.Sprintf("/stock-opnames/%d/start", sessionID), "")
	require.Equal(t, http.StatusOK, w.Code)

	w = getPath(t, counterRouter, fmt.Sprintf("/stock-opnames/%d", sessionID))
	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data Session `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Data.Items)
	for _, it := range resp.Data.Items {
		w = putJSON(t, counterRouter, fmt.Sprintf("/stock-opnames/items/%d/count", it.ID),
			fmt.Sprintf(`{"physical_qty":%v,"remarks":"ok"}`, it.OpeningQty))
		require.Equal(t, http.StatusOK, w.Code)
	}

	w = postJSON(t, counterRouter, fmt.Sprintf("/stock-opnames/%d/submit", sessionID), "")
	require.Equal(t, http.StatusOK, w.Code)
	return sessionID
}

func TestHandler_OpenSession(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	resetStockOpname(ctx, t)
	managerID := 9771
	insertTestUserWithRole(ctx, t, managerID, "so_hdl_open_9771", 3)
	insertTestStore(ctx, t, 9771)
	p := insertTestProductStore(ctx, t, "SO-HDL-OPEN-001", 9771)
	insertTestStock(ctx, t, p, 10)

	counterID := 9772
	insertTestUserWithRole(ctx, t, counterID, "so_hdl_open_9772", 5)
	managerRouter := setupStockOpnameRouterAs(managerID, "manager")
	counterRouter := setupStockOpnameRouterAs(counterID, "cashier")
	sessionID := createHandlerSession(ctx, t, managerRouter, "store", 9771)
	w := postJSON(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/assignments", sessionID),
		fmt.Sprintf(`{"user_id":%d,"role":"counter"}`, counterID))
	require.Equal(t, http.StatusCreated, w.Code)

	t.Run("open without comment returns 422", func(t *testing.T) {
		w := postJSON(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/open", sessionID), `{"comment":""}`)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), "SO-402")
	})

	t.Run("open with comment moves draft to open", func(t *testing.T) {
		w := postJSON(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/open", sessionID), `{"comment":"ready"}`)
		assert.Equal(t, http.StatusOK, w.Code)

		w = getPath(t, managerRouter, fmt.Sprintf("/stock-opnames/%d", sessionID))
		require.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data Session `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, StatusOpen, resp.Data.Status)
		require.NotNil(t, resp.Data.OpenedBy)
		assert.Equal(t, managerID, *resp.Data.OpenedBy)
	})

	t.Run("counter starts counting an open session", func(t *testing.T) {
		w := postJSON(t, counterRouter, fmt.Sprintf("/stock-opnames/%d/start", sessionID), "")
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHandler_CancelSession(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	resetStockOpname(ctx, t)
	managerID := 9773
	insertTestUserWithRole(ctx, t, managerID, "so_hdl_cancel_9773", 3)
	insertTestStore(ctx, t, 9773)
	p := insertTestProductStore(ctx, t, "SO-HDL-CANCEL-001", 9773)
	insertTestStock(ctx, t, p, 4)

	managerRouter := setupStockOpnameRouterAs(managerID, "manager")
	sessionID := createHandlerSession(ctx, t, managerRouter, "store", 9773)

	t.Run("cancel draft session", func(t *testing.T) {
		w := postJSON(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/cancel", sessionID), "")
		assert.Equal(t, http.StatusOK, w.Code)

		w = getPath(t, managerRouter, fmt.Sprintf("/stock-opnames/%d", sessionID))
		require.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data Session `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, StatusCancelled, resp.Data.Status)
	})

	t.Run("cancelled session cannot be re-cancelled", func(t *testing.T) {
		w := postJSON(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/cancel", sessionID), "")
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "SO-003")
	})

	t.Run("cancel missing session returns 404", func(t *testing.T) {
		w := postJSON(t, managerRouter, "/stock-opnames/999999/cancel", "")
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "SO-002")
	})

	t.Run("cancel with invalid id returns 400", func(t *testing.T) {
		w := postJSON(t, managerRouter, "/stock-opnames/abc/cancel", "")
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_CancelInVerification(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	resetStockOpname(ctx, t)
	managerID := 9776
	counterID := 9777
	insertTestUserWithRole(ctx, t, managerID, "so_hdl_cancelflow_9776", 3)
	insertTestUserWithRole(ctx, t, counterID, "so_hdl_cancelflow_9777", 5)
	insertTestStore(ctx, t, 9776)
	p := insertTestProductStore(ctx, t, "SO-HDL-CANCELFLOW-001", 9776)
	insertTestStock(ctx, t, p, 5)

	managerRouter := setupStockOpnameRouterAs(managerID, "manager")
	counterRouter := setupStockOpnameRouterAs(counterID, "cashier")
	sessionID := runHandlerToVerification(ctx, t, managerRouter, counterRouter, counterID, 9776)

	t.Run("session in verification is not cancellable", func(t *testing.T) {
		w := postJSON(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/cancel", sessionID), "")
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "SO-003")
	})
}

func TestHandler_ReassignCounter(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	resetStockOpname(ctx, t)
	managerID := 9774
	counterID := 9775
	insertTestUserWithRole(ctx, t, managerID, "so_hdl_reassign_9774", 3)
	insertTestUserWithRole(ctx, t, counterID, "so_hdl_reassign_9775", 5)
	insertTestStore(ctx, t, 9774)
	p := insertTestProductStore(ctx, t, "SO-HDL-REASSIGN-001", 9774)
	insertTestStock(ctx, t, p, 5)

	managerRouter := setupStockOpnameRouterAs(managerID, "manager")
	sessionID := createHandlerSession(ctx, t, managerRouter, "store", 9774)

	var supervisorAssignmentID int
	t.Run("assign counter and supervisor", func(t *testing.T) {
		w := postJSON(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/assignments", sessionID),
			fmt.Sprintf(`{"user_id":%d,"role":"counter"}`, counterID))
		assert.Equal(t, http.StatusCreated, w.Code)
		w = postJSON(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/assignments", sessionID),
			fmt.Sprintf(`{"user_id":%d,"role":"supervisor"}`, managerID))
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("reassign supervisor assignment within role", func(t *testing.T) {
		w := getPath(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/assignments", sessionID))
		require.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data []Assignment `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Len(t, resp.Data, 2)
		for _, a := range resp.Data {
			if a.Role == AssignmentRoleSupervisor {
				supervisorAssignmentID = a.ID
			}
		}
		require.NotZero(t, supervisorAssignmentID)

		w = putJSON(t, managerRouter,
			fmt.Sprintf("/stock-opnames/%d/assignments/%d", sessionID, supervisorAssignmentID),
			`{"role":"supervisor"}`)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("promoting a cashier assignment to supervisor returns 400", func(t *testing.T) {
		w := getPath(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/assignments", sessionID))
		require.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data []Assignment `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		var counterAssignmentID int
		for _, a := range resp.Data {
			if a.Role == AssignmentRoleCounter {
				counterAssignmentID = a.ID
			}
		}
		require.NotZero(t, counterAssignmentID)

		w = putJSON(t, managerRouter,
			fmt.Sprintf("/stock-opnames/%d/assignments/%d", sessionID, counterAssignmentID),
			`{"role":"supervisor"}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "SO-104")
	})

	t.Run("reassign missing assignment returns 404", func(t *testing.T) {
		w := putJSON(t, managerRouter,
			fmt.Sprintf("/stock-opnames/%d/assignments/999999", sessionID), `{"role":"counter"}`)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "SO-103")
	})

	t.Run("reassign with invalid session id returns 400", func(t *testing.T) {
		w := putJSON(t, managerRouter, "/stock-opnames/abc/assignments/1", `{"role":"counter"}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_RejectAndResume(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	resetStockOpname(ctx, t)
	managerID := 9778
	counterID := 9779
	insertTestUserWithRole(ctx, t, managerID, "so_hdl_reject_9778", 3)
	insertTestUserWithRole(ctx, t, counterID, "so_hdl_reject_9779", 5)
	insertTestStore(ctx, t, 9778)
	p := insertTestProductStore(ctx, t, "SO-HDL-REJECT-001", 9778)
	insertTestStock(ctx, t, p, 6)

	managerRouter := setupStockOpnameRouterAs(managerID, "manager")
	counterRouter := setupStockOpnameRouterAs(counterID, "cashier")
	sessionID := runHandlerToVerification(ctx, t, managerRouter, counterRouter, counterID, 9778)

	t.Run("reject without comment returns 422", func(t *testing.T) {
		w := postJSON(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/reject", sessionID), `{"comment":""}`)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), "SO-402")
	})

	t.Run("reject moves verification to needs_recount", func(t *testing.T) {
		w := postJSON(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/reject", sessionID), `{"comment":"recount"}`)
		assert.Equal(t, http.StatusOK, w.Code)

		w = getPath(t, managerRouter, fmt.Sprintf("/stock-opnames/%d", sessionID))
		require.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data Session `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, StatusNeedsRecount, resp.Data.Status)
	})

	t.Run("resume returns session to counting", func(t *testing.T) {
		w := postJSON(t, counterRouter, fmt.Sprintf("/stock-opnames/%d/resume", sessionID), "")
		assert.Equal(t, http.StatusOK, w.Code)

		w = getPath(t, managerRouter, fmt.Sprintf("/stock-opnames/%d", sessionID))
		require.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data Session `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, StatusCounting, resp.Data.Status)
	})

	t.Run("resume on a counting session returns 409", func(t *testing.T) {
		w := postJSON(t, counterRouter, fmt.Sprintf("/stock-opnames/%d/resume", sessionID), "")
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "SO-003")
	})
}

func TestHandler_RequestRecount(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	resetStockOpname(ctx, t)
	managerID := 9780
	counterID := 9781
	insertTestUserWithRole(ctx, t, managerID, "so_hdl_recount_9780", 3)
	insertTestUserWithRole(ctx, t, counterID, "so_hdl_recount_9781", 5)
	insertTestStore(ctx, t, 9780)
	p := insertTestProductStore(ctx, t, "SO-HDL-RECNT-001", 9780)
	insertTestStock(ctx, t, p, 7)

	managerRouter := setupStockOpnameRouterAs(managerID, "manager")
	counterRouter := setupStockOpnameRouterAs(counterID, "cashier")
	sessionID := runHandlerToVerification(ctx, t, managerRouter, counterRouter, counterID, 9780)

	t.Run("recount without comment returns 422", func(t *testing.T) {
		w := postJSON(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/recount", sessionID), `{"comment":""}`)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), "SO-402")
	})

	t.Run("recount records a request and moves to needs_recount", func(t *testing.T) {
		w := postJSON(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/recount", sessionID), `{"comment":"recheck"}`)
		assert.Equal(t, http.StatusOK, w.Code)

		w = getPath(t, managerRouter, fmt.Sprintf("/stock-opnames/%d", sessionID))
		require.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data Session `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, StatusNeedsRecount, resp.Data.Status)

		var reqCount int
		require.NoError(t, dbPool.QueryRow(ctx,
			`SELECT COUNT(*) FROM stock_opname_recount_requests WHERE stock_opname_id = $1`, sessionID).Scan(&reqCount))
		assert.Equal(t, 1, reqCount)
	})

	t.Run("resume after recount request", func(t *testing.T) {
		w := postJSON(t, counterRouter, fmt.Sprintf("/stock-opnames/%d/resume", sessionID), "")
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHandler_ListAndGetAdjustments(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	resetStockOpname(ctx, t)
	managerID := 9782
	counterID := 9783
	insertTestUserWithRole(ctx, t, managerID, "so_hdl_adj_9782", 3)
	insertTestUserWithRole(ctx, t, counterID, "so_hdl_adj_9783", 5)
	insertTestStore(ctx, t, 9782)
	p := insertTestProductStore(ctx, t, "SO-HDL-ADJ-001", 9782)
	insertTestStock(ctx, t, p, 8)

	managerRouter := setupStockOpnameRouterAs(managerID, "manager")
	counterRouter := setupStockOpnameRouterAs(counterID, "cashier")
	sessionID := runHandlerToVerification(ctx, t, managerRouter, counterRouter, counterID, 9782)

	w := postJSON(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/verify", sessionID), `{"comment":"ok"}`)
	require.Equal(t, http.StatusOK, w.Code)
	w = postJSON(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/post-adjustment", sessionID), `{"comment":"ok"}`)
	require.Equal(t, http.StatusOK, w.Code)
	var postResp struct {
		Data Adjustment `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &postResp))
	require.NotZero(t, postResp.Data.ID)
	adjustmentNumber := postResp.Data.AdjustmentNumber

	t.Run("list adjustments returns the posted document", func(t *testing.T) {
		w := getPath(t, managerRouter, "/stock-opnames/adjustments")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), adjustmentNumber)
	})

	t.Run("list adjustments filters by status", func(t *testing.T) {
		w := getPath(t, managerRouter, "/stock-opnames/adjustments?status=posted")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), adjustmentNumber)
	})

	t.Run("get adjustment by id", func(t *testing.T) {
		w := getPath(t, managerRouter, fmt.Sprintf("/stock-opnames/adjustments/%d", postResp.Data.ID))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), adjustmentNumber)
		assert.Contains(t, w.Body.String(), "SO-")
	})

	t.Run("get missing adjustment returns 404", func(t *testing.T) {
		w := getPath(t, managerRouter, "/stock-opnames/adjustments/999999")
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "SO-408")
	})

	t.Run("get adjustment with invalid id returns 400", func(t *testing.T) {
		w := getPath(t, managerRouter, "/stock-opnames/adjustments/not-a-number")
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
