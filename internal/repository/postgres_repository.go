package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"retail-pos-system/internal/config"
	"retail-pos-system/internal/domain"

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
	var lastLogin sql.NullTime

	err := r.db.QueryRow(ctx, `
		SELECT id, username, email, password_hash, role_id, store_id, is_active, created_at, updated_at, last_login
		FROM users WHERE username = $1 AND deleted_at IS NULL
	`, username).Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.RoleID, &storeID, &u.IsActive, &createdAt, &updatedAt, &lastLogin)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}
	u.CreatedAt = createdAt.Format(time.RFC3339)
	u.UpdatedAt = updatedAt.Format(time.RFC3339)
	if lastLogin.Valid {
		u.LastLogin = lastLogin.Time.Format(time.RFC3339)
	}
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
	var lastLogin sql.NullTime

	err := r.db.QueryRow(ctx, `
		SELECT id, username, email, password_hash, role_id, store_id, is_active, created_at, updated_at, last_login
		FROM users WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.RoleID, &storeID, &u.IsActive, &createdAt, &updatedAt, &lastLogin)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	u.CreatedAt = createdAt.Format(time.RFC3339)
	u.UpdatedAt = updatedAt.Format(time.RFC3339)
	if lastLogin.Valid {
		u.LastLogin = lastLogin.Time.Format(time.RFC3339)
	}
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

func (r *postgresRepository) GetAllUsers(ctx context.Context, limit, offset int, search string, sortBy string, sortDir string, roleID int, isActive *bool) ([]domain.User, int, error) {
	var users []domain.User
	var total int

	validSortColumns := map[string]bool{
		"id": true, "username": true, "email": true, "role_id": true,
		"is_active": true, "created_at": true, "last_login": true, "updated_at": true,
	}
	if !validSortColumns[sortBy] {
		sortBy = "id"
	}
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "desc"
	}

	query := `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`
	args := []interface{}{}
	argIdx := 1
	if search != "" {
		query += fmt.Sprintf(" AND (username ILIKE $%d OR email ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}
	if roleID > 0 {
		query += fmt.Sprintf(" AND role_id = $%d", argIdx)
		args = append(args, roleID)
		argIdx++
	}
	if isActive != nil {
		query += fmt.Sprintf(" AND is_active = $%d", argIdx)
		args = append(args, *isActive)
	}

	err := r.db.QueryRow(ctx, query, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	query = `SELECT id, username, email, password_hash, role_id, store_id, is_active, created_at, updated_at, last_login FROM users WHERE deleted_at IS NULL`
	args2 := []interface{}{}
	argIdx2 := 1
	if search != "" {
		query += fmt.Sprintf(" AND (username ILIKE $%d OR email ILIKE $%d)", argIdx2, argIdx2)
		args2 = append(args2, "%"+search+"%")
		argIdx2++
	}
	if roleID > 0 {
		query += fmt.Sprintf(" AND role_id = $%d", argIdx2)
		args2 = append(args2, roleID)
		argIdx2++
	}
	if isActive != nil {
		query += fmt.Sprintf(" AND is_active = $%d", argIdx2)
		args2 = append(args2, *isActive)
		argIdx2++
	}
	textSortColumns := map[string]bool{"username": true, "email": true}
	sortExpr := sortBy
	if textSortColumns[sortBy] {
		sortExpr = "LOWER(" + sortBy + ")"
	}
	query += fmt.Sprintf(" ORDER BY %s %s LIMIT $%d OFFSET $%d", sortExpr, sortDir, argIdx2, argIdx2+1)
	args2 = append(args2, limit, offset)

	rows, err := r.db.Query(ctx, query, args2...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var u domain.User
		var storeID sql.NullInt64
		var createdAt, updatedAt time.Time
		var lastLogin sql.NullTime

		err = rows.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.RoleID, &storeID, &u.IsActive, &createdAt, &updatedAt, &lastLogin)
		if err != nil {
			continue
		}
		u.CreatedAt = createdAt.Format(time.RFC3339)
		u.UpdatedAt = updatedAt.Format(time.RFC3339)
		if lastLogin.Valid {
			u.LastLogin = lastLogin.Time.Format(time.RFC3339)
		}
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
	var createdAt, updatedAt time.Time
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, role_id, store_id, is_active)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at, updated_at
	`, user.Username, user.Email, user.Password, user.RoleID, user.StoreID, user.IsActive).Scan(&user.ID, &createdAt, &updatedAt)
	if err != nil {
		return err
	}
	user.CreatedAt = createdAt.Format(time.RFC3339)
	user.UpdatedAt = updatedAt.Format(time.RFC3339)
	return nil
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

