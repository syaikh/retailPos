package stockopname

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/inventory"
	"retail-pos-system/internal/product"
)

// insertTestStockLocation inserts (or updates) a rack-scoped product_stock row.
func insertTestStockLocation(ctx context.Context, t *testing.T, productID, warehouseID, locationID, quantity int) {
	t.Helper()
	_, err := dbPool.Exec(ctx,
		`INSERT INTO product_stock (product_id, warehouse_id, store_id, location_id, quantity)
		 VALUES ($1, $2, NULL, $3, $4)
		 ON CONFLICT ON CONSTRAINT uq_product_stock DO UPDATE SET quantity = EXCLUDED.quantity`,
		productID, warehouseID, locationID, quantity,
	)
	require.NoError(t, err)
}

func insertTestStorageLocation(ctx context.Context, t *testing.T, code string, warehouseID int) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(ctx,
		`INSERT INTO storage_locations (code, name, warehouse_id, is_active)
		 VALUES ($1, $2, $3, true) RETURNING id`,
		code, "Test Rack "+code, warehouseID).Scan(&id)
	require.NoError(t, err)
	return id
}

func TestRepository_GetLocationScope_Errors(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()
	resetStockOpname(ctx, t)

	t.Run("missing location", func(t *testing.T) {
		_, _, err := repo.GetLocationScope(ctx, dbPool, 999999)
		require.ErrorIs(t, err, ErrLocationNotFound)
	})

	t.Run("inactive location", func(t *testing.T) {
		_, err := dbPool.Exec(ctx,
			`INSERT INTO warehouses (id, name, code, store_id, is_active) VALUES (9811, 'Test WH 9811', 'SO-WH-9811', NULL, true) ON CONFLICT (id) DO NOTHING`,
		)
		require.NoError(t, err)
		var locID int
		err = dbPool.QueryRow(ctx,
			`INSERT INTO storage_locations (code, name, warehouse_id, is_active)
			 VALUES ('SO-LOC-9811', 'Rack Inactive 9811', 9811, false) RETURNING id`,
		).Scan(&locID)
		require.NoError(t, err)

		_, _, err = repo.GetLocationScope(ctx, dbPool, locID)
		require.ErrorIs(t, err, ErrLocationInactive)
	})
}

func TestService_LocationScope_InactiveLocationWithProduct(t *testing.T) {
	repo := NewRepository(dbPool)
	svc := NewService(repo, nil)
	svc.SetStockApplier(inventory.StockApplier{StockSyncer: product.StockSyncer{}})
	ctx := context.Background()
	resetStockOpname(ctx, t)
	insertTestUserWithRole(ctx, t, 9813, "so_loc_bad_9813", 3)

	_, err := dbPool.Exec(ctx,
		`INSERT INTO warehouses (id, name, code, store_id, is_active) VALUES (9813, 'Test WH 9813', 'SO-WH-9813', NULL, true) ON CONFLICT (id) DO NOTHING`,
	)
	require.NoError(t, err)
	var locID int
	err = dbPool.QueryRow(ctx,
		`INSERT INTO storage_locations (code, name, warehouse_id, is_active)
		 VALUES ('SO-LOC-9813', 'Rack Inactive 9813', 9813, false) RETURNING id`,
	).Scan(&locID)
	require.NoError(t, err)

	// The rack must carry a product so the candidate universe is non-empty and
	// the flow reaches the location validation (GetLocationScope).
	p := insertTestProduct(ctx, t, "SO-LOC-INACTIVE-001")
	insertTestStockLocation(ctx, t, p, 9813, locID, 4)

	_, err = svc.CreateSession(ctx, &CreateSessionRequest{
		ScopeType: "location", ScopeID: int64(locID),
	}, 9813)
	require.ErrorIs(t, err, ErrLocationInactive)
}

