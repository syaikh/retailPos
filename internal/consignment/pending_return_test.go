package consignment

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

func TestService_PendingReturn(t *testing.T) {
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	t.Run("pull moves qty from available to pending", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "PR-PULL")
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

		row, err := svc.repo.GetConsignmentStock(ctx, svc.repo.db, product)
		require.NoError(t, err)
		require.Equal(t, 6, row.AvailableQty)
		require.Equal(t, 4, row.PendingReturnQty)

		// Goods leave the sellable product_stock (BR-26/AC-C20).
		require.Equal(t, 6, globalStockQty(ctx, t, product))
	})

	t.Run("rejects qty above available", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "PR-OVER")
		svc, _, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		_, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 5}},
		}, userID, &store)
		require.NoError(t, err)

		_, err = svc.CreatePendingReturn(ctx, &CreatePendingReturnRequest{
			ProductID: product,
			Qty:       6,
			Reason:    ReasonDamaged,
		}, userID, &store)
		require.ErrorIs(t, err, ErrInsufficientConsignmentStock)
	})

	t.Run("rejects invalid reason", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "PR-REASON")
		svc, _, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		_, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 5}},
		}, userID, &store)
		require.NoError(t, err)

		_, err = svc.CreatePendingReturn(ctx, &CreatePendingReturnRequest{
			ProductID: product,
			Qty:       2,
			Reason:    "bogus",
		}, userID, &store)
		require.ErrorIs(t, err, ErrInvalidReason)
	})

	t.Run("EC-06 EC-08 damaged and expired become pending returns", func(t *testing.T) {
		skuD := insertTestProduct(ctx, t, "PR-DMG")
		skuE := insertTestProduct(ctx, t, "PR-EXP")
		svc, sup, store := setupArrangement(t, skuD, skuE)
		userID := insertTestUser(ctx, t)

		for _, p := range []int{skuD, skuE} {
			_, err := svc.CreateReceipt(ctx, &ReceiptRequest{
				ArrangementID: arrID(t, svc, store),
				Items:         []ReceiptItemRequest{{ProductID: p, AcceptedQty: 8}},
			}, userID, &store)
			require.NoError(t, err)
		}

		prD, err := svc.CreatePendingReturn(ctx, &CreatePendingReturnRequest{ProductID: skuD, Qty: 2, Reason: ReasonDamaged}, userID, &store)
		require.NoError(t, err)
		require.Equal(t, ReasonDamaged, prD.Reason)

		prE, err := svc.CreatePendingReturn(ctx, &CreatePendingReturnRequest{ProductID: skuE, Qty: 3, Reason: ReasonExpired}, userID, &store)
		require.NoError(t, err)
		require.Equal(t, ReasonExpired, prE.Reason)

		// Both still list as open pending returns.
		list, err := svc.ListPendingReturns(ctx, sup, &store)
		require.NoError(t, err)
		require.Len(t, list, 2)
	})

	t.Run("EC-09 return does not create a settlement", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "PR-EC09")
		svc, sup, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		_, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 8}},
		}, userID, &store)
		require.NoError(t, err)

		// Supplier asks for unsold goods back: full return from available.
		ret, err := svc.CreateReturn(ctx, &ReturnRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReturnItemRequest{{ProductID: product, Qty: 8, Reason: ReasonOther}},
		}, userID, &store)
		require.NoError(t, err)
		require.NotEmpty(t, ret.ReturnNumber)

		// No sale happened, so a settlement preview must be empty.
		_, err = svc.GetSettlementPreview(ctx, sup, &store)
		require.ErrorIs(t, err, ErrEmptySettlement)
	})

	t.Run("EC-10 eligible customer return becomes pending return", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "PR-EC10")
		svc, _, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		_, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 10}},
		}, userID, &store)
		require.NoError(t, err)

		// Customer returns a damaged good: pending return to supplier, NOT
		// back to available stock.
		pr, err := svc.CreatePendingReturn(ctx, &CreatePendingReturnRequest{
			ProductID: product,
			Qty:       2,
			Reason:    ReasonCustomerReturn,
		}, userID, &store)
		require.NoError(t, err)
		require.Equal(t, ReasonCustomerReturn, pr.Reason)

		row, err := svc.repo.GetConsignmentStock(ctx, svc.repo.db, product)
		require.NoError(t, err)
		require.Equal(t, 8, row.AvailableQty)
		require.Equal(t, 2, row.PendingReturnQty)
		require.Equal(t, 8, globalStockQty(ctx, t, product))
	})

	t.Run("rejects pending return for non-consignment product", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "PR-NONE")
		svc, _, store := setupArrangement(t)
		userID := insertTestUser(ctx, t)

		_, err := svc.CreatePendingReturn(ctx, &CreatePendingReturnRequest{
			ProductID: product,
			Qty:       2,
			Reason:    ReasonDamaged,
		}, userID, &store)
		require.ErrorIs(t, err, ErrConsignmentNotFound)
	})
}

