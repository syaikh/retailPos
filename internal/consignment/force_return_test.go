package consignment

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

// TestService_ForceReturn covers the store-sovereign escape hatch (§15.9):
// releasing every remaining unit of ownership for an arrangement in one formal
// return, without touching unsettled sales and without ending the arrangement.
func TestService_ForceReturn(t *testing.T) {
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	t.Run("releases available stock and open pending returns in one document", func(t *testing.T) {
		productA := insertTestProduct(ctx, t, "FRC-A")
		productB := insertTestProduct(ctx, t, "FRC-B")
		svc, _, store := setupArrangement(t, productA, productB)
		userID := insertTestUser(ctx, t)

		_, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items: []ReceiptItemRequest{
				{ProductID: productA, AcceptedQty: 10},
				{ProductID: productB, AcceptedQty: 7},
			},
		}, userID, &store)
		require.NoError(t, err)

		// Pull part of productA into a pending return: available and pending now
		// legitimately coexist for the same product, which is exactly the shape
		// that used to trip the duplicate-product guard in CreateReturn.
		_, err = svc.CreatePendingReturn(ctx, &CreatePendingReturnRequest{
			ProductID: productA, Qty: 4, Reason: ReasonOther,
		}, userID, &store)
		require.NoError(t, err)

		ret, err := svc.ForceReturn(ctx, arrID(t, svc, store), userID, &store)
		require.NoError(t, err)
		require.NotNil(t, ret)

		// Two items for productA (6 available + 4 pending) and one for productB.
		require.Len(t, ret.Items, 3)
		total := 0
		for _, it := range ret.Items {
			require.Equal(t, ReasonTermination, it.Reason)
			total += it.Qty
		}
		require.Equal(t, 17, total)

		// Ownership fully released: no consignment stock row remains.
		row, err := svc.repo.GetConsignmentStock(ctx, svc.repo.db, productA)
		require.NoError(t, err)
		require.Nil(t, row)
		row, err = svc.repo.GetConsignmentStock(ctx, svc.repo.db, productB)
		require.NoError(t, err)
		require.Nil(t, row)

		// No open pending returns left behind.
		pending, err := svc.repo.ListOpenPendingReturnsByArrangement(ctx, svc.repo.db, arrID(t, svc, store))
		require.NoError(t, err)
		require.Empty(t, pending)
	})

	t.Run("does not end the arrangement", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "FRC-KEEP")
		svc, _, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)
		id := arrID(t, svc, store)

		_, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: id,
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 3}},
		}, userID, &store)
		require.NoError(t, err)

		_, err = svc.ForceReturn(ctx, id, userID, &store)
		require.NoError(t, err)

		a, err := svc.GetArrangement(ctx, id, &store)
		require.NoError(t, err)
		require.Equal(t, StatusActive, a.Status)
	})

	t.Run("released SKU can be claimed by another consignment supplier", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "FRC-RELEASE")
		svcA, _, storeA := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		_, err := svcA.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svcA, storeA),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 5}},
		}, userID, &storeA)
		require.NoError(t, err)
		_, err = svcA.ForceReturn(ctx, arrID(t, svcA, storeA), userID, &storeA)
		require.NoError(t, err)

		row, err := svcA.repo.GetConsignmentStock(ctx, svcA.repo.db, product)
		require.NoError(t, err)
		require.Nil(t, row)

		// And the release is real: a second consignment supplier can now set
		// terms for the SKU and take ownership of it with its own receipt.
		svcB, supplierB, storeB := setupArrangement(t, product)
		_, err = svcB.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svcB, storeB),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 3}},
		}, userID, &storeB)
		require.NoError(t, err)

		row, err = svcB.repo.GetConsignmentStock(ctx, svcB.repo.db, product)
		require.NoError(t, err)
		require.NotNil(t, row, "the released SKU must be claimable by another supplier")
		require.Equal(t, supplierB, row.SupplierID)
		require.Equal(t, 3, row.AvailableQty)
		require.Equal(t, 3, globalStockQty(ctx, t, product))
	})

	t.Run("returns ErrNothingToReturn when nothing is owned", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "FRC-EMPTY")
		svc, _, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		_, err := svc.ForceReturn(ctx, arrID(t, svc, store), userID, &store)
		require.ErrorIs(t, err, ErrNothingToReturn)
	})

	t.Run("rejects an ended arrangement", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "FRC-ENDED")
		svc, _, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)
		id := arrID(t, svc, store)

		_, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: id,
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 3}},
		}, userID, &store)
		require.NoError(t, err)
		_, err = svc.ForceReturn(ctx, id, userID, &store)
		require.NoError(t, err)
		_, err = svc.EndArrangement(ctx, id, &store)
		require.NoError(t, err)

		_, err = svc.ForceReturn(ctx, id, userID, &store)
		require.ErrorIs(t, err, ErrArrangementEnded)
	})

	t.Run("enforces the store claim scope", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "FRC-SCOPE")
		svc, _, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		_, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 3}},
		}, userID, &store)
		require.NoError(t, err)

		other := insertTestStore(ctx, t)
		_, err = svc.ForceReturn(ctx, arrID(t, svc, store), userID, &other)
		require.ErrorIs(t, err, ErrStoreForbidden)
	})
}
