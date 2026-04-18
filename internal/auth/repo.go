package auth

import (
	"context"
	"database/sql"
	"time"
)

type AuthRepo interface {
	StoreRefreshToken(ctx context.Context, tokenHash string, userID int, expiresAt time.Time) error
	DeleteRefreshToken(ctx context.Context, tokenHash string) error
	ValidateRefreshTokenHash(ctx context.Context, tokenHash string) (int, error)
	LogLoginAttempt(ctx context.Context, username string, success bool, ip string) error
}

type postgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) AuthRepo {
	return &postgresRepo{db: db}
}

func (r *postgresRepo) StoreRefreshToken(ctx context.Context, tokenHash string, userID int, expiresAt time.Time) error {
	query := `INSERT INTO refresh_tokens (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`
	_, err := r.db.ExecContext(ctx, query, tokenHash, userID, expiresAt)
	return err
}

func (r *postgresRepo) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	query := `DELETE FROM refresh_tokens WHERE token_hash = $1`
	_, err := r.db.ExecContext(ctx, query, tokenHash)
	return err
}

func (r *postgresRepo) ValidateRefreshTokenHash(ctx context.Context, tokenHash string) (int, error) {
	query := `SELECT user_id FROM refresh_tokens WHERE token_hash = $1 AND expires_at > $2`
	var userID int
	err := r.db.QueryRowContext(ctx, query, tokenHash, time.Now()).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, ErrInvalidToken // Abstract err, to be defined in service.go
		}
		return 0, err
	}
	return userID, nil
}

func (r *postgresRepo) LogLoginAttempt(ctx context.Context, username string, success bool, ip string) error {
	query := `INSERT INTO audit_logs (username, success, ip_address) VALUES ($1, $2, $3)`
	_, err := r.db.ExecContext(ctx, query, username, success, ip)
	return err
}
