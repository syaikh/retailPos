package appsettings

import (
	"context"
	"fmt"
	"strings"

	"retail-pos-system/internal/shared"
)

// Repository provides read/write access to the app_settings key-value table.
type Repository struct {
	db shared.DBPool
}

// NewRepository returns a new Repository.
func NewRepository(db shared.DBPool) *Repository {
	return &Repository{db: db}
}

// GetAll returns every setting as a map keyed by setting name.
func (r *Repository) GetAll(ctx context.Context) (map[string]string, error) {
	rows, err := r.db.Query(ctx, "SELECT key, value FROM app_settings")
	if err != nil {
		return nil, fmt.Errorf("query app_settings: %w", err)
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, fmt.Errorf("scan app_settings row: %w", err)
		}
		result[k] = v
	}
	return result, rows.Err()
}

// GetMultiple returns only the requested keys (unknown keys are silently omitted).
func (r *Repository) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	if len(keys) == 0 {
		return make(map[string]string), nil
	}
	placeholders := make([]string, len(keys))
	args := make([]interface{}, len(keys))
	for i, k := range keys {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = k
	}
	query := fmt.Sprintf("SELECT key, value FROM app_settings WHERE key IN (%s)", strings.Join(placeholders, ", "))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query app_settings: %w", err)
	}
	defer rows.Close()

	result := make(map[string]string, len(keys))
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, fmt.Errorf("scan app_settings row: %w", err)
		}
		result[k] = v
	}
	return result, rows.Err()
}

// UpsertMultiple inserts or updates all provided settings in a single transaction.
func (r *Repository) UpsertMultiple(ctx context.Context, settings map[string]string) error {
	if len(settings) == 0 {
		return nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for k, v := range settings {
		_, err := tx.Exec(ctx,
			`INSERT INTO app_settings (key, value, updated_at)
			 VALUES ($1, $2, now())
			 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`,
			k, v,
		)
		if err != nil {
			return fmt.Errorf("upsert %q: %w", k, err)
		}
	}

	return tx.Commit(ctx)
}
