package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
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
		role, _ := r.GetRoleByID(ctx, u.RoleID)
		if role != nil {
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
	err := r.db.QueryRow(ctx, "SELECT id, name, description, is_system, created_at FROM roles WHERE id = $1", id).Scan(
		&role.ID, &role.Name, &role.Description, &isSystem, &role.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("role not found")
		}
		return nil, err
	}
	role.IsSystem = isSystem
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
		SELECT p.id, p.sku, p.name, p.barcode, p.category_id, c.name as category_name, p.price, p.cost, p.stock, p.stock_min, p.stock_max,
		       p.store_id, p.is_active, p.created_at, p.updated_at
		FROM products p 
		LEFT JOIN categories c ON p.category_id = c.id 
		WHERE p.id = $1 AND p.deleted_at IS NULL`

	args := []interface{}{id}
	if storeID != nil {
		query += fmt.Sprintf(" AND p.store_id = $%d", len(args)+1)
		args = append(args, *storeID)
	}

	err := r.db.QueryRow(ctx, query, args...).Scan(&p.ID, &p.SKU, &p.Name, &barcode, &categoryIDVal, &categoryName, &p.Price, &p.Cost, &p.Stock, &p.StockMin, &p.StockMax,
		&storeIDVal, &p.IsActive, &createdAt, &updatedAt)
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
		SELECT p.id, p.sku, p.name, p.barcode, p.category_id, c.name as category_name, p.price, p.cost, p.stock, p.stock_min, p.stock_max,
		       p.store_id, p.is_active, p.created_at, p.updated_at
		FROM products p 
		LEFT JOIN categories c ON p.category_id = c.id 
		WHERE p.sku = $1 AND p.deleted_at IS NULL`

	args := []interface{}{sku}
	if storeID != nil {
		query += fmt.Sprintf(" AND p.store_id = $%d", len(args)+1)
		args = append(args, *storeID)
	}

	err := r.db.QueryRow(ctx, query, args...).Scan(&p.ID, &p.SKU, &p.Name, &barcode, &categoryIDVal, &categoryName, &p.Price, &p.Cost, &p.Stock, &p.StockMin, &p.StockMax,
		&storeIDVal, &p.IsActive, &createdAt, &updatedAt)
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

