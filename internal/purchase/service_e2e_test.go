package purchase

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/inventory"
	"retail-pos-system/internal/product"
	"retail-pos-system/internal/storagelocation"
)

func newWiredInvRepo() *inventory.Repository {
	repo := inventory.NewRepository(dbPool)
	repo.SetProductMetaProvider(product.ProductMetaLookup{})
	repo.SetLocationRackProvider(storagelocation.RackProvider{})
	return repo
}

func ensureTestStore(t *testing.T) {
	t.Helper()
	_, err := dbPool.Exec(context.Background(), `
		INSERT INTO stores (id, name, is_active) VALUES (1, 'E2E Store', true) ON CONFLICT (id) DO NOTHING
	`)
	require.NoError(t, err)
}

func TestE2E_MultiItemGoodsReceiptAdjustsStockViaEvent(t *testing.T) {
	svc, bus, ctx := newSvc(t)
	ensureTestStore(t)

	invRepo := newWiredInvRepo()
	invSvc := inventory.NewService(invRepo, bus)
	bus.Subscribe(inventory.NewPurchaseReceiptListener(invRepo, invSvc))

	supplierID := insertTestSupplier(ctx, t, "E2E Multi Supplier")
	prodA := insertStoreTestProduct(ctx, t, "E2E-MULTI-A", "E2E Product A", 10000, 100, 1)
	prodB := insertStoreTestProduct(ctx, t, "E2E-MULTI-B", "E2E Product B", 12000, 200, 1)
	userID := insertTestUser(ctx, t, "e2e_multi_user")

	po := &Order{
		SupplierID: supplierID,
		StoreID:    1,
		CreatedBy:  userID,
		UpdatedBy:  userID,
	}
	items := []OrderItem{
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

	stockA, err := waitForStoreStock(t, prodA, 0, 110)
	require.NoError(t, err)
	assert.Equal(t, 110, stockA.Quantity)

	stockB, err := waitForStoreStock(t, prodB, 0, 205)
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
	ensureTestStore(t)

	invRepo := newWiredInvRepo()
	invSvc := inventory.NewService(invRepo, bus)
	bus.Subscribe(inventory.NewPurchaseReceiptListener(invRepo, invSvc))

	supplierID := insertTestSupplier(ctx, t, "E2E Partial Supplier")
	prodA := insertStoreTestProduct(ctx, t, "E2E-PART-A", "E2E Partial A", 10000, 50, 1)
	prodB := insertStoreTestProduct(ctx, t, "E2E-PART-B", "E2E Partial B", 12000, 60, 1)
	userID := insertTestUser(ctx, t, "e2e_partial_user")

	po := &Order{
		SupplierID: supplierID,
		StoreID:    1,
		CreatedBy:  userID,
		UpdatedBy:  userID,
	}
	items := []OrderItem{
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

	stockA, err := waitForStoreStock(t, prodA, 0, 54)
	require.NoError(t, err)
	assert.Equal(t, 54, stockA.Quantity)

	grItems2 := []CreateGRItemInput{
		{PurchaseOrderItemID: po.Items[1].ID, QtyGood: 5},
	}
	_, err = svc.CreateGoodsReceipt(ctx, po.ID, userID, 1, grItems2)
	require.NoError(t, err)

	stockB, err := waitForStoreStock(t, prodB, 0, 65)
	require.NoError(t, err)
	assert.Equal(t, 65, stockB.Quantity)

	stockA, err = storeStockByProductID(ctx, prodA, 0)
	require.NoError(t, err)
	assert.Equal(t, 54, stockA.Quantity, "first receipt must not be re-applied")
}

func waitForStoreStock(t *testing.T, productID, storeID, want int) (*inventory.ProductStock, error) {
	t.Helper()
	ctx := context.Background()
	deadline := time.Now().Add(5 * time.Second)
	var last *inventory.ProductStock
	for time.Now().Before(deadline) {
		stock, err := storeStockByProductID(ctx, productID, storeID)
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

// storeStockByProductID reads a product_stock bucket; storeID 0 selects the
// global bucket, where store-scoped goods receipts route their adjustments.
func storeStockByProductID(ctx context.Context, productID, storeID int) (*inventory.ProductStock, error) {
	var quantity int
	var err error
	if storeID == 0 {
		err = dbPool.QueryRow(ctx, `
			SELECT quantity FROM product_stock
			WHERE product_id = $1 AND store_id IS NULL AND warehouse_id IS NULL AND location_id IS NULL
		`, productID).Scan(&quantity)
	} else {
		err = dbPool.QueryRow(ctx, `
			SELECT quantity FROM product_stock
			WHERE product_id = $1 AND store_id = $2 AND warehouse_id IS NULL AND location_id IS NULL
		`, productID, storeID).Scan(&quantity)
	}
	if err != nil {
		return nil, err
	}
	return &inventory.ProductStock{ProductID: productID, Quantity: quantity}, nil
}
