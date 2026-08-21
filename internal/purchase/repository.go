package purchase

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

type Repository struct {
	db shared.DBPool
}

func NewRepository(db shared.DBPool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.Begin(ctx)
}

func (r *Repository) GetNextPONumber(ctx context.Context) (string, error) {
	var seq int
	err := r.db.QueryRow(ctx, `SELECT nextval('po_seq')`).Scan(&seq)
	if err != nil {
		return "", fmt.Errorf("failed to get next PO number: %w", err)
	}
	year := time.Now().In(shared.JakartaLocation()).Year()
	return fmt.Sprintf("PO-%d-%06d", year, seq), nil
}

func (r *Repository) GetNextGRNumber(ctx context.Context) (string, error) {
	var seq int
	err := r.db.QueryRow(ctx, `SELECT nextval('gr_seq')`).Scan(&seq)
	if err != nil {
		return "", fmt.Errorf("failed to get next GR number: %w", err)
	}
	year := time.Now().In(shared.JakartaLocation()).Year()
	return fmt.Sprintf("GR-%d-%06d", year, seq), nil
}

func (r *Repository) GetNextDONumber(ctx context.Context) (string, error) {
	var seq int
	err := r.db.QueryRow(ctx, `SELECT nextval('do_seq')`).Scan(&seq)
	if err != nil {
		return "", fmt.Errorf("failed to get next DO number: %w", err)
	}
	year := time.Now().In(shared.JakartaLocation()).Year()
	return fmt.Sprintf("DO-%d-%06d", year, seq), nil
}

func (r *Repository) CreatePurchaseOrder(ctx context.Context, tx pgx.Tx, po *Order, items []OrderItem) error {
	var createdAt, updatedAt time.Time
	err := tx.QueryRow(ctx, `
		INSERT INTO purchase_orders (
			po_number, supplier_id, store_id, warehouse_id, status, expected_date,
			payment_term, delivery_address, supplier_reference_number, notes,
			subtotal, grand_total, created_by, updated_by
		)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, '')::date, NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''), NULLIF($10, ''),
		        $11, $12, $13, $14)
		RETURNING id, created_at, updated_at
	`,
		po.PONumber, po.SupplierID, po.StoreID, po.WarehouseID, po.Status,
		po.ExpectedDate, po.PaymentTerm, po.DeliveryAddress, po.SupplierReferenceNumber, po.Notes,
		po.Subtotal, po.GrandTotal, po.CreatedBy, po.UpdatedBy,
	).Scan(&po.ID, &createdAt, &updatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert purchase order: %w", err)
	}
	po.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	po.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)

	if len(items) > 0 {
		rows := make([][]interface{}, len(items))
		for i, item := range items {
			rows[i] = []interface{}{
				po.ID, item.ProductID, item.QtyOrdered, item.UnitCost, item.DiscountAmount,
				item.Subtotal, item.ProductName, item.SKU, item.Barcode, item.UOMID, item.UOMName, item.Notes,
			}
		}
		_, err = tx.CopyFrom(ctx, pgx.Identifier{"purchase_order_items"},
			[]string{"purchase_order_id", "product_id", "qty_ordered", "unit_cost", "discount_amount",
				"subtotal", "product_name", "sku", "barcode", "uom_id", "uom_name", "notes"},
			pgx.CopyFromRows(rows),
		)
		if err != nil {
			return fmt.Errorf("batch insert purchase order items: %w", err)
		}
	}

	return nil
}

func (r *Repository) UpdatePurchaseOrder(ctx context.Context, tx pgx.Tx, po *Order, items []OrderItem) error {
	_, err := tx.Exec(ctx, `
		UPDATE purchase_orders
		SET supplier_id = $2, expected_date = NULLIF($3, '')::date, payment_term = NULLIF($4, ''), delivery_address = NULLIF($5, ''),
		    supplier_reference_number = NULLIF($6, ''), notes = NULLIF($7, ''),
		    subtotal = $8, grand_total = $9, updated_by = $10, updated_at = NOW()
		WHERE id = $1 AND status = 'draft'
	`,
		po.ID, po.SupplierID, po.ExpectedDate, po.PaymentTerm, po.DeliveryAddress,
		po.SupplierReferenceNumber, po.Notes, po.Subtotal, po.GrandTotal, po.UpdatedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to update purchase order: %w", err)
	}

	if len(items) > 0 {
		_, err = tx.Exec(ctx, `DELETE FROM purchase_order_items WHERE purchase_order_id = $1`, po.ID)
		if err != nil {
			return fmt.Errorf("failed to delete old items: %w", err)
		}
		rows := make([][]interface{}, len(items))
		for i, item := range items {
			rows[i] = []interface{}{
				po.ID, item.ProductID, item.QtyOrdered, item.UnitCost, item.DiscountAmount,
				item.Subtotal, item.ProductName, item.SKU, item.Barcode, item.UOMID, item.UOMName, item.Notes,
			}
		}
		_, err = tx.CopyFrom(ctx, pgx.Identifier{"purchase_order_items"},
			[]string{"purchase_order_id", "product_id", "qty_ordered", "unit_cost", "discount_amount",
				"subtotal", "product_name", "sku", "barcode", "uom_id", "uom_name", "notes"},
			pgx.CopyFromRows(rows),
		)
		if err != nil {
			return fmt.Errorf("batch insert purchase order items: %w", err)
		}
	}

	return nil
}

