package consignment

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

// seedStoreOwnedStock writes the global product_stock bucket directly to model
// a product the store still owns (BR-02).
func seedStoreOwnedStock(ctx context.Context, t *testing.T, productID, qty int) {
	t.Helper()
	_, err := dbPool.Exec(ctx, `
		INSERT INTO product_stock (product_id, quantity, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT ON CONSTRAINT uq_product_stock DO UPDATE SET quantity = EXCLUDED.quantity
	`, productID, qty)
	require.NoError(t, err)
}

func TestService_ReceiptConflictMatrix(t *testing.T) {
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	t.Run("happy path records stock and ledger", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "REC-HAPPY")
		svc, sup, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		rec, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 10}},
		}, userID, &store)
		require.NoError(t, err)
		require.NotEmpty(t, rec.ReceiptNumber)
		require.Equal(t, sup, rec.SupplierID)
		require.NotEmpty(t, rec.SupplierName, "SupplierName should be hydrated")
		require.NotEmpty(t, rec.ReceivedByUsername, "ReceivedByUsername should be hydrated")
		require.Len(t, rec.Items, 1)
		require.Equal(t, 10, rec.Items[0].AcceptedQty)
		require.NotEmpty(t, rec.Items[0].ProductSKU, "ProductSKU should be hydrated")
		require.NotEmpty(t, rec.Items[0].ProductName, "ProductName should be hydrated")

		// Global sellable stock increased.
		require.Equal(t, 10, globalStockQty(ctx, t, product))

		// Ownership ledger row exists.
		row, err := svc.repo.GetConsignmentStock(ctx, svc.repo.db, product)
		require.NoError(t, err)
		require.NotNil(t, row)
		require.Equal(t, sup, row.SupplierID)
		require.Equal(t, 10, row.AvailableQty)
		require.Equal(t, 0, row.PendingReturnQty)
	})

	t.Run("BR-02 rejects product with store-owned stock", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "REC-BR02")
		svc, _, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)
		seedStoreOwnedStock(ctx, t, product, 5)

		_, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 3}},
		}, userID, &store)
		require.ErrorIs(t, err, ErrConflictStoreStock)
	})

	t.Run("BR-02 zero store stock allows first consignment", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "REC-BR02-ZERO")
		svc, _, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)
		seedStoreOwnedStock(ctx, t, product, 0)

		rec, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 3}},
		}, userID, &store)
		require.NoError(t, err)
		require.NotEmpty(t, rec.ReceiptNumber)
	})

	t.Run("EC-01 same supplier top-up allowed", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "REC-EC01")
		svc, _, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		_, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 5}},
		}, userID, &store)
		require.NoError(t, err)

		rec, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 7}},
		}, userID, &store)
		require.NoError(t, err)
		require.NotEmpty(t, rec.ReceiptNumber)
		require.Equal(t, 12, globalStockQty(ctx, t, product))
	})

	t.Run("EC-02 same supplier top-up with pending return allowed", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "REC-EC02")
		svc, _, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		_, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 10}},
		}, userID, &store)
		require.NoError(t, err)

		pr, err := svc.CreatePendingReturn(ctx, &CreatePendingReturnRequest{
			ProductID: product,
			Qty:       4,
			Reason:    ReasonDamaged,
		}, userID, &store)
		require.NoError(t, err)
		require.Equal(t, 4, pr.Qty)

		rec, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 6}},
		}, userID, &store)
		require.NoError(t, err)
		require.NotEmpty(t, rec.ReceiptNumber)

		// Pending return stays untouched by the top-up.
		row, err := svc.repo.GetConsignmentStock(ctx, svc.repo.db, product)
		require.NoError(t, err)
		require.Equal(t, 12, row.AvailableQty)
		require.Equal(t, 4, row.PendingReturnQty)
	})

	t.Run("BR-03 other supplier with stock rejected", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "REC-BR03")
		svcA, _, storeA := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)
		_, err := svcA.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svcA, storeA),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 5}},
		}, userID, &storeA)
		require.NoError(t, err)

		// Supplier B creates arrangement without terms (can't set terms on owned product).
		svcB, _, storeB, _ := setupArrangementNoTerms(t)
		_, err = svcB.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svcB, storeB),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 3}},
		}, userID, &storeB)
		require.ErrorIs(t, err, ErrConflictOtherSupplier)
	})

	t.Run("EC-12 pending return blocks other supplier", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "REC-EC12")
		svcA, _, storeA := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)
		_, err := svcA.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svcA, storeA),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 5}},
		}, userID, &storeA)
		require.NoError(t, err)

		pr, err := svcA.CreatePendingReturn(ctx, &CreatePendingReturnRequest{
			ProductID: product,
			Qty:       5,
			Reason:    ReasonDamaged,
		}, userID, &storeA)
		require.NoError(t, err)
		require.Equal(t, 5, pr.Qty)

		// Available is now 0 but the pending return keeps ownership (BR-05b).
		row, err := svcA.repo.GetConsignmentStock(ctx, svcA.repo.db, product)
		require.NoError(t, err)
		require.Equal(t, 0, row.AvailableQty)
		require.Equal(t, 5, row.PendingReturnQty)

		svcB, _, storeB, _ := setupArrangementNoTerms(t)
		_, err = svcB.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svcB, storeB),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 2}},
		}, userID, &storeB)
		require.ErrorIs(t, err, ErrPendingReturnBlocksTransfer)
	})

	t.Run("ownership released transfers to new supplier", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "REC-RELEASE")
		svcA, _, storeA := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)
		_, err := svcA.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svcA, storeA),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 5}},
		}, userID, &storeA)
		require.NoError(t, err)

		// Return everything so available = 0 (BR-05b release condition).
		_, err = svcA.CreateReturn(ctx, &ReturnRequest{
			ArrangementID: arrID(t, svcA, storeA),
			Items:         []ReturnItemRequest{{ProductID: product, Qty: 5, Reason: ReasonOther}},
		}, userID, &storeA)
		require.NoError(t, err)

		// Full return releases ownership: the ledger row is deleted when both
		// available and pending return reach zero (BR-05b release condition).
		row, err := svcA.repo.GetConsignmentStock(ctx, svcA.repo.db, product)
		require.NoError(t, err)
		require.Nil(t, row)

		// New supplier can now take the SKU.
		svcB, supB, storeB := setupArrangement(t, product)
		rec, err := svcB.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svcB, storeB),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 8}},
		}, userID, &storeB)
		require.NoError(t, err)
		require.NotEmpty(t, rec.ReceiptNumber)

		row, err = svcB.repo.GetConsignmentStock(ctx, svcB.repo.db, product)
		require.NoError(t, err)
		require.Equal(t, supB, row.SupplierID)
	})

	t.Run("sold-out row taken over by new supplier refreshes owner columns", func(t *testing.T) {
		// A sale (not a return) empties available_qty, leaving a 0/0 ledger row
		// behind. A new supplier must be able to take the SKU over, and the
		// receipt upsert must refresh the stale owner columns (M1).
		product := insertTestProduct(ctx, t, "REC-SOLD-OUT")
		svcA, _, storeA := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)
		_, err := svcA.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svcA, storeA),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 5}},
		}, userID, &storeA)
		require.NoError(t, err)

		// Simulate checkout draining available to zero via the sale path
		// (ReduceAvailable is the same repo call the checkout provider uses).
		tx, err := svcA.repo.BeginTx(ctx)
		require.NoError(t, err)
		err = svcA.repo.ReduceAvailable(ctx, tx, product, 5)
		require.NoError(t, err)
		require.NoError(t, tx.Commit(ctx))

		row, err := svcA.repo.GetConsignmentStock(ctx, svcA.repo.db, product)
		require.NoError(t, err)
		require.NotNil(t, row)
		require.Equal(t, 0, row.AvailableQty)
		require.Equal(t, 0, row.PendingReturnQty)

		// New supplier takes the SKU over; ownership must not stay stale.
		svcB, supB, storeB := setupArrangement(t, product)
		rec, err := svcB.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svcB, storeB),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 8}},
		}, userID, &storeB)
		require.NoError(t, err)
		require.NotEmpty(t, rec.ReceiptNumber)

		row, err = svcB.repo.GetConsignmentStock(ctx, svcB.repo.db, product)
		require.NoError(t, err)
		require.NotNil(t, row)
		require.Equal(t, supB, row.SupplierID)
		require.Equal(t, arrID(t, svcB, storeB), row.ArrangementID)
		require.Equal(t, storeB, row.StoreID)
		require.Equal(t, 8, row.AvailableQty)
	})

	t.Run("multi-sku independent ownership", func(t *testing.T) {
		skuA := insertTestProduct(ctx, t, "REC-INDEP-A")
		skuB := insertTestProduct(ctx, t, "REC-INDEP-B")
		svcA, _, storeA := setupArrangement(t, skuA, skuB)
		userID := insertTestUser(ctx, t)

		// skuA is store-owned, so only skuB can be consigned. The whole
		// receipt is atomic: skuA's conflict aborts skuB's acceptance too.
		seedStoreOwnedStock(ctx, t, skuA, 3)
		_, err := svcA.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svcA, storeA),
			Items: []ReceiptItemRequest{
				{ProductID: skuA, AcceptedQty: 2},
				{ProductID: skuB, AcceptedQty: 4},
			},
		}, userID, &storeA)
		require.ErrorIs(t, err, ErrConflictStoreStock)

		// Nothing was recorded.
		row, err := svcA.repo.GetConsignmentStock(ctx, svcA.repo.db, skuB)
		require.NoError(t, err)
		require.Nil(t, row)
		require.Equal(t, 0, globalStockQty(ctx, t, skuB))
	})
}

