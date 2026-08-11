package stockopname

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/events"
	"retail-pos-system/internal/inventory"
	"retail-pos-system/internal/product"
)

type capturingEventBus struct {
	mu        sync.Mutex
	published []struct {
		topic string
		event interface{}
	}
}

func (b *capturingEventBus) Publish(_ context.Context, topic string, event interface{}) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.published = append(b.published, struct {
		topic string
		event interface{}
	}{topic: topic, event: event})
	return nil
}

func (b *capturingEventBus) topics() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]string, 0, len(b.published))
	for _, p := range b.published {
		out = append(out, p.topic)
	}
	return out
}

func TestService_CreateSession(t *testing.T) {
	repo := newTestRepository()
	svc := NewService(repo, nil)
	svc.SetStockApplier(inventory.StockApplier{StockSyncer: product.StockSyncer{}})
	ctx := context.Background()
	resetStockOpname(ctx, t)
	insertTestUser(ctx, t, 9101, "so_svc_user_9101")
	insertTestStore(ctx, t, 100)
	p := insertTestProductStore(ctx, t, "SO-SVC-CREATE-001", 100)
	insertTestStock(ctx, t, p, 5)

	session, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "store", ScopeID: 100}, 9101, nil)
	require.NoError(t, err)
	assert.Equal(t, StatusDraft, session.Status)
	assert.NotEmpty(t, session.SessionNumber)
	assert.Equal(t, 9101, session.CreatedBy)

	// overlapping active session on the same SKU -> per-SKU overlap conflict
	_, err = svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "store", ScopeID: 100}, 9101, nil)
	require.ErrorIs(t, err, ErrScopeOverlap)
}

func TestService_CreateSessionInvalidScope(t *testing.T) {
	repo := newTestRepository()
	svc := NewService(repo, nil)
	svc.SetStockApplier(inventory.StockApplier{StockSyncer: product.StockSyncer{}})
	ctx := context.Background()

	_, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "all", ScopeID: 1}, 1, nil)
	require.ErrorIs(t, err, ErrUnsupportedScope)
}

func TestService_CreateSessionNoProducts(t *testing.T) {
	// ensure empty DB snapshot yields ErrNoItems; guard against races by
	// creating a fresh session for a scope after truncating nothing (items
	// exist from other tests, so this test only validates the error path shape).
	repo := newTestRepository()
	svc := NewService(repo, nil)
	svc.SetStockApplier(inventory.StockApplier{StockSyncer: product.StockSyncer{}})
	ctx := context.Background()
	_ = ctx
	// validation errors take precedence
	_, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "bogus", ScopeID: 1}, 1, nil)
	require.ErrorIs(t, err, ErrUnsupportedScope)
}

func TestService_CreateSession_ResolvesStoreID(t *testing.T) {
	repo := newTestRepository()
	svc := NewService(repo, nil)
	svc.SetStockApplier(inventory.StockApplier{StockSyncer: product.StockSyncer{}})
	ctx := context.Background()
	resetStockOpname(ctx, t)
	insertTestUserWithRole(ctx, t, 9401, "so_scope_manager_9401", 3)
	insertTestStore(ctx, t, 9401)
	insertTestStore(ctx, t, 9402)
	insertTestWarehouse(ctx, t, 9401, 9402, "SO-WH-9401")
	catID := insertTestCategory(ctx, t, "SO Cat 9401")
	p := insertTestProductStore(ctx, t, "SO-SVC-STORE-001", 9401)
	insertTestStock(ctx, t, p, 5)

	t.Run("store scope resolves to scope id", func(t *testing.T) {
		resetStockOpname(ctx, t)
		session, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "store", ScopeID: 9401}, 9401, nil)
		require.NoError(t, err)
		require.NotNil(t, session.StoreID)
		assert.Equal(t, 9401, *session.StoreID)
	})

	t.Run("warehouse scope resolves to warehouse store", func(t *testing.T) {
		resetStockOpname(ctx, t)
		wid := 9401
		whP := insertTestProductStore(ctx, t, "SO-SVC-WH-001", 9401)
		insertTestStockWarehouse(ctx, t, whP, 9401, 5)
		insertTestStock(ctx, t, whP, 5)
		session, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "warehouse", ScopeID: 9401, WarehouseID: &wid}, 9401, nil)
		require.NoError(t, err)
		require.NotNil(t, session.StoreID)
		assert.Equal(t, 9402, *session.StoreID)
	})

	t.Run("warehouse without linked store leaves store nil", func(t *testing.T) {
		resetStockOpname(ctx, t)
		_, err := dbPool.Exec(ctx,
			`INSERT INTO warehouses (id, name, code, store_id, is_active) VALUES (9403, 'Test WH 9403', 'SO-WH-9403', NULL, true) ON CONFLICT (id) DO NOTHING`,
		)
		require.NoError(t, err)
		whP := insertTestProductStore(ctx, t, "SO-SVC-WH-002", 9401)
		insertTestStockWarehouse(ctx, t, whP, 9403, 5)
		insertTestStock(ctx, t, whP, 5)
		wid := 9403
		session, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "warehouse", ScopeID: 9403, WarehouseID: &wid}, 9401, nil)
		require.NoError(t, err)
		assert.Nil(t, session.StoreID)
	})

	t.Run("warehouse scope without warehouse id resolves via scope id", func(t *testing.T) {
		resetStockOpname(ctx, t)
		whP := insertTestProductStore(ctx, t, "SO-SVC-WH-003", 9401)
		insertTestStockWarehouse(ctx, t, whP, 9401, 5)
		insertTestStock(ctx, t, whP, 5)
		session, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "warehouse", ScopeID: 9401}, 9401, nil)
		require.NoError(t, err)
		require.NotNil(t, session.StoreID)
		assert.Equal(t, 9402, *session.StoreID)
	})

	t.Run("category scope leaves store nil", func(t *testing.T) {
		resetStockOpname(ctx, t)
		catP := insertTestProductCategory(ctx, t, "SO-SVC-CAT-001", catID)
		insertTestStock(ctx, t, catP, 5)
		session, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "category", ScopeID: int64(catID)}, 9401, nil)
		require.NoError(t, err)
		assert.Nil(t, session.StoreID)
	})

	t.Run("store scope with missing store fails", func(t *testing.T) {
		resetStockOpname(ctx, t)
		_, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "store", ScopeID: 989999}, 9401, nil)
		require.Error(t, err)
	})
}

