package user

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"retail-pos-system/internal/shared"
	"retail-pos-system/pkg/cache"

	"github.com/jackc/pgx/v5"
)

type Repository struct {
	db    shared.DBPool
	cache *cache.Cache
}

func NewRepository(db shared.DBPool) *Repository {
	return &Repository{db: db}
}

// SetCache enables caching for this repository.
func (r *Repository) SetCache(c *cache.Cache) {
	r.cache = c
}

// ==================== USER ====================

func (r *Repository) GetByID(ctx context.Context, id int) (*User, error) {
	return r.getUserByID(ctx, id)
}

func (r *Repository) GetByUsername(ctx context.Context, username string) (*User, error) {
	if r.cache != nil {
		key := fmt.Sprintf("user:username:%s", username)
		if v, ok := r.cache.Get(key); ok {
			u := v.(User)
			return &u, nil
		}
	}
	var u User
	var storeID sql.NullInt64
	var createdAt, updatedAt time.Time
	var lastLogin sql.NullTime

	err := r.db.QueryRow(ctx, `
		SELECT id, username, email, password_hash, role_id, store_id, is_active, created_at, updated_at, last_login
		FROM users WHERE username = $1 AND deleted_at IS NULL
	`, username).Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.RoleID, &storeID, &u.IsActive, &createdAt, &updatedAt, &lastLogin)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}
	u.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	u.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	if lastLogin.Valid {
		u.LastLogin = lastLogin.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
	}
	if storeID.Valid {
		v := int(storeID.Int64)
		u.StoreID = &v
	}
	if u.RoleID > 0 {
		role, err := r.GetRoleByID(ctx, u.RoleID)
		if err != nil {
			slog.Warn("GetRoleByID error", "role_id", u.RoleID, "error", err)
		} else if role != nil {
			u.Role = *role
		}
	}
	if r.cache != nil {
		r.cache.Set(fmt.Sprintf("user:username:%s", username), u)
	}
	return &u, nil
}

