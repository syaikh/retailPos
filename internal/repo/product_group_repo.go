package repo

import (
	"database/sql"
	"fmt"
	model "retailPos/internal/model"
	"strconv"
	"strings"
)

type ProductGroupRepo struct {
	db *sql.DB
}

func NewProductGroupRepo(db *sql.DB) *ProductGroupRepo {
	return &ProductGroupRepo{db: db}
}

func (r *ProductGroupRepo) GetAll(limit, offset int, search string, sortBy, sortDir string) ([]model.ProductGroup, int, error) {
	query := `SELECT id, name, description, created_at, (SELECT COUNT(*) FROM products p WHERE p.group_id = product_groups.id AND p.deleted_at IS NULL) as product_count FROM product_groups WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM product_groups WHERE 1=1`
	args := []any{}
	placeholderIdx := 1

	if search != "" {
		filter := " AND (name ILIKE $" + strconv.Itoa(placeholderIdx) + " OR description ILIKE $" + strconv.Itoa(placeholderIdx) + ")"
		query += filter
		countQuery += filter
		args = append(args, "%"+search+"%")
		placeholderIdx++
	}

	var total int
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	validSortFields := map[string]string{
		"id":   "id",
		"name": "name",
	}
	sortField, ok := validSortFields[sortBy]
	if !ok {
		sortField = "id"
	}
	if strings.ToLower(sortDir) != "asc" {
		sortDir = "DESC"
	} else {
		sortDir = "ASC"
	}
	query += fmt.Sprintf(" ORDER BY %s %s", sortField, sortDir)
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	groups := []model.ProductGroup{}
	for rows.Next() {
		var g model.ProductGroup
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.CreatedAt, &g.ProductCount); err != nil {
			return nil, 0, err
		}
		groups = append(groups, g)
	}
	return groups, total, nil
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
