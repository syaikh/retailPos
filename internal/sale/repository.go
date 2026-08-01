package sale

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strconv"
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

func (r *Repository) CreateSale(ctx context.Context, tx pgx.Tx, sale *Sale, items []SaleItem) error {
	var createdAt, updatedAt time.Time
	err := tx.QueryRow(ctx, `
		INSERT INTO sales (invoice_number, cashier_id, store_id, customer_id, shift_id, subtotal, discount, tax, total_amount, payment_method, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`, sale.InvoiceNumber, sale.CashierID, sale.StoreID, sale.CustomerID, sale.ShiftID, sale.Subtotal, sale.Discount, sale.Tax, sale.TotalAmount, sale.PaymentMethod, sale.Status).
		Scan(&sale.ID, &createdAt, &updatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert sale: %w", err)
	}
	sale.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	sale.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)

	if len(items) > 0 {
		rows := make([][]interface{}, len(items))
		for i, item := range items {
			origPrice := item.UnitPrice
			if item.OriginalPrice != nil {
				origPrice = *item.OriginalPrice
			}
			productName := item.Name
			if item.ProductName != "" {
				productName = item.ProductName
			}
			rows[i] = []interface{}{
				sale.ID, item.ProductID, item.Quantity, item.UnitPrice, item.Subtotal,
				item.DPPAmount, item.TaxAmount,
				item.PricingRuleID, item.PricingRuleName, item.PricingRuleType, item.PricingType,
				origPrice,
				item.Cost, item.TaxClassID, item.TaxRate,
				time.Now(), productName,
			}
		}
		_, err = tx.CopyFrom(ctx, pgx.Identifier{"sale_items"},
			[]string{"sale_id", "product_id", "quantity", "unit_price", "subtotal", "dpp_amount", "tax_amount",
				"pricing_rule_id", "pricing_rule_name", "pricing_rule_type", "pricing_type", "original_price",
				"cost", "tax_class_id", "tax_rate", "snapshot_created_at", "product_name"},
			pgx.CopyFromRows(rows),
		)
		if err != nil {
			return fmt.Errorf("batch insert sale items: %w", err)
		}
	}

	return nil
}

