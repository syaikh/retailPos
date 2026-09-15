package consignment

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

// seedOtherSupplierStock creates a consignment_stock row for productID under a
// supplier different from the caller's arrangement supplier.  The receipt
// creates both the consignment_stock ledger and the global product_stock qty,
// exactly matching a real received-goods flow.
func seedOtherSupplierStock(t *testing.T, productID, store int) {
	t.Helper()
	ctx := context.Background()
	otherSvc, _, otherStore := setupArrangement(t, productID)
	otherUser := insertTestUser(ctx, t)
	_, err := otherSvc.CreateReceipt(ctx, &ReceiptRequest{
		ArrangementID: arrID(t, otherSvc, otherStore),
		Items:         []ReceiptItemRequest{{ProductID: productID, AcceptedQty: 5}},
	}, otherUser, &otherStore)
	require.NoError(t, err)
}

// TestService_ListAddTermProductOptions pins the eligible-products picker for
// the add-term modal: products already covered by a term, store-owned products
// with remaining global stock, and products with live consignment stock under
// another supplier must not be offered, while free products must.
func TestService_ListAddTermProductOptions(t *testing.T) {
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	t.Run("excludes term'ed, store-owned, and other-supplier-live products", func(t *testing.T) {
		svc, _, store, userID := setupArrangementNoTerms(t)
		storeP := store

		free := insertTestProduct(ctx, t, "AVL-FREE")
		termed := insertTestProduct(ctx, t, "AVL-TERMED")
		storeOwned := insertTestProduct(ctx, t, "AVL-OWNED")
		seedStoreOwnedStock(ctx, t, storeOwned, 5)

		_, err := svc.SetTerms(ctx, arrID(t, svc, store), []SetTermsRequest{
			{ProductID: termed, Price: 10000, StoreShareType: ShareTypePercentage, StoreShareValue: 20},
		}, userID, &storeP)
		require.NoError(t, err)

		// Live consignment stock under a DIFFERENT supplier than the test's.
		liveProduct := insertTestProduct(ctx, t, "AVL-LIVE")
		seedOtherSupplierStock(t, liveProduct, store)

		options, err := svc.ListAddTermProductOptions(ctx, arrID(t, svc, store), &storeP)
		require.NoError(t, err)

		ids := make(map[int]bool, len(options))
		for _, o := range options {
			ids[o.ID] = true
		}
		require.True(t, ids[free], "free product should be offered")
		require.False(t, ids[termed], "already-term'ed product should be excluded")
		require.False(t, ids[storeOwned], "store-owned product with stock should be excluded")
		require.False(t, ids[liveProduct], "product with live consignment stock under another supplier should be excluded")
	})

	t.Run("ended arrangement offers nothing", func(t *testing.T) {
		svc, _, store, _ := setupArrangementNoTerms(t)
		storeP := store
		arrID := arrID(t, svc, store)
		_, err := dbPool.Exec(ctx, `UPDATE consignment_arrangements SET status = 'ended', ended_at = NOW() WHERE id = $1`, arrID)
		require.NoError(t, err)

		options, err := svc.ListAddTermProductOptions(ctx, arrID, &storeP)
		require.NoError(t, err)
		require.Empty(t, options)
	})
}
