package brand

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByID(ctx context.Context, id int) (*Brand, error) {
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
	b.CreatedAt = createdAt.Format(time.RFC3339)
	return &b, nil
}

func (r *Repository) GetAll(ctx context.Context) ([]Brand, error) {
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
		b.CreatedAt = createdAt.Format(time.RFC3339)
		brands = append(brands, b)
	}
	return brands, nil
}

func (r *Repository) GetIDByName(ctx context.Context, name string) (int, error) {
	var id int
	err := r.db.QueryRow(ctx, "SELECT id FROM brands WHERE name = $1 AND is_active = true", name).Scan(&id)
	return id, err
}

func (r *Repository) Create(ctx context.Context, brand *Brand) error {
	var createdAt, updatedAt time.Time
	return r.db.QueryRow(ctx, `
		INSERT INTO brands (name, description, is_active)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`, brand.Name, brand.Description, brand.IsActive).Scan(&brand.ID, &createdAt, &updatedAt)
}

func (r *Repository) Update(ctx context.Context, brand *Brand) error {
	_, err := r.db.Exec(ctx, `
		UPDATE brands SET name = $1, description = $2, is_active = $3, updated_at = NOW()
		WHERE id = $4
	`, brand.Name, brand.Description, brand.IsActive, brand.ID)
	return err
}

func (r *Repository) Delete(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, "DELETE FROM brands WHERE id = $1", id)
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
		b.CreatedAt = createdAt.Format(time.RFC3339)
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

	return result
}
