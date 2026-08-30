package purchase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type noopBus struct{}

func (noopBus) Publish(ctx context.Context, topic string, event interface{}) error { return nil }

type fakeProductLookup struct {
	names map[int]ProductInfo
	err   error
}

func (l fakeProductLookup) GetProductNamesByIDs(ctx context.Context, ids []int) (map[int]ProductInfo, error) {
	if l.err != nil {
		return nil, l.err
	}
	result := make(map[int]ProductInfo, len(ids))
	for _, id := range ids {
		if info, ok := l.names[id]; ok {
			result[id] = info
		}
	}
	return result, nil
}

func newMockSvc(t *testing.T) (pgxmock.PgxPoolIface, Service, context.Context) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { mock.Close() })
	svc := NewService(NewRepository(mock), noopBus{})
	svc.SetProductLookup(fakeProductLookup{names: map[int]ProductInfo{1: {Name: "Produk", SKU: "SKU-1"}}})
	return mock, svc, context.Background()
}

var poColumns = []string{
	"id", "po_number", "supplier_id", "store_id", "warehouse_id", "status", "expected_date",
	"payment_term", "delivery_address", "supplier_reference_number",
	"approval_status", "payment_status", "invoice_status", "currency_code", "exchange_rate",
	"approved_by", "approved_at",
	"subtotal", "discount_amount", "tax_amount", "grand_total", "notes",
	"confirmed_at", "confirmed_by", "cancelled_at", "cancelled_by",
	"created_by", "updated_by", "created_at", "updated_at",
}

var poItemColumns = []string{
	"id", "purchase_order_id", "product_id", "qty_ordered", "qty_received",
	"unit_cost", "discount_amount", "subtotal", "product_name", "sku", "barcode",
	"uom_id", "uom_name", "notes", "created_at", "updated_at",
}

func poRow(now time.Time, status string) *pgxmock.Rows {
	return pgxmock.NewRows(poColumns).AddRow(1, "PO-2026-000001", 2, 3, nil, status, nil,
		nil, nil, nil,
		nil, nil, nil, nil, nil,
		nil, nil,
		100000, 0, 0, 100000, nil,
		nil, nil, nil, nil,
		8, 9, now, now)
}

func poItemsRow(now time.Time, id, productID, qtyOrdered, qtyReceived int) *pgxmock.Rows {
	return pgxmock.NewRows(poItemColumns).AddRow(id, 1, productID, qtyOrdered, qtyReceived,
		10000, 0, 100000, "Produk", "SKU-1", "8901",
		nil, "pcs", nil, now, now)
}

func expectPOFetch(mock pgxmock.PgxPoolIface, now time.Time, status string, withItems bool) {
	mock.ExpectQuery("SELECT po.id, po.po_number").WithArgs(1).WillReturnRows(poRow(now, status))
	if withItems {
		mock.ExpectQuery("FROM purchase_order_items").WithArgs(1).WillReturnRows(poItemsRow(now, 10, 1, 10, 0))
	} else {
		mock.ExpectQuery("FROM purchase_order_items").WithArgs(1).WillReturnRows(pgxmock.NewRows(poItemColumns))
	}
}

