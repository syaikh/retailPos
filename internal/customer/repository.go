package customer

import (
	"context"
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

type ImportResult struct {
	Inserted int      `json:"inserted"`
	Updated  int      `json:"updated"`
	Errors   []string `json:"errors"`
}

func (r *ImportResult) AddError(row int, msg string) {
	r.Errors = append(r.Errors, fmt.Sprintf("row %d: %s", row, msg))
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByPhone(ctx context.Context, phone string) (*Customer, error) {
	var c Customer
	var createdAt, updatedAt time.Time
	err := r.db.QueryRow(ctx, `
		SELECT id, name, phone, email, address, tax_id, loyalty_points, total_spent, last_purchase_at, note, is_active, is_walk_in, created_at, updated_at
		FROM customers
		WHERE phone = $1
	`, phone).Scan(&c.ID, &c.Name, &c.Phone, &c.Email, &c.Address, &c.TaxID, &c.LoyaltyPoints, &c.TotalSpent, &c.LastPurchaseAt, &c.Note, &c.IsActive, &c.IsWalkIn, &createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("customer not found")
		}
		return nil, err
	}
	c.CreatedAt = createdAt.In(jakartaLoc).Format(time.RFC3339)
	c.UpdatedAt = updatedAt.In(jakartaLoc).Format(time.RFC3339)
	return &c, nil
}

func (r *Repository) GetCustomerByID(ctx context.Context, id int) (*Customer, error) {
	var c Customer
	var createdAt, updatedAt time.Time
	err := r.db.QueryRow(ctx, `
		SELECT id, name, phone, email, address, tax_id, loyalty_points, total_spent, last_purchase_at, note, is_active, is_walk_in, created_at, updated_at
		FROM customers WHERE id = $1
	`, id).Scan(&c.ID, &c.Name, &c.Phone, &c.Email, &c.Address, &c.TaxID, &c.LoyaltyPoints, &c.TotalSpent, &c.LastPurchaseAt, &c.Note, &c.IsActive, &c.IsWalkIn, &createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("customer not found")
		}
		return nil, err
	}
	c.CreatedAt = createdAt.In(jakartaLoc).Format(time.RFC3339)
	c.UpdatedAt = updatedAt.In(jakartaLoc).Format(time.RFC3339)
	return &c, nil
}

