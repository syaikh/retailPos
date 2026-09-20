package consignment

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

func TestService_ArrangementLifecycle(t *testing.T) {
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	t.Run("create rejects missing store for superadmin", func(t *testing.T) {
		userID := insertTestUser(ctx, t)
		sup := insertTestSupplier(ctx, t, "NoStore Supplier", true)

		svc := newTestService(t)
		_, err := svc.CreateArrangement(ctx, &CreateArrangementRequest{SupplierID: sup, StoreID: 0}, userID, nil)
		require.ErrorIs(t, err, ErrStoreRequired)
	})

	t.Run("create rejects non-consignment supplier", func(t *testing.T) {
		userID := insertTestUser(ctx, t)
		storeID := insertTestStore(ctx, t)
		normal := insertTestSupplier(ctx, t, "Normal Supplier", false)

		svc := newTestService(t)
		_, err := svc.CreateArrangement(ctx, &CreateArrangementRequest{SupplierID: normal, StoreID: storeID}, userID, nil)
		require.ErrorIs(t, err, ErrNotConsignmentSupplier)
	})

	t.Run("create rejects duplicate active arrangement", func(t *testing.T) {
		userID := insertTestUser(ctx, t)
		storeID := insertTestStore(ctx, t)
		sup := insertTestSupplier(ctx, t, "Dup Supplier", true)

		svc := newTestService(t)
		_, err := svc.CreateArrangement(ctx, &CreateArrangementRequest{SupplierID: sup, StoreID: storeID}, userID, nil)
		require.NoError(t, err)
		_, err = svc.CreateArrangement(ctx, &CreateArrangementRequest{SupplierID: sup, StoreID: storeID}, userID, nil)
		require.ErrorIs(t, err, ErrActiveArrangementExists)
	})

	t.Run("store scope enforced", func(t *testing.T) {
		userID := insertTestUser(ctx, t)
		storeA := insertTestStore(ctx, t)
		storeB := insertTestStore(ctx, t)
		sup := insertTestSupplier(ctx, t, "Scope Supplier", true)

		svc := newTestService(t)
		arr, err := svc.CreateArrangement(ctx, &CreateArrangementRequest{SupplierID: sup, StoreID: storeA}, userID, nil)
		require.NoError(t, err)

		claimsB := storeB
		_, err = svc.GetArrangement(ctx, arr.ID, &claimsB)
		require.ErrorIs(t, err, ErrStoreForbidden)

		// Listing from the owner store sees the arrangement.
		claimsA := storeA
		arrs, _, err := svc.ListArrangements(ctx, &claimsA, 0, 0, "", "")
		require.NoError(t, err)
		require.NotEmpty(t, arrs)
	})

	t.Run("get arrangement loads terms", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "ARR-TERMS-PROD")
		svc, sup, store := setupArrangement(t, product)

		arrs, _, err := svc.ListArrangements(ctx, &store, 0, 0, "", "")
		require.NoError(t, err)
		require.Len(t, arrs, 1)
		require.Equal(t, sup, arrs[0].SupplierID)
		require.Equal(t, StatusActive, arrs[0].Status)
	})

	t.Run("get arrangement hydrates term product names", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "ARR-TERM-HYD")
		svc, _, store := setupArrangement(t, product)

		arrs, _, err := svc.ListArrangements(ctx, &store, 0, 0, "", "")
		require.NoError(t, err)

		got, err := svc.GetArrangement(ctx, arrs[0].ID, &store)
		require.NoError(t, err)
		require.Len(t, got.Terms, 1)
		require.NotEmpty(t, got.Terms[0].ProductSKU, "Term ProductSKU should be hydrated")
		require.NotEmpty(t, got.Terms[0].ProductName, "Term ProductName should be hydrated")
	})
}