func TestService_LocationScope_UnknownLocation(t *testing.T) {
	// Regression for review finding #4: an unknown location used to be masked
	// by ErrNoItems (500) when the rack carried no products, because location
	// validation ran after the candidate-universe resolution. It must surface
	// as ErrLocationNotFound (404) regardless of rack contents.
	repo := NewRepository(dbPool)
	svc := NewService(repo, nil)
	svc.SetStockApplier(inventory.StockApplier{StockSyncer: product.StockSyncer{}})
	ctx := context.Background()
	resetStockOpname(ctx, t)
	insertTestUserWithRole(ctx, t, 9819, "so_loc_missing_9819", 3)

	_, err := svc.CreateSession(ctx, &CreateSessionRequest{
		ScopeType: "location", ScopeID: 999999,
	}, 9819)
	require.ErrorIs(t, err, ErrLocationNotFound)
}

func TestService_CreateSession_LocationFailureDoesNotBurnSequence(t *testing.T) {
	// Regression for review finding #3: a failed CreateSession (inactive
	// location) must not advance so_seq, so session numbers stay contiguous.
	repo := NewRepository(dbPool)
	svc := NewService(repo, nil)
	svc.SetStockApplier(inventory.StockApplier{StockSyncer: product.StockSyncer{}})
	ctx := context.Background()
	resetStockOpname(ctx, t)
	insertTestUserWithRole(ctx, t, 9818, "so_loc_seq_9818", 3)

	_, err := dbPool.Exec(ctx,
		`INSERT INTO warehouses (id, name, code, store_id, is_active) VALUES (9818, 'Test WH 9818', 'SO-WH-9818', NULL, true) ON CONFLICT (id) DO NOTHING`,
	)
	require.NoError(t, err)
	var locID int
	err = dbPool.QueryRow(ctx,
		`INSERT INTO storage_locations (code, name, warehouse_id, is_active)
		 VALUES ('SO-LOC-9818', 'Rack Inactive 9818', 9818, false) RETURNING id`,
	).Scan(&locID)
	require.NoError(t, err)

	p := insertTestProduct(ctx, t, "SO-LOC-SEQ-001")
	insertTestStockLocation(ctx, t, p, 9818, locID, 4)

	var before int
	require.NoError(t, dbPool.QueryRow(ctx, `SELECT last_value FROM so_seq`).Scan(&before))

	_, err = svc.CreateSession(ctx, &CreateSessionRequest{
		ScopeType: "location", ScopeID: int64(locID),
	}, 9818)
	require.ErrorIs(t, err, ErrLocationInactive)

	var after int
	require.NoError(t, dbPool.QueryRow(ctx, `SELECT last_value FROM so_seq`).Scan(&after))
	assert.Equal(t, before, after, "failed CreateSession must not consume so_seq")
}

func TestService_LocationScope_RequiresSoleScope(t *testing.T) {
	repo := NewRepository(dbPool)
	svc := NewService(repo, nil)
	svc.SetStockApplier(inventory.StockApplier{StockSyncer: product.StockSyncer{}})
	ctx := context.Background()
	resetStockOpname(ctx, t)
	insertTestUserWithRole(ctx, t, 9801, "so_loc_manager_9801", 3)

	_, err := dbPool.Exec(ctx,
		`INSERT INTO warehouses (id, name, code, store_id, is_active) VALUES (9801, 'Test WH 9801', 'SO-WH-9801', NULL, true) ON CONFLICT (id) DO NOTHING`,
	)
	require.NoError(t, err)
	locID := insertTestStorageLocation(ctx, t, "SO-LOC-9801", 9801)

	// location must be the only scope
	_, err = svc.CreateSession(ctx, &CreateSessionRequest{
		Scopes: []Scope{
			{ScopeType: "location", ScopeID: int64(locID)},
			{ScopeType: "store", ScopeID: 1},
		},
	}, 9801)
	require.ErrorIs(t, err, ErrLocationScopeSingle)
}

