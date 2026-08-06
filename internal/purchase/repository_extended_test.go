package purchase

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_GetNextPONumber(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	num1, err := repo.GetNextPONumber(ctx)
	require.NoError(t, err)
	assert.Contains(t, num1, "PO-")

	num2, err := repo.GetNextPONumber(ctx)
	require.NoError(t, err)
	assert.Contains(t, num2, "PO-")
	assert.NotEqual(t, num1, num2, "PO numbers should be sequential and unique")
}

func TestRepository_GetNextGRNumber(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	num1, err := repo.GetNextGRNumber(ctx)
	require.NoError(t, err)
	assert.Contains(t, num1, "GR-")

	num2, err := repo.GetNextGRNumber(ctx)
	require.NoError(t, err)
	assert.Contains(t, num2, "GR-")
	assert.NotEqual(t, num1, num2, "GR numbers should be sequential and unique")
}

func TestRepository_DeletePurchaseOrder(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	supplierID := insertTestSupplier(ctx, t, "Repo Delete Supplier")
	prodID := insertTestProduct(ctx, t, "REPO-DEL", "Repo Delete", 10000, 100)
	userID := insertTestUser(ctx, t, "repo_del_user")

	po := &Order{SupplierID: supplierID, StoreID: 1, Status: StatusDraft, CreatedBy: userID, UpdatedBy: userID}
	items := []OrderItem{{ProductID: prodID, QtyOrdered: 3, UnitCost: 5000, ProductName: "Repo Delete", SKU: "REPO-DEL"}}

	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	po.PONumber, err = repo.GetNextPONumber(ctx)
	require.NoError(t, err)
	err = repo.CreatePurchaseOrder(ctx, tx, po, items)
	require.NoError(t, err)
	err = tx.Commit(ctx)
	require.NoError(t, err)

	tx, err = repo.BeginTx(ctx)
	require.NoError(t, err)
	err = repo.DeletePurchaseOrder(ctx, tx, po.ID)
	require.NoError(t, err)
	err = tx.Commit(ctx)
	require.NoError(t, err)

	_, err = repo.GetPurchaseOrderByID(ctx, po.ID, nil)
	assert.Error(t, err, "deleted purchase order should not be found")
}

func TestRepository_DeletePurchaseOrder_ConfirmedSilentlyNoops(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	supplierID := insertTestSupplier(ctx, t, "Repo Del Confirm Supplier")
	prodID := insertTestProduct(ctx, t, "REPO-DELCF", "Repo Del Confirm", 10000, 100)
	userID := insertTestUser(ctx, t, "repo_del_cf_user")

	po := &Order{SupplierID: supplierID, StoreID: 1, Status: StatusDraft, CreatedBy: userID, UpdatedBy: userID}
	items := []OrderItem{{ProductID: prodID, QtyOrdered: 3, UnitCost: 5000, ProductName: "Repo Del Confirm", SKU: "REPO-DELCF"}}

	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	po.PONumber, err = repo.GetNextPONumber(ctx)
	require.NoError(t, err)
	err = repo.CreatePurchaseOrder(ctx, tx, po, items)
	require.NoError(t, err)
	err = tx.Commit(ctx)
	require.NoError(t, err)

	tx, err = repo.BeginTx(ctx)
	require.NoError(t, err)
	err = repo.LockPurchaseOrderForUpdate(ctx, tx, po.ID)
	require.NoError(t, err)
	err = repo.ConfirmPurchaseOrder(ctx, tx, po.ID, userID, time.Now().Format(time.RFC3339))
	require.NoError(t, err)
	err = tx.Commit(ctx)
	require.NoError(t, err)

	tx, err = repo.BeginTx(ctx)
	require.NoError(t, err)
	err = repo.DeletePurchaseOrder(ctx, tx, po.ID)
	assert.NoError(t, err)
	err = tx.Commit(ctx)
	assert.NoError(t, err)

	existing, err := repo.GetPurchaseOrderByID(ctx, po.ID, nil)
	assert.NoError(t, err, "get confirmed PO after delete attempt should not error")
	assert.NotNil(t, existing, "confirmed PO should still exist after delete attempt")
	assert.Equal(t, StatusConfirmed, existing.Status)
}