func TestService_LazyEnded(t *testing.T) {
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	t.Run("stale visit derives ended on read", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "LAZY-ENDED-PROD")
		svc, _, store := setupArrangement(t, product)

		arrs, _, err := svc.ListArrangements(ctx, &store, 0, 0, "", "")
		require.NoError(t, err)
		require.Len(t, arrs, 1)
		arr := arrs[0]

		// Force the last visit into the past (beyond the 14-day window).
		_, err = dbPool.Exec(ctx, `
			UPDATE consignment_arrangements
			SET last_visit_at = now() - interval '15 days'
			WHERE id = $1
		`, arr.ID)
		require.NoError(t, err)

		got, err := svc.GetArrangement(ctx, arr.ID, &store)
		require.NoError(t, err)
		require.Equal(t, StatusEnded, got.Status)
	})

	t.Run("fresh visit stays active", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "FRESH-PROD")
		svc, _, store := setupArrangement(t, product)

		arrs, _, err := svc.ListArrangements(ctx, &store, 0, 0, "", "")
		require.NoError(t, err)
		got, err := svc.GetArrangement(ctx, arrs[0].ID, &store)
		require.NoError(t, err)
		require.Equal(t, StatusActive, got.Status)
	})

	t.Run("ended arrangement rejects terms change", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "ENDED-TERMS-PROD")
		svc, _, store := setupArrangement(t, product)

		arrs, _, err := svc.ListArrangements(ctx, &store, 0, 0, "", "")
		require.NoError(t, err)
		_, err = dbPool.Exec(ctx, `
			UPDATE consignment_arrangements SET last_visit_at = now() - interval '15 days' WHERE id = $1
		`, arrs[0].ID)
		require.NoError(t, err)

		userID := insertTestUser(ctx, t)
		_, err = svc.SetTerms(ctx, arrs[0].ID, []SetTermsRequest{
			{ProductID: product, Price: 11000, StoreShareType: ShareTypePercentage, StoreShareValue: 25},
		}, userID, &store)
		require.ErrorIs(t, err, ErrArrangementEnded)
	})

	t.Run("supplier visit refreshes last_visit_at", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "VISIT-PROD")
		svc, _, store := setupArrangement(t, product)

		arrs, _, err := svc.ListArrangements(ctx, &store, 0, 0, "", "")
		require.NoError(t, err)
		arrID := arrs[0].ID

		_, err = dbPool.Exec(ctx, `UPDATE consignment_arrangements SET last_visit_at = now() - interval '15 days' WHERE id = $1`, arrID)
		require.NoError(t, err)

		// A receipt touches the visit (the receipt path calls TouchVisit).
		userID := insertTestUser(ctx, t)
		rec, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID,
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 5}},
		}, userID, &store)
		require.NoError(t, err)
		require.NotEmpty(t, rec.ReceiptNumber)

		got, err := svc.GetArrangement(ctx, arrID, &store)
		require.NoError(t, err)
		require.Equal(t, StatusActive, got.Status)
		require.NotEmpty(t, got.SupplierName, "SupplierName should be hydrated on GetArrangement")
		require.NotEmpty(t, got.StoreName, "StoreName should be hydrated on GetArrangement")

		lastVisit, err := time.Parse(time.RFC3339, *got.LastVisitAt)
		require.NoError(t, err)
		require.WithinDuration(t, time.Now(), lastVisit, time.Minute)
	})
}

