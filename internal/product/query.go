package product

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"retail-pos-system/internal/shared"
)

func (r *Repository) GetAllProducts(ctx context.Context, limit, offset int, search string, categoryIDs []int, sortBy, sortDir string, maxStock *int, storeID *int, status string) ([]Product, int, error) {
	var products []Product
	var total int

	query := `SELECT COUNT(*)
		FROM v_products_full v
		WHERE 1=1`
	args := []interface{}{}
	argIdx := 1
	if search != "" {
		query += fmt.Sprintf(" AND v.search_vector @@ plainto_tsquery('english', $%d)", argIdx)
		args = append(args, search)
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
		query2 += fmt.Sprintf(" AND v.search_vector @@ plainto_tsquery('english', $%d)", argIdx2)
		args2 = append(args2, search)
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
		p.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		p.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return products, total, nil
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
		if err == sql.ErrNoRows {
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
	p.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	p.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)

	return &p, nil
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
		p.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		p.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return products, nil
}
