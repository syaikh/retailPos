package stockopname

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// insertLocationRackFixture sets up a warehouse, an active storage location,
// and a product that has both a global stock row and a rack row (10 units
// each), mirroring the state a real rack-scoped cycle count starts from.
func insertLocationRackFixture(ctx context.Context, t *testing.T, warehouseID int, prefix string) (locID, productID int) {
	t.Helper()
	_, err := dbPool.Exec(ctx,
		`INSERT INTO warehouses (id, name, code, store_id, is_active) VALUES ($1, $2, $3, NULL, true)
		 ON CONFLICT (id) DO NOTHING`,
		warehouseID, "Test WH "+prefix, "SO-HDL-WH-"+prefix,
	)
	require.NoError(t, err)
	locID = insertTestStorageLocation(ctx, t, prefix+"-LOC", warehouseID)
	productID = insertTestProduct(ctx, t, prefix+"-001")
	insertTestStock(ctx, t, productID, 10)
	insertTestStockLocation(ctx, t, productID, warehouseID, locID, 10)
	_, err = dbPool.Exec(ctx, `UPDATE products SET stock = 10 WHERE id = $1`, productID)
	require.NoError(t, err)
	return locID, productID
}

func TestHandler_CreateLocationSession(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	resetStockOpname(ctx, t)
	insertTestUserWithRole(ctx, t, 9751, "so_hdl_loc_9751", 3)
	locID, _ := insertLocationRackFixture(ctx, t, 9751, "LOC01")

	r := setupStockOpnameRouter()
	w := postJSON(t, r, "/stock-opnames", fmt.Sprintf(`{"scope_type":"location","scope_id":%d}`, locID))
	assert.Equal(t, http.StatusCreated, w.Code)

	var resp struct {
		Data Session `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "location", resp.Data.ScopeType)
	assert.Equal(t, int64(locID), resp.Data.ScopeID)
	require.NotNil(t, resp.Data.LocationID)
	assert.Equal(t, locID, *resp.Data.LocationID)
	require.NotNil(t, resp.Data.WarehouseID)
	assert.Equal(t, 9751, *resp.Data.WarehouseID)
	assert.Nil(t, resp.Data.StoreID, "rack-scoped session must not carry a store")
}

func TestHandler_ListAndGetLocationSession(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	resetStockOpname(ctx, t)
	insertTestUserWithRole(ctx, t, 9752, "so_hdl_loc_9752", 3)
	locID, _ := insertLocationRackFixture(ctx, t, 9752, "LOC02")

	r := setupStockOpnameRouter()
	sessionID := createHandlerSession(ctx, t, r, "location", int64(locID))

	t.Run("list includes the rack-scoped session", func(t *testing.T) {
		w := getPath(t, r, "/stock-opnames")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "SO-")
	})

	t.Run("get session persists location and warehouse", func(t *testing.T) {
		w := getPath(t, r, fmt.Sprintf("/stock-opnames/%d", sessionID))
		assert.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Data Session `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, sessionID, resp.Data.ID)
		assert.Equal(t, "location", resp.Data.ScopeType)
		require.NotNil(t, resp.Data.LocationID)
		assert.Equal(t, locID, *resp.Data.LocationID)
		require.NotNil(t, resp.Data.WarehouseID)
		assert.Equal(t, 9752, *resp.Data.WarehouseID)
	})
}

func TestHandler_CreateLocationSession_Errors(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	resetStockOpname(ctx, t)
	insertTestUserWithRole(ctx, t, 9753, "so_hdl_loc_9753", 3)
	locID, _ := insertLocationRackFixture(ctx, t, 9753, "LOC03")

	t.Run("location scope must be used alone", func(t *testing.T) {
		r := setupStockOpnameRouter()
		body := fmt.Sprintf(`{"scopes":[{"scope_type":"location","scope_id":%d},{"scope_type":"store","scope_id":1}]}`, locID)
		w := postJSON(t, r, "/stock-opnames", body)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "SO-401")
	})

	t.Run("inactive location is rejected", func(t *testing.T) {
		_, err := dbPool.Exec(ctx,
			`INSERT INTO warehouses (id, name, code, store_id, is_active) VALUES (9754, 'Test WH SO-HDL-LOC-INA', 'SO-HDL-WH-9754', NULL, true) ON CONFLICT (id) DO NOTHING`,
		)
		require.NoError(t, err)
		var inaLocID int
		err = dbPool.QueryRow(ctx,
			`INSERT INTO storage_locations (code, name, warehouse_id, is_active)
			 VALUES ('SO-HDL-LOC-INA', 'Rack Inactive 9754', 9754, false) RETURNING id`,
		).Scan(&inaLocID)
		require.NoError(t, err)
		p := insertTestProduct(ctx, t, "SO-HDL-LOC-INA-001")
		insertTestStockLocation(ctx, t, p, 9754, inaLocID, 4)

		r := setupStockOpnameRouter()
		w := postJSON(t, r, "/stock-opnames", fmt.Sprintf(`{"scope_type":"location","scope_id":%d}`, inaLocID))
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "SO-411")
	})

	t.Run("unknown location yields 404", func(t *testing.T) {
		// Location validation runs before the product-universe resolution, so
		// a missing location must surface as ErrLocationNotFound (404), not be
		// masked by an empty candidate universe (ErrNoItems, 500).
		r := setupStockOpnameRouter()
		w := postJSON(t, r, "/stock-opnames", `{"scope_type":"location","scope_id":999999}`)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "SO-410")
	})
}