func TestRepository_GetAllPurchaseOrders(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	supplierID := insertTestSupplier(ctx, t, "Repo List All Supplier")
	prodID := insertTestProduct(ctx, t, "REPO-LIST", "Repo List", 10000, 100)
	userID := insertTestUser(ctx, t, "repo_list_user")

	for i := 0; i < 5; i++ {
		po := &Order{SupplierID: supplierID, StoreID: 1, Status: StatusDraft, CreatedBy: userID, UpdatedBy: userID}
		items := []OrderItem{{ProductID: prodID, QtyOrdered: 1, UnitCost: 5000, ProductName: "Repo List", SKU: "REPO-LIST"}}
		tx, err := repo.BeginTx(ctx)
		require.NoError(t, err)
		po.PONumber, err = repo.GetNextPONumber(ctx)
		require.NoError(t, err)
		err = repo.CreatePurchaseOrder(ctx, tx, po, items)
		require.NoError(t, err)
		err = tx.Commit(ctx)
		require.NoError(t, err)
	}

	t.Run("returns all with pagination", func(t *testing.T) {
		pos, total, err := repo.GetAllPurchaseOrders(ctx, 10, 0, "", "", "", "", "", "", "", nil, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 5)
		assert.GreaterOrEqual(t, len(pos), 5)
	})

	t.Run("respects limit", func(t *testing.T) {
		pos, total, err := repo.GetAllPurchaseOrders(ctx, 2, 0, "", "", "", "", "", "", "", nil, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 5)
		assert.Len(t, pos, 2)
	})

	t.Run("respects offset", func(t *testing.T) {
		pos1, _, _ := repo.GetAllPurchaseOrders(ctx, 1, 0, "", "", "", "", "", "", "", nil, nil)
		pos2, _, _ := repo.GetAllPurchaseOrders(ctx, 1, 1, "", "", "", "", "", "", "", nil, nil)
		if len(pos1) > 0 && len(pos2) > 0 {
			assert.NotEqual(t, pos1[0].ID, pos2[0].ID)
		}
	})

	t.Run("filters by supplier", func(t *testing.T) {
		supplierIDStr := strconv.Itoa(supplierID)
		pos, total, err := repo.GetAllPurchaseOrders(ctx, 10, 0, "", "", "", "", supplierIDStr, "", "", nil, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 5)
		for _, po := range pos {
			assert.Equal(t, supplierID, po.SupplierID)
		}
	})

	t.Run("filters by status", func(t *testing.T) {
		pos, total, err := repo.GetAllPurchaseOrders(ctx, 10, 0, "", "", "", "draft", "", "", "", nil, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 5)
		for _, po := range pos {
			assert.Equal(t, "draft", po.Status)
		}
	})

	t.Run("returns empty for non-matching search", func(t *testing.T) {
		pos, total, err := repo.GetAllPurchaseOrders(ctx, 10, 0, "ZZZZNONEXISTENT", "", "", "", "", "", "", nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, pos)
	})

	t.Run("returns nil slice as empty when no results", func(t *testing.T) {
		pos, total, err := repo.GetAllPurchaseOrders(ctx, 10, 0, "ZZZZNONEXISTENT", "", "", "", "", "", "", nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, pos)
	})

	t.Run("sorts by po_number ascending", func(t *testing.T) {
		pos, _, err := repo.GetAllPurchaseOrders(ctx, 10, 0, "", "po_number", "asc", "", "", "", "", nil, nil)
		require.NoError(t, err)
		if len(pos) >= 2 {
			assert.LessOrEqual(t, pos[0].PONumber, pos[1].PONumber)
		}
	})

	t.Run("sorts by po_number descending", func(t *testing.T) {
		pos, _, err := repo.GetAllPurchaseOrders(ctx, 10, 0, "", "po_number", "desc", "", "", "", "", nil, nil)
		require.NoError(t, err)
		if len(pos) >= 2 {
			assert.GreaterOrEqual(t, pos[0].PONumber, pos[1].PONumber)
		}
	})
}

