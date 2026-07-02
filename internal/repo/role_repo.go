package repo

import (
	"database/sql"
	model "retailPos/internal/model"
)

type RoleRepo struct {
	db *sql.DB
}

func NewRoleRepo(db *sql.DB) *RoleRepo {
	return &RoleRepo{db: db}
}

// GetRoleByID fetches a role by ID
func (r *RoleRepo) GetRoleByID(id int) (*model.Role, error) {
	query := `SELECT id, name, description, is_system, created_at, updated_at FROM roles WHERE id = $1`
	var role model.Role
	err := r.db.QueryRow(query, id).Scan(
		&role.ID, &role.Name, &role.Description,
		&role.IsSystem, &role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

// GetRoleByName fetches a role by name
func (r *RoleRepo) GetRoleByName(name string) (*model.Role, error) {
	query := `SELECT id, name, description, is_system, created_at, updated_at FROM roles WHERE name = $1`
	var role model.Role
	err := r.db.QueryRow(query, name).Scan(
		&role.ID, &role.Name, &role.Description,
		&role.IsSystem, &role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

// GetUserRole fetches the complete role for a user via JOIN
func (r *RoleRepo) GetUserRole(userID int) (*model.Role, error) {
	query := `
		SELECT ro.id, ro.name, ro.description, ro.is_system, ro.created_at, ro.updated_at
		FROM users u
		JOIN roles ro ON u.role_id = ro.id
		WHERE u.id = $1
	`
	var role model.Role
	err := r.db.QueryRow(query, userID).Scan(
		&role.ID, &role.Name, &role.Description,
		&role.IsSystem, &role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

// ListPermissions returns all Permission structs for a given roleID
func (r *RoleRepo) ListPermissions(roleID int) ([]model.Permission, error) {
	query := `
		SELECT p.id, p.code, p.description, p.category, p.created_at
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = $1
		ORDER BY p.category, p.code
	`
	rows, err := r.db.Query(query, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []model.Permission
	for rows.Next() {
		var p model.Permission
		if err := rows.Scan(
			&p.ID, &p.Code, &p.Description, &p.Category, &p.CreatedAt,
		); err != nil {
			return nil, err
		}
		permissions = append(permissions, p)
	}
	return permissions, nil
}

// ListUserPermissions returns all permission codes (strings) for a userID
func (r *RoleRepo) ListUserPermissions(userID int) ([]string, error) {
	query := `
		SELECT DISTINCT p.code
		FROM users u
		JOIN roles r ON u.role_id = r.id
		JOIN role_permissions rp ON r.id = rp.role_id
		JOIN permissions p ON rp.permission_id = p.id
		WHERE u.id = $1
		ORDER BY p.code
	`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		permissions = append(permissions, code)
	}
	return permissions, nil
}

// HasPermission checks if a user has a specific permission code
func (r *RoleRepo) HasPermission(userID int, permissionCode string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM users u
			JOIN roles r ON u.role_id = r.id
			JOIN role_permissions rp ON r.id = rp.role_id
			JOIN permissions p ON rp.permission_id = p.id
			WHERE u.id = $1 AND p.code = $2
		)
	`
	var exists bool
	err := r.db.QueryRow(query, userID, permissionCode).Scan(&exists)
	return exists, err
}

// CreateRole inserts a new role
func (r *RoleRepo) CreateRole(name, description string, isSystem bool) (*model.Role, error) {
	query := `INSERT INTO roles (name, description, is_system) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`
	var role model.Role
	err := r.db.QueryRow(query, name, description, isSystem).Scan(&role.ID, &role.CreatedAt, &role.UpdatedAt)
	if err != nil {
		return nil, err
	}
	role.Name = name
	role.Description = description
	role.IsSystem = isSystem
	return &role, nil
}

// AssignPermissionToRole adds a permission to a role (idempotent)
func (r *RoleRepo) AssignPermissionToRole(roleID, permissionID int) error {
	query := `INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	_, err := r.db.Exec(query, roleID, permissionID)
	return err
}

// RevokePermissionFromRole removes a permission from a role
func (r *RoleRepo) RevokePermissionFromRole(roleID, permissionID int) error {
	query := `DELETE FROM role_permissions WHERE role_id = $1 AND permission_id = $2`
	_, err := r.db.Exec(query, roleID, permissionID)
	return err
}

// GetAllPermissions returns all permissions ordered by category, code
func (r *RoleRepo) GetAllPermissions() ([]model.Permission, error) {
	query := `SELECT id, code, description, category, created_at FROM permissions ORDER BY category, code`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []model.Permission
	for rows.Next() {
		var p model.Permission
		if err := rows.Scan(&p.ID, &p.Code, &p.Description, &p.Category, &p.CreatedAt); err != nil {
			return nil, err
		}
		permissions = append(permissions, p)
	}
	return permissions, nil
}

// GetAllRoles returns all roles (basic info)
func (r *RoleRepo) GetAllRoles() ([]model.Role, error) {
	query := `SELECT id, name, description, is_system, created_at, updated_at FROM roles ORDER BY id`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []model.Role
	for rows.Next() {
		var role model.Role
		if err := rows.Scan(
			&role.ID, &role.Name, &role.Description,
			&role.IsSystem, &role.CreatedAt, &role.UpdatedAt,
		); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, nil
}

// GetRolePermissions returns permission codes for a role
func (r *RoleRepo) GetRolePermissions(roleID int) ([]string, error) {
	query := `
		SELECT p.code
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = $1
		ORDER BY p.code
	`
	rows, err := r.db.Query(query, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var codes []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	return codes, nil
}

// UpdateUserRole updates a user's role_id
func (r *RoleRepo) UpdateUserRole(userID, roleID int) error {
	query := `UPDATE users SET role_id = $1 WHERE id = $2`
	_, err := r.db.Exec(query, roleID, userID)
	return err
}

// DeleteRole deletes a role if it's not a system role and has no assigned users
func (r *RoleRepo) DeleteRole(id int) error {
	var count int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM users WHERE role_id = $1", id).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return sql.ErrConnDone
	}
	_, err := r.db.Exec("DELETE FROM roles WHERE id = $1", id)
	return err
}

// ClearRolePermissions removes all permissions from a role
func (r *RoleRepo) ClearRolePermissions(roleID int) error {
	_, err := r.db.Exec("DELETE FROM role_permissions WHERE role_id = $1", roleID)
	return err
}
