package supplier

import (
	"context"
	"fmt"
	"time"

	"retail-pos-system/internal/shared"

	"github.com/jackc/pgx/v5"
)

type Repository struct {
	db shared.DBPool
}

func NewRepository(db shared.DBPool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByID(ctx context.Context, id int) (*Supplier, error) {
	var s Supplier
	var createdAt, updatedAt time.Time

	err := r.db.QueryRow(ctx, `
		SELECT id, name, code, contact_name, email, phone, address, notes, is_active, created_at, updated_at
		FROM suppliers WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(
		&s.ID, &s.Name, &s.Code, &s.ContactName,
		&s.Email, &s.Phone, &s.Address, &s.Notes,
		&s.IsActive, &createdAt, &updatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrSupplierNotFound
		}
		return nil, err
	}

	s.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	s.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &s, nil
}

func (r *Repository) GetByCode(ctx context.Context, code string) (*Supplier, error) {
	var s Supplier
	var createdAt, updatedAt time.Time

	err := r.db.QueryRow(ctx, `
		SELECT id, name, code, contact_name, email, phone, address, notes, is_active, created_at, updated_at
		FROM suppliers WHERE code = $1 AND deleted_at IS NULL
	`, code).Scan(
		&s.ID, &s.Name, &s.Code, &s.ContactName,
		&s.Email, &s.Phone, &s.Address, &s.Notes,
		&s.IsActive, &createdAt, &updatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrSupplierNotFound
		}
		return nil, err
	}

	s.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	s.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &s, nil
}

func (r *Repository) Create(ctx context.Context, supplier *Supplier) error {
	var createdAt, updatedAt time.Time
	err := r.db.QueryRow(ctx, `
		INSERT INTO suppliers (name, code, contact_name, email, phone, address, notes, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`, supplier.Name, supplier.Code, supplier.ContactName,
		supplier.Email, supplier.Phone, supplier.Address, supplier.Notes, supplier.IsActive,
	).Scan(&supplier.ID, &createdAt, &updatedAt)
	if err != nil {
		return fmt.Errorf("insert supplier: %w", err)
	}
	supplier.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	supplier.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return nil
}

func (r *Repository) Update(ctx context.Context, supplier *Supplier) error {
	_, err := r.db.Exec(ctx, `
		UPDATE suppliers
		SET name = $1, code = $2, contact_name = $3, email = $4, phone = $5,
		    address = $6, notes = $7, is_active = $8, updated_at = NOW()
		WHERE id = $9 AND deleted_at IS NULL
	`, supplier.Name, supplier.Code, supplier.ContactName,
		supplier.Email, supplier.Phone, supplier.Address,
		supplier.Notes, supplier.IsActive, supplier.ID)
	if err != nil {
		return fmt.Errorf("update supplier: %w", err)
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, `
		UPDATE suppliers SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("delete supplier: %w", err)
	}
	return nil
}

func (r *Repository) GetAll(ctx context.Context, limit, offset int, search string, isActive *bool) ([]Supplier, int, error) {
	countQuery := `SELECT COUNT(*) FROM suppliers WHERE deleted_at IS NULL`
	dataQuery := `
		SELECT id, name, code, contact_name, email, phone, address, notes, is_active, created_at, updated_at
		FROM suppliers WHERE deleted_at IS NULL`

	var args []interface{}
	argIdx := 1

	if search != "" {
		filter := fmt.Sprintf(" AND (name ILIKE $%d OR code ILIKE $%d OR contact_name ILIKE $%d)", argIdx, argIdx+1, argIdx+2)
		countQuery += filter
		dataQuery += filter
		args = append(args, "%"+search+"%", "%"+search+"%", "%"+search+"%")
		argIdx += 3
	}
	if isActive != nil {
		filter := fmt.Sprintf(" AND is_active = $%d", argIdx)
		countQuery += filter
		dataQuery += filter
		args = append(args, *isActive)
		argIdx++
	}

	var total int
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	dataQuery += fmt.Sprintf(" ORDER BY name ASC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	suppliers, err := scanSuppliers(rows)
	if err != nil {
		return nil, 0, err
	}
	return suppliers, total, nil
}

func (r *Repository) LinkProduct(ctx context.Context, ps *ProductSupplier) error {
	var createdAt time.Time
	err := r.db.QueryRow(ctx, `
		INSERT INTO product_suppliers (product_id, supplier_id, supplier_sku, unit_cost, lead_time_days, is_preferred)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`, ps.ProductID, ps.SupplierID, ps.SupplierSKU, ps.UnitCost, ps.LeadTimeDays, ps.IsPreferred,
	).Scan(&ps.ID, &createdAt)
	if err != nil {
		return fmt.Errorf("link product supplier: %w", err)
	}
	ps.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return nil
}

func (r *Repository) UnlinkProduct(ctx context.Context, productID, supplierID int) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM product_suppliers WHERE product_id = $1 AND supplier_id = $2
	`, productID, supplierID)
	if err != nil {
		return fmt.Errorf("unlink product supplier: %w", err)
	}
	return nil
}

func (r *Repository) GetProductSupplier(ctx context.Context, productID, supplierID int) (*ProductSupplier, error) {
	var ps ProductSupplier
	var createdAt time.Time

	err := r.db.QueryRow(ctx, `
		SELECT id, product_id, supplier_id, supplier_sku, unit_cost, lead_time_days, is_preferred, created_at
		FROM product_suppliers WHERE product_id = $1 AND supplier_id = $2
	`, productID, supplierID).Scan(
		&ps.ID, &ps.ProductID, &ps.SupplierID, &ps.SupplierSKU,
		&ps.UnitCost, &ps.LeadTimeDays, &ps.IsPreferred, &createdAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrProductSupplierNotFound
		}
		return nil, err
	}

	ps.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &ps, nil
}

func (r *Repository) GetPreferredSupplier(ctx context.Context, productID int) (*ProductSupplier, error) {
	var ps ProductSupplier
	var createdAt time.Time

	err := r.db.QueryRow(ctx, `
		SELECT id, product_id, supplier_id, supplier_sku, unit_cost, lead_time_days, is_preferred, created_at
		FROM product_suppliers WHERE product_id = $1 AND is_preferred = true
	`, productID).Scan(
		&ps.ID, &ps.ProductID, &ps.SupplierID, &ps.SupplierSKU,
		&ps.UnitCost, &ps.LeadTimeDays, &ps.IsPreferred, &createdAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrProductSupplierNotFound
		}
		return nil, err
	}

	ps.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &ps, nil
}

func (r *Repository) SetPreferredSupplier(ctx context.Context, productID, supplierID int) error {
	_, err := r.db.Exec(ctx, `
		UPDATE product_suppliers SET is_preferred = false
		WHERE product_id = $1 AND is_preferred = true
	`, productID)
	if err != nil {
		return fmt.Errorf("clear preferred supplier: %w", err)
	}

	_, err = r.db.Exec(ctx, `
		UPDATE product_suppliers SET is_preferred = true
		WHERE product_id = $1 AND supplier_id = $2
	`, productID, supplierID)
	if err != nil {
		return fmt.Errorf("set preferred supplier: %w", err)
	}
	return nil
}

func (r *Repository) UpdateProductSupplier(ctx context.Context, ps *ProductSupplier) error {
	_, err := r.db.Exec(ctx, `
		UPDATE product_suppliers
		SET supplier_sku = $1, unit_cost = $2, lead_time_days = $3, is_preferred = $4, updated_at = NOW()
		WHERE product_id = $5 AND supplier_id = $6
	`, ps.SupplierSKU, ps.UnitCost, ps.LeadTimeDays, ps.IsPreferred, ps.ProductID, ps.SupplierID)
	if err != nil {
		return fmt.Errorf("update product supplier: %w", err)
	}
	return nil
}

func (r *Repository) GetSuppliersByProductID(ctx context.Context, productID int) ([]ProductSupplier, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ps.id, ps.product_id, ps.supplier_id, ps.supplier_sku, ps.unit_cost, ps.lead_time_days, ps.is_preferred, ps.created_at,
		       s.name, s.code
		FROM product_suppliers ps
		JOIN suppliers s ON ps.supplier_id = s.id AND s.deleted_at IS NULL
		WHERE ps.product_id = $1
		ORDER BY ps.is_preferred DESC, s.name ASC
	`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ProductSupplier
	for rows.Next() {
		var ps ProductSupplier
		var createdAt time.Time
		var supplierName, supplierCode string

		err := rows.Scan(
			&ps.ID, &ps.ProductID, &ps.SupplierID, &ps.SupplierSKU,
			&ps.UnitCost, &ps.LeadTimeDays, &ps.IsPreferred, &createdAt,
			&supplierName, &supplierCode,
		)
		if err != nil {
			return nil, err
		}
		ps.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		ps.SupplierName = &supplierName
		ps.SupplierCode = &supplierCode
		result = append(result, ps)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *Repository) GetProductsBySupplierID(ctx context.Context, supplierID int) ([]ProductSupplier, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ps.id, ps.product_id, ps.supplier_id, ps.supplier_sku, ps.unit_cost, ps.lead_time_days, ps.is_preferred, ps.created_at,
		       p.name, p.sku
		FROM product_suppliers ps
		JOIN products p ON ps.product_id = p.id AND p.deleted_at IS NULL
		WHERE ps.supplier_id = $1
		ORDER BY p.name ASC
	`, supplierID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ProductSupplier
	for rows.Next() {
		var ps ProductSupplier
		var createdAt time.Time
		var productName, productSKU string

		err := rows.Scan(
			&ps.ID, &ps.ProductID, &ps.SupplierID, &ps.SupplierSKU,
			&ps.UnitCost, &ps.LeadTimeDays, &ps.IsPreferred, &createdAt,
			&productName, &productSKU,
		)
		if err != nil {
			return nil, err
		}
		ps.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		ps.ProductName = &productName
		ps.ProductSKU = &productSKU
		result = append(result, ps)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *Repository) HasPreferredSupplier(ctx context.Context, productID int) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM product_suppliers WHERE product_id = $1 AND is_preferred = true)
	`, productID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check preferred supplier: %w", err)
	}
	return exists, nil
}

func scanSuppliers(rows pgx.Rows) ([]Supplier, error) {
	var suppliers []Supplier
	for rows.Next() {
		var s Supplier
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&s.ID, &s.Name, &s.Code, &s.ContactName,
			&s.Email, &s.Phone, &s.Address, &s.Notes,
			&s.IsActive, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, err
		}
		s.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		s.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		suppliers = append(suppliers, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return suppliers, nil
}

type SupplierImportRow struct {
	Row         int
	Code        string
	Name        string
	ContactName string
	Phone       string
	Email       string
	Address     string
	Notes       string
	IsActive    bool
}

type SupplierImportPayload struct {
	Code        string
	Name        string
	ContactName *string
	Phone       *string
	Email       *string
	Address     *string
	Notes       *string
	IsActive    bool
}

func (r *Repository) BulkInsertSuppliers(ctx context.Context, payloads []SupplierImportPayload) (int, error) {
	if len(payloads) == 0 {
		return 0, nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	count := 0
	for _, p := range payloads {
		_, err := tx.Exec(ctx, `
			INSERT INTO suppliers (name, code, contact_name, email, phone, address, notes, is_active)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, p.Name, p.Code, p.ContactName, p.Email, p.Phone, p.Address, p.Notes, p.IsActive)
		if err != nil {
			return count, fmt.Errorf("insert supplier: %w", err)
		}
		count++
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return count, nil
}

func (r *Repository) BulkUpdateSuppliers(ctx context.Context, payloads []SupplierImportPayload) (int, error) {
	if len(payloads) == 0 {
		return 0, nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	count := 0
	for _, p := range payloads {
		tag, err := tx.Exec(ctx, `
			UPDATE suppliers
			SET name = $1, contact_name = $2, email = $3, phone = $4,
			    address = $5, notes = $6, is_active = $7, updated_at = NOW()
			WHERE code = $8 AND deleted_at IS NULL
		`, p.Name, p.ContactName, p.Email, p.Phone, p.Address, p.Notes, p.IsActive, p.Code)
		if err != nil {
			return count, fmt.Errorf("update supplier: %w", err)
		}
		if tag.RowsAffected() > 0 {
			count++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return count, nil
}

func (r *Repository) GetAllForExport(ctx context.Context) ([]Supplier, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, code, contact_name, email, phone, address, notes, is_active, created_at, updated_at
		FROM suppliers WHERE deleted_at IS NULL ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSuppliers(rows)
}