func (r *Repository) CreateSalePayments(ctx context.Context, tx pgx.Tx, saleID int, payments []SalePayment) error {
	if len(payments) == 0 {
		return nil
	}
	rows := make([][]interface{}, len(payments))
	for i, p := range payments {
		var refNum interface{}
		if p.ReferenceNumber != "" {
			refNum = p.ReferenceNumber
		}
		rows[i] = []interface{}{
			saleID, p.PaymentMethodID, p.PaymentMethodCode, p.Amount, refNum,
		}
	}
	_, err := tx.CopyFrom(ctx, pgx.Identifier{"sale_payments"},
		[]string{"sale_id", "payment_method_id", "payment_method_code", "amount", "reference_number"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("batch insert sale payments: %w", err)
	}
	return nil
}

func (r *Repository) UpdateShiftTotals(ctx context.Context, tx pgx.Tx, shiftID int, totalAmount int, payments []SalePayment) error {
	cashSales := 0
	nonCashSales := 0
	for _, p := range payments {
		if strings.EqualFold(p.PaymentMethodCode, "CASH") {
			cashSales += p.Amount
		} else {
			nonCashSales += p.Amount
		}
	}
	_, err := tx.Exec(ctx, `
		UPDATE shifts
		SET cash_sales = cash_sales + $1,
		    non_cash_sales = non_cash_sales + $2,
		    total_sales = total_sales + $3,
		    transaction_count = transaction_count + 1,
		    updated_at = NOW()
		WHERE id = $4 AND status = 'open'
	`, cashSales, nonCashSales, totalAmount, shiftID)
	return err
}

func (r *Repository) GetSaleByID(ctx context.Context, id int, storeID *int) (*Sale, error) {
	var sale Sale
	var itemsJSON []byte
	var paymentsJSON []byte
	var createdAt, updatedAt time.Time

	query := `
		SELECT s.id, s.invoice_number, s.cashier_id, s.customer_id, s.store_id,
		       s.subtotal, s.discount, s.tax, s.total_amount, s.payment_method, s.status,
		       s.created_at, s.updated_at, COALESCE(c.name, '') as customer_name,
		       COALESCE(
		           (SELECT jsonb_agg(jsonb_build_object(
		               'id', si.id, 'sale_id', si.sale_id, 'product_id', si.product_id,
		               'name', COALESCE(si.product_name, p.name), 'quantity', si.quantity, 'unit_price', si.unit_price,
		               'subtotal', si.subtotal, 'dpp_amount', si.dpp_amount, 'tax_amount', si.tax_amount,
		               'pricing_rule_id', si.pricing_rule_id, 'pricing_rule_name', si.pricing_rule_name,
		               'pricing_rule_type', si.pricing_rule_type, 'pricing_type', si.pricing_type,
		               'original_price', si.original_price,
		               'cost', si.cost, 'tax_class_id', si.tax_class_id, 'tax_rate', si.tax_rate,
		               'snapshot_created_at', si.snapshot_created_at, 'product_name', si.product_name
		           )) FROM sale_items si
		           JOIN products p ON si.product_id = p.id
		           WHERE si.sale_id = s.id),
		       '[]'::jsonb
		   ) as items,
		   COALESCE(
		           (SELECT jsonb_agg(jsonb_build_object(
		               'id', sp.id, 'sale_id', sp.sale_id,
		               'payment_method_id', sp.payment_method_id,
		               'payment_method_code', sp.payment_method_code,
		               'amount', sp.amount,
		               'reference_number', sp.reference_number,
		               'created_at', sp.created_at
		           )) FROM sale_payments sp
		           WHERE sp.sale_id = s.id),
		       '[]'::jsonb
		   ) as payments
		FROM sales s
		LEFT JOIN customers c ON s.customer_id = c.id
		WHERE s.id = $1`
	args := []interface{}{id}
	if storeID != nil {
		query += fmt.Sprintf(" AND s.store_id = $%d", len(args)+1)
		args = append(args, *storeID)
	}

	err := r.db.QueryRow(ctx, query, args...).Scan(
		&sale.ID, &sale.InvoiceNumber, &sale.CashierID, &sale.CustomerID, &sale.StoreID,
		&sale.Subtotal, &sale.Discount, &sale.Tax,
		&sale.TotalAmount, &sale.PaymentMethod, &sale.Status,
		&createdAt, &updatedAt, &sale.CustomerName, &itemsJSON, &paymentsJSON,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrSaleNotFound
		}
		return nil, err
	}
	sale.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	sale.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)

	if len(itemsJSON) > 0 {
		if err := json.Unmarshal(itemsJSON, &sale.Items); err != nil {
			slog.Warn("failed to unmarshal sale items", "sale_id", sale.ID, "error", err)
		}
	}
	if len(paymentsJSON) > 0 {
		if err := json.Unmarshal(paymentsJSON, &sale.Payments); err != nil {
			slog.Warn("failed to unmarshal sale payments", "sale_id", sale.ID, "error", err)
		}
	}

	return &sale, nil
}

func (r *Repository) buildSaleFilter(search, startDate, endDate string, storeID *int, paymentMethods string, minTotal, maxTotal, cashierID *int) *shared.QueryBuilder {
	qb := shared.NewQueryBuilder()
	if search != "" {
		qb.AddClause(" AND (s.invoice_number ILIKE $%[1]d OR s.id IN (SELECT DISTINCT si.sale_id FROM sale_items si JOIN products p ON si.product_id = p.id WHERE p.name ILIKE $%[1]d) OR s.customer_id IN (SELECT c2.id FROM customers c2 WHERE c2.name ILIKE $%[1]d))", "%"+search+"%")
	}
	if startDate != "" {
		if start, err := time.ParseInLocation("2006-01-02", startDate, shared.JakartaLocation()); err == nil {
			qb.AddClause(" AND s.created_at >= $%d", start)
		}
	}
	if endDate != "" {
		if end, err := time.ParseInLocation("2006-01-02", endDate, shared.JakartaLocation()); err == nil {
			qb.AddClause(" AND s.created_at < $%d", end.Add(24*time.Hour))
		}
	}
	if storeID != nil {
		qb.AddClause(" AND s.store_id = $%d", *storeID)
	}
	if paymentMethods != "" {
		qb.AddClause(" AND s.payment_method = ANY(string_to_array($%d, ','))", paymentMethods)
	}
	if minTotal != nil {
		qb.AddClause(" AND s.total_amount >= $%d", *minTotal)
	}
	if maxTotal != nil {
		qb.AddClause(" AND s.total_amount <= $%d", *maxTotal)
	}
	if cashierID != nil {
		qb.AddClause(" AND s.cashier_id = $%d", *cashierID)
	}
	return qb
}