func TestService_SetTermsValidation(t *testing.T) {
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)
	userID := insertTestUser(ctx, t)

	t.Run("invalid share type rejected", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "TERMS-TYPE-PROD")
		svc, _, store := setupArrangement(t, product)
		arrs, _, _ := svc.ListArrangements(ctx, &store, 0, 0, "", "")

		_, err := svc.SetTerms(ctx, arrs[0].ID, []SetTermsRequest{
			{ProductID: product, Price: 10000, StoreShareType: "flat", StoreShareValue: 1000},
		}, userID, &store)
		require.ErrorIs(t, err, ErrInvalidShareType)
	})

	t.Run("percentage >= 100 rejected", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "TERMS-PCT-PROD")
		svc, _, store := setupArrangement(t, product)
		arrs, _, _ := svc.ListArrangements(ctx, &store, 0, 0, "", "")

		_, err := svc.SetTerms(ctx, arrs[0].ID, []SetTermsRequest{
			{ProductID: product, Price: 10000, StoreShareType: ShareTypePercentage, StoreShareValue: 100},
		}, userID, &store)
		require.ErrorIs(t, err, ErrInvalidShareValueForType)
	})

	t.Run("zero share value rejected", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "TERMS-ZERO-PROD")
		svc, _, store := setupArrangement(t, product)
		arrs, _, _ := svc.ListArrangements(ctx, &store, 0, 0, "", "")

		_, err := svc.SetTerms(ctx, arrs[0].ID, []SetTermsRequest{
			{ProductID: product, Price: 10000, StoreShareType: ShareTypeFixedAmount, StoreShareValue: 0},
		}, userID, &store)
		require.ErrorIs(t, err, ErrInvalidShareValue)
	})

	t.Run("product with store-owned stock rejected", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "TERMS-STORE-STOCK")
		svc, _, store := setupArrangement(t, product)
		arrs, _, _ := svc.ListArrangements(ctx, &store, 0, 0, "", "")
		seedStoreOwnedStock(ctx, t, product, 5)

		_, err := svc.SetTerms(ctx, arrs[0].ID, []SetTermsRequest{
			{ProductID: product, Price: 10000, StoreShareType: ShareTypePercentage, StoreShareValue: 20},
		}, userID, &store)
		require.ErrorIs(t, err, ErrConflictStoreStock)
	})

	t.Run("product consigned to another supplier rejected", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "TERMS-OTHER-SUP")
		svcA, _, storeA := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		// Supplier A receives stock.
		_, err := svcA.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svcA, storeA),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 5}},
		}, userID, &storeA)
		require.NoError(t, err)

		// Supplier B creates arrangement (no terms yet).
		supplierB := insertTestSupplier(ctx, t, "Supplier B", true)
		storeB := insertTestStore(ctx, t)
		svcB := newTestService(t)
		arrB, err := svcB.CreateArrangement(ctx, &CreateArrangementRequest{SupplierID: supplierB, StoreID: storeB}, userID, nil)
		require.NoError(t, err)

		// Supplier B tries to set terms on the same product.
		_, err = svcB.SetTerms(ctx, arrB.ID, []SetTermsRequest{
			{ProductID: product, Price: 12000, StoreShareType: ShareTypePercentage, StoreShareValue: 30},
		}, userID, &storeB)
		require.ErrorIs(t, err, ErrConflictOtherSupplier)
	})

	t.Run("zero price rejected", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "TERMS-ZERO-PRICE")
		svc, _, store := setupArrangement(t, product)
		arrs, _, _ := svc.ListArrangements(ctx, &store, 0, 0, "", "")

		_, err := svc.SetTerms(ctx, arrs[0].ID, []SetTermsRequest{
			{ProductID: product, Price: 0, StoreShareType: ShareTypePercentage, StoreShareValue: 20},
		}, userID, &store)
		require.ErrorIs(t, err, ErrInvalidPrice)
	})

	t.Run("fixed_amount share >= price rejected", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "TERMS-SHARE-OVER")
		svc, _, store := setupArrangement(t, product)
		arrs, _, _ := svc.ListArrangements(ctx, &store, 0, 0, "", "")

		_, err := svc.SetTerms(ctx, arrs[0].ID, []SetTermsRequest{
			{ProductID: product, Price: 10000, StoreShareType: ShareTypeFixedAmount, StoreShareValue: 10000},
		}, userID, &store)
		require.ErrorIs(t, err, ErrFixedShareExceedsPrice)
	})

	t.Run("duplicate product rejected", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "TERMS-DUP-PROD")
		svc, _, store := setupArrangement(t, product)
		arrs, _, _ := svc.ListArrangements(ctx, &store, 0, 0, "", "")

		_, err := svc.SetTerms(ctx, arrs[0].ID, []SetTermsRequest{
			{ProductID: product, Price: 10000, StoreShareType: ShareTypePercentage, StoreShareValue: 20},
			{ProductID: product, Price: 12000, StoreShareType: ShareTypeFixedAmount, StoreShareValue: 3000},
		}, userID, &store)
		require.ErrorIs(t, err, ErrDuplicateProduct)
	})

	t.Run("negative price rejected", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "TERMS-NEG-PRICE")
		svc, _, store := setupArrangement(t, product)
		arrs, _, _ := svc.ListArrangements(ctx, &store, 0, 0, "", "")

		_, err := svc.SetTerms(ctx, arrs[0].ID, []SetTermsRequest{
			{ProductID: product, Price: -1000, StoreShareType: ShareTypePercentage, StoreShareValue: 20},
		}, userID, &store)
		require.ErrorIs(t, err, ErrInvalidPrice)
	})

	t.Run("fixed_amount share less than price accepted", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "TERMS-FIXED-OK")
		svc, _, store := setupArrangement(t, product)
		arrs, _, _ := svc.ListArrangements(ctx, &store, 0, 0, "", "")

		terms, err := svc.SetTerms(ctx, arrs[0].ID, []SetTermsRequest{
			{ProductID: product, Price: 10000, StoreShareType: ShareTypeFixedAmount, StoreShareValue: 5000},
		}, userID, &store)
		require.NoError(t, err)
		require.Len(t, terms, 1)
		require.Equal(t, ShareTypeFixedAmount, terms[0].StoreShareType)
		require.Equal(t, 5000.0, terms[0].StoreShareValue)
		require.NotEmpty(t, terms[0].ProductSKU, "ProductSKU should be hydrated after SetTerms")
		require.NotEmpty(t, terms[0].ProductName, "ProductName should be hydrated after SetTerms")
	})

	t.Run("consignment ledger prevents store-owned conflict", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "TERMS-LEDGER-OK")
		svc, _, store := setupArrangement(t, product)
		arrs, _, _ := svc.ListArrangements(ctx, &store, 0, 0, "", "")

		// Receipt creates both the consignment_stock ledger AND product_stock qty.
		_, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrs[0].ID,
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 5}},
		}, userID, &store)
		require.NoError(t, err)

		// hasStoreOwnedStock must return false because the ledger exists,
		// even though product_stock qty > 0.
		terms, err := svc.SetTerms(ctx, arrs[0].ID, []SetTermsRequest{
			{ProductID: product, Price: 12000, StoreShareType: ShareTypePercentage, StoreShareValue: 25},
		}, userID, &store)
		require.NoError(t, err)
		require.Len(t, terms, 1)
	})

	t.Run("terms change replaces previous terms", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "TERMS-REPLACE-PROD")
		svc, _, store := setupArrangement(t, product)
		arrs, _, _ := svc.ListArrangements(ctx, &store, 0, 0, "", "")

		_, err := svc.SetTerms(ctx, arrs[0].ID, []SetTermsRequest{
			{ProductID: product, Price: 12000, StoreShareType: ShareTypeFixedAmount, StoreShareValue: 2500},
		}, userID, &store)
		require.NoError(t, err)

		got, err := svc.GetArrangement(ctx, arrs[0].ID, &store)
		require.NoError(t, err)
		require.Len(t, got.Terms, 1)
		require.Equal(t, 12000, got.Terms[0].Price)
		require.Equal(t, ShareTypeFixedAmount, got.Terms[0].StoreShareType)
		require.Equal(t, 2500.0, got.Terms[0].StoreShareValue)
	})

	t.Run("receipt without a term is rejected", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "TERMS-MISSING-PROD")
		svc, sup, store := setupArrangement(t, insertTestProduct(ctx, t, "TERMS-OTHER-PROD"))
		_ = sup

		user := insertTestUser(ctx, t)
		arrs, _, _ := svc.ListArrangements(ctx, &store, 0, 0, "", "")
		_, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrs[0].ID,
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 2}},
		}, user, &store)
		require.Error(t, err)
	})
}