func TestHandler_LocationFullFlow_ReconcilesRackAndGlobal(t *testing.T) {
	skipIfNoDB(t)
	ctx := context.Background()
	resetStockOpname(ctx, t)

	managerID := 9755
	counterID := 9756
	insertTestUserWithRole(ctx, t, managerID, "so_hdl_loc_mgr_9755", 3)
	insertTestUserWithRole(ctx, t, counterID, "so_hdl_loc_cnt_9756", 5)

	locID, p := insertLocationRackFixture(ctx, t, 9755, "LOC05")

	managerRouter := setupStockOpnameRouterAs(managerID, "manager")
	counterRouter := setupStockOpnameRouterAs(counterID, "cashier")
	sessionID := createHandlerSession(ctx, t, managerRouter, "location", int64(locID))

	w := getPath(t, managerRouter, fmt.Sprintf("/stock-opnames/%d", sessionID))
	require.Equal(t, http.StatusOK, w.Code)
	var sessionResp struct {
		Data Session `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &sessionResp))
	require.NotEmpty(t, sessionResp.Data.Items)
	require.Equal(t, 1, len(sessionResp.Data.Items))
	itemID := sessionResp.Data.Items[0].ID

	t.Run("assign counter", func(t *testing.T) {
		body := fmt.Sprintf(`{"user_id":%d,"role":"counter"}`, counterID)
		w := postJSON(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/assignments", sessionID), body)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("counter counts 13 against expected 10", func(t *testing.T) {
		w := postJSON(t, counterRouter, fmt.Sprintf("/stock-opnames/%d/start", sessionID), "")
		assert.Equal(t, http.StatusOK, w.Code)
		w = putJSON(t, counterRouter, fmt.Sprintf("/stock-opnames/items/%d/count", itemID), `{"physical_qty":13,"remarks":"rack reconcile"}`)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("submit and verify", func(t *testing.T) {
		w := postJSON(t, counterRouter, fmt.Sprintf("/stock-opnames/%d/submit", sessionID), "")
		assert.Equal(t, http.StatusOK, w.Code)
		w = postJSON(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/verify", sessionID), `{"comment":"ok"}`)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("verified but not posted leaves stock untouched", func(t *testing.T) {
		assertStockQty(ctx, t, p, locID, 10)
		assertGlobalQty(ctx, t, p, 10)
		assertProductsStock(ctx, t, p, 10)
	})

	var adjustmentID int
	t.Run("post adjustment writes ledger and reconciles rack + global", func(t *testing.T) {
		w := postJSON(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/post-adjustment", sessionID), `{"comment":"ok"}`)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "IA-")
		var resp struct {
			Data Adjustment `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.NotEmpty(t, resp.Data.AdjustmentNumber)
		adjustmentID = resp.Data.ID
		require.NotZero(t, adjustmentID)

		assertStockQty(ctx, t, p, locID, 13)
		assertGlobalQty(ctx, t, p, 13)
		assertProductsStock(ctx, t, p, 13)

		var adjCount int
		require.NoError(t, dbPool.QueryRow(ctx,
			`SELECT COUNT(*) FROM inventory_adjustments WHERE session_id = $1`, sessionID).Scan(&adjCount))
		assert.Equal(t, 1, adjCount)

		var adjItemCount int
		require.NoError(t, dbPool.QueryRow(ctx,
			`SELECT COUNT(*) FROM inventory_adjustment_items WHERE adjustment_id = $1`, adjustmentID).Scan(&adjItemCount))
		assert.Equal(t, 1, adjItemCount)

		var mvCount int
		require.NoError(t, dbPool.QueryRow(ctx,
			`SELECT COUNT(*) FROM inventory_movements WHERE product_id = $1 AND type = 'stock_opname'`, p).Scan(&mvCount))
		assert.Equal(t, 1, mvCount)
	})

	t.Run("close posted session", func(t *testing.T) {
		w := postJSON(t, managerRouter, fmt.Sprintf("/stock-opnames/%d/close", sessionID), "")
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