func (r *postgresRepository) UpdateLastLogin(ctx context.Context, userID int) error {
	_, err := r.db.Exec(ctx, "UPDATE users SET last_login = NOW() WHERE id = $1", userID)
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
  role.CreatedAt = createdAt
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
 		rl.CreatedAt = createdAt
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
	var categoryIDVal, storeIDVal, brandIDVal, unitOfMeasureIDVal, weightGramsVal sql.NullInt64
	var categoryName, brandName, unitOfMeasure, description sql.NullString
	var createdAt, updatedAt time.Time

	query := `
		SELECT v.id, v.sku, v.name, v.barcode, v.category_id, v.category_name, v.price, v.cost, v.stock, v.status,
		       v.store_id, v.brand_id, v.brand_name, v.unit_of_measure_id, v.unit_of_measure, v.weight_grams, v.description,
		       v.created_at, v.updated_at
		FROM v_products_full v
		WHERE v.id = $1`

	args := []interface{}{id}
	if storeID != nil {
		query += fmt.Sprintf(" AND v.store_id = $%d", len(args)+1)
		args = append(args, *storeID)
	}

	err := r.db.QueryRow(ctx, query, args...).Scan(&p.ID, &p.SKU, &p.Name, &barcode, &categoryIDVal, &categoryName, &p.Price, &p.Cost, &p.Stock, &p.Status,
		&storeIDVal, &brandIDVal, &brandName, &unitOfMeasureIDVal, &unitOfMeasure, &weightGramsVal, &description,
		&createdAt, &updatedAt)
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
	if brandIDVal.Valid {
		v := int(brandIDVal.Int64)
		p.BrandID = &v
	}
	if brandName.Valid {
		p.BrandName = &brandName.String
	}
	if unitOfMeasureIDVal.Valid {
		v := int(unitOfMeasureIDVal.Int64)
		p.UnitOfMeasureID = &v
	}
	if unitOfMeasure.Valid {
		p.UnitOfMeasure = &unitOfMeasure.String
	}
	if weightGramsVal.Valid {
		v := int(weightGramsVal.Int64)
		p.WeightGrams = &v
	}
	if description.Valid {
		p.Description = &description.String
	}
	if storeIDVal.Valid {
		v := int(storeIDVal.Int64)
		p.StoreID = &v
	}
	p.CreatedAt = createdAt.In(jakartaLoc).Format(time.RFC3339)
	p.UpdatedAt = updatedAt.In(jakartaLoc).Format(time.RFC3339)

	return &p, nil
}

func (r *postgresRepository) GetProductBySKU(ctx context.Context, sku string, storeID *int) (*domain.Product, error) {
	var p domain.Product
	var barcode sql.NullString
	var categoryIDVal, storeIDVal, brandIDVal, unitOfMeasureIDVal, weightGramsVal sql.NullInt64
	var categoryName, brandName, unitOfMeasure, description sql.NullString
	var createdAt, updatedAt time.Time

	query := `
		SELECT v.id, v.sku, v.name, v.barcode, v.category_id, v.category_name, v.price, v.cost, v.stock, v.status,
		       v.store_id, v.brand_id, v.brand_name, v.unit_of_measure_id, v.unit_of_measure, v.weight_grams, v.description,
		       v.created_at, v.updated_at
		FROM v_products_full v
		WHERE v.sku = $1`

	args := []interface{}{sku}
	if storeID != nil {
		query += fmt.Sprintf(" AND p.store_id = $%d", len(args)+1)
		args = append(args, *storeID)
	}

	err := r.db.QueryRow(ctx, query, args...).Scan(&p.ID, &p.SKU, &p.Name, &barcode, &categoryIDVal, &categoryName, &p.Price, &p.Cost, &p.Stock, &p.Status,
		&storeIDVal, &brandIDVal, &brandName, &unitOfMeasureIDVal, &unitOfMeasure, &weightGramsVal, &description,
		&createdAt, &updatedAt)
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
	if brandIDVal.Valid {
		v := int(brandIDVal.Int64)
		p.BrandID = &v
	}
	if brandName.Valid {
		p.BrandName = &brandName.String
	}
	if unitOfMeasureIDVal.Valid {
		v := int(unitOfMeasureIDVal.Int64)
		p.UnitOfMeasureID = &v
	}
	if unitOfMeasure.Valid {
		p.UnitOfMeasure = &unitOfMeasure.String
	}
	if weightGramsVal.Valid {
		v := int(weightGramsVal.Int64)
		p.WeightGrams = &v
	}
	if description.Valid {
		p.Description = &description.String
	}
	if storeIDVal.Valid {
		v := int(storeIDVal.Int64)
		p.StoreID = &v
	}
	p.CreatedAt = createdAt.In(jakartaLoc).Format(time.RFC3339)
	p.UpdatedAt = updatedAt.In(jakartaLoc).Format(time.RFC3339)

	return &p, nil
}

