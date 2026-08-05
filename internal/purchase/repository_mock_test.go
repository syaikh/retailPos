package purchase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jackc/pgx/v5"
)

func newPurchaseMockRepo(t *testing.T) (pgxmock.PgxPoolIface, *Repository, context.Context) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { mock.Close() })
	return mock, NewRepository(mock), context.Background()
}

func anyArgs(n int) []any {
	args := make([]any, n)
	for i := range args {
		args[i] = pgxmock.AnyArg()
	}
	return args
}

func TestRepositoryMock_SequenceAndTxErrors(t *testing.T) {
	boom := errors.New("boom")

	cases := []struct {
		name string
		run  func(mock pgxmock.PgxPoolIface, repo *Repository, ctx context.Context)
	}{
		{"get next po number error", func(mock pgxmock.PgxPoolIface, repo *Repository, ctx context.Context) {
			mock.ExpectQuery("SELECT nextval\\('po_seq'\\)").WillReturnError(boom)
			_, err := repo.GetNextPONumber(ctx)
			assert.ErrorContains(t, err, "failed to get next PO number")
		}},
		{"get next gr number error", func(mock pgxmock.PgxPoolIface, repo *Repository, ctx context.Context) {
			mock.ExpectQuery("SELECT nextval\\('gr_seq'\\)").WillReturnError(boom)
			_, err := repo.GetNextGRNumber(ctx)
			assert.ErrorContains(t, err, "failed to get next GR number")
		}},
		{"get next do number error", func(mock pgxmock.PgxPoolIface, repo *Repository, ctx context.Context) {
			mock.ExpectQuery("SELECT nextval\\('do_seq'\\)").WillReturnError(boom)
			_, err := repo.GetNextDONumber(ctx)
			assert.ErrorContains(t, err, "failed to get next DO number")
		}},
		{"begin tx error", func(mock pgxmock.PgxPoolIface, repo *Repository, ctx context.Context) {
			mock.ExpectBegin().WillReturnError(boom)
			_, err := repo.BeginTx(ctx)
			assert.Error(t, err)
		}},
		{"create po insert error", func(mock pgxmock.PgxPoolIface, repo *Repository, ctx context.Context) {
			mock.ExpectBegin()
			tx, err := repo.BeginTx(ctx)
			require.NoError(t, err)
			mock.ExpectQuery("INSERT INTO purchase_orders").WithArgs(anyArgs(14)...).WillReturnError(boom)
			err = repo.CreatePurchaseOrder(ctx, tx, &PurchaseOrder{}, nil)
			assert.ErrorContains(t, err, "failed to insert purchase order")
		}},
		{"update po exec error", func(mock pgxmock.PgxPoolIface, repo *Repository, ctx context.Context) {
			mock.ExpectBegin()
			tx, err := repo.BeginTx(ctx)
			require.NoError(t, err)
			mock.ExpectExec("UPDATE purchase_orders").WithArgs(anyArgs(10)...).WillReturnError(boom)
			err = repo.UpdatePurchaseOrder(ctx, tx, &PurchaseOrder{}, nil)
			assert.ErrorContains(t, err, "failed to update purchase order")
		}},
		{"update po delete items error", func(mock pgxmock.PgxPoolIface, repo *Repository, ctx context.Context) {
			mock.ExpectBegin()
			tx, err := repo.BeginTx(ctx)
			require.NoError(t, err)
			mock.ExpectExec("UPDATE purchase_orders").WithArgs(anyArgs(10)...).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			mock.ExpectExec("DELETE FROM purchase_order_items").WithArgs(1).WillReturnError(boom)
			err = repo.UpdatePurchaseOrder(ctx, tx, &PurchaseOrder{ID: 1}, []PurchaseOrderItem{{}})
			assert.ErrorContains(t, err, "failed to delete old items")
		}},
		{"delete po error", func(mock pgxmock.PgxPoolIface, repo *Repository, ctx context.Context) {
			mock.ExpectBegin()
			tx, err := repo.BeginTx(ctx)
			require.NoError(t, err)
			mock.ExpectExec("DELETE FROM purchase_orders").WithArgs(1).WillReturnError(boom)
			err = repo.DeletePurchaseOrder(ctx, tx, 1)
			assert.ErrorContains(t, err, "failed to delete purchase order")
		}},
		{"confirm po error", func(mock pgxmock.PgxPoolIface, repo *Repository, ctx context.Context) {
			mock.ExpectBegin()
			tx, err := repo.BeginTx(ctx)
			require.NoError(t, err)
			mock.ExpectExec("SET status = 'confirmed'").WithArgs(anyArgs(3)...).WillReturnError(boom)
			err = repo.ConfirmPurchaseOrder(ctx, tx, 1, 1, "2026-01-01T00:00:00Z")
			assert.ErrorContains(t, err, "failed to confirm purchase order")
		}},
		{"confirm po no rows", func(mock pgxmock.PgxPoolIface, repo *Repository, ctx context.Context) {
			mock.ExpectBegin()
			tx, err := repo.BeginTx(ctx)
			require.NoError(t, err)
			mock.ExpectExec("SET status = 'confirmed'").WithArgs(anyArgs(3)...).WillReturnResult(pgxmock.NewResult("UPDATE", 0))
			err = repo.ConfirmPurchaseOrder(ctx, tx, 1, 1, "2026-01-01T00:00:00Z")
			assert.ErrorIs(t, err, ErrPurchaseOrderAlreadyConfirmed)
		}},
		{"cancel po error", func(mock pgxmock.PgxPoolIface, repo *Repository, ctx context.Context) {
			mock.ExpectBegin()
			tx, err := repo.BeginTx(ctx)
			require.NoError(t, err)
			mock.ExpectExec("SET status = 'cancelled'").WithArgs(anyArgs(3)...).WillReturnError(boom)
			err = repo.CancelPurchaseOrder(ctx, tx, 1, 1, "2026-01-01T00:00:00Z")
			assert.ErrorContains(t, err, "failed to cancel purchase order")
		}},
		{"cancel po no rows", func(mock pgxmock.PgxPoolIface, repo *Repository, ctx context.Context) {
			mock.ExpectBegin()
			tx, err := repo.BeginTx(ctx)
			require.NoError(t, err)
			mock.ExpectExec("SET status = 'cancelled'").WithArgs(anyArgs(3)...).WillReturnResult(pgxmock.NewResult("UPDATE", 0))
			err = repo.CancelPurchaseOrder(ctx, tx, 1, 1, "2026-01-01T00:00:00Z")
			assert.ErrorContains(t, err, "could not be cancelled")
		}},
		{"lock po error", func(mock pgxmock.PgxPoolIface, repo *Repository, ctx context.Context) {
			mock.ExpectBegin()
			tx, err := repo.BeginTx(ctx)
			require.NoError(t, err)
			mock.ExpectQuery("SELECT id FROM purchase_orders").WithArgs(1).WillReturnError(boom)
			err = repo.LockPurchaseOrderForUpdate(ctx, tx, 1)
			assert.ErrorContains(t, err, "failed to lock purchase order")
		}},
		{"update qty received error", func(mock pgxmock.PgxPoolIface, repo *Repository, ctx context.Context) {
			mock.ExpectBegin()
			tx, err := repo.BeginTx(ctx)
			require.NoError(t, err)
			mock.ExpectExec("SET qty_received = qty_received").WithArgs(1, 5).WillReturnError(boom)
			err = repo.UpdatePOItemQtyReceived(ctx, tx, 1, 5)
			assert.ErrorContains(t, err, "failed to update qty_received")
		}},
		{"recalculate totals error", func(mock pgxmock.PgxPoolIface, repo *Repository, ctx context.Context) {
			mock.ExpectBegin()
			tx, err := repo.BeginTx(ctx)
			require.NoError(t, err)
			mock.ExpectQuery("SELECT COALESCE\\(SUM").WithArgs(1).WillReturnError(boom)
			err = repo.RecalculatePOStatus(ctx, tx, 1)
			assert.ErrorContains(t, err, "failed to calculate totals")
		}},
		{"recalculate update error", func(mock pgxmock.PgxPoolIface, repo *Repository, ctx context.Context) {
			mock.ExpectBegin()
			tx, err := repo.BeginTx(ctx)
			require.NoError(t, err)
			mock.ExpectQuery("SELECT COALESCE\\(SUM").WithArgs(1).WillReturnRows(
				pgxmock.NewRows([]string{"total_ordered", "total_received"}).AddRow(10, 3))
			mock.ExpectExec("SET status = $2").WithArgs(1, "partial_received").WillReturnError(boom)
			err = repo.RecalculatePOStatus(ctx, tx, 1)
			assert.ErrorContains(t, err, "failed to update po status")
		}},
		{"create goods receipt insert error", func(mock pgxmock.PgxPoolIface, repo *Repository, ctx context.Context) {
			mock.ExpectBegin()
			tx, err := repo.BeginTx(ctx)
			require.NoError(t, err)
			mock.ExpectQuery("INSERT INTO goods_receipts").WithArgs(anyArgs(9)...).WillReturnError(boom)
			err = repo.CreateGoodsReceipt(ctx, tx, &GoodsReceipt{}, nil)
			assert.ErrorContains(t, err, "failed to insert goods receipt")
		}},
		{"create po copyfrom error", func(mock pgxmock.PgxPoolIface, repo *Repository, ctx context.Context) {
			mock.ExpectBegin()
			tx, err := repo.BeginTx(ctx)
			require.NoError(t, err)
			mock.ExpectQuery("INSERT INTO purchase_orders").WithArgs(anyArgs(14)...).WillReturnRows(
				pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(1, time.Now(), time.Now()))
			err = repo.CreatePurchaseOrder(ctx, tx, &PurchaseOrder{}, []PurchaseOrderItem{{}})
			assert.ErrorContains(t, err, "batch insert purchase order items")
		}},
		{"update po copyfrom error", func(mock pgxmock.PgxPoolIface, repo *Repository, ctx context.Context) {
			mock.ExpectBegin()
			tx, err := repo.BeginTx(ctx)
			require.NoError(t, err)
			mock.ExpectExec("UPDATE purchase_orders").WithArgs(anyArgs(10)...).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			mock.ExpectExec("DELETE FROM purchase_order_items").WithArgs(1).WillReturnResult(pgxmock.NewResult("DELETE", 1))
			err = repo.UpdatePurchaseOrder(ctx, tx, &PurchaseOrder{ID: 1}, []PurchaseOrderItem{{}})
			assert.ErrorContains(t, err, "batch insert purchase order items")
		}},
		{"create goods receipt copyfrom error", func(mock pgxmock.PgxPoolIface, repo *Repository, ctx context.Context) {
			mock.ExpectBegin()
			tx, err := repo.BeginTx(ctx)
			require.NoError(t, err)
			mock.ExpectQuery("INSERT INTO goods_receipts").WithArgs(anyArgs(9)...).WillReturnRows(
				pgxmock.NewRows([]string{"id", "created_at"}).AddRow(1, time.Now()))
			err = repo.CreateGoodsReceipt(ctx, tx, &GoodsReceipt{}, []GoodsReceiptItem{{}})
			assert.ErrorContains(t, err, "batch insert goods receipt items")
		}},
		{"recalculate confirmed", func(mock pgxmock.PgxPoolIface, repo *Repository, ctx context.Context) {
			mock.ExpectBegin()
			tx, err := repo.BeginTx(ctx)
			require.NoError(t, err)
			mock.ExpectQuery("SELECT COALESCE\\(SUM").WithArgs(1).WillReturnRows(
				pgxmock.NewRows([]string{"total_ordered", "total_received"}).AddRow(10, 0))
			mock.ExpectExec("SET status = \\$2").WithArgs(1, "confirmed").WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			err = repo.RecalculatePOStatus(ctx, tx, 1)
			assert.NoError(t, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		}},
		{"recalculate fully received", func(mock pgxmock.PgxPoolIface, repo *Repository, ctx context.Context) {
			mock.ExpectBegin()
			tx, err := repo.BeginTx(ctx)
			require.NoError(t, err)
			mock.ExpectQuery("SELECT COALESCE\\(SUM").WithArgs(1).WillReturnRows(
				pgxmock.NewRows([]string{"total_ordered", "total_received"}).AddRow(10, 10))
			mock.ExpectExec("SET status = \\$2").WithArgs(1, "fully_received").WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			err = repo.RecalculatePOStatus(ctx, tx, 1)
			assert.NoError(t, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mock, repo, ctx := newPurchaseMockRepo(t)
			c.run(mock, repo, ctx)
		})
	}
}

func TestRepositoryMock_GetPurchaseOrderByID(t *testing.T) {
	boom := errors.New("boom")
	now := time.Now()

	poRow := func() *pgxmock.Rows {
		return pgxmock.NewRows([]string{
			"id", "po_number", "supplier_id", "store_id", "warehouse_id", "status", "expected_date",
			"payment_term", "delivery_address", "supplier_reference_number",
			"approval_status", "payment_status", "invoice_status", "currency_code", "exchange_rate",
			"approved_by", "approved_at",
			"subtotal", "discount_amount", "tax_amount", "grand_total", "notes",
			"confirmed_at", "confirmed_by", "cancelled_at", "cancelled_by",
			"created_by", "updated_by", "created_at", "updated_at",
		}).AddRow(1, "PO-2026-000001", 2, 3, 4, "draft", now,
			"30 days", "Jl. Sudirman", "REF-1",
			"approved", "paid", "unpaid", "IDR", 15000,
			5, now,
			100000, 0, 0, 100000, "note",
			now, 6, now, 7,
			8, 9, now, now)
	}

	poItemRow := func() *pgxmock.Rows {
		return pgxmock.NewRows([]string{
			"id", "purchase_order_id", "product_id", "qty_ordered", "qty_received",
			"unit_cost", "discount_amount", "subtotal", "product_name", "sku", "barcode",
			"uom_id", "uom_name", "notes", "created_at", "updated_at",
		}).AddRow(1, 1, 2, 10, 0, 10000, 0, 100000, "Produk", "SKU-1", "8901",
			3, "pcs", "note", now, now)
	}

	t.Run("full data", func(t *testing.T) {
		mock, repo, ctx := newPurchaseMockRepo(t)
		storeID := 3
		mock.ExpectQuery("SELECT po.id, po.po_number").WithArgs(1, 3).WillReturnRows(poRow())
		mock.ExpectQuery("FROM purchase_order_items").WithArgs(1).WillReturnRows(poItemRow())

		po, err := repo.GetPurchaseOrderByID(ctx, 1, &storeID)
		require.NoError(t, err)
		require.NotNil(t, po)
		require.NotNil(t, po.WarehouseID)
		assert.Equal(t, 4, *po.WarehouseID)
		require.NotNil(t, po.ApprovedBy)
		assert.Equal(t, 5, *po.ApprovedBy)
		assert.Equal(t, now.Format("2006-01-02"), po.ExpectedDate)
		assert.NotEmpty(t, po.ApprovedAt)
		assert.Equal(t, "note", po.Notes)
		assert.Equal(t, "30 days", po.PaymentTerm)
		assert.Equal(t, "Jl. Sudirman", po.DeliveryAddress)
		assert.Equal(t, "REF-1", po.SupplierReferenceNumber)
		require.NotNil(t, po.ConfirmedBy)
		assert.Equal(t, 6, *po.ConfirmedBy)
		require.NotNil(t, po.CancelledBy)
		assert.Equal(t, 7, *po.CancelledBy)
		require.Len(t, po.Items, 1)
		require.NotNil(t, po.Items[0].UOMID)
		assert.Equal(t, 3, *po.Items[0].UOMID)
		assert.Equal(t, "SKU-1", po.Items[0].SKU)
		assert.Equal(t, "note", po.Items[0].Notes)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock, repo, ctx := newPurchaseMockRepo(t)
		mock.ExpectQuery("SELECT po.id, po.po_number").WithArgs(1).WillReturnError(pgx.ErrNoRows)
		_, err := repo.GetPurchaseOrderByID(ctx, 1, nil)
		assert.ErrorIs(t, err, ErrPurchaseOrderNotFound)
	})

	t.Run("query error", func(t *testing.T) {
		mock, repo, ctx := newPurchaseMockRepo(t)
		mock.ExpectQuery("SELECT po.id, po.po_number").WithArgs(1).WillReturnError(boom)
		_, err := repo.GetPurchaseOrderByID(ctx, 1, nil)
		assert.Equal(t, boom, err)
	})

	t.Run("items error", func(t *testing.T) {
		mock, repo, ctx := newPurchaseMockRepo(t)
		mock.ExpectQuery("SELECT po.id, po.po_number").WithArgs(1).WillReturnRows(poRow())
		mock.ExpectQuery("FROM purchase_order_items").WithArgs(1).WillReturnError(boom)
		_, err := repo.GetPurchaseOrderByID(ctx, 1, nil)
		assert.ErrorContains(t, err, "failed to load po items")
	})

	t.Run("items scan error", func(t *testing.T) {
		mock, repo, ctx := newPurchaseMockRepo(t)
		mock.ExpectQuery("SELECT po.id, po.po_number").WithArgs(1).WillReturnRows(poRow())
		mock.ExpectQuery("FROM purchase_order_items").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{"id", "purchase_order_id", "product_id", "qty_ordered", "qty_received",
				"unit_cost", "discount_amount", "subtotal", "product_name", "sku", "barcode",
				"uom_id", "uom_name", "notes", "created_at"}).AddRow(1, 1, 2, 10, 0, 10000, 0, 100000, "P", "S", "B", 3, "pcs", "n", now))
		_, err := repo.GetPurchaseOrderByID(ctx, 1, nil)
		assert.Error(t, err)
	})
}

