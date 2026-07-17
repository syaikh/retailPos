package store

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

func (r *Repository) GetAll(ctx context.Context, limit, offset int, search string, isActive *bool) ([]Store, int, error) {
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
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM stores WHERE %s", where)
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count stores: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, name, COALESCE(address, ''), COALESCE(phone, ''), is_active, created_at
		FROM stores
		WHERE %s
		ORDER BY id ASC
		LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list stores: %w", err)
	}
	defer rows.Close()

	var stores []Store
	for rows.Next() {
		var s Store
		var createdAt time.Time
		if err := rows.Scan(&s.ID, &s.Name, &s.Address, &s.Phone, &s.IsActive, &createdAt); err != nil {
			return nil, 0, fmt.Errorf("scan store: %w", err)
		}
		s.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		stores = append(stores, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate stores: %w", err)
	}
	if stores == nil {
		stores = []Store{}
	}
	return stores, total, nil
}

func (r *Repository) GetByID(ctx context.Context, id int) (*Store, error) {
	var s Store
	var createdAt time.Time
	err := r.db.QueryRow(ctx, `
		SELECT id, name, COALESCE(address, ''), COALESCE(phone, ''), is_active, created_at
		FROM stores WHERE id = $1`, id).Scan(
		&s.ID, &s.Name, &s.Address, &s.Phone, &s.IsActive, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("get store by id: %w", err)
	}
	s.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &s, nil
}

func (r *Repository) Create(ctx context.Context, s *Store) error {
	var createdAt time.Time
	err := r.db.QueryRow(ctx, `
		INSERT INTO stores (name, address, phone, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`,
		s.Name, s.Address, s.Phone, s.IsActive,
	).Scan(&s.ID, &createdAt)
	if err != nil {
		return fmt.Errorf("create store: %w", err)
	}
	s.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return nil
}

func (r *Repository) Update(ctx context.Context, s *Store) error {
	_, err := r.db.Exec(ctx, `
		UPDATE stores
		SET name = $1, address = $2, phone = $3, is_active = $4
		WHERE id = $5`,
		s.Name, s.Address, s.Phone, s.IsActive, s.ID)
	if err != nil {
		return fmt.Errorf("update store: %w", err)
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, `DELETE FROM stores WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete store: %w", err)
	}
	return nil
}

func (r *Repository) GetAllActive(ctx context.Context) ([]Store, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, COALESCE(address, ''), COALESCE(phone, ''), is_active, created_at
		FROM stores WHERE is_active = true
		ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("list active stores: %w", err)
	}
	defer rows.Close()

	var stores []Store
	for rows.Next() {
		var s Store
		var createdAt time.Time
		if err := rows.Scan(&s.ID, &s.Name, &s.Address, &s.Phone, &s.IsActive, &createdAt); err != nil {
			return nil, fmt.Errorf("scan store: %w", err)
		}
		s.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		stores = append(stores, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stores: %w", err)
	}
	if stores == nil {
		stores = []Store{}
	}
	return stores, nil
}

func (r *Repository) GetByName(ctx context.Context, name string) (*Store, error) {
	var s Store
	var createdAt time.Time
	err := r.db.QueryRow(ctx, `
		SELECT id, name, COALESCE(address, ''), COALESCE(phone, ''), is_active, created_at
		FROM stores WHERE LOWER(name) = LOWER($1)`, name).Scan(
		&s.ID, &s.Name, &s.Address, &s.Phone, &s.IsActive, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("get store by name: %w", err)
	}
	s.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &s, nil
}
