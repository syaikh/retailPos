package user

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/config"
	"retail-pos-system/internal/shared"
)

const (
	maxLoginFailuresPerIP       = 10
	maxLoginFailuresPerUsername = 5
	loginFailureWindow          = 15 * time.Minute
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrTokenExpired       = errors.New("token has expired")
	ErrInvalidPassword    = errors.New("current password is incorrect")
)

type AuthService struct {
	dbPool        shared.DBPool
	repo          *Repository
	auditSvc      audit.AuditCreator
	jwtSecret     string
	refreshSecret string
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

type AuthClaims struct {
	ID          int      `json:"id"`
	Username    string   `json:"username"`
	RoleID      int      `json:"role_id"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
	StoreID     *int     `json:"store_id,omitempty"`
	ReportsToID *int     `json:"reports_to,omitempty"`
	jwt.RegisteredClaims
}

func NewAuthService(repo *Repository, auditSvc audit.AuditCreator, cfg *config.Config) *AuthService {
	if cfg.JWTSecret == "" {
		panic("FATAL: JWT_SECRET environment variable is required.")
	}
	return &AuthService{
		dbPool:        repo.db,
		repo:          repo,
		auditSvc:      auditSvc,
		jwtSecret:     cfg.JWTSecret,
		refreshSecret: cfg.JWTSecretRefresh,
		accessTTL:     15 * time.Minute,
		refreshTTL:    7 * 24 * time.Hour,
	}
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*LoginResponse, error) {
	ip, _ := ctx.Value(shared.CtxKeyIPAddress).(string)
	ua, _ := ctx.Value(shared.CtxKeyUserAgent).(string)

	if ip != "" {
		count, err := s.repo.CountRecentLoginFailures(ctx, ip, time.Now().Add(-loginFailureWindow))
		if err != nil {
			slog.Warn("failed to count recent login failures", "error", err)
		} else if count >= maxLoginFailuresPerIP {
			s.logFailure(ctx, username, ip, ua, "rate limited")
			return nil, ErrInvalidCredentials
		}
	}

	unameCount, err := s.repo.CountRecentLoginFailuresByUsername(ctx, username, time.Now().Add(-loginFailureWindow))
	if err != nil {
		slog.Warn("failed to count recent login failures by username", "error", err)
	} else if unameCount >= maxLoginFailuresPerUsername {
		s.logFailure(ctx, username, ip, ua, "account locked")
		return nil, ErrInvalidCredentials
	}

	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		s.logFailure(ctx, username, ip, ua, "user not found")
		return nil, ErrInvalidCredentials
	}
	if !user.IsActive {
		s.logFailure(ctx, username, ip, ua, "inactive account")
		return nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		s.logFailure(ctx, username, ip, ua, "invalid password")
		return nil, ErrInvalidCredentials
	}

	if err := s.repo.UpdateLastLogin(ctx, user.ID); err != nil {
		slog.Warn("failed to update last_login", "user", user.ID, "error", err)
	}

	permissions, err := s.repo.GetRolePermissions(ctx, user.RoleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role permissions: %w", err)
	}
	perms := make([]string, len(permissions))
	for i, p := range permissions {
		perms[i] = p.Code
	}
	accessToken, err := s.generateToken(user, perms, s.accessTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}
	refreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	if err := s.storeRefreshToken(ctx, user.ID, refreshToken); err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	user.Password = ""
	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         *user,
	}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, oldRefreshToken string) (string, string, error) {
	claims, err := s.parseRefreshToken(oldRefreshToken)
	if err != nil {
		return "", "", err
	}

	tx, err := s.dbPool.Begin(ctx)
	if err != nil {
		return "", "", fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tokenHash := hashToken(oldRefreshToken)
	var deletedID int
	err = tx.QueryRow(ctx, `
		DELETE FROM refresh_tokens WHERE user_id = $1 AND token_hash = $2 RETURNING id
	`, claims.ID, tokenHash).Scan(&deletedID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", errors.New("invalid refresh token")
		}
		return "", "", fmt.Errorf("failed to invalidate old refresh token: %w", err)
	}

	user, err := s.repo.GetByID(ctx, claims.ID)
	if err != nil {
		return "", "", ErrUserNotFound
	}

	permissions, err := s.repo.GetRolePermissions(ctx, user.RoleID)
	if err != nil {
		return "", "", fmt.Errorf("failed to get role permissions: %w", err)
	}
	perms := make([]string, len(permissions))
	for i, p := range permissions {
		perms[i] = p.Code
	}

	newAccessToken, err := s.generateToken(user, perms, s.accessTTL)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate new access token: %w", err)
	}

	newRefreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate new refresh token: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, user.ID, hashToken(newRefreshToken), time.Now().Add(s.refreshTTL)); err != nil {
		return "", "", fmt.Errorf("failed to store new refresh token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", "", fmt.Errorf("commit refresh token rotation: %w", err)
	}

	return newAccessToken, newRefreshToken, nil
}

func (s *AuthService) Logout(ctx context.Context, userID int, refreshToken string) error {
	return s.deleteRefreshToken(ctx, userID, refreshToken)
}

func (s *AuthService) ValidateToken(tokenString string) (*AuthClaims, error) {
	return s.parseToken(tokenString)
}

func (s *AuthService) ChangePassword(ctx context.Context, userID int, currentPassword, newPassword string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPassword)); err != nil {
		return ErrInvalidPassword
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), 14)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	if err := s.repo.UpdatePassword(ctx, userID, string(hashed)); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	if err := s.repo.DeleteUserRefreshTokens(ctx, userID); err != nil {
		slog.Warn("failed to delete refresh tokens after password change", "user", userID, "error", err)
	}

	return nil
}

func (s *AuthService) logFailure(ctx context.Context, username, ip, ua, reason string) {
	if s.auditSvc == nil {
		return
	}
	_ = s.auditSvc.CreateAuditLog(ctx, &audit.AuditLog{
		Action:      "login_failed",
		EntityType:  "auth",
		Username:    username,
		IPAddress:   ip,
		UserAgent:   ua,
		Description: fmt.Sprintf("Failed login for %s: %s", username, reason),
	})
}

func (s *AuthService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}

func (s *AuthService) generateToken(user *User, permissions []string, ttl time.Duration) (string, error) {
	claims := AuthClaims{
		ID:          user.ID,
		Username:    user.Username,
		RoleID:      user.RoleID,
		Role:        user.Role.Name,
		Permissions: permissions,
		StoreID:     user.StoreID,
		ReportsToID: user.ReportsToID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "retail-pos-system",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *AuthService) generateRefreshToken(user *User) (string, error) {
	claims := AuthClaims{
		ID:          user.ID,
		Username:    user.Username,
		RoleID:      user.RoleID,
		Role:        user.Role.Name,
		ReportsToID: user.ReportsToID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.refreshTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "retail-pos-system-refresh",
			ID:        uuid.New().String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.refreshSecret))
}

func (s *AuthService) parseToken(tokenString string) (*AuthClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	if claims, ok := token.Claims.(*AuthClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}

func (s *AuthService) parseRefreshToken(tokenString string) (*AuthClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.refreshSecret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("failed to parse refresh token: %w", err)
	}
	if claims, ok := token.Claims.(*AuthClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid refresh token")
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func (s *AuthService) storeRefreshToken(ctx context.Context, userID int, token string) error {
	tokenHash := hashToken(token)
	_, err := s.dbPool.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, tokenHash, time.Now().Add(s.refreshTTL))
	return err
}

func (s *AuthService) deleteRefreshToken(ctx context.Context, userID int, token string) error {
	tokenHash := hashToken(token)
	_, err := s.dbPool.Exec(ctx, "DELETE FROM refresh_tokens WHERE user_id = $1 AND token_hash = $2", userID, tokenHash)
	return err
}
