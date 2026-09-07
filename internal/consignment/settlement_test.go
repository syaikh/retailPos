package consignment

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

// insertTestSale creates a minimal sales row and returns its id.
func insertTestSale(ctx context.Context, t *testing.T, storeID int, invoice string) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(ctx, `
		INSERT INTO sales (invoice_number, cashier_id, store_id, customer_id, subtotal, total_amount, payment_method)
		VALUES ($1, $2, $3, $4, 0, 0, 'cash')
		RETURNING id
	`, invoice, insertTestUser(ctx, t), storeID, insertTestCustomer(ctx, t)).Scan(&id)
	require.NoError(t, err)
	return id
}

// insertTestCustomer creates a walk-in customer (sales.customer_id NOT NULL).
func insertTestCustomer(ctx context.Context, t *testing.T) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(ctx, `
		INSERT INTO customers (name, phone, email) VALUES ($1, $2, $3)
		ON CONFLICT (phone) DO UPDATE SET name = EXCLUDED.name
		RETURNING id
	`, "Walk-in", "consignment-walkin", "consignment-walkin@test.com").Scan(&id)
	require.NoError(t, err)
	return id
}

// insertConsignmentSaleItem persists a consignment sale line directly, as the
// checkout provider would, so settlement math can be tested in isolation.
func insertConsignmentSaleItem(ctx context.Context, t *testing.T, svc *Service, rec shared.ConsignmentSaleRecord) {
	t.Helper()
	tx, err := svc.repo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()
	require.NoError(t, svc.repo.InsertConsignmentSaleItem(ctx, tx, rec))
	require.NoError(t, tx.Commit(ctx))
}