func (r *Repository) GetAllCustomers(ctx context.Context, limit, offset int, search string, isActive *bool) ([]Customer, int, error) {
	args := []interface{}{}
	argIdx := 1
	countQuery := `SELECT COUNT(*) FROM customers WHERE is_walk_in = false`
	if search != "" {
		countQuery += fmt.Sprintf(" AND (name ILIKE $%d OR phone ILIKE $%d OR email ILIKE $%d)", argIdx, argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}
	if isActive != nil {
		countQuery += fmt.Sprintf(" AND is_active = $%d", argIdx)
		args = append(args, *isActive)
	}
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT id, name, phone, email, address, tax_id, loyalty_points, total_spent, last_purchase_at, note, is_active, is_walk_in, created_at, updated_at FROM customers WHERE is_walk_in = false`
	queryArgs := []interface{}{}
	argIdx2 := 1
	if search != "" {
		query += fmt.Sprintf(" AND (name ILIKE $%d OR phone ILIKE $%d OR email ILIKE $%d)", argIdx2, argIdx2, argIdx2)
		queryArgs = append(queryArgs, "%"+search+"%")
		argIdx2++
	}
	if isActive != nil {
		query += fmt.Sprintf(" AND is_active = $%d", argIdx2)
		queryArgs = append(queryArgs, *isActive)
		argIdx2++
	}
	query += fmt.Sprintf(" ORDER BY id DESC LIMIT $%d OFFSET $%d", argIdx2, argIdx2+1)
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
		if err := rows.Scan(&c.ID, &c.Name, &c.Phone, &c.Email, &c.Address, &c.TaxID, &c.LoyaltyPoints, &c.TotalSpent, &c.LastPurchaseAt, &c.Note, &c.IsActive, &c.IsWalkIn, &createdAt, &updatedAt); err != nil {
			return nil, 0, err
		}
		c.CreatedAt = createdAt.In(jakartaLoc).Format(time.RFC3339)
		c.UpdatedAt = updatedAt.In(jakartaLoc).Format(time.RFC3339)
		customers = append(customers, c)
	}
	return customers, total, nil
}

func (r *Repository) CreateCustomer(ctx context.Context, customer *Customer) error {
	var createdAt, updatedAt time.Time
	return r.db.QueryRow(ctx, `
		INSERT INTO customers (name, phone, email, address, tax_id, note, is_active, is_walk_in)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`, customer.Name, customer.Phone, customer.Email, customer.Address, customer.TaxID, customer.Note, customer.IsActive, customer.IsWalkIn).
		Scan(&customer.ID, &createdAt, &updatedAt)
}

func (r *Repository) UpdateCustomer(ctx context.Context, customer *Customer, id int) error {
	_, err := r.db.Exec(ctx, `
		UPDATE customers
		SET name = $1, phone = $2, email = $3, address = $4, tax_id = $5, note = $6, is_active = $7, updated_at = NOW()
		WHERE id = $8
	`, customer.Name, customer.Phone, customer.Email, customer.Address, customer.TaxID, customer.Note, customer.IsActive, id)
	return err
}

func (r *Repository) DeleteCustomer(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, `UPDATE customers SET is_active = false, updated_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *Repository) BulkUpdateCustomersStatus(ctx context.Context, ids []int, isActive bool) error {
	_, err := r.db.Exec(ctx, `
		UPDATE customers SET is_active = $1, updated_at = NOW()
		WHERE id = ANY($2) AND is_walk_in = false
	`, isActive, ids)
	return err
}

func (r *Repository) BulkDeleteCustomers(ctx context.Context, ids []int) error {
	_, err := r.db.Exec(ctx, `
		UPDATE customers SET is_active = false, updated_at = NOW()
		WHERE id = ANY($1) AND is_walk_in = false
	`, ids)
	return err
}

func (r *Repository) GetAllCustomersForExport(ctx context.Context) ([]Customer, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, phone, email, address, tax_id, loyalty_points, total_spent, last_purchase_at, note, is_active, is_walk_in, created_at, updated_at
		FROM customers WHERE is_walk_in = false
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var customers []Customer
	for rows.Next() {
		var c Customer
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&c.ID, &c.Name, &c.Phone, &c.Email, &c.Address, &c.TaxID, &c.LoyaltyPoints, &c.TotalSpent, &c.LastPurchaseAt, &c.Note, &c.IsActive, &c.IsWalkIn, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		c.CreatedAt = createdAt.In(jakartaLoc).Format(time.RFC3339)
		c.UpdatedAt = updatedAt.In(jakartaLoc).Format(time.RFC3339)
		customers = append(customers, c)
	}
	return customers, nil
}

func (r *Repository) BulkUpsertCustomers(ctx context.Context, records []CustomerImportRow) ImportResult {
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
	rows, err := r.db.Query(ctx, "SELECT id, phone FROM customers WHERE phone = ANY($1) AND is_walk_in = false", phones)
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
			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d)", offset+1, offset+2, offset+3, offset+4, offset+5, offset+6, offset+7))
			valueArgs = append(valueArgs, rec.Name, rec.Phone, rec.Email, rec.Address, rec.Note, rec.IsActive, updateIDs[i])
		}

		query := fmt.Sprintf(`
			UPDATE customers SET
				name = data.name,
				phone = data.phone,
				email = NULLIF(data.email, ''),
				address = NULLIF(data.address, ''),
				note = NULLIF(data.note, ''),
				is_active = data.is_active,
				updated_at = NOW()
			FROM (VALUES %s) AS data(name text, phone text, email text, address text, note text, is_active boolean, id int)
			WHERE customers.id = data.id
		`, strings.Join(valueStrings, ", "))

		_, err := r.db.Exec(ctx, query, valueArgs...)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("batch update failed: %v", err))
		} else {
			result.Updated = len(updateRecords)
		}
	}

	if len(insertRecords) > 0 {
		valueStrings := make([]string, 0, len(insertRecords))
		valueArgs := make([]interface{}, 0, len(insertRecords)*7)
		for _, rec := range insertRecords {
			offset := len(valueArgs)
			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, NULLIF($%d, ''), NULLIF($%d, ''), NULLIF($%d, ''), $%d, $%d, false)", offset+1, offset+2, offset+3, offset+4, offset+5, offset+6, offset+7))
			valueArgs = append(valueArgs, rec.Name, rec.Phone, rec.Email, rec.Address, rec.Note, rec.IsActive, rec.TaxID)
		}

		query := fmt.Sprintf(`
			INSERT INTO customers (name, phone, email, address, note, is_active, tax_id, is_walk_in)
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
