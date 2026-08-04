package purchase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/eventbus"
)

func newSvc(t *testing.T) (*Service, *eventbus.Bus, context.Context) {
	t.Helper()
	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()
	t.Cleanup(bus.Shutdown)
	return NewService(repo, bus), bus, context.Background()
}

func TestPurchaseService_CreateDraft(t *testing.T) {
	svc, _, ctx := newSvc(t)

	supplierID := insertTestSupplier(t, ctx, "Svc Draft Supplier")
	prodID := insertTestProduct(t, ctx, "SVC-DRAFT-001", "Draft Product", 10000, 100)
	userID := insertTestUser(t, ctx, "po_draft_user")

	po := &PurchaseOrder{
		SupplierID: supplierID,
		StoreID:    1,
		CreatedBy:  userID,
		UpdatedBy:  userID,
	}
	items := []PurchaseOrderItem{
		{
			ProductID:   prodID,
			QtyOrdered:  5,
			UnitCost:    8000,
			ProductName: "Draft Product",
			SKU:         "SVC-DRAFT-001",
		},
	}

	err := svc.CreateDraft(ctx, po, items)
	require.NoError(t, err)
	assert.Greater(t, po.ID, 0)
	assert.Equal(t, StatusDraft, po.Status)
	assert.Equal(t, 40000, po.Subtotal)
	assert.Equal(t, 40000, po.GrandTotal)
}

func TestPurchaseService_Confirm(t *testing.T) {
	svc, _, ctx := newSvc(t)

	supplierID := insertTestSupplier(t, ctx, "Svc Confirm Supplier")
	prodID := insertTestProduct(t, ctx, "SVC-CONF-001", "Confirm Product", 10000, 100)
	userID := insertTestUser(t, ctx, "po_confirm_svc_user")

	po := &PurchaseOrder{
		SupplierID: supplierID,
		StoreID:    1,
		CreatedBy:  userID,
		UpdatedBy:  userID,
	}
	items := []PurchaseOrderItem{
		{
			ProductID:   prodID,
			QtyOrdered:  5,
			UnitCost:    8000,
			ProductName: "Confirm Product",
			SKU:         "SVC-CONF-001",
		},
	}

	err := svc.CreateDraft(ctx, po, items)
	require.NoError(t, err)

	err = svc.Confirm(ctx, po.ID, userID)
	require.NoError(t, err)

	fetched, err := svc.GetDetail(ctx, po.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, StatusConfirmed, fetched.Status)
	assert.NotNil(t, fetched.ConfirmedBy)
	assert.Equal(t, userID, *fetched.ConfirmedBy)
}

func TestPurchaseService_CancelWithReceiptsFails(t *testing.T) {
	svc, _, ctx := newSvc(t)

	supplierID := insertTestSupplier(t, ctx, "Svc Cancel Supplier")
	prodID := insertTestProduct(t, ctx, "SVC-CANCEL-001", "Cancel Product", 10000, 100)
	userID := insertTestUser(t, ctx, "po_cancel_svc_user")

	po := &PurchaseOrder{
		SupplierID: supplierID,
		StoreID:    1,
		CreatedBy:  userID,
		UpdatedBy:  userID,
	}
	items := []PurchaseOrderItem{
		{
			ProductID:   prodID,
			QtyOrdered:  10,
			UnitCost:    8000,
			ProductName: "Cancel Product",
			SKU:         "SVC-CANCEL-001",
		},
	}

	err := svc.CreateDraft(ctx, po, items)
	require.NoError(t, err)
	err = svc.Confirm(ctx, po.ID, userID)
	require.NoError(t, err)

	po, err = svc.GetDetail(ctx, po.ID, nil)
	require.NoError(t, err)

	grItems := []CreateGRItemInput{
		{
			PurchaseOrderItemID: po.Items[0].ID,
			QtyGood:             5,
			QtyDamaged:          1,
		},
	}
	_, err = svc.CreateGoodsReceipt(ctx, po.ID, userID, 1, grItems)
	require.NoError(t, err)

	err = svc.Cancel(ctx, po.ID, userID)
	assert.ErrorIs(t, err, ErrPurchaseOrderHasReceipts)
}

