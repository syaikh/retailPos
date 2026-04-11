package repo

import (
	"database/sql"
	model "retailPos/internal/model"
)

type ProductGroupRepo struct {
	db *sql.DB
}

func NewProductGroupRepo(db *sql.DB) *ProductGroupRepo {
	return &ProductGroupRepo{db: db}
}

func (r *ProductGroupRepo) GetAll() ([]model.ProductGroup, error) {
	query := `SELECT id, name, description, created_at FROM product_groups ORDER BY id DESC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []model.ProductGroup
	for rows.Next() {
		var g model.ProductGroup
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.CreatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	
	if groups == nil {
		groups = make([]model.ProductGroup, 0)
	}
	return groups, nil
}

func (r *ProductGroupRepo) Create(g *model.ProductGroup) error {
	query := `INSERT INTO product_groups (name, description) VALUES ($1, $2) RETURNING id, created_at`
	return r.db.QueryRow(query, g.Name, g.Description).Scan(&g.ID, &g.CreatedAt)
}

func (r *ProductGroupRepo) Update(g *model.ProductGroup) error {
	query := `UPDATE product_groups SET name = $1, description = $2 WHERE id = $3`
	_, err := r.db.Exec(query, g.Name, g.Description, g.ID)
	return err
}

func (r *ProductGroupRepo) Delete(id int) error {
	// Database foreign key constraint without CASCADE will naturally prevent deletion
	// if there are products still linking to this group.
	query := `DELETE FROM product_groups WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}
