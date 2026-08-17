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
	db                   shared.DBPool
	productNameProvider  ProductNameProvider
	customerNameProvider CustomerNameProvider
}

func NewRepository(db shared.DBPool) *Repository {
	return &Repository{db: db}
}

// SetProductNameProvider wires the product-owned name lookup port (see
// ProductNameProvider). Must be called before any read that needs a product
// name or search — an unwired repository fails fast at runtime.
func (r *Repository) SetProductNameProvider(p ProductNameProvider) {
	r.productNameProvider = p
}

// SetCustomerNameProvider wires the customer-owned name lookup port (see
// CustomerNameProvider). Must be called before any read that needs a customer
// name or search — an unwired repository fails fast at runtime.
func (r *Repository) SetCustomerNameProvider(p CustomerNameProvider) {
	r.customerNameProvider = p
}

func (r *Repository) productNamesByIDs(ctx context.Context, ids []int) (map[int]string, error) {
	if r.productNameProvider == nil {
		return nil, fmt.Errorf("sale repository: product name provider not wired; call SetProductNameProvider")
	}
	return r.productNameProvider.ProductNamesByIDs(ctx, r.db, ids)
}

func (r *Repository) customerNamesByIDs(ctx context.Context, ids []int) (map[int]string, error) {
	if r.customerNameProvider == nil {
		return nil, fmt.Errorf("sale repository: customer name provider not wired; call SetCustomerNameProvider")
	}
	return r.customerNameProvider.CustomerNamesByIDs(ctx, r.db, ids)
}

func (r *Repository) productIDsByName(ctx context.Context, search string) ([]int, error) {
	if r.productNameProvider == nil {
		return nil, fmt.Errorf("sale repository: product name provider not wired; call SetProductNameProvider")
	}
	return r.productNameProvider.ProductIDsByName(ctx, r.db, search)
}

func (r *Repository) customerIDsByName(ctx context.Context, search string) ([]int, error) {
	if r.customerNameProvider == nil {
		return nil, fmt.Errorf("sale repository: customer name provider not wired; call SetCustomerNameProvider")
	}
	return r.customerNameProvider.CustomerIDsByName(ctx, r.db, search)
}

// resolveSearchIDs resolves a free-text search into matching product and
// customer IDs so the filter can match by ANY($n) instead of cross-module
// JOINs against products/customers.
func (r *Repository) resolveSearchIDs(ctx context.Context, search string) (productIDs, customerIDs []int, err error) {
	if search == "" {
		return nil, nil, nil
	}
	pattern := "%" + search + "%"
	productIDs, err = r.productIDsByName(ctx, pattern)
	if err != nil {
		return nil, nil, err
	}
	customerIDs, err = r.customerIDsByName(ctx, pattern)
	if err != nil {
		return nil, nil, err
	}
	return productIDs, customerIDs, nil
}