func (r *postgresRepository) GetAllProducts(ctx context.Context, limit, offset int, search string, categoryIDs []int, sortBy, sortDir string, maxStock *int, storeID *int, status string) ([]domain.Product, int, error) {
	var products []domain.Product
	var total int

	query := `SELECT COUNT(*) 
		FROM v_products_full v 
		WHERE 1=1`
	args := []interface{}{}
	argIdx := 1
	if search != "" {
		query += fmt.Sprintf(" AND (v.name ILIKE $%d OR v.sku ILIKE $%d OR v.barcode ILIKE $%d)", argIdx, argIdx, argIdx)
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
		query += fmt.Sprintf(" AND v.category_id IN (%s)", strings.Join(placeholders, ","))
	}
	if maxStock != nil {
		query += fmt.Sprintf(" AND v.stock <= $%d", argIdx)
		args = append(args, *maxStock)
		argIdx++
	}
	if storeID != nil {
		query += fmt.Sprintf(" AND v.store_id = $%d", argIdx)
		args = append(args, *storeID)
		argIdx++
	}
	if status != "" {
		query += fmt.Sprintf(" AND v.status = $%d", argIdx)
		args = append(args, status)
	}

	err := r.db.QueryRow(ctx, query, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count products: %w", err)
	}

	query2 := `SELECT v.id, v.sku, v.name, v.barcode, v.category_id, v.category_name, v.price, v.cost, v.stock, v.status, v.store_id, 
		       v.brand_id, v.brand_name, v.unit_of_measure_id, v.unit_of_measure, v.weight_grams, v.description,
		       v.created_at, v.updated_at 
		FROM v_products_full v 
		WHERE 1=1`
	args2 := []interface{}{}
	argIdx2 := 1
	if search != "" {
		query2 += fmt.Sprintf(" AND (v.name ILIKE $%d OR v.sku ILIKE $%d OR v.barcode ILIKE $%d)", argIdx2, argIdx2, argIdx2)
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
		query2 += fmt.Sprintf(" AND v.category_id IN (%s)", strings.Join(placeholders, ","))
	}
	if maxStock != nil {
		query2 += fmt.Sprintf(" AND v.stock <= $%d", argIdx2)
		args2 = append(args2, *maxStock)
		argIdx2++
	}
	if storeID != nil {
		query2 += fmt.Sprintf(" AND v.store_id = $%d", argIdx2)
		args2 = append(args2, *storeID)
		argIdx2++
	}
	if status != "" {
		query2 += fmt.Sprintf(" AND v.status = $%d", argIdx2)
		args2 = append(args2, status)
		argIdx2++
	}
	if sortBy != "" {
		query2 += fmt.Sprintf(" ORDER BY %s", sortBy)
		if sortDir != "" {
			query2 += " " + sortDir
		}
	} else {
		query2 += " ORDER BY v.id DESC"
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
		var categoryIDVal, storeIDVal, brandIDVal, unitOfMeasureIDVal, weightGramsVal sql.NullInt64
		var categoryName, brandName, unitOfMeasure, descriptionVal sql.NullString
		var createdAt, updatedAt time.Time

		err = rows.Scan(&p.ID, &p.SKU, &p.Name, &barcodeVal, &categoryIDVal, &categoryName, &p.Price, &p.Cost, &p.Stock, &p.Status,
			&storeIDVal, &brandIDVal, &brandName, &unitOfMeasureIDVal, &unitOfMeasure, &weightGramsVal, &descriptionVal,
			&createdAt, &updatedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan product: %w", err)
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
		if brandIDVal.Valid {
			v := int(brandIDVal.Int64)
			p.BrandID = &v
		}
		if brandName.Valid {
			p.BrandName = &brandName.String
		}
		if unitOfMeasureIDVal.Valid {
			v := int(unitOfMeasureIDVal.Int64)
			p.UnitOfMeasureID = &v
		}
		if unitOfMeasure.Valid {
			p.UnitOfMeasure = &unitOfMeasure.String
		}
		if weightGramsVal.Valid {
			v := int(weightGramsVal.Int64)
			p.WeightGrams = &v
		}
		if descriptionVal.Valid {
			p.Description = &descriptionVal.String
		}
		if storeIDVal.Valid {
			v := int(storeIDVal.Int64)
			p.StoreID = &v
		}
		p.CreatedAt = createdAt.In(jakartaLoc).Format(time.RFC3339)
		p.UpdatedAt = updatedAt.In(jakartaLoc).Format(time.RFC3339)
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
	var weightGrams, defaultDiscount, description interface{}
	
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
	if product.Description != nil {
		description = *product.Description
	}

	err := r.db.QueryRow(ctx, `
		INSERT INTO products (sku, name, barcode, category_id, price, cost, stock, store_id, status,
		                    brand_id, description, tax_class_id, weight_grams, unit_of_measure_id, default_discount_percent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_at, updated_at
	`, product.SKU, product.Name, barcode, categoryID, product.Price, product.Cost, product.Stock, storeIDVal, product.Status,
		brandID, description, taxClassID, weightGrams, unitOfMeasureID, defaultDiscount).
		Scan(&product.ID, &product.CreatedAt, &product.UpdatedAt)
	if err != nil {
		return err
	}

	if storeIDVal != nil {
		_, err = r.db.Exec(ctx, `
			INSERT INTO product_stock (product_id, store_id, quantity) VALUES ($1, $2, $3)
		`, product.ID, storeIDVal, product.Stock)
		if err != nil {
			return fmt.Errorf("failed to initialize product stock: %w", err)
		}
	}
	_, err = r.db.Exec(ctx, `
		UPDATE products SET stock = $1 WHERE id = $2
	`, product.Stock, product.ID)
	if err != nil {
		return fmt.Errorf("failed to sync product stock: %w", err)
	}
	return nil
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
	var weightGrams, defaultDiscount, description interface{}
	
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
	if product.Description != nil {
		description = *product.Description
	} else {
		description = nil
	}

	_, err := r.db.Exec(ctx, `
		UPDATE products SET sku = $1, name = $2, barcode = $3, category_id = $4, price = $5,
			cost = $6, stock = $7, store_id = $8, status = $9, updated_at = NOW(),
			brand_id = $10, description = $11, tax_class_id = $12, weight_grams = $13, unit_of_measure_id = $14, default_discount_percent = $15
		WHERE id = $16
	`, product.SKU, product.Name, barcode, categoryID, product.Price, product.Cost, product.Stock, storeIDVal, product.Status,
		brandID, description, taxClassID, weightGrams, unitOfMeasureID, defaultDiscount, product.ID)
	if err != nil {
		return err
	}

	if storeIDVal != nil {
		_, err = r.db.Exec(ctx, `
			INSERT INTO product_stock (product_id, store_id, quantity) VALUES ($1, $2, $3)
		`, product.ID, storeIDVal, product.Stock)
		if err != nil {
			return fmt.Errorf("failed to sync product stock: %w", err)
		}
	}
	_, err = r.db.Exec(ctx, `
		UPDATE products SET stock = $1 WHERE id = $2
	`, product.Stock, product.ID)
	if err != nil {
		return fmt.Errorf("failed to sync product stock: %w", err)
	}

	return nil
}

func (r *postgresRepository) DeleteProduct(ctx context.Context, id int, storeID *int) error {
	_, err := r.db.Exec(ctx, "UPDATE products SET deleted_at = NOW(), status = 'archived' WHERE id = $1", id)
	return err
}

func (r *postgresRepository) BulkUpdateProductStatus(ctx context.Context, ids []int, status string, storeID *int) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.db.Exec(ctx, "UPDATE products SET status = $1 WHERE id = ANY($2)", status, ids)
	return err
}

// ==================== CATEGORY ====================

func (r *postgresRepository) ListCategories(ctx context.Context) ([]domain.Category, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, COALESCE(slug,''), COALESCE(description,''), is_active, created_at
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
		err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &c.IsActive, &createdAt)
		if err != nil {
			return nil, err
		}
		c.CreatedAt = createdAt.In(jakartaLoc).Format(time.RFC3339)
		c.UpdatedAt = ""
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

// GetAllCategories returns paginated categories with product count (for management page)
func (r *postgresRepository) GetAllCategories(ctx context.Context, limit, offset int, search string) ([]domain.Category, int, error) {
	// COUNT query
	countQuery := `SELECT COUNT(*) FROM categories WHERE 1=1`
	args := []interface{}{}
	argIdx := 1
	if search != "" {
		countQuery += fmt.Sprintf(" AND (name ILIKE $%d OR slug ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+search+"%")
	}
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count categories: %w", err)
	}

	// DATA query — LEFT JOIN + GROUP BY (optimized)
	query := `SELECT c.id, c.name, COALESCE(c.slug, ''), COALESCE(c.description, ''), c.is_active,
			  COUNT(p.id) AS product_count,
			  c.created_at, COALESCE(c.updated_at, c.created_at)
			  FROM categories c
			  LEFT JOIN products p ON p.category_id = c.id AND p.deleted_at IS NULL
			  WHERE 1=1`
	args2 := []interface{}{}
	argIdx2 := 1
	if search != "" {
		query += fmt.Sprintf(" AND (c.name ILIKE $%d OR c.slug ILIKE $%d)", argIdx2, argIdx2)
		args2 = append(args2, "%"+search+"%")
		argIdx2++
	}
	query += " GROUP BY c.id"
	query += fmt.Sprintf(" ORDER BY c.name ASC LIMIT $%d OFFSET $%d", argIdx2, argIdx2+1)
	args2 = append(args2, limit, offset)

	rows, err := r.db.Query(ctx, query, args2...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query categories: %w", err)
	}
	defer rows.Close()

	var categories []domain.Category
	for rows.Next() {
		var c domain.Category
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &c.IsActive, &c.ProductCount, &createdAt, &updatedAt); err != nil {
			continue
		}
		c.CreatedAt = createdAt.In(jakartaLoc).Format(time.RFC3339)
		c.UpdatedAt = updatedAt.In(jakartaLoc).Format(time.RFC3339)
		categories = append(categories, c)
	}
	return categories, total, nil
}