func TestServiceMock_CreateDraft_Errors(t *testing.T) {
	boom := errors.New("boom")
	now := time.Now()

	t.Run("product lookup error", func(t *testing.T) {
		_, svc, ctx := newMockSvc(t)
		svc.SetProductLookup(fakeProductLookup{err: boom})
		err := svc.CreateDraft(ctx, &Order{SupplierID: 2, StoreID: 3}, []OrderItem{{ProductID: 1, QtyOrdered: 5, UnitCost: 100}})
		assert.ErrorContains(t, err, "lookup products")
	})

	t.Run("begin error", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		mock.ExpectBegin().WillReturnError(boom)
		err := svc.CreateDraft(ctx, &Order{SupplierID: 2, StoreID: 3}, []OrderItem{{ProductID: 1, QtyOrdered: 5, UnitCost: 100}})
		assert.ErrorContains(t, err, "begin transaction")
	})

	t.Run("po number error", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT nextval\\('po_seq'\\)").WillReturnError(boom)
		err := svc.CreateDraft(ctx, &Order{SupplierID: 2, StoreID: 3}, []OrderItem{{ProductID: 1, QtyOrdered: 5, UnitCost: 100}})
		assert.ErrorContains(t, err, "failed to get next PO number")
	})

	t.Run("create po error", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT nextval\\('po_seq'\\)").WillReturnRows(pgxmock.NewRows([]string{"nextval"}).AddRow(1))
		mock.ExpectQuery("INSERT INTO purchase_orders").WithArgs(anyArgs(14)...).WillReturnError(boom)
		err := svc.CreateDraft(ctx, &Order{SupplierID: 2, StoreID: 3}, []OrderItem{{ProductID: 1, QtyOrdered: 5, UnitCost: 100}})
		assert.ErrorContains(t, err, "failed to insert purchase order")
	})

	t.Run("commit error", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT nextval\\('po_seq'\\)").WillReturnRows(pgxmock.NewRows([]string{"nextval"}).AddRow(1))
		mock.ExpectQuery("INSERT INTO purchase_orders").WithArgs(anyArgs(14)...).WillReturnRows(
			pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(1, now, now))
		mock.ExpectCopyFrom(pgx.Identifier{"purchase_order_items"}, []string{
			"purchase_order_id", "product_id", "qty_ordered", "unit_cost", "discount_amount",
			"subtotal", "product_name", "sku", "barcode", "uom_id", "uom_name", "notes"})
		mock.ExpectCommit().WillReturnError(boom)
		err := svc.CreateDraft(ctx, &Order{SupplierID: 2, StoreID: 3}, []OrderItem{{ProductID: 1, QtyOrdered: 5, UnitCost: 100}})
		assert.ErrorContains(t, err, "commit transaction")
	})
}

func TestServiceMock_UpdateDraft_Errors(t *testing.T) {
	boom := errors.New("boom")
	now := time.Now()

	t.Run("product lookup error", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		expectPOFetch(mock, now, StatusDraft, true)
		svc.SetProductLookup(fakeProductLookup{err: boom})
		err := svc.UpdateDraft(ctx, 1, &Order{SupplierID: 2, UpdatedBy: 9}, []OrderItem{{ProductID: 1, QtyOrdered: 5, UnitCost: 100}})
		assert.ErrorContains(t, err, "lookup products")
	})

	t.Run("begin error", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		expectPOFetch(mock, now, StatusDraft, true)
		mock.ExpectBegin().WillReturnError(boom)
		err := svc.UpdateDraft(ctx, 1, &Order{SupplierID: 2, UpdatedBy: 9}, []OrderItem{{ProductID: 1, QtyOrdered: 5, UnitCost: 100}})
		assert.ErrorContains(t, err, "begin transaction")
	})

	t.Run("update po error", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		expectPOFetch(mock, now, StatusDraft, true)
		mock.ExpectBegin()
		mock.ExpectExec("UPDATE purchase_orders").WithArgs(anyArgs(10)...).WillReturnError(boom)
		err := svc.UpdateDraft(ctx, 1, &Order{SupplierID: 2, UpdatedBy: 9}, []OrderItem{{ProductID: 1, QtyOrdered: 5, UnitCost: 100}})
		assert.ErrorContains(t, err, "failed to update purchase order")
	})

	t.Run("commit error", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		expectPOFetch(mock, now, StatusDraft, true)
		mock.ExpectBegin()
		mock.ExpectExec("UPDATE purchase_orders").WithArgs(anyArgs(10)...).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec("DELETE FROM purchase_order_items").WithArgs(1).WillReturnResult(pgxmock.NewResult("DELETE", 1))
		mock.ExpectCopyFrom(pgx.Identifier{"purchase_order_items"}, []string{
			"purchase_order_id", "product_id", "qty_ordered", "unit_cost", "discount_amount",
			"subtotal", "product_name", "sku", "barcode", "uom_id", "uom_name", "notes"})
		mock.ExpectCommit().WillReturnError(boom)
		err := svc.UpdateDraft(ctx, 1, &Order{SupplierID: 2, UpdatedBy: 9}, []OrderItem{{ProductID: 1, QtyOrdered: 5, UnitCost: 100}})
		assert.ErrorContains(t, err, "commit transaction")
	})
}