func (r *Repository) getUserByID(ctx context.Context, id int) (*User, error) {
	var u User
	var storeID sql.NullInt64
	var createdAt, updatedAt time.Time
	var lastLogin sql.NullTime

	err := r.db.QueryRow(ctx, `
		SELECT id, username, email, password_hash, role_id, store_id, is_active, created_at, updated_at, last_login
		FROM users WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.RoleID, &storeID, &u.IsActive, &createdAt, &updatedAt, &lastLogin)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	u.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	u.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	if lastLogin.Valid {
		u.LastLogin = lastLogin.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
	}
	if storeID.Valid {
		v := int(storeID.Int64)
		u.StoreID = &v
	}
	if u.RoleID > 0 {
		role, _ := r.GetRoleByID(ctx, u.RoleID)
		if role != nil {
			u.Role = *role
		}
	}
	return &u, nil
}

func (r *Repository) GetAllUsers(ctx context.Context, limit, offset int, search string, sortBy string, sortDir string, roleID int, isActive *bool) ([]User, int, error) {
	var users []User
	var total int

	validSortColumns := map[string]bool{
		"id": true, "username": true, "email": true, "role_id": true,
		"is_active": true, "created_at": true, "last_login": true, "updated_at": true,
	}
	if !validSortColumns[sortBy] {
		sortBy = "id"
	}
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "desc"
	}

	query := `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`
	args := []interface{}{}
	argIdx := 1
	if search != "" {
		query += fmt.Sprintf(" AND (username ILIKE $%d OR email ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}
	if roleID > 0 {
		query += fmt.Sprintf(" AND role_id = $%d", argIdx)
		args = append(args, roleID)
		argIdx++
	}
	if isActive != nil {
		query += fmt.Sprintf(" AND is_active = $%d", argIdx)
		args = append(args, *isActive)
	}

	err := r.db.QueryRow(ctx, query, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	query = `SELECT u.id, u.username, u.email, u.password_hash, u.role_id, u.store_id, u.is_active,
	                 u.created_at, u.updated_at, u.last_login,
	                 COALESCE(r.id, 0), COALESCE(r.name, ''), COALESCE(r.description, ''), COALESCE(r.is_system, false), r.created_at
	          FROM users u
	          LEFT JOIN roles r ON r.id = u.role_id
	          WHERE u.deleted_at IS NULL`
	args2 := []interface{}{}
	argIdx2 := 1
	if search != "" {
		query += fmt.Sprintf(" AND (u.username ILIKE $%d OR u.email ILIKE $%d)", argIdx2, argIdx2)
		args2 = append(args2, "%"+search+"%")
		argIdx2++
	}
	if roleID > 0 {
		query += fmt.Sprintf(" AND u.role_id = $%d", argIdx2)
		args2 = append(args2, roleID)
		argIdx2++
	}
	if isActive != nil {
		query += fmt.Sprintf(" AND u.is_active = $%d", argIdx2)
		args2 = append(args2, *isActive)
		argIdx2++
	}
	allowedSortBy := map[string]string{"id": "u.id", "username": "LOWER(u.username)", "email": "LOWER(u.email)", "role_id": "r.name", "is_active": "u.is_active", "created_at": "u.created_at", "updated_at": "u.updated_at", "last_login": "u.last_login"}
	allowedSortDir := map[string]bool{"asc": true, "desc": true}
	var sortExpr string
	if col, ok := allowedSortBy[sortBy]; ok {
		sortExpr = col
	} else {
		sortExpr = "u.id"
	}
	if sortDir == "" || !allowedSortDir[sortDir] {
		sortDir = "DESC"
	}
	query += fmt.Sprintf(" ORDER BY %s %s LIMIT $%d OFFSET $%d", sortExpr, sortDir, argIdx2, argIdx2+1)
	args2 = append(args2, limit, offset)

	rows, err := r.db.Query(ctx, query, args2...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var u User
		var storeID sql.NullInt64
		var createdAt, updatedAt time.Time
		var lastLogin sql.NullTime
		var roleIDVal int
		var roleName, roleDesc string
		var roleIsSystem bool
		var roleCreatedAt sql.NullTime

		err = rows.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.RoleID, &storeID, &u.IsActive,
			&createdAt, &updatedAt, &lastLogin,
			&roleIDVal, &roleName, &roleDesc, &roleIsSystem, &roleCreatedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user row: %w", err)
		}
		u.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		u.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		if lastLogin.Valid {
			u.LastLogin = lastLogin.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
		}
		if storeID.Valid {
			v := int(storeID.Int64)
			u.StoreID = &v
		}
		if roleIDVal > 0 {
			u.Role.ID = roleIDVal
			u.Role.Name = roleName
			u.Role.Description = roleDesc
			u.Role.IsSystem = roleIsSystem
			if roleCreatedAt.Valid {
				u.Role.CreatedAt = roleCreatedAt.Time
			}
		}
		users = append(users, u)
	}
	if rows.Err() != nil {
		return nil, 0, fmt.Errorf("rows iteration error: %w", rows.Err())
	}
	return users, total, nil
}

func (r *Repository) CreateUser(ctx context.Context, user *User) error {
	var createdAt, updatedAt time.Time
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, role_id, store_id, is_active)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at, updated_at
	`, user.Username, user.Email, user.Password, user.RoleID, user.StoreID, user.IsActive).Scan(&user.ID, &createdAt, &updatedAt)
	if err != nil {
		return err
	}
	user.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	user.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return nil
}

func (r *Repository) UpdateUser(ctx context.Context, user *User) error {
	if user.Password != "" {
		_, err := r.db.Exec(ctx, `
			UPDATE users SET username = $1, email = $2, password_hash = $3, role_id = $4, store_id = $5, is_active = $6, updated_at = NOW()
			WHERE id = $7
		`, user.Username, user.Email, user.Password, user.RoleID, user.StoreID, user.IsActive, user.ID)
		return err
	}
	_, err := r.db.Exec(ctx, `
		UPDATE users SET username = $1, email = $2, role_id = $3, store_id = $4, is_active = $5, updated_at = NOW()
		WHERE id = $6
	`, user.Username, user.Email, user.RoleID, user.StoreID, user.IsActive, user.ID)
	return err
}

func (r *Repository) DeleteUser(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, "UPDATE users SET deleted_at = NOW() WHERE id = $1", id)
	return err
}

func (r *Repository) UpdateLastLogin(ctx context.Context, userID int) error {
	_, err := r.db.Exec(ctx, "UPDATE users SET last_login = NOW() WHERE id = $1", userID)
	return err
}

func (r *Repository) UpdatePassword(ctx context.Context, userID int, hashedPassword string) error {
	_, err := r.db.Exec(ctx, "UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2", hashedPassword, userID)
	return err
}

func (r *Repository) DeleteUserRefreshTokens(ctx context.Context, userID int) error {
	_, err := r.db.Exec(ctx, "DELETE FROM refresh_tokens WHERE user_id = $1", userID)
	return err
}

// ==================== ROLE ====================