func arrID(t *testing.T, svc *Service, store int) int {
	t.Helper()
	arrs, _, err := svc.ListArrangements(context.Background(), &store, 0, 0, "", "")
	require.NoError(t, err)
	require.NotEmpty(t, arrs)
	return arrs[0].ID
}

func TestService_EditReceipt(t *testing.T) {
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	t.Run("happy path edits qty and price", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "EDT-HAPPY")
		svc, _, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		rec, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 10}},
		}, userID, &store)
		require.NoError(t, err)
		require.Len(t, rec.Items, 1)
		itemID := rec.Items[0].ID

		// Edit: increase qty to 15, change price.
		updated, err := svc.EditReceipt(ctx, rec.ID, EditReceiptInput{
			Items: []EditReceiptItemInput{{
				ID:          itemID,
				AcceptedQty: 15,
				Price:       12000,
			}},
			Reason: "supplier delivered extra",
		}, userID, "127.0.0.1")
		require.NoError(t, err)
		require.Equal(t, 15, updated.Items[0].AcceptedQty)
		require.Equal(t, 12000, updated.Items[0].Price)

		// Global stock increased by delta (+5).
		require.Equal(t, 15, globalStockQty(ctx, t, product))

		// Consignment ledger updated.
		row, err := svc.repo.GetConsignmentStock(ctx, svc.repo.db, product)
		require.NoError(t, err)
		require.Equal(t, 15, row.AvailableQty)
	})

	t.Run("happy path edits only notes", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "EDT-NOTES")
		svc, _, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		rec, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 5}},
		}, userID, &store)
		require.NoError(t, err)

		notes := "updated notes"
		updated, err := svc.EditReceipt(ctx, rec.ID, EditReceiptInput{
			Items: []EditReceiptItemInput{{
				ID:          rec.Items[0].ID,
				AcceptedQty: 5,
				Price:       10000,
			}},
			Notes:  &notes,
			Reason: "note update",
		}, userID, "127.0.0.1")
		require.NoError(t, err)
		require.Equal(t, "updated notes", updated.Notes)
		// Stock unchanged.
		require.Equal(t, 5, globalStockQty(ctx, t, product))
	})

	t.Run("decrease qty updates stock", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "EDT-DEC")
		svc, _, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		rec, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 10}},
		}, userID, &store)
		require.NoError(t, err)

		updated, err := svc.EditReceipt(ctx, rec.ID, EditReceiptInput{
			Items: []EditReceiptItemInput{{
				ID:          rec.Items[0].ID,
				AcceptedQty: 3,
				Price:       10000,
			}},
			Reason: "supplier recalled some",
		}, userID, "127.0.0.1")
		require.NoError(t, err)
		require.Equal(t, 3, updated.Items[0].AcceptedQty)
		require.Equal(t, 3, globalStockQty(ctx, t, product))
	})

	t.Run("rejects missing reason", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "EDT-NO-REASON")
		svc, _, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		rec, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 5}},
		}, userID, &store)
		require.NoError(t, err)

		_, err = svc.EditReceipt(ctx, rec.ID, EditReceiptInput{
			Items:  []EditReceiptItemInput{{ID: rec.Items[0].ID, AcceptedQty: 5, Price: 10000}},
			Reason: "",
		}, userID, "127.0.0.1")
		require.ErrorIs(t, err, ErrEditReasonRequired)
	})

	t.Run("rejects nonexistent receipt", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "EDT-NOEXIST")
		svc, _, _ := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		_, err := svc.EditReceipt(ctx, 999999, EditReceiptInput{
			Items:  []EditReceiptItemInput{{ID: 1, AcceptedQty: 1, Price: 10000}},
			Reason: "test",
		}, userID, "127.0.0.1")
		require.ErrorIs(t, err, ErrReceiptNotFound)
	})

	t.Run("rejects nonexistent item ID", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "EDT-NOITEM")
		svc, _, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		rec, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 5}},
		}, userID, &store)
		require.NoError(t, err)

		_, err = svc.EditReceipt(ctx, rec.ID, EditReceiptInput{
			Items:  []EditReceiptItemInput{{ID: 999999, AcceptedQty: 5, Price: 10000}},
			Reason: "bad item",
		}, userID, "127.0.0.1")
		require.ErrorIs(t, err, ErrReceiptEditItemNotFound)
	})

	t.Run("rejects price change when receipt has sales", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "EDT-PRICE-SALE")
		svc, sup, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		rec, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 10}},
		}, userID, &store)
		require.NoError(t, err)

		// Simulate a sale by inserting a consignment sale item.
		saleID := insertTestSale(ctx, t, store, "EDT-PRICE-SALE-INV")
		insertConsignmentSaleItem(ctx, t, svc, shared.ConsignmentSaleRecord{
			SaleID: saleID, InvoiceNumber: "EDT-PRICE-SALE-INV", ProductID: product,
			SupplierID: sup, ArrangementID: arrID(t, svc, store), StoreID: store,
			Quantity: 3, UnitPrice: 10000, Subtotal: 30000, StoreShareType: "percentage", StoreShareValue: 20,
		})

		_, err = svc.EditReceipt(ctx, rec.ID, EditReceiptInput{
			Items:  []EditReceiptItemInput{{ID: rec.Items[0].ID, AcceptedQty: 10, Price: 12000}},
			Reason: "price bump",
		}, userID, "127.0.0.1")
		require.ErrorIs(t, err, ErrReceiptHasSales)
	})

	t.Run("rejects qty change when receipt has sales", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "EDT-QTY-SALE")
		svc, sup, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		rec, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 10}},
		}, userID, &store)
		require.NoError(t, err)

		saleID := insertTestSale(ctx, t, store, "EDT-QTY-SALE-INV")
		insertConsignmentSaleItem(ctx, t, svc, shared.ConsignmentSaleRecord{
			SaleID: saleID, InvoiceNumber: "EDT-QTY-SALE-INV", ProductID: product,
			SupplierID: sup, ArrangementID: arrID(t, svc, store), StoreID: store,
			Quantity: 2, UnitPrice: 10000, Subtotal: 20000, StoreShareType: "percentage", StoreShareValue: 20,
		})

		_, err = svc.EditReceipt(ctx, rec.ID, EditReceiptInput{
			Items:  []EditReceiptItemInput{{ID: rec.Items[0].ID, AcceptedQty: 8, Price: 10000}},
			Reason: "qty change",
		}, userID, "127.0.0.1")
		require.ErrorIs(t, err, ErrReceiptHasSales)
	})

	t.Run("rejects qty change when receipt has pending returns", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "EDT-QTY-RET")
		svc, _, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		rec, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 10}},
		}, userID, &store)
		require.NoError(t, err)

		_, err = svc.CreatePendingReturn(ctx, &CreatePendingReturnRequest{
			ProductID: product, Qty: 3, Reason: ReasonDamaged,
		}, userID, &store)
		require.NoError(t, err)

		_, err = svc.EditReceipt(ctx, rec.ID, EditReceiptInput{
			Items:  []EditReceiptItemInput{{ID: rec.Items[0].ID, AcceptedQty: 7, Price: 10000}},
			Reason: "pending ret block",
		}, userID, "127.0.0.1")
		require.ErrorIs(t, err, ErrReceiptHasPendingReturns)
	})

	t.Run("rejects edit when receipt is settled", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "EDT-SETTLED")
		svc, sup, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		rec, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 10}},
		}, userID, &store)
		require.NoError(t, err)

		// Simulate a sale + settlement.
		saleID := insertTestSale(ctx, t, store, "EDT-SETTLED-INV")
		insertConsignmentSaleItem(ctx, t, svc, shared.ConsignmentSaleRecord{
			SaleID: saleID, InvoiceNumber: "EDT-SETTLED-INV", ProductID: product,
			SupplierID: sup, ArrangementID: arrID(t, svc, store), StoreID: store,
			Quantity: 5, UnitPrice: 10000, Subtotal: 50000, StoreShareType: "percentage", StoreShareValue: 20,
		})

		_, err = svc.CreateSettlement(ctx, &CreateSettlementRequest{
			SupplierID: sup,
		}, userID, &store)
		require.NoError(t, err)

		_, err = svc.EditReceipt(ctx, rec.ID, EditReceiptInput{
			Items:  []EditReceiptItemInput{{ID: rec.Items[0].ID, AcceptedQty: 10, Price: 10000}},
			Reason: "settled",
		}, userID, "127.0.0.1")
		require.ErrorIs(t, err, ErrReceiptIsSettled)
	})

	t.Run("negative stock rejected", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "EDT-NEG-STOCK")
		svc, _, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		rec, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 5}},
		}, userID, &store)
		require.NoError(t, err)

		// Sell 4 of the 5 via ReduceAvailable so only 1 remains.
		tx, err := svc.repo.BeginTx(ctx)
		require.NoError(t, err)
		require.NoError(t, svc.repo.ReduceAvailable(ctx, tx, product, 4))
		require.NoError(t, tx.Commit(ctx))

		// Try to reduce below zero.
		_, err = svc.EditReceipt(ctx, rec.ID, EditReceiptInput{
			Items:  []EditReceiptItemInput{{ID: rec.Items[0].ID, AcceptedQty: 0, Price: 10000}},
			Reason: "negative",
		}, userID, "127.0.0.1")
		require.ErrorIs(t, err, ErrNegativeStock)
	})

	t.Run("audit trail entries created", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "EDT-AUDIT")
		svc, _, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		rec, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 10}},
		}, userID, &store)
		require.NoError(t, err)

		_, err = svc.EditReceipt(ctx, rec.ID, EditReceiptInput{
			Items: []EditReceiptItemInput{{
				ID:          rec.Items[0].ID,
				AcceptedQty: 12,
				Price:       15000,
			}},
			Reason: "audit test",
		}, userID, "10.0.0.1")
		require.NoError(t, err)

		// Check audit trail entries exist.
		var count int
		err = dbPool.QueryRow(ctx, `
			SELECT COUNT(*) FROM consignment_receipt_edits WHERE receipt_id = $1
		`, rec.ID).Scan(&count)
		require.NoError(t, err)
		// 2 entries: qty change + price change.
		require.Equal(t, 2, count)

		// Verify IP address stored.
		var ip string
		err = dbPool.QueryRow(ctx, `
			SELECT ip_address::text FROM consignment_receipt_edits
			WHERE receipt_id = $1 AND ip_address IS NOT NULL LIMIT 1
		`, rec.ID).Scan(&ip)
		require.NoError(t, err)
		require.Equal(t, "10.0.0.1/32", ip)
	})

	t.Run("same qty no-op produces no audit for qty", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "EDT-NOOP")
		svc, _, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		rec, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 5}},
		}, userID, &store)
		require.NoError(t, err)

		// Edit notes only — no qty/price audit entries.
		notes := "just notes"
		_, err = svc.EditReceipt(ctx, rec.ID, EditReceiptInput{
			Items:  []EditReceiptItemInput{{ID: rec.Items[0].ID, AcceptedQty: 5, Price: 10000}},
			Notes:  &notes,
			Reason: "notes only",
		}, userID, "127.0.0.1")
		require.NoError(t, err)

		var count int
		err = dbPool.QueryRow(ctx, `
			SELECT COUNT(*) FROM consignment_receipt_edits
			WHERE receipt_id = $1 AND field LIKE 'item_%'
		`, rec.ID).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 0, count, "no item audit entries when qty/price unchanged")
	})

	t.Run("IsConsignmentOwned returns true after receipt", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "EDT-OWNED")
		svc, _, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		_, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 5}},
		}, userID, &store)
		require.NoError(t, err)

		owned, err := svc.IsConsignmentOwned(ctx, product)
		require.NoError(t, err)
		require.True(t, owned)
	})

	t.Run("IsConsignmentOwned returns false for non-consignment product", func(t *testing.T) {
		svc, _, _ := setupArrangement(t)
		product := insertTestProduct(ctx, t, "EDT-NOT-OWNED")

		owned, err := svc.IsConsignmentOwned(ctx, product)
		require.NoError(t, err)
		require.False(t, owned)
	})
}