// GetCategoryByID returns a category by ID
func (r *postgresRepository) GetCategoryByID(ctx context.Context, id int) (*domain.Category, error) {
	var c domain.Category
	var createdAt, updatedAt time.Time
	err := r.db.QueryRow(ctx, `
		SELECT id, name, COALESCE(slug, ''), COALESCE(description, ''), is_active,
		       created_at, COALESCE(updated_at, created_at)
		FROM categories WHERE id = $1
	`, id).Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &c.IsActive, &createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("category not found")
		}
		return nil, err
	}
	c.CreatedAt = createdAt.In(jakartaLoc).Format(time.RFC3339)
	c.UpdatedAt = updatedAt.In(jakartaLoc).Format(time.RFC3339)
	return &c, nil
}

// SlugExists checks if a slug exists, optionally excluding an ID for updates
func (r *postgresRepository) SlugExists(ctx context.Context, slug string, excludeID int) (bool, error) {
	var exists bool
	query := `SELECT EXISTS (SELECT 1 FROM categories WHERE slug = $1 AND id != $2)`
	err := r.db.QueryRow(ctx, query, slug, excludeID).Scan(&exists)
	return exists, err
}

// CreateCategory creates a category with auto-generated slug
func (r *postgresRepository) CreateCategory(ctx context.Context, category *domain.Category) error {
	if category.Slug == "" {
		category.Slug = generateSlug(category.Name)
	}

	// Resolve slug collision: append -2, -3, etc.
	baseSlug := category.Slug
	suffix := 1
	for {
		exists, err := r.SlugExists(ctx, category.Slug, 0)
		if err != nil {
			return fmt.Errorf("failed to check slug uniqueness: %w", err)
		}
		if !exists {
			break
		}
		suffix++
		category.Slug = fmt.Sprintf("%s-%d", baseSlug, suffix)
		if len(category.Slug) > 120 {
		 truncLen := 120 - len(fmt.Sprintf("-%d", suffix))
			if truncLen > 0 && len(baseSlug) >= truncLen {
				category.Slug = fmt.Sprintf("%s-%d", baseSlug[:truncLen], suffix)
			} else {
				break
			}
		}
	}

	var createdAt, updatedAt time.Time
	err := r.db.QueryRow(ctx, `
		INSERT INTO categories (name, slug, description, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`, category.Name, category.Slug, category.Description, category.IsActive).Scan(&category.ID, &createdAt, &updatedAt)
	if err != nil {
		return err
	}
	category.CreatedAt = createdAt.In(jakartaLoc).Format(time.RFC3339)
	category.UpdatedAt = updatedAt.In(jakartaLoc).Format(time.RFC3339)
	return nil
}

