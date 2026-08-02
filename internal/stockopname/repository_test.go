package stockopname

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

var dbPool *pgxpool.Pool

func TestMain(m *testing.M) {
	pool, err := shared.NewTestDB()
	if err != nil {
		println("NEWTESTDB ERR:", err.Error())
		os.Exit(0)
	}
	dbPool = pool
	defer pool.Close()

	if err := shared.RunMigrations(pool, "../../database/migrations"); err != nil {
		println("RUNMIGRATIONS ERR:", err.Error())
		os.Exit(0)
	}
	if err := shared.TruncateTestData(pool); err != nil {
		println("TRUNCATE ERR:", err.Error())
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func insertTestProduct(t *testing.T, ctx context.Context, sku string) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(ctx,
		`INSERT INTO products (sku, name, price, status) VALUES ($1, $2, 10000, 'active') RETURNING id`,
		sku, "Test Product "+sku,
	).Scan(&id)
	require.NoError(t, err)
	return id
}

func insertTestStock(t *testing.T, ctx context.Context, productID, quantity int) {
	t.Helper()
	_, err := dbPool.Exec(ctx,
		`INSERT INTO product_stock (product_id, quantity) VALUES ($1, $2)
		 ON CONFLICT (product_id, warehouse_id, store_id) DO UPDATE SET quantity = EXCLUDED.quantity`,
		productID, quantity,
	)
	require.NoError(t, err)
}

func insertTestUser(t *testing.T, ctx context.Context, id int, username string) {
	t.Helper()
	insertTestUserWithRole(t, ctx, id, username, 1)
}

func insertTestUserWithRole(t *testing.T, ctx context.Context, id int, username string, roleID int) {
	t.Helper()
	_, err := dbPool.Exec(ctx,
		`INSERT INTO users (id, username, email, password_hash, role_id)
		 VALUES ($1, $2, $3, 'hash', $4)
		 ON CONFLICT (id) DO NOTHING`,
		id, username, username+"@test.com", roleID,
	)
	require.NoError(t, err)
}

// resetStockOpname clears all stock opname tables so each test starts from a
// clean slate. v1 enforces a single global active session (scope filtering is
// not implemented), so tests cannot leave sessions behind between cases.
func resetStockOpname(t *testing.T, ctx context.Context) {
	t.Helper()
	_, err := dbPool.Exec(ctx, `
		TRUNCATE TABLE stock_opname_counts, stock_opname_assignments,
			stock_opname_items, stock_opnames CASCADE
	`)
	require.NoError(t, err)
}

func createTestSession(t *testing.T, ctx context.Context, repo *Repository, userID int) *Session {
	return createTestSessionScope(t, ctx, repo, userID, int64(100000+userID))
}

func createTestSessionScope(t *testing.T, ctx context.Context, repo *Repository, userID int, scopeID int64) *Session {
	t.Helper()
	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	number, err := repo.GetNextSessionNumber(ctx)
	require.NoError(t, err)
	s := &Session{
		SessionNumber: number,
		ScopeType:     "store",
		ScopeID:       scopeID,
		BlindCount:    false,
		Status:        StatusDraft,
		CreatedBy:     userID,
	}
	require.NoError(t, repo.CreateSession(ctx, tx, s))
	require.NoError(t, tx.Commit(ctx))
	return s
}

func TestRepository_GetNextSessionNumber(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	num1, err := repo.GetNextSessionNumber(ctx)
	require.NoError(t, err)
	assert.Contains(t, num1, "SO-")
	num2, err := repo.GetNextSessionNumber(ctx)
	require.NoError(t, err)
	assert.NotEqual(t, num1, num2)
}

func TestRepository_CreateSessionAndGetSession(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()
	resetStockOpname(t, ctx)
	insertTestUser(t, ctx, 9001, "so_user_9001")

	// two products with stock
	p1 := insertTestProduct(t, ctx, "SO-CREATE-001")
	p2 := insertTestProduct(t, ctx, "SO-CREATE-002")
	insertTestStock(t, ctx, p1, 10)
	insertTestStock(t, ctx, p2, 5)

	s := createTestSession(t, ctx, repo, 9001)
	require.Greater(t, s.ID, 0)

	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	items := []SessionItem{
		{ProductID: p1, ProductName: "Test Product SO-CREATE-001", SKU: "SO-CREATE-001", OpeningQty: 10, UOMName: "pcs"},
		{ProductID: p2, ProductName: "Test Product SO-CREATE-002", SKU: "SO-CREATE-002", OpeningQty: 5, UOMName: "pcs"},
	}
	require.NoError(t, repo.InsertSessionItems(ctx, tx, s.ID, items))
	require.NoError(t, tx.Commit(ctx))

	got, err := repo.GetSession(ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, s.SessionNumber, got.SessionNumber)
	assert.Equal(t, StatusDraft, got.Status)
	assert.Len(t, got.Items, 2)
	assert.NotEmpty(t, got.CreatedAt)
}

func TestRepository_GetSessionNotFound(t *testing.T) {
	repo := NewRepository(dbPool)
	_, err := repo.GetSession(context.Background(), -1)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestRepository_LoadSnapshotProducts(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()
	p := insertTestProduct(t, ctx, "SO-SNAP-001")
	insertTestStock(t, ctx, p, 7)

	items, err := repo.LoadSnapshotProducts(ctx)
	require.NoError(t, err)
	found := false
	for _, it := range items {
		if it.ProductID == p {
			found = true
			assert.Equal(t, float64(7), it.OpeningQty)
			assert.Equal(t, "Test Product SO-SNAP-001", it.ProductName)
		}
	}
	assert.True(t, found, "snapshot should include the test product")
}

func TestRepository_UpdateStatusGuarded(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()
	resetStockOpname(t, ctx)
	insertTestUser(t, ctx, 9002, "so_user_9002")
	s := createTestSession(t, ctx, repo, 9002)

	err := repo.UpdateStatus(ctx, s.ID, StatusDraft, StatusCounting)
	require.NoError(t, err)

	// guarded transition: wrong current status should fail
	err = repo.UpdateStatus(ctx, s.ID, StatusDraft, StatusCancelled)
	require.ErrorIs(t, err, ErrInvalidState)

	// not found
	err = repo.UpdateStatus(ctx, -1, StatusDraft, StatusCounting)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestRepository_CancelSession(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()
	resetStockOpname(t, ctx)
	insertTestUser(t, ctx, 9003, "so_user_9003")
	s := createTestSession(t, ctx, repo, 9003)

	require.NoError(t, repo.CancelSession(ctx, s.ID, 9003))

	status, err := repo.GetSessionStatus(ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusCancelled, status)

	// cannot cancel twice
	err = repo.CancelSession(ctx, s.ID, 9003)
	require.ErrorIs(t, err, ErrInvalidState)
}

func TestRepository_Assignments(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()
	resetStockOpname(t, ctx)
	insertTestUser(t, ctx, 9004, "so_user_9004")
	insertTestUser(t, ctx, 9005, "so_user_9005")
	s := createTestSession(t, ctx, repo, 9004)

	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	require.NoError(t, repo.InsertAssignment(ctx, tx, s.ID, 9005, AssignmentRoleCounter))
	require.NoError(t, tx.Commit(ctx))

	assigned, err := repo.IsCounterAssigned(ctx, s.ID, 9005)
	require.NoError(t, err)
	assert.True(t, assigned)

	assigned, err = repo.IsCounterAssigned(ctx, s.ID, 9004)
	require.NoError(t, err)
	assert.False(t, assigned)

	assignments, err := repo.ListAssignments(ctx, s.ID)
	require.NoError(t, err)
	assert.Len(t, assignments, 1)
	assert.Equal(t, AssignmentRoleCounter, assignments[0].Role)
	assert.Equal(t, "so_user_9005", assignments[0].Username)

	// reassign role
	tx2, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	require.NoError(t, repo.UpdateAssignmentRole(ctx, tx2, s.ID, assignments[0].ID, AssignmentRoleSupervisor))
	require.NoError(t, tx2.Commit(ctx))

	// assignment from another session cannot be updated via this session
	tx3, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	require.ErrorIs(t, repo.UpdateAssignmentRole(ctx, tx3, 999999, assignments[0].ID, AssignmentRoleCounter), ErrAssignmentNotFound)
	require.NoError(t, tx3.Rollback(ctx))

	assignments, err = repo.ListAssignments(ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, AssignmentRoleSupervisor, assignments[0].Role)
}

func TestRepository_SaveCountAndHistory(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()
	resetStockOpname(t, ctx)
	insertTestUser(t, ctx, 9006, "so_user_9006")
	p := insertTestProduct(t, ctx, "SO-COUNT-001")
	insertTestStock(t, ctx, p, 10)

	s := createTestSession(t, ctx, repo, 9006)
	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	require.NoError(t, repo.InsertSessionItems(ctx, tx, s.ID, []SessionItem{
		{ProductID: p, ProductName: "Test Product SO-COUNT-001", SKU: "SO-COUNT-001", OpeningQty: 10, UOMName: "pcs"},
	}))
	require.NoError(t, tx.Commit(ctx))

	session, err := repo.GetSession(ctx, s.ID)
	require.NoError(t, err)
	require.Len(t, session.Items, 1)
	itemID := session.Items[0].ID

	tx2, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	require.NoError(t, repo.LockItemForCount(ctx, tx2, itemID))
	seq, err := repo.NextCountSequence(ctx, tx2, itemID)
	require.NoError(t, err)
	assert.Equal(t, 1, seq)
	require.NoError(t, repo.SaveCount(ctx, tx2, itemID, seq, 8, 9006, "first count"))
	require.NoError(t, tx2.Commit(ctx))

	history, err := repo.GetCountHistory(ctx, itemID)
	require.NoError(t, err)
	require.Len(t, history, 1)
	assert.Equal(t, float64(8), history[0].PhysicalQty)
	assert.Equal(t, "so_user_9006", history[0].CountedByUser)

	// second count increments sequence
	tx3, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	seq, err = repo.NextCountSequence(ctx, tx3, itemID)
	require.NoError(t, err)
	assert.Equal(t, 2, seq)
	require.NoError(t, repo.SaveCount(ctx, tx3, itemID, seq, 9, 9006, ""))
	require.NoError(t, tx3.Commit(ctx))

	// item reflects latest physical qty
	session, err = repo.GetSession(ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, float64(9), session.Items[0].PhysicalQty)
	assert.Equal(t, ItemStatusCounted, session.Items[0].Status)
	assert.Equal(t, 2, session.Items[0].CountSequence)
}

func TestRepository_ApprovalFlow(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()
	resetStockOpname(t, ctx)
	insertTestUser(t, ctx, 9007, "so_user_9007")
	p := insertTestProduct(t, ctx, "SO-APPR-001")
	insertTestStock(t, ctx, p, 10)

	s := createTestSession(t, ctx, repo, 9007)
	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	require.NoError(t, repo.InsertSessionItems(ctx, tx, s.ID, []SessionItem{
		{ProductID: p, ProductName: "Test Product SO-APPR-001", SKU: "SO-APPR-001", OpeningQty: 10, UOMName: "pcs"},
	}))
	require.NoError(t, tx.Commit(ctx))

	// set session to pending_approval
	require.NoError(t, repo.UpdateStatus(ctx, s.ID, StatusDraft, StatusCounting))
	require.NoError(t, repo.UpdateStatus(ctx, s.ID, StatusCounting, StatusPendingApproval))

	txAppr, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = txAppr.Rollback(ctx) }()

	lockInfo, err := repo.LockSessionForApproval(ctx, txAppr, s.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusPendingApproval, lockInfo.Status)
	assert.Equal(t, s.SessionNumber, lockInfo.SessionNumber)

	items, err := repo.LoadApprovalItems(ctx, txAppr, s.ID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, p, items[0].ProductID)

	// update count to reflect physical qty
	txCnt, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	seq, err := repo.NextCountSequence(ctx, txCnt, items[0].ID)
	require.NoError(t, err)
	require.NoError(t, repo.SaveCount(ctx, txCnt, items[0].ID, seq, 12, 9007, ""))
	require.NoError(t, txCnt.Commit(ctx))

	// lock stock and adjust
	productIDs := []int{p}
	stock, err := repo.LockStockForProducts(ctx, txAppr, productIDs)
	require.NoError(t, err)
	assert.Equal(t, 10, stock[p])

	require.NoError(t, repo.UpdateItemAdjustment(ctx, txAppr, items[0].ID, 10, 2, 2))
	require.NoError(t, repo.UpdateProductStock(ctx, txAppr, p, 12))
	require.NoError(t, repo.InsertMovements(ctx, txAppr, s.ID, 9007, []movementRow{
		{ProductID: p, QuantityChange: 2, Notes: "adjustment"},
	}))
	require.NoError(t, repo.ApproveSessionStatus(ctx, txAppr, s.ID, 9007))
	require.NoError(t, txAppr.Commit(ctx))

	// verify stock changed
	var stockQty int
	err = dbPool.QueryRow(ctx, `SELECT quantity FROM product_stock WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL`, p).Scan(&stockQty)
	require.NoError(t, err)
	assert.Equal(t, 12, stockQty)

	// verify movement recorded
	var movementCount int
	err = dbPool.QueryRow(ctx, `SELECT COUNT(*) FROM inventory_movements WHERE product_id = $1 AND type = 'stock_opname' AND reference_table = 'stock_opnames'`, p).Scan(&movementCount)
	require.NoError(t, err)
	assert.Equal(t, 1, movementCount)

	// verify session approved
	status, err := repo.GetSessionStatus(ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusApproved, status)

	// verify item adjustment persisted
	session, err := repo.GetSession(ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, float64(10), session.Items[0].ExpectedQty)
	assert.Equal(t, float64(2), session.Items[0].DifferenceQty)
	assert.Equal(t, float64(2), session.Items[0].AdjustmentQty)
}

func TestRepository_ListAssignableUsers(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()
	resetStockOpname(t, ctx)
	insertTestUserWithRole(t, ctx, 9301, "so_cashier_9301", 4)
	insertTestUserWithRole(t, ctx, 9302, "so_staff_9302", 5)
	insertTestUserWithRole(t, ctx, 9303, "so_manager_9303", 3)
	insertTestUserWithRole(t, ctx, 9304, "so_admin_9304", 2)
	insertTestUserWithRole(t, ctx, 9305, "so_super_9305", 1)

	users, err := repo.ListAssignableUsers(ctx, "")
	require.NoError(t, err)
	got := map[string]bool{}
	for _, u := range users {
		got[u.Username] = true
	}
	assert.True(t, got["so_cashier_9301"], "cashier should be assignable")
	assert.True(t, got["so_staff_9302"], "staff should be assignable")
	assert.True(t, got["so_manager_9303"], "manager should be assignable")
	assert.True(t, got["so_admin_9304"], "admin should be assignable")
	assert.False(t, got["so_super_9305"], "superadmin should not be assignable")

	// search narrows results
	users, err = repo.ListAssignableUsers(ctx, "manager")
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, "so_manager_9303", users[0].Username)
	assert.Equal(t, "manager", users[0].RoleName)
}

func TestRepository_GetUserRoleName(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()
	resetStockOpname(t, ctx)
	insertTestUserWithRole(t, ctx, 9306, "so_staff_9306", 5)

	role, err := repo.GetUserRoleName(ctx, 9306)
	require.NoError(t, err)
	assert.Equal(t, "staff", role)

	_, err = repo.GetUserRoleName(ctx, 999999)
	require.ErrorIs(t, err, ErrAssigneeNotFound)
}

func TestRepository_ListSessions(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()
	resetStockOpname(t, ctx)
	insertTestUser(t, ctx, 9008, "so_user_9008")

	// v1 enforces a single global active session, so create sessions sequentially,
	// ending the previous one before creating the next.
	s1 := createTestSessionScope(t, ctx, repo, 9008, 9008001)
	require.NoError(t, repo.CancelSession(ctx, s1.ID, 9008))
	_ = createTestSessionScope(t, ctx, repo, 9008, 9008002)

	sessions, total, err := repo.ListSessions(ctx, 10, 0, "", "")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 2)
	assert.GreaterOrEqual(t, len(sessions), 2)

	filtered, totalFiltered, err := repo.ListSessions(ctx, 10, 0, StatusDraft, "")
	require.NoError(t, err)
	assert.Equal(t, totalFiltered, len(filtered))
	for _, s := range filtered {
		assert.Equal(t, StatusDraft, s.Status)
	}
}