func (r *postgresRepository) GetAllProducts(ctx context.Context, limit, offset int, search string, categoryID *int, sortBy, sortDir string, maxStock *int, storeID *int) ([]domain.Product, int, error) {
	var products []domain.Product
	var total int

	query := `SELECT COUNT(*) 
		FROM products p 
		WHERE p.deleted_at IS NULL`
	args := []interface{}{}
	if search != "" {
		query += " AND (p.name ILIKE $1 OR p.sku ILIKE $1 OR p.barcode ILIKE $1)"
		args = append(args, "%"+search+"%")
	}
	if categoryID != nil {
		query += fmt.Sprintf(" AND p.category_id = $%d", len(args)+1)
		args = append(args, *categoryID)
	}
	if maxStock != nil {
		query += fmt.Sprintf(" AND p.stock <= $%d", len(args)+1)
		args = append(args, *maxStock)
	}
	if storeID != nil {
		query += fmt.Sprintf(" AND p.store_id = $%d", len(args)+1)
		args = append(args, *storeID)
	}

	err := r.db.QueryRow(ctx, query, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count products: %w", err)
	}

	query = `SELECT p.id, p.sku, p.name, p.barcode, p.category_id, c.name as category_name, p.price, p.cost, p.stock, p.stock_min, p.stock_max, p.store_id, p.is_active, p.created_at, p.updated_at 
		FROM products p 
		LEFT JOIN categories c ON p.category_id = c.id 
		WHERE p.deleted_at IS NULL`
	args2 := []interface{}{}
	if search != "" {
		query += " AND (p.name ILIKE $1 OR p.sku ILIKE $1 OR p.barcode ILIKE $1)"
		args2 = append(args2, "%"+search+"%")
	}
	if categoryID != nil {
		query += fmt.Sprintf(" AND p.category_id = $%d", len(args2)+1)
		args2 = append(args2, *categoryID)
	}
	if maxStock != nil {
		query += fmt.Sprintf(" AND p.stock <= $%d", len(args2)+1)
		args2 = append(args2, *maxStock)
	}
	if storeID != nil {
		query += fmt.Sprintf(" AND p.store_id = $%d", len(args2)+1)
		args2 = append(args2, *storeID)
	}
	if sortBy != "" {
		query += fmt.Sprintf(" ORDER BY %s", sortBy)
		if sortDir != "" {
			query += " " + sortDir
		}
	} else {
		query += " ORDER BY p.id DESC"
	}
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args2)+1, len(args2)+2)
	args2 = append(args2, limit, offset)

	rows, err := r.db.Query(ctx, query, args2...)
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

		err = rows.Scan(&p.ID, &p.SKU, &p.Name, &barcodeVal, &categoryIDVal, &categoryName, &p.Price, &p.Cost, &p.Stock, &p.StockMin, &p.StockMax,
			&storeIDVal, &p.IsActive, &createdAt, &updatedAt)
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

	return r.db.QueryRow(ctx, `
		INSERT INTO products (sku, name, barcode, category_id, price, cost, stock, stock_min, stock_max, store_id, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`, product.SKU, product.Name, barcode, categoryID, product.Price, product.Cost, product.Stock, product.StockMin, product.StockMax, storeIDVal, product.IsActive).
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

	_, err := r.db.Exec(ctx, `
		UPDATE products SET sku = $1, name = $2, barcode = $3, category_id = $4, price = $5,
			cost = $6, stock = $7, stock_min = $8, stock_max = $9, store_id = $10, is_active = $11, updated_at = NOW()
		WHERE id = $12
	`, product.SKU, product.Name, barcode, categoryID, product.Price, product.Cost, product.Stock, product.StockMin, product.StockMax, storeIDVal, product.IsActive, product.ID)
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
		return err
	}
	sale.CreatedAt = createdAt.Format(time.RFC3339)
	sale.UpdatedAt = updatedAt.Format(time.RFC3339)

	for i := range items {
		_, err = tx.Exec(ctx, `
			INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal)
			VALUES ($1, $2, $3, $4, $5)
		`, sale.ID, items[i].ProductID, items[i].Quantity, items[i].UnitPrice, items[i].Subtotal)
		if err != nil {
			return err
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

	query := `SELECT COUNT(*) FROM sales WHERE 1=1`
	args := []interface{}{}
	if search != "" {
		query += " AND invoice_number ILIKE $" + fmt.Sprintf("%d", len(args)+1)
		args = append(args, "%"+search+"%")
	}
	if startDate != "" {
		query += " AND DATE(created_at) >= $" + fmt.Sprintf("%d", len(args)+1)
		args = append(args, startDate)
	}
	if endDate != "" {
		query += " AND DATE(created_at) <= $" + fmt.Sprintf("%d", len(args)+1)
		args = append(args, endDate)
	}
	if storeID != nil {
		query += fmt.Sprintf(" AND store_id = $%d", len(args)+1)
		args = append(args, *storeID)
	}

	err := r.db.QueryRow(ctx, query, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query = `SELECT id, invoice_number, cashier_id, store_id, subtotal, discount, tax, total_amount, payment_method, status, created_at, updated_at FROM sales WHERE 1=1`
	args2 := []interface{}{}
	if search != "" {
		query += " AND invoice_number ILIKE $" + fmt.Sprintf("%d", len(args2)+1)
		args2 = append(args2, "%"+search+"%")
	}
	if startDate != "" {
		query += " AND created_at >= $" + fmt.Sprintf("%d", len(args2)+1)
		args2 = append(args2, startDate+" 00:00:00")
	}
	if endDate != "" {
		query += " AND created_at <= $" + fmt.Sprintf("%d", len(args2)+1)
		args2 = append(args2, endDate+" 23:59:59")
	}
	if storeID != nil {
		query += fmt.Sprintf(" AND store_id = $%d", len(args2)+1)
		args2 = append(args2, *storeID)
	}
	if sortBy != "" {
		query += fmt.Sprintf(" ORDER BY %s", sortBy)
		if sortDir != "" {
			query += " " + sortDir
		}
	} else {
		query += " ORDER BY created_at DESC"
	}
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args2)+1, len(args2)+2)
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
		INSERT INTO audit_logs (user_id, username, role, action, entity_type, entity_id, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, log.UserID, log.Username, log.Role, log.Action, log.EntityType, log.EntityID, log.IPAddress)
	return err
}

func (r *postgresRepository) GetAuditLogs(ctx context.Context, limit, offset int, userID *int) ([]domain.AuditLog, int, error) {
	var logs []domain.AuditLog
	var total int

	query := `SELECT COUNT(*) FROM audit_logs`
	args := []interface{}{}
	if userID != nil {
		query += fmt.Sprintf(" WHERE user_id = $%d", len(args)+1)
		args = append(args, *userID)
	}

	err := r.db.QueryRow(ctx, query, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query = `SELECT id, user_id, username, role, action, entity_type, entity_id, ip_address, created_at FROM audit_logs`
	args2 := []interface{}{}
	if userID != nil {
		query += fmt.Sprintf(" WHERE user_id = $%d", len(args2)+1)
		args2 = append(args2, *userID)
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args2)+1, len(args2)+2)
	args2 = append(args2, limit, offset)

	rows, err := r.db.Query(ctx, query, args2...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var log domain.AuditLog
		err = rows.Scan(&log.ID, &log.UserID, &log.Username, &log.Role, &log.Action, &log.EntityType, &log.EntityID, &log.IPAddress, &log.CreatedAt)
		if err != nil {
			continue
		}
		logs = append(logs, log)
	}
	return logs, total, nil
}

// ==================== ADAPTERS ====================

func (r *postgresRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	return r.CreateAuditLog(ctx, log)
}

func (r *postgresRepository) GetAll(ctx context.Context, limit, offset int, userID *int) ([]domain.AuditLog, int, error) {
	return r.GetAuditLogs(ctx, limit, offset, userID)
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
