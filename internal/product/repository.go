package product

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"retail-pos-system/internal/shared"
	"retail-pos-system/pkg/cache"

	"github.com/jackc/pgx/v5"
)

type Repository struct {
	db          shared.DBPool
	cache       *cache.Cache
	stockWriter ProductStockWriter
}

func NewRepository(db shared.DBPool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) SetCache(c *cache.Cache) {
	r.cache = c
}

// SetProductStockWriter wires the product_stock row writer port. internal/
// inventory provides the production implementation; the composition root MUST
// call this before any product stock write path runs (see ProductStockWriter).
func (r *Repository) SetProductStockWriter(w ProductStockWriter) {
	r.stockWriter = w
}

// setStoreStock writes a single product_stock row through the wired port. The
// caller's tx must be used to preserve atomicity. An unwired writer fails fast.
func (r *Repository) setStoreStock(ctx context.Context, tx pgx.Tx, productID int, storeID *int, quantity int) error {
	if r.stockWriter == nil {
		return errors.New("product: product_stock writer not wired; set ProductStockWriter")
	}
	return r.stockWriter.SetStoreStock(ctx, tx, shared.StockRowSet{ProductID: productID, StoreID: storeID, Quantity: quantity})
}

// setStoreStockBatch writes many product_stock rows through the wired port in
// a single transaction. An unwired writer fails fast.
func (r *Repository) setStoreStockBatch(ctx context.Context, items []shared.StockRowSet) error {
	if r.stockWriter == nil {
		return errors.New("product: product_stock writer not wired; set ProductStockWriter")
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := r.stockWriter.SetStoreStockBatch(ctx, tx, items); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

const productSelectCols = `
		SELECT v.id, v.sku, v.name, v.barcode, v.category_id, v.category_name, v.price, v.cost, v.stock, v.status, v.store_id,
		       v.brand_id, v.brand_name, v.unit_of_measure_id, v.unit_of_measure, v.weight_grams, v.description,
		       v.tax_class_id, v.tax_rate,
		       v.supplier_id, v.supplier_name,
		       v.created_at, v.updated_at
		FROM v_products_full v`

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

func (r *Repository) GetProductsByIDs(ctx context.Context, ids []int, storeID *int) ([]Product, error) {
	if len(ids) == 0 {
		return []Product{}, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	whereClause := fmt.Sprintf("v.id IN (%s)", strings.Join(placeholders, ","))
	if storeID != nil {
		whereClause += fmt.Sprintf(" AND v.store_id = $%d", len(ids)+1)
		args = append(args, *storeID)
	}

	query := fmt.Sprintf(`%s
		WHERE %s
		ORDER BY v.name`, productSelectCols, whereClause)

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
		INSERT INTO products (sku, name, barcode, category_id, price, cost, store_id, status,
		                    brand_id, description, tax_class_id, weight_grams, unit_of_measure_id, default_discount_percent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, created_at, updated_at
	`, product.SKU, product.Name, barcode, categoryID, product.Price, product.Cost, storeIDVal, product.Status,
		brandID, description, taxClassID, weightGrams, unitOfMeasureID, defaultDiscount).
		Scan(&product.ID, &createdTime, &updatedTime)
	if err != nil {
		return err
	}
	product.CreatedAt = createdTime.In(shared.JakartaLocation()).Format(time.RFC3339)
	product.UpdatedAt = updatedTime.In(shared.JakartaLocation()).Format(time.RFC3339)

	if err := r.setStoreStock(ctx, tx, product.ID, product.StoreID, product.Stock); err != nil {
		return err
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
		cost = $6, store_id = $7, status = $8, updated_at = NOW(),
		brand_id = $9, description = $10, tax_class_id = $11, weight_grams = $12, unit_of_measure_id = $13, default_discount_percent = $14
		WHERE id = $15`
	args := []interface{}{product.SKU, product.Name, barcode, categoryID, product.Price, product.Cost, storeIDVal, product.Status,
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

	if product.StoreID != nil {
		if err := r.setStoreStock(ctx, tx, product.ID, product.StoreID, product.Stock); err != nil {
			return err
		}
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
		UPDATE products SET sku = $1, name = $2, barcode = $3, category_id = $4, price = $5, cost = $6,
		    store_id = $7, status = $8, deleted_at = NULL, updated_at = NOW()
		WHERE id = $9
	`, product.SKU, product.Name, barcode, categoryID, product.Price, product.Cost, storeIDVal, product.Status, product.ID)
	if err != nil {
		return err
	}

	if product.StoreID != nil {
		if err := r.setStoreStock(ctx, tx, product.ID, product.StoreID, product.Stock); err != nil {
			return err
		}
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

func (r *Repository) GetActiveProductOptions(ctx context.Context) ([]Option, error) {
	query := "SELECT id, sku, name FROM products WHERE deleted_at IS NULL AND status = 'active' ORDER BY name"

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var options []Option
	for rows.Next() {
		var opt Option
		if err := rows.Scan(&opt.ID, &opt.SKU, &opt.Name); err != nil {
			return nil, err
		}
		options = append(options, opt)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return options, nil
}