// uniqueInts returns the deduplicated IDs from ids, preserving first-seen order.
func uniqueInts(ids []int) []int {
	seen := make(map[int]bool, len(ids))
	out := make([]int, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// productIDsOf returns the deduplicated product IDs referenced by items.
func productIDsOf(items []Item) []int {
	ids := make([]int, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ProductID)
	}
	return uniqueInts(ids)
}

func (r *Repository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.Begin(ctx)
}

func (r *Repository) CreateSale(ctx context.Context, tx pgx.Tx, sale *Sale, items []Item) error {
	var createdAt, updatedAt time.Time
	err := tx.QueryRow(ctx, `
		INSERT INTO sales (invoice_number, cashier_id, store_id, customer_id, shift_id, subtotal, discount, tax, total_amount, payment_method, status, hold_note)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at
	`, sale.InvoiceNumber, sale.CashierID, sale.StoreID, sale.CustomerID, sale.ShiftID, sale.Subtotal, sale.Discount, sale.Tax, sale.TotalAmount, sale.PaymentMethod, sale.Status, sale.HoldNote).
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
				item.PricingRuleID, item.PricingRuleName, item.PricingRuleType, item.Type,
				origPrice,
				item.Cost, item.TaxClassID, item.TaxRate,
				time.Now().In(shared.JakartaLocation()), productName,
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

func (r *Repository) CreateSalePayments(ctx context.Context, tx pgx.Tx, saleID int, payments []Payment) error {
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

func (r *Repository) GetSaleByID(ctx context.Context, id int, storeID *int) (*Sale, error) {
	var sale Sale
	var itemsJSON []byte
	var paymentsJSON []byte
	var createdAt, updatedAt time.Time

	query := `
		SELECT s.id, s.invoice_number, s.cashier_id, s.customer_id, s.store_id,
		       s.subtotal, s.discount, s.tax, s.total_amount, s.payment_method, s.status,
		       s.created_at, s.updated_at,
		       COALESCE(
		           (SELECT jsonb_agg(jsonb_build_object(
		               'id', si.id, 'sale_id', si.sale_id, 'product_id', si.product_id,
		               'name', si.product_name, 'quantity', si.quantity, 'unit_price', si.unit_price,
		               'subtotal', si.subtotal, 'dpp_amount', si.dpp_amount, 'tax_amount', si.tax_amount,
		               'pricing_rule_id', si.pricing_rule_id, 'pricing_rule_name', si.pricing_rule_name,
		               'pricing_rule_type', si.pricing_rule_type, 'pricing_type', si.pricing_type,
		               'original_price', si.original_price,
		               'cost', si.cost, 'tax_class_id', si.tax_class_id, 'tax_rate', si.tax_rate,
		               'snapshot_created_at', si.snapshot_created_at, 'product_name', si.product_name
		           )) FROM sale_items si
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
		&createdAt, &updatedAt, &itemsJSON, &paymentsJSON,
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

	if sale.CustomerID != nil {
		names, err := r.customerNamesByIDs(ctx, []int{*sale.CustomerID})
		if err != nil {
			return nil, fmt.Errorf("resolve customer name for sale %d: %w", sale.ID, err)
		}
		sale.CustomerName = names[*sale.CustomerID]
	}
	if len(sale.Items) > 0 {
		names, err := r.productNamesByIDs(ctx, uniqueInts(productIDsOf(sale.Items)))
		if err != nil {
			return nil, fmt.Errorf("resolve product names for sale %d: %w", sale.ID, err)
		}
		for i := range sale.Items {
			if sale.Items[i].ProductName == "" {
				sale.Items[i].Name = names[sale.Items[i].ProductID]
			}
		}
	}

	return &sale, nil
}

func (r *Repository) buildSaleFilter(productIDs, customerIDs []int, search, startDate, endDate string, storeID *int, paymentMethods string, minTotal, maxTotal, cashierID *int) *shared.QueryBuilder {
	qb := shared.NewQueryBuilder()
	if search != "" {
		qb.AddClause(" AND (s.invoice_number ILIKE $%[1]d OR s.id IN (SELECT DISTINCT si.sale_id FROM sale_items si WHERE si.product_id = ANY($%[2]d)) OR s.customer_id = ANY($%[3]d))", "%"+search+"%", productIDs, customerIDs)
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
		qb.AddClause(`
			AND (
				s.payment_method = ANY(string_to_array($%d, ','))
				OR EXISTS (
					SELECT 1 FROM sale_payments sp
					WHERE sp.sale_id = s.id
					  AND sp.payment_method_code = ANY(string_to_array($%d, ','))
				)
			)`, paymentMethods, paymentMethods)
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

	productIDs, customerIDs, err := r.resolveSearchIDs(ctx, search)
	if err != nil {
		return nil, 0, err
	}
	qb := r.buildSaleFilter(productIDs, customerIDs, search, startDate, endDate, storeID, paymentMethods, minTotal, maxTotal, cashierID)
	countQuery := "SELECT COUNT(*) FROM sales s WHERE " + qb.Where()
	err = r.db.QueryRow(ctx, countQuery, qb.Args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `SELECT s.id, s.invoice_number, s.cashier_id, s.store_id, s.customer_id, COALESCE(c.name, ''), s.subtotal, s.discount, s.tax, s.total_amount, s.payment_method, s.status, s.created_at, s.updated_at
		FROM sales s
		LEFT JOIN customers c ON c.id = s.customer_id
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
		var customerIDVal sql.NullInt64
		var customerNameVal string
		var createdAt, updatedAt time.Time
		err = rows.Scan(&s.ID, &s.InvoiceNumber, &s.CashierID, &storeIDVal, &customerIDVal, &customerNameVal, &s.Subtotal, &s.Discount, &s.Tax,
			&s.TotalAmount, &s.PaymentMethod, &s.Status, &createdAt, &updatedAt)
		if err != nil {
			continue
		}
		if storeIDVal.Valid {
			v := int(storeIDVal.Int64)
			s.StoreID = &v
		}
		if customerIDVal.Valid {
			v := int(customerIDVal.Int64)
			s.CustomerID = &v
		}
		s.CustomerName = customerNameVal
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

		var itemProductIDs []int
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
				SELECT si.id, si.sale_id, si.product_id, si.quantity, si.unit_price, si.subtotal
				FROM sale_items si
				WHERE si.sale_id IN (%s)
			`, strings.Join(placeholders, ","))

			itemRows, err := r.db.Query(ctx, itemQuery, args3...)
			if err == nil {
				for itemRows.Next() {
					var item Item
					err = itemRows.Scan(&item.ID, &item.SaleID, &item.ProductID, &item.Quantity, &item.UnitPrice, &item.Subtotal)
					if err == nil {
						itemProductIDs = append(itemProductIDs, item.ProductID)
						if s, ok := saleMap[item.SaleID]; ok {
							s.Items = append(s.Items, item)
						}
					}
				}
				itemRows.Close()
			}
		}

		if len(itemProductIDs) > 0 {
			names, err := r.productNamesByIDs(ctx, uniqueInts(itemProductIDs))
			if err != nil {
				return nil, 0, err
			}
			for i := range sales {
				for j := range sales[i].Items {
					sales[i].Items[j].Name = names[sales[i].Items[j].ProductID]
				}
			}
		}
	}

	// Resolve customer names (customers table is owned by the referensi
	// context; internal/sale reads it via CustomerNameProvider).
	customerIDList := make([]int, 0, len(sales))
	seenCustomer := make(map[int]bool)
	for i := range sales {
		if sales[i].CustomerID != nil && !seenCustomer[*sales[i].CustomerID] {
			seenCustomer[*sales[i].CustomerID] = true
			customerIDList = append(customerIDList, *sales[i].CustomerID)
		}
	}
	if len(customerIDList) > 0 {
		names, err := r.customerNamesByIDs(ctx, customerIDList)
		if err != nil {
			return nil, 0, err
		}
		for i := range sales {
			if sales[i].CustomerID != nil {
				sales[i].CustomerName = names[*sales[i].CustomerID]
			}
		}
	}

	return sales, total, nil
}

func (r *Repository) GetSalesForExport(ctx context.Context, search, startDate, endDate string, paymentMethods string, minTotal, maxTotal *int, storeID *int) ([]ExportRow, error) {
	productIDs, customerIDs, err := r.resolveSearchIDs(ctx, search)
	if err != nil {
		return nil, err
	}
	qb := r.buildSaleFilter(productIDs, customerIDs, search, startDate, endDate, storeID, paymentMethods, minTotal, maxTotal, nil)
	query := `SELECT s.invoice_number, s.created_at, s.customer_id,
		COALESCE(si_counts.cnt, 0) as items_count,
		COALESCE(sp_codes.payment_codes, s.payment_method) as payment_method, s.total_amount
		FROM sales s
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

	var result []ExportRow
	rowCustomerIDs := make([]int, 0, 16)
	customerIDSet := make(map[int]bool)
	for rows.Next() {
		var row ExportRow
		var customerIDVal sql.NullInt64
		var createdAt time.Time
		if err := rows.Scan(&row.InvoiceNumber, &createdAt, &customerIDVal, &row.ItemCount, &row.PaymentMethod, &row.TotalAmount); err != nil {
			return nil, fmt.Errorf("scan sale export row: %w", err)
		}
		row.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		result = append(result, row)
		if customerIDVal.Valid {
			id := int(customerIDVal.Int64)
			rowCustomerIDs = append(rowCustomerIDs, id)
			customerIDSet[id] = true
		} else {
			rowCustomerIDs = append(rowCustomerIDs, 0)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	names := map[int]string{}
	if len(customerIDSet) > 0 {
		customerIDList := make([]int, 0, len(customerIDSet))
		for id := range customerIDSet {
			customerIDList = append(customerIDList, id)
		}
		var err error
		names, err = r.customerNamesByIDs(ctx, customerIDList)
		if err != nil {
			return nil, err
		}
	}
	for i := range result {
		if rowCustomerIDs[i] != 0 {
			result[i].CustomerName = names[rowCustomerIDs[i]]
		}
	}

	return result, nil
}

func (r *Repository) StreamSalesExportCSV(ctx context.Context, w io.Writer, search, startDate, endDate, paymentMethods string, minTotal, maxTotal *int, storeID *int) error {
	productIDs, customerIDs, err := r.resolveSearchIDs(ctx, search)
	if err != nil {
		return err
	}
	qb := r.buildSaleFilter(productIDs, customerIDs, search, startDate, endDate, storeID, paymentMethods, minTotal, maxTotal, nil)
	query := `SELECT s.invoice_number, s.created_at, s.customer_id,
		COALESCE(si_counts.cnt, 0) as items_count,
		COALESCE(sp_codes.payment_codes, s.payment_method) as payment_method, s.total_amount
		FROM sales s
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

	type csvRow struct {
		invoiceNumber, createdAt, paymentMethod string
		customerID                              int
		itemCount                               int
		totalAmount                             int64
	}
	var buffered []csvRow
	for rows.Next() {
		var invoiceNumber, paymentMethod string
		var customerIDVal sql.NullInt64
		var createdAt time.Time
		var itemCount int
		var totalAmount int64
		if err := rows.Scan(&invoiceNumber, &createdAt, &customerIDVal, &itemCount, &paymentMethod, &totalAmount); err != nil {
			return fmt.Errorf("scan sale export row: %w", err)
		}
		buffered = append(buffered, csvRow{
			invoiceNumber: invoiceNumber,
			createdAt:     createdAt.In(shared.JakartaLocation()).Format(time.RFC3339),
			customerID:    int(customerIDVal.Int64),
			itemCount:     itemCount,
			totalAmount:   totalAmount,
			paymentMethod: paymentMethod,
		})
	}
	if err := rows.Err(); err != nil {
		return err
	}

	names := map[int]string{}
	customerIDSet := make(map[int]bool)
	for _, row := range buffered {
		if row.customerID != 0 {
			customerIDSet[row.customerID] = true
		}
	}
	if len(customerIDSet) > 0 {
		customerIDList := make([]int, 0, len(customerIDSet))
		for id := range customerIDSet {
			customerIDList = append(customerIDList, id)
		}
		names, err = r.customerNamesByIDs(ctx, customerIDList)
		if err != nil {
			return err
		}
	}

	cw := csv.NewWriter(w)
	_ = shared.WriteCSVRow(cw, []string{"Invoice Number", "Date", "Customer", "Items", "Payment Method", "Total Amount"})

	for _, row := range buffered {
		customerName := names[row.customerID]
		_ = shared.WriteCSVRow(cw, []string{
			row.invoiceNumber,
			row.createdAt,
			customerName,
			strconv.Itoa(row.itemCount),
			row.paymentMethod,
			fmt.Sprintf("%d", row.totalAmount),
		})
		cw.Flush()
	}
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

// GetParkedSales lists parked/recalled sales. A nil ownerID returns every
// parked sale (manager/elevated view); a non-nil ownerID restricts the results
// to that cashier's own sales (P2-6 owner scoping).
func (r *Repository) GetParkedSales(ctx context.Context, ownerID, storeID *int) ([]Sale, error) {
	query := `
		SELECT s.id, s.invoice_number, s.cashier_id, s.store_id, s.subtotal, s.discount, s.tax, s.total_amount, s.payment_method, s.status, s.customer_id, s.hold_note, s.created_at, s.updated_at
		FROM sales s
		WHERE s.status IN ('parked', 'recalled')
	`
	args := []interface{}{}
	if ownerID != nil {
		args = append(args, *ownerID)
		query += fmt.Sprintf(` AND s.cashier_id = $%d`, len(args))
	}
	if storeID != nil {
		args = append(args, *storeID)
		query += fmt.Sprintf(` AND s.store_id = $%d`, len(args))
	}
	query += ` ORDER BY s.created_at DESC`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query parked sales: %w", err)
	}
	defer rows.Close()

	var sales []Sale
	var saleIDs []int
	for rows.Next() {
		var s Sale
		var storeIDVal, customerIDVal sql.NullInt64
		var holdNoteVal sql.NullString
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&s.ID, &s.InvoiceNumber, &s.CashierID, &storeIDVal, &s.Subtotal, &s.Discount, &s.Tax,
			&s.TotalAmount, &s.PaymentMethod, &s.Status, &customerIDVal, &holdNoteVal, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan parked sale: %w", err)
		}
		if storeIDVal.Valid {
			v := int(storeIDVal.Int64)
			s.StoreID = &v
		}
		if customerIDVal.Valid {
			v := int(customerIDVal.Int64)
			s.CustomerID = &v
		}
		if holdNoteVal.Valid {
			s.HoldNote = holdNoteVal.String
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
			SELECT si.id, si.sale_id, si.product_id, si.quantity, si.unit_price, si.subtotal
			FROM sale_items si
			WHERE si.sale_id IN (%s)
		`, strings.Join(placeholders, ","))
		itemRows, err := r.db.Query(ctx, itemQuery, args...)
		if err != nil {
			slog.Warn("failed to load items for parked sales", "error", err)
		} else {
			var itemProductIDs []int
			for itemRows.Next() {
				var item Item
				if scanErr := itemRows.Scan(&item.ID, &item.SaleID, &item.ProductID, &item.Quantity, &item.UnitPrice, &item.Subtotal); scanErr != nil {
					slog.Warn("failed to scan item row for parked sale", "sale_id", item.SaleID, "error", scanErr)
					continue
				}
				itemProductIDs = append(itemProductIDs, item.ProductID)
				if s, ok := saleMap[item.SaleID]; ok {
					s.Items = append(s.Items, item)
				}
			}
			itemRows.Close()

			if len(itemProductIDs) > 0 {
				names, err := r.productNamesByIDs(ctx, uniqueInts(itemProductIDs))
				if err != nil {
					return nil, err
				}
				for i := range sales {
					for j := range sales[i].Items {
						sales[i].Items[j].Name = names[sales[i].Items[j].ProductID]
					}
				}
			}
		}
	}

	return sales, nil
}

func (r *Repository) GetParkedSaleByID(ctx context.Context, id int, ownerID, storeID *int) (*Sale, error) {
	var sale Sale
	var storeIDVal, customerIDVal sql.NullInt64
	var holdNoteVal sql.NullString
	var createdAt, updatedAt time.Time

	query := `
		SELECT s.id, s.invoice_number, s.cashier_id, s.store_id, s.subtotal, s.discount, s.tax, s.total_amount, s.payment_method, s.status, s.customer_id, s.hold_note, s.created_at, s.updated_at
		FROM sales s
		WHERE s.id = $1 AND s.status IN ('parked', 'recalled')
	`
	args := []interface{}{id}
	if ownerID != nil {
		args = append(args, *ownerID)
		query += fmt.Sprintf(` AND s.cashier_id = $%d`, len(args))
	}
	if storeID != nil {
		args = append(args, *storeID)
		query += fmt.Sprintf(` AND s.store_id = $%d`, len(args))
	}
	err := r.db.QueryRow(ctx, query, args...).Scan(&sale.ID, &sale.InvoiceNumber, &sale.CashierID, &storeIDVal, &sale.Subtotal, &sale.Discount, &sale.Tax,
		&sale.TotalAmount, &sale.PaymentMethod, &sale.Status, &customerIDVal, &holdNoteVal, &createdAt, &updatedAt)
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
	if customerIDVal.Valid {
		v := int(customerIDVal.Int64)
		sale.CustomerID = &v
	}
	if holdNoteVal.Valid {
		sale.HoldNote = holdNoteVal.String
	}
	sale.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	sale.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)

	itemRows, err := r.db.Query(ctx, `
		SELECT si.id, si.sale_id, si.product_id, si.quantity, si.unit_price, si.subtotal
		FROM sale_items si
		WHERE si.sale_id = $1
	`, sale.ID)
	if err != nil {
		slog.Warn("failed to load items for parked sale by id", "sale_id", sale.ID, "error", err)
	} else {
		var itemProductIDs []int
		for itemRows.Next() {
			var item Item
			if scanErr := itemRows.Scan(&item.ID, &item.SaleID, &item.ProductID, &item.Quantity, &item.UnitPrice, &item.Subtotal); scanErr != nil {
				slog.Warn("failed to scan item row for parked sale", "sale_id", sale.ID, "error", scanErr)
				continue
			}
			itemProductIDs = append(itemProductIDs, item.ProductID)
			sale.Items = append(sale.Items, item)
		}
		itemRows.Close()

		if len(itemProductIDs) > 0 {
			names, err := r.productNamesByIDs(ctx, uniqueInts(itemProductIDs))
			if err != nil {
				return nil, err
			}
			for j := range sale.Items {
				sale.Items[j].Name = names[sale.Items[j].ProductID]
			}
		}
	}

	return &sale, nil
}

func (r *Repository) RecallSale(ctx context.Context, saleID int, ownerID, storeID *int) (*Sale, error) {
	var sale Sale
	var storeIDVal sql.NullInt64
	var createdAt, updatedAt time.Time

	query := `
		UPDATE sales SET status = 'recalled', updated_at = NOW()
		WHERE id = $1 AND status IN ('parked', 'recalled')
	`
	args := []interface{}{saleID}
	if ownerID != nil {
		args = append(args, *ownerID)
		query += fmt.Sprintf(` AND cashier_id = $%d`, len(args))
	}
	if storeID != nil {
		args = append(args, *storeID)
		query += fmt.Sprintf(` AND store_id = $%d`, len(args))
	}
	query += `
		RETURNING id, invoice_number, cashier_id, store_id, subtotal, discount, tax, total_amount, payment_method, status, created_at, updated_at
	`
	err := r.db.QueryRow(ctx, query, args...).Scan(&sale.ID, &sale.InvoiceNumber, &sale.CashierID, &storeIDVal, &sale.Subtotal, &sale.Discount, &sale.Tax,
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
		SELECT si.id, si.sale_id, si.product_id, si.quantity, si.unit_price, si.subtotal
		FROM sale_items si
		WHERE si.sale_id = $1
	`, sale.ID)
	if err != nil {
		slog.Warn("failed to load items for recalled sale", "sale_id", sale.ID, "error", err)
	} else {
		var itemProductIDs []int
		for itemRows.Next() {
			var item Item
			if scanErr := itemRows.Scan(&item.ID, &item.SaleID, &item.ProductID, &item.Quantity, &item.UnitPrice, &item.Subtotal); scanErr != nil {
				slog.Warn("failed to scan item row for recalled sale", "sale_id", sale.ID, "error", scanErr)
				continue
			}
			itemProductIDs = append(itemProductIDs, item.ProductID)
			sale.Items = append(sale.Items, item)
		}
		itemRows.Close()

		if len(itemProductIDs) > 0 {
			names, err := r.productNamesByIDs(ctx, uniqueInts(itemProductIDs))
			if err != nil {
				return nil, err
			}
			for j := range sale.Items {
				sale.Items[j].Name = names[sale.Items[j].ProductID]
			}
		}
	}

	return &sale, nil
}

func (r *Repository) CancelParkedSale(ctx context.Context, saleID int, ownerID, storeID *int) error {
	query := `
		UPDATE sales SET status = 'cancelled', updated_at = NOW()
		WHERE id = $1 AND status IN ('parked', 'recalled')
	`
	args := []interface{}{saleID}
	if ownerID != nil {
		args = append(args, *ownerID)
		query += fmt.Sprintf(` AND cashier_id = $%d`, len(args))
	}
	if storeID != nil {
		args = append(args, *storeID)
		query += fmt.Sprintf(` AND store_id = $%d`, len(args))
	}
	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("cancel parked sale: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSaleNotFound
	}
	return nil
}

func (r *Repository) ConsumeParkedSale(ctx context.Context, tx pgx.Tx, parkedSaleID int, ownerID, storeID *int) error {
	query := `
		UPDATE sales SET status = 'cancelled', updated_at = NOW()
		WHERE id = $1 AND status = 'recalled'
	`
	args := []interface{}{parkedSaleID}
	if ownerID != nil {
		args = append(args, *ownerID)
		query += fmt.Sprintf(` AND cashier_id = $%d`, len(args))
	}
	if storeID != nil {
		args = append(args, *storeID)
		query += fmt.Sprintf(` AND store_id = $%d`, len(args))
	}
	tag, err := tx.Exec(ctx, query, args...)
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
