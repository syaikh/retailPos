package customergroup

import (
	"context"
	"fmt"
	"strings"
	"time"

	"retail-pos-system/internal/shared"
)

const selectColumns = `cg.id, cg.name, COALESCE(cg.description, ''), cg.is_active, COALESCE(cg.color, ''),
	COALESCE(cc.cnt, 0), cg.created_at, cg.updated_at`

const baseJoin = `FROM customer_groups cg
	LEFT JOIN (SELECT customer_group_id, COUNT(*) AS cnt FROM customers GROUP BY customer_group_id) cc ON cc.customer_group_id = cg.id`

type Repository struct {
	db shared.DBPool
}

func NewRepository(db shared.DBPool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) scanGroup(scanner interface{ Scan(...interface{}) error }) (*CustomerGroup, error) {
	var cg CustomerGroup
	var createdAt, updatedAt time.Time
	if err := scanner.Scan(&cg.ID, &cg.Name, &cg.Description, &cg.IsActive, &cg.Color, &cg.CustomerCount, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	cg.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	cg.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &cg, nil
}

func (r *Repository) GetAll(ctx context.Context, limit, offset int, search string, isActive *bool, hasCustomers *bool) ([]CustomerGroup, int, error) {
	where := "1=1"
	args := []interface{}{}
	argIdx := 1

	if search != "" {
		where += fmt.Sprintf(" AND (LOWER(cg.name) LIKE LOWER($%d) OR LOWER(cg.description) LIKE LOWER($%d))", argIdx, argIdx)
		args = append(args, "%"+strings.ToLower(search)+"%")
		argIdx++
	}
	if isActive != nil {
		where += fmt.Sprintf(" AND cg.is_active = $%d", argIdx)
		args = append(args, *isActive)
		argIdx++
	}
	if hasCustomers != nil {
		if *hasCustomers {
			where += " AND cc.cnt > 0"
		} else {
			where += " AND (cc.cnt = 0 OR cc.cnt IS NULL)"
		}
	}

	var total int
	countJoin := ""
	if hasCustomers != nil {
		countJoin = " LEFT JOIN (SELECT customer_group_id, COUNT(*) AS cnt FROM customers GROUP BY customer_group_id) cc ON cc.customer_group_id = cg.id"
	}
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM customer_groups cg%s WHERE %s", countJoin, where)
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count customer groups: %w", err)
	}

	query := fmt.Sprintf(`SELECT %s %s WHERE %s ORDER BY cg.id ASC LIMIT $%d OFFSET $%d`,
		selectColumns, baseJoin, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list customer groups: %w", err)
	}
	defer rows.Close()

	var groups []CustomerGroup
	for rows.Next() {
		cg, err := r.scanGroup(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan customer group: %w", err)
		}
		groups = append(groups, *cg)
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
	query := fmt.Sprintf(`SELECT %s %s WHERE cg.id = $1`, selectColumns, baseJoin)
	return r.scanGroup(r.db.QueryRow(ctx, query, id))
}

func (r *Repository) GetByName(ctx context.Context, name string) (*CustomerGroup, error) {
	query := fmt.Sprintf(`SELECT %s %s WHERE LOWER(cg.name) = LOWER($1)`, selectColumns, baseJoin)
	return r.scanGroup(r.db.QueryRow(ctx, query, name))
}

func (r *Repository) Create(ctx context.Context, cg *CustomerGroup) error {
	var createdAt, updatedAt time.Time
	color := cg.Color
	if color == "" {
		color = "#6C5CE7"
	}
	err := r.db.QueryRow(ctx, `
		INSERT INTO customer_groups (name, description, is_active, color)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`,
		cg.Name, cg.Description, cg.IsActive, color,
	).Scan(&cg.ID, &createdAt, &updatedAt)
	if err != nil {
		return fmt.Errorf("create customer group: %w", err)
	}
	cg.Color = color
	cg.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	cg.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return nil
}

func (r *Repository) Update(ctx context.Context, cg *CustomerGroup) error {
	_, err := r.db.Exec(ctx, `
		UPDATE customer_groups
		SET name = $1, description = $2, is_active = $3, color = $4, updated_at = NOW()
		WHERE id = $5`,
		cg.Name, cg.Description, cg.IsActive, cg.Color, cg.ID)
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
	query := fmt.Sprintf(`SELECT %s %s WHERE cg.is_active = true ORDER BY cg.name ASC`, selectColumns, baseJoin)
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list active customer groups: %w", err)
	}
	defer rows.Close()

	var groups []CustomerGroup
	for rows.Next() {
		cg, err := r.scanGroup(rows)
		if err != nil {
			return nil, fmt.Errorf("scan customer group: %w", err)
		}
		groups = append(groups, *cg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate customer groups: %w", err)
	}
	if groups == nil {
		groups = []CustomerGroup{}
	}
	return groups, nil
}

func (r *Repository) GetAllForExport(ctx context.Context) ([]CustomerGroup, error) {
	query := fmt.Sprintf(`SELECT %s %s ORDER BY cg.id ASC`, selectColumns, baseJoin)
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("export customer groups: %w", err)
	}
	defer rows.Close()

	var groups []CustomerGroup
	for rows.Next() {
		cg, err := r.scanGroup(rows)
		if err != nil {
			return nil, fmt.Errorf("scan customer group: %w", err)
		}
		groups = append(groups, *cg)
	}
	return groups, rows.Err()
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

func (r *Repository) BulkUpdate(ctx context.Context, ids []int, isActive bool) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	query := `UPDATE customer_groups SET is_active = $1, updated_at = NOW() WHERE id = ANY($2)`
	result, err := r.db.Exec(ctx, query, isActive, ids)
	if err != nil {
		return 0, fmt.Errorf("bulk update customer groups: %w", err)
	}
	rowsAffected := result.RowsAffected()
	return int(rowsAffected), nil
}

func (r *Repository) BulkDelete(ctx context.Context, ids []int) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result, err := r.db.Exec(ctx, `DELETE FROM customer_groups WHERE id = ANY($1)`, ids)
	if err != nil {
		return 0, fmt.Errorf("bulk delete customer groups: %w", err)
	}
	rowsAffected := result.RowsAffected()
	return int(rowsAffected), nil
}

type BulkUpsertResult struct {
	Inserted int
	Updated  int
	Errors   []string
}

func (r *Repository) BulkUpsertCustomerGroups(ctx context.Context, records []CustomerGroupImportRow) BulkUpsertResult {
	result := BulkUpsertResult{}
	for _, row := range records {
		existing, err := r.GetByName(ctx, row.Name)
		if err != nil {
			cg := &CustomerGroup{
				Name:        row.Name,
				Description: row.Description,
				IsActive:    row.IsActive,
				Color:       "#6C5CE7",
			}
			if err := r.Create(ctx, cg); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("row %d: %v", row.Row, err))
				continue
			}
			result.Inserted++
			continue
		}
		existing.Description = row.Description
		existing.IsActive = row.IsActive
		if err := r.Update(ctx, existing); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("row %d: %v", row.Row, err))
			continue
		}
		result.Updated++
	}
	return result
}
