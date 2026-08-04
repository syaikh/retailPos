package purchase

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
		os.Exit(1)
	}
	dbPool = pool
	defer pool.Close()

	if err := shared.RunMigrations(pool, "../../database/migrations"); err != nil {
		os.Exit(1)
	}

	if err := shared.TruncateTestData(pool); err != nil {
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func insertTestSupplier(t *testing.T, ctx context.Context, name string) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(ctx, `
		INSERT INTO suppliers (name, code, is_active)
		VALUES ($1, $2, true)
		RETURNING id
	`, name, name).Scan(&id)
	require.NoError(t, err)
	return id
}

func insertTestProduct(t *testing.T, ctx context.Context, sku, name string, price, stock int) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(ctx, `
		INSERT INTO products (sku, name, price, cost, status)
		VALUES ($1, $2, $3, $4, 'active')
		RETURNING id
	`, sku, name, price, price/2).Scan(&id)
	require.NoError(t, err)
	_, err = dbPool.Exec(ctx, `
		INSERT INTO product_stock (product_id, quantity)
		VALUES ($1, $2)
	`, id, stock)
	require.NoError(t, err)
	return id
}

func insertTestUser(t *testing.T, ctx context.Context, username string) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, role_id)
		VALUES ($1, $2, 'hash', 1)
		ON CONFLICT (username) DO UPDATE SET email = excluded.email
		RETURNING id
	`, username, username+"@test.com").Scan(&id)
	require.NoError(t, err)
	return id
}

func TestPurchaseRepository_CreateAndGetPO(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	supplierID := insertTestSupplier(t, ctx, "Test Supplier PO")
	prodID := insertTestProduct(t, ctx, "PO-REPO-001", "PO Repo Product", 10000, 100)
	userID := insertTestUser(t, ctx, "po_repo_user")

	po := &PurchaseOrder{
		SupplierID: supplierID,
		StoreID:    1,
		Status:     StatusDraft,
		CreatedBy:  userID,
		UpdatedBy:  userID,
	}
	items := []PurchaseOrderItem{
		{
			ProductID:   prodID,
			QtyOrdered:  10,
			UnitCost:    8000,
			ProductName: "PO Repo Product",
			SKU:         "PO-REPO-001",
		},
	}

	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)

	poNumber, err := repo.GetNextPONumber(ctx)
	require.NoError(t, err)
	po.PONumber = poNumber

	err = repo.CreatePurchaseOrder(ctx, tx, po, items)
	require.NoError(t, err)
	err = tx.Commit(ctx)
	require.NoError(t, err)

	assert.Greater(t, po.ID, 0)
	assert.Equal(t, "draft", po.Status)

	fetched, err := repo.GetPurchaseOrderByID(ctx, po.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, po.PONumber, fetched.PONumber)
	assert.Equal(t, supplierID, fetched.SupplierID)
	assert.Len(t, fetched.Items, 1)
	assert.Equal(t, prodID, fetched.Items[0].ProductID)
	assert.Equal(t, "PO Repo Product", fetched.Items[0].ProductName)
	assert.Equal(t, "PO-REPO-001", fetched.Items[0].SKU)
}

func TestPurchaseRepository_ConfirmAndCancel(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	supplierID := insertTestSupplier(t, ctx, "Test Supplier Confirm")
	prodID := insertTestProduct(t, ctx, "PO-CONF-001", "PO Confirm Product", 10000, 100)
	userID := insertTestUser(t, ctx, "po_confirm_user")

	po := &PurchaseOrder{
		SupplierID: supplierID,
		StoreID:    1,
		Status:     StatusDraft,
		CreatedBy:  userID,
		UpdatedBy:  userID,
	}
	items := []PurchaseOrderItem{
		{
			ProductID:   prodID,
			QtyOrdered:  5,
			UnitCost:    8000,
			ProductName: "PO Confirm Product",
			SKU:         "PO-CONF-001",
		},
	}

	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	poNumber, err := repo.GetNextPONumber(ctx)
	require.NoError(t, err)
	po.PONumber = poNumber
	err = repo.CreatePurchaseOrder(ctx, tx, po, items)
	require.NoError(t, err)
	err = tx.Commit(ctx)
	require.NoError(t, err)

	tx, err = repo.BeginTx(ctx)
	require.NoError(t, err)
	err = repo.LockPurchaseOrderForUpdate(ctx, tx, po.ID)
	require.NoError(t, err)
	err = repo.ConfirmPurchaseOrder(ctx, tx, po.ID, userID, "2026-07-27T10:00:00+07:00")
	require.NoError(t, err)
	err = tx.Commit(ctx)
	require.NoError(t, err)

	fetched, err := repo.GetPurchaseOrderByID(ctx, po.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, StatusConfirmed, fetched.Status)
	assert.NotNil(t, fetched.ConfirmedBy)
	assert.Equal(t, userID, *fetched.ConfirmedBy)

	tx, err = repo.BeginTx(ctx)
	require.NoError(t, err)
	err = repo.LockPurchaseOrderForUpdate(ctx, tx, po.ID)
	require.NoError(t, err)
	err = repo.CancelPurchaseOrder(ctx, tx, po.ID, userID, "2026-07-27T11:00:00+07:00")
	require.NoError(t, err)
	err = tx.Commit(ctx)
	require.NoError(t, err)

	fetched, err = repo.GetPurchaseOrderByID(ctx, po.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, StatusCancelled, fetched.Status)
}

func TestPurchaseRepository_GoodsReceipt(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	supplierID := insertTestSupplier(t, ctx, "Test Supplier GR")
	prodID := insertTestProduct(t, ctx, "PO-GR-001", "PO GR Product", 10000, 100)
	userID := insertTestUser(t, ctx, "po_gr_user")

	po := &PurchaseOrder{
		SupplierID: supplierID,
		StoreID:    1,
		Status:     StatusDraft,
		CreatedBy:  userID,
		UpdatedBy:  userID,
	}
	items := []PurchaseOrderItem{
		{
			ProductID:   prodID,
			QtyOrdered:  20,
			UnitCost:    8000,
			ProductName: "PO GR Product",
			SKU:         "PO-GR-001",
		},
	}

	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	poNumber, err := repo.GetNextPONumber(ctx)
	require.NoError(t, err)
	po.PONumber = poNumber
	err = repo.CreatePurchaseOrder(ctx, tx, po, items)
	require.NoError(t, err)
	err = tx.Commit(ctx)
	require.NoError(t, err)

	fetchedPO, err := repo.GetPurchaseOrderByID(ctx, po.ID, nil)
	require.NoError(t, err)
	require.Len(t, fetchedPO.Items, 1)

	tx, err = repo.BeginTx(ctx)
	require.NoError(t, err)
	err = repo.LockPurchaseOrderForUpdate(ctx, tx, po.ID)
	require.NoError(t, err)
	err = repo.ConfirmPurchaseOrder(ctx, tx, po.ID, userID, "2026-07-27T10:00:00+07:00")
	require.NoError(t, err)
	err = tx.Commit(ctx)
	require.NoError(t, err)

	grNumber, err := repo.GetNextGRNumber(ctx)
	require.NoError(t, err)
	gr := &GoodsReceipt{
		GRNumber:            grNumber,
		PurchaseOrderID:     po.ID,
		StoreID:             1,
		ReceivedBy:          userID,
		DeliveryOrderNumber: "DO-001",
	}
	grItems := []GoodsReceiptItem{
		{
			PurchaseOrderItemID: fetchedPO.Items[0].ID,
			ProductID:           prodID,
			QtyGood:             10,
			QtyDamaged:          2,
			UnitCost:            8000,
			ProductName:         "PO GR Product",
		},
	}

	tx, err = repo.BeginTx(ctx)
	require.NoError(t, err)
	err = repo.LockPurchaseOrderForUpdate(ctx, tx, po.ID)
	require.NoError(t, err)
	err = repo.CreateGoodsReceipt(ctx, tx, gr, grItems)
	require.NoError(t, err)
	err = repo.UpdatePOItemQtyReceived(ctx, tx, fetchedPO.Items[0].ID, 12)
	require.NoError(t, err)
	err = repo.RecalculatePOStatus(ctx, tx, po.ID)
	require.NoError(t, err)
	err = tx.Commit(ctx)
	require.NoError(t, err)

	fetched, err := repo.GetPurchaseOrderByID(ctx, po.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, StatusPartialReceived, fetched.Status)
	assert.Equal(t, 12, fetched.Items[0].QtyReceived)

	receipts, err := repo.GetReceiptsByPOID(ctx, po.ID, nil)
	require.NoError(t, err)
	assert.Len(t, receipts, 1)
	assert.Equal(t, "DO-001", receipts[0].DeliveryOrderNumber)
	assert.Len(t, receipts[0].Items, 1)
	assert.Equal(t, 10, receipts[0].Items[0].QtyGood)
	assert.Equal(t, 2, receipts[0].Items[0].QtyDamaged)
}
