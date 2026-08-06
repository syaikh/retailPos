package purchase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPurchaseService_CreateDraft_ValidationBranches(t *testing.T) {
	svc, _, ctx := newSvc(t)
	item := []OrderItem{{ProductID: 1, QtyOrdered: 1, UnitCost: 100}}

	err := svc.CreateDraft(ctx, &Order{SupplierID: 1, StoreID: 1}, nil)
	assert.ErrorContains(t, err, "items cannot be empty")
	assert.ErrorIs(t, err, ErrInvalidInput)

	err = svc.CreateDraft(ctx, &Order{StoreID: 1}, item)
	assert.ErrorContains(t, err, "supplier_id is required")
	assert.ErrorIs(t, err, ErrInvalidInput)

	err = svc.CreateDraft(ctx, &Order{SupplierID: 1}, item)
	assert.ErrorContains(t, err, "store_id is required")
	assert.ErrorIs(t, err, ErrInvalidInput)

	err = svc.CreateDraft(ctx, &Order{SupplierID: 1, StoreID: 1}, []OrderItem{{QtyOrdered: 1, UnitCost: 100}})
	assert.ErrorContains(t, err, "product_id is required")
	assert.ErrorIs(t, err, ErrInvalidInput)

	err = svc.CreateDraft(ctx, &Order{SupplierID: 1, StoreID: 1}, []OrderItem{{ProductID: 1, QtyOrdered: 0, UnitCost: 100}})
	assert.ErrorContains(t, err, "qty_ordered must be greater than 0")
	assert.ErrorIs(t, err, ErrInvalidInput)

	err = svc.CreateDraft(ctx, &Order{SupplierID: 1, StoreID: 1}, []OrderItem{{ProductID: 1, QtyOrdered: 1, UnitCost: -5}})
	assert.ErrorContains(t, err, "unit_cost cannot be negative")
	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestPurchaseService_CreateDraft_Clamps(t *testing.T) {
	svc, _, ctx := newSvc(t)
	supplierID := insertTestSupplier(ctx, t, "Svc Clamp Supplier")
	prodID := insertTestProduct(ctx, t, "SVC-CLAMP-001", "Clamp Product", 10000, 100)

	po := &Order{
		SupplierID:     supplierID,
		StoreID:        1,
		DiscountAmount: 500000,
		TaxAmount:      1000,
	}
	items := []OrderItem{
		{ProductID: prodID, QtyOrdered: 1, UnitCost: 10000, DiscountAmount: 20000},
	}

	err := svc.CreateDraft(ctx, po, items)
	require.NoError(t, err)
	assert.Equal(t, 0, po.Subtotal)
	assert.Equal(t, 0, po.GrandTotal)
	assert.Equal(t, 0, items[0].Subtotal)
}

func TestPurchaseService_UpdateDraft_ValidationBranches(t *testing.T) {
	svc, _, ctx := newSvc(t)
	supplierID := insertTestSupplier(ctx, t, "Svc UpdVal Supplier")
	prodID := insertTestProduct(ctx, t, "SVC-UPVAL-001", "UpdVal Product", 10000, 100)
	userID := insertTestUser(ctx, t, "po_upval_user")

	createPO := func() int {
		po := &Order{SupplierID: supplierID, StoreID: 1, CreatedBy: userID, UpdatedBy: userID}
		items := []OrderItem{{ProductID: prodID, QtyOrdered: 5, UnitCost: 8000}}
		err := svc.CreateDraft(ctx, po, items)
		require.NoError(t, err)
		return po.ID
	}

	err := svc.UpdateDraft(ctx, createPO(), &Order{SupplierID: 1, UpdatedBy: 1}, nil)
	assert.ErrorContains(t, err, "items cannot be empty")
	assert.ErrorIs(t, err, ErrInvalidInput)

	err = svc.UpdateDraft(ctx, createPO(), &Order{UpdatedBy: 1}, []OrderItem{{ProductID: 1, QtyOrdered: 1, UnitCost: 100}})
	assert.ErrorContains(t, err, "supplier_id is required")
	assert.ErrorIs(t, err, ErrInvalidInput)

	err = svc.UpdateDraft(ctx, createPO(), &Order{SupplierID: 1}, []OrderItem{{ProductID: 1, QtyOrdered: 1, UnitCost: 100}})
	assert.ErrorContains(t, err, "updated_by is required")
	assert.ErrorIs(t, err, ErrInvalidInput)

	err = svc.UpdateDraft(ctx, createPO(), &Order{SupplierID: 1, UpdatedBy: 1}, []OrderItem{{QtyOrdered: 1, UnitCost: 100}})
	assert.ErrorContains(t, err, "product_id is required")
	assert.ErrorIs(t, err, ErrInvalidInput)

	err = svc.UpdateDraft(ctx, createPO(), &Order{SupplierID: 1, UpdatedBy: 1}, []OrderItem{{ProductID: 1, QtyOrdered: 0, UnitCost: 100}})
	assert.ErrorContains(t, err, "qty_ordered must be greater than 0")
	assert.ErrorIs(t, err, ErrInvalidInput)

	err = svc.UpdateDraft(ctx, createPO(), &Order{SupplierID: 1, UpdatedBy: 1}, []OrderItem{{ProductID: 1, QtyOrdered: 1, UnitCost: -5}})
	assert.ErrorContains(t, err, "unit_cost cannot be negative")
	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestPurchaseService_UpdateDraft_Clamps(t *testing.T) {
	svc, _, ctx := newSvc(t)
	supplierID := insertTestSupplier(ctx, t, "Svc UpdClamp Supplier")
	prodID := insertTestProduct(ctx, t, "SVC-UPC-001", "UpdClamp Product", 10000, 100)
	userID := insertTestUser(ctx, t, "po_upclamp_user")

	po := &Order{SupplierID: supplierID, StoreID: 1, CreatedBy: userID, UpdatedBy: userID}
	items := []OrderItem{{ProductID: prodID, QtyOrdered: 5, UnitCost: 8000}}
	err := svc.CreateDraft(ctx, po, items)
	require.NoError(t, err)

	update := &Order{SupplierID: supplierID, UpdatedBy: userID, DiscountAmount: 500000}
	updItems := []OrderItem{{ProductID: prodID, QtyOrdered: 5, UnitCost: 8000, DiscountAmount: 100000}}
	err = svc.UpdateDraft(ctx, po.ID, update, updItems)
	require.NoError(t, err)

	updated, err := svc.GetDetail(ctx, po.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, updated.Subtotal)
	assert.Equal(t, 0, updated.GrandTotal)
}

func TestPurchaseService_UpdateDraft_NotFound(t *testing.T) {
	svc, _, ctx := newSvc(t)
	item := []OrderItem{{ProductID: 1, QtyOrdered: 1, UnitCost: 100}}

	err := svc.UpdateDraft(ctx, 999999, &Order{SupplierID: 1, UpdatedBy: 1}, item)
	assert.ErrorIs(t, err, ErrPurchaseOrderNotFound)
}

func TestPurchaseService_UpdateDraft_NotDraft(t *testing.T) {
	svc, _, ctx := newSvc(t)
	supplierID := insertTestSupplier(ctx, t, "Svc UpdNotDraft Supplier")
	prodID := insertTestProduct(ctx, t, "SVC-UPDND-001", "UpdNotDraft Product", 10000, 100)
	userID := insertTestUser(ctx, t, "po_updnd_user")

	po := &Order{SupplierID: supplierID, StoreID: 1, CreatedBy: userID, UpdatedBy: userID}
	items := []OrderItem{{ProductID: prodID, QtyOrdered: 5, UnitCost: 8000}}
	err := svc.CreateDraft(ctx, po, items)
	require.NoError(t, err)

	err = svc.Confirm(ctx, po.ID, userID)
	require.NoError(t, err)

	_, err = svc.GetDetail(ctx, po.ID, nil)
	require.NoError(t, err)

	err = svc.UpdateDraft(ctx, po.ID, &Order{SupplierID: supplierID, UpdatedBy: userID}, items)
	assert.ErrorIs(t, err, ErrPurchaseOrderNotDraft)
}

func TestPurchaseService_DeleteDraft_NotDraft(t *testing.T) {
	svc, _, ctx := newSvc(t)
	supplierID := insertTestSupplier(ctx, t, "Svc DelNotDraft Supplier")
	prodID := insertTestProduct(ctx, t, "SVC-DELND-001", "DelNotDraft Product", 10000, 100)
	userID := insertTestUser(ctx, t, "po_delnd_user")

	po := &Order{SupplierID: supplierID, StoreID: 1, CreatedBy: userID, UpdatedBy: userID}
	items := []OrderItem{{ProductID: prodID, QtyOrdered: 5, UnitCost: 8000}}
	err := svc.CreateDraft(ctx, po, items)
	require.NoError(t, err)

	err = svc.Confirm(ctx, po.ID, userID)
	require.NoError(t, err)

	err = svc.DeleteDraft(ctx, po.ID)
	assert.ErrorIs(t, err, ErrPurchaseOrderNotDraft)
}

func TestPurchaseService_Confirm_NotFound(t *testing.T) {
	svc, _, ctx := newSvc(t)
	err := svc.Confirm(ctx, 999999, 1)
	assert.Error(t, err)
}

func TestPurchaseService_Cancel_NotFound(t *testing.T) {
	svc, _, ctx := newSvc(t)
	err := svc.Cancel(ctx, 999999, 1)
	assert.Error(t, err)
}

func TestPurchaseService_Cancel_AlreadyCancelled(t *testing.T) {
	svc, _, ctx := newSvc(t)
	supplierID := insertTestSupplier(ctx, t, "Svc CancelX2 Supplier")
	prodID := insertTestProduct(ctx, t, "SVC-CANC2-001", "CancelX2 Product", 10000, 100)
	userID := insertTestUser(ctx, t, "po_canc2_user")

	po := &Order{SupplierID: supplierID, StoreID: 1, CreatedBy: userID, UpdatedBy: userID}
	items := []OrderItem{{ProductID: prodID, QtyOrdered: 5, UnitCost: 8000}}
	err := svc.CreateDraft(ctx, po, items)
	require.NoError(t, err)

	err = svc.Cancel(ctx, po.ID, userID)
	require.NoError(t, err)

	err = svc.Cancel(ctx, po.ID, userID)
	assert.ErrorIs(t, err, ErrPurchaseOrderCancelled)
}

func TestPurchaseService_GetDetail_PassesStoreScope(t *testing.T) {
	svc, _, ctx := newSvc(t)
	storeID := 1
	po, err := svc.GetDetail(ctx, 999999, &storeID)
	assert.Nil(t, po)
	assert.Error(t, err)
}

func TestPurchaseService_List_PassesThrough(t *testing.T) {
	svc, _, ctx := newSvc(t)
	storeID := 1
	shifts, total, err := svc.List(ctx, 10, 0, "", "created_at", "DESC", "", "", "", "", &storeID)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, total, 0)
	_ = shifts
}

func TestPurchaseService_GetReceipts_PassesStoreScope(t *testing.T) {
	svc, _, ctx := newSvc(t)
	storeID := 1
	receipts, err := svc.GetReceipts(ctx, 999999, &storeID)
	assert.Empty(t, receipts)
	assert.Error(t, err)
}