func TestRepository_LockPurchaseOrderForUpdate_NotFound(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	err = repo.LockPurchaseOrderForUpdate(ctx, tx, 999999)
	assert.Error(t, err)
}

func TestRepository_UpdatePOItemQtyReceived(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	supplierID := insertTestSupplier(ctx, t, "Repo Qty Supplier")
	prodID := insertTestProduct(ctx, t, "REPO-QTY", "Repo Qty", 10000, 100)
	userID := insertTestUser(ctx, t, "repo_qty_user")

	po := &Order{SupplierID: supplierID, StoreID: 1, Status: StatusDraft, CreatedBy: userID, UpdatedBy: userID}
	items := []OrderItem{{ProductID: prodID, QtyOrdered: 10, UnitCost: 5000, ProductName: "Repo Qty", SKU: "REPO-QTY"}}
	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	po.PONumber, err = repo.GetNextPONumber(ctx)
	require.NoError(t, err)
	err = repo.CreatePurchaseOrder(ctx, tx, po, items)
	require.NoError(t, err)
	err = tx.Commit(ctx)
	require.NoError(t, err)

	fetched, err := repo.GetPurchaseOrderByID(ctx, po.ID, nil)
	require.NoError(t, err)
	require.Len(t, fetched.Items, 1)

	tx, err = repo.BeginTx(ctx)
	require.NoError(t, err)
	err = repo.UpdatePOItemQtyReceived(ctx, tx, fetched.Items[0].ID, 5)
	require.NoError(t, err)
	err = repo.UpdatePOItemQtyReceived(ctx, tx, fetched.Items[0].ID, 3)
	require.NoError(t, err)
	err = tx.Commit(ctx)
	require.NoError(t, err)

	fetched, err = repo.GetPurchaseOrderByID(ctx, po.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, 8, fetched.Items[0].QtyReceived)
}

func TestRepository_ConfirmPurchaseOrder_WrongStatus(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	supplierID := insertTestSupplier(ctx, t, "Repo Confirm Wrong Supplier")
	prodID := insertTestProduct(ctx, t, "REPO-CONFWR", "Repo Confirm Wrong", 10000, 100)
	userID := insertTestUser(ctx, t, "repo_conf_wr_user")

	po := &Order{SupplierID: supplierID, StoreID: 1, Status: StatusDraft, CreatedBy: userID, UpdatedBy: userID}
	items := []OrderItem{{ProductID: prodID, QtyOrdered: 5, UnitCost: 8000, ProductName: "Repo Confirm Wrong", SKU: "REPO-CONFWR"}}
	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	po.PONumber, err = repo.GetNextPONumber(ctx)
	require.NoError(t, err)
	err = repo.CreatePurchaseOrder(ctx, tx, po, items)
	require.NoError(t, err)
	err = tx.Commit(ctx)
	require.NoError(t, err)

	tx, err = repo.BeginTx(ctx)
	require.NoError(t, err)
	err = repo.LockPurchaseOrderForUpdate(ctx, tx, po.ID)
	require.NoError(t, err)
	err = repo.ConfirmPurchaseOrder(ctx, tx, po.ID, userID, time.Now().Format(time.RFC3339))
	require.NoError(t, err)
	err = tx.Commit(ctx)
	require.NoError(t, err)

	tx, err = repo.BeginTx(ctx)
	require.NoError(t, err)
	err = repo.LockPurchaseOrderForUpdate(ctx, tx, po.ID)
	require.NoError(t, err)
	err = repo.ConfirmPurchaseOrder(ctx, tx, po.ID, userID, time.Now().Format(time.RFC3339))
	assert.ErrorIs(t, err, ErrPurchaseOrderAlreadyConfirmed)
	_ = tx.Rollback(ctx)
}