func TestServiceMock_DeleteDraft_Errors(t *testing.T) {
	boom := errors.New("boom")
	now := time.Now()

	t.Run("begin error", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		expectPOFetch(mock, now, StatusDraft, false)
		mock.ExpectBegin().WillReturnError(boom)
		err := svc.DeleteDraft(ctx, 1)
		assert.ErrorContains(t, err, "begin transaction")
	})

	t.Run("delete error", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		expectPOFetch(mock, now, StatusDraft, false)
		mock.ExpectBegin()
		mock.ExpectExec("DELETE FROM purchase_orders").WithArgs(1).WillReturnError(boom)
		err := svc.DeleteDraft(ctx, 1)
		assert.ErrorContains(t, err, "failed to delete purchase order")
	})

	t.Run("commit error", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		expectPOFetch(mock, now, StatusDraft, false)
		mock.ExpectBegin()
		mock.ExpectExec("DELETE FROM purchase_orders").WithArgs(1).WillReturnResult(pgxmock.NewResult("DELETE", 1))
		mock.ExpectCommit().WillReturnError(boom)
		err := svc.DeleteDraft(ctx, 1)
		assert.ErrorContains(t, err, "commit transaction")
	})
}

func TestServiceMock_Confirm_Errors(t *testing.T) {
	boom := errors.New("boom")
	now := time.Now()

	t.Run("begin error", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		mock.ExpectBegin().WillReturnError(boom)
		err := svc.Confirm(ctx, 1, 9)
		assert.ErrorContains(t, err, "begin transaction")
	})

	t.Run("lock error", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		mock.ExpectBegin()
		expectPOFetch(mock, now, StatusDraft, false)
		mock.ExpectQuery("SELECT id FROM purchase_orders").WithArgs(1).WillReturnError(boom)
		err := svc.Confirm(ctx, 1, 9)
		assert.ErrorContains(t, err, "failed to lock purchase order")
	})

	t.Run("confirm error", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		mock.ExpectBegin()
		expectPOFetch(mock, now, StatusDraft, false)
		mock.ExpectQuery("SELECT id FROM purchase_orders").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectExec("SET status = 'confirmed'").WithArgs(anyArgs(3)...).WillReturnError(boom)
		err := svc.Confirm(ctx, 1, 9)
		assert.ErrorContains(t, err, "failed to confirm purchase order")
	})

	t.Run("commit error", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		mock.ExpectBegin()
		expectPOFetch(mock, now, StatusDraft, false)
		mock.ExpectQuery("SELECT id FROM purchase_orders").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectExec("SET status = 'confirmed'").WithArgs(anyArgs(3)...).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit().WillReturnError(boom)
		err := svc.Confirm(ctx, 1, 9)
		assert.ErrorContains(t, err, "boom")
	})
}

func TestServiceMock_Cancel_Errors(t *testing.T) {
	boom := errors.New("boom")
	now := time.Now()

	t.Run("begin error", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		mock.ExpectBegin().WillReturnError(boom)
		err := svc.Cancel(ctx, 1, 9)
		assert.ErrorContains(t, err, "begin transaction")
	})

	t.Run("lock error", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		mock.ExpectBegin()
		expectPOFetch(mock, now, StatusConfirmed, true)
		mock.ExpectQuery("SELECT id FROM purchase_orders").WithArgs(1).WillReturnError(boom)
		err := svc.Cancel(ctx, 1, 9)
		assert.ErrorContains(t, err, "failed to lock purchase order")
	})

	t.Run("cancel error", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		mock.ExpectBegin()
		expectPOFetch(mock, now, StatusConfirmed, true)
		mock.ExpectQuery("SELECT id FROM purchase_orders").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectExec("SET status = 'cancelled'").WithArgs(anyArgs(3)...).WillReturnError(boom)
		err := svc.Cancel(ctx, 1, 9)
		assert.ErrorContains(t, err, "failed to cancel purchase order")
	})

	t.Run("commit error", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		mock.ExpectBegin()
		expectPOFetch(mock, now, StatusConfirmed, true)
		mock.ExpectQuery("SELECT id FROM purchase_orders").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectExec("SET status = 'cancelled'").WithArgs(anyArgs(3)...).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit().WillReturnError(boom)
		err := svc.Cancel(ctx, 1, 9)
		assert.ErrorContains(t, err, "boom")
	})
}

