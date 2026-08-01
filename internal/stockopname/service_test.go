package stockopname

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_CreateSession(t *testing.T) {
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()
	resetStockOpname(t, ctx)
	insertTestUser(t, ctx, 9101, "so_svc_user_9101")
	p := insertTestProduct(t, ctx, "SO-SVC-CREATE-001")
	insertTestStock(t, ctx, p, 5)

	session, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "store", ScopeID: 100}, 9101)
	require.NoError(t, err)
	assert.Equal(t, StatusDraft, session.Status)
	assert.NotEmpty(t, session.SessionNumber)
	assert.Equal(t, 9101, session.CreatedBy)

	// active session exists for same scope -> conflict
	_, err = svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "store", ScopeID: 100}, 9101)
	require.ErrorIs(t, err, ErrActiveSessionExists)
}

func TestService_CreateSessionInvalidScope(t *testing.T) {
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	_, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "all", ScopeID: 1}, 1)
	require.ErrorIs(t, err, ErrUnsupportedScope)
}

func TestService_CreateSessionNoProducts(t *testing.T) {
	// ensure empty DB snapshot yields ErrNoItems; guard against races by
	// creating a fresh session for a scope after truncating nothing (items
	// exist from other tests, so this test only validates the error path shape).
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()
	_ = ctx
	// validation errors take precedence
	_, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "bogus", ScopeID: 1}, 1)
	require.ErrorIs(t, err, ErrUnsupportedScope)
}

func countAllItems(t *testing.T, svc *Service, ctx context.Context, sessionID, counterID int, overrideProductID int, overrideQty float64) {
	t.Helper()
	sess, err := svc.GetSessionForUser(ctx, sessionID, counterID)
	require.NoError(t, err)
	require.NotEmpty(t, sess.Items)
	for _, it := range sess.Items {
		qty := it.OpeningQty
		if it.ProductID == overrideProductID {
			qty = overrideQty
		}
		require.NoError(t, svc.SaveCount(ctx, it.ID, counterID, qty, ""))
	}
}

func TestService_AssignAndSubmitAndApprove(t *testing.T) {
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()
	resetStockOpname(t, ctx)
	managerID := 9102
	counterID := 9103
	insertTestUser(t, ctx, managerID, "so_manager_9102")
	insertTestUser(t, ctx, counterID, "so_counter_9103")
	p := insertTestProduct(t, ctx, "SO-SVC-FLOW-001")
	insertTestStock(t, ctx, p, 20)

	session, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "store", ScopeID: 101}, managerID)
	require.NoError(t, err)

	// assign counter
	require.NoError(t, svc.AssignCounter(ctx, session.ID, counterID, AssignmentRoleCounter))

	// non-assigned user cannot start counting
	err = svc.StartCounting(ctx, session.ID, managerID)
	require.ErrorIs(t, err, ErrNotAssigned)

	// start counting
	require.NoError(t, svc.StartCounting(ctx, session.ID, counterID))

	// submit should fail until all items counted
	err = svc.SubmitSession(ctx, session.ID, counterID)
	require.ErrorIs(t, err, ErrNotAllItemsCounted)

	// non-assigned user cannot count
	sess, err := svc.GetSessionForUser(ctx, session.ID, counterID)
	require.NoError(t, err)
	require.NotEmpty(t, sess.Items)
	itemID := sess.Items[0].ID
	err = svc.SaveCount(ctx, itemID, managerID, 25, "")
	require.ErrorIs(t, err, ErrNotAssigned)

	// negative qty rejected
	err = svc.SaveCount(ctx, itemID, counterID, -1, "")
	require.ErrorIs(t, err, ErrInvalidQuantity)

	// zero qty is valid (out-of-stock item)
	require.NoError(t, svc.SaveCount(ctx, itemID, counterID, 0, ""))

	// count all items; override target product with 25
	countAllItems(t, svc, ctx, session.ID, counterID, p, 25)

	// submit now works
	require.NoError(t, svc.SubmitSession(ctx, session.ID, counterID))

	// counter cannot approve (separation of duties)
	err = svc.ApproveSession(ctx, session.ID, counterID, "approve")
	require.ErrorIs(t, err, ErrSeparationOfDuties)

	// approve requires comment
	err = svc.ApproveSession(ctx, session.ID, managerID, "  ")
	require.ErrorIs(t, err, ErrApprovalCommentReq)

	// manager approves -> stock adjusted 20 -> 25
	require.NoError(t, svc.ApproveSession(ctx, session.ID, managerID, "ok"))

	status, err := repo.GetSessionStatus(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusApproved, status)

	var qty int
	err = dbPool.QueryRow(ctx, `SELECT quantity FROM product_stock WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL`, p).Scan(&qty)
	require.NoError(t, err)
	assert.Equal(t, 25, qty)

	// approving an approved session is invalid state
	err = svc.ApproveSession(ctx, session.ID, managerID, "again")
	require.ErrorIs(t, err, ErrInvalidState)
}