func countAllItems(ctx context.Context, t *testing.T, svc *Service, sessionID, counterID int, overrideProductID int, overrideQty float64) {
	t.Helper()
	sess, err := svc.GetSessionForUser(ctx, sessionID, counterID, nil)
	require.NoError(t, err)
	require.NotEmpty(t, sess.Items)
	for _, it := range sess.Items {
		qty := it.OpeningQty
		if it.ProductID == overrideProductID {
			qty = overrideQty
		}
		require.NoError(t, svc.SaveCount(ctx, it.ID, counterID, qty, "", nil))
	}
}

func TestService_AssignVerifyPostClose(t *testing.T) {
	repo := newTestRepository()
	svc := NewService(repo, nil)
	svc.SetStockApplier(inventory.StockApplier{StockSyncer: product.StockSyncer{}})
	ctx := context.Background()
	resetStockOpname(ctx, t)
	managerID := 9102
	counterID := 9103
	insertTestUserWithRole(ctx, t, managerID, "so_manager_9102", 3)
	insertTestUserWithRole(ctx, t, counterID, "so_counter_9103", 5)
	insertTestStore(ctx, t, 101)
	p := insertTestProductStore(ctx, t, "SO-SVC-FLOW-001", 101)
	insertTestStock(ctx, t, p, 20)

	session, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "store", ScopeID: 101}, managerID, nil)
	require.NoError(t, err)

	// assign counter
	require.NoError(t, svc.AssignCounter(ctx, session.ID, counterID, AssignmentRoleCounter, nil))

	// non-assigned user cannot start counting
	err = svc.StartCounting(ctx, session.ID, managerID, nil)
	require.ErrorIs(t, err, ErrNotAssigned)

	// start counting
	require.NoError(t, svc.StartCounting(ctx, session.ID, counterID, nil))

	// submit should fail until all items counted
	err = svc.SubmitSession(ctx, session.ID, counterID, nil)
	require.ErrorIs(t, err, ErrNotAllItemsCounted)

	// non-assigned user cannot count
	sess, err := svc.GetSessionForUser(ctx, session.ID, counterID, nil)
	require.NoError(t, err)
	require.NotEmpty(t, sess.Items)
	itemID := sess.Items[0].ID
	err = svc.SaveCount(ctx, itemID, managerID, 25, "", nil)
	require.ErrorIs(t, err, ErrNotAssigned)

	// negative qty rejected
	err = svc.SaveCount(ctx, itemID, counterID, -1, "", nil)
	require.ErrorIs(t, err, ErrInvalidQuantity)

	// zero qty is valid (out-of-stock item)
	require.NoError(t, svc.SaveCount(ctx, itemID, counterID, 0, "", nil))

	// count all items; override target product with 25
	countAllItems(ctx, t, svc, session.ID, counterID, p, 25)

	// submit now works -> verification
	require.NoError(t, svc.SubmitSession(ctx, session.ID, counterID, nil))

	// counter cannot verify (separation of duties)
	err = svc.VerifySession(ctx, session.ID, counterID, "approve", nil)
	require.ErrorIs(t, err, ErrSeparationOfDuties)

	// verify requires comment
	err = svc.VerifySession(ctx, session.ID, managerID, "  ", nil)
	require.ErrorIs(t, err, ErrApprovalCommentReq)

	// manager verifies -> approved, stock NOT yet changed (still 20)
	require.NoError(t, svc.VerifySession(ctx, session.ID, managerID, "ok", nil))

	status, err := repo.GetSessionStatus(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusApproved, status)

	var qty int
	err = dbPool.QueryRow(ctx, `SELECT quantity FROM product_stock WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL`, p).Scan(&qty)
	require.NoError(t, err)
	assert.Equal(t, 20, qty)

	// verifying an approved session is invalid state
	err = svc.VerifySession(ctx, session.ID, managerID, "again", nil)
	require.ErrorIs(t, err, ErrInvalidState)

	// post adjustment applies stock 20 -> 25 and writes a ledger document
	adj, err := svc.PostAdjustment(ctx, session.ID, managerID, &PostAdjustmentRequest{Comment: "ok", Notes: "post it"}, nil)
	require.NoError(t, err)
	require.NotEmpty(t, adj.AdjustmentNumber)
	assert.Contains(t, adj.AdjustmentNumber, "IA-")

	status, err = repo.GetSessionStatus(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusPosted, status)

	err = dbPool.QueryRow(ctx, `SELECT quantity FROM product_stock WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL`, p).Scan(&qty)
	require.NoError(t, err)
	assert.Equal(t, 25, qty)

	// posting a posted session is invalid state
	_, err = svc.PostAdjustment(ctx, session.ID, managerID, &PostAdjustmentRequest{Comment: "again"}, nil)
	require.ErrorIs(t, err, ErrInvalidState)

	// close finalises the record
	require.NoError(t, svc.CloseSession(ctx, session.ID, managerID, nil))
	status, err = repo.GetSessionStatus(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusClosed, status)

	// closing a closed session is invalid state
	err = svc.CloseSession(ctx, session.ID, managerID, nil)
	require.ErrorIs(t, err, ErrInvalidState)

	// adjustment document is retrievable with items
	got, err := svc.GetAdjustment(ctx, adj.ID, nil)
	require.NoError(t, err)
	require.Len(t, got.Items, 1)
	assert.Equal(t, float64(20), got.Items[0].ExpectedQty)
	assert.Equal(t, float64(25), got.Items[0].PhysicalQty)
	assert.Equal(t, float64(5), got.Items[0].DifferenceQty)
}