func TestRepository_LocationScope_ProductUniverse(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()
	resetStockOpname(ctx, t)

	_, err := dbPool.Exec(ctx,
		`INSERT INTO warehouses (id, name, code, store_id, is_active) VALUES (9802, 'Test WH 9802', 'SO-WH-9802', NULL, true) ON CONFLICT (id) DO NOTHING`,
	)
	require.NoError(t, err)
	locID := insertTestStorageLocation(ctx, t, "SO-LOC-9802", 9802)

	onRack := insertTestProduct(ctx, t, "SO-LOC-SCOPE-001")
	globalOnly := insertTestProduct(ctx, t, "SO-LOC-SCOPE-002")
	insertTestStockLocation(ctx, t, onRack, 9802, locID, 4)
	insertTestStock(ctx, t, globalOnly, 7)

	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	ids, err := repo.ScopeProductIDs(ctx, tx, Scope{ScopeType: "location", ScopeID: int64(locID)})
	require.NoError(t, err)
	assert.Contains(t, ids, onRack)
	assert.NotContains(t, ids, globalOnly, "global-only product must not be in location scope")

	name, err := repo.ResolveScopeName(ctx, tx, "location", int64(locID))
	require.NoError(t, err)
	assert.Equal(t, "Test Rack SO-LOC-9802", name)
}

func TestRepository_CreateSession_PersistsLocationID(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()
	resetStockOpname(ctx, t)
	insertTestUser(ctx, t, 9803, "so_loc_rt_9803")

	_, err := dbPool.Exec(ctx,
		`INSERT INTO warehouses (id, name, code, store_id, is_active) VALUES (9803, 'Test WH 9803', 'SO-WH-9803', NULL, true) ON CONFLICT (id) DO NOTHING`,
	)
	require.NoError(t, err)
	locID := insertTestStorageLocation(ctx, t, "SO-LOC-9803", 9803)
	p := insertTestProduct(ctx, t, "SO-LOC-RT-001")
	insertTestStockLocation(ctx, t, p, 9803, locID, 3)

	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	lid := locID
	s := &Session{
		SessionNumber: "SO-LOC-RT-9803",
		ScopeType:     "location",
		ScopeID:       int64(locID),
		LocationID:    &lid,
		WarehouseID:   intPtr(9803),
		BlindCount:    false,
		Status:        StatusDraft,
		CreatedBy:     9803,
	}
	require.NoError(t, repo.CreateSession(ctx, tx, s))
	require.NoError(t, tx.Commit(ctx))

	got, err := repo.GetSession(ctx, s.ID)
	require.NoError(t, err)
	require.NotNil(t, got.LocationID)
	assert.Equal(t, locID, *got.LocationID)
	require.NotNil(t, got.WarehouseID)
	assert.Equal(t, 9803, *got.WarehouseID)

	sessions, _, err := repo.ListSessions(ctx, 10, 0, "", "")
	require.NoError(t, err)
	var listed *Session
	for i := range sessions {
		if sessions[i].ID == s.ID {
			listed = &sessions[i]
			break
		}
	}
	require.NotNil(t, listed)
	require.NotNil(t, listed.LocationID)
	assert.Equal(t, locID, *listed.LocationID)
}

