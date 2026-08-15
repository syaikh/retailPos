package consignment

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

// seedSettlement creates a consignment sale + settlement for a supplier and
// returns the service and settlement id.
func seedSettlement(t *testing.T, sku string) (*Service, int, int, int) {
	t.Helper()
	ctx := context.Background()
	product := insertTestProduct(ctx, t, sku)
	svc, sup, store := setupArrangement(t, product)
	userID := insertTestUser(ctx, t)

	saleID := insertTestSale(ctx, t, store, sku+"-1")
	insertConsignmentSaleItem(ctx, t, svc, shared.ConsignmentSaleRecord{
		SaleID: saleID, InvoiceNumber: sku + "-1", ProductID: product,
		SupplierID: sup, ArrangementID: arrID(t, svc, store), StoreID: store,
		Quantity: 5, UnitPrice: 10000, Subtotal: 50000,
		StoreShareType: ShareTypePercentage, StoreShareValue: 20,
	})
	st, err := svc.CreateSettlement(ctx, &CreateSettlementRequest{SupplierID: sup}, userID, &store)
	require.NoError(t, err)
	return svc, st.ID, sup, store
}

func TestService_Payout(t *testing.T) {
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	t.Run("full payout marks settlement paid", func(t *testing.T) {
		svc, stID, _, store := seedSettlement(t, "PAY-FULL")
		userID := insertTestUser(ctx, t)
		pmID := insertTestPaymentMethod(ctx, t, "PAY-FULL-PM")

		st, err := svc.GetSettlement(ctx, stID, &store)
		require.NoError(t, err)
		require.Equal(t, 40000, st.TotalPayable)
		require.Equal(t, SettlementPendingPayment, st.Status)

		payout, err := svc.CreatePayout(ctx, stID, &CreatePayoutRequest{
			PaymentMethodID: pmID,
			Amount:          40000,
			ReferenceNumber: "REF-1",
		}, userID, &store)
		require.NoError(t, err)
		require.NotEmpty(t, payout.PayoutNumber)
		require.Equal(t, 40000, payout.Amount)

		st, err = svc.GetSettlement(ctx, stID, &store)
		require.NoError(t, err)
		require.Equal(t, SettlementPaid, st.Status)
		require.NotNil(t, st.PaidAt)
		require.Len(t, st.Payouts, 1)
	})

	t.Run("partial payouts until fully paid", func(t *testing.T) {
		svc, stID, _, store := seedSettlement(t, "PAY-PART")
		userID := insertTestUser(ctx, t)
		pmID := insertTestPaymentMethod(ctx, t, "PAY-PART-PM")

		_, err := svc.CreatePayout(ctx, stID, &CreatePayoutRequest{PaymentMethodID: pmID, Amount: 15000}, userID, &store)
		require.NoError(t, err)

		st, err := svc.GetSettlement(ctx, stID, &store)
		require.NoError(t, err)
		require.Equal(t, SettlementPendingPayment, st.Status) // not yet fully paid

		_, err = svc.CreatePayout(ctx, stID, &CreatePayoutRequest{PaymentMethodID: pmID, Amount: 25000}, userID, &store)
		require.NoError(t, err)

		st, err = svc.GetSettlement(ctx, stID, &store)
		require.NoError(t, err)
		require.Equal(t, SettlementPaid, st.Status)
		require.Len(t, st.Payouts, 2)
	})

	t.Run("overpayment rejected", func(t *testing.T) {
		svc, stID, _, store := seedSettlement(t, "PAY-OVER")
		userID := insertTestUser(ctx, t)
		pmID := insertTestPaymentMethod(ctx, t, "PAY-OVER-PM")

		_, err := svc.CreatePayout(ctx, stID, &CreatePayoutRequest{PaymentMethodID: pmID, Amount: 50001}, userID, &store)
		require.ErrorIs(t, err, ErrInvalidPayoutAmount)
	})

	t.Run("payout on paid settlement rejected", func(t *testing.T) {
		svc, stID, _, store := seedSettlement(t, "PAY-AGAIN")
		userID := insertTestUser(ctx, t)
		pmID := insertTestPaymentMethod(ctx, t, "PAY-AGAIN-PM")

		_, err := svc.CreatePayout(ctx, stID, &CreatePayoutRequest{PaymentMethodID: pmID, Amount: 40000}, userID, &store)
		require.NoError(t, err)

		_, err = svc.CreatePayout(ctx, stID, &CreatePayoutRequest{PaymentMethodID: pmID, Amount: 1}, userID, &store)
		require.ErrorIs(t, err, ErrSettlementAlreadyPaid)
	})

	t.Run("unknown payment method rejected", func(t *testing.T) {
		svc, stID, _, store := seedSettlement(t, "PAY-PM")
		userID := insertTestUser(ctx, t)

		_, err := svc.CreatePayout(ctx, stID, &CreatePayoutRequest{PaymentMethodID: 999999, Amount: 10000}, userID, &store)
		require.ErrorIs(t, err, ErrPaymentMethodNotFound)
	})

	t.Run("store scope enforced", func(t *testing.T) {
		svc, stID, _, _ := seedSettlement(t, "PAY-SCOPE")
		userID := insertTestUser(ctx, t)
		pmID := insertTestPaymentMethod(ctx, t, "PAY-SCOPE-PM")
		otherStore := insertTestStore(ctx, t)

		_, err := svc.CreatePayout(ctx, stID, &CreatePayoutRequest{PaymentMethodID: pmID, Amount: 10000}, userID, &otherStore)
		require.ErrorIs(t, err, ErrStoreForbidden)
	})
}