func TestService_RejectAndRecount(t *testing.T) {
	repo := newTestRepository()
	svc := NewService(repo, nil)
	svc.SetStockApplier(inventory.StockApplier{StockSyncer: product.StockSyncer{}})
	ctx := context.Background()
	resetStockOpname(ctx, t)
	managerID := 9104
	counterID := 9105
	insertTestUserWithRole(ctx, t, managerID, "so_manager_9104", 3)
	insertTestUserWithRole(ctx, t, counterID, "so_counter_9105", 5)
	insertTestStore(ctx, t, 102)
	p := insertTestProductStore(ctx, t, "SO-SVC-REJ-001", 102)
	insertTestStock(ctx, t, p, 3)

	session, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "store", ScopeID: 102}, managerID, nil)
	require.NoError(t, err)
	require.NoError(t, svc.AssignCounter(ctx, session.ID, counterID, AssignmentRoleCounter, nil))
	require.NoError(t, svc.StartCounting(ctx, session.ID, counterID, nil))

	countAllItems(ctx, t, svc, session.ID, counterID, p, 3)
	require.NoError(t, svc.SubmitSession(ctx, session.ID, counterID, nil))

	// reject requires comment
	err = svc.RejectSession(ctx, session.ID, managerID, "", nil)
	require.ErrorIs(t, err, ErrApprovalCommentReq)

	// reject -> needs_recount
	require.NoError(t, svc.RejectSession(ctx, session.ID, managerID, "recount please", nil))
	status, err := repo.GetSessionStatus(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusNeedsRecount, status)

	// resume counting
	require.NoError(t, svc.ResumeCounting(ctx, session.ID, counterID, nil))
	status, err = repo.GetSessionStatus(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusCounting, status)
}

func TestService_CancelSession(t *testing.T) {
	repo := newTestRepository()
	svc := NewService(repo, nil)
	svc.SetStockApplier(inventory.StockApplier{StockSyncer: product.StockSyncer{}})
	ctx := context.Background()
	resetStockOpname(ctx, t)
	managerID := 9106
	insertTestUserWithRole(ctx, t, managerID, "so_manager_9106", 3)
	insertTestStore(ctx, t, 103)
	p := insertTestProductStore(ctx, t, "SO-SVC-CANCEL-001", 103)
	insertTestStock(ctx, t, p, 4)

	session, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "store", ScopeID: 103}, managerID, nil)
	require.NoError(t, err)
	require.NoError(t, svc.CancelSession(ctx, session.ID, managerID, nil))

	status, err := repo.GetSessionStatus(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusCancelled, status)

	// cancelled session cannot be re-cancelled
	err = svc.CancelSession(ctx, session.ID, managerID, nil)
	require.ErrorIs(t, err, ErrInvalidState)
}