// UpdateCategory updates a category
func (r *postgresRepository) UpdateCategory(ctx context.Context, category *domain.Category) error {
	// Regenerate slug if name changed
	newSlug := generateSlug(category.Name)
	if newSlug != category.Slug {
		// Check for collision (excluding current ID)
		exists, err := r.SlugExists(ctx, newSlug, category.ID)
		if err != nil {
			return fmt.Errorf("failed to check slug uniqueness: %w", err)
		}
		if exists {
			suffix := 2
			for {
				candidate := fmt.Sprintf("%s-%d", newSlug, suffix)
				ex, err := r.SlugExists(ctx, candidate, category.ID)
				if err != nil {
					return err
				}
				if !ex {
					newSlug = candidate
					break
				}
				suffix++
			}
		}
		category.Slug = newSlug
	}

	_, err := r.db.Exec(ctx, `
		UPDATE categories SET name = $1, slug = $2, description = $3, is_active = $4, updated_at = NOW()
		WHERE id = $5
	`, category.Name, category.Slug, category.Description, category.IsActive, category.ID)
	return err
}

// DeleteCategory deletes a category (FK RESTRICT handles race condition)
func (r *postgresRepository) DeleteCategory(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, "DELETE FROM categories WHERE id = $1", id)
	return err
}

// HasActiveProducts checks if category has products using EXISTS (early exit)
func (r *postgresRepository) HasActiveProducts(ctx context.Context, categoryID int) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM products
			WHERE category_id = $1 AND deleted_at IS NULL
			LIMIT 1
		)
	`, categoryID).Scan(&exists)
	return exists, err
}

// generateSlug creates a URL-friendly slug from a name
func generateSlug(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	replacements := []struct{ from, to string }{
		{" ", "-"}, {"'", ""}, {`"`, ""}, {"&", "and"}, {"/", "-"},
		{"+", "plus"}, {"=", "equals"}, {"?", ""}, {"!", ""}, {"@", "at"},
		{"#", "number"}, {"%", "percent"}, {"(", ""}, {")", ""},
	}
	for _, r := range replacements {
		slug = strings.ReplaceAll(slug, r.from, r.to)
	}
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")
	if len(slug) > 120 {
		slug = slug[:120]
	}
	return slug
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
	if err != nil {
		return err
	}

	if storeIDVal != nil {
		_, err = r.db.Exec(ctx, `
			INSERT INTO product_stock (product_id, store_id, quantity) VALUES ($1, $2, $3)
		`, product.ID, storeIDVal, product.Stock)
		if err != nil {
			return fmt.Errorf("failed to restore product stock: %w", err)
		}
	}
	_, err = r.db.Exec(ctx, `
		UPDATE products SET stock = $1 WHERE id = $2
	`, product.Stock, product.ID)
	if err != nil {
		return fmt.Errorf("failed to sync product stock: %w", err)
	}

	return nil
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

func (r *postgresRepository) GetSaleByID(ctx context.Context, id int) (*domain.Sale, error) {
	var sale domain.Sale
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

func (r *postgresRepository) GetAllSales(ctx context.Context, limit, offset int, search string, sortBy, sortDir, startDate, endDate string, storeID *int, paymentMethods string, minTotal, maxTotal *int) ([]domain.Sale, int, error) {
	var sales []domain.Sale
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
		var s domain.Sale
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
					var item domain.SaleItem
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

func (r *postgresRepository) GetSalesForExport(ctx context.Context, search, startDate, endDate string, paymentMethods string, minTotal, maxTotal *int) ([]domain.SaleExportRow, error) {
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

	var result []domain.SaleExportRow
	for rows.Next() {
		var row domain.SaleExportRow
		var createdAt time.Time
		if err := rows.Scan(&row.InvoiceNumber, &createdAt, &row.CustomerName, &row.ItemCount, &row.PaymentMethod, &row.TotalAmount); err != nil {
			continue
		}
		row.CreatedAt = createdAt.Format(time.RFC3339)
		result = append(result, row)
	}
	return result, nil
}

// ==================== AUDIT ====================

func (r *postgresRepository) CreateAuditLog(ctx context.Context, log *domain.AuditLog) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO audit_logs (user_id, role, action, entity_type, entity_id, ip_address, user_agent, old_values, new_values)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, log.UserID, log.Role, log.Action, log.EntityType, log.EntityID, log.IPAddress, log.UserAgent, log.OldValues, log.NewValues)
	return err
}

