package salereturn

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

var dbPool *pgxpool.Pool

func TestMain(m *testing.M) {
	pool, err := shared.NewTestDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "test database: %v\n", err)
		os.Exit(1)
	}
	dbPool = pool

	if err := shared.RunMigrations(pool, "../../database/migrations"); err != nil {
		fmt.Fprintf(os.Stderr, "migrations: %v\n", err)
		os.Exit(1)
	}
	if err := shared.TruncateTestData(pool); err != nil {
		fmt.Fprintf(os.Stderr, "truncate: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()
	pool.Close()
	os.Exit(code)
}

// seeded holds the ids of the sale fixture a return hangs off.
type seeded struct {
	userID      int
	shiftID     int
	productID   int
	saleID      int
	itemID      int
	cashPM      int
	cardPM      int
	totalRefund int
}

func uniqueSuffix() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// seedSaleFixture inserts a user, open shift, product, completed sale with one
// line and a cash payment. TruncateTestData restores the baseline store (id 1)
// and payment_methods/roles survive it, so those come from the baseline.
func seedSaleFixture(t *testing.T) *seeded {
	t.Helper()
	ctx := context.Background()
	sfx := uniqueSuffix()

	var roleID int
	require.NoError(t, dbPool.QueryRow(ctx, `SELECT id FROM roles ORDER BY id LIMIT 1`).Scan(&roleID))

	f := &seeded{}
	require.NoError(t, dbPool.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, role_id)
		VALUES ($1, $2, 'x', $3) RETURNING id
	`, "salereturn_"+sfx, "salereturn_"+sfx+"@test.local", roleID).Scan(&f.userID))

	require.NoError(t, dbPool.QueryRow(ctx, `
		INSERT INTO shifts (user_id, status, opening_balance, opened_at)
		VALUES ($1, 'open', 0, NOW()) RETURNING id
	`, f.userID).Scan(&f.shiftID))

	require.NoError(t, dbPool.QueryRow(ctx, `
		INSERT INTO products (sku, name, price, cost, status)
		VALUES ($1, 'Kabel HDMI', 5000, 2500, 'active') RETURNING id
	`, "SR-"+sfx).Scan(&f.productID))

	f.cashPM = insertTestPaymentMethod(t, "CASH")
	f.cardPM = insertTestPaymentMethod(t, "CARD")

	// customer_id is NULL explicitly: customers are truncated and no FK row
	// exists. store_id is the baseline store restored by TruncateTestData.
	require.NoError(t, dbPool.QueryRow(ctx, `
		INSERT INTO sales (invoice_number, cashier_id, customer_id, shift_id, store_id,
		                   payment_method, status, subtotal, discount, tax, total_amount,
		                   change_due, created_at)
		VALUES ($1, $2, NULL, $3, 1, 'Cash', 'completed', 15000, 0, 1486, 15000, 500, NOW())
		RETURNING id
	`, "SR-INV-"+sfx, f.userID, f.shiftID).Scan(&f.saleID))

	require.NoError(t, dbPool.QueryRow(ctx, `
		INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal,
		                        dpp_amount, tax_amount, tax_rate, product_name)
		VALUES ($1, $2, 3, 5000, 15000, 13514, 1486, 11, 'Kabel HDMI')
		RETURNING id
	`, f.saleID, f.productID).Scan(&f.itemID))

	_, err := dbPool.Exec(ctx, `
		INSERT INTO sale_payments (sale_id, payment_method_id, payment_method_code, amount)
		VALUES ($1, $2, 'CASH', 15500)
	`, f.saleID, f.cashPM)
	require.NoError(t, err)

	f.totalRefund = 10000
	return f
}

func insertTestPaymentMethod(t *testing.T, code string) int {
	t.Helper()
	var id int
	require.NoError(t, dbPool.QueryRow(context.Background(), `
		INSERT INTO payment_methods (code, name, is_active, requires_reference)
		VALUES ($1, $2, true, false)
		ON CONFLICT (code) DO UPDATE SET is_active = true
		RETURNING id
	`, code, code).Scan(&id))
	return id
}

func mustInsertReturn(ctx context.Context, t *testing.T, repo *Repository, f *seeded, status string, expiresAt *time.Time) *Return {
	t.Helper()
	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	num, err := repo.NextReturnNumber(ctx, tx)
	require.NoError(t, err)
	ret := &Return{
		ReturnNumber:      num,
		OriginalSaleID:    f.saleID,
		CashierID:         f.userID,
		StoreID:           1,
		ShiftID:           f.shiftID,
		ReturnType:        ReturnTypeRefund,
		Reason:            "barang rusak",
		Condition:         ConditionDamaged,
		Subtotal:          10000,
		TaxReversal:       909,
		TotalRefundAmount: 10000,
		Status:            status,
		ExpiresAt:         expiresAt,
	}
	require.NoError(t, repo.InsertReturn(ctx, tx, ret))
	require.NoError(t, tx.Commit(ctx))
	return ret
}

// insertReturnWithItem additionally attaches one return item so the returned
// quantity guard has rows to count.
func insertReturnWithItem(ctx context.Context, t *testing.T, repo *Repository, f *seeded, status string, expiresAt *time.Time) *Return {
	t.Helper()
	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	num, err := repo.NextReturnNumber(ctx, tx)
	require.NoError(t, err)
	ret := &Return{
		ReturnNumber:      num,
		OriginalSaleID:    f.saleID,
		CashierID:         f.userID,
		StoreID:           1,
		ShiftID:           f.shiftID,
		ReturnType:        ReturnTypeRefund,
		Reason:            "barang rusak",
		Condition:         ConditionDamaged,
		Subtotal:          f.totalRefund,
		TaxReversal:       909,
		TotalRefundAmount: f.totalRefund,
		Status:            status,
		ExpiresAt:         expiresAt,
	}
	require.NoError(t, repo.InsertReturn(ctx, tx, ret))
	items := []ReturnItem{{
		SaleItemID: f.itemID, ProductID: f.productID, ProductName: "Kabel HDMI",
		Quantity: 1, UnitPrice: f.totalRefund, Subtotal: f.totalRefund,
		DppAmount: 9091, TaxAmount: 909, TaxRate: 11,
		Condition: ConditionDamaged, Outcome: OutcomeWrittenOff,
	}}
	require.NoError(t, repo.InsertItems(ctx, tx, ret.ID, items))
	require.NoError(t, tx.Commit(ctx))
	return ret
}

func ptrInt(v int) *int { return &v }

func TestRepository_NextReturnNumber(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	repo := NewRepository(dbPool)

	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	n1, err := repo.NextReturnNumber(ctx, tx)
	require.NoError(t, err)
	n2, err := repo.NextReturnNumber(ctx, tx)
	require.NoError(t, err)

	var y1, s1, y2, s2 int
	_, err = fmt.Sscanf(n1, "RET-%d-%d", &y1, &s1)
	require.NoError(t, err, "format RET-<year>-<seq>: %q", n1)
	_, err = fmt.Sscanf(n2, "RET-%d-%d", &y2, &s2)
	require.NoError(t, err, "format RET-<year>-<seq>: %q", n2)

	assert.Equal(t, time.Now().In(shared.JakartaLocation()).Year(), y1, "year is minted in Jakarta time")
	assert.Equal(t, y1, y2)
	assert.Equal(t, s1+1, s2, "the sequence advances per mint")
}

func TestRepository_RoundTrip(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	repo := NewRepository(dbPool)
	f := seedSaleFixture(t)

	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	num, err := repo.NextReturnNumber(ctx, tx)
	require.NoError(t, err)
	ret := &Return{
		ReturnNumber:      num,
		OriginalSaleID:    f.saleID,
		CashierID:         f.userID,
		StoreID:           1,
		ShiftID:           f.shiftID,
		ReturnType:        ReturnTypeRefund,
		Reason:            "barang rusak",
		Condition:         ConditionDamaged,
		Subtotal:          10000,
		TaxReversal:       909,
		TotalRefundAmount: 10000,
		Status:            ReturnStatusCompleted,
	}
	require.NoError(t, repo.InsertReturn(ctx, tx, ret))
	require.Positive(t, ret.ID)
	require.False(t, ret.CreatedAt.IsZero())

	items := []ReturnItem{{
		SaleItemID: f.itemID, ProductID: f.productID, ProductName: "Kabel HDMI",
		Quantity: 1, UnitPrice: 10000, Subtotal: 10000,
		DppAmount: 9091, TaxAmount: 909, TaxRate: 11,
		Condition: ConditionDamaged, Outcome: OutcomeWrittenOff,
		Replacements: []Replacement{{ProductID: f.productID, Quantity: 1}},
	}}
	require.NoError(t, repo.InsertItems(ctx, tx, ret.ID, items))
	require.Positive(t, items[0].ID)
	require.Positive(t, items[0].Replacements[0].ID)
	require.Equal(t, items[0].ID, items[0].Replacements[0].ItemID)

	require.NoError(t, repo.InsertPayments(ctx, tx, ret.ID, []ReturnPayment{
		{PaymentMethodID: f.cashPM, PaymentMethodCode: "CASH", Amount: 10000},
	}))
	require.NoError(t, tx.Commit(ctx))

	// Read back with and without a store scope.
	loaded, err := repo.GetReturn(ctx, ptrInt(1), ret.ID)
	require.NoError(t, err)
	assert.Equal(t, num, loaded.ReturnNumber)
	assert.Equal(t, ReturnStatusCompleted, loaded.Status)
	require.Len(t, loaded.Items, 1)
	assert.Equal(t, 1, loaded.Items[0].Quantity)
	require.Len(t, loaded.Items[0].Replacements, 1)
	assert.Equal(t, 1, loaded.Items[0].Replacements[0].Quantity)
	require.Len(t, loaded.Payments, 1)
	assert.Equal(t, 10000, loaded.Payments[0].Amount)
	assert.Nil(t, loaded.Approval)

	scoped, err := repo.GetReturn(ctx, ptrInt(2), ret.ID)
	assert.ErrorIs(t, err, ErrReturnNotFound, "another store must not see the return")
	assert.Nil(t, scoped)

	missing, err := repo.GetReturn(ctx, nil, 99999999)
	assert.ErrorIs(t, err, ErrReturnNotFound)
	assert.Nil(t, missing)

	// GetReturnForUpdate must load the proposal items inside the transaction:
	// ApproveReturn re-plans from ret.Items (regression guard).
	tx2, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback(ctx) }()
	locked, err := repo.GetReturnForUpdate(ctx, tx2, ret.ID)
	require.NoError(t, err)
	require.Len(t, locked.Items, 1, "items must be present on the in-tx read")
	assert.Equal(t, f.itemID, locked.Items[0].SaleItemID)
}

func TestRepository_PendingApprovalLifecycle(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	repo := NewRepository(dbPool)
	f := seedSaleFixture(t)
	future := time.Now().Add(5 * time.Minute)

	// Create: header pending + approval row.
	active := mustInsertReturn(ctx, t, repo, f, ReturnStatusPendingApproval, &future)
	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	reason := "barang rusak berat"
	require.NoError(t, repo.InsertApproval(ctx, tx, active.ID, f.userID, &reason, future))
	require.NoError(t, tx.Commit(ctx))

	loaded, err := repo.GetReturn(ctx, nil, active.ID)
	require.NoError(t, err)
	require.NotNil(t, loaded.Approval)
	assert.Equal(t, "pending", loaded.Approval.Status)
	require.NotNil(t, loaded.Approval.Reason)
	assert.Equal(t, reason, *loaded.Approval.Reason)

	// Approve.
	tx2, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	require.NoError(t, repo.ApproveApproval(ctx, tx2, active.ID, f.userID, nil))
	require.NoError(t, tx2.Commit(ctx))
	loaded, err = repo.GetReturn(ctx, nil, active.ID)
	require.NoError(t, err)
	require.NotNil(t, loaded.Approval)
	assert.Equal(t, "approved", loaded.Approval.Status)
	require.NotNil(t, loaded.Approval.ApprovedBy)
	assert.Equal(t, f.userID, *loaded.Approval.ApprovedBy)

	// Reject cancels the header.
	rejected := mustInsertReturn(ctx, t, repo, f, ReturnStatusPendingApproval, &future)
	tx3, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	rej := "di luar kebijakan"
	require.NoError(t, repo.InsertApproval(ctx, tx3, rejected.ID, f.userID, nil, future))
	require.NoError(t, repo.RejectApproval(ctx, tx3, rejected.ID, f.userID, rej))
	require.NoError(t, tx3.Commit(ctx))
	loaded, err = repo.GetReturn(ctx, nil, rejected.ID)
	require.NoError(t, err)
	assert.Equal(t, ReturnStatusCancelled, loaded.Status)
	require.NotNil(t, loaded.Approval)
	assert.Equal(t, "rejected", loaded.Approval.Status)
	require.NotNil(t, loaded.Approval.RejectedReason)
	assert.Equal(t, rej, *loaded.Approval.RejectedReason)

	// Expire flips both rows.
	stale := mustInsertReturn(ctx, t, repo, f, ReturnStatusPendingApproval, &future)
	tx4, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	require.NoError(t, repo.InsertApproval(ctx, tx4, stale.ID, f.userID, nil, future))
	require.NoError(t, repo.ExpireReturn(ctx, tx4, stale.ID))
	require.NoError(t, tx4.Commit(ctx))
	loaded, err = repo.GetReturn(ctx, nil, stale.ID)
	require.NoError(t, err)
	assert.Equal(t, ReturnStatusExpired, loaded.Status)
	require.NotNil(t, loaded.Approval)
	assert.Equal(t, "expired", loaded.Approval.Status)
}

func TestRepository_ListPending_HidesExpiredAndDecided(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	repo := NewRepository(dbPool)
	f := seedSaleFixture(t)

	future := time.Now().Add(5 * time.Minute)
	past := time.Now().Add(-time.Minute)

	active := mustInsertReturn(ctx, t, repo, f, ReturnStatusPendingApproval, &future)
	_ = mustInsertReturn(ctx, t, repo, f, ReturnStatusPendingApproval, &past) // elapsed window
	_ = mustInsertReturn(ctx, t, repo, f, ReturnStatusCompleted, &future)

	returns, total, err := repo.ListPending(ctx, nil, 20, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total, "only the pending return still inside its window is listable")
	require.Len(t, returns, 1)
	assert.Equal(t, active.ID, returns[0].ID)
}

func TestRepository_ReturnedQtyBySale_CountsOnlyBindingReturns(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	repo := NewRepository(dbPool)
	f := seedSaleFixture(t)

	future := time.Now().Add(5 * time.Minute)
	past := time.Now().Add(-time.Minute)

	_ = insertReturnWithItem(ctx, t, repo, f, ReturnStatusCompleted, nil)
	_ = insertReturnWithItem(ctx, t, repo, f, ReturnStatusCancelled, &future)
	_ = insertReturnWithItem(ctx, t, repo, f, ReturnStatusPendingApproval, &past) // expired: freed again
	_ = insertReturnWithItem(ctx, t, repo, f, ReturnStatusExpired, nil)           // flipped by a past approval attempt: freed too
	active := insertReturnWithItem(ctx, t, repo, f, ReturnStatusPendingApproval, &future)

	returned, err := repo.ReturnedQtyBySale(ctx, dbPool, f.saleID, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, returned[f.itemID], "completed + pending-active count; cancelled and expired do not")

	returned, err = repo.ReturnedQtyBySale(ctx, dbPool, f.saleID, active.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, returned[f.itemID], "excluding the return being approved removes its own rows")
}

func TestRepository_ReturnTotalsForShift(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	repo := NewRepository(dbPool)
	f := seedSaleFixture(t)

	// The unique open-shift-per-user constraint means the comparison shift
	// must belong to a different user.
	var otherRoleID int
	require.NoError(t, dbPool.QueryRow(ctx, `SELECT id FROM roles ORDER BY id LIMIT 1`).Scan(&otherRoleID))
	var otherUserID int
	require.NoError(t, dbPool.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, role_id)
		VALUES ($1, $2, 'x', $3) RETURNING id
	`, "salereturn_other_"+uniqueSuffix(), "salereturn_other_"+uniqueSuffix()+"@test.local", otherRoleID).Scan(&otherUserID))
	var otherShiftID int
	require.NoError(t, dbPool.QueryRow(ctx, `
		INSERT INTO shifts (user_id, status, opening_balance, opened_at)
		VALUES ($1, 'open', 0, NOW()) RETURNING id
	`, otherUserID).Scan(&otherShiftID))

	completed := insertReturnWithItem(ctx, t, repo, f, ReturnStatusCompleted, nil)
	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	require.NoError(t, repo.InsertPayments(ctx, tx, completed.ID, []ReturnPayment{
		{PaymentMethodID: f.cashPM, PaymentMethodCode: "CASH", Amount: 7000},
		{PaymentMethodID: f.cardPM, PaymentMethodCode: "CARD", Amount: 3000},
	}))
	require.NoError(t, tx.Commit(ctx))

	// A pending return must not touch shift totals yet.
	soon := time.Now().Add(time.Minute)
	_ = insertReturnWithItem(ctx, t, repo, f, ReturnStatusPendingApproval, &soon)

	// A completed return on another shift must not leak into this one.
	other := &seeded{userID: f.userID, shiftID: otherShiftID, saleID: f.saleID, itemID: f.itemID, productID: f.productID, cashPM: f.cashPM, totalRefund: 1000}
	otherRet := insertReturnWithItem(ctx, t, repo, other, ReturnStatusCompleted, nil)
	tx2, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	require.NoError(t, repo.InsertPayments(ctx, tx2, otherRet.ID, []ReturnPayment{
		{PaymentMethodID: f.cashPM, PaymentMethodCode: "CASH", Amount: 1000},
	}))
	require.NoError(t, tx2.Commit(ctx))

	totals, err := repo.ReturnTotalsForShift(ctx, dbPool, f.shiftID)
	require.NoError(t, err)
	assert.Equal(t, 7000, totals.CashRefund)
	assert.Equal(t, 3000, totals.NonCashRefund)
	assert.Equal(t, 10000, totals.TotalRefund)

	otherTotals, err := repo.ReturnTotalsForShift(ctx, dbPool, otherShiftID)
	require.NoError(t, err)
	assert.Equal(t, 1000, otherTotals.CashRefund)
	assert.Equal(t, 0, otherTotals.NonCashRefund)
	assert.Equal(t, 1000, otherTotals.TotalRefund)
}