func TestService_CheckoutInsufficientStock(t *testing.T) {
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	// Checkout deducts from the ledger; selling more than available must fail.
	t.Run("sale above available aborts", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "CHK-OUT")
		svc, _, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		_, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 5}},
		}, userID, &store)
		require.NoError(t, err)

		provider := NewCheckoutProvider(svc.repo)
		tx, err := svc.repo.BeginTx(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		_, err = provider.ResolveAndDeductConsignment(ctx, tx, []shared.ConsignmentCheckoutItem{
			{ProductID: product, Quantity: 6, UnitPrice: 10000},
		})
		require.ErrorIs(t, err, ErrInsufficientConsignmentStock)
	})

	t.Run("consignment lines deducted and recorded", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "CHK-DEDUCT")
		svc, sup, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		_, err := svc.CreateReceipt(ctx, &ReceiptRequest{
			ArrangementID: arrID(t, svc, store),
			Items:         []ReceiptItemRequest{{ProductID: product, AcceptedQty: 10}},
		}, userID, &store)
		require.NoError(t, err)

		provider := NewCheckoutProvider(svc.repo)
		tx, err := svc.repo.BeginTx(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		records, err := provider.ResolveAndDeductConsignment(ctx, tx, []shared.ConsignmentCheckoutItem{
			{ProductID: product, Quantity: 4, UnitPrice: 12000},
		})
		require.NoError(t, err)
		require.Len(t, records, 1)
		require.Equal(t, sup, records[0].SupplierID)
		require.Equal(t, 4, records[0].Quantity)
		require.Equal(t, 12000, records[0].UnitPrice)
		require.Equal(t, 48000, records[0].Subtotal)

		// The actual sale price drives the store share snapshot.
		require.Equal(t, ShareTypePercentage, records[0].StoreShareType)
		require.Equal(t, 20.0, records[0].StoreShareValue)
	})

	t.Run("store-owned lines skipped", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "CHK-STORE")
		svc, _, _ := setupArrangement(t, product)

		provider := NewCheckoutProvider(svc.repo)
		tx, err := svc.repo.BeginTx(ctx)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback(ctx) }()

		// No consignment_stock row: treated as store-owned, skipped.
		records, err := provider.ResolveAndDeductConsignment(ctx, tx, []shared.ConsignmentCheckoutItem{
			{ProductID: product, Quantity: 2, UnitPrice: 9000},
		})
		require.NoError(t, err)
		require.Empty(t, records)
	})
}