func (r *postgresRepository) GetAuditLogs(ctx context.Context, limit, offset int, userID *int, search string, action string, entityType string, startDate *time.Time, endDate *time.Time) ([]domain.AuditLog, int, error) {
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
		query += fmt.Sprintf(" AND (u.username ILIKE $%d OR al.role ILIKE $%d OR al.action ILIKE $%d OR al.entity_type ILIKE $%d OR al.ip_address::text ILIKE $%d)", len(args)+1, len(args)+2, len(args)+3, len(args)+4, len(args)+5)
		args = append(args, "%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	if entityType != "" {
		query += fmt.Sprintf(" AND al.entity_type = $%d", len(args)+1)
		args = append(args, entityType)
	}
	if startDate != nil {
		query += fmt.Sprintf(" AND al.created_at >= $%d", len(args)+1)
		args = append(args, *startDate)
	}
	if endDate != nil {
		query += fmt.Sprintf(" AND al.created_at < $%d", len(args)+1)
		args = append(args, endDate.Add(24*time.Hour))
	}

	err := r.db.QueryRow(ctx, query, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query = `SELECT al.id, al.user_id, COALESCE(u.username, 'Unknown'), COALESCE(al.role, ''), al.action, al.entity_type, al.entity_id, COALESCE(al.ip_address::text, ''), COALESCE(al.user_agent, ''), COALESCE(al.old_values, '{}'::jsonb), COALESCE(al.new_values, '{}'::jsonb), al.created_at::text FROM audit_logs al LEFT JOIN users u ON al.user_id = u.id WHERE 1=1`
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
		query += fmt.Sprintf(" AND (u.username ILIKE $%d OR al.role ILIKE $%d OR al.action ILIKE $%d OR al.entity_type ILIKE $%d OR al.ip_address::text ILIKE $%d)", len(args2)+1, len(args2)+2, len(args2)+3, len(args2)+4, len(args2)+5)
		args2 = append(args2, "%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	if entityType != "" {
		query += fmt.Sprintf(" AND al.entity_type = $%d", len(args2)+1)
		args2 = append(args2, entityType)
	}
	if startDate != nil {
		query += fmt.Sprintf(" AND al.created_at >= $%d", len(args2)+1)
		args2 = append(args2, *startDate)
	}
	if endDate != nil {
		query += fmt.Sprintf(" AND al.created_at < $%d", len(args2)+1)
		args2 = append(args2, endDate.Add(24*time.Hour))
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

func (r *postgresRepository) GetAll(ctx context.Context, limit, offset int, userID *int, search string, action string, entityType string, startDate *time.Time, endDate *time.Time) ([]domain.AuditLog, int, error) {
	return r.GetAuditLogs(ctx, limit, offset, userID, search, action, entityType, startDate, endDate)
}

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
		),
		current_peak_hour AS (
			SELECT COALESCE(MAX(hourly_total), 0) as peak_revenue
			FROM (
				SELECT SUM(total_amount) as hourly_total
				FROM sales
				WHERE created_at >= $1 AND created_at < $2
					AND status = 'completed'
				GROUP BY EXTRACT(HOUR FROM (created_at AT TIME ZONE 'Asia/Jakarta'))
			) hourly
		),
		previous_peak_hour AS (
			SELECT COALESCE(MAX(hourly_total), 0) as peak_revenue
			FROM (
				SELECT SUM(total_amount) as hourly_total
				FROM sales
				WHERE created_at >= $3 AND created_at < $4
					AND status = 'completed'
				GROUP BY EXTRACT(HOUR FROM (created_at AT TIME ZONE 'Asia/Jakarta'))
			) hourly
		),
		current_peak_month AS (
			SELECT COALESCE(MAX(monthly_total), 0) as peak_revenue
			FROM (
				SELECT SUM(total_amount) as monthly_total
				FROM sales
				WHERE created_at >= $1 AND created_at < $2
					AND status = 'completed'
				GROUP BY EXTRACT(YEAR FROM (created_at AT TIME ZONE 'Asia/Jakarta')),
				         EXTRACT(MONTH FROM (created_at AT TIME ZONE 'Asia/Jakarta'))
			) monthly
		),
		previous_peak_month AS (
			SELECT COALESCE(MAX(monthly_total), 0) as peak_revenue
			FROM (
				SELECT SUM(total_amount) as monthly_total
				FROM sales
				WHERE created_at >= $3 AND created_at < $4
					AND status = 'completed'
				GROUP BY EXTRACT(YEAR FROM (created_at AT TIME ZONE 'Asia/Jakarta')),
				         EXTRACT(MONTH FROM (created_at AT TIME ZONE 'Asia/Jakarta'))
			) monthly
		)
		SELECT
			cp.revenue, cp.orders,
			pp.revenue, pp.orders,
			cpeak_hour.peak_revenue, ppeak_hour.peak_revenue,
			cpeak_month.peak_revenue, ppeak_month.peak_revenue
		FROM current_period cp, previous_period pp,
		     current_peak_hour cpeak_hour, previous_peak_hour ppeak_hour,
		     current_peak_month cpeak_month, previous_peak_month ppeak_month`

	var result domain.PeriodComparison
	err := r.db.QueryRow(ctx, query,
		currentStart, currentEnd,
		previousStart, previousEnd,
	).Scan(
		&result.CurrentRevenue,
		&result.CurrentOrders,
		&result.PreviousRevenue,
		&result.PreviousOrders,
		&result.PeakRevenueHour,
		&result.PreviousPeakRevenue,
		&result.PeakRevenueMonth,
		&result.PreviousPeakRevenueMonth,
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

func (r *postgresRepository) GetDualChartData(
	ctx context.Context,
	currentStart, currentEnd, previousStart, previousEnd time.Time,
) (current, previous []ChartDataPoint, err error) {

	// Pass dates as YYYY-MM-DD strings and use AT TIME ZONE 'Asia/Jakarta'
	// so PostgreSQL correctly interprets them as WIB midnight, not UTC midnight
	cs := currentStart.Format("2006-01-02")
	ce := currentEnd.Format("2006-01-02")
	ps := previousStart.Format("2006-01-02")
	pe := previousEnd.Format("2006-01-02")

	query := `
		WITH date_series AS (
			SELECT generate_series(($1::date AT TIME ZONE 'Asia/Jakarta'), ($2::date AT TIME ZONE 'Asia/Jakarta'), '1 day') AS dt
		),
		current_agg AS (
			SELECT (created_at AT TIME ZONE 'Asia/Jakarta')::date AS dt,
				   COALESCE(SUM(total_amount), 0) AS revenue
			FROM sales
			WHERE created_at >= ($1::date AT TIME ZONE 'Asia/Jakarta') AND created_at < (($2::date + 1) AT TIME ZONE 'Asia/Jakarta')
				AND status = 'completed'
			GROUP BY 1
		),
		previous_agg AS (
			SELECT (created_at AT TIME ZONE 'Asia/Jakarta')::date AS dt,
				   COALESCE(SUM(total_amount), 0) AS revenue
			FROM sales
			WHERE created_at >= ($3::date AT TIME ZONE 'Asia/Jakarta') AND created_at < (($4::date + 1) AT TIME ZONE 'Asia/Jakarta')
				AND status = 'completed'
			GROUP BY 1
		)
		SELECT (ds.dt AT TIME ZONE 'Asia/Jakarta')::date,
			   COALESCE(c.revenue, 0),
			   COALESCE(p.revenue, 0)
		FROM date_series ds
		LEFT JOIN current_agg c ON c.dt = (ds.dt AT TIME ZONE 'Asia/Jakarta')::date
		LEFT JOIN previous_agg p ON p.dt = (ds.dt AT TIME ZONE 'Asia/Jakarta')::date - ($1::date - $3::date)
		ORDER BY 1`

	rows, err := r.db.Query(ctx, query, cs, ce, ps, pe)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var c ChartDataPoint
		var p ChartDataPoint
		var prevTotal int
		var currentDate time.Time
		if err := rows.Scan(&currentDate, &c.Total, &prevTotal); err != nil {
			return nil, nil, fmt.Errorf("scan row: %w", err)
		}
		c.Date = currentDate.Format("2006-01-02")
		// Compute prev date from currentDate using offset computed in SQL-compatible way
		// Offset = currentStart - previousStart in days
		offsetDays := int(currentStart.Sub(previousStart).Hours() / 24)
		prevTime := currentDate.AddDate(0, 0, -offsetDays)
		p.Date = prevTime.Format("2006-01-02")
		p.Total = prevTotal
		current = append(current, c)
		previous = append(previous, p)
	}

	return current, previous, rows.Err()
}

func (r *postgresRepository) GetLiveDashboardStats(ctx context.Context, storeID *int) (todaysRevenue, todaysSales, totalProducts, lowStockCount int, err error) {
	jakartaNow := time.Now().In(mustLoadJakarta())
	todayStart := time.Date(jakartaNow.Year(), jakartaNow.Month(), jakartaNow.Day(), 0, 0, 0, 0, mustLoadJakarta())
	todayEnd := todayStart.Add(24 * time.Hour)

	todayQuery := `
		SELECT COALESCE(SUM(total_amount), 0), COUNT(*)
		FROM sales
		WHERE created_at >= $1 AND created_at < $2
		  AND status = 'completed'`

	args := []interface{}{todayStart, todayEnd}
	argIdx := 3
	if storeID != nil {
		todayQuery += fmt.Sprintf(" AND store_id = $%d", argIdx)
		args = append(args, *storeID)
	}

	if err := r.db.QueryRow(ctx, todayQuery, args...).Scan(&todaysRevenue, &todaysSales); err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to query live dashboard sales: %w", err)
	}

	productsQuery := `SELECT COUNT(*) FROM products WHERE deleted_at IS NULL`
	args2 := []interface{}{}
	argIdx2 := 1
	if storeID != nil {
		productsQuery += fmt.Sprintf(" AND store_id = $%d", argIdx2)
		args2 = append(args2, *storeID)
	}
	if err := r.db.QueryRow(ctx, productsQuery, args2...).Scan(&totalProducts); err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to query live dashboard products: %w", err)
	}

	cfg := config.Load()
	stockQuery := `SELECT COUNT(*) FROM product_stock WHERE quantity <= $1`
	stockArgs := []interface{}{cfg.StockCriticalThreshold}
	stockIdx := 2
	if storeID != nil {
		stockQuery += fmt.Sprintf(" AND store_id = $%d", stockIdx)
		stockArgs = append(stockArgs, *storeID)
	}
	if err := r.db.QueryRow(ctx, stockQuery, stockArgs...).Scan(&lowStockCount); err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to query live dashboard low stock: %w", err)
	}

	return
}

// GetAvailableYears returns distinct years that have sales data
func (r *postgresRepository) GetAvailableYears(ctx context.Context, storeID *int) ([]int, error) {
	query := `
		SELECT DISTINCT EXTRACT(YEAR FROM (created_at AT TIME ZONE 'Asia/Jakarta'))::integer as year
		FROM sales
		WHERE status = 'completed'
	`
	args := []interface{}{}
	argIdx := 1

	if storeID != nil {
		query += fmt.Sprintf(" AND store_id = $%d", argIdx)
		args = append(args, *storeID)
	}

	query += " ORDER BY year DESC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch available years: %w", err)
	}
	defer rows.Close()

	var years []int
	for rows.Next() {
		var year int
		if err := rows.Scan(&year); err != nil {
			continue
		}
		years = append(years, year)
	}

	return years, nil
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

// GetStockByProductID returns the product_stock row for a product (generic store/warehouse)
func (r *postgresRepository) GetStockByProductID(ctx context.Context, productID int) (*domain.ProductStock, error) {
	var ps domain.ProductStock
	var storeID, warehouseID sql.NullInt64
	var reorderPoint, reorderQuantity sql.NullInt64
	var lastRestockedAt sql.NullTime
	var createdAt, updatedAt time.Time
	err := r.db.QueryRow(ctx, `
		SELECT id, product_id, warehouse_id, store_id, quantity, reorder_point, reorder_quantity, last_restocked_at, created_at, updated_at
		FROM product_stock
		WHERE product_id = $1
		ORDER BY id ASC LIMIT 1
	`, productID).Scan(&ps.ID, &ps.ProductID, &warehouseID, &storeID, &ps.Quantity, &reorderPoint, &reorderQuantity, &lastRestockedAt, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	if warehouseID.Valid {
		v := int(warehouseID.Int64)
		ps.WarehouseID = &v
	}
	if storeID.Valid {
		v := int(storeID.Int64)
		ps.StoreID = &v
	}
	if reorderPoint.Valid {
		v := int(reorderPoint.Int64)
		ps.ReorderPoint = v
	}
	if reorderQuantity.Valid {
		v := int(reorderQuantity.Int64)
		ps.ReorderQuantity = v
	}
	if lastRestockedAt.Valid {
		ps.LastRestockedAt = lastRestockedAt.Time.Format(time.RFC3339)
	}
	ps.CreatedAt = createdAt.Format(time.RFC3339)
	ps.UpdatedAt = updatedAt.Format(time.RFC3339)
	return &ps, nil
}

// AdjustStock updates product stock by quantity_change and records an inventory movement in a single transaction.
func (r *postgresRepository) AdjustStock(ctx context.Context, productID int, quantityChange int, userID *int, notes string) error {
	if quantityChange == 0 {
		return fmt.Errorf("quantity change must not be zero")
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var currentStock int
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(quantity, 0) FROM product_stock
		WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL
		FOR UPDATE
	`, productID).Scan(&currentStock)
	if err != nil {
		if err == pgx.ErrNoRows {
			currentStock = 0
		} else {
			return fmt.Errorf("failed to load product stock: %w", err)
		}
	}

	newStock := currentStock + quantityChange
	if newStock < 0 {
		return fmt.Errorf("insufficient stock: current %d, requested %d", currentStock, quantityChange)
	}

	cmd, err := tx.Exec(ctx, `
		UPDATE product_stock
		SET quantity = $2, updated_at = NOW()
		WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL
	`, productID, newStock)
	if err != nil {
		return fmt.Errorf("failed to update stock: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		_, err = tx.Exec(ctx, `
			INSERT INTO product_stock (product_id, quantity, updated_at)
			VALUES ($1, $2, NOW())
		`, productID, newStock)
		if err != nil {
			return fmt.Errorf("failed to insert stock: %w", err)
		}
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO inventory_movements (product_id, quantity_change, type, reference_id, reference_table, user_id, notes)
		VALUES ($1, $2, $3, NULL, NULL, $4, $5)
	`, productID, quantityChange, "adjustment", userID, notes)
	if err != nil {
		return fmt.Errorf("failed to record inventory movement: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit stock adjustment: %w", err)
	}

	return nil
}

// GetNextInvoiceNumber generates the next invoice number in format INV-YYYY-XXXXXX
// XXXXXX is a sequential 6-digit number within the current year
func (r *postgresRepository) GetNextInvoiceNumber(ctx context.Context) (string, error) {
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

func (r *postgresRepository) GetAllActive(ctx context.Context) ([]domain.PaymentMethod, error) {
	var methods []domain.PaymentMethod
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
		var m domain.PaymentMethod
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

func (r *postgresRepository) GetPaymentMethodByCode(ctx context.Context, code string) (*domain.PaymentMethod, error) {
	var m domain.PaymentMethod
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

func (r *postgresRepository) GetPaymentMethodByID(ctx context.Context, id int) (*domain.PaymentMethod, error) {
	var m domain.PaymentMethod
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

func (r *postgresRepository) GetByPhone(ctx context.Context, phone string) (*domain.Customer, error) {
	var c domain.Customer
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

func (r *postgresRepository) GetCustomerByID(ctx context.Context, id int) (*domain.Customer, error) {
	var c domain.Customer
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

func (r *postgresRepository) GetAllCustomers(ctx context.Context, limit, offset int, search string, isActive *bool) ([]domain.Customer, int, error) {
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

	var customers []domain.Customer
	for rows.Next() {
		var c domain.Customer
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

func (r *postgresRepository) CreateCustomer(ctx context.Context, customer *domain.Customer) error {
	var createdAt, updatedAt time.Time
	return r.db.QueryRow(ctx, `
		INSERT INTO customers (name, phone, email, address, tax_id, note, is_active, is_walk_in)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`, customer.Name, customer.Phone, customer.Email, customer.Address, customer.TaxID, customer.Note, customer.IsActive, customer.IsWalkIn).
		Scan(&customer.ID, &createdAt, &updatedAt)
}

func (r *postgresRepository) UpdateCustomer(ctx context.Context, customer *domain.Customer, id int) error {
	_, err := r.db.Exec(ctx, `
		UPDATE customers
		SET name = $1, phone = $2, email = $3, address = $4, tax_id = $5, note = $6, is_active = $7, updated_at = NOW()
		WHERE id = $8
	`, customer.Name, customer.Phone, customer.Email, customer.Address, customer.TaxID, customer.Note, customer.IsActive, id)
	return err
}

func (r *postgresRepository) DeleteCustomer(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, `UPDATE customers SET is_active = false, updated_at = NOW() WHERE id = $1`, id)
	return err
}