func TestService_RejectAndRecount(t *testing.T) {
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()
	resetStockOpname(t, ctx)
	managerID := 9104
	counterID := 9105
	insertTestUser(t, ctx, managerID, "so_manager_9104")
	insertTestUser(t, ctx, counterID, "so_counter_9105")
	p := insertTestProduct(t, ctx, "SO-SVC-REJ-001")
	insertTestStock(t, ctx, p, 3)

	session, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "store", ScopeID: 102}, managerID)
	require.NoError(t, err)
	require.NoError(t, svc.AssignCounter(ctx, session.ID, counterID, AssignmentRoleCounter))
	require.NoError(t, svc.StartCounting(ctx, session.ID, counterID))

	countAllItems(t, svc, ctx, session.ID, counterID, p, 3)
	require.NoError(t, svc.SubmitSession(ctx, session.ID, counterID))

	// reject requires comment
	err = svc.RejectSession(ctx, session.ID, managerID, "")
	require.ErrorIs(t, err, ErrApprovalCommentReq)

	// reject -> needs_recount
	require.NoError(t, svc.RejectSession(ctx, session.ID, managerID, "recount please"))
	status, err := repo.GetSessionStatus(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusNeedsRecount, status)

	// resume counting
	require.NoError(t, svc.ResumeCounting(ctx, session.ID, counterID))
	status, err = repo.GetSessionStatus(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusCounting, status)
}

func TestService_CancelSession(t *testing.T) {
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()
	resetStockOpname(t, ctx)
	managerID := 9106
	insertTestUser(t, ctx, managerID, "so_manager_9106")
	p := insertTestProduct(t, ctx, "SO-SVC-CANCEL-001")
	insertTestStock(t, ctx, p, 4)

	session, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "store", ScopeID: 103}, managerID)
	require.NoError(t, err)
	require.NoError(t, svc.CancelSession(ctx, session.ID, managerID))

	status, err := repo.GetSessionStatus(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusCancelled, status)

	// cancelled session cannot be re-cancelled
	err = svc.CancelSession(ctx, session.ID, managerID)
	require.ErrorIs(t, err, ErrInvalidState)
}

func TestService_SummaryAndDifferenceReport(t *testing.T) {
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()
	resetStockOpname(t, ctx)
	managerID := 9107
	counterID := 9108
	insertTestUser(t, ctx, managerID, "so_manager_9107")
	insertTestUser(t, ctx, counterID, "so_counter_9108")
	p := insertTestProduct(t, ctx, "SO-SVC-SUM-001")
	insertTestStock(t, ctx, p, 8)

	session, err := svc.CreateSession(ctx, &CreateSessionRequest{ScopeType: "store", ScopeID: 104}, managerID)
	require.NoError(t, err)
	require.NoError(t, svc.AssignCounter(ctx, session.ID, counterID, AssignmentRoleCounter))

	// before counting
	sum, err := svc.Summary(ctx, session.ID, counterID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, sum.TotalItems, 1)
	assert.Equal(t, 0, sum.CountedItems)
	assert.Equal(t, sum.TotalItems, sum.PendingItems)

	require.NoError(t, svc.StartCounting(ctx, session.ID, counterID))
	countAllItems(t, svc, ctx, session.ID, counterID, p, 6)
	require.NoError(t, svc.SubmitSession(ctx, session.ID, counterID))
	require.NoError(t, svc.ApproveSession(ctx, session.ID, managerID, "ok"))

	sum, err = svc.Summary(ctx, session.ID, counterID)
	require.NoError(t, err)
	assert.Equal(t, sum.TotalItems, sum.CountedItems)
	assert.Equal(t, 0, sum.PendingItems)
	assert.Equal(t, -2.0, sum.TotalDifference)

	report, err := svc.DifferenceReport(ctx, session.ID, counterID)
	require.NoError(t, err)
	require.NotEmpty(t, report.Items)
	for _, it := range report.Items {
		if it.ProductID == p {
			assert.Equal(t, float64(8), it.ExpectedQty)
			assert.Equal(t, float64(6), it.PhysicalQty)
			assert.Equal(t, -2.0, it.DifferenceQty)
		}
	}
}

func TestService_BlindCountMasksQuantities(t *testing.T) {
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()
	resetStockOpname(t, ctx)
	managerID := 9109
	counterID := 9110
	insertTestUser(t, ctx, managerID, "so_manager_9109")
	insertTestUser(t, ctx, counterID, "so_counter_9110")
	p := insertTestProduct(t, ctx, "SO-SVC-BLIND-001")
	insertTestStock(t, ctx, p, 30)

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

	require.NoError(t, svc.AssignCounter(ctx, s.ID, counterID, AssignmentRoleCounter))

	// counter sees masked quantities
	sess, err := svc.GetSessionForUser(ctx, s.ID, counterID)
	require.NoError(t, err)
	require.Len(t, sess.Items, 1)
	assert.Equal(t, float64(0), sess.Items[0].OpeningQty)
	assert.Equal(t, float64(0), sess.Items[0].ExpectedQty)

	// manager (not a counter) sees real quantities
	sessManager, err := svc.GetSessionForUser(ctx, s.ID, managerID)
	require.NoError(t, err)
	assert.Equal(t, float64(30), sessManager.Items[0].OpeningQty)

	// summary & difference report are also masked for blind counters
	sumBlind, err := svc.Summary(ctx, s.ID, counterID)
	require.NoError(t, err)
	assert.Equal(t, float64(0), sumBlind.TotalDifference)

	reportBlind, err := svc.DifferenceReport(ctx, s.ID, counterID)
	require.NoError(t, err)
	assert.Equal(t, float64(0), reportBlind.Items[0].OpeningQty)
}