func TestService_ListArrangementsPaginationSearch(t *testing.T) {
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	userID := insertTestUser(ctx, t)
	storeID := insertTestStore(ctx, t)
	svc := newTestService(t)

	sups := []string{"PT Sumber Makmur", "PT Sumber Rejeki", "CV Berkah Abadi"}
	for _, name := range sups {
		sup := insertTestSupplier(ctx, t, name, true)
		_, err := svc.CreateArrangement(ctx, &CreateArrangementRequest{SupplierID: sup, StoreID: storeID}, userID, nil)
		require.NoError(t, err)
	}

	claims := storeID

	t.Run("lists all when no filters", func(t *testing.T) {
		arrs, total, err := svc.ListArrangements(ctx, &claims, 0, 0, "", "")
		require.NoError(t, err)
		require.Equal(t, 3, total)
		require.Len(t, arrs, 3)
	})

	t.Run("paginates with limit and offset", func(t *testing.T) {
		arrs, total, err := svc.ListArrangements(ctx, &claims, 2, 0, "", "")
		require.NoError(t, err)
		require.Equal(t, 3, total)
		require.Len(t, arrs, 2)

		next, _, err := svc.ListArrangements(ctx, &claims, 2, 2, "", "")
		require.NoError(t, err)
		require.Len(t, next, 1)
	})

	t.Run("searches by supplier name", func(t *testing.T) {
		arrs, total, err := svc.ListArrangements(ctx, &claims, 0, 0, "sumber", "")
		require.NoError(t, err)
		require.Equal(t, 2, total)
		require.Len(t, arrs, 2)
		for _, a := range arrs {
			require.Contains(t, a.SupplierName, "Sumber")
		}
	})

	t.Run("filters by status", func(t *testing.T) {
		arrs, total, err := svc.ListArrangements(ctx, &claims, 0, 0, "", StatusActive)
		require.NoError(t, err)
		require.Equal(t, 3, total)
		require.Len(t, arrs, 3)
		for _, a := range arrs {
			require.Equal(t, StatusActive, a.Status)
		}
	})

	t.Run("searches by numeric arrangement ID", func(t *testing.T) {
		arrs, _, err := svc.ListArrangements(ctx, &claims, 0, 0, "", "")
		require.NoError(t, err)
		require.NotEmpty(t, arrs)
		target := arrs[0]

		arrs, total, err := svc.ListArrangements(ctx, &claims, 0, 0, fmt.Sprintf("%d", target.ID), "")
		require.NoError(t, err)
		require.Equal(t, 1, total)
		require.Len(t, arrs, 1)
		require.Equal(t, target.ID, arrs[0].ID)
	})

	t.Run("search by non-existent ID returns empty", func(t *testing.T) {
		arrs, total, err := svc.ListArrangements(ctx, &claims, 0, 0, "999999", "")
		require.NoError(t, err)
		require.Equal(t, 0, total)
		require.Empty(t, arrs)
	})

	t.Run("supplier name and store name are hydrated", func(t *testing.T) {
		arrs, _, err := svc.ListArrangements(ctx, &claims, 0, 0, "", "")
		require.NoError(t, err)
		for _, a := range arrs {
			require.NotEmpty(t, a.SupplierName, "SupplierName should be hydrated")
			require.NotEmpty(t, a.StoreName, "StoreName should be hydrated")
		}
	})
}