func TestService_SettlementMath(t *testing.T) {
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	t.Run("percentage share preview", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "SET-PCT")
		svc, sup, store := setupArrangement(t, product)

		// A sale of 5 units at the actual sale price 10,000, 20% store share.
		saleID := insertTestSale(ctx, t, store, "SET-PCT-1")
		insertConsignmentSaleItem(ctx, t, svc, shared.ConsignmentSaleRecord{
			SaleID: saleID, InvoiceNumber: "SET-PCT-1", ProductID: product,
			SupplierID: sup, ArrangementID: arrID(t, svc, store), StoreID: store,
			Quantity: 5, UnitPrice: 10000, Subtotal: 50000,
			StoreShareType: ShareTypePercentage, StoreShareValue: 20,
		})

		preview, err := svc.GetSettlementPreview(ctx, sup, &store)
		require.NoError(t, err)
		require.Equal(t, 50000, preview.TotalSaleValue)
		require.Equal(t, 10000, preview.TotalStoreShare) // 20% of 50,000
		require.Equal(t, 40000, preview.TotalPayable)
		require.Len(t, preview.Items, 1)
	})

	t.Run("fixed amount share preview", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "SET-FIXED")
		svc, sup, store := setupArrangement(t, product)

		saleID := insertTestSale(ctx, t, store, "SET-FIXED-1")
		insertConsignmentSaleItem(ctx, t, svc, shared.ConsignmentSaleRecord{
			SaleID: saleID, InvoiceNumber: "SET-FIXED-1", ProductID: product,
			SupplierID: sup, ArrangementID: arrID(t, svc, store), StoreID: store,
			Quantity: 4, UnitPrice: 8000, Subtotal: 32000,
			StoreShareType: ShareTypeFixedAmount, StoreShareValue: 1500,
		})

		preview, err := svc.GetSettlementPreview(ctx, sup, &store)
		require.NoError(t, err)
		require.Equal(t, 32000, preview.TotalSaleValue)
		require.Equal(t, 6000, preview.TotalStoreShare) // 1500 × 4
		require.Equal(t, 26000, preview.TotalPayable)
	})

	t.Run("qty greater than one percentage is quantity weighted", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "SET-QTY-PCT")
		svc, sup, store := setupArrangement(t, product)

		// 3 units at 10,000 with a 20% share → store share = 6000.
		saleID := insertTestSale(ctx, t, store, "SET-QTY-PCT-1")
		insertConsignmentSaleItem(ctx, t, svc, shared.ConsignmentSaleRecord{
			SaleID: saleID, InvoiceNumber: "SET-QTY-PCT-1", ProductID: product,
			SupplierID: sup, ArrangementID: arrID(t, svc, store), StoreID: store,
			Quantity: 3, UnitPrice: 10000, Subtotal: 30000,
			StoreShareType: ShareTypePercentage, StoreShareValue: 20,
		})

		preview, err := svc.GetSettlementPreview(ctx, sup, &store)
		require.NoError(t, err)
		require.Equal(t, 30000, preview.TotalSaleValue)
		require.Equal(t, 6000, preview.TotalStoreShare)
		require.Equal(t, 24000, preview.TotalPayable)
	})

	t.Run("qty greater than one fixed amount is quantity weighted", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "SET-QTY-FIXED")
		svc, sup, store := setupArrangement(t, product)

		saleID := insertTestSale(ctx, t, store, "SET-QTY-FIXED-1")
		insertConsignmentSaleItem(ctx, t, svc, shared.ConsignmentSaleRecord{
			SaleID: saleID, InvoiceNumber: "SET-QTY-FIXED-1", ProductID: product,
			SupplierID: sup, ArrangementID: arrID(t, svc, store), StoreID: store,
			Quantity: 3, UnitPrice: 10000, Subtotal: 30000,
			StoreShareType: ShareTypeFixedAmount, StoreShareValue: 2500,
		})

		preview, err := svc.GetSettlementPreview(ctx, sup, &store)
		require.NoError(t, err)
		require.Equal(t, 7500, preview.TotalStoreShare) // 2500 × 3
		require.Equal(t, 22500, preview.TotalPayable)
	})

	t.Run("EC-03 EC-04 terms change after sale uses snapshotted share", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "SET-SNAP")
		svc, sup, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		saleID := insertTestSale(ctx, t, store, "SET-SNAP-1")
		insertConsignmentSaleItem(ctx, t, svc, shared.ConsignmentSaleRecord{
			SaleID: saleID, InvoiceNumber: "SET-SNAP-1", ProductID: product,
			SupplierID: sup, ArrangementID: arrID(t, svc, store), StoreID: store,
			Quantity: 2, UnitPrice: 10000, Subtotal: 20000,
			StoreShareType: ShareTypePercentage, StoreShareValue: 20,
		})

		// Terms change before settlement: new 30% share. The old sale must
		// still settle with its snapshotted 20%.
		_, err := svc.SetTerms(ctx, arrID(t, svc, store), []SetTermsRequest{
			{ProductID: product, Price: 12000, StoreShareType: ShareTypePercentage, StoreShareValue: 30},
		}, userID, &store)
		require.NoError(t, err)

		preview, err := svc.GetSettlementPreview(ctx, sup, &store)
		require.NoError(t, err)
		require.Equal(t, 20000, preview.TotalSaleValue)
		require.Equal(t, 4000, preview.TotalStoreShare) // still 20%
		require.Equal(t, 16000, preview.TotalPayable)
	})

	t.Run("empty preview rejected", func(t *testing.T) {
		svc, sup, store := setupArrangement(t)
		_, err := svc.GetSettlementPreview(ctx, sup, &store)
		require.ErrorIs(t, err, ErrEmptySettlement)
	})

	t.Run("EC-03 price changes don't affect unsettled subtotal", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "SET-PRICE")
		svc, sup, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		saleID := insertTestSale(ctx, t, store, "SET-PRICE-1")
		insertConsignmentSaleItem(ctx, t, svc, shared.ConsignmentSaleRecord{
			SaleID: saleID, InvoiceNumber: "SET-PRICE-1", ProductID: product,
			SupplierID: sup, ArrangementID: arrID(t, svc, store), StoreID: store,
			Quantity: 1, UnitPrice: 10000, Subtotal: 10000,
			StoreShareType: ShareTypePercentage, StoreShareValue: 20,
		})

		// Term price was 10000 at sale; a later receipt item uses term.Price,
		// but a NEW sale at the new term price is snapshotted independently.
		_, err := svc.SetTerms(ctx, arrID(t, svc, store), []SetTermsRequest{
			{ProductID: product, Price: 15000, StoreShareType: ShareTypePercentage, StoreShareValue: 20},
		}, userID, &store)
		require.NoError(t, err)

		preview, err := svc.GetSettlementPreview(ctx, sup, &store)
		require.NoError(t, err)
		require.Equal(t, 10000, preview.TotalSaleValue)
		require.Equal(t, 2000, preview.TotalStoreShare)
	})
}

