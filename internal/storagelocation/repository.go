package storagelocation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"retail-pos-system/internal/shared"

	"github.com/jackc/pgx/v5"
)

const selectColumns = `sl.id, sl.code, sl.name, sl.warehouse_id, sl.store_id,
	COALESCE(sl.notes, ''), sl.is_active, sl.created_at, sl.updated_at`

const baseFrom = `FROM storage_locations sl`

type Repository struct {
	db                     shared.DBPool
	storeExistenceProvider ExistenceProvider
}

func NewRepository(db shared.DBPool) *Repository {
	return &Repository{db: db}
}

// SetStoreExistenceProvider wires the store-owned implementation of the
// ExistenceProvider port (ADR §2.4). It MUST be called before any
// create/update path validates a store/warehouse reference — an unwired
// repository fails fast at runtime.
func (r *Repository) SetStoreExistenceProvider(p ExistenceProvider) {
	r.storeExistenceProvider = p
}

func (r *Repository) scanLocation(scanner interface{ Scan(...interface{}) error }) (*StorageLocation, error) {
	var sl StorageLocation
	var createdAt, updatedAt time.Time
	if err := scanner.Scan(&sl.ID, &sl.Code, &sl.Name, &sl.WarehouseID, &sl.StoreID, &sl.Notes, &sl.IsActive, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	sl.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	sl.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &sl, nil
}

func (r *Repository) GetAll(ctx context.Context, limit, offset int, search string, isActive *bool) ([]StorageLocation, int, error) {
	where := "1=1"
	args := []interface{}{}
	argIdx := 1

	if search != "" {
		where += fmt.Sprintf(" AND (LOWER(sl.name) LIKE LOWER($%d) OR LOWER(sl.code) LIKE LOWER($%d))", argIdx, argIdx)
		args = append(args, "%"+strings.ToLower(search)+"%")
		argIdx++
	}
	if isActive != nil {
		where += fmt.Sprintf(" AND sl.is_active = $%d", argIdx)
		args = append(args, *isActive)
		argIdx++
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) %s WHERE %s", baseFrom, where)
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count storage locations: %w", err)
	}

	query := fmt.Sprintf(`SELECT %s %s WHERE %s ORDER BY sl.code ASC LIMIT $%d OFFSET $%d`,
		selectColumns, baseFrom, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list storage locations: %w", err)
	}
	defer rows.Close()

	var locations []StorageLocation
	for rows.Next() {
		sl, err := r.scanLocation(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan storage location: %w", err)
		}
		locations = append(locations, *sl)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate storage locations: %w", err)
	}
	if locations == nil {
		locations = []StorageLocation{}
	}
	return locations, total, nil
}

func (r *Repository) GetByID(ctx context.Context, id int) (*StorageLocation, error) {
	query := fmt.Sprintf(`SELECT %s %s WHERE sl.id = $1`, selectColumns, baseFrom)
	sl, err := r.scanLocation(r.db.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("storage location not found")
		}
		return nil, err
	}
	return sl, nil
}

func (r *Repository) GetByCode(ctx context.Context, code string) (*StorageLocation, error) {
	query := fmt.Sprintf(`SELECT %s %s WHERE LOWER(sl.code) = LOWER($1)`, selectColumns, baseFrom)
	sl, err := r.scanLocation(r.db.QueryRow(ctx, query, code))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("storage location not found")
		}
		return nil, err
	}
	return sl, nil
}

func (r *Repository) Create(ctx context.Context, sl *StorageLocation) error {
	var createdAt, updatedAt time.Time
	err := r.db.QueryRow(ctx, `
		INSERT INTO storage_locations (code, name, warehouse_id, store_id, notes, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`,
		sl.Code, sl.Name, sl.WarehouseID, sl.StoreID, sl.Notes, sl.IsActive,
	).Scan(&sl.ID, &createdAt, &updatedAt)
	if err != nil {
		return fmt.Errorf("create storage location: %w", err)
	}
	sl.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	sl.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return nil
}

func (r *Repository) Update(ctx context.Context, sl *StorageLocation) error {
	_, err := r.db.Exec(ctx, `
		UPDATE storage_locations
		SET code = $1, name = $2, warehouse_id = $3, store_id = $4, notes = $5, is_active = $6, updated_at = NOW()
		WHERE id = $7`,
		sl.Code, sl.Name, sl.WarehouseID, sl.StoreID, sl.Notes, sl.IsActive, sl.ID)
	if err != nil {
		return fmt.Errorf("update storage location: %w", err)
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, `DELETE FROM storage_locations WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete storage location: %w", err)
	}
	return nil
}

func (r *Repository) GetAllActive(ctx context.Context) ([]StorageLocation, error) {
	query := fmt.Sprintf(`SELECT %s %s WHERE sl.is_active = true ORDER BY sl.code ASC`, selectColumns, baseFrom)
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list active storage locations: %w", err)
	}
	defer rows.Close()

	var locations []StorageLocation
	for rows.Next() {
		sl, err := r.scanLocation(rows)
		if err != nil {
			return nil, fmt.Errorf("scan storage location: %w", err)
		}
		locations = append(locations, *sl)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate storage locations: %w", err)
	}
	if locations == nil {
		locations = []StorageLocation{}
	}
	return locations, nil
}

func (r *Repository) CodeExists(ctx context.Context, code string, excludeID int) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM storage_locations WHERE LOWER(code) = LOWER($1) AND id != $2`,
		code, excludeID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check code exists: %w", err)
	}
	return count > 0, nil
}

func (r *Repository) WarehouseExists(ctx context.Context, id int) (bool, error) {
	if r.storeExistenceProvider == nil {
		return false, errors.New("storagelocation repository: store existence provider not wired; call SetStoreExistenceProvider")
	}
	return r.storeExistenceProvider.WarehouseExists(ctx, r.db, id)
}

func (r *Repository) StoreExists(ctx context.Context, id int) (bool, error) {
	if r.storeExistenceProvider == nil {
		return false, errors.New("storagelocation repository: store existence provider not wired; call SetStoreExistenceProvider")
	}
	return r.storeExistenceProvider.StoreExists(ctx, r.db, id)
}

func (r *Repository) BulkUpdate(ctx context.Context, ids []int, isActive bool) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	query := `UPDATE storage_locations SET is_active = $1, updated_at = NOW() WHERE id = ANY($2)`
	result, err := r.db.Exec(ctx, query, isActive, ids)
	if err != nil {
		return 0, fmt.Errorf("bulk update storage locations: %w", err)
	}
	return int(result.RowsAffected()), nil
}

func (r *Repository) BulkDelete(ctx context.Context, ids []int) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result, err := r.db.Exec(ctx, `DELETE FROM storage_locations WHERE id = ANY($1)`, ids)
	if err != nil {
		return 0, fmt.Errorf("bulk delete storage locations: %w", err)
	}
	return int(result.RowsAffected()), nil
}
