package customer

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
	c.CreatedAt = createdAt.Format(time.RFC3339)
	c.UpdatedAt = updatedAt.Format(time.RFC3339)
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
	c.CreatedAt = createdAt.Format(time.RFC3339)
	c.UpdatedAt = updatedAt.Format(time.RFC3339)
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
		c.CreatedAt = createdAt.Format(time.RFC3339)
		c.UpdatedAt = updatedAt.Format(time.RFC3339)
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
