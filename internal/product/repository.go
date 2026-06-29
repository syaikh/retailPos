package product

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"retail-pos-system/internal/importutil"

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

func (r *Repository) GetProductPrice(ctx context.Context, id int) (int, error) {
	var price int
	err := r.db.QueryRow(ctx, "SELECT price FROM products WHERE id = $1", id).Scan(&price)
	if err != nil {
		return 0, fmt.Errorf("get product price: %w", err)
	}
	return price, nil
}

// ==================== PRODUCT ====================

func (r *Repository) GetProductByID(ctx context.Context, id int, storeID *int) (*Product, error) {
	var p Product
	var barcode sql.NullString
	var categoryIDVal, storeIDVal, brandIDVal, unitOfMeasureIDVal, weightGramsVal sql.NullInt64
	var taxClassIDVal sql.NullInt64
	var taxRateVal sql.NullFloat64
	var categoryName, brandName, unitOfMeasure, description sql.NullString
	var createdAt, updatedAt time.Time

	query := `
		SELECT v.id, v.sku, v.name, v.barcode, v.category_id, v.category_name, v.price, v.cost, v.stock, v.status,
		       v.store_id, v.brand_id, v.brand_name, v.unit_of_measure_id, v.unit_of_measure, v.weight_grams, v.description,
		       v.tax_class_id, v.tax_rate,
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
		&taxClassIDVal, &taxRateVal,
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
	if taxClassIDVal.Valid {
		v := int(taxClassIDVal.Int64)
		p.TaxClassID = &v
	}
	if taxRateVal.Valid {
		v := taxRateVal.Float64
		p.TaxRate = &v
	}
	p.CreatedAt = createdAt.In(jakartaLoc).Format(time.RFC3339)
	p.UpdatedAt = updatedAt.In(jakartaLoc).Format(time.RFC3339)

	return &p, nil
}

func (r *Repository) GetProductBySKU(ctx context.Context, sku string, storeID *int) (*Product, error) {
	var p Product
	var barcode sql.NullString
	var categoryIDVal, storeIDVal, brandIDVal, unitOfMeasureIDVal, weightGramsVal sql.NullInt64
	var taxClassIDVal sql.NullInt64
	var taxRateVal sql.NullFloat64
	var categoryName, brandName, unitOfMeasure, description sql.NullString
	var createdAt, updatedAt time.Time

	query := `
		SELECT v.id, v.sku, v.name, v.barcode, v.category_id, v.category_name, v.price, v.cost, v.stock, v.status,
		       v.store_id, v.brand_id, v.brand_name, v.unit_of_measure_id, v.unit_of_measure, v.weight_grams, v.description,
		       v.tax_class_id, v.tax_rate,
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
		&taxClassIDVal, &taxRateVal,
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
	if taxClassIDVal.Valid {
		v := int(taxClassIDVal.Int64)
		p.TaxClassID = &v
	}
	if taxRateVal.Valid {
		v := taxRateVal.Float64
		p.TaxRate = &v
	}
	p.CreatedAt = createdAt.In(jakartaLoc).Format(time.RFC3339)
	p.UpdatedAt = updatedAt.In(jakartaLoc).Format(time.RFC3339)

	return &p, nil
}

