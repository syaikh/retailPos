package repo

import (
	"context"
	"database/sql"
	"fmt"
	model "retailPos/internal/model"
	"strconv"
	"strings"
)

type ProductRepo struct {
	db *sql.DB
}

func NewProductRepo(db *sql.DB) *ProductRepo {
	return &ProductRepo{db: db}
}

func (r *ProductRepo) Create(p *model.Product) error {
	var query string
	var args []any
	if p.StoreID != nil {
		query = `INSERT INTO products (name, sku, barcode, price, stock, group_id, store_id) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, created_at, updated_at`
		args = []any{p.Name, p.SKU, p.Barcode, p.Price, p.Stock, p.GroupID, p.StoreID}
	} else {
		query = `INSERT INTO products (name, sku, barcode, price, stock, group_id) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at, updated_at`
		args = []any{p.Name, p.SKU, p.Barcode, p.Price, p.Stock, p.GroupID}
	}
	return r.db.QueryRow(query, args...).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (r *ProductRepo) GetAll(limit, offset int, search string, groupID *int, sortBy, sortDir string, maxStock *int, storeID *int) ([]model.Product, int, error) {
	// Base query
	query := `SELECT id, name, sku, barcode, price, stock, group_id, store_id, created_at, updated_at, deleted_at, restored_at FROM products WHERE deleted_at IS NULL`
	countQuery := `SELECT COUNT(*) FROM products WHERE deleted_at IS NULL`
	args := []any{}
	placeholderIdx := 1

	// Store filter
	if storeID != nil {
		filter := " AND store_id = $" + strconv.Itoa(placeholderIdx)
		query += filter
		countQuery += filter
		args = append(args, *storeID)
		placeholderIdx++
	}

	// Search filter
	if search != "" {
		filter := " AND (name ILIKE $" + strconv.Itoa(placeholderIdx) + " OR sku ILIKE $" + strconv.Itoa(placeholderIdx) + " OR barcode ILIKE $" + strconv.Itoa(placeholderIdx) + ")"
		query += filter
		countQuery += filter
		args = append(args, "%"+search+"%")
		placeholderIdx++
	}

	// Group filter
	if groupID != nil {
		filter := " AND group_id = $" + strconv.Itoa(placeholderIdx)
		query += filter
		countQuery += filter
		args = append(args, *groupID)
		placeholderIdx++
	}

	// Max Stock filter
	if maxStock != nil {
		filter := " AND stock < $" + strconv.Itoa(placeholderIdx)
		query += filter
		countQuery += filter
		args = append(args, *maxStock)
		placeholderIdx++
	}

	// Total count
	var total int
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Sorting
	validSortFields := map[string]string{
		"id":         "id",
		"name":       "name",
		"sku":        "sku",
		"barcode":    "barcode",
		"price":      "price",
		"stock":      "stock",
		"created_at": "created_at",
	}

	sortField, ok := validSortFields[sortBy]
	if !ok {
		sortField = "id"
	}
	if strings.ToLower(sortDir) != "desc" {
		sortDir = "ASC"
	} else {
		sortDir = "DESC"
	}
	query += fmt.Sprintf(" ORDER BY %s %s", sortField, sortDir)

	// Pagination
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var products []model.Product
	for rows.Next() {
		var p model.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.SKU, &p.Barcode, &p.Price, &p.Stock, &p.GroupID, &p.StoreID, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt, &p.RestoredAt); err != nil {
			return nil, 0, err
		}
		products = append(products, p)
	}
	return products, total, nil
}

