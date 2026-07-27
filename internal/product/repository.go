package product

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"retail-pos-system/internal/shared"
	"retail-pos-system/pkg/cache"

	"github.com/jackc/pgx/v5"
)

type Repository struct {
	db    shared.DBPool
	cache *cache.Cache
}

func NewRepository(db shared.DBPool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) SetCache(c *cache.Cache) {
	r.cache = c
}

const productSelectCols = `
		SELECT v.id, v.sku, v.name, v.barcode, v.category_id, v.category_name, v.price, v.cost, v.stock, v.status, v.store_id,
		       v.brand_id, v.brand_name, v.unit_of_measure_id, v.unit_of_measure, v.weight_grams, v.description,
		       v.tax_class_id, v.tax_rate,
		       ps_preferred.supplier_id, ps_preferred.supplier_name,
		       v.created_at, v.updated_at
		FROM v_products_full v
		LEFT JOIN LATERAL (
			SELECT s.id as supplier_id, s.name as supplier_name
			FROM product_suppliers ps
			JOIN suppliers s ON ps.supplier_id = s.id AND s.deleted_at IS NULL
			WHERE ps.product_id = v.id AND ps.is_preferred = true
			LIMIT 1
		) ps_preferred ON true`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanProduct(row rowScanner) (*Product, error) {
	var p Product
	var barcode sql.NullString
	var categoryIDVal, storeIDVal, brandIDVal, unitOfMeasureIDVal, weightGramsVal sql.NullInt64
	var taxClassIDVal sql.NullInt64
	var taxRateVal sql.NullFloat64
	var categoryName, brandName, unitOfMeasure, description sql.NullString
	var supplierIDVal sql.NullInt64
	var supplierNameVal sql.NullString
	var createdAt, updatedAt time.Time

	err := row.Scan(&p.ID, &p.SKU, &p.Name, &barcode, &categoryIDVal, &categoryName, &p.Price, &p.Cost, &p.Stock, &p.Status,
		&storeIDVal, &brandIDVal, &brandName, &unitOfMeasureIDVal, &unitOfMeasure, &weightGramsVal, &description,
		&taxClassIDVal, &taxRateVal,
		&supplierIDVal, &supplierNameVal,
		&createdAt, &updatedAt)
	if err != nil {
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
	if taxClassIDVal.Valid {
		v := int(taxClassIDVal.Int64)
		p.TaxClassID = &v
	}
	if taxRateVal.Valid {
		v := taxRateVal.Float64
		p.TaxRate = &v
	}
	if supplierIDVal.Valid {
		v := int(supplierIDVal.Int64)
		p.SupplierID = &v
	}
	if supplierNameVal.Valid {
		p.SupplierName = &supplierNameVal.String
	}
	p.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	p.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)

	return &p, nil
}

func (r *Repository) GetProductPrice(ctx context.Context, id int) (int, error) {
	if r.cache != nil {
		key := fmt.Sprintf("product:price:%d", id)
		if v, ok := r.cache.Get(key); ok {
			return v.(int), nil
		}
	}
	var price int
	err := r.db.QueryRow(ctx, "SELECT price FROM products WHERE id = $1", id).Scan(&price)
	if err != nil {
		return 0, fmt.Errorf("get product price: %w", err)
	}
	if r.cache != nil {
		r.cache.SetWithTTL(fmt.Sprintf("product:price:%d", id), price, 5*time.Minute)
	}
	return price, nil
}

func (r *Repository) GetProductPrices(ctx context.Context, ids []int) (map[int]int, error) {
	prices := make(map[int]int, len(ids))
	if len(ids) == 0 {
		return prices, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf("SELECT id, price FROM products WHERE id IN (%s)", strings.Join(placeholders, ","))
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("batch get product prices: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, price int
		if err := rows.Scan(&id, &price); err != nil {
			return nil, err
		}
		prices[id] = price
		if r.cache != nil {
			r.cache.SetWithTTL(fmt.Sprintf("product:price:%d", id), price, 5*time.Minute)
		}
	}
	return prices, rows.Err()
}

func (r *Repository) GetProductsByIDs(ctx context.Context, ids []int) ([]Product, error) {
	if len(ids) == 0 {
		return []Product{}, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`%s
		WHERE v.id IN (%s)
		ORDER BY v.name`, productSelectCols, strings.Join(placeholders, ","))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("batch get products by ids: %w", err)
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}
		products = append(products, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return products, nil
}

// ==================== CORE CRUD ====================

func (r *Repository) GetProductByID(ctx context.Context, id int, storeID *int) (*Product, error) {
	query := fmt.Sprintf(`%s
		WHERE v.id = $1`, productSelectCols)

	args := []interface{}{id}
	if storeID != nil {
		query += fmt.Sprintf(" AND v.store_id = $%d", len(args)+1)
		args = append(args, *storeID)
	}

	p, err := scanProduct(r.db.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("product not found")
		}
		return nil, err
	}
	return p, nil
}