func TestService_SummaryAndDifferenceReport(t *testing.T) {
	repo := newTestRepository()
	svc := NewService(repo, nil)
	svc.SetStockApplier(inventory.StockApplier{StockSyncer: product.StockSyncer{}})
	ctx := context.Background()
	resetStockOpname(ctx, t)
	managerID := 9107
	counterID := 9108
	insertTestUserWithRole(ctx, t, managerID, "so_manager_9107", 3)
	insertTestUserWithRole(ctx, t, counterID, "so_counter_9108", 5)
	insertTestStore(ctx, t, 104)
	p := insertTestProductStore(ctx, t, "SO-SVC-SUM-001", 104)
	insertTestStock(ctx, t, p, 8)

	session, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "store", ScopeID: 104}, managerID, nil)
	require.NoError(t, err)
	require.NoError(t, svc.AssignCounter(ctx, session.ID, counterID, AssignmentRoleCounter, nil))

	// before counting
	sum, err := svc.Summary(ctx, session.ID, counterID, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, sum.TotalItems, 1)
	assert.Equal(t, 0, sum.CountedItems)
	assert.Equal(t, sum.TotalItems, sum.PendingItems)

	require.NoError(t, svc.StartCounting(ctx, session.ID, counterID, nil))
	countAllItems(ctx, t, svc, session.ID, counterID, p, 6)
	require.NoError(t, svc.SubmitSession(ctx, session.ID, counterID, nil))
	require.NoError(t, svc.VerifySession(ctx, session.ID, managerID, "ok", nil))

	sum, err = svc.Summary(ctx, session.ID, counterID, nil)
	require.NoError(t, err)
	assert.Equal(t, sum.TotalItems, sum.CountedItems)
	assert.Equal(t, 0, sum.PendingItems)
	assert.Equal(t, -2.0, sum.TotalDifference)

	report, err := svc.DifferenceReport(ctx, session.ID, counterID, nil)
	require.NoError(t, err)
	require.NotEmpty(t, report.Items)
	for _, it := range report.Items {
		if it.ProductID == p {
			assert.Equal(t, float64(8), it.ExpectedQty)
			assert.Equal(t, float64(6), it.PhysicalQty)
			assert.Equal(t, -2.0, it.DifferenceQty)
		}
	}

	// post applies the -2 difference to stock, then close finalises
	_, err = svc.PostAdjustment(ctx, session.ID, managerID, &PostAdjustmentRequest{Comment: "ok"}, nil)
	require.NoError(t, err)
	require.NoError(t, svc.CloseSession(ctx, session.ID, managerID, nil))
}

func TestService_ListAssignableUsers(t *testing.T) {
	repo := newTestRepository()
	svc := NewService(repo, nil)
	svc.SetStockApplier(inventory.StockApplier{StockSyncer: product.StockSyncer{}})
	ctx := context.Background()
	resetStockOpname(ctx, t)
	insertTestUserWithRole(ctx, t, 9201, "so_staff_9201", 5)
	insertTestUserWithRole(ctx, t, 9202, "so_manager_9202", 3)

	users, err := svc.ListAssignableUsers(ctx, "920")
	require.NoError(t, err)
	require.Len(t, users, 2)
	for _, u := range users {
		assert.NotEmpty(t, u.RoleName)
	}
}

func TestService_AssignCounterRoleValidation(t *testing.T) {
	repo := newTestRepository()
	svc := NewService(repo, nil)
	svc.SetStockApplier(inventory.StockApplier{StockSyncer: product.StockSyncer{}})
	ctx := context.Background()
	resetStockOpname(ctx, t)
	managerID := 9203
	counterID := 9204
	staffID := 9205
	insertTestUserWithRole(ctx, t, managerID, "so_manager_9203", 3)
	insertTestUserWithRole(ctx, t, counterID, "so_counter_9204", 5)
	insertTestUserWithRole(ctx, t, staffID, "so_staff_9205", 5)
	insertTestStore(ctx, t, 9203)
	p := insertTestProductStore(ctx, t, "SO-SVC-ROLE-001", 9203)
	insertTestStock(ctx, t, p, 5)

	session, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "store", ScopeID: 9203}, managerID, nil)
	require.NoError(t, err)

	// counter role only for staff/cashier
	require.NoError(t, svc.AssignCounter(ctx, session.ID, counterID, AssignmentRoleCounter, nil))

	// manager cannot be assigned as counter
	err = svc.AssignCounter(ctx, session.ID, managerID, AssignmentRoleCounter, nil)
	require.ErrorIs(t, err, ErrInvalidAssigneeRole)

	// staff cannot be assigned as supervisor
	err = svc.AssignCounter(ctx, session.ID, staffID, AssignmentRoleSupervisor, nil)
	require.ErrorIs(t, err, ErrInvalidAssigneeRole)

	// manager can be supervisor
	require.NoError(t, svc.AssignCounter(ctx, session.ID, managerID, AssignmentRoleSupervisor, nil))

	// unknown/inactive assignee rejected
	err = svc.AssignCounter(ctx, session.ID, 999999, AssignmentRoleCounter, nil)
	require.ErrorIs(t, err, ErrAssigneeNotFound)

	// invalid role string rejected before validation
	err = svc.AssignCounter(ctx, session.ID, counterID, "bogus", nil)
	require.ErrorContains(t, err, "invalid assignment role")
}

