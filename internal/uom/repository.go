package uom

import (
	"context"
	"fmt"
	"strings"
	"time"

	"retail-pos-system/internal/shared"
	"retail-pos-system/pkg/cache"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ImportResult struct {
	Inserted int      `json:"inserted"`
	Updated  int      `json:"updated"`
	Errors   []string `json:"errors"`
}

func (r *ImportResult) AddError(row int, msg string) {
	r.Errors = append(r.Errors, fmt.Sprintf("row %d: %s", row, msg))
}

type Repository struct {
	db    *pgxpool.Pool
	cache *cache.Cache
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) SetCache(c *cache.Cache) {
	r.cache = c
}

func (r *Repository) GetByID(ctx context.Context, id int) (*UnitOfMeasure, error) {
	if r.cache != nil {
		key := fmt.Sprintf("uom:%d", id)
		if v, ok := r.cache.Get(key); ok {
			u := v.(UnitOfMeasure)
			return &u, nil
		}
	}
	var u UnitOfMeasure
	var createdAt time.Time
	err := r.db.QueryRow(ctx, "SELECT id, code, name, COALESCE(description,''), is_active, created_at FROM units_of_measure WHERE id = $1", id).Scan(
		&u.ID, &u.Code, &u.Name, &u.Description, &u.IsActive, &createdAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("unit of measure not found")
		}
		return nil, err
	}
	u.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	if r.cache != nil {
		r.cache.Set(fmt.Sprintf("uom:%d", id), u)
	}
	return &u, nil
}

func (r *Repository) GetAll(ctx context.Context) ([]UnitOfMeasure, error) {
	if r.cache != nil {
		if v, ok := r.cache.Get("uoms:all"); ok {
			return v.([]UnitOfMeasure), nil
		}
	}
	rows, err := r.db.Query(ctx, "SELECT id, code, name, COALESCE(description,''), is_active, created_at FROM units_of_measure WHERE is_active = true ORDER BY code")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var units []UnitOfMeasure
	for rows.Next() {
		var u UnitOfMeasure
		var createdAt time.Time
		if err := rows.Scan(&u.ID, &u.Code, &u.Name, &u.Description, &u.IsActive, &createdAt); err != nil {
			return nil, err
		}
		u.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		units = append(units, u)
	}
	if r.cache != nil && units != nil {
		r.cache.Set("uoms:all", units)
	}
	return units, nil
}

func (r *Repository) GetIDByCode(ctx context.Context, code string) (int, error) {
	var id int
	err := r.db.QueryRow(ctx, "SELECT id FROM units_of_measure WHERE code = $1 AND is_active = true", code).Scan(&id)
	return id, err
}

func (r *Repository) GetIDsByCodes(ctx context.Context, codes []string) (map[string]int, error) {
	if len(codes) == 0 {
		return nil, nil
	}
	rows, err := r.db.Query(ctx, "SELECT id, code FROM units_of_measure WHERE code = ANY($1) AND is_active = true", codes)
	if err != nil {
		return nil, fmt.Errorf("batch get UoM IDs: %w", err)
	}
	defer rows.Close()
	result := make(map[string]int, len(codes))
	for rows.Next() {
		var id int
		var code string
		if err := rows.Scan(&id, &code); err != nil {
			return nil, fmt.Errorf("scan UoM: %w", err)
		}
		result[code] = id
	}
	return result, rows.Err()
}

func (r *Repository) Create(ctx context.Context, uom *UnitOfMeasure) error {
	var createdAt time.Time
	err := r.db.QueryRow(ctx, `
		INSERT INTO units_of_measure (code, name, description, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`, uom.Code, uom.Name, uom.Description, uom.IsActive).Scan(&uom.ID, &createdAt)
	if err != nil {
		return err
	}
	uom.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	if r.cache != nil {
		r.cache.FlushByPrefix("uom:")
		r.cache.Delete("uoms:all")
	}
	return nil
}

func (r *Repository) Update(ctx context.Context, uom *UnitOfMeasure) error {
	_, err := r.db.Exec(ctx, `
		UPDATE units_of_measure SET code = $1, name = $2, description = $3, is_active = $4
		WHERE id = $5
	`, uom.Code, uom.Name, uom.Description, uom.IsActive, uom.ID)
	if err == nil && r.cache != nil {
		r.cache.Delete(fmt.Sprintf("uom:%d", uom.ID))
		r.cache.Delete("uoms:all")
	}
	return err
}

func (r *Repository) Delete(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, "DELETE FROM units_of_measure WHERE id = $1", id)
	if err == nil && r.cache != nil {
		r.cache.Delete(fmt.Sprintf("uom:%d", id))
		r.cache.Delete("uoms:all")
	}
	return err
}

func (r *Repository) GetAllForExport(ctx context.Context) ([]UnitOfMeasure, error) {
	rows, err := r.db.Query(ctx, "SELECT id, code, name, COALESCE(description,''), is_active, created_at FROM units_of_measure ORDER BY code")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var units []UnitOfMeasure
	for rows.Next() {
		var u UnitOfMeasure
		var createdAt time.Time
		if err := rows.Scan(&u.ID, &u.Code, &u.Name, &u.Description, &u.IsActive, &createdAt); err != nil {
			return nil, err
		}
		u.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		units = append(units, u)
	}
	return units, nil
}

func (r *Repository) BulkUpsert(ctx context.Context, records []UOMImportRow) ImportResult {
	result := ImportResult{Errors: []string{}}
	if len(records) == 0 {
		return result
	}

	valueStrings := make([]string, 0, len(records))
	valueArgs := make([]interface{}, 0, len(records)*4)
	for _, rec := range records {
		if rec.Code == "" {
			result.AddError(rec.Row, "Code is required")
			continue
		}
		if rec.Name == "" {
			result.AddError(rec.Row, "Name is required")
			continue
		}
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d)", len(valueArgs)+1, len(valueArgs)+2, len(valueArgs)+3, len(valueArgs)+4))
		valueArgs = append(valueArgs, rec.Code, rec.Name, rec.Description, rec.IsActive)
	}

	if len(valueStrings) == 0 {
		return result
	}

	query := fmt.Sprintf(`
		INSERT INTO units_of_measure (code, name, description, is_active)
		VALUES %s
		ON CONFLICT (code) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			is_active = EXCLUDED.is_active
		RETURNING (xmax = 0) AS is_insert
	`, strings.Join(valueStrings, ", "))

	rows, err := r.db.Query(ctx, query, valueArgs...)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("batch upsert failed: %v", err))
		return result
	}
	defer rows.Close()

	for rows.Next() {
		var isInsert bool
		if err := rows.Scan(&isInsert); err != nil {
			continue
		}
		if isInsert {
			result.Inserted++
		} else {
			result.Updated++
		}
	}
	if err := rows.Err(); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("rows iteration failed: %v", err))
	}

	return result
}