func (r *Repository) GetProductBySKU(ctx context.Context, sku string, storeID *int) (*Product, error) {
	query := fmt.Sprintf(`%s
		WHERE v.sku = $1`, productSelectCols)

	args := []interface{}{sku}
	if storeID != nil {
		query += fmt.Sprintf(" AND v.store_id = $%d", len(args)+1)
		args = append(args, *storeID)
	}

	p, err := scanProduct(r.db.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("product not found")
		}
		return nil, err
	}
	return p, nil
}

func (r *Repository) CreateProduct(ctx context.Context, product *Product) error {
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

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var createdTime, updatedTime time.Time
	err = tx.QueryRow(ctx, `
		INSERT INTO products (sku, name, barcode, category_id, price, cost, stock, store_id, status,
		                    brand_id, description, tax_class_id, weight_grams, unit_of_measure_id, default_discount_percent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_at, updated_at
	`, product.SKU, product.Name, barcode, categoryID, product.Price, product.Cost, product.Stock, storeIDVal, product.Status,
		brandID, description, taxClassID, weightGrams, unitOfMeasureID, defaultDiscount).
		Scan(&product.ID, &createdTime, &updatedTime)
	if err != nil {
		return err
	}
	product.CreatedAt = createdTime.In(shared.JakartaLocation()).Format(time.RFC3339)
	product.UpdatedAt = updatedTime.In(shared.JakartaLocation()).Format(time.RFC3339)

	_, err = tx.Exec(ctx, `
		INSERT INTO product_stock (product_id, store_id, quantity) VALUES ($1, $2, $3)
	`, product.ID, storeIDVal, product.Stock)
	if err != nil {
		return fmt.Errorf("failed to initialize product stock: %w", err)
	}
	_, err = tx.Exec(ctx, `
		UPDATE products SET stock = $1 WHERE id = $2
	`, product.Stock, product.ID)
	if err != nil {
		return fmt.Errorf("failed to sync product stock: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	if r.cache != nil {
		r.cache.Delete(fmt.Sprintf("product:%d", product.ID))
		r.cache.Delete(fmt.Sprintf("product:price:%d", product.ID))
	}
	return nil
}

func (r *Repository) UpdateProduct(ctx context.Context, product *Product, storeID *int) error {
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

	query := `UPDATE products SET sku = $1, name = $2, barcode = $3, category_id = $4, price = $5,
		cost = $6, stock = $7, store_id = $8, status = $9, updated_at = NOW(),
		brand_id = $10, description = $11, tax_class_id = $12, weight_grams = $13, unit_of_measure_id = $14, default_discount_percent = $15
		WHERE id = $16`
	args := []interface{}{product.SKU, product.Name, barcode, categoryID, product.Price, product.Cost, product.Stock, storeIDVal, product.Status,
		brandID, description, taxClassID, weightGrams, unitOfMeasureID, defaultDiscount, product.ID}
	if storeID != nil {
		query += fmt.Sprintf(" AND store_id = $%d", len(args)+1)
		args = append(args, *storeID)
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	if storeIDVal != nil {
		_, err = tx.Exec(ctx, `
			INSERT INTO product_stock (product_id, store_id, quantity) VALUES ($1, $2, $3)
			ON CONFLICT (product_id, warehouse_id, store_id) DO UPDATE SET quantity = $3
		`, product.ID, storeIDVal, product.Stock)
		if err != nil {
			return fmt.Errorf("failed to sync product stock: %w", err)
		}
	}
	_, err = tx.Exec(ctx, `
		UPDATE products SET stock = $1 WHERE id = $2
	`, product.Stock, product.ID)
	if err != nil {
		return fmt.Errorf("failed to sync product stock: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	if r.cache != nil {
		r.cache.Delete(fmt.Sprintf("product:%d", product.ID))
		r.cache.Delete(fmt.Sprintf("product:price:%d", product.ID))
	}
	return nil
}

func (r *Repository) DeleteProduct(ctx context.Context, id int, storeID *int) error {
	query := "UPDATE products SET deleted_at = NOW(), status = 'archived' WHERE id = $1"
	args := []interface{}{id}
	if storeID != nil {
		query += fmt.Sprintf(" AND store_id = $%d", len(args)+1)
		args = append(args, *storeID)
	}
	_, err := r.db.Exec(ctx, query, args...)
	if err == nil && r.cache != nil {
		r.cache.Delete(fmt.Sprintf("product:%d", id))
		r.cache.Delete(fmt.Sprintf("product:price:%d", id))
	}
	return err
}

func (r *Repository) RestoreProduct(ctx context.Context, product *Product) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

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

	_, err = tx.Exec(ctx, `
		UPDATE products SET sku = $1, name = $2, barcode = $3, category_id = $4, price = $5, cost = $6, stock = $7,
		    store_id = $8, status = $9, deleted_at = NULL, updated_at = NOW()
		WHERE id = $10
	`, product.SKU, product.Name, barcode, categoryID, product.Price, product.Cost, product.Stock, storeIDVal, product.Status, product.ID)
	if err != nil {
		return err
	}

	if storeIDVal != nil {
		_, err = tx.Exec(ctx, `
			INSERT INTO product_stock (product_id, store_id, quantity) VALUES ($1, $2, $3)
		`, product.ID, storeIDVal, product.Stock)
		if err != nil {
			return fmt.Errorf("failed to restore product stock: %w", err)
		}
	}
	_, err = tx.Exec(ctx, `
		UPDATE products SET stock = $1 WHERE id = $2
	`, product.Stock, product.ID)
	if err != nil {
		return fmt.Errorf("failed to sync product stock: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	if r.cache != nil {
		r.cache.Delete(fmt.Sprintf("product:%d", product.ID))
		r.cache.Delete(fmt.Sprintf("product:price:%d", product.ID))
	}
	return nil
}

func (r *Repository) GetNextSKU(ctx context.Context) (string, error) {
	var skuNum int
	err := r.db.QueryRow(ctx, "SELECT nextval('sku_seq')").Scan(&skuNum)
	if err != nil {
		return "", fmt.Errorf("failed to get next SKU: %w", err)
	}
	return fmt.Sprintf("SKU-%06d", skuNum), nil
}

// Tax class operations
func (r *Repository) GetTaxClassByID(ctx context.Context, id int) (*TaxClass, error) {
	var tc TaxClass
	var createdAt time.Time
	err := r.db.QueryRow(ctx, "SELECT id, name, rate_percent, COALESCE(description,''), is_active, created_at FROM tax_classes WHERE id = $1", id).Scan(
		&tc.ID, &tc.Name, &tc.RatePercent, &tc.Description, &tc.IsActive, &createdAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("tax class not found")
		}
		return nil, err
	}
	tc.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &tc, nil
}

func (r *Repository) GetAllTaxClasses(ctx context.Context) ([]TaxClass, error) {
	rows, err := r.db.Query(ctx, "SELECT id, name, rate_percent, COALESCE(description,''), is_active, created_at FROM tax_classes WHERE is_active = true ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var taxClasses []TaxClass
	for rows.Next() {
		var tc TaxClass
		var createdAt time.Time
		if err := rows.Scan(&tc.ID, &tc.Name, &tc.RatePercent, &tc.Description, &tc.IsActive, &createdAt); err != nil {
			return nil, err
		}
		tc.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		taxClasses = append(taxClasses, tc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return taxClasses, nil
}

func (r *Repository) GetTaxClassIDByName(ctx context.Context, name string) (int, error) {
	var id int
	err := r.db.QueryRow(ctx, "SELECT id FROM tax_classes WHERE name = $1 AND is_active = true", name).Scan(&id)
	return id, err
}

// Warehouse operations
func (r *Repository) GetWarehouseByID(ctx context.Context, id int) (*Warehouse, error) {
	var w Warehouse
	var storeID sql.NullInt64
	var createdAt time.Time
	err := r.db.QueryRow(ctx, "SELECT id, name, code, COALESCE(address,''), store_id, is_active, created_at FROM warehouses WHERE id = $1", id).Scan(
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
	w.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &w, nil
}

func (r *Repository) GetAllWarehouses(ctx context.Context, storeID *int) ([]Warehouse, error) {
	query := "SELECT id, name, code, COALESCE(address,''), store_id, is_active, created_at FROM warehouses WHERE is_active = true"
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

	var warehouses []Warehouse
	for rows.Next() {
		var w Warehouse
		var storeIDVal sql.NullInt64
		var createdAt time.Time
		if err := rows.Scan(&w.ID, &w.Name, &w.Code, &w.Address, &storeIDVal, &w.IsActive, &createdAt); err != nil {
			return nil, err
		}
		if storeIDVal.Valid {
			v := int(storeIDVal.Int64)
			w.StoreID = &v
		}
		w.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		warehouses = append(warehouses, w)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return warehouses, nil
}

func (r *Repository) GetOrCreateCategoryIDByName(ctx context.Context, name string) (int, error) {
	var id int
	err := r.db.QueryRow(ctx, "SELECT id FROM categories WHERE name = $1 AND is_active = true", name).Scan(&id)
	if err == nil {
		return id, nil
	}
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = strings.ReplaceAll(slug, " ", "-")
	err = r.db.QueryRow(ctx, `
		INSERT INTO categories (name, slug, description, is_active)
		VALUES ($1, $2, '', true)
		RETURNING id
	`, name, slug).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to auto-create category: %w", err)
	}
	return id, nil
}
