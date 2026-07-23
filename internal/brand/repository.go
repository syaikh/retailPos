package brand

import (
	"context"
	"fmt"
	"strings"
	"time"

	"retail-pos-system/internal/shared"
	"retail-pos-system/pkg/cache"

	"github.com/jackc/pgx/v5"
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
	db    shared.DBPool
	cache *cache.Cache
}

func NewRepository(db shared.DBPool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) SetCache(c *cache.Cache) {
	r.cache = c
}

func (r *Repository) GetByID(ctx context.Context, id int) (*Brand, error) {
	if r.cache != nil {
		key := fmt.Sprintf("brand:%d", id)
		if v, ok := r.cache.Get(key); ok {
			b := v.(Brand)
			return &b, nil
		}
	}
	var b Brand
	var createdAt, updatedAt time.Time
	err := r.db.QueryRow(ctx, "SELECT id, name, description, is_active, created_at, updated_at FROM brands WHERE id = $1", id).Scan(
		&b.ID, &b.Name, &b.Description, &b.IsActive, &createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("brand not found")
		}
		return nil, err
	}
	b.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	if r.cache != nil {
		r.cache.Set(fmt.Sprintf("brand:%d", id), b)
	}
	return &b, nil
}

func (r *Repository) GetAll(ctx context.Context) ([]Brand, error) {
	if r.cache != nil {
		if v, ok := r.cache.Get("brands:all"); ok {
			return v.([]Brand), nil
		}
	}
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
		b.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		brands = append(brands, b)
	}
	if r.cache != nil && brands != nil {
		r.cache.Set("brands:all", brands)
	}
	return brands, nil
}

func (r *Repository) GetAllPaginated(ctx context.Context, limit, offset int, search string) ([]Brand, int, error) {
	args := []interface{}{}
	where := "WHERE is_active = true"
	argIdx := 1

	if search != "" {
		where += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d)", argIdx, argIdx+1)
		args = append(args, "%"+search+"%", "%"+search+"%")
		argIdx += 2
	}

	var total int
	err := r.db.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM brands %s", where), args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, fmt.Sprintf(
		"SELECT id, name, description, is_active, created_at, updated_at FROM brands %s ORDER BY name LIMIT $%d OFFSET $%d",
		where, argIdx, argIdx+1,
	), append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var brands []Brand
	for rows.Next() {
		var b Brand
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&b.ID, &b.Name, &b.Description, &b.IsActive, &createdAt, &updatedAt); err != nil {
			return nil, 0, err
		}
		b.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		brands = append(brands, b)
	}
	return brands, total, nil
}

func (r *Repository) GetIDByName(ctx context.Context, name string) (int, error) {
	var id int
	err := r.db.QueryRow(ctx, "SELECT id FROM brands WHERE name = $1 AND is_active = true", name).Scan(&id)
	return id, err
}

func (r *Repository) GetIDsByNames(ctx context.Context, names []string) (map[string]int, error) {
	if len(names) == 0 {
		return nil, nil
	}
	rows, err := r.db.Query(ctx, "SELECT id, name FROM brands WHERE name = ANY($1) AND is_active = true", names)
	if err != nil {
		return nil, fmt.Errorf("batch get brand IDs: %w", err)
	}
	defer rows.Close()
	result := make(map[string]int, len(names))
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("scan brand: %w", err)
		}
		result[name] = id
	}
	return result, rows.Err()
}

func (r *Repository) Create(ctx context.Context, brand *Brand) error {
	var createdAt, updatedAt time.Time
	err := r.db.QueryRow(ctx, `
		INSERT INTO brands (name, description, is_active)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`, brand.Name, brand.Description, brand.IsActive).Scan(&brand.ID, &createdAt, &updatedAt)
	if err == nil && r.cache != nil {
		r.cache.FlushByPrefix("brand:")
		r.cache.Delete("brands:all")
	}
	return err
}

func (r *Repository) Update(ctx context.Context, brand *Brand) error {
	_, err := r.db.Exec(ctx, `
		UPDATE brands SET name = $1, description = $2, is_active = $3, updated_at = NOW()
		WHERE id = $4
	`, brand.Name, brand.Description, brand.IsActive, brand.ID)
	if err == nil && r.cache != nil {
		r.cache.Delete(fmt.Sprintf("brand:%d", brand.ID))
		r.cache.Delete("brands:all")
	}
	return err
}

func (r *Repository) Delete(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, "DELETE FROM brands WHERE id = $1", id)
	if err == nil && r.cache != nil {
		r.cache.Delete(fmt.Sprintf("brand:%d", id))
		r.cache.Delete("brands:all")
	}
	return err
}

func (r *Repository) GetAllForExport(ctx context.Context) ([]Brand, error) {
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
		b.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		brands = append(brands, b)
	}
	return brands, nil
}

func (r *Repository) BulkUpsert(ctx context.Context, records []BrandImportRow) ImportResult {
	result := ImportResult{Errors: []string{}}
	if len(records) == 0 {
		return result
	}

	valueStrings := make([]string, 0, len(records))
	valueArgs := make([]interface{}, 0, len(records)*3)
	for _, rec := range records {
		if rec.Name == "" {
			result.AddError(rec.Row, "Name is required")
			continue
		}
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d)", len(valueArgs)+1, len(valueArgs)+2, len(valueArgs)+3))
		valueArgs = append(valueArgs, rec.Name, rec.Description, rec.IsActive)
	}

	if len(valueStrings) == 0 {
		return result
	}

	query := fmt.Sprintf(`
		INSERT INTO brands (name, description, is_active)
		VALUES %s
		ON CONFLICT (name) DO UPDATE SET
			description = EXCLUDED.description,
			is_active = EXCLUDED.is_active,
			updated_at = NOW()
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
		result.Errors = append(result.Errors, fmt.Sprintf("rows iteration: %v", err))
	}

	if r.cache != nil {
		r.cache.FlushByPrefix("brand:")
		r.cache.Delete("brands:all")
	}

	return result
}
