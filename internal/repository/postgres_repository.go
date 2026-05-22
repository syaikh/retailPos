package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"retail-pos-system/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var jakartaLoc *time.Location

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *postgresRepository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) GetDB() *pgxpool.Pool {
	return r.db
}

// ==================== USER ====================

func (r *postgresRepository) GetByID(id int) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return r.getUserByID(ctx, id)
}

func (r *postgresRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	var u domain.User
	var storeID sql.NullInt64
	var createdAt, updatedAt time.Time

	err := r.db.QueryRow(ctx, `
		SELECT id, username, email, password_hash, role_id, store_id, is_active, created_at, updated_at
		FROM users WHERE username = $1 AND deleted_at IS NULL
	`, username).Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.RoleID, &storeID, &u.IsActive, &createdAt, &updatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}
	u.CreatedAt = createdAt.Format(time.RFC3339)
	u.UpdatedAt = updatedAt.Format(time.RFC3339)
	if storeID.Valid {
		v := int(storeID.Int64)
		u.StoreID = &v
	}
	if u.RoleID > 0 {
		role, err := r.GetRoleByID(ctx, u.RoleID)
		if err != nil {
			log.Printf("GetRoleByID error for role_id %d: %v", u.RoleID, err)
		} else if role != nil {
			u.Role = *role
		}
	}
	return &u, nil
}

func (r *postgresRepository) getUserByID(ctx context.Context, id int) (*domain.User, error) {
	var u domain.User
	var storeID sql.NullInt64
	var createdAt, updatedAt time.Time

	err := r.db.QueryRow(ctx, `
		SELECT id, username, email, password_hash, role_id, store_id, is_active, created_at, updated_at
		FROM users WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.RoleID, &storeID, &u.IsActive, &createdAt, &updatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	u.CreatedAt = createdAt.Format(time.RFC3339)
	u.UpdatedAt = updatedAt.Format(time.RFC3339)
	if storeID.Valid {
		v := int(storeID.Int64)
		u.StoreID = &v
	}
	if u.RoleID > 0 {
		role, _ := r.GetRoleByID(ctx, u.RoleID)
		if role != nil {
			u.Role = *role
		}
	}
	return &u, nil
}

func (r *postgresRepository) GetAllUsers(ctx context.Context, limit, offset int, search string) ([]domain.User, int, error) {
	var users []domain.User
	var total int

	query := `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`
	args := []interface{}{}
	if search != "" {
		query += " AND (username ILIKE $1 OR email ILIKE $1)"
		args = append(args, "%"+search+"%")
	}

	err := r.db.QueryRow(ctx, query, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	query = `SELECT id, username, email, password_hash, role_id, store_id, is_active, created_at, updated_at FROM users WHERE deleted_at IS NULL`
	args2 := []interface{}{}
	if search != "" {
		query += " AND (username ILIKE $1 OR email ILIKE $1)"
		args2 = append(args2, "%"+search+"%")
	}
	query += fmt.Sprintf(" ORDER BY id LIMIT $%d OFFSET $%d", len(args2)+1, len(args2)+2)
	args2 = append(args2, limit, offset)

	rows, err := r.db.Query(ctx, query, args2...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var u domain.User
		var storeID sql.NullInt64
		var createdAt, updatedAt time.Time

		err = rows.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.RoleID, &storeID, &u.IsActive, &createdAt, &updatedAt)
		if err != nil {
			continue
		}
		u.CreatedAt = createdAt.Format(time.RFC3339)
		u.UpdatedAt = updatedAt.Format(time.RFC3339)
		if storeID.Valid {
			v := int(storeID.Int64)
			u.StoreID = &v
		}
		if u.RoleID > 0 {
			role, _ := r.GetRoleByID(ctx, u.RoleID)
			if role != nil {
				u.Role = *role
			}
		}
		users = append(users, u)
	}
	return users, total, nil
}

func (r *postgresRepository) CreateUser(ctx context.Context, user *domain.User) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, role_id, store_id, is_active)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at, updated_at
	`, user.Username, user.Email, user.Password, user.RoleID, user.StoreID, user.IsActive).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *postgresRepository) UpdateUser(ctx context.Context, user *domain.User) error {
	if user.Password != "" {
		_, err := r.db.Exec(ctx, `
			UPDATE users SET username = $1, email = $2, password_hash = $3, role_id = $4, store_id = $5, is_active = $6, updated_at = NOW()
			WHERE id = $7
		`, user.Username, user.Email, user.Password, user.RoleID, user.StoreID, user.IsActive, user.ID)
		return err
	}
	_, err := r.db.Exec(ctx, `
		UPDATE users SET username = $1, email = $2, role_id = $3, store_id = $4, is_active = $5, updated_at = NOW()
		WHERE id = $6
	`, user.Username, user.Email, user.RoleID, user.StoreID, user.IsActive, user.ID)
	return err
}

func (r *postgresRepository) DeleteUser(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, "UPDATE users SET deleted_at = NOW() WHERE id = $1", id)
	return err
}

// ==================== ROLE ====================

func (r *postgresRepository) GetRoleByID(ctx context.Context, id int) (*domain.Role, error) {
	var role domain.Role
	var isSystem bool
	var createdAt time.Time
	err := r.db.QueryRow(ctx, "SELECT id, name, description, is_system, created_at FROM roles WHERE id = $1", id).Scan(
		&role.ID, &role.Name, &role.Description, &isSystem, &createdAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("role not found")
		}
		return nil, err
	}
	role.IsSystem = isSystem
	role.CreatedAt = createdAt.Format(time.RFC3339)
	return &role, nil
}

