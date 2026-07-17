package customer

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"retail-pos-system/internal/shared"

	"github.com/jackc/pgx/v5"
)

type ImportResult struct {
	Inserted int      `json:"inserted"`
	Updated  int      `json:"updated"`
	Errors   []string `json:"errors"`
}

func (r *ImportResult) AddError(row int, msg string) {
	r.Errors = append(r.Errors, fmt.Sprintf("row %d: %s", row, msg))
}

type Repository struct {
	db shared.DBPool
}

func NewRepository(db shared.DBPool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByPhone(ctx context.Context, phone string, storeID *int) (*Customer, error) {
	var c Customer
	var createdAt, updatedAt time.Time
	var storeIDVal int
	query := `
		SELECT c.id, c.name, c.phone, c.email, c.address, c.tax_id, c.customer_group_id, cg.name,
		       c.loyalty_points, c.total_spent, c.last_purchase_at, c.note, c.is_active, c.is_walk_in, c.store_id, c.created_at, c.updated_at
		FROM customers c
		LEFT JOIN customer_groups cg ON cg.id = c.customer_group_id
		WHERE c.phone = $1`
	args := []interface{}{phone}
	if storeID != nil {
		query += " AND (c.store_id = $2 OR c.is_walk_in = true)"
		args = append(args, *storeID)
	}
	err := scanCustomerRow(r.db.QueryRow(ctx, query, args...), &c, &createdAt, &updatedAt, &storeIDVal)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("customer not found")
		}
		return nil, err
	}
	if !c.IsWalkIn {
		c.StoreID = &storeIDVal
	}
	c.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	c.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &c, nil
}

func (r *Repository) GetCustomerByID(ctx context.Context, id int, storeID *int) (*Customer, error) {
	var c Customer
	var createdAt, updatedAt time.Time
	var storeIDVal int
	query := `
		SELECT c.id, c.name, c.phone, c.email, c.address, c.tax_id, c.customer_group_id, cg.name,
		       c.loyalty_points, c.total_spent, c.last_purchase_at, c.note, c.is_active, c.is_walk_in, c.store_id, c.created_at, c.updated_at
		FROM customers c
		LEFT JOIN customer_groups cg ON cg.id = c.customer_group_id
		WHERE c.id = $1`
	args := []interface{}{id}
	if storeID != nil {
		query += " AND (c.store_id = $2 OR c.is_walk_in = true)"
		args = append(args, *storeID)
	}
	err := scanCustomerRow(r.db.QueryRow(ctx, query, args...), &c, &createdAt, &updatedAt, &storeIDVal)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("customer not found")
		}
		return nil, err
	}
	if !c.IsWalkIn {
		c.StoreID = &storeIDVal
	}
	c.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	c.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &c, nil
}