func TestService_EndArrangement(t *testing.T) {
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	t.Run("ends active arrangement with zero stock", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "END-ZERO-STOCK")
		svc, _, store := setupArrangement(t, product)

		arrs, _, err := svc.ListArrangements(ctx, &store, 0, 0, "", "")
		require.NoError(t, err)
		arr := arrs[0]
		require.Equal(t, StatusActive, arr.Status)

		got, err := svc.EndArrangement(ctx, arr.ID, &store)
		require.NoError(t, err)
		require.Equal(t, StatusEnded, got.Status)
		require.NotNil(t, got.EndedAt)
	})

	t.Run("rejects ending when stock remains", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "END-HAS-STOCK")
		svc, _, store := setupArrangement(t, product)

		arrs, _, err := svc.ListArrangements(ctx, &store, 0, 0, "", "")
		require.NoError(t, err)
		arr := arrs[0]

		// Simulate received goods (stock > 0).
		userID := insertTestUser(ctx, t)
		_, err = svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arr.ID,
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 5}},
		}, userID, &store)
		require.NoError(t, err)

		_, err = svc.EndArrangement(ctx, arr.ID, &store)
		require.ErrorIs(t, err, ErrStockMustBeReturned)
	})

	t.Run("rejects ending already ended arrangement", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "END-ALREADY-ENDED")
		svc, _, store := setupArrangement(t, product)

		arrs, _, err := svc.ListArrangements(ctx, &store, 0, 0, "", "")
		require.NoError(t, err)
		arr := arrs[0]

		_, err = svc.EndArrangement(ctx, arr.ID, &store)
		require.NoError(t, err)

		_, err = svc.EndArrangement(ctx, arr.ID, &store)
		require.ErrorIs(t, err, ErrArrangementEnded)
	})

	t.Run("store scope enforced", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "END-STORE-SCOPE")
		svc, _, store := setupArrangement(t, product)

		arrs, _, err := svc.ListArrangements(ctx, &store, 0, 0, "", "")
		require.NoError(t, err)
		arr := arrs[0]

		otherStore := insertTestStore(ctx, t)
		_, err = svc.EndArrangement(ctx, arr.ID, &otherStore)
		require.ErrorIs(t, err, ErrStoreForbidden)
	})
}