func (r *postgresRepository) GetAllRoles(ctx context.Context) ([]domain.Role, error) {
	rows, err := r.db.Query(ctx, "SELECT id, name, description, is_system, created_at FROM roles ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []domain.Role
	for rows.Next() {
		var rl domain.Role
		var isSystem bool
		var createdAt time.Time
		if err := rows.Scan(&rl.ID, &rl.Name, &rl.Description, &isSystem, &createdAt); err != nil {
			return nil, err
		}
		rl.IsSystem = isSystem
		rl.CreatedAt = createdAt.Format(time.RFC3339)
		roles = append(roles, rl)
	}
	return roles, nil
}

func (r *postgresRepository) CreateRole(ctx context.Context, role *domain.Role) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO roles (name, description, is_system) VALUES ($1, $2, $3)
		RETURNING id, created_at
	`, role.Name, role.Description, role.IsSystem).Scan(&role.ID, &role.CreatedAt)
}

func (r *postgresRepository) UpdateRole(ctx context.Context, role *domain.Role) error {
	_, err := r.db.Exec(ctx, "UPDATE roles SET name = $1, description = $2 WHERE id = $3", role.Name, role.Description, role.ID)
	return err
}

func (r *postgresRepository) DeleteRole(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, "DELETE FROM roles WHERE id = $1", id)
	return err
}

func (r *postgresRepository) GetRolePermissions(ctx context.Context, roleID int) ([]domain.Permission, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.id, p.code, p.name, p.created_at
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = $1
	`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []domain.Permission
	for rows.Next() {
		var p domain.Permission
		var createdAt time.Time
		if err := rows.Scan(&p.ID, &p.Code, &p.Name, &createdAt); err != nil {
			return nil, err
		}
		p.CreatedAt = createdAt.Format(time.RFC3339)
		perms = append(perms, p)
	}
	return perms, nil
}

func (r *postgresRepository) UpdateRolePermissions(ctx context.Context, roleID int, permissionIDs []int) error {
	_, err := r.db.Exec(ctx, "DELETE FROM role_permissions WHERE role_id = $1", roleID)
	if err != nil {
		return err
	}
	for _, pid := range permissionIDs {
		_, err = r.db.Exec(ctx, "INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", roleID, pid)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *postgresRepository) GetAllPermissions(ctx context.Context) ([]domain.Permission, error) {
	rows, err := r.db.Query(ctx, "SELECT id, code, name, created_at FROM permissions ORDER BY code")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []domain.Permission
	for rows.Next() {
		var p domain.Permission
		var createdAt time.Time
		if err := rows.Scan(&p.ID, &p.Code, &p.Name, &createdAt); err != nil {
			return nil, err
		}
		p.CreatedAt = createdAt.Format(time.RFC3339)
		perms = append(perms, p)
	}
	return perms, nil
}

// ==================== PRODUCT ====================

func (r *postgresRepository) GetProductByID(ctx context.Context, id int, storeID *int) (*domain.Product, error) {
	var p domain.Product
	var barcode sql.NullString
	var categoryIDVal, storeIDVal sql.NullInt64
	var categoryName sql.NullString
	var createdAt, updatedAt time.Time

	query := `
		SELECT p.id, p.sku, p.name, p.barcode, p.category_id, c.name as category_name, p.price, p.cost, p.stock, p.status,
		       p.store_id, p.created_at, p.updated_at
		FROM products p 
		LEFT JOIN categories c ON p.category_id = c.id 
		WHERE p.id = $1 AND p.deleted_at IS NULL`

	args := []interface{}{id}
	if storeID != nil {
		query += fmt.Sprintf(" AND p.store_id = $%d", len(args)+1)
		args = append(args, *storeID)
	}

	err := r.db.QueryRow(ctx, query, args...).Scan(&p.ID, &p.SKU, &p.Name, &barcode, &categoryIDVal, &categoryName, &p.Price, &p.Cost, &p.Stock, &p.Status,
		&storeIDVal, &createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("product not found")
		}
		return nil, err
	}

	if barcode.Valid {
		p.Barcode = &barcode.String
	}
	if categoryIDVal.Valid {
		v := int(categoryIDVal.Int64)
		p.CategoryID = &v
	}
	if categoryName.Valid {
		p.CategoryName = &categoryName.String
	}
	if storeIDVal.Valid {
		v := int(storeIDVal.Int64)
		p.StoreID = &v
	}
	p.CreatedAt = createdAt.Format(time.RFC3339)
	p.UpdatedAt = updatedAt.Format(time.RFC3339)

	return &p, nil
}

func (r *postgresRepository) GetProductBySKU(ctx context.Context, sku string, storeID *int) (*domain.Product, error) {
	var p domain.Product
	var barcode sql.NullString
	var categoryIDVal, storeIDVal sql.NullInt64
	var categoryName sql.NullString
	var createdAt, updatedAt time.Time

	query := `
		SELECT p.id, p.sku, p.name, p.barcode, p.category_id, c.name as category_name, p.price, p.cost, p.stock, p.status,
		       p.store_id, p.created_at, p.updated_at
		FROM products p 
		LEFT JOIN categories c ON p.category_id = c.id 
		WHERE p.sku = $1 AND p.deleted_at IS NULL`

	args := []interface{}{sku}
	if storeID != nil {
		query += fmt.Sprintf(" AND p.store_id = $%d", len(args)+1)
		args = append(args, *storeID)
	}

	err := r.db.QueryRow(ctx, query, args...).Scan(&p.ID, &p.SKU, &p.Name, &barcode, &categoryIDVal, &categoryName, &p.Price, &p.Cost, &p.Stock, &p.Status,
		&storeIDVal, &createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("product not found")
		}
		return nil, err
	}

	if barcode.Valid {
		p.Barcode = &barcode.String
	}
	if categoryIDVal.Valid {
		v := int(categoryIDVal.Int64)
		p.CategoryID = &v
	}
	if categoryName.Valid {
		p.CategoryName = &categoryName.String
	}
	if storeIDVal.Valid {
		v := int(storeIDVal.Int64)
		p.StoreID = &v
	}
	p.CreatedAt = createdAt.Format(time.RFC3339)
	p.UpdatedAt = updatedAt.Format(time.RFC3339)

	return &p, nil
}