func TestService_SettlementLifecycle(t *testing.T) {
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	t.Run("create settlement links sale items", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "SET-LIFE")
		svc, sup, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		saleID := insertTestSale(ctx, t, store, "SET-LIFE-1")
		insertConsignmentSaleItem(ctx, t, svc, shared.ConsignmentSaleRecord{
			SaleID: saleID, InvoiceNumber: "SET-LIFE-1", ProductID: product,
			SupplierID: sup, ArrangementID: arrID(t, svc, store), StoreID: store,
			Quantity: 5, UnitPrice: 10000, Subtotal: 50000,
			StoreShareType: ShareTypePercentage, StoreShareValue: 20,
		})

		st, err := svc.CreateSettlement(ctx, &CreateSettlementRequest{SupplierID: sup}, userID, &store)
		require.NoError(t, err)
		require.NotEmpty(t, st.SettlementNumber)
		require.Equal(t, 50000, st.TotalSaleValue)
		require.Equal(t, 10000, st.TotalStoreShare)
		require.Equal(t, 40000, st.TotalPayable)
		require.Equal(t, SettlementPendingPayment, st.Status)
		require.Len(t, st.Items, 1)
		require.NotEmpty(t, st.SupplierName, "SupplierName should be hydrated")
		require.NotEmpty(t, st.Items[0].ProductName, "ProductName should be hydrated in settlement items")

		// Items are settled now.
		_, err = svc.GetSettlementPreview(ctx, sup, &store)
		require.ErrorIs(t, err, ErrEmptySettlement)
	})

	t.Run("EC-03 EC-04 settled items not re-settled", func(t *testing.T) {
		product := insertTestProduct(ctx, t, "SET-ONCE")
		svc, sup, store := setupArrangement(t, product)
		userID := insertTestUser(ctx, t)

		saleID := insertTestSale(ctx, t, store, "SET-ONCE-1")
		insertConsignmentSaleItem(ctx, t, svc, shared.ConsignmentSaleRecord{
			SaleID: saleID, InvoiceNumber: "SET-ONCE-1", ProductID: product,
			SupplierID: sup, ArrangementID: arrID(t, svc, store), StoreID: store,
			Quantity: 3, UnitPrice: 10000, Subtotal: 30000,
			StoreShareType: ShareTypePercentage, StoreShareValue: 20,
		})

		_, err := svc.CreateSettlement(ctx, &CreateSettlementRequest{SupplierID: sup}, userID, &store)
		require.NoError(t, err)

		// A second settlement attempt finds nothing unsettled.
		_, err = svc.CreateSettlement(ctx, &CreateSettlementRequest{SupplierID: sup}, userID, &store)
		require.ErrorIs(t, err, ErrEmptySettlement)
	})

	t.Run("EC-04 store share snapshot per item", func(t *testing.T) {
		skuA := insertTestProduct(ctx, t, "SET-SNAP-A")
		skuB := insertTestProduct(ctx, t, "SET-SNAP-B")
		svc, sup, store := setupArrangement(t, skuA, skuB)

		saleA := insertTestSale(ctx, t, store, "SET-SNAP-A-1")
		insertConsignmentSaleItem(ctx, t, svc, shared.ConsignmentSaleRecord{
			SaleID: saleA, InvoiceNumber: "SET-SNAP-A-1", ProductID: skuA,
			SupplierID: sup, ArrangementID: arrID(t, svc, store), StoreID: store,
			Quantity: 2, UnitPrice: 10000, Subtotal: 20000,
			StoreShareType: ShareTypePercentage, StoreShareValue: 20,
		})
		saleB := insertTestSale(ctx, t, store, "SET-SNAP-B-1")
		insertConsignmentSaleItem(ctx, t, svc, shared.ConsignmentSaleRecord{
			SaleID: saleB, InvoiceNumber: "SET-SNAP-B-1", ProductID: skuB,
			SupplierID: sup, ArrangementID: arrID(t, svc, store), StoreID: store,
			Quantity: 1, UnitPrice: 10000, Subtotal: 10000,
			StoreShareType: ShareTypePercentage, StoreShareValue: 30,
		})

		preview, err := svc.GetSettlementPreview(ctx, sup, &store)
		require.NoError(t, err)
		require.Equal(t, 30000, preview.TotalSaleValue)
		require.Equal(t, 7000, preview.TotalStoreShare) // 4000 + 3000
		require.Equal(t, 23000, preview.TotalPayable)
	})
}

func TestService_ListSettlementsWithNilStore(t *testing.T) {
	ctx := context.Background()
	_ = shared.TruncateTestData(dbPool)

	product := insertTestProduct(ctx, t, "LIST-SETTLE")
	svc, sup, store := setupArrangement(t, product)
	userID := insertTestUser(ctx, t)

	saleID := insertTestSale(ctx, t, store, "LIST-SETTLE-1")
	insertConsignmentSaleItem(ctx, t, svc, shared.ConsignmentSaleRecord{
		SaleID: saleID, InvoiceNumber: "LIST-SETTLE-1", ProductID: product,
		SupplierID: sup, ArrangementID: arrID(t, svc, store), StoreID: store,
		Quantity: 5, UnitPrice: 10000, Subtotal: 50000,
		StoreShareType: ShareTypePercentage, StoreShareValue: 20,
	})

	_, err := svc.CreateSettlement(ctx, &CreateSettlementRequest{SupplierID: sup}, userID, &store)
	require.NoError(t, err)

	scoped, err := svc.ListSettlements(ctx, sup, &store)
	require.NoError(t, err)
	require.Len(t, scoped, 1)
	require.NotEmpty(t, scoped[0].SupplierName, "SupplierName should be hydrated on list")

	admin, err := svc.ListSettlements(ctx, sup, nil)
	require.NoError(t, err)
	require.Len(t, admin, 1)
	require.NotEmpty(t, admin[0].SupplierName, "SupplierName should be hydrated on admin list")
}
