package repo

import (
	"database/sql"
	model "retailPos/internal/model"
)

type ProductRepo struct {
	db *sql.DB
}

func NewProductRepo(db *sql.DB) *ProductRepo {
	return &ProductRepo{db: db}
}

func (r *ProductRepo) Create(p *model.Product) error {
	query := `INSERT INTO products (name, sku, price, stock, group_id) VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at, updated_at`
	return r.db.QueryRow(query, p.Name, p.SKU, p.Price, p.Stock, p.GroupID).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (r *ProductRepo) GetAll() ([]model.Product, error) {
	query := `SELECT id, name, sku, price, stock, group_id, created_at, updated_at FROM products WHERE deleted_at IS NULL`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []model.Product
	for rows.Next() {
		var p model.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.SKU, &p.Price, &p.Stock, &p.GroupID, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, nil
}

func (r *ProductRepo) GetBySKU(sku string) (*model.Product, error) {
	query := `SELECT id, name, sku, price, stock, group_id, created_at, updated_at FROM products WHERE sku = $1 AND deleted_at IS NULL`
	var p model.Product
	if err := r.db.QueryRow(query, sku).Scan(&p.ID, &p.Name, &p.SKU, &p.Price, &p.Stock, &p.GroupID, &p.CreatedAt, &p.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *ProductRepo) UpdateStock(id int, delta int) error {
	query := `UPDATE products SET stock = stock + $1, updated_at = NOW() WHERE id = $2 AND stock + $1 >= 0`
	_, err := r.db.Exec(query, delta, id)
	return err
}