func (r *postgresRepository) GetAllProducts(ctx context.Context, limit, offset int, search string, categoryIDs []int, sortBy, sortDir string, maxStock *int, storeID *int) ([]domain.Product, int, error) {
	var products []domain.Product
	var total int

	query := `SELECT COUNT(*) 
		FROM products p 
		WHERE p.deleted_at IS NULL`
	args := []interface{}{}
	argIdx := 1
	if search != "" {
		query += fmt.Sprintf(" AND (p.name ILIKE $%d OR p.sku ILIKE $%d OR p.barcode ILIKE $%d)", argIdx, argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}
	if len(categoryIDs) > 0 {
		placeholders := make([]string, len(categoryIDs))
		for i, cid := range categoryIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIdx)
			args = append(args, cid)
			argIdx++
		}
		query += fmt.Sprintf(" AND p.category_id IN (%s)", strings.Join(placeholders, ","))
	}
	if maxStock != nil {
		query += fmt.Sprintf(" AND p.stock <= $%d", argIdx)
		args = append(args, *maxStock)
		argIdx++
	}
	if storeID != nil {
		query += fmt.Sprintf(" AND p.store_id = $%d", argIdx)
		args = append(args, *storeID)
		argIdx++
	}

	err := r.db.QueryRow(ctx, query, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count products: %w", err)
	}

	query2 := `SELECT p.id, p.sku, p.name, p.barcode, p.category_id, c.name as category_name, p.price, p.cost, p.stock, p.status, p.store_id, p.created_at, p.updated_at 
		FROM products p 
		LEFT JOIN categories c ON p.category_id = c.id 
		WHERE p.deleted_at IS NULL`
	args2 := []interface{}{}
	argIdx2 := 1
	if search != "" {
		query2 += fmt.Sprintf(" AND (p.name ILIKE $%d OR p.sku ILIKE $%d OR p.barcode ILIKE $%d)", argIdx2, argIdx2, argIdx2)
		args2 = append(args2, "%"+search+"%")
		argIdx2++
	}
	if len(categoryIDs) > 0 {
		placeholders := make([]string, len(categoryIDs))
		for i, cid := range categoryIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIdx2)
			args2 = append(args2, cid)
			argIdx2++
		}
		query2 += fmt.Sprintf(" AND p.category_id IN (%s)", strings.Join(placeholders, ","))
	}
	if maxStock != nil {
		query2 += fmt.Sprintf(" AND p.stock <= $%d", argIdx2)
		args2 = append(args2, *maxStock)
		argIdx2++
	}
	if storeID != nil {
		query2 += fmt.Sprintf(" AND p.store_id = $%d", argIdx2)
		args2 = append(args2, *storeID)
		argIdx2++
	}
	if sortBy != "" {
		query2 += fmt.Sprintf(" ORDER BY %s", sortBy)
		if sortDir != "" {
			query2 += " " + sortDir
		}
	} else {
		query2 += " ORDER BY p.id DESC"
	}
	query2 += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx2, argIdx2+1)
	args2 = append(args2, limit, offset)

	rows, err := r.db.Query(ctx, query2, args2...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query products: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var p domain.Product
		var barcodeVal sql.NullString
		var categoryIDVal, storeIDVal sql.NullInt64
		var categoryName sql.NullString
		var createdAt, updatedAt time.Time

		err = rows.Scan(&p.ID, &p.SKU, &p.Name, &barcodeVal, &categoryIDVal, &categoryName, &p.Price, &p.Cost, &p.Stock, &p.Status,
			&storeIDVal, &createdAt, &updatedAt)
		if err != nil {
			continue
		}
		if barcodeVal.Valid {
			p.Barcode = &barcodeVal.String
		}
		if categoryIDVal.Valid {
			v := int(categoryIDVal.Int64)
			p.CategoryID = &v
		}
		if categoryName.Valid {
			p.CategoryName = &categoryName.String
		}
		if storeIDVal.Valid {
			v := int(storeIDVal.Int64)
			p.StoreID = &v
		}
		p.CreatedAt = createdAt.Format(time.RFC3339)
		p.UpdatedAt = updatedAt.Format(time.RFC3339)
		products = append(products, p)
	}
	return products, total, nil
}

