package category

import (
	"context"
	"fmt"
	"log"
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

var jakartaLoc *time.Location

func init() {
	var err error
	jakartaLoc, err = time.LoadLocation("Asia/Jakarta")
	if err != nil {
		log.Printf("Warning: failed to load Asia/Jakarta timezone: %v. Falling back to UTC.", err)
		jakartaLoc = time.UTC
	}
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ==================== CATEGORY ====================

func (r *Repository) ListCategories(ctx context.Context) ([]Category, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, COALESCE(slug,''), COALESCE(description,''), is_active, created_at
		FROM categories
		WHERE is_active = true
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var c Category
		var createdAt time.Time
		err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &c.IsActive, &createdAt)
		if err != nil {
			return nil, err
		}
		c.CreatedAt = createdAt.In(jakartaLoc).Format(time.RFC3339)
		c.UpdatedAt = ""
		categories = append(categories, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *Repository) GetCategoryIDByName(ctx context.Context, name string) (int, error) {
	var id int
	query := "SELECT id FROM categories WHERE name = $1 AND is_active = true"
	err := r.db.QueryRow(ctx, query, name).Scan(&id)
	return id, err
}

// GetAllCategories returns paginated categories with product count (for management page)
func (r *Repository) GetAllCategories(ctx context.Context, limit, offset int, search string) ([]Category, int, error) {
	// COUNT query
	countQuery := `SELECT COUNT(*) FROM categories WHERE 1=1`
	args := []interface{}{}
	argIdx := 1
	if search != "" {
		countQuery += fmt.Sprintf(" AND (name ILIKE $%d OR slug ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+search+"%")
	}
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count categories: %w", err)
	}

	// DATA query — LEFT JOIN + GROUP BY (optimized)
	query := `SELECT c.id, c.name, COALESCE(c.slug, ''), COALESCE(c.description, ''), c.is_active,
			  COUNT(p.id) AS product_count,
			  c.created_at, COALESCE(c.updated_at, c.created_at)
			  FROM categories c
			  LEFT JOIN products p ON p.category_id = c.id AND p.deleted_at IS NULL
			  WHERE 1=1`
	args2 := []interface{}{}
	argIdx2 := 1
	if search != "" {
		query += fmt.Sprintf(" AND (c.name ILIKE $%d OR c.slug ILIKE $%d)", argIdx2, argIdx2)
		args2 = append(args2, "%"+search+"%")
		argIdx2++
	}
	query += " GROUP BY c.id"
	query += fmt.Sprintf(" ORDER BY c.name ASC LIMIT $%d OFFSET $%d", argIdx2, argIdx2+1)
	args2 = append(args2, limit, offset)

	rows, err := r.db.Query(ctx, query, args2...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query categories: %w", err)
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var c Category
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &c.IsActive, &c.ProductCount, &createdAt, &updatedAt); err != nil {
			continue
		}
		c.CreatedAt = createdAt.In(jakartaLoc).Format(time.RFC3339)
		c.UpdatedAt = updatedAt.In(jakartaLoc).Format(time.RFC3339)
		categories = append(categories, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return categories, total, nil
}

// GetCategoryByID returns a category by ID
func (r *Repository) GetCategoryByID(ctx context.Context, id int) (*Category, error) {
	var c Category
	var createdAt, updatedAt time.Time
	err := r.db.QueryRow(ctx, `
		SELECT id, name, COALESCE(slug, ''), COALESCE(description, ''), is_active,
		       created_at, COALESCE(updated_at, created_at)
		FROM categories WHERE id = $1
	`, id).Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &c.IsActive, &createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("category not found")
		}
		return nil, err
	}
	c.CreatedAt = createdAt.In(jakartaLoc).Format(time.RFC3339)
	c.UpdatedAt = updatedAt.In(jakartaLoc).Format(time.RFC3339)
	return &c, nil
}

// SlugExists checks if a slug exists, optionally excluding an ID for updates
func (r *Repository) SlugExists(ctx context.Context, slug string, excludeID int) (bool, error) {
	var exists bool
	query := `SELECT EXISTS (SELECT 1 FROM categories WHERE slug = $1 AND id != $2)`
	err := r.db.QueryRow(ctx, query, slug, excludeID).Scan(&exists)
	return exists, err
}

// CreateCategory creates a category with auto-generated slug
func (r *Repository) CreateCategory(ctx context.Context, category *Category) error {
	if category.Slug == "" {
		category.Slug = generateSlug(category.Name)
	}

	// Resolve slug collision: append -2, -3, etc.
	baseSlug := category.Slug
	suffix := 1
	for {
		exists, err := r.SlugExists(ctx, category.Slug, 0)
		if err != nil {
			return fmt.Errorf("failed to check slug uniqueness: %w", err)
		}
		if !exists {
			break
		}
		suffix++
		category.Slug = fmt.Sprintf("%s-%d", baseSlug, suffix)
		if len(category.Slug) > 120 {
			truncLen := 120 - len(fmt.Sprintf("-%d", suffix))
			if truncLen > 0 && len(baseSlug) >= truncLen {
				category.Slug = fmt.Sprintf("%s-%d", baseSlug[:truncLen], suffix)
			} else {
				break
			}
		}
	}

	var createdAt, updatedAt time.Time
	err := r.db.QueryRow(ctx, `
		INSERT INTO categories (name, slug, description, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`, category.Name, category.Slug, category.Description, category.IsActive).Scan(&category.ID, &createdAt, &updatedAt)
	if err != nil {
		return err
	}
	category.CreatedAt = createdAt.In(jakartaLoc).Format(time.RFC3339)
	category.UpdatedAt = updatedAt.In(jakartaLoc).Format(time.RFC3339)
	return nil
}

// UpdateCategory updates a category
func (r *Repository) UpdateCategory(ctx context.Context, category *Category) error {
	// Regenerate slug if name changed
	newSlug := generateSlug(category.Name)
	if newSlug != category.Slug {
		// Check for collision (excluding current ID)
		exists, err := r.SlugExists(ctx, newSlug, category.ID)
		if err != nil {
			return fmt.Errorf("failed to check slug uniqueness: %w", err)
		}
		if exists {
			suffix := 2
			for {
				candidate := fmt.Sprintf("%s-%d", newSlug, suffix)
				ex, err := r.SlugExists(ctx, candidate, category.ID)
				if err != nil {
					return err
				}
				if !ex {
					newSlug = candidate
					break
				}
				suffix++
			}
		}
		category.Slug = newSlug
	}

	_, err := r.db.Exec(ctx, `
		UPDATE categories SET name = $1, slug = $2, description = $3, is_active = $4, updated_at = NOW()
		WHERE id = $5
	`, category.Name, category.Slug, category.Description, category.IsActive, category.ID)
	return err
}

// DeleteCategory deletes a category (FK RESTRICT handles race condition)
func (r *Repository) DeleteCategory(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, "DELETE FROM categories WHERE id = $1", id)
	return err
}

