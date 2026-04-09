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
	query := `SELECT id, username, password_hash, role, created_at FROM users WHERE username = $1`
	var u model.User
	if err := r.db.QueryRow(query, username).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}