func TestRepositoryMock_GetAllPurchaseOrders(t *testing.T) {
	boom := errors.New("boom")
	now := time.Now()

	listRow := func() *pgxmock.Rows {
		return pgxmock.NewRows([]string{
			"id", "po_number", "supplier_id", "store_id", "status", "expected_date",
			"payment_term",
			"subtotal", "discount_amount", "tax_amount", "grand_total", "notes",
			"confirmed_at", "confirmed_by", "cancelled_at", "cancelled_by",
			"created_by", "updated_by", "created_at", "updated_at",
		}).AddRow(1, "PO-2026-000001", 2, 3, "confirmed", now,
			"30 days",
			100000, 0, 0, 100000, "note",
			now, 6, nil, nil,
			8, 9, now, now)
	}

	itemRow := func() *pgxmock.Rows {
		return pgxmock.NewRows([]string{
			"id", "purchase_order_id", "product_id", "qty_ordered", "qty_received",
			"unit_cost", "discount_amount", "subtotal", "product_name", "sku", "barcode",
			"uom_id", "uom_name", "notes", "created_at", "updated_at",
		}).AddRow(1, 1, 2, 10, 0, 10000, 0, 100000, "Produk", "SKU-1", "8901",
			3, "pcs", "note", now, now)
	}

	t.Run("full with filters", func(t *testing.T) {
		mock, repo, ctx := newPurchaseMockRepo(t)
		mock.ExpectQuery("SELECT COUNT").WithArgs(anyArgs(7)...).WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery("SELECT po.id, po.po_number").WithArgs(anyArgs(9)...).WillReturnRows(listRow())
		mock.ExpectQuery("FROM purchase_order_items").WithArgs(1).WillReturnRows(itemRow())

		storeID := 3
		pos, total, err := repo.GetAllPurchaseOrders(ctx, 10, 0, "PO", "po_number", "ASC", "confirmed", "2",
			"2026-01-01", "2026-01-31", &storeID, []int{2})
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		require.Len(t, pos, 1)
		assert.Equal(t, "30 days", pos[0].PaymentTerm)
		assert.Equal(t, now.Format("2006-01-02"), pos[0].ExpectedDate)
		assert.Equal(t, "note", pos[0].Notes)
		assert.NotEmpty(t, pos[0].ConfirmedAt)
		require.Len(t, pos[0].Items, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count error", func(t *testing.T) {
		mock, repo, ctx := newPurchaseMockRepo(t)
		mock.ExpectQuery("SELECT COUNT").WillReturnError(boom)
		_, _, err := repo.GetAllPurchaseOrders(ctx, 10, 0, "", "", "", "", "", "", "", nil, nil)
		assert.Error(t, err)
	})

	t.Run("list query error", func(t *testing.T) {
		mock, repo, ctx := newPurchaseMockRepo(t)
		mock.ExpectQuery("SELECT COUNT").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery("SELECT po.id, po.po_number").WithArgs(anyArgs(2)...).WillReturnError(boom)
		_, _, err := repo.GetAllPurchaseOrders(ctx, 10, 0, "", "", "", "", "", "", "", nil, nil)
		assert.Error(t, err)
	})

	t.Run("scan error", func(t *testing.T) {
		mock, repo, ctx := newPurchaseMockRepo(t)
		mock.ExpectQuery("SELECT COUNT").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery("SELECT po.id, po.po_number").WithArgs(anyArgs(2)...).WillReturnRows(
			pgxmock.NewRows([]string{"id", "po_number", "supplier_id", "store_id", "status", "expected_date",
				"payment_term",
				"subtotal", "discount_amount", "tax_amount", "grand_total", "notes",
				"confirmed_at", "confirmed_by", "cancelled_at", "cancelled_by",
				"created_by", "updated_by", "created_at"}).AddRow(1, "PO", 2, 3, "confirmed", now,
				"30 days", 100000, 0, 0, 100000, "note", now, 6, nil, nil, 8, 9, now))
		_, _, err := repo.GetAllPurchaseOrders(ctx, 10, 0, "", "", "", "", "", "", "", nil, nil)
		assert.Error(t, err)
	})

	t.Run("batch items error", func(t *testing.T) {
		mock, repo, ctx := newPurchaseMockRepo(t)
		mock.ExpectQuery("SELECT COUNT").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery("SELECT po.id, po.po_number").WithArgs(anyArgs(2)...).WillReturnRows(listRow())
		mock.ExpectQuery("FROM purchase_order_items").WithArgs(1).WillReturnError(boom)
		pos, _, err := repo.GetAllPurchaseOrders(ctx, 10, 0, "", "", "", "", "", "", "", nil, nil)
		require.NoError(t, err)
		require.Len(t, pos, 1)
		assert.Empty(t, pos[0].Items)
	})

	t.Run("empty result", func(t *testing.T) {
		mock, repo, ctx := newPurchaseMockRepo(t)
		mock.ExpectQuery("SELECT COUNT").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery("SELECT po.id, po.po_number").WithArgs(anyArgs(2)...).WillReturnRows(pgxmock.NewRows([]string{"id"}))
		pos, total, err := repo.GetAllPurchaseOrders(ctx, 10, 0, "", "", "", "", "", "", "", nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, pos)
	})
}

func TestRepositoryMock_BatchGetPOItems(t *testing.T) {
	boom := errors.New("boom")
	now := time.Now()

	t.Run("empty ids", func(t *testing.T) {
		_, repo, ctx := newPurchaseMockRepo(t)
		res, err := repo.batchGetPOItems(ctx, nil)
		require.NoError(t, err)
		assert.Nil(t, res)
	})

	t.Run("query error", func(t *testing.T) {
		mock, repo, ctx := newPurchaseMockRepo(t)
		mock.ExpectQuery("FROM purchase_order_items").WithArgs(1, 2).WillReturnError(boom)
		_, err := repo.batchGetPOItems(ctx, []int{1, 2})
		assert.Error(t, err)
	})

	t.Run("scan error", func(t *testing.T) {
		mock, repo, ctx := newPurchaseMockRepo(t)
		mock.ExpectQuery("FROM purchase_order_items").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{"id", "purchase_order_id", "product_id", "qty_ordered", "qty_received",
				"unit_cost", "discount_amount", "subtotal", "product_name", "sku", "barcode",
				"uom_id", "uom_name", "notes", "created_at"}).AddRow(1, 1, 2, 10, 0, 10000, 0, 100000, "P", "S", "B", 3, "pcs", "n", now))
		_, err := repo.batchGetPOItems(ctx, []int{1})
		assert.Error(t, err)
	})
}

