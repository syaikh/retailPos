package sale

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var jakartaLoc *time.Location

func init() {
	var err error
	jakartaLoc, err = time.LoadLocation("Asia/Jakarta")
	if err != nil {
		log.Printf("Warning: failed to load Asia/Jakarta timezone: %v. Falling back to UTC.", err)
		jakartaLoc = time.UTC
	}
}

func mustLoadJakarta() *time.Location {
	if jakartaLoc == nil {
		var err error
		jakartaLoc, err = time.LoadLocation("Asia/Jakarta")
		if err != nil {
			log.Printf("Warning: failed to load Asia/Jakarta timezone: %v. Falling back to UTC.", err)
			jakartaLoc = time.UTC
		}
	}
	return jakartaLoc
}

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
	sale.CreatedAt = createdAt.Format(time.RFC3339)
	sale.UpdatedAt = updatedAt.Format(time.RFC3339)

	for i := range items {
		// 1. Insert sale item
		_, err = tx.Exec(ctx, `
			INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal, dpp_amount, tax_amount)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, sale.ID, items[i].ProductID, items[i].Quantity, items[i].UnitPrice, items[i].Subtotal, items[i].DPPAmount, items[i].TaxAmount)
		if err != nil {
			return fmt.Errorf("failed to insert sale item for product %d: %w", items[i].ProductID, err)
		}

		// 2. Update product stock in product_stock table; insert row if missing
		cmd, err := tx.Exec(ctx, `
			UPDATE product_stock
			SET quantity = quantity - $1, updated_at = NOW()
			WHERE product_id = $2 AND warehouse_id IS NULL AND store_id IS NULL
		`, items[i].Quantity, items[i].ProductID)
		if err != nil {
			return fmt.Errorf("failed to update stock for product %d: %w", items[i].ProductID, err)
		}
		if cmd.RowsAffected() == 0 {
			var currentQty int
			if err := tx.QueryRow(ctx, `
				SELECT COALESCE(quantity, 0) FROM product_stock
				WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL
			`, items[i].ProductID).Scan(&currentQty); err != nil {
				return fmt.Errorf("failed to query current stock for product %d: %w", items[i].ProductID, err)
			}
			newQty := currentQty - items[i].Quantity
			if newQty < 0 {
				newQty = 0
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO product_stock (product_id, quantity, updated_at)
				VALUES ($1, GREATEST(0, $2), NOW())
			`, items[i].ProductID, newQty)
			if err != nil {
				return fmt.Errorf("failed to insert stock row for product %d: %w", items[i].ProductID, err)
			}
		}

		// 3. Record inventory movement
		_, err = tx.Exec(ctx, `
			INSERT INTO inventory_movements (product_id, quantity_change, type, reference_id, reference_table, user_id, notes)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, items[i].ProductID, -items[i].Quantity, "sale", sale.ID, "sales", sale.CashierID, fmt.Sprintf("Sale %s", sale.InvoiceNumber))
		if err != nil {
			return fmt.Errorf("failed to record inventory movement for product %d: %w", items[i].ProductID, err)
		}
	}

	return nil
}

func (r *Repository) GetSaleByID(ctx context.Context, id int) (*Sale, error) {
	var sale Sale
	var storeID sql.NullInt64
	var createdAt, updatedAt time.Time

	err := r.db.QueryRow(ctx, `
		SELECT s.id, s.invoice_number, s.cashier_id, s.customer_id, s.store_id, s.subtotal, s.discount, s.tax, s.total_amount, s.payment_method, s.status, s.created_at, s.updated_at, COALESCE(c.name, '') as customer_name
		FROM sales s
		LEFT JOIN customers c ON s.customer_id = c.id
		WHERE s.id = $1
	`, id).Scan(&sale.ID, &sale.InvoiceNumber, &sale.CashierID, &sale.CustomerID, &sale.StoreID, &sale.Subtotal, &sale.Discount, &sale.Tax,
		&sale.TotalAmount, &sale.PaymentMethod, &sale.Status, &createdAt, &updatedAt, &sale.CustomerName)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("sale not found")
		}
		return nil, err
	}
	if storeID.Valid {
		v := int(storeID.Int64)
		sale.StoreID = &v
	}
	sale.CreatedAt = createdAt.Format(time.RFC3339)
	sale.UpdatedAt = updatedAt.Format(time.RFC3339)

	// Load sale items
	itemRows, err := r.db.Query(ctx, `
			SELECT si.id, si.sale_id, si.product_id, p.name, si.quantity, si.unit_price, si.subtotal, si.dpp_amount, si.tax_amount
			FROM sale_items si
			JOIN products p ON si.product_id = p.id
			WHERE si.sale_id = $1
		`, sale.ID)
	if err != nil {
		log.Printf("Warning: failed to load items for sale %d: %v", sale.ID, err)
	} else {
		for itemRows.Next() {
			var item SaleItem
			if scanErr := itemRows.Scan(&item.ID, &item.SaleID, &item.ProductID, &item.Name, &item.Quantity, &item.UnitPrice, &item.Subtotal, &item.DPPAmount, &item.TaxAmount); scanErr != nil {
				log.Printf("Warning: failed to scan item row: %v", scanErr)
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
		if start, err := time.ParseInLocation("2006-01-02", startDate, mustLoadJakarta()); err == nil {
			countQuery += fmt.Sprintf(" AND s.created_at >= $%d", argIdx)
			countArgs = append(countArgs, start)
			argIdx++
		}
	}
	if endDate != "" {
		// Use Asia/Jakarta timezone for date filtering
		if end, err := time.ParseInLocation("2006-01-02", endDate, mustLoadJakarta()); err == nil {
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
		if start, err := time.ParseInLocation("2006-01-02", startDate, mustLoadJakarta()); err == nil {
			query += fmt.Sprintf(" AND s.created_at >= $%d", argIdx2)
			args2 = append(args2, start)
			argIdx2++
		}
	}
	if endDate != "" {
		// Use Asia/Jakarta timezone for date filtering
		if end, err := time.ParseInLocation("2006-01-02", endDate, mustLoadJakarta()); err == nil {
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
		s.CreatedAt = createdAt.Format(time.RFC3339)
		s.UpdatedAt = updatedAt.Format(time.RFC3339)

		sales = append(sales, s)
		saleIDs = append(saleIDs, s.ID)
	}

	// Batch load all sale items in chunks to avoid PostgreSQL parameter limit
	if len(saleIDs) > 0 {
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
						// Find the sale and append the item
						for j := range sales {
							if sales[j].ID == item.SaleID {
								sales[j].Items = append(sales[j].Items, item)
								break
							}
						}
					}
				}
				itemRows.Close()
			}
		}
	}

	return sales, total, nil
}

func (r *Repository) GetSalesForExport(ctx context.Context, search, startDate, endDate string, paymentMethods string, minTotal, maxTotal *int) ([]SaleExportRow, error) {
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
		if start, err := time.ParseInLocation("2006-01-02", startDate, mustLoadJakarta()); err == nil {
			query += fmt.Sprintf(" AND s.created_at >= $%d", argIdx)
			args = append(args, start)
			argIdx++
		}
	}
	if endDate != "" {
		if end, err := time.ParseInLocation("2006-01-02", endDate, mustLoadJakarta()); err == nil {
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
			continue
		}
		row.CreatedAt = createdAt.Format(time.RFC3339)
		result = append(result, row)
	}
	return result, nil
}

func (r *Repository) GetNextInvoiceNumber(ctx context.Context) (string, error) {
	now := time.Now().In(mustLoadJakarta())
	year := now.Year()
	yearStr := fmt.Sprintf("%d", year)

	var maxSeq int
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(MAX(
			CAST(SUBSTRING(invoice_number FROM '\d+$') AS INTEGER)
		 ), 0)
		 FROM sales
		 WHERE invoice_number LIKE $1
	`, "INV-"+yearStr+"-%").Scan(&maxSeq)
	if err != nil {
		return "", fmt.Errorf("failed to get next invoice number: %w", err)
	}

	return fmt.Sprintf("INV-%d-%06d", year, maxSeq+1), nil
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
		m.CreatedAt = createdAt.Format(time.RFC3339)
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
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("payment method not found")
		}
		return nil, err
	}
	m.CreatedAt = createdAt.Format(time.RFC3339)
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
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("payment method not found")
		}
		return nil, err
	}
	m.CreatedAt = createdAt.Format(time.RFC3339)
	return &m, nil
}