func TestService_Return(t *testing.T) {
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	t.Run("free return reduces available and global stock", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "RET-FREE")
		svc, _, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		_, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 10}},
		}, userID, &store)
		require.NoError(t, err)

		ret, err := svc.CreateReturn(ctx, &ReturnRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReturnItemRequest{{ProductID: product, Qty: 3, Reason: ReasonOther}},
		}, userID, &store)
		require.NoError(t, err)
		require.NotEmpty(t, ret.ReturnNumber)
		require.Len(t, ret.Items, 1)

		row, err := svc.repo.GetConsignmentStock(ctx, svc.repo.db, product)
		require.NoError(t, err)
		require.Equal(t, 7, row.AvailableQty)
		require.Equal(t, 7, globalStockQty(ctx, t, product))
	})

	t.Run("pending return resolved by formal return", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "RET-PR")
		svc, sup, store := setupArrangement(t, product)
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

		prID := pr.ID
		ret, err := svc.CreateReturn(ctx, &ReturnRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReturnItemRequest{{ProductID: product, Qty: 4, Reason: ReasonDamaged, PendingReturnID: &prID}},
		}, userID, &store)
		require.NoError(t, err)
		require.Len(t, ret.Items, 1)
		require.NotNil(t, ret.Items[0].PendingReturnID)

		// pending_return drained; available untouched by this path.
		row, err := svc.repo.GetConsignmentStock(ctx, svc.repo.db, product)
		require.NoError(t, err)
		require.Equal(t, 6, row.AvailableQty)
		require.Equal(t, 0, row.PendingReturnQty)

		// pending return no longer open.
		list, err := svc.ListPendingReturns(ctx, sup, &store)
		require.NoError(t, err)
		require.Empty(t, list)

		// Global stock reflects the removal of returned goods (6 soldable left).
		require.Equal(t, 6, globalStockQty(ctx, t, product))
	})

	t.Run("return of pending return qty beyond PR rejected", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "RET-PR-OVER")
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

		prID := pr.ID
		_, err = svc.CreateReturn(ctx, &ReturnRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReturnItemRequest{{ProductID: product, Qty: 5, Reason: ReasonDamaged, PendingReturnID: &prID}},
		}, userID, &store)
		require.ErrorIs(t, err, ErrPendingReturnNotFound)
	})

	t.Run("partial return keeps pending return open with leftover qty", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "RET-PR-PARTIAL")
		svc, sup, store := setupArrangement(t, product)
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

		prID := pr.ID
		// Return only 3 of the 4 pulled units.
		ret, err := svc.CreateReturn(ctx, &ReturnRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReturnItemRequest{{ProductID: product, Qty: 3, Reason: ReasonDamaged, PendingReturnID: &prID}},
		}, userID, &store)
		require.NoError(t, err)
		require.Len(t, ret.Items, 1)

		// 1 unit stays in pending_return and the PR stays open (not orphaned).
		row, err := svc.repo.GetConsignmentStock(ctx, svc.repo.db, product)
		require.NoError(t, err)
		require.Equal(t, 6, row.AvailableQty)
		require.Equal(t, 1, row.PendingReturnQty)

		list, err := svc.ListPendingReturns(ctx, sup, &store)
		require.NoError(t, err)
		require.Len(t, list, 1)
		require.Equal(t, 1, list[0].Qty)
		require.Equal(t, PendingReturnOpen, list[0].Status)

		// The leftover can be returned later in full, closing the PR.
		ret2, err := svc.CreateReturn(ctx, &ReturnRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReturnItemRequest{{ProductID: product, Qty: 1, Reason: ReasonDamaged, PendingReturnID: &prID}},
		}, userID, &store)
		require.NoError(t, err)
		require.Len(t, ret2.Items, 1)

		row, err = svc.repo.GetConsignmentStock(ctx, svc.repo.db, product)
		require.NoError(t, err)
		require.Equal(t, 6, row.AvailableQty)
		require.Equal(t, 0, row.PendingReturnQty)

		list, err = svc.ListPendingReturns(ctx, sup, &store)
		require.NoError(t, err)
		require.Empty(t, list)
	})

	t.Run("EC-07 customer-damaged goods stay with store", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "RET-EC07")
		svc, _, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		_, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 10}},
		}, userID, &store)
		require.NoError(t, err)

		// Customer damage is the store's responsibility: it is NOT a pending
		// return. Only available stock shrinks (the store absorbs the loss).
		row, err := svc.repo.GetConsignmentStock(ctx, svc.repo.db, product)
		require.NoError(t, err)
		require.Equal(t, 10, row.AvailableQty)
		require.Equal(t, 0, row.PendingReturnQty)
		require.Equal(t, 10, globalStockQty(ctx, t, product))
	})

	t.Run("return of product from another supplier rejected", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "RET-OTHER")
		svcA, _, storeA := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)
		_, err := svcA.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svcA, storeA),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 5}},
		}, userID, &storeA)
		require.NoError(t, err)

		svcB, _, storeB, _ := setupArrangementNoTerms(t)
		_, err = svcB.CreateReturn(ctx, &ReturnRequest{
			ArrangementID: arrID(t, svcB, storeB),
			Items:         []ReturnItemRequest{{ProductID: product, Qty: 2, Reason: ReasonOther}},
		}, userID, &storeB)
		require.ErrorIs(t, err, ErrConflictOtherSupplier)
	})

	t.Run("full return releases ownership for new supplier", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "RET-FULL")
		svcA, _, storeA := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)
		_, err := svcA.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svcA, storeA),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 5}},
		}, userID, &storeA)
		require.NoError(t, err)

		_, err = svcA.CreateReturn(ctx, &ReturnRequest{
			ArrangementID: arrID(t, svcA, storeA),
			Items:         []ReturnItemRequest{{ProductID: product, Qty: 5, Reason: ReasonOther}},
		}, userID, &storeA)
		require.NoError(t, err)

		row, err := svcA.repo.GetConsignmentStock(ctx, svcA.repo.db, product)
		require.NoError(t, err)
		require.Nil(t, row)
	})
}