func TestRepositoryMock_GetReceiptsByPOID(t *testing.T) {
	boom := errors.New("boom")
	now := time.Now()

	grRow := func() *pgxmock.Rows {
		return pgxmock.NewRows([]string{
			"id", "gr_number", "purchase_order_id", "store_id", "received_by", "received_at",
			"delivery_order_number", "shipping_method", "driver_name", "vehicle_plate_number",
			"notes", "created_at",
		}).AddRow(1, "GR-2026-000001", 1, 3, 5, now,
			"DO-1", "Truck", "Budi", "B 1234",
			"note", now)
	}

	t.Run("full", func(t *testing.T) {
		mock, repo, ctx := newPurchaseMockRepo(t)
		mock.ExpectQuery("SELECT id, gr_number").WithArgs(1).WillReturnRows(grRow())
		mock.ExpectQuery("FROM goods_receipt_items").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{
				"id", "goods_receipt_id", "purchase_order_item_id", "product_id",
				"qty_good", "qty_damaged", "unit_cost", "product_name", "supplier_id", "notes", "created_at",
			}).AddRow(1, 1, 2, 3, 5, 0, 10000, "Produk", 9, "note", now))

		receipts, err := repo.GetReceiptsByPOID(ctx, 1, nil)
		require.NoError(t, err)
		require.Len(t, receipts, 1)
		assert.Equal(t, "DO-1", receipts[0].DeliveryOrderNumber)
		assert.Equal(t, "note", receipts[0].Notes)
		require.Len(t, receipts[0].Items, 1)
		require.NotNil(t, receipts[0].Items[0].SupplierID)
		assert.Equal(t, 9, *receipts[0].Items[0].SupplierID)
		assert.Equal(t, "note", receipts[0].Items[0].Notes)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("store mismatch", func(t *testing.T) {
		mock, repo, ctx := newPurchaseMockRepo(t)
		storeID := 99
		mock.ExpectQuery("SELECT store_id FROM purchase_orders").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{"store_id"}).AddRow(3))
		_, err := repo.GetReceiptsByPOID(ctx, 1, &storeID)
		assert.ErrorIs(t, err, ErrPurchaseOrderNotFound)
	})

	t.Run("store check query error", func(t *testing.T) {
		mock, repo, ctx := newPurchaseMockRepo(t)
		storeID := 3
		mock.ExpectQuery("SELECT store_id FROM purchase_orders").WithArgs(1).WillReturnError(boom)
		_, err := repo.GetReceiptsByPOID(ctx, 1, &storeID)
		assert.Error(t, err)
	})

	t.Run("list query error", func(t *testing.T) {
		mock, repo, ctx := newPurchaseMockRepo(t)
		mock.ExpectQuery("SELECT id, gr_number").WithArgs(1).WillReturnError(boom)
		_, err := repo.GetReceiptsByPOID(ctx, 1, nil)
		assert.Error(t, err)
	})

	t.Run("scan error", func(t *testing.T) {
		mock, repo, ctx := newPurchaseMockRepo(t)
		mock.ExpectQuery("SELECT id, gr_number").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{"id", "gr_number", "purchase_order_id", "store_id", "received_by", "received_at",
				"delivery_order_number", "shipping_method", "driver_name", "vehicle_plate_number", "notes"}).AddRow(1, "GR", 1, 3, 5, now, "DO", "T", "B", "P", "n"))
		_, err := repo.GetReceiptsByPOID(ctx, 1, nil)
		assert.Error(t, err)
	})

	t.Run("batch items error swallowed", func(t *testing.T) {
		mock, repo, ctx := newPurchaseMockRepo(t)
		mock.ExpectQuery("SELECT id, gr_number").WithArgs(1).WillReturnRows(grRow())
		mock.ExpectQuery("FROM goods_receipt_items").WithArgs(1).WillReturnError(boom)
		receipts, err := repo.GetReceiptsByPOID(ctx, 1, nil)
		require.NoError(t, err)
		require.Len(t, receipts, 1)
		assert.Empty(t, receipts[0].Items)
	})
}