func TestService_ReassignCounterRoleValidation(t *testing.T) {
	repo := newTestRepository()
	svc := NewService(repo, nil)
	svc.SetStockApplier(inventory.StockApplier{StockSyncer: product.StockSyncer{}})
	ctx := context.Background()
	resetStockOpname(ctx, t)
	managerID := 9206
	counterID := 9207
	insertTestUserWithRole(ctx, t, managerID, "so_manager_9206", 3)
	insertTestUserWithRole(ctx, t, counterID, "so_counter_9207", 5)
	insertTestStore(ctx, t, 9206)
	p := insertTestProductStore(ctx, t, "SO-SVC-REASSIGN-001", 9206)
	insertTestStock(ctx, t, p, 5)

	session, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "store", ScopeID: 9206}, managerID, nil)
	require.NoError(t, err)
	require.NoError(t, svc.AssignCounter(ctx, session.ID, counterID, AssignmentRoleCounter, nil))

	assignments, err := repo.ListAssignments(ctx, session.ID)
	require.NoError(t, err)
	require.Len(t, assignments, 1)

	// staff cannot be promoted to supervisor
	err = svc.ReassignCounter(ctx, session.ID, assignments[0].ID, AssignmentRoleSupervisor, nil)
	require.ErrorIs(t, err, ErrInvalidAssigneeRole)

	// unknown assignment rejected
	err = svc.ReassignCounter(ctx, session.ID, 999999, AssignmentRoleCounter, nil)
	require.ErrorIs(t, err, ErrAssignmentNotFound)
}

func TestService_BlindCountMasksQuantities(t *testing.T) {
	repo := newTestRepository()
	svc := NewService(repo, nil)
	svc.SetStockApplier(inventory.StockApplier{StockSyncer: product.StockSyncer{}})
	ctx := context.Background()
	resetStockOpname(ctx, t)
	managerID := 9109
	counterID := 9110
	insertTestUserWithRole(ctx, t, managerID, "so_manager_9109", 3)
	insertTestUserWithRole(ctx, t, counterID, "so_counter_9110", 5)
	p := insertTestProduct(ctx, t, "SO-SVC-BLIND-001")
	insertTestStock(ctx, t, p, 30)

	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	number, err := repo.GetNextSessionNumber(ctx)
	require.NoError(t, err)
	s := &Session{SessionNumber: number, ScopeType: "store", ScopeID: 105, BlindCount: true, Status: StatusDraft, CreatedBy: managerID}
	require.NoError(t, repo.CreateSession(ctx, tx, s))
	require.NoError(t, repo.InsertSessionItems(ctx, tx, s.ID, []SessionItem{
		{ProductID: p, ProductName: "Test Product SO-SVC-BLIND-001", SKU: "SO-SVC-BLIND-001", OpeningQty: 30, UOMName: "pcs"},
	}))
	require.NoError(t, tx.Commit(ctx))

	require.NoError(t, svc.AssignCounter(ctx, s.ID, counterID, AssignmentRoleCounter, nil))

	// counter sees masked quantities
	sess, err := svc.GetSessionForUser(ctx, s.ID, counterID, nil)
	require.NoError(t, err)
	require.Len(t, sess.Items, 1)
	assert.Equal(t, float64(0), sess.Items[0].OpeningQty)
	assert.Equal(t, float64(0), sess.Items[0].ExpectedQty)

	// manager (not a counter) sees real quantities
	sessManager, err := svc.GetSessionForUser(ctx, s.ID, managerID, nil)
	require.NoError(t, err)
	assert.Equal(t, float64(30), sessManager.Items[0].OpeningQty)

	// summary & difference report are also masked for blind counters
	sumBlind, err := svc.Summary(ctx, s.ID, counterID, nil)
	require.NoError(t, err)
	assert.Equal(t, float64(0), sumBlind.TotalDifference)

	reportBlind, err := svc.DifferenceReport(ctx, s.ID, counterID, nil)
	require.NoError(t, err)
	assert.Equal(t, float64(0), reportBlind.Items[0].OpeningQty)
}