func (r *Repository) GetAllProducts(ctx context.Context, limit, offset int, search string, categoryIDs []int, sortBy, sortDir string, maxStock *int, storeID *int, status string) ([]Product, int, error) {
	var products []Product
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
		       v.tax_class_id, v.tax_rate,
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
	allowedSortBy := map[string]bool{"v.id": true, "v.name": true, "v.sku": true, "v.barcode": true, "v.price": true, "v.status": true, "v.created_at": true, "v.updated_at": true, "v.stock": true, "category_name": true, "brand_name": true}
	allowedSortDir := map[string]bool{"ASC": true, "DESC": true}
	if sortBy != "" && allowedSortBy[sortBy] {
		query2 += fmt.Sprintf(" ORDER BY %s", sortBy)
		if sortDir != "" && allowedSortDir[sortDir] {
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
		var p Product
		var barcodeVal sql.NullString
		var categoryIDVal, storeIDVal, brandIDVal, unitOfMeasureIDVal, weightGramsVal sql.NullInt64
		var taxClassIDVal sql.NullInt64
		var taxRateVal sql.NullFloat64
		var categoryName, brandName, unitOfMeasure, descriptionVal sql.NullString
		var createdAt, updatedAt time.Time

		err = rows.Scan(&p.ID, &p.SKU, &p.Name, &barcodeVal, &categoryIDVal, &categoryName, &p.Price, &p.Cost, &p.Stock, &p.Status,
			&storeIDVal, &brandIDVal, &brandName, &unitOfMeasureIDVal, &unitOfMeasure, &weightGramsVal, &descriptionVal,
			&taxClassIDVal, &taxRateVal,
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
		if taxClassIDVal.Valid {
			v := int(taxClassIDVal.Int64)
			p.TaxClassID = &v
		}
		if taxRateVal.Valid {
			v := taxRateVal.Float64
			p.TaxRate = &v
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

	var createdTime, updatedTime time.Time
	err := r.db.QueryRow(ctx, `
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
	product.CreatedAt = createdTime.Format(time.RFC3339)
	product.UpdatedAt = updatedTime.Format(time.RFC3339)

	_, err = r.db.Exec(ctx, `
		INSERT INTO product_stock (product_id, store_id, quantity) VALUES ($1, $2, $3)
	`, product.ID, storeIDVal, product.Stock)
	if err != nil {
		return fmt.Errorf("failed to initialize product stock: %w", err)
	}
	_, err = r.db.Exec(ctx, `
		UPDATE products SET stock = $1 WHERE id = $2
	`, product.Stock, product.ID)
	if err != nil {
		return fmt.Errorf("failed to sync product stock: %w", err)
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

func (r *Repository) DeleteProduct(ctx context.Context, id int, storeID *int) error {
	_, err := r.db.Exec(ctx, "UPDATE products SET deleted_at = NOW(), status = 'archived' WHERE id = $1", id)
	return err
}

func (r *Repository) BulkUpdateProductStatus(ctx context.Context, ids []int, status string, storeID *int) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.db.Exec(ctx, "UPDATE products SET status = $1 WHERE id = ANY($2)", status, ids)
	return err
}

func (r *Repository) GetDeletedProductByBarcode(ctx context.Context, barcode string, storeID *int) (*Product, error) {
	var p Product
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

func (r *Repository) RestoreProduct(ctx context.Context, product *Product) error {
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

func (r *Repository) GetNextSKU(ctx context.Context) (string, error) {
	var skuNum int
	err := r.db.QueryRow(ctx, "SELECT nextval('sku_seq')").Scan(&skuNum)
	if err != nil {
		return "", fmt.Errorf("failed to get next SKU: %w", err)
	}
	return fmt.Sprintf("SKU-%06d", skuNum), nil
}

// Brand operations
func (r *Repository) GetBrandByID(ctx context.Context, id int) (*Brand, error) {
	var brand Brand
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

func (r *Repository) GetAllBrands(ctx context.Context) ([]Brand, error) {
	rows, err := r.db.Query(ctx, "SELECT id, name, description, is_active, created_at, updated_at FROM brands WHERE is_active = true ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var brands []Brand
	for rows.Next() {
		var b Brand
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&b.ID, &b.Name, &b.Description, &b.IsActive, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		b.CreatedAt = createdAt.Format(time.RFC3339)

		brands = append(brands, b)
	}
	return brands, nil
}

func (r *Repository) CreateBrand(ctx context.Context, brand *Brand) error {
	var createdAt, updatedAt time.Time
	return r.db.QueryRow(ctx, `
		INSERT INTO brands (name, description, is_active) 
		VALUES ($1, $2, $3) 
		RETURNING id, created_at, updated_at
	`, brand.Name, brand.Description, brand.IsActive).Scan(&brand.ID, &createdAt, &updatedAt)
}

func (r *Repository) UpdateBrand(ctx context.Context, brand *Brand) error {
	_, err := r.db.Exec(ctx, `
		UPDATE brands SET name = $1, description = $2, is_active = $3, updated_at = NOW() 
		WHERE id = $4
	`, brand.Name, brand.Description, brand.IsActive, brand.ID)
	return err
}

func (r *Repository) DeleteBrand(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, "DELETE FROM brands WHERE id = $1", id)
	return err
}

func (r *Repository) GetBrandIDByName(ctx context.Context, name string) (int, error) {
	var id int
	err := r.db.QueryRow(ctx, "SELECT id FROM brands WHERE name = $1 AND is_active = true", name).Scan(&id)
	return id, err
}

func (r *Repository) GetAllBrandsForExport(ctx context.Context) ([]Brand, error) {
	rows, err := r.db.Query(ctx, "SELECT id, name, COALESCE(description,''), is_active, created_at, updated_at FROM brands ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var brands []Brand
	for rows.Next() {
		var b Brand
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&b.ID, &b.Name, &b.Description, &b.IsActive, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		b.CreatedAt = createdAt.Format(time.RFC3339)
		brands = append(brands, b)
	}
	return brands, nil
}

func (r *Repository) BulkUpsertBrands(ctx context.Context, records []BrandImportRow) importutil.ImportResult {
	result := importutil.ImportResult{Errors: []string{}}

	for _, rec := range records {
		if rec.Name == "" {
			result.AddError(rec.Row, "Name is required")
			continue
		}

		var existingID int
		err := r.db.QueryRow(ctx, "SELECT id FROM brands WHERE name = $1", rec.Name).Scan(&existingID)
		isUpdate := err == nil

		_, err = r.db.Exec(ctx, `
			INSERT INTO brands (name, description, is_active)
			VALUES ($1, $2, $3)
			ON CONFLICT (name) DO UPDATE SET
				description = EXCLUDED.description,
				is_active = EXCLUDED.is_active,
				updated_at = NOW()
		`, rec.Name, rec.Description, rec.IsActive)

		if err != nil {
			result.AddError(rec.Row, fmt.Sprintf("failed to upsert: %v", err))
			continue
		}

		if isUpdate {
			result.Updated++
		} else {
			result.Inserted++
		}
	}

	return result
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
	tc.CreatedAt = createdAt.Format(time.RFC3339)
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
		tc.CreatedAt = createdAt.Format(time.RFC3339)
		taxClasses = append(taxClasses, tc)
	}
	return taxClasses, nil
}

func (r *Repository) GetTaxClassIDByName(ctx context.Context, name string) (int, error) {
	var id int
	err := r.db.QueryRow(ctx, "SELECT id FROM tax_classes WHERE name = $1 AND is_active = true", name).Scan(&id)
	return id, err
}

// Unit of measure operations
func (r *Repository) GetUnitOfMeasureByID(ctx context.Context, id int) (*UnitOfMeasure, error) {
	var uom UnitOfMeasure
	var createdAt time.Time
	err := r.db.QueryRow(ctx, "SELECT id, code, name, COALESCE(description,''), is_active, created_at FROM units_of_measure WHERE id = $1", id).Scan(
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

func (r *Repository) GetAllUnitsOfMeasure(ctx context.Context) ([]UnitOfMeasure, error) {
	rows, err := r.db.Query(ctx, "SELECT id, code, name, COALESCE(description,''), is_active, created_at FROM units_of_measure WHERE is_active = true ORDER BY code")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var uoms []UnitOfMeasure
	for rows.Next() {
		var uom UnitOfMeasure
		var createdAt time.Time
		if err := rows.Scan(&uom.ID, &uom.Code, &uom.Name, &uom.Description, &uom.IsActive, &createdAt); err != nil {
			return nil, err
		}
		uom.CreatedAt = createdAt.Format(time.RFC3339)
		uoms = append(uoms, uom)
	}
	return uoms, nil
}

func (r *Repository) GetUnitOfMeasureIDByCode(ctx context.Context, code string) (int, error) {
	var id int
	err := r.db.QueryRow(ctx, "SELECT id FROM units_of_measure WHERE code = $1 AND is_active = true", code).Scan(&id)
	return id, err
}

func (r *Repository) CreateUnitOfMeasure(ctx context.Context, uom *UnitOfMeasure) error {
	var createdAt time.Time
	err := r.db.QueryRow(ctx, `
		INSERT INTO units_of_measure (code, name, description, is_active) 
		VALUES ($1, $2, $3, $4) 
		RETURNING id, created_at
	`, uom.Code, uom.Name, uom.Description, uom.IsActive).Scan(&uom.ID, &createdAt)
	if err != nil {
		return err
	}
	uom.CreatedAt = createdAt.Format(time.RFC3339)
	return nil
}

func (r *Repository) UpdateUnitOfMeasure(ctx context.Context, uom *UnitOfMeasure) error {
	_, err := r.db.Exec(ctx, `
		UPDATE units_of_measure SET code = $1, name = $2, description = $3, is_active = $4
		WHERE id = $5
	`, uom.Code, uom.Name, uom.Description, uom.IsActive, uom.ID)
	return err
}

func (r *Repository) DeleteUnitOfMeasure(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, "DELETE FROM units_of_measure WHERE id = $1", id)
	return err
}

func (r *Repository) GetAllUnitsOfMeasureForExport(ctx context.Context) ([]UnitOfMeasure, error) {
	rows, err := r.db.Query(ctx, "SELECT id, code, name, COALESCE(description,''), is_active, created_at FROM units_of_measure ORDER BY code")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var uoms []UnitOfMeasure
	for rows.Next() {
		var uom UnitOfMeasure
		var createdAt time.Time
		if err := rows.Scan(&uom.ID, &uom.Code, &uom.Name, &uom.Description, &uom.IsActive, &createdAt); err != nil {
			return nil, err
		}
		uom.CreatedAt = createdAt.Format(time.RFC3339)
		uoms = append(uoms, uom)
	}
	return uoms, nil
}

func (r *Repository) BulkUpsertUnitsOfMeasure(ctx context.Context, records []UnitOfMeasureImportRow) importutil.ImportResult {
	result := importutil.ImportResult{Errors: []string{}}

	for _, rec := range records {
		if rec.Code == "" {
			result.AddError(rec.Row, "Code is required")
			continue
		}
		if rec.Name == "" {
			result.AddError(rec.Row, "Name is required")
			continue
		}

		var existingID int
		err := r.db.QueryRow(ctx, "SELECT id FROM units_of_measure WHERE code = $1", rec.Code).Scan(&existingID)
		isUpdate := err == nil

		_, err = r.db.Exec(ctx, `
			INSERT INTO units_of_measure (code, name, description, is_active)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (code) DO UPDATE SET
				name = EXCLUDED.name,
				description = EXCLUDED.description,
				is_active = EXCLUDED.is_active
		`, rec.Code, rec.Name, rec.Description, rec.IsActive)

		if err != nil {
			result.AddError(rec.Row, fmt.Sprintf("failed to upsert: %v", err))
			continue
		}

		if isUpdate {
			result.Updated++
		} else {
			result.Inserted++
		}
	}

	return result
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
	w.CreatedAt = createdAt.Format(time.RFC3339)
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
		w.CreatedAt = createdAt.Format(time.RFC3339)
		warehouses = append(warehouses, w)
	}
	return warehouses, nil
}

func (r *Repository) GetAllProductsForExport(ctx context.Context) ([]Product, error) {
	rows, err := r.db.Query(ctx, `
		SELECT v.id, v.sku, v.name, v.barcode, v.category_id, v.category_name, v.price, v.cost, v.stock, v.status,
		       v.store_id, v.brand_id, v.brand_name, v.unit_of_measure_id, v.unit_of_measure, v.weight_grams, v.description,
		       v.tax_class_id, v.tax_rate,
		       v.created_at, v.updated_at
		FROM v_products_full v
		ORDER BY v.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		var barcodeVal sql.NullString
		var categoryIDVal, storeIDVal, brandIDVal, unitOfMeasureIDVal, weightGramsVal sql.NullInt64
		var taxClassIDVal sql.NullInt64
		var taxRateVal sql.NullFloat64
		var categoryName, brandName, unitOfMeasure, descriptionVal sql.NullString
		var createdAt, updatedAt time.Time

		err = rows.Scan(&p.ID, &p.SKU, &p.Name, &barcodeVal, &categoryIDVal, &categoryName, &p.Price, &p.Cost, &p.Stock, &p.Status,
			&storeIDVal, &brandIDVal, &brandName, &unitOfMeasureIDVal, &unitOfMeasure, &weightGramsVal, &descriptionVal,
			&taxClassIDVal, &taxRateVal,
			&createdAt, &updatedAt)
		if err != nil {
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
		if taxClassIDVal.Valid {
			v := int(taxClassIDVal.Int64)
			p.TaxClassID = &v
		}
		if taxRateVal.Valid {
			v := taxRateVal.Float64
			p.TaxRate = &v
		}
		if storeIDVal.Valid {
			v := int(storeIDVal.Int64)
			p.StoreID = &v
		}
		p.CreatedAt = createdAt.In(jakartaLoc).Format(time.RFC3339)
		p.UpdatedAt = updatedAt.In(jakartaLoc).Format(time.RFC3339)
		products = append(products, p)
	}
	return products, nil
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

func (r *Repository) GetOrCreateBrandIDByName(ctx context.Context, name string) (int, error) {
	id, err := r.GetBrandIDByName(ctx, name)
	if err == nil {
		return id, nil
	}
	var brandID int
	err = r.db.QueryRow(ctx, `
		INSERT INTO brands (name, description, is_active)
		VALUES ($1, '', true)
		RETURNING id
	`, name).Scan(&brandID)
	if err != nil {
		return 0, fmt.Errorf("failed to auto-create brand: %w", err)
	}
	return brandID, nil
}

func (r *Repository) GetOrCreateUnitOfMeasureIDByCode(ctx context.Context, code string) (int, error) {
	id, err := r.GetUnitOfMeasureIDByCode(ctx, code)
	if err == nil {
		return id, nil
	}
	var uomID int
	err = r.db.QueryRow(ctx, `
		INSERT INTO units_of_measure (code, name, description, is_active)
		VALUES ($1, $1, '', true)
		RETURNING id
	`, code).Scan(&uomID)
	if err != nil {
		return 0, fmt.Errorf("failed to auto-create unit of measure: %w", err)
	}
	return uomID, nil
}

func (r *Repository) BulkUpsertProduct(ctx context.Context, p ProductImportPayload) (inserted bool, err error) {
	var existingID int
	err = r.db.QueryRow(ctx, "SELECT id FROM products WHERE sku = $1 AND deleted_at IS NULL", p.SKU).Scan(&existingID)
	isUpdate := err == nil

	var barcode interface{}
	if p.Barcode != nil {
		barcode = *p.Barcode
	} else {
		barcode = nil
	}
	var categoryID interface{}
	if p.CategoryID != nil {
		categoryID = *p.CategoryID
	} else {
		categoryID = nil
	}
	var brandID interface{}
	if p.BrandID != nil {
		brandID = *p.BrandID
	} else {
		brandID = nil
	}
	var uomID interface{}
	if p.UnitOfMeasureID != nil {
		uomID = *p.UnitOfMeasureID
	} else {
		uomID = nil
	}
	var weightGrams interface{}
	if p.WeightGrams != nil {
		weightGrams = *p.WeightGrams
	} else {
		weightGrams = nil
	}
	var description interface{}
	if p.Description != nil {
		description = *p.Description
	} else {
		description = nil
	}

	if isUpdate {
		_, err = r.db.Exec(ctx, `
			UPDATE products SET name = $1, barcode = $2, category_id = $3, brand_id = $4,
			       price = $5, cost = $6, status = $7, unit_of_measure_id = $8,
			       weight_grams = $9, description = $10, updated_at = NOW()
			WHERE id = $11 AND deleted_at IS NULL
		`, p.Name, barcode, categoryID, brandID, p.Price, p.Cost, p.Status, uomID,
			weightGrams, description, existingID)
		if err != nil {
			return false, err
		}
		return false, nil
	}

	var createdTime time.Time
	err = r.db.QueryRow(ctx, `
		INSERT INTO products (sku, name, barcode, category_id, price, cost, stock, status,
		                    brand_id, description, weight_grams, unit_of_measure_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at
	`, p.SKU, p.Name, barcode, categoryID, p.Price, p.Cost, p.Stock, p.Status,
		brandID, description, weightGrams, uomID).Scan(&existingID, &createdTime)
	if err != nil {
		return false, err
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO product_stock (product_id, quantity) VALUES ($1, $2)
	`, existingID, p.Stock)
	if err != nil {
		return false, fmt.Errorf("failed to initialize stock: %w", err)
	}

	return true, nil
}