func (r *Repository) DeletePurchaseOrder(ctx context.Context, tx pgx.Tx, id int) error {
	_, err := tx.Exec(ctx, `DELETE FROM purchase_orders WHERE id = $1 AND status = 'draft'`, id)
	if err != nil {
		return fmt.Errorf("failed to delete purchase order: %w", err)
	}
	return nil
}

func (r *Repository) ConfirmPurchaseOrder(ctx context.Context, tx pgx.Tx, id, userID int, confirmedAt string) error {
	result, err := tx.Exec(ctx, `
		UPDATE purchase_orders
		SET status = 'confirmed', confirmed_at = $2, confirmed_by = $3, updated_at = NOW()
		WHERE id = $1 AND status = 'draft'
	`, id, confirmedAt, userID)
	if err != nil {
		return fmt.Errorf("failed to confirm purchase order: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrPurchaseOrderAlreadyConfirmed
	}
	return nil
}

func (r *Repository) CancelPurchaseOrder(ctx context.Context, tx pgx.Tx, id, userID int, cancelledAt string) error {
	result, err := tx.Exec(ctx, `
		UPDATE purchase_orders
		SET status = 'cancelled', cancelled_at = $2, cancelled_by = $3, updated_at = NOW()
		WHERE id = $1 AND status IN ('draft', 'confirmed')
		AND NOT EXISTS (SELECT 1 FROM purchase_order_items WHERE purchase_order_id = $1 AND qty_received > 0)
	`, id, cancelledAt, userID)
	if err != nil {
		return fmt.Errorf("failed to cancel purchase order: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("purchase order could not be cancelled; it may have receipts or an invalid status")
	}
	return nil
}

func (r *Repository) LockPurchaseOrderForUpdate(ctx context.Context, tx pgx.Tx, id int) error {
	var idVal int
	err := tx.QueryRow(ctx, `SELECT id FROM purchase_orders WHERE id = $1 FOR UPDATE`, id).Scan(&idVal)
	if err != nil {
		return fmt.Errorf("failed to lock purchase order: %w", err)
	}
	return nil
}

func (r *Repository) UpdatePOItemQtyReceived(ctx context.Context, tx pgx.Tx, poItemID, qtyReceived int) error {
	_, err := tx.Exec(ctx, `
		UPDATE purchase_order_items
		SET qty_received = qty_received + $2, updated_at = NOW()
		WHERE id = $1
	`, poItemID, qtyReceived)
	if err != nil {
		return fmt.Errorf("failed to update qty_received: %w", err)
	}
	return nil
}

func (r *Repository) RecalculatePOStatus(ctx context.Context, tx pgx.Tx, poID int) error {
	var totalOrdered, totalReceived int
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(qty_ordered), 0), COALESCE(SUM(qty_received), 0)
		FROM purchase_order_items
		WHERE purchase_order_id = $1
	`, poID).Scan(&totalOrdered, &totalReceived)
	if err != nil {
		return fmt.Errorf("failed to calculate totals: %w", err)
	}

	var newStatus string
	if totalReceived == 0 {
		newStatus = "confirmed"
	} else if totalReceived >= totalOrdered {
		newStatus = "fully_received"
	} else {
		newStatus = "partial_received"
	}

	_, err = tx.Exec(ctx, `
		UPDATE purchase_orders
		SET status = $2, updated_at = NOW()
		WHERE id = $1
	`, poID, newStatus)
	if err != nil {
		return fmt.Errorf("failed to update po status: %w", err)
	}
	return nil
}

func (r *Repository) CreateGoodsReceipt(ctx context.Context, tx pgx.Tx, gr *GoodsReceipt, items []GoodsReceiptItem) error {
	var createdAt time.Time
	err := tx.QueryRow(ctx, `
		INSERT INTO goods_receipts (
			gr_number, purchase_order_id, store_id, received_by,
			delivery_order_number, shipping_method, driver_name, vehicle_plate_number, notes
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at
	`,
		gr.GRNumber, gr.PurchaseOrderID, gr.StoreID, gr.ReceivedBy,
		gr.DeliveryOrderNumber, gr.ShippingMethod, gr.DriverName, gr.VehiclePlateNumber, gr.Notes,
	).Scan(&gr.ID, &createdAt)
	if err != nil {
		return fmt.Errorf("failed to insert goods receipt: %w", err)
	}
	gr.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)

	if len(items) > 0 {
		rows := make([][]interface{}, len(items))
		for i, item := range items {
			rows[i] = []interface{}{
				gr.ID, item.PurchaseOrderItemID, item.ProductID, item.QtyGood, item.QtyDamaged,
				item.UnitCost, item.ProductName, item.SupplierID, item.Notes,
			}
		}
		_, err = tx.CopyFrom(ctx, pgx.Identifier{"goods_receipt_items"},
			[]string{"goods_receipt_id", "purchase_order_item_id", "product_id", "qty_good", "qty_damaged",
				"unit_cost", "product_name", "supplier_id", "notes"},
			pgx.CopyFromRows(rows),
		)
		if err != nil {
			return fmt.Errorf("batch insert goods receipt items: %w", err)
		}
	}

	return nil
}

func (r *Repository) GetPurchaseOrderByID(ctx context.Context, id int, storeID *int) (*Order, error) {
	var po Order
	var warehouseID, confirmedBy, cancelledBy, approvedBy sql.NullInt64
	var expectedDate sql.NullTime
	var confirmedAt, cancelledAt, approvedAt sql.NullTime
	var paymentTerm, deliveryAddress, supplierRef, notes, approvalStatus, paymentStatus, invoiceStatus, currencyCode sql.NullString
	var exchangeRate sql.NullInt64
	var createdAt, updatedAt time.Time

	query := `
		SELECT po.id, po.po_number, po.supplier_id, po.store_id, po.warehouse_id, po.status, po.expected_date,
		       po.payment_term, po.delivery_address, po.supplier_reference_number,
		       po.approval_status, po.payment_status, po.invoice_status, po.currency_code, po.exchange_rate,
		       po.approved_by, po.approved_at,
		       po.subtotal, po.discount_amount, po.tax_amount, po.grand_total, po.notes,
		       po.confirmed_at, po.confirmed_by, po.cancelled_at, po.cancelled_by,
		       po.created_by, po.updated_by, po.created_at, po.updated_at
		FROM purchase_orders po
		WHERE po.id = $1
	`
	args := []interface{}{id}
	if storeID != nil {
		query += " AND po.store_id = $2"
		args = append(args, *storeID)
	}

	err := r.db.QueryRow(ctx, query, args...).Scan(
		&po.ID, &po.PONumber, &po.SupplierID, &po.StoreID, &warehouseID, &po.Status, &expectedDate,
		&paymentTerm, &deliveryAddress, &supplierRef,
		&approvalStatus, &paymentStatus, &invoiceStatus, &currencyCode, &exchangeRate,
		&approvedBy, &approvedAt,
		&po.Subtotal, &po.DiscountAmount, &po.TaxAmount, &po.GrandTotal, &notes,
		&confirmedAt, &confirmedBy, &cancelledAt, &cancelledBy,
		&po.CreatedBy, &po.UpdatedBy, &createdAt, &updatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrPurchaseOrderNotFound
		}
		return nil, err
	}

	if warehouseID.Valid {
		v := int(warehouseID.Int64)
		po.WarehouseID = &v
	}
	if confirmedBy.Valid {
		v := int(confirmedBy.Int64)
		po.ConfirmedBy = &v
	}
	if cancelledBy.Valid {
		v := int(cancelledBy.Int64)
		po.CancelledBy = &v
	}
	if approvedBy.Valid {
		v := int(approvedBy.Int64)
		po.ApprovedBy = &v
	}
	if expectedDate.Valid {
		po.ExpectedDate = expectedDate.Time.Format("2006-01-02")
	}
	if confirmedAt.Valid {
		po.ConfirmedAt = confirmedAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
	}
	if cancelledAt.Valid {
		po.CancelledAt = cancelledAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
	}
	if approvedAt.Valid {
		po.ApprovedAt = approvedAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
	}
	if notes.Valid {
		po.Notes = notes.String
	}
	if paymentTerm.Valid {
		po.PaymentTerm = paymentTerm.String
	}
	if deliveryAddress.Valid {
		po.DeliveryAddress = deliveryAddress.String
	}
	if supplierRef.Valid {
		po.SupplierReferenceNumber = supplierRef.String
	}
	if approvalStatus.Valid {
		po.ApprovalStatus = approvalStatus.String
	}
	if paymentStatus.Valid {
		po.PaymentStatus = paymentStatus.String
	}
	if invoiceStatus.Valid {
		po.InvoiceStatus = invoiceStatus.String
	}
	if currencyCode.Valid {
		po.CurrencyCode = currencyCode.String
	}
	if exchangeRate.Valid {
		po.ExchangeRate = int(exchangeRate.Int64)
	}

	po.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	po.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)

	items, err := r.getPOItems(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load po items: %w", err)
	}
	po.Items = items

	return &po, nil
}

func (r *Repository) getPOItems(ctx context.Context, poID int) ([]OrderItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, purchase_order_id, product_id, qty_ordered, qty_received,
		       unit_cost, discount_amount, subtotal, product_name, sku, barcode,
		       uom_id, uom_name, notes, created_at, updated_at
		FROM purchase_order_items
		WHERE purchase_order_id = $1
		ORDER BY id ASC
	`, poID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []OrderItem
	for rows.Next() {
		var item OrderItem
		var uomID sql.NullInt64
		var sku, barcode, notes sql.NullString
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&item.ID, &item.PurchaseOrderID, &item.ProductID, &item.QtyOrdered, &item.QtyReceived,
			&item.UnitCost, &item.DiscountAmount, &item.Subtotal, &item.ProductName,
			&sku, &barcode, &uomID, &item.UOMName, &notes, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, err
		}
		if sku.Valid {
			item.SKU = sku.String
		}
		if barcode.Valid {
			item.Barcode = barcode.String
		}
		if uomID.Valid {
			v := int(uomID.Int64)
			item.UOMID = &v
		}
		if notes.Valid {
			item.Notes = notes.String
		}
		item.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		item.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		items = append(items, item)
	}
	return items, nil
}

func (r *Repository) GetAllPurchaseOrders(ctx context.Context, limit, offset int, search, sortBy, sortDir, status, supplierID, startDate, endDate string, storeID *int, supplierIDs []int) ([]Order, int, error) {
	qb := shared.NewQueryBuilder()
	if search != "" {
		qb.AddClause(" AND (po.po_number ILIKE $%[1]d OR po.supplier_id = ANY($%[2]d))", "%"+search+"%", supplierIDs)
	}
	if status != "" {
		qb.AddClause(" AND po.status = $%d", status)
	}
	if supplierID != "" {
		qb.AddClause(" AND po.supplier_id = $%d", supplierID)
	}
	if storeID != nil {
		qb.AddClause(" AND po.store_id = $%d", *storeID)
	}
	if startDate != "" {
		if start, err := time.ParseInLocation("2006-01-02", startDate, shared.JakartaLocation()); err == nil {
			qb.AddClause(" AND po.created_at >= $%d", start)
		}
	}
	if endDate != "" {
		if end, err := time.ParseInLocation("2006-01-02", endDate, shared.JakartaLocation()); err == nil {
			qb.AddClause(" AND po.created_at < $%d", end.Add(24*time.Hour))
		}
	}

	allowedSortBy := map[string]bool{
		"created_at": true, "updated_at": true, "po_number": true, "status": true,
		"supplier_id": true, "expected_date": true, "grand_total": true,
	}
	if !allowedSortBy[sortBy] {
		sortBy = "created_at"
	}
	sortDir = strings.ToUpper(sortDir)
	if sortDir != "ASC" && sortDir != "DESC" {
		sortDir = "DESC"
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM purchase_orders po WHERE " + qb.Where()
	if err := r.db.QueryRow(ctx, countQuery, qb.Args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT po.id, po.po_number, po.supplier_id, po.store_id, po.status, po.expected_date,
		       po.payment_term,
		       po.subtotal, po.discount_amount, po.tax_amount, po.grand_total, po.notes,
		       po.confirmed_at, po.confirmed_by, po.cancelled_at, po.cancelled_by,
		       po.created_by, po.updated_by, po.created_at, po.updated_at
		FROM purchase_orders po
		WHERE %s
		ORDER BY po.%s %s
		LIMIT $%d OFFSET $%d
	`, qb.Where(), sortBy, sortDir, len(qb.Args)+1, len(qb.Args)+2)

	args := append(qb.Args, limit, offset)
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var pos []Order
	for rows.Next() {
		var po Order
		var confirmedBy, cancelledBy sql.NullInt64
		var confirmedAt, cancelledAt, expectedDate sql.NullTime
		var paymentTerm sql.NullString
		var notes sql.NullString
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&po.ID, &po.PONumber, &po.SupplierID, &po.StoreID, &po.Status, &expectedDate,
			&paymentTerm,
			&po.Subtotal, &po.DiscountAmount, &po.TaxAmount, &po.GrandTotal, &notes,
			&confirmedAt, &confirmedBy, &cancelledAt, &cancelledBy,
			&po.CreatedBy, &po.UpdatedBy, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		if paymentTerm.Valid {
			po.PaymentTerm = paymentTerm.String
		}
		if expectedDate.Valid {
			po.ExpectedDate = expectedDate.Time.Format("2006-01-02")
		}
		if notes.Valid {
			po.Notes = notes.String
		}
		if confirmedAt.Valid {
			po.ConfirmedAt = confirmedAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
		}
		if cancelledAt.Valid {
			po.CancelledAt = cancelledAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
		}
		if confirmedBy.Valid {
			v := int(confirmedBy.Int64)
			po.ConfirmedBy = &v
		}
		if cancelledBy.Valid {
			v := int(cancelledBy.Int64)
			po.CancelledBy = &v
		}
		po.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		po.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)

		pos = append(pos, po)
	}

	if len(pos) > 0 {
		poIDs := make([]int, len(pos))
		for i, po := range pos {
			poIDs[i] = po.ID
		}
		itemsMap, err := r.batchGetPOItems(ctx, poIDs)
		if err != nil {
			slog.Warn("failed to load po items", "error", err)
		} else {
			for i := range pos {
				pos[i].Items = itemsMap[pos[i].ID]
			}
		}
	}

	return pos, total, nil
}

func (r *Repository) batchGetPOItems(ctx context.Context, poIDs []int) (map[int][]OrderItem, error) {
	if len(poIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(poIDs))
	args := make([]interface{}, len(poIDs))
	for i, id := range poIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT id, purchase_order_id, product_id, qty_ordered, qty_received,
		       unit_cost, discount_amount, subtotal, product_name, sku, barcode,
		       uom_id, uom_name, notes, created_at, updated_at
		FROM purchase_order_items
		WHERE purchase_order_id IN (%s)
		ORDER BY id ASC
	`, strings.Join(placeholders, ",")), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	itemsMap := make(map[int][]OrderItem)
	for rows.Next() {
		var item OrderItem
		var uomID sql.NullInt64
		var sku, barcode, notes sql.NullString
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&item.ID, &item.PurchaseOrderID, &item.ProductID, &item.QtyOrdered, &item.QtyReceived,
			&item.UnitCost, &item.DiscountAmount, &item.Subtotal, &item.ProductName,
			&sku, &barcode, &uomID, &item.UOMName, &notes, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, err
		}
		if sku.Valid {
			item.SKU = sku.String
		}
		if barcode.Valid {
			item.Barcode = barcode.String
		}
		if uomID.Valid {
			v := int(uomID.Int64)
			item.UOMID = &v
		}
		if notes.Valid {
			item.Notes = notes.String
		}
		item.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		item.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		itemsMap[item.PurchaseOrderID] = append(itemsMap[item.PurchaseOrderID], item)
	}
	return itemsMap, nil
}

func (r *Repository) GetReceiptsByPOID(ctx context.Context, poID int, storeID *int) ([]GoodsReceipt, error) {
	if storeID != nil {
		var poStoreID int
		err := r.db.QueryRow(ctx, "SELECT store_id FROM purchase_orders WHERE id = $1", poID).Scan(&poStoreID)
		if err != nil {
			return nil, err
		}
		if poStoreID != *storeID {
			return nil, ErrPurchaseOrderNotFound
		}
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, gr_number, purchase_order_id, store_id, received_by, received_at,
		       delivery_order_number, shipping_method, driver_name, vehicle_plate_number,
		       notes, created_at
		FROM goods_receipts
		WHERE purchase_order_id = $1
		ORDER BY created_at ASC
	`, poID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var receipts []GoodsReceipt
	for rows.Next() {
		var gr GoodsReceipt
		var receivedBy int
		var receivedAt, createdAt time.Time
		var doNumber, shippingMethod, driverName, vehiclePlate, notes sql.NullString

		err := rows.Scan(
			&gr.ID, &gr.GRNumber, &gr.PurchaseOrderID, &gr.StoreID, &receivedBy, &receivedAt,
			&doNumber, &shippingMethod, &driverName, &vehiclePlate, &notes, &createdAt,
		)
		if err != nil {
			return nil, err
		}
		gr.ReceivedBy = receivedBy
		gr.ReceivedAt = receivedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		gr.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		if doNumber.Valid {
			gr.DeliveryOrderNumber = doNumber.String
		}
		if shippingMethod.Valid {
			gr.ShippingMethod = shippingMethod.String
		}
		if driverName.Valid {
			gr.DriverName = driverName.String
		}
		if vehiclePlate.Valid {
			gr.VehiclePlateNumber = vehiclePlate.String
		}
		if notes.Valid {
			gr.Notes = notes.String
		}
		receipts = append(receipts, gr)
	}

	if len(receipts) > 0 {
		grIDs := make([]int, len(receipts))
		for i, gr := range receipts {
			grIDs[i] = gr.ID
		}
		itemsMap, err := r.batchGetGRItems(ctx, grIDs)
		if err != nil {
			slog.Warn("failed to load gr items", "error", err)
		} else {
			for i := range receipts {
				receipts[i].Items = itemsMap[receipts[i].ID]
			}
		}
	}

	return receipts, nil
}

func (r *Repository) batchGetGRItems(ctx context.Context, grIDs []int) (map[int][]GoodsReceiptItem, error) {
	if len(grIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(grIDs))
	args := make([]interface{}, len(grIDs))
	for i, id := range grIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT id, goods_receipt_id, purchase_order_item_id, product_id,
		       qty_good, qty_damaged, unit_cost, product_name, supplier_id, notes, created_at
		FROM goods_receipt_items
		WHERE goods_receipt_id IN (%s)
		ORDER BY id ASC
	`, strings.Join(placeholders, ",")), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	itemsMap := make(map[int][]GoodsReceiptItem)
	for rows.Next() {
		var item GoodsReceiptItem
		var supplierID sql.NullInt64
		var notes sql.NullString
		var createdAt time.Time

		err := rows.Scan(
			&item.ID, &item.GoodsReceiptID, &item.PurchaseOrderItemID, &item.ProductID,
			&item.QtyGood, &item.QtyDamaged, &item.UnitCost, &item.ProductName,
			&supplierID, &notes, &createdAt,
		)
		if err != nil {
			return nil, err
		}
		if supplierID.Valid {
			v := int(supplierID.Int64)
			item.SupplierID = &v
		}
		if notes.Valid {
			item.Notes = notes.String
		}
		item.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		itemsMap[item.GoodsReceiptID] = append(itemsMap[item.GoodsReceiptID], item)
	}
	return itemsMap, nil
}

func (r *Repository) getGRItems(ctx context.Context, grID int) ([]GoodsReceiptItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, goods_receipt_id, purchase_order_item_id, product_id,
		       qty_good, qty_damaged, unit_cost, product_name, supplier_id, notes, created_at
		FROM goods_receipt_items
		WHERE goods_receipt_id = $1
		ORDER BY id ASC
	`, grID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []GoodsReceiptItem
	for rows.Next() {
		var item GoodsReceiptItem
		var supplierID sql.NullInt64
		var notes sql.NullString
		var createdAt time.Time

		err := rows.Scan(
			&item.ID, &item.GoodsReceiptID, &item.PurchaseOrderItemID, &item.ProductID,
			&item.QtyGood, &item.QtyDamaged, &item.UnitCost, &item.ProductName,
			&supplierID, &notes, &createdAt,
		)
		if err != nil {
			return nil, err
		}
		if supplierID.Valid {
			v := int(supplierID.Int64)
			item.SupplierID = &v
		}
		if notes.Valid {
			item.Notes = notes.String
		}
		item.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		items = append(items, item)
	}
	return items, nil
}