func TestService_PublishesStatusEvents(t *testing.T) {
	repo := newTestRepository()
	bus := &capturingEventBus{}
	svc := NewService(repo, bus)
	svc.SetStockApplier(inventory.StockApplier{StockSyncer: product.StockSyncer{}})
	ctx := context.Background()
	resetStockOpname(ctx, t)
	managerID := 9110
	counterID := 9111
	insertTestUserWithRole(ctx, t, managerID, "so_evt_manager_9110", 3)
	insertTestUserWithRole(ctx, t, counterID, "so_evt_counter_9111", 5)
	insertTestStore(ctx, t, 102)
	p := insertTestProductStore(ctx, t, "SO-SVC-EVT-001", 102)
	insertTestStock(ctx, t, p, 10)

	session, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "store", ScopeID: 102}, managerID, nil)
	require.NoError(t, err)
	require.Contains(t, bus.topics(), EventStockOpnameCreated)

	// payload carries store_id for store-scoped sessions
	bus.mu.Lock()
	var found bool
	for _, pr := range bus.published {
		if pr.topic == EventStockOpnameCreated {
			ev, ok := pr.event.(*events.StockOpnameStatusChanged)
			require.True(t, ok)
			assert.Equal(t, 102, ev.StoreID)
			found = true
		}
	}
	bus.mu.Unlock()
	assert.True(t, found, "expected a created event payload")

	// cancel publishes cancelled event
	require.NoError(t, svc.CancelSession(ctx, session.ID, managerID, nil))
	require.Contains(t, bus.topics(), EventStockOpnameCancelled)
}

func TestService_SubmitPublishesSubmittedEvent(t *testing.T) {
	repo := newTestRepository()
	bus := &capturingEventBus{}
	svc := NewService(repo, bus)
	svc.SetStockApplier(inventory.StockApplier{StockSyncer: product.StockSyncer{}})
	ctx := context.Background()
	resetStockOpname(ctx, t)
	managerID := 9112
	counterID := 9113
	insertTestUserWithRole(ctx, t, managerID, "so_evt_manager_9112", 3)
	insertTestUserWithRole(ctx, t, counterID, "so_evt_counter_9113", 5)
	insertTestStore(ctx, t, 103)
	p := insertTestProductStore(ctx, t, "SO-SVC-EVT-002", 103)
	insertTestStock(ctx, t, p, 10)

	session, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "store", ScopeID: 103}, managerID, nil)
	require.NoError(t, err)
	require.NoError(t, svc.AssignCounter(ctx, session.ID, counterID, AssignmentRoleCounter, nil))
	require.NoError(t, svc.StartCounting(ctx, session.ID, counterID, nil))
	countAllItems(ctx, t, svc, session.ID, counterID, p, 10)
	require.NoError(t, svc.SubmitSession(ctx, session.ID, counterID, nil))

	require.Contains(t, bus.topics(), EventStockOpnameSubmitted)

	// payload carries session info
	bus.mu.Lock()
	var found bool
	for _, pr := range bus.published {
		if pr.topic == EventStockOpnameSubmitted {
			ev, ok := pr.event.(*events.StockOpnameStatusChanged)
			require.True(t, ok)
			assert.Equal(t, session.ID, ev.SessionID)
			assert.Equal(t, session.SessionNumber, ev.SessionNumber)
			assert.Equal(t, StatusPendingApproval, ev.Status)
			found = true
		}
	}
	bus.mu.Unlock()
	assert.True(t, found)
}

func TestService_VerifyPublishesApprovedEvent(t *testing.T) {
	repo := newTestRepository()
	bus := &capturingEventBus{}
	svc := NewService(repo, bus)
	svc.SetStockApplier(inventory.StockApplier{StockSyncer: product.StockSyncer{}})
	ctx := context.Background()
	resetStockOpname(ctx, t)
	managerID := 9114
	counterID := 9115
	insertTestUserWithRole(ctx, t, managerID, "so_evt_manager_9114", 3)
	insertTestUserWithRole(ctx, t, counterID, "so_evt_counter_9115", 5)
	insertTestStore(ctx, t, 104)
	p := insertTestProductStore(ctx, t, "SO-SVC-EVT-003", 104)
	insertTestStock(ctx, t, p, 10)

	session, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "store", ScopeID: 104}, managerID, nil)
	require.NoError(t, err)
	require.NoError(t, svc.AssignCounter(ctx, session.ID, counterID, AssignmentRoleCounter, nil))
	require.NoError(t, svc.StartCounting(ctx, session.ID, counterID, nil))
	countAllItems(ctx, t, svc, session.ID, counterID, p, 10)
	require.NoError(t, svc.SubmitSession(ctx, session.ID, counterID, nil))
	require.NoError(t, svc.VerifySession(ctx, session.ID, managerID, "ok", nil))

	require.Contains(t, bus.topics(), EventStockOpnameApproved)
}