func (r *Repository) GetAllCustomers(ctx context.Context, limit, offset int, search string, isActive *bool, storeID *int, customerGroupID *int) ([]Customer, int, error) {
	args := []interface{}{}
	argIdx := 1
	countQuery := `SELECT COUNT(*) FROM customers WHERE is_walk_in = false`
	if storeID != nil {
		countQuery += fmt.Sprintf(" AND store_id = $%d", argIdx)
		args = append(args, *storeID)
		argIdx++
	}
	if search != "" {
		countQuery += fmt.Sprintf(" AND (name ILIKE $%d OR phone ILIKE $%d OR email ILIKE $%d)", argIdx, argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}
	if isActive != nil {
		countQuery += fmt.Sprintf(" AND is_active = $%d", argIdx)
		args = append(args, *isActive)
		argIdx++
	}
	if customerGroupID != nil {
		countQuery += fmt.Sprintf(" AND customer_group_id = $%d", argIdx)
		args = append(args, *customerGroupID)
	}
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT c.id, c.name, c.phone, c.email, c.address, c.tax_id, c.customer_group_id, cg.name,
	                 c.loyalty_points, c.total_spent, c.last_purchase_at, c.note, c.is_active, c.is_walk_in, c.store_id, c.created_at, c.updated_at
	          FROM customers c
	          LEFT JOIN customer_groups cg ON cg.id = c.customer_group_id
	          WHERE c.is_walk_in = false`
	queryArgs := []interface{}{}
	argIdx2 := 1
	if storeID != nil {
		query += fmt.Sprintf(" AND c.store_id = $%d", argIdx2)
		queryArgs = append(queryArgs, *storeID)
		argIdx2++
	}
	if search != "" {
		query += fmt.Sprintf(" AND (c.name ILIKE $%d OR c.phone ILIKE $%d OR c.email ILIKE $%d)", argIdx2, argIdx2, argIdx2)
		queryArgs = append(queryArgs, "%"+search+"%")
		argIdx2++
	}
	if isActive != nil {
		query += fmt.Sprintf(" AND c.is_active = $%d", argIdx2)
		queryArgs = append(queryArgs, *isActive)
		argIdx2++
	}
	if customerGroupID != nil {
		query += fmt.Sprintf(" AND c.customer_group_id = $%d", argIdx2)
		queryArgs = append(queryArgs, *customerGroupID)
		argIdx2++
	}
	query += fmt.Sprintf(" ORDER BY c.id DESC LIMIT $%d OFFSET $%d", argIdx2, argIdx2+1)
	queryArgs = append(queryArgs, limit, offset)

	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var customers []Customer
	for rows.Next() {
		var c Customer
		var createdAt, updatedAt time.Time
		var storeIDVal int
		if err := scanCustomerRow(rows, &c, &createdAt, &updatedAt, &storeIDVal); err != nil {
			return nil, 0, err
		}
		c.StoreID = &storeIDVal
		c.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		c.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		customers = append(customers, c)
	}
	return customers, total, nil
}

func (r *Repository) CreateCustomer(ctx context.Context, customer *Customer) error {
	var createdAt, updatedAt time.Time
	storeIDVal := 1
	if customer.StoreID != nil {
		storeIDVal = *customer.StoreID
	}
	return r.db.QueryRow(ctx, `
		INSERT INTO customers (name, phone, email, address, tax_id, note, is_active, is_walk_in, store_id, customer_group_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at
	`, customer.Name, customer.Phone, customer.Email, customer.Address, customer.TaxID, customer.Note, customer.IsActive, customer.IsWalkIn, storeIDVal, customer.CustomerGroupID).
		Scan(&customer.ID, &createdAt, &updatedAt)
}

func (r *Repository) UpdateCustomer(ctx context.Context, customer *Customer, id int, storeID *int) error {
	query := `
		UPDATE customers
		SET name = $1, phone = $2, email = $3, address = $4, tax_id = $5, note = $6, is_active = $7, customer_group_id = $8, updated_at = NOW()
		WHERE id = $9`
	args := []interface{}{customer.Name, customer.Phone, customer.Email, customer.Address, customer.TaxID, customer.Note, customer.IsActive, customer.CustomerGroupID, id}
	if storeID != nil {
		query += fmt.Sprintf(" AND store_id = $%d", len(args)+1)
		args = append(args, *storeID)
	}
	_, err := r.db.Exec(ctx, query, args...)
	return err
}

func (r *Repository) DeleteCustomer(ctx context.Context, id int, storeID *int) error {
	query := `UPDATE customers SET is_active = false, updated_at = NOW() WHERE id = $1`
	args := []interface{}{id}
	if storeID != nil {
		query += fmt.Sprintf(" AND store_id = $%d", len(args)+1)
		args = append(args, *storeID)
	}
	_, err := r.db.Exec(ctx, query, args...)
	return err
}

func (r *Repository) BulkUpdateCustomersStatus(ctx context.Context, ids []int, isActive bool, storeID *int) error {
	query := `UPDATE customers SET is_active = $1, updated_at = NOW() WHERE id = ANY($2) AND is_walk_in = false`
	args := []interface{}{isActive, ids}
	if storeID != nil {
		query += fmt.Sprintf(" AND store_id = $%d", len(args)+1)
		args = append(args, *storeID)
	}
	_, err := r.db.Exec(ctx, query, args...)
	return err
}

func (r *Repository) BulkDeleteCustomers(ctx context.Context, ids []int, storeID *int) error {
	query := `UPDATE customers SET is_active = false, updated_at = NOW() WHERE id = ANY($1) AND is_walk_in = false`
	args := []interface{}{ids}
	if storeID != nil {
		query += fmt.Sprintf(" AND store_id = $%d", len(args)+1)
		args = append(args, *storeID)
	}
	_, err := r.db.Exec(ctx, query, args...)
	return err
}

func (r *Repository) GetAllCustomersForExport(ctx context.Context, storeID *int) ([]Customer, error) {
	query := `
		SELECT c.id, c.name, c.phone, c.email, c.address, c.tax_id, c.customer_group_id, cg.name,
		       c.loyalty_points, c.total_spent, c.last_purchase_at, c.note, c.is_active, c.is_walk_in, c.store_id, c.created_at, c.updated_at
		FROM customers c
		LEFT JOIN customer_groups cg ON cg.id = c.customer_group_id
		WHERE c.is_walk_in = false`
	args := []interface{}{}
	if storeID != nil {
		query += " AND store_id = $1"
		args = append(args, *storeID)
	}
	query += " ORDER BY name"
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var customers []Customer
	for rows.Next() {
		var c Customer
		var createdAt, updatedAt time.Time
		var storeIDVal int
		if err := scanCustomerRow(rows, &c, &createdAt, &updatedAt, &storeIDVal); err != nil {
			return nil, err
		}
		c.StoreID = &storeIDVal
		c.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		c.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		customers = append(customers, c)
	}
	return customers, nil
}

func (r *Repository) BulkUpsertCustomers(ctx context.Context, records []CustomerImportRow, storeID *int) ImportResult {
	result := ImportResult{Errors: []string{}}
	if len(records) == 0 {
		return result
	}

	validRecords := make([]CustomerImportRow, 0, len(records))
	for _, rec := range records {
		if rec.Name == "" {
			result.AddError(rec.Row, "Name is required")
			continue
		}
		validRecords = append(validRecords, rec)
	}
	if len(validRecords) == 0 {
		return result
	}

	phones := make([]string, len(validRecords))
	for i, rec := range validRecords {
		phones[i] = rec.Phone
	}

	existingMap := make(map[string]int, len(validRecords))
	lookupQuery := "SELECT id, phone FROM customers WHERE phone = ANY($1) AND is_walk_in = false"
	lookupArgs := []interface{}{phones}
	if storeID != nil {
		lookupQuery += " AND store_id = $2"
		lookupArgs = append(lookupArgs, *storeID)
	}
	rows, err := r.db.Query(ctx, lookupQuery, lookupArgs...)
	if err == nil {
		for rows.Next() {
			var id int
			var phone string
			if err := rows.Scan(&id, &phone); err == nil {
				existingMap[phone] = id
			}
		}
		if err := rows.Err(); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("lookup rows iteration failed: %v", err))
		}
		rows.Close()
	}

	var updateIDs []int
	var updateRecords []CustomerImportRow
	var insertRecords []CustomerImportRow

	for _, rec := range validRecords {
		if id, ok := existingMap[rec.Phone]; ok {
			updateIDs = append(updateIDs, id)
			updateRecords = append(updateRecords, rec)
		} else {
			insertRecords = append(insertRecords, rec)
		}
	}

	if len(updateRecords) > 0 {
		valueStrings := make([]string, 0, len(updateRecords))
		valueArgs := make([]interface{}, 0, len(updateRecords)*7)
		for i, rec := range updateRecords {
			offset := len(valueArgs)
			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d::boolean, $%d::int)", offset+1, offset+2, offset+3, offset+4, offset+5, offset+6, offset+7))
			valueArgs = append(valueArgs, rec.Name, rec.Phone, rec.Email, rec.Address, rec.Note, rec.IsActive, updateIDs[i])
		}

		query := fmt.Sprintf(`
		UPDATE customers SET
			name = data.name,
			phone = data.phone,
			email = NULLIF(data.email, ''),
			address = NULLIF(data.address, ''),
			note = NULLIF(data.note, ''),
			is_active = data.is_active::boolean,
			updated_at = NOW()
		FROM (VALUES %s) AS data(name, phone, email, address, note, is_active, id)
		WHERE customers.id = data.id::int
		`, strings.Join(valueStrings, ", "))

		_, err := r.db.Exec(ctx, query, valueArgs...)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("batch update failed: %v", err))
		} else {
			result.Updated = len(updateRecords)
		}
	}

	if len(insertRecords) > 0 {
		storeIDVal := 1
		if storeID != nil {
			storeIDVal = *storeID
		}
		valueStrings := make([]string, 0, len(insertRecords))
		valueArgs := make([]interface{}, 0, len(insertRecords)*8)
		for _, rec := range insertRecords {
			offset := len(valueArgs)
			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, NULLIF($%d, ''), NULLIF($%d, ''), NULLIF($%d, ''), $%d, $%d, false, $%d)", offset+1, offset+2, offset+3, offset+4, offset+5, offset+6, offset+7, offset+8))
			valueArgs = append(valueArgs, rec.Name, rec.Phone, rec.Email, rec.Address, rec.Note, rec.IsActive, rec.TaxID, storeIDVal)
		}

		query := fmt.Sprintf(`
			INSERT INTO customers (name, phone, email, address, note, is_active, tax_id, is_walk_in, store_id)
			VALUES %s
		`, strings.Join(valueStrings, ", "))

		_, err := r.db.Exec(ctx, query, valueArgs...)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("batch insert failed: %v", err))
		} else {
			result.Inserted = len(insertRecords)
		}
	}

	return result
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

type scannable interface {
	Scan(dest ...any) error
}

func scanCustomerRow(src scannable, c *Customer, createdAt, updatedAt *time.Time, storeIDVal *int) error {
	var phone, email, address, taxID, lastPurchaseAt, note sql.NullString
	var customerGroupID sql.NullInt64
	var customerGroupName sql.NullString
	err := src.Scan(
		&c.ID, &c.Name, &phone, &email, &address, &taxID,
		&customerGroupID, &customerGroupName,
		&c.LoyaltyPoints, &c.TotalSpent, &lastPurchaseAt, &note,
		&c.IsActive, &c.IsWalkIn, storeIDVal, createdAt, updatedAt,
	)
	if err != nil {
		return err
	}
	c.Phone = strPtr(phone.String)
	c.Email = strPtr(email.String)
	c.Address = strPtr(address.String)
	c.TaxID = strPtr(taxID.String)
	c.LastPurchaseAt = strPtr(lastPurchaseAt.String)
	c.Note = strPtr(note.String)
	if customerGroupID.Valid {
		id := int(customerGroupID.Int64)
		c.CustomerGroupID = &id
	}
	if customerGroupName.Valid {
		c.CustomerGroupName = &customerGroupName.String
	}
	return nil
}