func (r *Repository) GetRoleByID(ctx context.Context, id int) (*Role, error) {
	if r.cache != nil {
		key := fmt.Sprintf("role:%d", id)
		if v, ok := r.cache.Get(key); ok {
			role := v.(Role)
			return &role, nil
		}
	}
	var role Role
	var isSystem bool
	var createdAt time.Time
	err := r.db.QueryRow(ctx, "SELECT id, name, description, is_system, created_at FROM roles WHERE id = $1", id).Scan(
		&role.ID, &role.Name, &role.Description, &isSystem, &createdAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("role not found")
		}
		return nil, err
	}
	role.IsSystem = isSystem
	role.CreatedAt = createdAt
	if r.cache != nil {
		r.cache.Set(fmt.Sprintf("role:%d", id), role)
	}
	return &role, nil
}

func (r *Repository) GetAllRoles(ctx context.Context) ([]Role, error) {
	rows, err := r.db.Query(ctx, `
		SELECT r.id, r.name, r.description, r.is_system, r.created_at,
		       COALESCE(ARRAY_AGG(p.code) FILTER (WHERE p.code IS NOT NULL), '{}') AS permissions
		FROM roles r
		LEFT JOIN role_permissions rp ON rp.role_id = r.id
		LEFT JOIN permissions p ON p.id = rp.permission_id
		GROUP BY r.id, r.name, r.description, r.is_system, r.created_at
		ORDER BY r.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var rl Role
		var isSystem bool
		var createdAt time.Time
		if err := rows.Scan(&rl.ID, &rl.Name, &rl.Description, &isSystem, &createdAt, &rl.Permissions); err != nil {
			return nil, err
		}
		rl.IsSystem = isSystem
		rl.CreatedAt = createdAt
		roles = append(roles, rl)
	}
	return roles, nil
}

func (r *Repository) CreateRole(ctx context.Context, role *Role) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO roles (name, description, is_system) VALUES ($1, $2, $3)
		RETURNING id, created_at
	`, role.Name, role.Description, role.IsSystem).Scan(&role.ID, &role.CreatedAt)
}

func (r *Repository) UpdateRole(ctx context.Context, role *Role) error {
	_, err := r.db.Exec(ctx, "UPDATE roles SET name = $1, description = $2 WHERE id = $3", role.Name, role.Description, role.ID)
	if err == nil && r.cache != nil {
		r.cache.Delete(fmt.Sprintf("role:%d", role.ID))
	}
	return err
}

func (r *Repository) DeleteRole(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, "DELETE FROM roles WHERE id = $1", id)
	if err == nil && r.cache != nil {
		r.cache.Delete(fmt.Sprintf("role:%d", id))
	}
	return err
}

func (r *Repository) CountUsersByRole(ctx context.Context, roleID int) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE role_id = $1 AND deleted_at IS NULL", roleID).Scan(&count)
	return count, err
}

func (r *Repository) GetRolePermissions(ctx context.Context, roleID int) ([]Permission, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.id, p.code, p.name, p.created_at
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = $1
	`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []Permission
	for rows.Next() {
		var p Permission
		var createdAt time.Time
		if err := rows.Scan(&p.ID, &p.Code, &p.Name, &createdAt); err != nil {
			return nil, err
		}
		p.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		perms = append(perms, p)
	}
	return perms, nil
}

func (r *Repository) UpdateRolePermissions(ctx context.Context, roleID int, permissionIDs []int) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, "DELETE FROM role_permissions WHERE role_id = $1", roleID)
	if err != nil {
		return err
	}
	for _, pid := range permissionIDs {
		_, err = tx.Exec(ctx, "INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", roleID, pid)
		if err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	if r.cache != nil {
		r.cache.Delete(fmt.Sprintf("role:%d", roleID))
	}
	return nil
}

func (r *Repository) GetAllPermissions(ctx context.Context) ([]Permission, error) {
	rows, err := r.db.Query(ctx, "SELECT id, code, name, created_at FROM permissions ORDER BY code")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []Permission
	for rows.Next() {
		var p Permission
		var createdAt time.Time
		if err := rows.Scan(&p.ID, &p.Code, &p.Name, &createdAt); err != nil {
			return nil, err
		}
		p.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		perms = append(perms, p)
	}
	return perms, nil
}

func (r *Repository) CountRecentLoginFailures(ctx context.Context, ip string, cutoff time.Time) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit_logs
		WHERE action = 'login_failed'
		AND ip_address = $1::inet
		AND created_at >= $2
	`, ip, cutoff).Scan(&count)
	return count, err
}
