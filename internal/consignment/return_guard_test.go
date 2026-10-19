package consignment

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

func TestSaleConsignmentGuard_ConsignmentProductIDs(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	guard := SaleConsignmentGuard{}

	t.Run("no consignment lines returns empty", func(t *testing.T) {
		svc := newTestService(t)
		storeID := insertTestStore(ctx, t)
		saleID := insertTestSale(ctx, t, storeID, "CG-EMPTY-1")
		ids, err := guard.ConsignmentProductIDs(ctx, txForTest(ctx, svc, t), saleID)
		require.NoError(t, err)
		// ids is built via append, so no rows means nil; the contract is
		// "empty means the sale has none", which Len 0 expresses.
		require.Len(t, ids, 0)
	})

	t.Run("unknown sale returns empty", func(t *testing.T) {
		svc := newTestService(t)
		tx, err := svc.repo.BeginTx(ctx)
		require.NoError(t, err)
		t.Cleanup(func() { _ = tx.Rollback(ctx) })
		ids, err := guard.ConsignmentProductIDs(ctx, tx, 999999)
		require.NoError(t, err)
		require.Len(t, ids, 0)
	})

	t.Run("returns distinct ordered product ids", func(t *testing.T) {
		p1 := insertTestProduct(ctx, t, "CG-P1")
		p2 := insertTestProduct(ctx, t, "CG-P2")
		svc, _, storeID := setupArrangement(t, p1, p2)
		saleID := insertTestSale(ctx, t, storeID, "CG-RET-1")

		arrID := arrID(t, svc, storeID)
		supplierID := getSupplierFromArr(ctx, t, svc, arrID)

		ins := func(pid int) {
			rec := shared.ConsignmentSaleRecord{
				SaleID:          saleID,
				InvoiceNumber:   "CG-RET-1",
				ProductID:       pid,
				SupplierID:      supplierID,
				ArrangementID:   arrID,
				StoreID:         storeID,
				Quantity:        1,
				UnitPrice:       10000,
				StoreShareType:  ShareTypePercentage,
				StoreShareValue: 20,
			}
			insertConsignmentSaleItem(ctx, t, svc, rec)
		}
		ins(p1)
		ins(p1)
		ins(p2)

		tx, err := svc.repo.BeginTx(ctx)
		require.NoError(t, err)
		t.Cleanup(func() { _ = tx.Rollback(ctx) })
		ids, err := guard.ConsignmentProductIDs(ctx, tx, saleID)
		require.NoError(t, err)
		require.Equal(t, []int{p1, p2}, ids)
	})
}

func txForTest(ctx context.Context, svc *Service, t *testing.T) pgx.Tx {
	t.Helper()
	tx, err := svc.repo.BeginTx(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback(ctx) })
	return tx
}

func getSupplierFromArr(ctx context.Context, t *testing.T, svc *Service, arrID int) int {
	t.Helper()
	var sup int
	err := svc.repo.db.QueryRow(ctx, `SELECT supplier_id FROM consignment_arrangements WHERE id = $1`, arrID).Scan(&sup)
	require.NoError(t, err)
	return sup
}