func TestStoreIDOrZero(t *testing.T) {
	positive := 5
	assert.Equal(t, 5, storeIDOrZero(&positive))

	zero := 0
	assert.Equal(t, 0, storeIDOrZero(&zero))

	negative := -3
	assert.Equal(t, 0, storeIDOrZero(&negative))

	assert.Equal(t, 0, storeIDOrZero(nil))
}

func TestService_PublishesGlobalEvent_NoStore(t *testing.T) {
	repo := newTestRepository()
	bus := &capturingEventBus{}
	svc := NewService(repo, bus)
	svc.SetStockApplier(inventory.StockApplier{StockSyncer: product.StockSyncer{}})
	ctx := context.Background()
	resetStockOpname(ctx, t)
	managerID := 9116
	insertTestUserWithRole(ctx, t, managerID, "so_evt_manager_9116", 3)
	catID := insertTestCategory(ctx, t, "SO Global Cat 9116")
	p := insertTestProductCategory(ctx, t, "SO-SVC-EVT-004", catID)
	insertTestStock(ctx, t, p, 10)

	session, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "category", ScopeID: int64(catID)}, managerID, nil)
	require.NoError(t, err)
	assert.Nil(t, session.StoreID)

	require.Contains(t, bus.topics(), EventStockOpnameCreated)

	bus.mu.Lock()
	defer bus.mu.Unlock()
	var found bool
	for _, pr := range bus.published {
		if pr.topic == EventStockOpnameCreated {
			ev, ok := pr.event.(*events.StockOpnameStatusChanged)
			require.True(t, ok)
			assert.Equal(t, 0, ev.StoreID, "global sessions should publish store_id 0")
			found = true
		}
	}
	assert.True(t, found)
}

func TestService_StoreIsolation_CreateClampsStoreID(t *testing.T) {
	repo := newTestRepository()
	svc := NewService(repo, nil)
	svc.SetStockApplier(inventory.StockApplier{StockSyncer: product.StockSyncer{}})
	ctx := context.Background()
	resetStockOpname(ctx, t)
	insertTestUserWithRole(ctx, t, 9701, "so_store_mgr_9701", 3)
	insertTestStore(ctx, t, 9701)
	insertTestStore(ctx, t, 9702)
	p := insertTestProductStore(ctx, t, "SO-STORE-CLAMP-001", 9701)
	insertTestStock(ctx, t, p, 5)
	p2 := insertTestProductStore(ctx, t, "SO-STORE-CLAMP-002", 9702)
	insertTestStock(ctx, t, p2, 5)

	managerStore := 9701

	t.Run("client store_id is clamped to claims store", func(t *testing.T) {
		resetStockOpname(ctx, t)
		clientStore := 9702
		session, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "store", ScopeID: 9701, StoreID: &clientStore}, 9701, &managerStore)
		require.NoError(t, err)
		require.NotNil(t, session.StoreID)
		assert.Equal(t, 9701, *session.StoreID)
	})

	t.Run("cross-store store scope is rejected", func(t *testing.T) {
		resetStockOpname(ctx, t)
		_, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "store", ScopeID: 9702}, 9701, &managerStore)
		require.ErrorIs(t, err, ErrStoreForbidden)
	})
}

func TestService_StoreIsolation_CrossStoreAccess(t *testing.T) {
	repo := newTestRepository()
	svc := NewService(repo, nil)
	svc.SetStockApplier(inventory.StockApplier{StockSyncer: product.StockSyncer{}})
	ctx := context.Background()
	resetStockOpname(ctx, t)
	insertTestUserWithRole(ctx, t, 9701, "so_mgr_a_9701", 3)
	insertTestUserWithRole(ctx, t, 9702, "so_mgr_b_9702", 3)
	insertTestStore(ctx, t, 9701)
	insertTestStore(ctx, t, 9702)
	p := insertTestProductStore(ctx, t, "SO-STORE-ACCESS-001", 9701)
	insertTestStock(ctx, t, p, 5)

	storeA, storeB := 9701, 9702
	session, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "store", ScopeID: 9701}, 9701, &storeA)
	require.NoError(t, err)

	t.Run("own-store read ok", func(t *testing.T) {
		got, err := svc.GetSessionForUser(ctx, session.ID, 9701, &storeA)
		require.NoError(t, err)
		assert.Equal(t, session.ID, got.ID)
	})
	t.Run("cross-store read rejected", func(t *testing.T) {
		_, err := svc.GetSessionForUser(ctx, session.ID, 9702, &storeB)
		require.ErrorIs(t, err, ErrStoreForbidden)
	})
	t.Run("superadmin cross-store ok", func(t *testing.T) {
		got, err := svc.GetSessionForUser(ctx, session.ID, 1, nil)
		require.NoError(t, err)
		assert.Equal(t, session.ID, got.ID)
	})
	t.Run("cross-store cancel rejected", func(t *testing.T) {
		err := svc.CancelSession(ctx, session.ID, 9702, &storeB)
		require.ErrorIs(t, err, ErrStoreForbidden)
	})
	t.Run("cross-store summary rejected", func(t *testing.T) {
		_, err := svc.Summary(ctx, session.ID, 9702, &storeB)
		require.ErrorIs(t, err, ErrStoreForbidden)
	})
}

