package consignment

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

func intPtr(v int) *int { return &v }

func TestReconciliationReasonValidation(t *testing.T) {
	t.Parallel()
	valid := []string{ReasonDamaged, ReasonExpired, ReasonMiscount, ReasonLost, ReasonStolen, ReasonUnrecordedReceipt, ReasonOther}
	for _, r := range valid {
		require.Truef(t, validReconciliationReason(r), "%q should be valid", r)
	}
	for _, r := range []string{"", "customer_return", "termination", "bogus"} {
		require.Falsef(t, validReconciliationReason(r), "%q should be invalid", r)
	}
	require.True(t, reasonCreatesLiability(ReasonLost))
	require.True(t, reasonCreatesLiability(ReasonStolen))
	require.False(t, reasonCreatesLiability(ReasonDamaged))
	require.False(t, reasonCreatesLiability(ReasonMiscount))
	require.False(t, reasonCreatesLiability(ReasonUnrecordedReceipt))
}

// receiveStock receives qty units of a product onto an arrangement so the count
// sheet has a system quantity to compare against.
func receiveStock(ctx context.Context, t *testing.T, svc *Service, product, qty, userID, store int) {
	t.Helper()
	_, err := svc.CreateReceipt(ctx, &ReceiptRequest{
		ArrangementID: arrID(t, svc, store),
		Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: qty}},
	}, userID, &store)
	require.NoError(t, err)
}

func TestService_ReconciliationLifecycle(t *testing.T) {
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	product := insertTestProduct(ctx, t, "RECON-LIFE")
	svc, sup, store := setupArrangement(t, product)
	userID := insertTestUser(ctx, t)
	arrangementID := arrID(t, svc, store)

	receiveStock(ctx, t, svc, product, 10, userID, store)
	require.Equal(t, 10, globalStockQty(ctx, t, product))

	rec, err := svc.CreateReconciliation(ctx, &CreateReconciliationRequest{ArrangementID: arrangementID}, userID, &store)
	require.NoError(t, err)
	require.NotEmpty(t, rec.ReconciliationNumber)
	require.Equal(t, ReconcileStatusDraft, rec.Status)
	require.Equal(t, sup, rec.SupplierID)
	require.NotEmpty(t, rec.SupplierName, "SupplierName should be hydrated")

	sheet, err := svc.GetReconciliationCountSheet(ctx, arrangementID, &store)
	require.NoError(t, err)
	require.Len(t, sheet, 1)
	require.Equal(t, 10, sheet[0].SystemAvailable)
	require.NotEmpty(t, sheet[0].ProductSKU, "count sheet should hydrate the product")

	// Count 7 of 10: a three-unit shortfall attributable to the store.
	counted, err := svc.SaveReconciliationCount(ctx, rec.ID, &SaveReconciliationCountRequest{
		Items: []ReconciliationItemInput{{ProductID: product, PhysicalAvailable: intPtr(7), Reason: ReasonLost}},
	}, userID, &store)
	require.NoError(t, err)
	require.Equal(t, ReconcileStatusCounted, counted.Status)
	require.Len(t, counted.Items, 1)
	require.Equal(t, 10, counted.Items[0].SystemAvailable)
	require.Equal(t, 7, counted.Items[0].PhysicalAvailable)
	require.Equal(t, -3, counted.Items[0].Variance)
	require.Equal(t, 3, counted.TotalShortfall)
	require.Zero(t, counted.Items[0].LiabilityAmount, "liability is priced at posting, not counting")

	// Posting is the only step that touches stock.
	posted, err := svc.PostReconciliation(ctx, rec.ID, userID, &store)
	require.NoError(t, err)
	require.Equal(t, ReconcileStatusPosted, posted.Status)
	require.Equal(t, 7, globalStockQty(ctx, t, product), "global sellable stock follows the variance")

	row, err := svc.repo.GetConsignmentStock(ctx, svc.repo.db, product)
	require.NoError(t, err)
	require.NotNil(t, row)
	require.Equal(t, 7, row.AvailableQty, "ownership ledger follows the variance")

	// Liability = entitlement (3 * 10000) minus the store's 20% share (6000).
	require.Equal(t, 24000, posted.Items[0].LiabilityAmount)
	require.Equal(t, 24000, posted.TotalLiability)

	// The open liability rides on the next settlement even with no sale items.
	preview, err := svc.GetSettlementPreview(ctx, sup, &store)
	require.NoError(t, err)
	require.Equal(t, 0, preview.TotalSaleValue)
	require.Equal(t, 24000, preview.TotalLiability)
	require.Equal(t, 24000, preview.TotalPayable)
	require.Empty(t, preview.Items, "liabilities are not settlement items")

	st, err := svc.CreateSettlement(ctx, &CreateSettlementRequest{SupplierID: sup}, userID, &store)
	require.NoError(t, err)
	require.Equal(t, 24000, st.TotalLiability)
	require.Equal(t, 24000, st.TotalPayable)

	// Settled: nothing left to settle.
	_, err = svc.CreateSettlement(ctx, &CreateSettlementRequest{SupplierID: sup}, userID, &store)
	require.ErrorIs(t, err, ErrEmptySettlement)
}

