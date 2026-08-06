package purchase

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/inventory"
)

func TestE2E_MultiItemGoodsReceiptAdjustsStockViaEvent(t *testing.T) {
	svc, bus, ctx := newSvc(t)

	invRepo := inventory.NewRepository(dbPool)
	invSvc := inventory.NewService(invRepo, bus)
	bus.Subscribe(inventory.NewPurchaseReceiptListener(invRepo, invSvc))

	supplierID := insertTestSupplier(t, ctx, "E2E Multi Supplier")
	prodA := insertTestProduct(t, ctx, "E2E-MULTI-A", "E2E Product A", 10000, 100)
	prodB := insertTestProduct(t, ctx, "E2E-MULTI-B", "E2E Product B", 12000, 200)
	userID := insertTestUser(t, ctx, "e2e_multi_user")

	po := &PurchaseOrder{
		SupplierID: supplierID,
		StoreID:    1,
		CreatedBy:  userID,
		UpdatedBy:  userID,
	}
	items := []PurchaseOrderItem{
		{ProductID: prodA, QtyOrdered: 10, UnitCost: 8000, ProductName: "E2E Product A", SKU: "E2E-MULTI-A"},
		{ProductID: prodB, QtyOrdered: 5, UnitCost: 9000, ProductName: "E2E Product B", SKU: "E2E-MULTI-B"},
	}
	require.NoError(t, svc.CreateDraft(ctx, po, items))
	require.NoError(t, svc.Confirm(ctx, po.ID, userID))

	po, err := svc.GetDetail(ctx, po.ID, nil)
	require.NoError(t, err)
	require.Len(t, po.Items, 2)

	grItems := []CreateGRItemInput{
		{PurchaseOrderItemID: po.Items[0].ID, QtyGood: 10, QtyDamaged: 0},
		{PurchaseOrderItemID: po.Items[1].ID, QtyGood: 5, QtyDamaged: 0},
	}
	gr, err := svc.CreateGoodsReceipt(ctx, po.ID, userID, 1, grItems)
	require.NoError(t, err)
	require.Greater(t, gr.ID, 0)

	stockA, err := waitForStock(t, invRepo, prodA, 110)
	require.NoError(t, err)
	assert.Equal(t, 110, stockA.Quantity)

	stockB, err := waitForStock(t, invRepo, prodB, 205)
	require.NoError(t, err)
	assert.Equal(t, 205, stockB.Quantity)

	fetched, err := svc.GetDetail(ctx, po.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, StatusFullyReceived, fetched.Status)
	assert.Equal(t, 10, fetched.Items[0].QtyReceived)
	assert.Equal(t, 5, fetched.Items[1].QtyReceived)
}

func TestE2E_PartialReceiptAcrossItemsAdjustsStockViaEvent(t *testing.T) {
	svc, bus, ctx := newSvc(t)

	invRepo := inventory.NewRepository(dbPool)
	invSvc := inventory.NewService(invRepo, bus)
	bus.Subscribe(inventory.NewPurchaseReceiptListener(invRepo, invSvc))

	supplierID := insertTestSupplier(t, ctx, "E2E Partial Supplier")
	prodA := insertTestProduct(t, ctx, "E2E-PART-A", "E2E Partial A", 10000, 50)
	prodB := insertTestProduct(t, ctx, "E2E-PART-B", "E2E Partial B", 12000, 60)
	userID := insertTestUser(t, ctx, "e2e_partial_user")

	po := &PurchaseOrder{
		SupplierID: supplierID,
		StoreID:    1,
		CreatedBy:  userID,
		UpdatedBy:  userID,
	}
	items := []PurchaseOrderItem{
		{ProductID: prodA, QtyOrdered: 10, UnitCost: 8000, ProductName: "E2E Partial A", SKU: "E2E-PART-A"},
		{ProductID: prodB, QtyOrdered: 10, UnitCost: 9000, ProductName: "E2E Partial B", SKU: "E2E-PART-B"},
	}
	require.NoError(t, svc.CreateDraft(ctx, po, items))
	require.NoError(t, svc.Confirm(ctx, po.ID, userID))

	po, err := svc.GetDetail(ctx, po.ID, nil)
	require.NoError(t, err)
	require.Len(t, po.Items, 2)

	grItems1 := []CreateGRItemInput{
		{PurchaseOrderItemID: po.Items[0].ID, QtyGood: 4},
	}
	_, err = svc.CreateGoodsReceipt(ctx, po.ID, userID, 1, grItems1)
	require.NoError(t, err)

	stockA, err := waitForStock(t, invRepo, prodA, 54)
	require.NoError(t, err)
	assert.Equal(t, 54, stockA.Quantity)

	grItems2 := []CreateGRItemInput{
		{PurchaseOrderItemID: po.Items[1].ID, QtyGood: 5},
	}
	_, err = svc.CreateGoodsReceipt(ctx, po.ID, userID, 1, grItems2)
	require.NoError(t, err)

	stockB, err := waitForStock(t, invRepo, prodB, 65)
	require.NoError(t, err)
	assert.Equal(t, 65, stockB.Quantity)

	stockA, err = invRepo.GetStockByProductID(ctx, prodA)
	require.NoError(t, err)
	assert.Equal(t, 54, stockA.Quantity, "first receipt must not be re-applied")
}

func waitForStock(t *testing.T, repo *inventory.Repository, productID, want int) (*inventory.ProductStock, error) {
	t.Helper()
	ctx := context.Background()
	deadline := time.Now().Add(5 * time.Second)
	var last *inventory.ProductStock
	for time.Now().Before(deadline) {
		stock, err := repo.GetStockByProductID(ctx, productID)
		if err != nil {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		last = stock
		if stock.Quantity == want {
			return stock, nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	if last != nil {
		return last, fmt.Errorf("timed out waiting for stock: got %d, want %d", last.Quantity, want)
	}
	return nil, fmt.Errorf("timed out waiting for stock row for product %d", productID)
}
