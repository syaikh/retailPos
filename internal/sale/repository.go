package sale

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"retail-pos-system/internal/shared"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.Begin(ctx)
}

func (r *Repository) CreateSale(ctx context.Context, tx pgx.Tx, sale *Sale, items []SaleItem) error {
	var createdAt, updatedAt time.Time
	err := tx.QueryRow(ctx, `
		INSERT INTO sales (invoice_number, cashier_id, store_id, customer_id, subtotal, discount, tax, total_amount, payment_method, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at
	`, sale.InvoiceNumber, sale.CashierID, sale.StoreID, sale.CustomerID, sale.Subtotal, sale.Discount, sale.Tax, sale.TotalAmount, sale.PaymentMethod, sale.Status).
		Scan(&sale.ID, &createdAt, &updatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert sale: %w", err)
	}
	sale.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	sale.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)

	for i := range items {
		_, err = tx.Exec(ctx, `
			INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal, dpp_amount, tax_amount)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, sale.ID, items[i].ProductID, items[i].Quantity, items[i].UnitPrice, items[i].Subtotal, items[i].DPPAmount, items[i].TaxAmount)
		if err != nil {
			return fmt.Errorf("failed to insert sale item for product %d: %w", items[i].ProductID, err)
		}
	}

	return nil
}

func (r *Repository) GetSaleByID(ctx context.Context, id int, storeID *int) (*Sale, error) {
	var sale Sale
	var storeIDFromDB sql.NullInt64
	var createdAt, updatedAt time.Time

	query := `
		SELECT s.id, s.invoice_number, s.cashier_id, s.customer_id, s.store_id, s.subtotal, s.discount, s.tax, s.total_amount, s.payment_method, s.status, s.created_at, s.updated_at, COALESCE(c.name, '') as customer_name
		FROM sales s
		LEFT JOIN customers c ON s.customer_id = c.id
		WHERE s.id = $1`
	args := []interface{}{id}
	if storeID != nil {
		query += fmt.Sprintf(" AND s.store_id = $%d", len(args)+1)
		args = append(args, *storeID)
	}

	err := r.db.QueryRow(ctx, query, args...).Scan(&sale.ID, &sale.InvoiceNumber, &sale.CashierID, &sale.CustomerID, &sale.StoreID, &sale.Subtotal, &sale.Discount, &sale.Tax,
		&sale.TotalAmount, &sale.PaymentMethod, &sale.Status, &createdAt, &updatedAt, &sale.CustomerName)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrSaleNotFound
		}
		return nil, err
	}
	if storeIDFromDB.Valid {
		v := int(storeIDFromDB.Int64)
		sale.StoreID = &v
	}
	sale.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	sale.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)

	// Load sale items
	itemRows, err := r.db.Query(ctx, `
			SELECT si.id, si.sale_id, si.product_id, p.name, si.quantity, si.unit_price, si.subtotal, si.dpp_amount, si.tax_amount
			FROM sale_items si
			JOIN products p ON si.product_id = p.id
			WHERE si.sale_id = $1
		`, sale.ID)
	if err != nil {
		slog.Warn("failed to load items for sale", "sale_id", sale.ID, "error", err)
	} else {
		for itemRows.Next() {
			var item SaleItem
			if scanErr := itemRows.Scan(&item.ID, &item.SaleID, &item.ProductID, &item.Name, &item.Quantity, &item.UnitPrice, &item.Subtotal, &item.DPPAmount, &item.TaxAmount); scanErr != nil {
				slog.Warn("failed to scan item row", "sale_id", sale.ID, "error", scanErr)
				continue
			}
			sale.Items = append(sale.Items, item)
		}
		itemRows.Close()
	}

	return &sale, nil
}

func (r *Repository) GetAllSales(ctx context.Context, limit, offset int, search string, sortBy, sortDir, startDate, endDate string, storeID *int, paymentMethods string, minTotal, maxTotal *int) ([]Sale, int, error) {
	var sales []Sale
	var total int

	// ---- COUNT QUERY ----
	// When searching by product name we need to check sale_items + products,
	// so use a sub-select to avoid a messy multi-join count.
	countQuery := `SELECT COUNT(*) FROM sales s WHERE 1=1`
	countArgs := []interface{}{}
	argIdx := 1

	if search != "" {
		countQuery += fmt.Sprintf(" AND (s.invoice_number ILIKE $%d OR s.id IN (SELECT DISTINCT si.sale_id FROM sale_items si JOIN products p ON si.product_id = p.id WHERE p.name ILIKE $%d) OR s.customer_id IN (SELECT c2.id FROM customers c2 WHERE c2.name ILIKE $%d))", argIdx, argIdx, argIdx)
		countArgs = append(countArgs, "%"+search+"%")
		argIdx++
	}
	if startDate != "" {
		// Use Asia/Jakarta timezone for date filtering
		if start, err := time.ParseInLocation("2006-01-02", startDate, shared.JakartaLocation()); err == nil {
			countQuery += fmt.Sprintf(" AND s.created_at >= $%d", argIdx)
			countArgs = append(countArgs, start)
			argIdx++
		}
	}
	if endDate != "" {
		// Use Asia/Jakarta timezone for date filtering
		if end, err := time.ParseInLocation("2006-01-02", endDate, shared.JakartaLocation()); err == nil {
			countQuery += fmt.Sprintf(" AND s.created_at < $%d", argIdx)
			countArgs = append(countArgs, end.Add(24*time.Hour))
			argIdx++
		}
	}
	if storeID != nil {
		countQuery += fmt.Sprintf(" AND s.store_id = $%d", argIdx)
		countArgs = append(countArgs, *storeID)
		argIdx++
	}
	if paymentMethods != "" {
		countQuery += fmt.Sprintf(" AND s.payment_method = ANY(string_to_array($%d, ','))", argIdx)
		countArgs = append(countArgs, paymentMethods)
		argIdx++
	}
	if minTotal != nil {
		countQuery += fmt.Sprintf(" AND s.total_amount >= $%d", argIdx)
		countArgs = append(countArgs, *minTotal)
		argIdx++
	}
	if maxTotal != nil {
		countQuery += fmt.Sprintf(" AND s.total_amount <= $%d", argIdx)
		countArgs = append(countArgs, *maxTotal)
	}

	err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// ---- DATA QUERY ----
	query := `SELECT s.id, s.invoice_number, s.cashier_id, s.store_id, s.subtotal, s.discount, s.tax, s.total_amount, s.payment_method, s.status, s.created_at, s.updated_at, COALESCE(c.name, '') as customer_name
		FROM sales s
		LEFT JOIN customers c ON s.customer_id = c.id
		WHERE 1=1`
	args2 := []interface{}{}
	argIdx2 := 1

	if search != "" {
		query += fmt.Sprintf(" AND (s.invoice_number ILIKE $%d OR s.id IN (SELECT DISTINCT si.sale_id FROM sale_items si JOIN products p ON si.product_id = p.id WHERE p.name ILIKE $%d) OR s.customer_id IN (SELECT c2.id FROM customers c2 WHERE c2.name ILIKE $%d))", argIdx2, argIdx2, argIdx2)
		args2 = append(args2, "%"+search+"%")
		argIdx2++
	}
	if startDate != "" {
		// Use Asia/Jakarta timezone for date filtering
		if start, err := time.ParseInLocation("2006-01-02", startDate, shared.JakartaLocation()); err == nil {
			query += fmt.Sprintf(" AND s.created_at >= $%d", argIdx2)
			args2 = append(args2, start)
			argIdx2++
		}
	}
	if endDate != "" {
		// Use Asia/Jakarta timezone for date filtering
		if end, err := time.ParseInLocation("2006-01-02", endDate, shared.JakartaLocation()); err == nil {
			query += fmt.Sprintf(" AND s.created_at < $%d", argIdx2)
			args2 = append(args2, end.Add(24*time.Hour))
			argIdx2++
		}
	}
	if storeID != nil {
		query += fmt.Sprintf(" AND s.store_id = $%d", argIdx2)
		args2 = append(args2, *storeID)
		argIdx2++
	}
	if paymentMethods != "" {
		query += fmt.Sprintf(" AND s.payment_method = ANY(string_to_array($%d, ','))", argIdx2)
		args2 = append(args2, paymentMethods)
		argIdx2++
	}
	if minTotal != nil {
		query += fmt.Sprintf(" AND s.total_amount >= $%d", argIdx2)
		args2 = append(args2, *minTotal)
		argIdx2++
	}
	if maxTotal != nil {
		query += fmt.Sprintf(" AND s.total_amount <= $%d", argIdx2)
		args2 = append(args2, *maxTotal)
		argIdx2++
	}
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
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx2, argIdx2+1)
	args2 = append(args2, limit, offset)

	rows, err := r.db.Query(ctx, query, args2...)
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
	query := `SELECT s.invoice_number, s.created_at, COALESCE(c.name, '') as customer_name,
		(SELECT COUNT(*) FROM sale_items si WHERE si.sale_id = s.id) as items_count,
		s.payment_method, s.total_amount
		FROM sales s
		LEFT JOIN customers c ON s.customer_id = c.id
		WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if search != "" {
		query += fmt.Sprintf(" AND (s.invoice_number ILIKE $%d OR s.id IN (SELECT DISTINCT si.sale_id FROM sale_items si JOIN products p ON si.product_id = p.id WHERE p.name ILIKE $%d) OR s.customer_id IN (SELECT c2.id FROM customers c2 WHERE c2.name ILIKE $%d))", argIdx, argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}
	if startDate != "" {
		if start, err := time.ParseInLocation("2006-01-02", startDate, shared.JakartaLocation()); err == nil {
			query += fmt.Sprintf(" AND s.created_at >= $%d", argIdx)
			args = append(args, start)
			argIdx++
		}
	}
	if endDate != "" {
		if end, err := time.ParseInLocation("2006-01-02", endDate, shared.JakartaLocation()); err == nil {
			query += fmt.Sprintf(" AND s.created_at < $%d", argIdx)
			args = append(args, end.Add(24*time.Hour))
			argIdx++
		}
	}
	if paymentMethods != "" {
		query += fmt.Sprintf(" AND s.payment_method = ANY(string_to_array($%d, ','))", argIdx)
		args = append(args, paymentMethods)
		argIdx++
	}
	if minTotal != nil {
		query += fmt.Sprintf(" AND s.total_amount >= $%d", argIdx)
		args = append(args, *minTotal)
		argIdx++
	}
	if maxTotal != nil {
		query += fmt.Sprintf(" AND s.total_amount <= $%d", argIdx)
		args = append(args, *maxTotal)
	}
	if storeID != nil {
		query += fmt.Sprintf(" AND s.store_id = $%d", argIdx)
		args = append(args, *storeID)
	}
	query += " ORDER BY s.created_at DESC"

	rows, err := r.db.Query(ctx, query, args...)
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
		WHERE code = $1
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