func TestService_ListWithNilStore(t *testing.T) {
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	t.Run("ListReceipts with nil store returns all stores", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "LIST-RCPT")
		svc, sup, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		_, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 5}},
		}, userID, &store)
		require.NoError(t, err)

		// Store-scoped user sees their receipts.
		scoped, err := svc.ListReceipts(ctx, sup, &store)
		require.NoError(t, err)
		require.Len(t, scoped, 1)

		// Admin (nil store) also sees them.
		admin, err := svc.ListReceipts(ctx, sup, nil)
		require.NoError(t, err)
		require.Len(t, admin, 1)
	})

	t.Run("ListPendingReturns with nil store returns all stores", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "LIST-PR")
		svc, sup, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		_, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 10}},
		}, userID, &store)
		require.NoError(t, err)

		_, err = svc.CreatePendingReturn(ctx, &CreatePendingReturnRequest{
			ProductID: product, Qty: 3, Reason: ReasonDamaged,
		}, userID, &store)
		require.NoError(t, err)

		scoped, err := svc.ListPendingReturns(ctx, sup, &store)
		require.NoError(t, err)
		require.Len(t, scoped, 1)

		admin, err := svc.ListPendingReturns(ctx, sup, nil)
		require.NoError(t, err)
		require.Len(t, admin, 1)
	})

	t.Run("ListReturns with nil store returns all stores", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "LIST-RET")
		svc, sup, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		_, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 5}},
		}, userID, &store)
		require.NoError(t, err)

		_, err = svc.CreateReturn(ctx, &ReturnRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReturnItemRequest{{ProductID: product, Qty: 2, Reason: ReasonOther}},
		}, userID, &store)
		require.NoError(t, err)

		scoped, err := svc.ListReturns(ctx, sup, &store)
		require.NoError(t, err)
		require.Len(t, scoped, 1)

		admin, err := svc.ListReturns(ctx, sup, nil)
		require.NoError(t, err)
		require.Len(t, admin, 1)
	})
}