func TestService_AddTerm(t *testing.T) {
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	t.Run("happy path adds single term", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "ADD-TERM-PROD")
		svc, _, store, _ := setupArrangementNoTerms(t)
		userID := insertTestUser(ctx, t)
		arrs, _, _ := svc.ListArrangements(ctx, &store, 0, 0, "", "")

		term, err := svc.AddTerm(ctx, arrs[0].ID, SetTermsRequest{
			ProductID: product, Price: 15000, StoreShareType: ShareTypePercentage, StoreShareValue: 25,
		}, userID, &store)
		require.NoError(t, err)
		require.Equal(t, product, term.ProductID)
		require.Equal(t, 15000, term.Price)
		require.Equal(t, ShareTypePercentage, term.StoreShareType)
		require.Equal(t, 25.0, term.StoreShareValue)
		require.Equal(t, "Test Product ADD-TERM-PROD", term.ProductName)

		got, err := svc.GetArrangement(ctx, arrs[0].ID, &store)
		require.NoError(t, err)
		require.Len(t, got.Terms, 1)
	})

	t.Run("duplicate product rejected", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "ADD-DUP-PROD")
		svc, _, store, _ := setupArrangementNoTerms(t)
		userID := insertTestUser(ctx, t)
		arrs, _, _ := svc.ListArrangements(ctx, &store, 0, 0, "", "")

		_, err := svc.AddTerm(ctx, arrs[0].ID, SetTermsRequest{
			ProductID: product, Price: 10000, StoreShareType: ShareTypePercentage, StoreShareValue: 20,
		}, userID, &store)
		require.NoError(t, err)

		_, err = svc.AddTerm(ctx, arrs[0].ID, SetTermsRequest{
			ProductID: product, Price: 12000, StoreShareType: ShareTypeFixedAmount, StoreShareValue: 3000,
		}, userID, &store)
		require.ErrorIs(t, err, ErrDuplicateProduct)
	})

	t.Run("store-owned stock rejected", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "ADD-STORE-OWNED")
		svc, _, store, _ := setupArrangementNoTerms(t)
		userID := insertTestUser(ctx, t)
		arrs, _, _ := svc.ListArrangements(ctx, &store, 0, 0, "", "")

		seedStoreOwnedStock(ctx, t, product, 10)

		_, err := svc.AddTerm(ctx, arrs[0].ID, SetTermsRequest{
			ProductID: product, Price: 10000, StoreShareType: ShareTypePercentage, StoreShareValue: 20,
		}, userID, &store)
		require.ErrorIs(t, err, ErrConflictStoreStock)
	})

	t.Run("invalid share type rejected", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "ADD-INVALID-SHARE")
		svc, _, store, _ := setupArrangementNoTerms(t)
		userID := insertTestUser(ctx, t)
		arrs, _, _ := svc.ListArrangements(ctx, &store, 0, 0, "", "")

		_, err := svc.AddTerm(ctx, arrs[0].ID, SetTermsRequest{
			ProductID: product, Price: 10000, StoreShareType: "flat", StoreShareValue: 20,
		}, userID, &store)
		require.ErrorIs(t, err, ErrInvalidShareType)
	})

	t.Run("percentage >= 100 rejected", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "ADD-PCT-OVER")
		svc, _, store, _ := setupArrangementNoTerms(t)
		userID := insertTestUser(ctx, t)
		arrs, _, _ := svc.ListArrangements(ctx, &store, 0, 0, "", "")

		_, err := svc.AddTerm(ctx, arrs[0].ID, SetTermsRequest{
			ProductID: product, Price: 10000, StoreShareType: ShareTypePercentage, StoreShareValue: 100,
		}, userID, &store)
		require.ErrorIs(t, err, ErrInvalidShareValueForType)
	})

	t.Run("ended arrangement rejected", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "ADD-ENDED")
		svc, _, store, _ := setupArrangementNoTerms(t)
		userID := insertTestUser(ctx, t)
		arrs, _, _ := svc.ListArrangements(ctx, &store, 0, 0, "", "")

		_, err := svc.EndArrangement(ctx, arrs[0].ID, &store)
		require.NoError(t, err)

		_, err = svc.AddTerm(ctx, arrs[0].ID, SetTermsRequest{
			ProductID: product, Price: 10000, StoreShareType: ShareTypePercentage, StoreShareValue: 20,
		}, userID, &store)
		require.ErrorIs(t, err, ErrArrangementEnded)
	})

	t.Run("zero price rejected", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "ADD-ZERO-PRICE")
		svc, _, store, _ := setupArrangementNoTerms(t)
		userID := insertTestUser(ctx, t)
		arrs, _, _ := svc.ListArrangements(ctx, &store, 0, 0, "", "")

		_, err := svc.AddTerm(ctx, arrs[0].ID, SetTermsRequest{
			ProductID: product, Price: 0, StoreShareType: ShareTypePercentage, StoreShareValue: 20,
		}, userID, &store)
		require.ErrorIs(t, err, ErrInvalidPrice)
	})

	t.Run("negative price rejected", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "ADD-NEG-PRICE")
		svc, _, store, _ := setupArrangementNoTerms(t)
		userID := insertTestUser(ctx, t)
		arrs, _, _ := svc.ListArrangements(ctx, &store, 0, 0, "", "")

		_, err := svc.AddTerm(ctx, arrs[0].ID, SetTermsRequest{
			ProductID: product, Price: -1000, StoreShareType: ShareTypePercentage, StoreShareValue: 20,
		}, userID, &store)
		require.ErrorIs(t, err, ErrInvalidPrice)
	})

	t.Run("other supplier conflict rejected", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "ADD-OTHER-SUP")
		svcA, _, storeA, userID := setupArrangementNoTerms(t)

		// Add a term for supplier A before creating a receipt.
		_, err := svcA.SetTerms(ctx, arrID(t, svcA, storeA), []SetTermsRequest{
			{ProductID: product, Price: 10000, StoreShareType: ShareTypePercentage, StoreShareValue: 20},
		}, userID, &storeA)
		require.NoError(t, err)

		// Supplier A creates an arrangement and receives stock.
		_, err = svcA.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svcA, storeA),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 5}},
		}, userID, &storeA)
		require.NoError(t, err)

		// Supplier B creates a separate arrangement.
		supplierB := insertTestSupplier(ctx, t, "AddTerm Supplier B", true)
		storeB := insertTestStore(ctx, t)
		svcB := newTestService(t)
		arrB, err := svcB.CreateArrangement(ctx, &CreateArrangementRequest{SupplierID: supplierB, StoreID: storeB}, userID, nil)
		require.NoError(t, err)

		// Supplier B tries to add a term on the same product that has stock under supplier A.
		_, err = svcB.AddTerm(ctx, arrB.ID, SetTermsRequest{
			ProductID: product, Price: 12000, StoreShareType: ShareTypePercentage, StoreShareValue: 30,
		}, userID, &storeB)
		require.ErrorIs(t, err, ErrConflictOtherSupplier)
	})

	t.Run("store scope enforced", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "ADD-STORE-SCOPE")
		svc, _, store, _ := setupArrangementNoTerms(t)
		userID := insertTestUser(ctx, t)
		arrs, _, _ := svc.ListArrangements(ctx, &store, 0, 0, "", "")

		otherStore := insertTestStore(ctx, t)
		_, err := svc.AddTerm(ctx, arrs[0].ID, SetTermsRequest{
			ProductID: product, Price: 10000, StoreShareType: ShareTypePercentage, StoreShareValue: 20,
		}, userID, &otherStore)
		require.ErrorIs(t, err, ErrStoreForbidden)
	})

	t.Run("fixed_amount share >= price rejected", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "ADD-FIXED-OVER")
		svc, _, store, _ := setupArrangementNoTerms(t)
		userID := insertTestUser(ctx, t)
		arrs, _, _ := svc.ListArrangements(ctx, &store, 0, 0, "", "")

		_, err := svc.AddTerm(ctx, arrs[0].ID, SetTermsRequest{
			ProductID: product, Price: 10000, StoreShareType: ShareTypeFixedAmount, StoreShareValue: 10000,
		}, userID, &store)
		require.ErrorIs(t, err, ErrFixedShareExceedsPrice)
	})
}