func TestService_LocationScope_FullFlow_ReconcilesRackAndGlobal(t *testing.T) {
	repo := NewRepository(dbPool)
	svc := NewService(repo, nil)
	svc.SetStockApplier(inventory.StockApplier{StockSyncer: product.StockSyncer{}})
	ctx := context.Background()
	resetStockOpname(ctx, t)

	managerID := 9804
	counterID := 9805
	insertTestUserWithRole(ctx, t, managerID, "so_loc_manager_9804", 3)
	insertTestUserWithRole(ctx, t, counterID, "so_loc_counter_9805", 5)

	_, err := dbPool.Exec(ctx,
		`INSERT INTO warehouses (id, name, code, store_id, is_active) VALUES (9804, 'Test WH 9804', 'SO-WH-9804', NULL, true) ON CONFLICT (id) DO NOTHING`,
	)
	require.NoError(t, err)
	locID := insertTestStorageLocation(ctx, t, "SO-LOC-9804", 9804)

	p := insertTestProduct(ctx, t, "SO-LOC-FLOW-001")
	insertTestStock(ctx, t, p, 10)
	insertTestStockLocation(ctx, t, p, 9804, locID, 10)
	_, err = dbPool.Exec(ctx, `UPDATE products SET stock = 10 WHERE id = $1`, p)
	require.NoError(t, err)

	session, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "location", ScopeID: int64(locID)}, managerID)
	require.NoError(t, err)
	require.NotNil(t, session.LocationID)
	assert.Equal(t, locID, *session.LocationID)
	require.NotNil(t, session.WarehouseID)
	assert.Equal(t, 9804, *session.WarehouseID)
	assert.Nil(t, session.StoreID)

	require.NoError(t, svc.AssignCounter(ctx, session.ID, counterID, AssignmentRoleCounter))
	require.NoError(t, svc.StartCounting(ctx, session.ID, counterID))

	// physical count 13 vs expected 10 -> +3
	countAllItems(ctx, t, svc, session.ID, counterID, p, 13)
	require.NoError(t, svc.SubmitSession(ctx, session.ID, counterID))
	require.NoError(t, svc.VerifySession(ctx, session.ID, managerID, "ok"))

	// verified but not yet posted: rack and global unchanged
	assertStockQty(ctx, t, p, locID, 10)
	assertGlobalQty(ctx, t, p, 10)
	assertProductsStock(ctx, t, p, 10)

	adj, err := svc.PostAdjustment(ctx, session.ID, managerID, &PostAdjustmentRequest{Comment: "ok", Notes: "location reconcile"})
	require.NoError(t, err)
	require.NotEmpty(t, adj.AdjustmentNumber)
	assert.Contains(t, adj.AdjustmentNumber, "IA-")

	status, err := repo.GetSessionStatus(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusPosted, status)

	// both rack row and global row advanced by +3; products.stock synced
	// (Option A: global recomputed as max(global-rack, 0) + newRack)
	assertStockQty(ctx, t, p, locID, 13)
	assertGlobalQty(ctx, t, p, 13)
	assertProductsStock(ctx, t, p, 13)

	// ledger document written (one line item per product; the rack delta is
	// applied to the rack row and the global row is reconciled to it)
	var adjCount int
	require.NoError(t, dbPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM inventory_adjustments WHERE session_id = $1`, session.ID).Scan(&adjCount))
	assert.Equal(t, 1, adjCount)

	var adjItemCount int
	require.NoError(t, dbPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM inventory_adjustment_items WHERE adjustment_id = $1`, adj.ID).Scan(&adjItemCount))
	assert.Equal(t, 1, adjItemCount)

	// stock opname movement recorded for the product
	var mvCount int
	require.NoError(t, dbPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM inventory_movements WHERE product_id = $1 AND type = 'stock_opname'`, p).Scan(&mvCount))
	assert.Equal(t, 1, mvCount)

	// close the posted session to leave a clean active-session state
	require.NoError(t, svc.CloseSession(ctx, session.ID, managerID))
}

func assertStockQty(ctx context.Context, t *testing.T, productID, locationID, want int) {
	t.Helper()
	var qty int
	err := dbPool.QueryRow(ctx,
		`SELECT quantity FROM product_stock WHERE product_id = $1 AND location_id = $2`,
		productID, locationID).Scan(&qty)
	require.NoError(t, err)
	assert.Equal(t, want, qty)
}

func assertGlobalQty(ctx context.Context, t *testing.T, productID, want int) {
	t.Helper()
	var qty int
	err := dbPool.QueryRow(ctx,
		`SELECT quantity FROM product_stock WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL AND location_id IS NULL`,
		productID).Scan(&qty)
	require.NoError(t, err)
	assert.Equal(t, want, qty)
}

func assertProductsStock(ctx context.Context, t *testing.T, productID, want int) {
	t.Helper()
	var qty int
	err := dbPool.QueryRow(ctx, `SELECT stock FROM products WHERE id = $1`, productID).Scan(&qty)
	require.NoError(t, err)
	assert.Equal(t, want, qty)
}

func intPtr(v int) *int { return &v }