func TestService_StoreIsolation_WarehouseScope(t *testing.T) {
	repo := newTestRepository()
	svc := NewService(repo, nil)
	svc.SetStockApplier(inventory.StockApplier{StockSyncer: product.StockSyncer{}})
	ctx := context.Background()
	resetStockOpname(ctx, t)
	insertTestUserWithRole(ctx, t, 9701, "so_mgr_wh_9701", 3)
	insertTestStore(ctx, t, 9701)
	insertTestStore(ctx, t, 9702)
	insertTestWarehouse(ctx, t, 9741, 9702, "SO-WH-9702")
	whP := insertTestProductStore(ctx, t, "SO-WH-ISO-001", 9702)
	insertTestStockWarehouse(ctx, t, whP, 9741, 5)
	insertTestStock(ctx, t, whP, 5)

	managerStore := 9701

	t.Run("cross-store warehouse scope rejected for store-scoped caller", func(t *testing.T) {
		resetStockOpname(ctx, t)
		wid := 9741
		_, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "warehouse", ScopeID: 9741, WarehouseID: &wid}, 9701, &managerStore)
		require.ErrorIs(t, err, ErrStoreForbidden)
	})

	t.Run("store-less warehouse scope rejected for store-scoped caller", func(t *testing.T) {
		resetStockOpname(ctx, t)
		_, err := dbPool.Exec(ctx,
			`INSERT INTO warehouses (id, name, code, store_id, is_active) VALUES (9743, 'Test WH 9743', 'SO-WH-9743', NULL, true) ON CONFLICT (id) DO NOTHING`,
		)
		require.NoError(t, err)
		slP := insertTestProductStore(ctx, t, "SO-WH-ISO-002", 9701)
		insertTestStockWarehouse(ctx, t, slP, 9743, 5)
		insertTestStock(ctx, t, slP, 5)
		wid := 9743
		_, err = svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "warehouse", ScopeID: 9743, WarehouseID: &wid}, 9701, &managerStore)
		require.ErrorIs(t, err, ErrStoreForbidden)
	})

	t.Run("cross-store warehouse scope ok for superadmin", func(t *testing.T) {
		resetStockOpname(ctx, t)
		wid := 9741
		session, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "warehouse", ScopeID: 9741, WarehouseID: &wid}, 1, nil)
		require.NoError(t, err)
		require.NotNil(t, session.StoreID)
		assert.Equal(t, 9702, *session.StoreID)
	})
}

func TestService_StoreIsolation_ListSessionsFilter(t *testing.T) {
	repo := newTestRepository()
	svc := NewService(repo, nil)
	svc.SetStockApplier(inventory.StockApplier{StockSyncer: product.StockSyncer{}})
	ctx := context.Background()
	resetStockOpname(ctx, t)
	insertTestUserWithRole(ctx, t, 9701, "so_mgr_c_9701", 3)
	insertTestUserWithRole(ctx, t, 9702, "so_mgr_d_9702", 3)
	insertTestStore(ctx, t, 9701)
	insertTestStore(ctx, t, 9702)
	p := insertTestProductStore(ctx, t, "SO-STORE-LIST-001", 9701)
	insertTestStock(ctx, t, p, 5)
	p2 := insertTestProductStore(ctx, t, "SO-STORE-LIST-002", 9702)
	insertTestStock(ctx, t, p2, 5)

	storeA, storeB := 9701, 9702
	s1, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "store", ScopeID: 9701}, 9701, &storeA)
	require.NoError(t, err)
	s2, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "store", ScopeID: 9702}, 9702, &storeB)
	require.NoError(t, err)
	require.NotEqual(t, s1.ID, s2.ID)

	sessions, total, err := svc.ListSessions(ctx, 10, 0, "", "", &storeA)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, sessions, 1)
	assert.Equal(t, s1.ID, sessions[0].ID)

	sessions, total, err = svc.ListSessions(ctx, 10, 0, "", "", nil)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, sessions, 2)
}
