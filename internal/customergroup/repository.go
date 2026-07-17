package customergroup

import (
	"context"
	"fmt"
	"strings"
	"time"

	"retail-pos-system/internal/shared"
)

type Repository struct {
	db shared.DBPool
}

func NewRepository(db shared.DBPool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetAll(ctx context.Context, limit, offset int, search string, isActive *bool) ([]CustomerGroup, int, error) {
	where := "1=1"
	args := []interface{}{}
	argIdx := 1

	if search != "" {
		where += fmt.Sprintf(" AND LOWER(name) LIKE LOWER($%d)", argIdx)
		args = append(args, "%"+strings.ToLower(search)+"%")
		argIdx++
	}
	if isActive != nil {
		where += fmt.Sprintf(" AND is_active = $%d", argIdx)
		args = append(args, *isActive)
		argIdx++
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM customer_groups WHERE %s", where)
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count customer groups: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, name, COALESCE(description, ''), is_active, created_at, updated_at
		FROM customer_groups
		WHERE %s
		ORDER BY id ASC
		LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list customer groups: %w", err)
	}
	defer rows.Close()

	var groups []CustomerGroup
	for rows.Next() {
		var cg CustomerGroup
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&cg.ID, &cg.Name, &cg.Description, &cg.IsActive, &createdAt, &updatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan customer group: %w", err)
		}
		cg.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		cg.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		groups = append(groups, cg)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate customer groups: %w", err)
	}
	if groups == nil {
		groups = []CustomerGroup{}
	}
	return groups, total, nil
}

func (r *Repository) GetByID(ctx context.Context, id int) (*CustomerGroup, error) {
	var cg CustomerGroup
	var createdAt, updatedAt time.Time
	err := r.db.QueryRow(ctx, `
		SELECT id, name, COALESCE(description, ''), is_active, created_at, updated_at
		FROM customer_groups WHERE id = $1`, id).Scan(
		&cg.ID, &cg.Name, &cg.Description, &cg.IsActive, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("get customer group by id: %w", err)
	}
	cg.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	cg.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &cg, nil
}

func (r *Repository) Create(ctx context.Context, cg *CustomerGroup) error {
	var createdAt, updatedAt time.Time
	err := r.db.QueryRow(ctx, `
		INSERT INTO customer_groups (name, description, is_active)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at`,
		cg.Name, cg.Description, cg.IsActive,
	).Scan(&cg.ID, &createdAt, &updatedAt)
	if err != nil {
		return fmt.Errorf("create customer group: %w", err)
	}
	cg.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	cg.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return nil
}

func (r *Repository) Update(ctx context.Context, cg *CustomerGroup) error {
	_, err := r.db.Exec(ctx, `
		UPDATE customer_groups
		SET name = $1, description = $2, is_active = $3, updated_at = NOW()
		WHERE id = $4`,
		cg.Name, cg.Description, cg.IsActive, cg.ID)
	if err != nil {
		return fmt.Errorf("update customer group: %w", err)
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, `DELETE FROM customer_groups WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete customer group: %w", err)
	}
	return nil
}

func (r *Repository) GetAllActive(ctx context.Context) ([]CustomerGroup, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, COALESCE(description, ''), is_active, created_at, updated_at
		FROM customer_groups WHERE is_active = true
		ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("list active customer groups: %w", err)
	}
	defer rows.Close()

	var groups []CustomerGroup
	for rows.Next() {
		var cg CustomerGroup
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&cg.ID, &cg.Name, &cg.Description, &cg.IsActive, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan customer group: %w", err)
		}
		cg.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		cg.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		groups = append(groups, cg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate customer groups: %w", err)
	}
	if groups == nil {
		groups = []CustomerGroup{}
	}
	return groups, nil
}

func (r *Repository) NameExists(ctx context.Context, name string, excludeID int) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM customer_groups WHERE LOWER(name) = LOWER($1) AND id != $2`,
		name, excludeID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check name exists: %w", err)
	}
	return count > 0, nil
}

func (r *Repository) GetByName(ctx context.Context, name string) (*CustomerGroup, error) {
	var cg CustomerGroup
	var createdAt, updatedAt time.Time
	err := r.db.QueryRow(ctx, `
		SELECT id, name, COALESCE(description, ''), is_active, created_at, updated_at
		FROM customer_groups WHERE LOWER(name) = LOWER($1)`, name).Scan(
		&cg.ID, &cg.Name, &cg.Description, &cg.IsActive, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("get customer group by name: %w", err)
	}
	cg.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	cg.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &cg, nil
}