func TestService_ReconciliationSurplus(t *testing.T) {
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	product := insertTestProduct(ctx, t, "RECON-SURP")
	svc, _, store := setupArrangement(t, product)
	userID := insertTestUser(ctx, t)
	arrangementID := arrID(t, svc, store)

	receiveStock(ctx, t, svc, product, 5, userID, store)

	rec, err := svc.CreateReconciliation(ctx, &CreateReconciliationRequest{ArrangementID: arrangementID}, userID, &store)
	require.NoError(t, err)

	// Found two extra units; a surplus requires a reason but never a liability.
	counted, err := svc.SaveReconciliationCount(ctx, rec.ID, &SaveReconciliationCountRequest{
		Items: []ReconciliationItemInput{{ProductID: product, PhysicalAvailable: intPtr(7), Reason: ReasonUnrecordedReceipt}},
	}, userID, &store)
	require.NoError(t, err)
	require.Equal(t, 2, counted.Items[0].Variance)
	require.Equal(t, 2, counted.TotalSurplus)

	posted, err := svc.PostReconciliation(ctx, rec.ID, userID, &store)
	require.NoError(t, err)
	require.Equal(t, 7, globalStockQty(ctx, t, product))
	require.Zero(t, posted.TotalLiability, "a surplus never creates a liability")
}

func TestService_ReconciliationValidation(t *testing.T) {
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	product := insertTestProduct(ctx, t, "RECON-VALID")
	other := insertTestProduct(ctx, t, "RECON-VALID-OTHER")
	svc, _, store := setupArrangement(t, product)
	userID := insertTestUser(ctx, t)
	arrangementID := arrID(t, svc, store)

	receiveStock(ctx, t, svc, product, 10, userID, store)

	newDraft := func(t *testing.T) *Reconciliation {
		t.Helper()
		rec, err := svc.CreateReconciliation(ctx, &CreateReconciliationRequest{ArrangementID: arrangementID}, userID, &store)
		require.NoError(t, err)
		return rec
	}

	t.Run("missing reason for non-zero variance", func(t *testing.T) {
		rec := newDraft(t)
		_, err := svc.SaveReconciliationCount(ctx, rec.ID, &SaveReconciliationCountRequest{
			Items: []ReconciliationItemInput{{ProductID: product, PhysicalAvailable: intPtr(8)}},
		}, userID, &store)
		require.ErrorIs(t, err, ErrReconciliationReasonRequired)
	})

	t.Run("invalid reason", func(t *testing.T) {
		rec := newDraft(t)
		_, err := svc.SaveReconciliationCount(ctx, rec.ID, &SaveReconciliationCountRequest{
			Items: []ReconciliationItemInput{{ProductID: product, PhysicalAvailable: intPtr(8), Reason: ReasonTermination}},
		}, userID, &store)
		require.ErrorIs(t, err, ErrInvalidReconciliationReason)
	})

	t.Run("product not in arrangement", func(t *testing.T) {
		rec := newDraft(t)
		_, err := svc.SaveReconciliationCount(ctx, rec.ID, &SaveReconciliationCountRequest{
			Items: []ReconciliationItemInput{{ProductID: other, PhysicalAvailable: intPtr(1), Reason: ReasonLost}},
		}, userID, &store)
		require.ErrorIs(t, err, ErrReconciliationProductMismatch)
	})

	t.Run("duplicate product", func(t *testing.T) {
		rec := newDraft(t)
		_, err := svc.SaveReconciliationCount(ctx, rec.ID, &SaveReconciliationCountRequest{
			Items: []ReconciliationItemInput{
				{ProductID: product, PhysicalAvailable: intPtr(8), Reason: ReasonLost},
				{ProductID: product, PhysicalAvailable: intPtr(9), Reason: ReasonLost},
			},
		}, userID, &store)
		require.ErrorIs(t, err, ErrDuplicateReconciliationProduct)
	})

	t.Run("count on a non-draft document", func(t *testing.T) {
		rec := newDraft(t)
		_, err := svc.SaveReconciliationCount(ctx, rec.ID, &SaveReconciliationCountRequest{
			Items: []ReconciliationItemInput{{ProductID: product, PhysicalAvailable: intPtr(10)}},
		}, userID, &store)
		require.NoError(t, err)
		_, err = svc.SaveReconciliationCount(ctx, rec.ID, &SaveReconciliationCountRequest{
			Items: []ReconciliationItemInput{{ProductID: product, PhysicalAvailable: intPtr(9), Reason: ReasonLost}},
		}, userID, &store)
		require.ErrorIs(t, err, ErrInvalidReconciliationStatus)
	})

	t.Run("post requires a counted document", func(t *testing.T) {
		rec := newDraft(t)
		_, err := svc.PostReconciliation(ctx, rec.ID, userID, &store)
		require.ErrorIs(t, err, ErrInvalidReconciliationStatus)
	})

	t.Run("store scope is enforced", func(t *testing.T) {
		rec := newDraft(t)
		otherStore := store + 999999
		_, err := svc.GetReconciliation(ctx, rec.ID, &otherStore)
		require.ErrorIs(t, err, ErrStoreForbidden)
	})
}