func TestService_RemoveTerm(t *testing.T) {
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	t.Run("happy path removes term", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "RM-TERM-PROD")
		svc, _, store := setupArrangement(t, product)
		arrs, _, _ := svc.ListArrangements(ctx, &store, 0, 0, "", "")

		err := svc.RemoveTerm(ctx, arrs[0].ID, product, &store)
		require.NoError(t, err)

		got, err := svc.GetArrangement(ctx, arrs[0].ID, &store)
		require.NoError(t, err)
		require.Len(t, got.Terms, 0)
	})

	t.Run("non-existent term returns error", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "RM-NONEXIST")
		svc, _, store, _ := setupArrangementNoTerms(t)
		arrs, _, _ := svc.ListArrangements(ctx, &store, 0, 0, "", "")

		err := svc.RemoveTerm(ctx, arrs[0].ID, product, &store)
		require.Error(t, err)
	})

	t.Run("ended arrangement rejected", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "RM-ENDED")
		svc, _, store := setupArrangement(t, product)
		arrs, _, _ := svc.ListArrangements(ctx, &store, 0, 0, "", "")

		_, err := svc.EndArrangement(ctx, arrs[0].ID, &store)
		require.NoError(t, err)

		err = svc.RemoveTerm(ctx, arrs[0].ID, product, &store)
		require.ErrorIs(t, err, ErrArrangementEnded)
	})

	t.Run("store scope enforced", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "RM-STORE-SCOPE")
		svc, _, store := setupArrangement(t, product)
		arrs, _, _ := svc.ListArrangements(ctx, &store, 0, 0, "", "")

		otherStore := insertTestStore(ctx, t)
		err := svc.RemoveTerm(ctx, arrs[0].ID, product, &otherStore)
		require.ErrorIs(t, err, ErrStoreForbidden)
	})

	t.Run("removed product can be re-added", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "RM-READD")
		svc, _, store, _ := setupArrangementNoTerms(t)
		userID := insertTestUser(ctx, t)
		arrs, _, _ := svc.ListArrangements(ctx, &store, 0, 0, "", "")

		_, err := svc.AddTerm(ctx, arrs[0].ID, SetTermsRequest{
			ProductID: product, Price: 10000, StoreShareType: ShareTypePercentage, StoreShareValue: 20,
		}, userID, &store)
		require.NoError(t, err)

		err = svc.RemoveTerm(ctx, arrs[0].ID, product, &store)
		require.NoError(t, err)

		_, err = svc.AddTerm(ctx, arrs[0].ID, SetTermsRequest{
			ProductID: product, Price: 12000, StoreShareType: ShareTypeFixedAmount, StoreShareValue: 3000,
		}, userID, &store)
		require.NoError(t, err)

		got, err := svc.GetArrangement(ctx, arrs[0].ID, &store)
		require.NoError(t, err)
		require.Len(t, got.Terms, 1)
		require.Equal(t, 12000, got.Terms[0].Price)
	})
}