func (r *Repository) GetAllSales(ctx context.Context, limit, offset int, search string, sortBy, sortDir, startDate, endDate string, storeID *int, paymentMethods string, minTotal, maxTotal, cashierID *int) ([]Sale, int, error) {
	var sales []Sale
	var total int

	qb := r.buildSaleFilter(search, startDate, endDate, storeID, paymentMethods, minTotal, maxTotal, cashierID)
	countQuery := "SELECT COUNT(*) FROM sales s WHERE " + qb.Where()
	err := r.db.QueryRow(ctx, countQuery, qb.Args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `SELECT s.id, s.invoice_number, s.cashier_id, s.store_id, s.subtotal, s.discount, s.tax, s.total_amount, s.payment_method, s.status, s.created_at, s.updated_at, COALESCE(c.name, '') as customer_name
		FROM sales s
		LEFT JOIN customers c ON s.customer_id = c.id
		WHERE ` + qb.Where()
	allowedSortBy := map[string]bool{"created_at": true, "total_amount": true, "invoice_number": true, "payment_method": true, "status": true}
	allowedSortDir := map[string]bool{"ASC": true, "DESC": true}
	if sortBy != "" && allowedSortBy[sortBy] {
		query += fmt.Sprintf(" ORDER BY %s", sortBy)
		if sortDir != "" && allowedSortDir[sortDir] {
			query += " " + sortDir
		}
	} else {
		query += " ORDER BY s.created_at DESC"
	}
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", qb.ArgIdx, qb.ArgIdx+1)
	qb.Args = append(qb.Args, limit, offset)

	rows, err := r.db.Query(ctx, query, qb.Args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	// Collect sale IDs for batch loading sale items (avoid N+1 query)
	// Note: PostgreSQL has a limit of ~32767 parameters, so we batch in groups of 1000
	var saleIDs []int
	for rows.Next() {
		var s Sale
		var storeIDVal sql.NullInt64
		var createdAt, updatedAt time.Time
		err = rows.Scan(&s.ID, &s.InvoiceNumber, &s.CashierID, &storeIDVal, &s.Subtotal, &s.Discount, &s.Tax,
			&s.TotalAmount, &s.PaymentMethod, &s.Status, &createdAt, &updatedAt, &s.CustomerName)
		if err != nil {
			continue
		}
		if storeIDVal.Valid {
			v := int(storeIDVal.Int64)
			s.StoreID = &v
		}
		s.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		s.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)

		sales = append(sales, s)
		saleIDs = append(saleIDs, s.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// Batch load all sale items in chunks to avoid PostgreSQL parameter limit
	if len(saleIDs) > 0 {
		// Build a map for O(1) sale lookup instead of O(n²) linear scan
		saleMap := make(map[int]*Sale, len(sales))
		for i := range sales {
			saleMap[sales[i].ID] = &sales[i]
		}

		// Process in chunks of 1000 to stay under PostgreSQL's parameter limit
		for i := 0; i < len(saleIDs); i += 1000 {
			end := i + 1000
			if end > len(saleIDs) {
				end = len(saleIDs)
			}
			chunk := saleIDs[i:end]

			placeholders := make([]string, len(chunk))
			args3 := make([]interface{}, len(chunk))
			for j, id := range chunk {
				placeholders[j] = fmt.Sprintf("$%d", j+1)
				args3[j] = id
			}
			itemQuery := fmt.Sprintf(`
				SELECT si.id, si.sale_id, si.product_id, p.name, si.quantity, si.unit_price, si.subtotal
				FROM sale_items si
				JOIN products p ON si.product_id = p.id
				WHERE si.sale_id IN (%s)
			`, strings.Join(placeholders, ","))

			itemRows, err := r.db.Query(ctx, itemQuery, args3...)
			if err == nil {
				for itemRows.Next() {
					var item SaleItem
					err = itemRows.Scan(&item.ID, &item.SaleID, &item.ProductID, &item.Name, &item.Quantity, &item.UnitPrice, &item.Subtotal)
					if err == nil {
						if s, ok := saleMap[item.SaleID]; ok {
							s.Items = append(s.Items, item)
						}
					}
				}
				itemRows.Close()
			}
		}
	}

	return sales, total, nil
}

func (r *Repository) GetSalesForExport(ctx context.Context, search, startDate, endDate string, paymentMethods string, minTotal, maxTotal *int, storeID *int) ([]SaleExportRow, error) {
	qb := r.buildSaleFilter(search, startDate, endDate, storeID, paymentMethods, minTotal, maxTotal, nil)
	query := `SELECT s.invoice_number, s.created_at, COALESCE(c.name, '') as customer_name,
		COALESCE(si_counts.cnt, 0) as items_count,
		COALESCE(sp_codes.payment_codes, s.payment_method) as payment_method, s.total_amount
		FROM sales s
		LEFT JOIN customers c ON s.customer_id = c.id
		LEFT JOIN (SELECT sale_id, COUNT(*) AS cnt FROM sale_items GROUP BY sale_id) si_counts ON si_counts.sale_id = s.id
		LEFT JOIN (
			SELECT sale_id, STRING_AGG(payment_method_code, ',' ORDER BY id) AS payment_codes
			FROM sale_payments
			GROUP BY sale_id
		) sp_codes ON sp_codes.sale_id = s.id
		WHERE ` + qb.Where() + " ORDER BY s.created_at DESC"

	rows, err := r.db.Query(ctx, query, qb.Args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []SaleExportRow
	for rows.Next() {
		var row SaleExportRow
		var createdAt time.Time
		if err := rows.Scan(&row.InvoiceNumber, &createdAt, &row.CustomerName, &row.ItemCount, &row.PaymentMethod, &row.TotalAmount); err != nil {
			return nil, fmt.Errorf("scan sale export row: %w", err)
		}
		row.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *Repository) StreamSalesExportCSV(ctx context.Context, w io.Writer, search, startDate, endDate, paymentMethods string, minTotal, maxTotal *int, storeID *int) error {
	qb := r.buildSaleFilter(search, startDate, endDate, storeID, paymentMethods, minTotal, maxTotal, nil)
	query := `SELECT s.invoice_number, s.created_at, COALESCE(c.name, '') as customer_name,
		COALESCE(si_counts.cnt, 0) as items_count,
		COALESCE(sp_codes.payment_codes, s.payment_method) as payment_method, s.total_amount
		FROM sales s
		LEFT JOIN customers c ON s.customer_id = c.id
		LEFT JOIN (SELECT sale_id, COUNT(*) AS cnt FROM sale_items GROUP BY sale_id) si_counts ON si_counts.sale_id = s.id
		LEFT JOIN (
			SELECT sale_id, STRING_AGG(payment_method_code, ',' ORDER BY id) AS payment_codes
			FROM sale_payments
			GROUP BY sale_id
		) sp_codes ON sp_codes.sale_id = s.id
		WHERE ` + qb.Where() + " ORDER BY s.created_at DESC"

	rows, err := r.db.Query(ctx, query, qb.Args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	cw := csv.NewWriter(w)
	_ = shared.WriteCSVRow(cw, []string{"Invoice Number", "Date", "Customer", "Items", "Payment Method", "Total Amount"})

	for rows.Next() {
		var invoiceNumber, customerName, paymentMethod string
		var createdAt time.Time
		var itemCount int
		var totalAmount int64
		if err := rows.Scan(&invoiceNumber, &createdAt, &customerName, &itemCount, &paymentMethod, &totalAmount); err != nil {
			return fmt.Errorf("scan sale export row: %w", err)
		}
		_ = shared.WriteCSVRow(cw, []string{
			invoiceNumber,
			createdAt.In(shared.JakartaLocation()).Format(time.RFC3339),
			customerName,
			strconv.Itoa(itemCount),
			paymentMethod,
			fmt.Sprintf("%d", totalAmount),
		})
		cw.Flush()
	}
	if err := rows.Err(); err != nil {
		return err
	}
	cw.Flush()
	return cw.Error()
}

func (r *Repository) GetNextInvoiceNumber(ctx context.Context) (string, error) {
	now := time.Now().In(shared.JakartaLocation())
	year := now.Year()

	var seq int
	err := r.db.QueryRow(ctx, `SELECT nextval('invoice_seq')`).Scan(&seq)
	if err != nil {
		return "", fmt.Errorf("failed to get next invoice sequence: %w", err)
	}

	return fmt.Sprintf("INV-%d-%06d", year, seq), nil
}

// ==================== PAYMENT METHODS ====================

func (r *Repository) GetAllActive(ctx context.Context) ([]PaymentMethod, error) {
	var methods []PaymentMethod
	rows, err := r.db.Query(ctx, `
		SELECT id, code, name, is_active, requires_reference, sort_order, created_at
		FROM payment_methods
		WHERE is_active = true
		ORDER BY sort_order ASC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var m PaymentMethod
		var createdAt time.Time
		err := rows.Scan(&m.ID, &m.Code, &m.Name, &m.IsActive, &m.RequiresReference, &m.SortOrder, &createdAt)
		if err != nil {
			return nil, err
		}
		m.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		methods = append(methods, m)
	}
	return methods, nil
}

func (r *Repository) GetPaymentMethodByCode(ctx context.Context, code string) (*PaymentMethod, error) {
	var m PaymentMethod
	var createdAt time.Time
	err := r.db.QueryRow(ctx, `
		SELECT id, code, name, is_active, requires_reference, sort_order, created_at
		FROM payment_methods
		WHERE UPPER(code) = UPPER($1)
	`, code).Scan(&m.ID, &m.Code, &m.Name, &m.IsActive, &m.RequiresReference, &m.SortOrder, &createdAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("payment method not found")
		}
		return nil, err
	}
	m.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &m, nil
}

// ==================== PARKED SALES ====================

func (r *Repository) GetParkedSales(ctx context.Context, cashierID int) ([]Sale, error) {
	rows, err := r.db.Query(ctx, `
		SELECT s.id, s.invoice_number, s.cashier_id, s.store_id, s.subtotal, s.discount, s.tax, s.total_amount, s.payment_method, s.status, s.created_at, s.updated_at
		FROM sales s
		WHERE s.cashier_id = $1 AND s.status IN ('parked', 'recalled')
		ORDER BY s.created_at DESC
	`, cashierID)
	if err != nil {
		return nil, fmt.Errorf("query parked sales: %w", err)
	}
	defer rows.Close()

	var sales []Sale
	var saleIDs []int
	for rows.Next() {
		var s Sale
		var storeIDVal sql.NullInt64
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&s.ID, &s.InvoiceNumber, &s.CashierID, &storeIDVal, &s.Subtotal, &s.Discount, &s.Tax,
			&s.TotalAmount, &s.PaymentMethod, &s.Status, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan parked sale: %w", err)
		}
		if storeIDVal.Valid {
			v := int(storeIDVal.Int64)
			s.StoreID = &v
		}
		s.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		s.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		sales = append(sales, s)
		saleIDs = append(saleIDs, s.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(saleIDs) > 0 {
		saleMap := make(map[int]*Sale, len(sales))
		for i := range sales {
			saleMap[sales[i].ID] = &sales[i]
		}
		placeholders := make([]string, len(saleIDs))
		args := make([]interface{}, len(saleIDs))
		for j, id := range saleIDs {
			placeholders[j] = fmt.Sprintf("$%d", j+1)
			args[j] = id
		}
		itemQuery := fmt.Sprintf(`
			SELECT si.id, si.sale_id, si.product_id, p.name, si.quantity, si.unit_price, si.subtotal
			FROM sale_items si
			JOIN products p ON si.product_id = p.id
			WHERE si.sale_id IN (%s)
		`, strings.Join(placeholders, ","))
		itemRows, err := r.db.Query(ctx, itemQuery, args...)
		if err != nil {
			slog.Warn("failed to load items for parked sales", "error", err)
		} else {
			for itemRows.Next() {
				var item SaleItem
				if scanErr := itemRows.Scan(&item.ID, &item.SaleID, &item.ProductID, &item.Name, &item.Quantity, &item.UnitPrice, &item.Subtotal); scanErr != nil {
					slog.Warn("failed to scan item row for parked sale", "sale_id", item.SaleID, "error", scanErr)
					continue
				}
				if s, ok := saleMap[item.SaleID]; ok {
					s.Items = append(s.Items, item)
				}
			}
			itemRows.Close()
		}
	}

	return sales, nil
}

func (r *Repository) GetParkedSaleByID(ctx context.Context, id int, cashierID int) (*Sale, error) {
	var sale Sale
	var storeIDVal sql.NullInt64
	var createdAt, updatedAt time.Time

	err := r.db.QueryRow(ctx, `
		SELECT s.id, s.invoice_number, s.cashier_id, s.store_id, s.subtotal, s.discount, s.tax, s.total_amount, s.payment_method, s.status, s.created_at, s.updated_at
		FROM sales s
		WHERE s.id = $1 AND s.cashier_id = $2 AND s.status IN ('parked', 'recalled')
	`, id, cashierID).Scan(&sale.ID, &sale.InvoiceNumber, &sale.CashierID, &storeIDVal, &sale.Subtotal, &sale.Discount, &sale.Tax,
		&sale.TotalAmount, &sale.PaymentMethod, &sale.Status, &createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrSaleNotFound
		}
		return nil, err
	}
	if storeIDVal.Valid {
		v := int(storeIDVal.Int64)
		sale.StoreID = &v
	}
	sale.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	sale.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)

	itemRows, err := r.db.Query(ctx, `
		SELECT si.id, si.sale_id, si.product_id, p.name, si.quantity, si.unit_price, si.subtotal
		FROM sale_items si
		JOIN products p ON si.product_id = p.id
		WHERE si.sale_id = $1
	`, sale.ID)
	if err != nil {
		slog.Warn("failed to load items for parked sale by id", "sale_id", sale.ID, "error", err)
	} else {
		for itemRows.Next() {
			var item SaleItem
			if scanErr := itemRows.Scan(&item.ID, &item.SaleID, &item.ProductID, &item.Name, &item.Quantity, &item.UnitPrice, &item.Subtotal); scanErr != nil {
				slog.Warn("failed to scan item row for parked sale", "sale_id", sale.ID, "error", scanErr)
				continue
			}
			sale.Items = append(sale.Items, item)
		}
		itemRows.Close()
	}

	return &sale, nil
}

func (r *Repository) RecallSale(ctx context.Context, saleID int) (*Sale, error) {
	var sale Sale
	var storeIDVal sql.NullInt64
	var createdAt, updatedAt time.Time

	err := r.db.QueryRow(ctx, `
		UPDATE sales SET status = 'recalled', updated_at = NOW()
		WHERE id = $1 AND status IN ('parked', 'recalled')
		RETURNING id, invoice_number, cashier_id, store_id, subtotal, discount, tax, total_amount, payment_method, status, created_at, updated_at
	`, saleID).Scan(&sale.ID, &sale.InvoiceNumber, &sale.CashierID, &storeIDVal, &sale.Subtotal, &sale.Discount, &sale.Tax,
		&sale.TotalAmount, &sale.PaymentMethod, &sale.Status, &createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrSaleNotFound
		}
		return nil, fmt.Errorf("recall sale: %w", err)
	}
	if storeIDVal.Valid {
		v := int(storeIDVal.Int64)
		sale.StoreID = &v
	}
	sale.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	sale.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)

	itemRows, err := r.db.Query(ctx, `
		SELECT si.id, si.sale_id, si.product_id, p.name, si.quantity, si.unit_price, si.subtotal
		FROM sale_items si
		JOIN products p ON si.product_id = p.id
		WHERE si.sale_id = $1
	`, sale.ID)
	if err != nil {
		slog.Warn("failed to load items for recalled sale", "sale_id", sale.ID, "error", err)
	} else {
		for itemRows.Next() {
			var item SaleItem
			if scanErr := itemRows.Scan(&item.ID, &item.SaleID, &item.ProductID, &item.Name, &item.Quantity, &item.UnitPrice, &item.Subtotal); scanErr != nil {
				slog.Warn("failed to scan item row for recalled sale", "sale_id", sale.ID, "error", scanErr)
				continue
			}
			sale.Items = append(sale.Items, item)
		}
		itemRows.Close()
	}

	return &sale, nil
}

func (r *Repository) CancelParkedSale(ctx context.Context, saleID int) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE sales SET status = 'cancelled', updated_at = NOW()
		WHERE id = $1 AND status IN ('parked', 'recalled')
	`, saleID)
	if err != nil {
		return fmt.Errorf("cancel parked sale: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSaleNotFound
	}
	return nil
}

func (r *Repository) ConsumeParkedSale(ctx context.Context, tx pgx.Tx, parkedSaleID int) error {
	tag, err := tx.Exec(ctx, `
		UPDATE sales SET status = 'cancelled', updated_at = NOW()
		WHERE id = $1 AND status = 'recalled'
	`, parkedSaleID)
	if err != nil {
		return fmt.Errorf("consume parked sale: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSaleNotFound
	}
	return nil
}

// ==================== PAYMENT METHODS ====================

func (r *Repository) GetPaymentMethodByID(ctx context.Context, id int) (*PaymentMethod, error) {
	var m PaymentMethod
	var createdAt time.Time
	err := r.db.QueryRow(ctx, `
		SELECT id, code, name, is_active, requires_reference, sort_order, created_at
		FROM payment_methods
		WHERE id = $1
	`, id).Scan(&m.ID, &m.Code, &m.Name, &m.IsActive, &m.RequiresReference, &m.SortOrder, &createdAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("payment method not found")
		}
		return nil, err
	}
	m.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &m, nil
}