func TestRepository_GetReceiptsByPOID_WithStoreID(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	supplierID := insertTestSupplier(ctx, t, "Repo Receipts Store Supplier")
	prodID := insertTestProduct(ctx, t, "REPO-RCPTST", "Repo Receipts Store", 10000, 100)
	userID := insertTestUser(ctx, t, "repo_rcpt_st_user")

	po := &Order{SupplierID: supplierID, StoreID: 1, Status: StatusDraft, CreatedBy: userID, UpdatedBy: userID}
	items := []OrderItem{{ProductID: prodID, QtyOrdered: 10, UnitCost: 8000, ProductName: "Repo Receipts Store", SKU: "REPO-RCPTST"}}
	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	po.PONumber, err = repo.GetNextPONumber(ctx)
	require.NoError(t, err)
	err = repo.CreatePurchaseOrder(ctx, tx, po, items)
	require.NoError(t, err)
	err = tx.Commit(ctx)
	require.NoError(t, err)

	fetchedPO, err := repo.GetPurchaseOrderByID(ctx, po.ID, nil)
	require.NoError(t, err)

	storeID := 1
	receipts, err := repo.GetReceiptsByPOID(ctx, po.ID, &storeID)
	require.NoError(t, err)
	assert.Empty(t, receipts)

	storeIDWrong := 999
	_, err = repo.GetReceiptsByPOID(ctx, po.ID, &storeIDWrong)
	assert.Error(t, err)

	tx, err = repo.BeginTx(ctx)
	require.NoError(t, err)
	err = repo.LockPurchaseOrderForUpdate(ctx, tx, po.ID)
	require.NoError(t, err)
	err = repo.ConfirmPurchaseOrder(ctx, tx, po.ID, userID, time.Now().Format(time.RFC3339))
	require.NoError(t, err)
	err = tx.Commit(ctx)
	require.NoError(t, err)

	grNumber, err := repo.GetNextGRNumber(ctx)
	require.NoError(t, err)
	gr := &GoodsReceipt{
		GRNumber:        grNumber,
		PurchaseOrderID: po.ID,
		StoreID:         1,
		ReceivedBy:      userID,
	}
	grItems := []GoodsReceiptItem{{
		PurchaseOrderItemID: fetchedPO.Items[0].ID,
		ProductID:           prodID,
		QtyGood:             5,
		QtyDamaged:          1,
		UnitCost:            8000,
		ProductName:         "Repo Receipts Store",
	}}
	tx, err = repo.BeginTx(ctx)
	require.NoError(t, err)
	err = repo.LockPurchaseOrderForUpdate(ctx, tx, po.ID)
	require.NoError(t, err)
	err = repo.CreateGoodsReceipt(ctx, tx, gr, grItems)
	require.NoError(t, err)
	err = repo.UpdatePOItemQtyReceived(ctx, tx, fetchedPO.Items[0].ID, 6)
	require.NoError(t, err)
	err = repo.RecalculatePOStatus(ctx, tx, po.ID)
	require.NoError(t, err)
	err = tx.Commit(ctx)
	require.NoError(t, err)

	receipts, err = repo.GetReceiptsByPOID(ctx, po.ID, &storeID)
	require.NoError(t, err)
	require.Len(t, receipts, 1)
	assert.Equal(t, grNumber, receipts[0].GRNumber)
	require.Len(t, receipts[0].Items, 1)
	assert.Equal(t, 5, receipts[0].Items[0].QtyGood)
	assert.Equal(t, 1, receipts[0].Items[0].QtyDamaged)
}
