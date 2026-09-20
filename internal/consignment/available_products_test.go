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

// TestService_SearchAvailableProducts verifies the search-based product
// assignment endpoint returns only eligible products and correctly indicates
// exact name matches.
func TestService_SearchAvailableProducts(t *testing.T) {
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	t.Run("returns matching products by name", func(t *testing.T) {
		svc, _, store, _ := setupArrangementNoTerms(t)

		prodA := insertTestProduct(ctx, t, "SRCH-ALPHA")
		prodB := insertTestProduct(ctx, t, "SRCH-BETA")

		results, exactMatch, err := svc.SearchAvailableProducts(ctx, arrID(t, svc, store), "SRCH", &store)
		require.NoError(t, err)
		require.False(t, exactMatch)

		ids := make(map[int]bool, len(results))
		for _, r := range results {
			ids[r.ID] = true
		}
		require.True(t, ids[prodA], "product A should be in results")
		require.True(t, ids[prodB], "product B should be in results")
	})

	t.Run("exact match flag set when name matches exactly", func(t *testing.T) {
		svc, _, store, _ := setupArrangementNoTerms(t)

		insertTestProduct(ctx, t, "EXACT-UNIQUE")

		results, exactMatch, err := svc.SearchAvailableProducts(ctx, arrID(t, svc, store), "Test Product EXACT-UNIQUE", &store)
		require.NoError(t, err)
		require.True(t, exactMatch, "exactMatch should be true when product name matches exactly")
		require.NotEmpty(t, results)
	})

	t.Run("excludes term'ed products", func(t *testing.T) {
		svc, _, store, userID := setupArrangementNoTerms(t)
		storeP := store

		termed := insertTestProduct(ctx, t, "SRCH-TERMED")
		_, err := svc.SetTerms(ctx, arrID(t, svc, store), []SetTermsRequest{
			{ProductID: termed, Price: 10000, StoreShareType: ShareTypePercentage, StoreShareValue: 20},
		}, userID, &storeP)
		require.NoError(t, err)

		results, _, err := svc.SearchAvailableProducts(ctx, arrID(t, svc, store), "SRCH-TERMED", &storeP)
		require.NoError(t, err)
		for _, r := range results {
			require.NotEqual(t, termed, r.ID, "term'ed product should be excluded from search results")
		}
	})

	t.Run("excludes store-owned products with stock", func(t *testing.T) {
		svc, _, store, _ := setupArrangementNoTerms(t)

		storeOwned := insertTestProduct(ctx, t, "SRCH-OWNED")
		seedStoreOwnedStock(ctx, t, storeOwned, 5)

		results, _, err := svc.SearchAvailableProducts(ctx, arrID(t, svc, store), "SRCH-OWNED", &store)
		require.NoError(t, err)
		for _, r := range results {
			require.NotEqual(t, storeOwned, r.ID, "store-owned product with stock should be excluded")
		}
	})

	t.Run("excludes products with live consignment stock under another supplier", func(t *testing.T) {
		svc, _, store, _ := setupArrangementNoTerms(t)

		liveProduct := insertTestProduct(ctx, t, "SRCH-LIVE")
		seedOtherSupplierStock(t, liveProduct, store)

		results, _, err := svc.SearchAvailableProducts(ctx, arrID(t, svc, store), "SRCH-LIVE", &store)
		require.NoError(t, err)
		for _, r := range results {
			require.NotEqual(t, liveProduct, r.ID, "product with other-supplier stock should be excluded")
		}
	})

	t.Run("returns empty for ended arrangement", func(t *testing.T) {
		svc, _, store, _ := setupArrangementNoTerms(t)
		insertTestProduct(ctx, t, "SRCH-ENDED")

		_, err := dbPool.Exec(ctx, `UPDATE consignment_arrangements SET status = 'ended', ended_at = NOW() WHERE id = $1`, arrID(t, svc, store))
		require.NoError(t, err)

		results, _, err := svc.SearchAvailableProducts(ctx, arrID(t, svc, store), "SRCH-ENDED", &store)
		require.NoError(t, err)
		require.Empty(t, results)
	})

	t.Run("store scope enforced", func(t *testing.T) {
		svc, _, store, _ := setupArrangementNoTerms(t)
		insertTestProduct(ctx, t, "SRCH-SCOPE")

		otherStore := insertTestStore(ctx, t)
		_, _, err := svc.SearchAvailableProducts(ctx, arrID(t, svc, store), "SRCH-SCOPE", &otherStore)
		require.ErrorIs(t, err, ErrStoreForbidden)
	})
}