func (r *postgresRepository) CreateProduct(ctx context.Context, product *domain.Product) error {
	var barcode interface{}
	if product.Barcode != nil {
		barcode = *product.Barcode
	} else {
		barcode = nil
	}
	var categoryID, storeIDVal interface{}
	if product.CategoryID != nil {
		categoryID = *product.CategoryID
	} else {
		categoryID = nil
	}
	if product.StoreID != nil {
		storeIDVal = *product.StoreID
	} else {
		storeIDVal = nil
	}

	// Phase 1 extension fields
	var brandID, taxClassID, unitOfMeasureID interface{}
	var weightGrams, defaultDiscount interface{}
	
	if product.BrandID != nil {
		brandID = *product.BrandID
	}
	if product.TaxClassID != nil {
		taxClassID = *product.TaxClassID
	}
	if product.UnitOfMeasureID != nil {
		unitOfMeasureID = *product.UnitOfMeasureID
	}
	if product.WeightGrams != nil {
		weightGrams = *product.WeightGrams
	}
	if product.DefaultDiscountPct != nil {
		defaultDiscount = *product.DefaultDiscountPct
	}

	return r.db.QueryRow(ctx, `
		INSERT INTO products (sku, name, barcode, category_id, price, cost, stock, store_id, status,
		                    brand_id, description, tax_class_id, weight_grams, unit_of_measure_id, default_discount_percent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_at, updated_at
	`, product.SKU, product.Name, barcode, categoryID, product.Price, product.Cost, product.Stock, storeIDVal, product.Status,
		brandID, product.Description, taxClassID, weightGrams, unitOfMeasureID, defaultDiscount).
		Scan(&product.ID, &product.CreatedAt, &product.UpdatedAt)
}


func (r *postgresRepository) UpdateProduct(ctx context.Context, product *domain.Product, storeID *int) error {
	var barcode interface{}
	if product.Barcode != nil {
		barcode = *product.Barcode
	} else {
		barcode = nil
	}
	var categoryID, storeIDVal interface{}
	if product.CategoryID != nil {
		categoryID = *product.CategoryID
	} else {
		categoryID = nil
	}
	if product.StoreID != nil {
		storeIDVal = *product.StoreID
	} else {
		storeIDVal = nil
	}

	// Phase 1 extension fields
	var brandID, taxClassID, unitOfMeasureID interface{}
	var weightGrams, defaultDiscount interface{}
	
	if product.BrandID != nil {
		brandID = *product.BrandID
	} else {
		brandID = nil
	}
	if product.TaxClassID != nil {
		taxClassID = *product.TaxClassID
	} else {
		taxClassID = nil
	}
	if product.UnitOfMeasureID != nil {
		unitOfMeasureID = *product.UnitOfMeasureID
	} else {
		unitOfMeasureID = nil
	}
	if product.WeightGrams != nil {
		weightGrams = *product.WeightGrams
	} else {
		weightGrams = nil
	}
	if product.DefaultDiscountPct != nil {
		defaultDiscount = *product.DefaultDiscountPct
	} else {
		defaultDiscount = nil
	}

	_, err := r.db.Exec(ctx, `
		UPDATE products SET sku = $1, name = $2, barcode = $3, category_id = $4, price = $5,
			cost = $6, stock = $7, store_id = $8, status = $9, updated_at = NOW(),
			brand_id = $10, description = $11, tax_class_id = $12, weight_grams = $13, unit_of_measure_id = $14, default_discount_percent = $15
		WHERE id = $16
	`, product.SKU, product.Name, barcode, categoryID, product.Price, product.Cost, product.Stock, storeIDVal, product.Status,
		brandID, product.Description, taxClassID, weightGrams, unitOfMeasureID, defaultDiscount, product.ID)
	return err
}

func (r *postgresRepository) DeleteProduct(ctx context.Context, id int, storeID *int) error {
	_, err := r.db.Exec(ctx, "UPDATE products SET deleted_at = NOW() WHERE id = $1", id)
	return err
}

// ==================== CATEGORY ====================

func (r *postgresRepository) ListCategories(ctx context.Context) ([]domain.Category, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, description, is_active, created_at
		FROM categories
		WHERE is_active = true
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []domain.Category
	for rows.Next() {
		var c domain.Category
		var createdAt time.Time
		err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.IsActive, &createdAt)
		if err != nil {
			return nil, err
		}
		c.CreatedAt = createdAt.Format(time.RFC3339)
		c.UpdatedAt = "" // Not stored in database
		categories = append(categories, c)
	}
	return categories, nil
}

func (r *postgresRepository) GetCategoryIDByName(ctx context.Context, name string) (int, error) {
	var id int
	query := "SELECT id FROM categories WHERE name = $1 AND is_active = true"
	err := r.db.QueryRow(ctx, query, name).Scan(&id)
	return id, err
}

func (r *postgresRepository) GetDeletedProductByBarcode(ctx context.Context, barcode string, storeID *int) (*domain.Product, error) {
	var p domain.Product
	var barcodeVal sql.NullString
	var categoryIDVal, storeIDVal sql.NullInt64
	var categoryName sql.NullString
	var createdAt, updatedAt time.Time

	query := `
		SELECT p.id, p.sku, p.name, p.barcode, p.category_id, c.name as category_name, p.price, p.cost, p.stock, p.status,
		       p.store_id, p.created_at, p.updated_at
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.id
		WHERE p.barcode = $1 AND p.deleted_at IS NOT NULL`

	args := []interface{}{barcode}
	if storeID != nil {
		query += fmt.Sprintf(" AND p.store_id = $%d", len(args)+1)
		args = append(args, *storeID)
	}

	err := r.db.QueryRow(ctx, query, args...).Scan(&p.ID, &p.SKU, &p.Name, &barcodeVal, &categoryIDVal, &categoryName, &p.Price, &p.Cost, &p.Stock, &p.Status,
		&storeIDVal, &createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("product not found")
		}
		return nil, err
	}

	if barcodeVal.Valid {
		p.Barcode = &barcodeVal.String
	}
	if categoryIDVal.Valid {
		v := int(categoryIDVal.Int64)
		p.CategoryID = &v
	}
	if categoryName.Valid {
		p.CategoryName = &categoryName.String
	}
	if storeIDVal.Valid {
		v := int(storeIDVal.Int64)
		p.StoreID = &v
	}
	p.CreatedAt = createdAt.Format(time.RFC3339)
	p.UpdatedAt = updatedAt.Format(time.RFC3339)

	return &p, nil
}