func TestServiceMock_CreateGoodsReceipt_Errors(t *testing.T) {
	boom := errors.New("boom")
	now := time.Now()

	t.Run("po fetch error", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		mock.ExpectQuery("SELECT po.id, po.po_number").WithArgs(1).WillReturnError(boom)
		_, err := svc.CreateGoodsReceipt(ctx, 1, 9, 3, []CreateGRItemInput{{PurchaseOrderItemID: 10, QtyGood: 5}})
		assert.Error(t, err)
	})

	t.Run("begin error", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		expectPOFetch(mock, now, StatusConfirmed, true)
		mock.ExpectBegin().WillReturnError(boom)
		_, err := svc.CreateGoodsReceipt(ctx, 1, 9, 3, []CreateGRItemInput{{PurchaseOrderItemID: 10, QtyGood: 5}})
		assert.ErrorContains(t, err, "begin transaction")
	})

	t.Run("lock error", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		expectPOFetch(mock, now, StatusConfirmed, true)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id FROM purchase_orders").WithArgs(1).WillReturnError(boom)
		_, err := svc.CreateGoodsReceipt(ctx, 1, 9, 3, []CreateGRItemInput{{PurchaseOrderItemID: 10, QtyGood: 5}})
		assert.ErrorContains(t, err, "failed to lock purchase order")
	})

	t.Run("second po fetch error", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		expectPOFetch(mock, now, StatusConfirmed, true)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id FROM purchase_orders").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectQuery("SELECT po.id, po.po_number").WithArgs(1).WillReturnError(boom)
		_, err := svc.CreateGoodsReceipt(ctx, 1, 9, 3, []CreateGRItemInput{{PurchaseOrderItemID: 10, QtyGood: 5}})
		assert.Error(t, err)
	})

	t.Run("second po status invalid", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		expectPOFetch(mock, now, StatusConfirmed, true)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id FROM purchase_orders").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectQuery("SELECT po.id, po.po_number").WithArgs(1).WillReturnRows(poRow(now, StatusCancelled))
		mock.ExpectQuery("FROM purchase_order_items").WithArgs(1).WillReturnRows(pgxmock.NewRows(poItemColumns))
		_, err := svc.CreateGoodsReceipt(ctx, 1, 9, 3, []CreateGRItemInput{{PurchaseOrderItemID: 10, QtyGood: 5}})
		assert.ErrorIs(t, err, ErrInvalidPOStatusForReceiving)
	})

	t.Run("gr number error", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		expectPOFetch(mock, now, StatusConfirmed, true)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id FROM purchase_orders").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{"id"}).AddRow(1))
		expectPOFetch(mock, now, StatusConfirmed, true)
		mock.ExpectQuery("SELECT nextval\\('gr_seq'\\)").WillReturnError(boom)
		_, err := svc.CreateGoodsReceipt(ctx, 1, 9, 3, []CreateGRItemInput{{PurchaseOrderItemID: 10, QtyGood: 5}})
		assert.ErrorContains(t, err, "failed to get next GR number")
	})

	t.Run("do number error", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		expectPOFetch(mock, now, StatusConfirmed, true)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id FROM purchase_orders").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{"id"}).AddRow(1))
		expectPOFetch(mock, now, StatusConfirmed, true)
		mock.ExpectQuery("SELECT nextval\\('gr_seq'\\)").WillReturnRows(pgxmock.NewRows([]string{"nextval"}).AddRow(1))
		mock.ExpectQuery("SELECT nextval\\('do_seq'\\)").WillReturnError(boom)
		_, err := svc.CreateGoodsReceipt(ctx, 1, 9, 3, []CreateGRItemInput{{PurchaseOrderItemID: 10, QtyGood: 5}})
		assert.ErrorContains(t, err, "failed to get next DO number")
	})

	t.Run("negative qty", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		expectPOFetch(mock, now, StatusConfirmed, true)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id FROM purchase_orders").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{"id"}).AddRow(1))
		expectPOFetch(mock, now, StatusConfirmed, true)
		mock.ExpectQuery("SELECT nextval\\('gr_seq'\\)").WillReturnRows(pgxmock.NewRows([]string{"nextval"}).AddRow(1))
		mock.ExpectQuery("SELECT nextval\\('do_seq'\\)").WillReturnRows(pgxmock.NewRows([]string{"nextval"}).AddRow(1))
		_, err := svc.CreateGoodsReceipt(ctx, 1, 9, 3, []CreateGRItemInput{{PurchaseOrderItemID: 10, QtyGood: -1}})
		assert.ErrorIs(t, err, ErrInvalidReceivingQty)
	})

	t.Run("no items after loop", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		expectPOFetch(mock, now, StatusConfirmed, true)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id FROM purchase_orders").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{"id"}).AddRow(1))
		expectPOFetch(mock, now, StatusConfirmed, true)
		mock.ExpectQuery("SELECT nextval\\('gr_seq'\\)").WillReturnRows(pgxmock.NewRows([]string{"nextval"}).AddRow(1))
		mock.ExpectQuery("SELECT nextval\\('do_seq'\\)").WillReturnRows(pgxmock.NewRows([]string{"nextval"}).AddRow(1))
		_, err := svc.CreateGoodsReceipt(ctx, 1, 9, 3, nil)
		assert.ErrorIs(t, err, ErrNoItemsToReceive)
	})

	t.Run("create gr error", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		expectPOFetch(mock, now, StatusConfirmed, true)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id FROM purchase_orders").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{"id"}).AddRow(1))
		expectPOFetch(mock, now, StatusConfirmed, true)
		mock.ExpectQuery("SELECT nextval\\('gr_seq'\\)").WillReturnRows(pgxmock.NewRows([]string{"nextval"}).AddRow(1))
		mock.ExpectQuery("SELECT nextval\\('do_seq'\\)").WillReturnRows(pgxmock.NewRows([]string{"nextval"}).AddRow(1))
		mock.ExpectQuery("INSERT INTO goods_receipts").WithArgs(anyArgs(9)...).WillReturnError(boom)
		_, err := svc.CreateGoodsReceipt(ctx, 1, 9, 3, []CreateGRItemInput{{PurchaseOrderItemID: 10, QtyGood: 5}})
		assert.ErrorContains(t, err, "failed to insert goods receipt")
	})

	t.Run("update qty error", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		expectPOFetch(mock, now, StatusConfirmed, true)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id FROM purchase_orders").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{"id"}).AddRow(1))
		expectPOFetch(mock, now, StatusConfirmed, true)
		mock.ExpectQuery("SELECT nextval\\('gr_seq'\\)").WillReturnRows(pgxmock.NewRows([]string{"nextval"}).AddRow(1))
		mock.ExpectQuery("SELECT nextval\\('do_seq'\\)").WillReturnRows(pgxmock.NewRows([]string{"nextval"}).AddRow(1))
		mock.ExpectQuery("INSERT INTO goods_receipts").WithArgs(anyArgs(9)...).WillReturnRows(
			pgxmock.NewRows([]string{"id", "created_at"}).AddRow(1, now))
		mock.ExpectCopyFrom(pgx.Identifier{"goods_receipt_items"}, []string{
			"goods_receipt_id", "purchase_order_item_id", "product_id", "qty_good", "qty_damaged",
			"unit_cost", "product_name", "supplier_id", "notes"})
		mock.ExpectExec("UPDATE purchase_order_items.*SET qty_received").WithArgs(10, 5).WillReturnError(boom)
		_, err := svc.CreateGoodsReceipt(ctx, 1, 9, 3, []CreateGRItemInput{{PurchaseOrderItemID: 10, QtyGood: 5}})
		assert.ErrorContains(t, err, "bulk update po item qty received")
	})

	t.Run("recalculate error", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		expectPOFetch(mock, now, StatusConfirmed, true)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id FROM purchase_orders").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{"id"}).AddRow(1))
		expectPOFetch(mock, now, StatusConfirmed, true)
		mock.ExpectQuery("SELECT nextval\\('gr_seq'\\)").WillReturnRows(pgxmock.NewRows([]string{"nextval"}).AddRow(1))
		mock.ExpectQuery("SELECT nextval\\('do_seq'\\)").WillReturnRows(pgxmock.NewRows([]string{"nextval"}).AddRow(1))
		mock.ExpectQuery("INSERT INTO goods_receipts").WithArgs(anyArgs(9)...).WillReturnRows(
			pgxmock.NewRows([]string{"id", "created_at"}).AddRow(1, now))
		mock.ExpectCopyFrom(pgx.Identifier{"goods_receipt_items"}, []string{
			"goods_receipt_id", "purchase_order_item_id", "product_id", "qty_good", "qty_damaged",
			"unit_cost", "product_name", "supplier_id", "notes"})
		mock.ExpectExec("UPDATE purchase_order_items.*SET qty_received").WithArgs(10, 5).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectQuery("SELECT COALESCE\\(SUM").WithArgs(1).WillReturnError(boom)
		_, err := svc.CreateGoodsReceipt(ctx, 1, 9, 3, []CreateGRItemInput{{PurchaseOrderItemID: 10, QtyGood: 5}})
		assert.ErrorContains(t, err, "failed to calculate totals")
	})

	t.Run("commit error", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		expectPOFetch(mock, now, StatusConfirmed, true)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id FROM purchase_orders").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{"id"}).AddRow(1))
		expectPOFetch(mock, now, StatusConfirmed, true)
		mock.ExpectQuery("SELECT nextval\\('gr_seq'\\)").WillReturnRows(pgxmock.NewRows([]string{"nextval"}).AddRow(1))
		mock.ExpectQuery("SELECT nextval\\('do_seq'\\)").WillReturnRows(pgxmock.NewRows([]string{"nextval"}).AddRow(1))
		mock.ExpectQuery("INSERT INTO goods_receipts").WithArgs(anyArgs(9)...).WillReturnRows(
			pgxmock.NewRows([]string{"id", "created_at"}).AddRow(1, now))
		mock.ExpectCopyFrom(pgx.Identifier{"goods_receipt_items"}, []string{
			"goods_receipt_id", "purchase_order_item_id", "product_id", "qty_good", "qty_damaged",
			"unit_cost", "product_name", "supplier_id", "notes"})
		mock.ExpectExec("UPDATE purchase_order_items.*SET qty_received").WithArgs(10, 5).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectQuery("SELECT COALESCE\\(SUM").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{"total_ordered", "total_received"}).AddRow(10, 5))
		mock.ExpectExec("SET status = \\$2").WithArgs(1, StatusPartialReceived).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit().WillReturnError(boom)
		_, err := svc.CreateGoodsReceipt(ctx, 1, 9, 3, []CreateGRItemInput{{PurchaseOrderItemID: 10, QtyGood: 5}})
		assert.ErrorContains(t, err, "commit transaction")
	})

	t.Run("success", func(t *testing.T) {
		mock, svc, ctx := newMockSvc(t)
		notes := "rush"
		expectPOFetch(mock, now, StatusConfirmed, true)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id FROM purchase_orders").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{"id"}).AddRow(1))
		expectPOFetch(mock, now, StatusConfirmed, true)
		mock.ExpectQuery("SELECT nextval\\('gr_seq'\\)").WillReturnRows(pgxmock.NewRows([]string{"nextval"}).AddRow(1))
		mock.ExpectQuery("SELECT nextval\\('do_seq'\\)").WillReturnRows(pgxmock.NewRows([]string{"nextval"}).AddRow(1))
		mock.ExpectQuery("INSERT INTO goods_receipts").WithArgs(anyArgs(9)...).WillReturnRows(
			pgxmock.NewRows([]string{"id", "created_at"}).AddRow(1, now))
		mock.ExpectCopyFrom(pgx.Identifier{"goods_receipt_items"}, []string{
			"goods_receipt_id", "purchase_order_item_id", "product_id", "qty_good", "qty_damaged",
			"unit_cost", "product_name", "supplier_id", "notes"})
		mock.ExpectExec("UPDATE purchase_order_items.*SET qty_received").WithArgs(10, 5).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectQuery("SELECT COALESCE\\(SUM").WithArgs(1).WillReturnRows(
			pgxmock.NewRows([]string{"total_ordered", "total_received"}).AddRow(10, 5))
		mock.ExpectExec("SET status = \\$2").WithArgs(1, StatusPartialReceived).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit()

		gr, err := svc.CreateGoodsReceipt(ctx, 1, 9, 3, []CreateGRItemInput{{PurchaseOrderItemID: 10, QtyGood: 5, Notes: &notes}})
		require.NoError(t, err)
		require.NotNil(t, gr)
		assert.Equal(t, "GR-2026-000001", gr.GRNumber)
		require.Len(t, gr.Items, 1)
		assert.Equal(t, "rush", gr.Items[0].Notes)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