func TestPurchaseService_PartialAndFullReceive(t *testing.T) {
	svc, _, ctx := newSvc(t)

	supplierID := insertTestSupplier(t, ctx, "Svc Receive Supplier")
	prodID := insertTestProduct(t, ctx, "SVC-RECV-001", "Receive Product", 10000, 100)
	userID := insertTestUser(t, ctx, "po_receive_svc_user")

	po := &PurchaseOrder{
		SupplierID: supplierID,
		StoreID:    1,
		CreatedBy:  userID,
		UpdatedBy:  userID,
	}
	items := []PurchaseOrderItem{
		{
			ProductID:   prodID,
			QtyOrdered:  10,
			UnitCost:    8000,
			ProductName: "Receive Product",
			SKU:         "SVC-RECV-001",
		},
	}

	err := svc.CreateDraft(ctx, po, items)
	require.NoError(t, err)
	err = svc.Confirm(ctx, po.ID, userID)
	require.NoError(t, err)

	po, err = svc.GetDetail(ctx, po.ID, nil)
	require.NoError(t, err)

	grItems1 := []CreateGRItemInput{
		{
			PurchaseOrderItemID: po.Items[0].ID,
			QtyGood:             4,
			QtyDamaged:          1,
		},
	}
	_, err = svc.CreateGoodsReceipt(ctx, po.ID, userID, 1, grItems1)
	require.NoError(t, err)

	fetched, err := svc.GetDetail(ctx, po.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, StatusPartialReceived, fetched.Status)
	assert.Equal(t, 5, fetched.Items[0].QtyReceived)

	grItems2 := []CreateGRItemInput{
		{
			PurchaseOrderItemID: po.Items[0].ID,
			QtyGood:             5,
			QtyDamaged:          0,
		},
	}
	_, err = svc.CreateGoodsReceipt(ctx, po.ID, userID, 1, grItems2)
	require.NoError(t, err)

	fetched, err = svc.GetDetail(ctx, po.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, StatusFullyReceived, fetched.Status)
	assert.Equal(t, 10, fetched.Items[0].QtyReceived)
}

func TestPurchaseService_OverReceivePrevented(t *testing.T) {
	svc, _, ctx := newSvc(t)

	supplierID := insertTestSupplier(t, ctx, "Svc OverReceive Supplier")
	prodID := insertTestProduct(t, ctx, "SVC-OVER-001", "OverReceive Product", 10000, 100)
	userID := insertTestUser(t, ctx, "po_over_receive_user")

	po := &PurchaseOrder{
		SupplierID: supplierID,
		StoreID:    1,
		CreatedBy:  userID,
		UpdatedBy:  userID,
	}
	items := []PurchaseOrderItem{
		{
			ProductID:   prodID,
			QtyOrdered:  10,
			UnitCost:    8000,
			ProductName: "OverReceive Product",
			SKU:         "SVC-OVER-001",
		},
	}

	err := svc.CreateDraft(ctx, po, items)
	require.NoError(t, err)
	err = svc.Confirm(ctx, po.ID, userID)
	require.NoError(t, err)

	po, err = svc.GetDetail(ctx, po.ID, nil)
	require.NoError(t, err)

	grItems := []CreateGRItemInput{
		{
			PurchaseOrderItemID: po.Items[0].ID,
			QtyGood:             11,
			QtyDamaged:          0,
		},
	}
	_, err = svc.CreateGoodsReceipt(ctx, po.ID, userID, 1, grItems)
	assert.ErrorIs(t, err, ErrOverReceiving)
}