// TestService_ReconciliationStaleOwnership covers the counted -> posted window:
// ownership can move between counting and posting, and applying the stale
// variance must not drag the ledger back to this arrangement (BR-03).
func TestService_ReconciliationStaleOwnership(t *testing.T) {
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	product := insertTestProduct(ctx, t, "RECON-STALE")
	svc, _, store := setupArrangement(t, product)
	userID := insertTestUser(ctx, t)
	arrangementID := arrID(t, svc, store)

	receiveStock(ctx, t, svc, product, 10, userID, store)

	// Count a three-unit shortfall attributable to the store.
	rec, err := svc.CreateReconciliation(ctx, &CreateReconciliationRequest{ArrangementID: arrangementID}, userID, &store)
	require.NoError(t, err)
	_, err = svc.SaveReconciliationCount(ctx, rec.ID, &SaveReconciliationCountRequest{
		Items: []ReconciliationItemInput{{ProductID: product, PhysicalAvailable: intPtr(7), Reason: ReasonLost}},
	}, userID, &store)
	require.NoError(t, err)

	// A different supplier takes over the SKU between count and post.
	otherSupplier := insertTestSupplier(ctx, t, "Konsinyasi Other Supplier", true)
	_, err = dbPool.Exec(ctx, `UPDATE consignment_stock SET supplier_id = $1 WHERE product_id = $2`, otherSupplier, product)
	require.NoError(t, err)

	_, err = svc.PostReconciliation(ctx, rec.ID, userID, &store)
	require.ErrorIs(t, err, ErrReconciliationStale)

	// The ledger still belongs to the new supplier; the stale post changed nothing.
	row, err := svc.repo.GetConsignmentStock(ctx, svc.repo.db, product)
	require.NoError(t, err)
	require.NotNil(t, row)
	require.Equal(t, otherSupplier, row.SupplierID)
	require.Equal(t, 10, row.AvailableQty)

	// A shortfall on a fully-released SKU (no ledger row) is stale too.
	rec2, err := svc.CreateReconciliation(ctx, &CreateReconciliationRequest{ArrangementID: arrangementID}, userID, &store)
	require.NoError(t, err)
	_, err = svc.SaveReconciliationCount(ctx, rec2.ID, &SaveReconciliationCountRequest{
		Items: []ReconciliationItemInput{{ProductID: product, PhysicalAvailable: intPtr(7), Reason: ReasonLost}},
	}, userID, &store)
	require.NoError(t, err)
	_, err = dbPool.Exec(ctx, `DELETE FROM consignment_stock WHERE product_id = $1`, product)
	require.NoError(t, err)

	_, err = svc.PostReconciliation(ctx, rec2.ID, userID, &store)
	require.ErrorIs(t, err, ErrReconciliationStale)
}