func (r *postgresRepository) RestoreProduct(ctx context.Context, product *domain.Product) error {
	var barcode interface{}
	if product.Barcode != nil {
		barcode = *product.Barcode
	}
	var categoryID, storeIDVal interface{}
	if product.CategoryID != nil {
		categoryID = *product.CategoryID
	}
	if product.StoreID != nil {
		storeIDVal = *product.StoreID
	}

	_, err := r.db.Exec(ctx, `
		UPDATE products SET sku = $1, name = $2, barcode = $3, category_id = $4, price = $5, cost = $6, stock = $7,
		    store_id = $8, status = $9, deleted_at = NULL, updated_at = NOW()
		WHERE id = $10
	`, product.SKU, product.Name, barcode, categoryID, product.Price, product.Cost, product.Stock, storeIDVal, product.Status, product.ID)
	return err
}

// ==================== SALE ====================

func (r *postgresRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.Begin(ctx)
}

func (r *postgresRepository) CreateSale(ctx context.Context, tx pgx.Tx, sale *domain.Sale, items []domain.SaleItem) error {
	var createdAt, updatedAt time.Time
	err := tx.QueryRow(ctx, `
		INSERT INTO sales (invoice_number, cashier_id, store_id, subtotal, discount, tax, total_amount, payment_method, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`, sale.InvoiceNumber, sale.CashierID, sale.StoreID, sale.Subtotal, sale.Discount, sale.Tax, sale.TotalAmount, sale.PaymentMethod, sale.Status).
		Scan(&sale.ID, &createdAt, &updatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert sale: %w", err)
	}
	sale.CreatedAt = createdAt.Format(time.RFC3339)
	sale.UpdatedAt = updatedAt.Format(time.RFC3339)

	for i := range items {
		// 1. Insert sale item
		_, err = tx.Exec(ctx, `
			INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal)
			VALUES ($1, $2, $3, $4, $5)
		`, sale.ID, items[i].ProductID, items[i].Quantity, items[i].UnitPrice, items[i].Subtotal)
		if err != nil {
			return fmt.Errorf("failed to insert sale item for product %d: %w", items[i].ProductID, err)
		}

		// 2. Update product stock (database constraint CHECK (stock >= 0) will prevent negative stock)
		cmd, err := tx.Exec(ctx, `
			UPDATE products 
			SET stock = stock - $1, updated_at = NOW() 
			WHERE id = $2 AND deleted_at IS NULL
		`, items[i].Quantity, items[i].ProductID)

		if err != nil {
			// This could be a constraint violation (insufficient stock)
			return fmt.Errorf("failed to update stock for product %d: %w", items[i].ProductID, err)
		}
		if cmd.RowsAffected() == 0 {
			return fmt.Errorf("product %d not found or already deleted", items[i].ProductID)
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

func (r *postgresRepository) GetSaleByID(ctx context.Context, id int) (*domain.Sale, error) {
	var sale domain.Sale
	var storeID sql.NullInt64
	var createdAt, updatedAt time.Time

	err := r.db.QueryRow(ctx, `
		SELECT id, invoice_number, cashier_id, store_id, subtotal, discount, tax, total_amount, payment_method, status, created_at, updated_at
		FROM sales WHERE id = $1
	`, id).Scan(&sale.ID, &sale.InvoiceNumber, &sale.CashierID, &storeID, &sale.Subtotal, &sale.Discount, &sale.Tax,
		&sale.TotalAmount, &sale.PaymentMethod, &sale.Status, &createdAt, &updatedAt)
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
			SELECT si.id, si.sale_id, si.product_id, p.name, si.quantity, si.unit_price, si.subtotal
			FROM sale_items si
			JOIN products p ON si.product_id = p.id
			WHERE si.sale_id = $1
		`, sale.ID)
	if err != nil {
		log.Printf("Warning: failed to load items for sale %d: %v", sale.ID, err)
	} else {
		for itemRows.Next() {
			var item domain.SaleItem
			if scanErr := itemRows.Scan(&item.ID, &item.SaleID, &item.ProductID, &item.Name, &item.Quantity, &item.UnitPrice, &item.Subtotal); scanErr != nil {
				log.Printf("Warning: failed to scan item row: %v", scanErr)
				continue
			}
			sale.Items = append(sale.Items, item)
		}
		itemRows.Close()
	}

	return &sale, nil
}

func (r *postgresRepository) GetAllSales(ctx context.Context, limit, offset int, search string, sortBy, sortDir, startDate, endDate string, storeID *int) ([]domain.Sale, int, error) {
	var sales []domain.Sale
	var total int

	// ---- COUNT QUERY ----
	// When searching by product name we need to check sale_items + products,
	// so use a sub-select to avoid a messy multi-join count.
	countQuery := `SELECT COUNT(*) FROM sales s WHERE 1=1`
	countArgs := []interface{}{}
	argIdx := 1

	if search != "" {
		countQuery += fmt.Sprintf(" AND (s.invoice_number ILIKE $%d OR s.id IN (SELECT DISTINCT si.sale_id FROM sale_items si JOIN products p ON si.product_id = p.id WHERE p.name ILIKE $%d))", argIdx, argIdx)
		countArgs = append(countArgs, "%"+search+"%")
		argIdx++
	}
	if startDate != "" {
		start, _ := time.ParseInLocation("2006-01-02", startDate, time.UTC)
		countQuery += fmt.Sprintf(" AND s.created_at >= $%d", argIdx)
		countArgs = append(countArgs, start)
		argIdx++
	}
	if endDate != "" {
		end, _ := time.ParseInLocation("2006-01-02", endDate, time.UTC)
		countQuery += fmt.Sprintf(" AND s.created_at < $%d", argIdx)
		countArgs = append(countArgs, end.Add(24*time.Hour-time.Nanosecond))
		argIdx++
	}
	if storeID != nil {
		countQuery += fmt.Sprintf(" AND s.store_id = $%d", argIdx)
		countArgs = append(countArgs, *storeID)
		argIdx++
	}

	err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// ---- DATA QUERY ----
	query := `SELECT s.id, s.invoice_number, s.cashier_id, s.store_id, s.subtotal, s.discount, s.tax, s.total_amount, s.payment_method, s.status, s.created_at, s.updated_at FROM sales s WHERE 1=1`
	args2 := []interface{}{}
	argIdx2 := 1

	if search != "" {
		query += fmt.Sprintf(" AND (s.invoice_number ILIKE $%d OR s.id IN (SELECT DISTINCT si.sale_id FROM sale_items si JOIN products p ON si.product_id = p.id WHERE p.name ILIKE $%d))", argIdx2, argIdx2)
		args2 = append(args2, "%"+search+"%")
		argIdx2++
	}
	if startDate != "" {
		start, _ := time.ParseInLocation("2006-01-02", startDate, time.UTC)
		query += fmt.Sprintf(" AND s.created_at >= $%d", argIdx2)
		args2 = append(args2, start)
		argIdx2++
	}
	if endDate != "" {
		end, _ := time.ParseInLocation("2006-01-02", endDate, time.UTC)
		query += fmt.Sprintf(" AND s.created_at < $%d", argIdx2)
		args2 = append(args2, end.Add(24*time.Hour-time.Nanosecond))
		argIdx2++
	}
	if storeID != nil {
		query += fmt.Sprintf(" AND s.store_id = $%d", argIdx2)
		args2 = append(args2, *storeID)
		argIdx2++
	}
	if sortBy != "" {
		query += fmt.Sprintf(" ORDER BY %s", sortBy)
		if sortDir != "" {
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

	for rows.Next() {
		var s domain.Sale
		var storeIDVal sql.NullInt64
		var createdAt, updatedAt time.Time
		err = rows.Scan(&s.ID, &s.InvoiceNumber, &s.CashierID, &storeIDVal, &s.Subtotal, &s.Discount, &s.Tax,
			&s.TotalAmount, &s.PaymentMethod, &s.Status, &createdAt, &updatedAt)
		if err != nil {
			continue
		}
		if storeIDVal.Valid {
			v := int(storeIDVal.Int64)
			s.StoreID = &v
		}
		s.CreatedAt = createdAt.Format(time.RFC3339)
		s.UpdatedAt = updatedAt.Format(time.RFC3339)

		// Load sale items
		itemRows, err := r.db.Query(ctx, `
			SELECT si.id, si.sale_id, si.product_id, p.name, si.quantity, si.unit_price, si.subtotal
			FROM sale_items si
			JOIN products p ON si.product_id = p.id
			WHERE si.sale_id = $1
		`, s.ID)
		if err == nil {
			for itemRows.Next() {
				var item domain.SaleItem
				err = itemRows.Scan(&item.ID, &item.SaleID, &item.ProductID, &item.Name, &item.Quantity, &item.UnitPrice, &item.Subtotal)
				if err == nil {
					s.Items = append(s.Items, item)
				}
			}
			itemRows.Close()
		}

		sales = append(sales, s)
	}
	return sales, total, nil
}

// ==================== AUDIT ====================

func (r *postgresRepository) CreateAuditLog(ctx context.Context, log *domain.AuditLog) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO audit_logs (user_id, role, action, entity_type, entity_id, ip_address, user_agent, old_values, new_values)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, log.UserID, log.Role, log.Action, log.EntityType, log.EntityID, log.IPAddress, log.UserAgent, log.OldValues, log.NewValues)
	return err
}

func (r *postgresRepository) GetAuditLogs(ctx context.Context, limit, offset int, userID *int, search string, action string) ([]domain.AuditLog, int, error) {
	var logs []domain.AuditLog
	var total int

	query := `SELECT COUNT(*) FROM audit_logs al LEFT JOIN users u ON al.user_id = u.id WHERE 1=1`
	args := []interface{}{}
	if userID != nil {
		query += fmt.Sprintf(" AND al.user_id = $%d", len(args)+1)
		args = append(args, *userID)
	}
	if action != "" {
		query += fmt.Sprintf(" AND al.action = $%d", len(args)+1)
		args = append(args, action)
	}
	if search != "" {
		query += fmt.Sprintf(" AND (u.username ILIKE $%d OR al.entity_type ILIKE $%d OR al.ip_address::text ILIKE $%d)", len(args)+1, len(args)+1, len(args)+1)
		args = append(args, "%"+search+"%")
	}

	err := r.db.QueryRow(ctx, query, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query = `SELECT al.id, al.user_id, COALESCE(u.username, 'Unknown'), COALESCE(al.role, ''), al.action, al.entity_type, al.entity_id, COALESCE(al.ip_address::text, ''), COALESCE(al.user_agent, ''), al.old_values, al.new_values, al.created_at::text FROM audit_logs al LEFT JOIN users u ON al.user_id = u.id WHERE 1=1`
	args2 := []interface{}{}
	if userID != nil {
		query += fmt.Sprintf(" AND al.user_id = $%d", len(args2)+1)
		args2 = append(args2, *userID)
	}
	if action != "" {
		query += fmt.Sprintf(" AND al.action = $%d", len(args2)+1)
		args2 = append(args2, action)
	}
	if search != "" {
		query += fmt.Sprintf(" AND (u.username ILIKE $%d OR al.entity_type ILIKE $%d OR al.ip_address ILIKE $%d)", len(args2)+1, len(args2)+1, len(args2)+1)
		args2 = append(args2, "%"+search+"%")
	}
	query += fmt.Sprintf(" ORDER BY al.created_at DESC LIMIT $%d OFFSET $%d", len(args2)+1, len(args2)+2)
	args2 = append(args2, limit, offset)

	rows, err := r.db.Query(ctx, query, args2...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var al domain.AuditLog
		err = rows.Scan(&al.ID, &al.UserID, &al.Username, &al.Role, &al.Action, &al.EntityType, &al.EntityID, &al.IPAddress, &al.UserAgent, &al.OldValues, &al.NewValues, &al.CreatedAt)
		if err != nil {
			log.Printf("Error scanning audit log row: %v", err)
			continue
		}
		logs = append(logs, al)
	}
	return logs, total, nil
}

// ==================== ADAPTERS ====================

func (r *postgresRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	return r.CreateAuditLog(ctx, log)
}

func (r *postgresRepository) GetAll(ctx context.Context, limit, offset int, userID *int, search string, action string) ([]domain.AuditLog, int, error) {
	return r.GetAuditLogs(ctx, limit, offset, userID, search, action)
}

// GetPeriodComparison fetches and calculates period comparison data
func (r *postgresRepository) GetPeriodComparison(
	ctx context.Context,
	currentStart, currentEnd time.Time,
	previousStart, previousEnd time.Time,
) (*domain.PeriodComparison, error) {

	query := `
		WITH current_period AS (
			SELECT
				COALESCE(SUM(total_amount), 0) as revenue,
				COUNT(*) as orders
			FROM sales
			WHERE created_at >= $1 AND created_at < $2
				AND status = 'completed'
		),
		previous_period AS (
			SELECT
				COALESCE(SUM(total_amount), 0) as revenue,
				COUNT(*) as orders
			FROM sales
			WHERE created_at >= $3 AND created_at < $4
				AND status = 'completed'
		)
		SELECT
			cp.revenue, cp.orders,
			pp.revenue, pp.orders
		FROM current_period cp, previous_period pp`

	var result domain.PeriodComparison
	err := r.db.QueryRow(ctx, query,
		currentStart, currentEnd,
		previousStart, previousEnd,
	).Scan(
		&result.CurrentRevenue,
		&result.CurrentOrders,
		&result.PreviousRevenue,
		&result.PreviousOrders,
	)

	if err != nil {
		return nil, err
	}

	// Calculate derived metrics
	days := int(currentEnd.Sub(currentStart).Hours() / 24)
	if days == 0 {
		days = 1
	}

	if result.CurrentOrders > 0 {
		result.CurrentAOV = result.CurrentRevenue / result.CurrentOrders
	}
	if result.PreviousOrders > 0 {
		result.PreviousAOV = result.PreviousRevenue / result.PreviousOrders
	}

	result.RevenuePerDay = result.CurrentRevenue / days
	result.PreviousRevenuePerDay = result.PreviousRevenue / days

	return &result, nil
}

// ==================== PHASE 1 EXTENSIONS: Brands, Tax, UOM, Warehouses ====================

// GetNextSKU generates the next SKU using sequence
func (r *postgresRepository) GetNextSKU(ctx context.Context) (string, error) {
	var skuNum int
	err := r.db.QueryRow(ctx, "SELECT nextval('sku_seq')").Scan(&skuNum)
	if err != nil {
		return "", fmt.Errorf("failed to get next SKU: %w", err)
	}
	return fmt.Sprintf("SKU-%06d", skuNum), nil
}

// Brand operations
func (r *postgresRepository) GetBrandByID(ctx context.Context, id int) (*domain.Brand, error) {
	var brand domain.Brand
	var createdAt, updatedAt time.Time
	err := r.db.QueryRow(ctx, "SELECT id, name, description, is_active, created_at, updated_at FROM brands WHERE id = $1", id).Scan(
		&brand.ID, &brand.Name, &brand.Description, &brand.IsActive, &createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("brand not found")
		}
		return nil, err
	}
	brand.CreatedAt = createdAt.Format(time.RFC3339)
	brand.UpdatedAt = updatedAt.Format(time.RFC3339)
	return &brand, nil
}

func (r *postgresRepository) GetAllBrands(ctx context.Context) ([]domain.Brand, error) {
	rows, err := r.db.Query(ctx, "SELECT id, name, description, is_active, created_at, updated_at FROM brands WHERE is_active = true ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var brands []domain.Brand
	for rows.Next() {
		var b domain.Brand
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&b.ID, &b.Name, &b.Description, &b.IsActive, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		b.CreatedAt = createdAt.Format(time.RFC3339)
		b.UpdatedAt = updatedAt.Format(time.RFC3339)
		brands = append(brands, b)
	}
	return brands, nil
}

func (r *postgresRepository) CreateBrand(ctx context.Context, brand *domain.Brand) error {
	var createdAt, updatedAt time.Time
	return r.db.QueryRow(ctx, `
		INSERT INTO brands (name, description, is_active) 
		VALUES ($1, $2, $3) 
		RETURNING id, created_at, updated_at
	`, brand.Name, brand.Description, brand.IsActive).Scan(&brand.ID, &createdAt, &updatedAt)
}

func (r *postgresRepository) UpdateBrand(ctx context.Context, brand *domain.Brand) error {
	_, err := r.db.Exec(ctx, `
		UPDATE brands SET name = $1, description = $2, is_active = $3, updated_at = NOW() 
		WHERE id = $4
	`, brand.Name, brand.Description, brand.IsActive, brand.ID)
	return err
}

func (r *postgresRepository) DeleteBrand(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, "DELETE FROM brands WHERE id = $1", id)
	return err
}

func (r *postgresRepository) GetBrandIDByName(ctx context.Context, name string) (int, error) {
	var id int
	err := r.db.QueryRow(ctx, "SELECT id FROM brands WHERE name = $1 AND is_active = true", name).Scan(&id)
	return id, err
}

// Tax class operations
func (r *postgresRepository) GetTaxClassByID(ctx context.Context, id int) (*domain.TaxClass, error) {
	var tc domain.TaxClass
	var createdAt time.Time
	err := r.db.QueryRow(ctx, "SELECT id, name, rate_percent, description, is_active, created_at FROM tax_classes WHERE id = $1", id).Scan(
		&tc.ID, &tc.Name, &tc.RatePercent, &tc.Description, &tc.IsActive, &createdAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("tax class not found")
		}
		return nil, err
	}
	tc.CreatedAt = createdAt.Format(time.RFC3339)
	return &tc, nil
}

func (r *postgresRepository) GetAllTaxClasses(ctx context.Context) ([]domain.TaxClass, error) {
	rows, err := r.db.Query(ctx, "SELECT id, name, rate_percent, description, is_active, created_at FROM tax_classes WHERE is_active = true ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var taxClasses []domain.TaxClass
	for rows.Next() {
		var tc domain.TaxClass
		var createdAt time.Time
		if err := rows.Scan(&tc.ID, &tc.Name, &tc.RatePercent, &tc.Description, &tc.IsActive, &createdAt); err != nil {
			return nil, err
		}
		tc.CreatedAt = createdAt.Format(time.RFC3339)
		taxClasses = append(taxClasses, tc)
	}
	return taxClasses, nil
}

func (r *postgresRepository) GetTaxClassIDByName(ctx context.Context, name string) (int, error) {
	var id int
	err := r.db.QueryRow(ctx, "SELECT id FROM tax_classes WHERE name = $1 AND is_active = true", name).Scan(&id)
	return id, err
}

// Unit of measure operations
func (r *postgresRepository) GetUnitOfMeasureByID(ctx context.Context, id int) (*domain.UnitOfMeasure, error) {
	var uom domain.UnitOfMeasure
	var createdAt time.Time
	err := r.db.QueryRow(ctx, "SELECT id, code, name, description, is_active, created_at FROM units_of_measure WHERE id = $1", id).Scan(
		&uom.ID, &uom.Code, &uom.Name, &uom.Description, &uom.IsActive, &createdAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("unit of measure not found")
		}
		return nil, err
	}
	uom.CreatedAt = createdAt.Format(time.RFC3339)
	return &uom, nil
}

func (r *postgresRepository) GetAllUnitsOfMeasure(ctx context.Context) ([]domain.UnitOfMeasure, error) {
	rows, err := r.db.Query(ctx, "SELECT id, code, name, description, is_active, created_at FROM units_of_measure WHERE is_active = true ORDER BY code")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var uoms []domain.UnitOfMeasure
	for rows.Next() {
		var uom domain.UnitOfMeasure
		var createdAt time.Time
		if err := rows.Scan(&uom.ID, &uom.Code, &uom.Name, &uom.Description, &uom.IsActive, &createdAt); err != nil {
			return nil, err
		}
		uom.CreatedAt = createdAt.Format(time.RFC3339)
		uoms = append(uoms, uom)
	}
	return uoms, nil
}

func (r *postgresRepository) GetUnitOfMeasureIDByCode(ctx context.Context, code string) (int, error) {
	var id int
	err := r.db.QueryRow(ctx, "SELECT id FROM units_of_measure WHERE code = $1 AND is_active = true", code).Scan(&id)
	return id, err
}

// Warehouse operations
func (r *postgresRepository) GetWarehouseByID(ctx context.Context, id int) (*domain.Warehouse, error) {
	var w domain.Warehouse
	var storeID sql.NullInt64
	var createdAt time.Time
	err := r.db.QueryRow(ctx, "SELECT id, name, code, address, store_id, is_active, created_at FROM warehouses WHERE id = $1", id).Scan(
		&w.ID, &w.Name, &w.Code, &w.Address, &storeID, &w.IsActive, &createdAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("warehouse not found")
		}
		return nil, err
	}
	if storeID.Valid {
		v := int(storeID.Int64)
		w.StoreID = &v
	}
	w.CreatedAt = createdAt.Format(time.RFC3339)
	return &w, nil
}

func (r *postgresRepository) GetAllWarehouses(ctx context.Context, storeID *int) ([]domain.Warehouse, error) {
	query := "SELECT id, name, code, address, store_id, is_active, created_at FROM warehouses WHERE is_active = true"
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

	var warehouses []domain.Warehouse
	for rows.Next() {
		var w domain.Warehouse
		var storeIDVal sql.NullInt64
		var createdAt time.Time
		if err := rows.Scan(&w.ID, &w.Name, &w.Code, &w.Address, &storeIDVal, &w.IsActive, &createdAt); err != nil {
			return nil, err
		}
		if storeIDVal.Valid {
			v := int(storeIDVal.Int64)
			w.StoreID = &v
		}
		w.CreatedAt = createdAt.Format(time.RFC3339)
		warehouses = append(warehouses, w)
	}
	return warehouses, nil
}