// HasActiveProducts checks if category has products using EXISTS (early exit)
func (r *Repository) HasActiveProducts(ctx context.Context, categoryID int) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM products
			WHERE category_id = $1 AND deleted_at IS NULL
			LIMIT 1
		)
	`, categoryID).Scan(&exists)
	return exists, err
}

// GetAllCategoriesForExport returns all categories without pagination
func (r *Repository) GetAllCategoriesForExport(ctx context.Context) ([]Category, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, COALESCE(slug,''), COALESCE(description,''), is_active, created_at
		FROM categories
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var c Category
		var createdAt time.Time
		err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &c.IsActive, &createdAt)
		if err != nil {
			return nil, err
		}
		c.CreatedAt = createdAt.In(jakartaLoc).Format(time.RFC3339)
		categories = append(categories, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return categories, nil
}

// BulkUpsertCategories inserts or updates categories in batch
func (r *Repository) BulkUpsertCategories(ctx context.Context, records []CategoryImportRow) ImportResult {
	result := ImportResult{Errors: []string{}}
	if len(records) == 0 {
		return result
	}

	valueStrings := make([]string, 0, len(records))
	valueArgs := make([]interface{}, 0, len(records)*4)
	for _, rec := range records {
		if rec.Name == "" {
			result.AddError(rec.Row, "Name is required")
			continue
		}
		slug := rec.Slug
		if slug == "" {
			slug = generateSlug(rec.Name)
		}
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d)", len(valueArgs)+1, len(valueArgs)+2, len(valueArgs)+3, len(valueArgs)+4))
		valueArgs = append(valueArgs, rec.Name, slug, rec.Description, rec.IsActive)
	}

	if len(valueStrings) == 0 {
		return result
	}

	query := fmt.Sprintf(`
		INSERT INTO categories (name, slug, description, is_active)
		VALUES %s
		ON CONFLICT (name) DO UPDATE SET
			slug = EXCLUDED.slug,
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
		result.Errors = append(result.Errors, fmt.Sprintf("rows iteration failed: %v", err))
	}

	return result
}

// generateSlug creates a URL-friendly slug from a name
func generateSlug(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	replacements := []struct{ from, to string }{
		{" ", "-"}, {"'", ""}, {`"`, ""}, {"&", "and"}, {"/", "-"},
		{"+", "plus"}, {"=", "equals"}, {"?", ""}, {"!", ""}, {"@", "at"},
		{"#", "number"}, {"%", "percent"}, {"(", ""}, {")", ""},
	}
	for _, r := range replacements {
		slug = strings.ReplaceAll(slug, r.from, r.to)
	}
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")
	if len(slug) > 120 {
		slug = slug[:120]
	}
	return slug
}
