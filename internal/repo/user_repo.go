package repo

import (
	"database/sql"
	model "retailPos/internal/model"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(u *model.User) error {
	query := `INSERT INTO users (username, password_hash, role) VALUES ($1, $2, $3) RETURNING id, created_at`
	return r.db.QueryRow(query, u.Username, u.PasswordHash, u.Role).Scan(&u.ID, &u.CreatedAt)
}

func (r *UserRepo) GetByUsername(username string) (*model.User, error) {
	query := `SELECT id, username, password_hash, role_id, role, created_at FROM users WHERE username = $1`
	var u model.User
	if err := r.db.QueryRow(query, username).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.RoleID, &u.Role, &u.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// GetByID selects user by ID including role_id and legacy role string
func (r *UserRepo) GetByID(id int) (*model.User, error) {
	query := `SELECT id, username, password_hash, role_id, role, created_at FROM users WHERE id = $1`
	var u model.User
	if err := r.db.QueryRow(query, id).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.RoleID, &u.Role, &u.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// GetUserRole returns the user's role via JOIN on role_id
func (r *UserRepo) GetUserRole(userID int) (*model.Role, error) {
	query := `
		SELECT r.id, r.name, r.description, r.is_system, r.created_at, r.updated_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
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

// ListUserPermissions returns all permission codes for a user (via their role)
func (r *UserRepo) ListUserPermissions(userID int) ([]string, error) {
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

// GetAll returns all users
func (r *UserRepo) GetAll() ([]model.User, error) {
	query := `SELECT id, username, password_hash, role_id, role, created_at FROM users ORDER BY id`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.RoleID, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}
