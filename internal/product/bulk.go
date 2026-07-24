package product

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (r *Repository) BulkUpdateProductStatus(ctx context.Context, ids []int, status string, storeID *int) error {
	if len(ids) == 0 {
		return nil
	}
	query := "UPDATE products SET status = $1 WHERE id = ANY($2)"
	args := []interface{}{status, ids}
	if storeID != nil {
		query += fmt.Sprintf(" AND store_id = $%d", len(args)+1)
		args = append(args, *storeID)
	}
	_, err := r.db.Exec(ctx, query, args...)
	return err
}

func (r *Repository) BulkUpsertProduct(ctx context.Context, p ProductImportPayload) (inserted bool, err error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var existingID int
	err = tx.QueryRow(ctx, "SELECT id FROM products WHERE sku = $1 AND deleted_at IS NULL", p.SKU).Scan(&existingID)
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
		_, err = tx.Exec(ctx, `
			UPDATE products SET name = $1, barcode = $2, category_id = $3, brand_id = $4,
			       price = $5, cost = $6, status = $7, unit_of_measure_id = $8,
			       weight_grams = $9, description = $10, updated_at = NOW()
			WHERE id = $11 AND deleted_at IS NULL
		`, p.Name, barcode, categoryID, brandID, p.Price, p.Cost, p.Status, uomID,
			weightGrams, description, existingID)
		if err != nil {
			return false, err
		}
		if err := tx.Commit(ctx); err != nil {
			return false, fmt.Errorf("commit transaction: %w", err)
		}
		return false, nil
	}

	var createdTime time.Time
	err = tx.QueryRow(ctx, `
		INSERT INTO products (sku, name, barcode, category_id, price, cost, stock, status,
		                    brand_id, description, weight_grams, unit_of_measure_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at
	`, p.SKU, p.Name, barcode, categoryID, p.Price, p.Cost, p.Stock, p.Status,
		brandID, description, weightGrams, uomID).Scan(&existingID, &createdTime)
	if err != nil {
		return false, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO product_stock (product_id, quantity) VALUES ($1, $2)
	`, existingID, p.Stock)
	if err != nil {
		return false, fmt.Errorf("failed to initialize stock: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit transaction: %w", err)
	}

	return true, nil
}

func (r *Repository) BulkInsertProducts(ctx context.Context, payloads []ProductImportPayload) (int, error) {
	if len(payloads) == 0 {
		return 0, nil
	}

	skus := make([]string, len(payloads))
	for i, p := range payloads {
		skus[i] = p.SKU
	}

	existingMap := make(map[string]int, len(payloads))
	rows, err := r.db.Query(ctx, "SELECT id, sku FROM products WHERE sku = ANY($1) AND deleted_at IS NULL", skus)
	if err != nil {
		return 0, fmt.Errorf("batch lookup: %w", err)
	}
	for rows.Next() {
		var id int
		var sku string
		if err := rows.Scan(&id, &sku); err == nil {
			existingMap[sku] = id
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("batch lookup rows iteration: %w", err)
	}
	rows.Close()

	var newPayloads []ProductImportPayload
	for _, p := range payloads {
		if _, exists := existingMap[p.SKU]; !exists {
			newPayloads = append(newPayloads, p)
		}
	}

	if len(newPayloads) == 0 {
		return 0, nil
	}

	valueStrings := make([]string, 0, len(newPayloads))
	valueArgs := make([]interface{}, 0, len(newPayloads)*13)
	for _, p := range newPayloads {
		offset := len(valueArgs)
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			offset+1, offset+2, offset+3, offset+4, offset+5, offset+6, offset+7, offset+8, offset+9, offset+10, offset+11, offset+12, offset+13))

		var barcode interface{}
		if p.Barcode != nil {
			barcode = *p.Barcode
		}
		var categoryID interface{}
		if p.CategoryID != nil {
			categoryID = *p.CategoryID
		}
		var brandID interface{}
		if p.BrandID != nil {
			brandID = *p.BrandID
		}
		var uomID interface{}
		if p.UnitOfMeasureID != nil {
			uomID = *p.UnitOfMeasureID
		}
		var weightGrams interface{}
		if p.WeightGrams != nil {
			weightGrams = *p.WeightGrams
		}
		var description interface{}
		if p.Description != nil {
			description = *p.Description
		}
		var storeID interface{}
		if p.StoreID != nil {
			storeID = *p.StoreID
		}

		valueArgs = append(valueArgs, p.SKU, p.Name, barcode, categoryID, p.Price, p.Cost, p.Stock, p.Status,
			brandID, description, weightGrams, uomID, storeID)
	}

	query := fmt.Sprintf(`
		INSERT INTO products (sku, name, barcode, category_id, price, cost, stock, status,
		                     brand_id, description, weight_grams, unit_of_measure_id, store_id)
		VALUES %s
		RETURNING id
	`, strings.Join(valueStrings, ", "))

	var newIDs []int
	rows, err = r.db.Query(ctx, query, valueArgs...)
	if err != nil {
		return 0, fmt.Errorf("batch insert: %w", err)
	}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err == nil {
			newIDs = append(newIDs, id)
		}
	}
	rows.Close()
	if rows.Err() != nil {
		return 0, fmt.Errorf("batch insert rows: %w", rows.Err())
	}

	if len(newIDs) > 0 {
		stockStrings := make([]string, 0, len(newIDs))
		stockArgs := make([]interface{}, 0, len(newIDs)*2)
		for i, id := range newIDs {
			offset := len(stockArgs)
			stockStrings = append(stockStrings, fmt.Sprintf("($%d, $%d)", offset+1, offset+2))
			stockArgs = append(stockArgs, id, newPayloads[i].Stock)
		}
		stockQuery := fmt.Sprintf(`
			INSERT INTO product_stock (product_id, quantity)
			VALUES %s
			ON CONFLICT ON CONSTRAINT uq_product_stock DO UPDATE SET quantity = EXCLUDED.quantity
		`, strings.Join(stockStrings, ", "))
		_, err = r.db.Exec(ctx, stockQuery, stockArgs...)
		if err != nil {
			return len(newIDs), fmt.Errorf("batch insert stock: %w", err)
		}
	}

	return len(newIDs), nil
}

func (r *Repository) BulkUpdateProducts(ctx context.Context, payloads []ProductImportPayload) (int, error) {
	if len(payloads) == 0 {
		return 0, nil
	}

	skus := make([]string, len(payloads))
	for i, p := range payloads {
		skus[i] = p.SKU
	}

	existingMap := make(map[string]int, len(payloads))
	rows, err := r.db.Query(ctx, "SELECT id, sku FROM products WHERE sku = ANY($1) AND deleted_at IS NULL", skus)
	if err != nil {
		return 0, fmt.Errorf("batch lookup: %w", err)
	}
	for rows.Next() {
		var id int
		var sku string
		if err := rows.Scan(&id, &sku); err == nil {
			existingMap[sku] = id
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("batch lookup rows iteration: %w", err)
	}
	rows.Close()

	type updateItem struct {
		id      int
		payload ProductImportPayload
	}
	var updates []updateItem
	for _, p := range payloads {
		if id, ok := existingMap[p.SKU]; ok {
			updates = append(updates, updateItem{id: id, payload: p})
		}
	}

	if len(updates) == 0 {
		return 0, nil
	}

	valueStrings := make([]string, 0, len(updates))
	valueArgs := make([]interface{}, 0, len(updates)*11)
	for _, d := range updates {
		offset := len(valueArgs)
		p := d.payload
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d::int, $%d::int, $%d::int, $%d::int, $%d, $%d::int, $%d::int, $%d, $%d::int)",
			offset+1, offset+2, offset+3, offset+4, offset+5, offset+6, offset+7, offset+8, offset+9, offset+10, offset+11))

		var barcode interface{}
		if p.Barcode != nil {
			barcode = *p.Barcode
		}
		var categoryID interface{}
		if p.CategoryID != nil {
			categoryID = *p.CategoryID
		}
		var brandID interface{}
		if p.BrandID != nil {
			brandID = *p.BrandID
		}
		var uomID interface{}
		if p.UnitOfMeasureID != nil {
			uomID = *p.UnitOfMeasureID
		}
		var weightGrams interface{}
		if p.WeightGrams != nil {
			weightGrams = *p.WeightGrams
		}
		var description interface{}
		if p.Description != nil {
			description = *p.Description
		}

		valueArgs = append(valueArgs, p.Name, barcode, categoryID, brandID, p.Price, p.Cost, p.Status,
			uomID, weightGrams, description, d.id)
	}

	query := fmt.Sprintf(`
		UPDATE products SET
			name = data.name,
			barcode = data.barcode,
			category_id = data.category_id::int,
			brand_id = data.brand_id::int,
			price = data.price::int,
			cost = data.cost::int,
			status = data.status,
			unit_of_measure_id = data.uom_id::int,
			weight_grams = data.weight_grams::int,
			description = data.description,
			updated_at = NOW()
		FROM (VALUES %s) AS data(name, barcode, category_id, brand_id,
		                         price, cost, status, uom_id,
		                         weight_grams, description, id)
		WHERE products.id = data.id::int
	`, strings.Join(valueStrings, ", "))

	_, err = r.db.Exec(ctx, query, valueArgs...)
	if err != nil {
		return 0, fmt.Errorf("batch update: %w", err)
	}

	return len(updates), nil
}