func (r *ProductRepo) GetBySKU(sku string, storeID *int) (*model.Product, error) {
	query := `SELECT id, name, sku, barcode, price, stock, group_id, store_id, created_at, updated_at, deleted_at, restored_at FROM products WHERE sku = $1 AND deleted_at IS NULL`
	args := []any{sku}

	if storeID != nil {
		query += " AND store_id = $2"
		args = append(args, *storeID)
	}

	var p model.Product
	if err := r.db.QueryRow(query, args...).Scan(&p.ID, &p.Name, &p.SKU, &p.Barcode, &p.Price, &p.Stock, &p.GroupID, &p.StoreID, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt, &p.RestoredAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *ProductRepo) GetByBarcode(barcode string, storeID *int) (*model.Product, error) {
	query := `SELECT id, name, sku, barcode, price, stock, group_id, store_id, created_at, updated_at, deleted_at, restored_at FROM products WHERE barcode = $1 AND deleted_at IS NULL`
	args := []any{barcode}

	if storeID != nil {
		query += " AND store_id = $2"
		args = append(args, *storeID)
	}

	var p model.Product
	if err := r.db.QueryRow(query, args...).Scan(&p.ID, &p.Name, &p.SKU, &p.Barcode, &p.Price, &p.Stock, &p.GroupID, &p.StoreID, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt, &p.RestoredAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *ProductRepo) GetByID(id int, storeID *int) (*model.Product, error) {
	query := `SELECT id, name, sku, barcode, price, stock, group_id, store_id, created_at, updated_at, deleted_at, restored_at FROM products WHERE id = $1 AND deleted_at IS NULL`
	args := []any{id}

	if storeID != nil {
		query += " AND store_id = $2"
		args = append(args, *storeID)
	}

	var p model.Product
	if err := r.db.QueryRow(query, args...).Scan(&p.ID, &p.Name, &p.SKU, &p.Barcode, &p.Price, &p.Stock, &p.GroupID, &p.StoreID, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt, &p.RestoredAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *ProductRepo) GetBySKUWithDeleted(sku string, storeID *int) (*model.Product, error) {
	query := `SELECT id, name, sku, barcode, price, stock, group_id, store_id, created_at, updated_at, deleted_at, restored_at FROM products WHERE sku = $1`
	args := []any{sku}

	if storeID != nil {
		query += " AND store_id = $2"
		args = append(args, *storeID)
	}

	var p model.Product
	if err := r.db.QueryRow(query, args...).Scan(&p.ID, &p.Name, &p.SKU, &p.Barcode, &p.Price, &p.Stock, &p.GroupID, &p.StoreID, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt, &p.RestoredAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *ProductRepo) GetByBarcodeWithDeleted(barcode string, storeID *int) (*model.Product, error) {
	query := `SELECT id, name, sku, barcode, price, stock, group_id, store_id, created_at, updated_at, deleted_at, restored_at FROM products WHERE barcode = $1`
	args := []any{barcode}

	if storeID != nil {
		query += " AND store_id = $2"
		args = append(args, *storeID)
	}

	var p model.Product
	if err := r.db.QueryRow(query, args...).Scan(&p.ID, &p.Name, &p.SKU, &p.Barcode, &p.Price, &p.Stock, &p.GroupID, &p.StoreID, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt, &p.RestoredAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *ProductRepo) Restore(p *model.Product) error {
	query := `UPDATE products SET name = $1, price = $2, stock = $3, group_id = $4, deleted_at = NULL, restored_at = NOW(), updated_at = NOW() WHERE id = $5 RETURNING restored_at, updated_at`
	return r.db.QueryRow(query, p.Name, p.Price, p.Stock, p.GroupID, p.ID).Scan(&p.RestoredAt, &p.UpdatedAt)
}

func (r *ProductRepo) UpdateStock(id int, delta int) error {
	query := `UPDATE products SET stock = stock + $1, updated_at = NOW() WHERE id = $2 AND stock + $1 >= 0`
	_, err := r.db.Exec(query, delta, id)
	return err
}

func (r *ProductRepo) Update(p *model.Product, storeID *int) error {
	query := `UPDATE products SET name = $1, sku = $2, barcode = $3, price = $4, stock = $5, group_id = $6, store_id = $7, updated_at = NOW() WHERE id = $8`
	_, err := r.db.Exec(query, p.Name, p.SKU, p.Barcode, p.Price, p.Stock, p.GroupID, p.StoreID, p.ID)
	return err
}

func (r *ProductRepo) Delete(id int, storeID *int) error {
	query := `UPDATE products SET deleted_at = NOW() WHERE id = $1`
	args := []any{id}

	if storeID != nil {
		query += " AND store_id = $2"
		args = append(args, *storeID)
	}

	_, err := r.db.Exec(query, args...)
	return err
}

func (r *ProductRepo) GetAllForExport(ctx context.Context) (<-chan model.Product, error) {
	query := `SELECT id, name, sku, barcode, price, stock, group_id, created_at, updated_at, deleted_at, restored_at 
              FROM products 
              WHERE deleted_at IS NULL`
	args := []any{}

	query += " ORDER BY created_at ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	productChan := make(chan model.Product, 100)

	go func() {
		defer rows.Close()
		defer close(productChan)
		for rows.Next() {
			var p model.Product
			if err := rows.Scan(&p.ID, &p.Name, &p.SKU, &p.Barcode, &p.Price, &p.Stock, &p.GroupID, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt, &p.RestoredAt); err != nil {
				continue
			}
			select {
			case productChan <- p:
			case <-ctx.Done():
				return
			}
		}
	}()

	return productChan, nil
}