func TestPurchaseService_UpdateDraft(t *testing.T) {
	svc, _, ctx := newSvc(t)

	supplierID := insertTestSupplier(t, ctx, "Svc Update Supplier")
	prodID := insertTestProduct(t, ctx, "SVC-UPD-001", "Update Product", 10000, 100)
	userID := insertTestUser(t, ctx, "po_update_user")

	po := &PurchaseOrder{
		SupplierID: supplierID,
		StoreID:    1,
		CreatedBy:  userID,
		UpdatedBy:  userID,
	}
	items := []PurchaseOrderItem{
		{
			ProductID:  prodID,
			QtyOrdered: 5,
			UnitCost:   8000,
		},
	}

	err := svc.CreateDraft(ctx, po, items)
	require.NoError(t, err)

	t.Run("updates draft PO with enriched product names", func(t *testing.T) {
		updated := &PurchaseOrder{
			SupplierID: supplierID,
			StoreID:    1,
			UpdatedBy:  userID,
		}
		updatedItems := []PurchaseOrderItem{
			{
				ProductID:  prodID,
				QtyOrdered: 10,
				UnitCost:   8000,
			},
		}

		err := svc.UpdateDraft(ctx, po.ID, updated, updatedItems)
		require.NoError(t, err)

		fetched, err := svc.GetDetail(ctx, po.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, StatusDraft, fetched.Status)
		assert.Equal(t, "Update Product", fetched.Items[0].ProductName)
		assert.Equal(t, "SVC-UPD-001", fetched.Items[0].SKU)
		assert.Equal(t, 10, fetched.Items[0].QtyOrdered)
		assert.Equal(t, 80000, fetched.Subtotal)
		assert.Equal(t, 80000, fetched.GrandTotal)
	})

	t.Run("rejects update when PO is confirmed", func(t *testing.T) {
		confirmPO := &PurchaseOrder{
			SupplierID: supplierID,
			StoreID:    1,
			CreatedBy:  userID,
			UpdatedBy:  userID,
		}
		confirmItems := []PurchaseOrderItem{
			{ProductID: prodID, QtyOrdered: 3, UnitCost: 8000},
		}
		err := svc.CreateDraft(ctx, confirmPO, confirmItems)
		require.NoError(t, err)
		err = svc.Confirm(ctx, confirmPO.ID, userID)
		require.NoError(t, err)

		err = svc.UpdateDraft(ctx, confirmPO.ID, confirmPO, confirmItems)
		assert.ErrorIs(t, err, ErrPurchaseOrderNotDraft)
	})

	t.Run("rejects update with zero UpdatedBy", func(t *testing.T) {
		err := svc.UpdateDraft(ctx, po.ID, &PurchaseOrder{SupplierID: supplierID, StoreID: 1}, items)
		assert.ErrorContains(t, err, "updated_by is required")
	})

	t.Run("empty items returns error", func(t *testing.T) {
		err := svc.UpdateDraft(ctx, po.ID, &PurchaseOrder{SupplierID: supplierID, StoreID: 1, UpdatedBy: userID}, nil)
		assert.ErrorContains(t, err, "items cannot be empty")
	})

	t.Run("duplicate product IDs returns error", func(t *testing.T) {
		dupItems := []PurchaseOrderItem{
			{ProductID: prodID, QtyOrdered: 2, UnitCost: 8000},
			{ProductID: prodID, QtyOrdered: 3, UnitCost: 8000},
		}
		err := svc.UpdateDraft(ctx, po.ID, &PurchaseOrder{SupplierID: supplierID, StoreID: 1, UpdatedBy: userID}, dupItems)
		assert.ErrorIs(t, err, ErrDuplicatePOItem)
	})

	t.Run("enriches items with product lookup even with discount_amount", func(t *testing.T) {
		discountPO := &PurchaseOrder{
			SupplierID: supplierID,
			StoreID:    1,
			CreatedBy:  userID,
			UpdatedBy:  userID,
		}
		discountItems := []PurchaseOrderItem{
			{ProductID: prodID, QtyOrdered: 10, UnitCost: 10000, DiscountAmount: 5000},
		}
		err := svc.CreateDraft(ctx, discountPO, discountItems)
		require.NoError(t, err)

		updated := &PurchaseOrder{
			SupplierID: supplierID,
			StoreID:    1,
			UpdatedBy:  userID,
		}
		updatedItems := []PurchaseOrderItem{
			{ProductID: prodID, QtyOrdered: 10, UnitCost: 10000, DiscountAmount: 5000},
		}
		err = svc.UpdateDraft(ctx, discountPO.ID, updated, updatedItems)
		require.NoError(t, err)

		fetched, err := svc.GetDetail(ctx, discountPO.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, "Update Product", fetched.Items[0].ProductName)
		assert.Equal(t, 10, fetched.Items[0].QtyOrdered)
		assert.Equal(t, 95000, fetched.Subtotal)
	})
}

func TestPurchaseService_DuplicateItemRejected(t *testing.T) {
	svc, _, ctx := newSvc(t)

	supplierID := insertTestSupplier(t, ctx, "Svc Dup Supplier")
	prodID := insertTestProduct(t, ctx, "SVC-DUP-001", "Dup Product", 10000, 100)
	userID := insertTestUser(t, ctx, "po_dup_user")

	po := &PurchaseOrder{
		SupplierID: supplierID,
		StoreID:    1,
		CreatedBy:  userID,
		UpdatedBy:  userID,
	}
	items := []PurchaseOrderItem{
		{
			ProductID:   prodID,
			QtyOrdered:  5,
			UnitCost:    8000,
			ProductName: "Dup Product",
			SKU:         "SVC-DUP-001",
		},
		{
			ProductID:   prodID,
			QtyOrdered:  3,
			UnitCost:    8000,
			ProductName: "Dup Product",
			SKU:         "SVC-DUP-001",
		},
	}

	err := svc.CreateDraft(ctx, po, items)
	assert.ErrorIs(t, err, ErrDuplicatePOItem)
}

func TestPurchaseService_ReceiveOnDraftFails(t *testing.T) {
	svc, _, ctx := newSvc(t)

	supplierID := insertTestSupplier(t, ctx, "Svc Draft Receive Supplier")
	prodID := insertTestProduct(t, ctx, "SVC-DRAFT-RECV-001", "Draft Receive Product", 10000, 100)
	userID := insertTestUser(t, ctx, "po_draft_recv_user")

	po := &PurchaseOrder{
		SupplierID: supplierID,
		StoreID:    1,
		CreatedBy:  userID,
		UpdatedBy:  userID,
	}
	items := []PurchaseOrderItem{
		{
			ProductID:   prodID,
			QtyOrdered:  5,
			UnitCost:    8000,
			ProductName: "Draft Receive Product",
			SKU:         "SVC-DRAFT-RECV-001",
		},
	}

	err := svc.CreateDraft(ctx, po, items)
	require.NoError(t, err)

	po, err = svc.GetDetail(ctx, po.ID, nil)
	require.NoError(t, err)

	grItems := []CreateGRItemInput{
		{
			PurchaseOrderItemID: po.Items[0].ID,
			QtyGood:             5,
			QtyDamaged:          0,
		},
	}
	_, err = svc.CreateGoodsReceipt(ctx, po.ID, userID, 1, grItems)
	assert.ErrorIs(t, err, ErrInvalidPOStatusForReceiving)
}