func TestRepositoryMock_BatchGetGRItems(t *testing.T) {
	boom := errors.New("boom")

	t.Run("empty ids", func(t *testing.T) {
		_, repo, ctx := newPurchaseMockRepo(t)
		res, err := repo.batchGetGRItems(ctx, nil)
		require.NoError(t, err)
		assert.Nil(t, res)
	})

	t.Run("query error", func(t *testing.T) {
		mock, repo, ctx := newPurchaseMockRepo(t)
		mock.ExpectQuery("FROM goods_receipt_items").WithArgs(1, 2).WillReturnError(boom)
		_, err := repo.batchGetGRItems(ctx, []int{1, 2})
		assert.Error(t, err)
	})

	t.Run("scan error", func(t *testing.T) {
		mock, repo, ctx := newPurchaseMockRepo(t)
		mock.ExpectQuery("FROM goods_receipt_items").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{
				"id", "goods_receipt_id", "purchase_order_item_id", "product_id",
				"qty_good", "qty_damaged", "unit_cost", "product_name", "supplier_id", "notes",
			}).AddRow(1, 1, 2, 3, 5, 0, 10000, "Produk", 9, "note"))
		_, err := repo.batchGetGRItems(ctx, []int{1})
		assert.Error(t, err)
	})
}

func TestRepositoryMock_GetGRItems(t *testing.T) {
	boom := errors.New("boom")
	now := time.Now()

	t.Run("query error", func(t *testing.T) {
		mock, repo, ctx := newPurchaseMockRepo(t)
		mock.ExpectQuery("FROM goods_receipt_items").WithArgs(1).WillReturnError(boom)
		_, err := repo.getGRItems(ctx, 1)
		assert.Error(t, err)
	})

	t.Run("scan error", func(t *testing.T) {
		mock, repo, ctx := newPurchaseMockRepo(t)
		mock.ExpectQuery("FROM goods_receipt_items").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{
				"id", "goods_receipt_id", "purchase_order_item_id", "product_id",
				"qty_good", "qty_damaged", "unit_cost", "product_name", "supplier_id", "notes",
			}).AddRow(1, 1, 2, 3, 5, 0, 10000, "Produk", 9, "note"))
		_, err := repo.getGRItems(ctx, 1)
		assert.Error(t, err)
	})

	t.Run("full", func(t *testing.T) {
		mock, repo, ctx := newPurchaseMockRepo(t)
		mock.ExpectQuery("FROM goods_receipt_items").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{
				"id", "goods_receipt_id", "purchase_order_item_id", "product_id",
				"qty_good", "qty_damaged", "unit_cost", "product_name", "supplier_id", "notes", "created_at",
			}).AddRow(1, 1, 2, 3, 5, 0, 10000, "Produk", 9, "note", now))

		items, err := repo.getGRItems(ctx, 1)
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.NotNil(t, items[0].SupplierID)
		assert.Equal(t, 9, *items[0].SupplierID)
		assert.Equal(t, "note", items[0].Notes)
		assert.NotEmpty(t, items[0].CreatedAt)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("nulls", func(t *testing.T) {
		mock, repo, ctx := newPurchaseMockRepo(t)
		mock.ExpectQuery("FROM goods_receipt_items").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{
				"id", "goods_receipt_id", "purchase_order_item_id", "product_id",
				"qty_good", "qty_damaged", "unit_cost", "product_name", "supplier_id", "notes", "created_at",
			}).AddRow(1, 1, 2, 3, 5, 0, 10000, "Produk", nil, nil, now))

		items, err := repo.getGRItems(ctx, 1)
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Nil(t, items[0].SupplierID)
		assert.Empty(t, items[0].Notes)
	})
}
