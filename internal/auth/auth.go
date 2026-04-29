package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"retail-pos-system/internal/domain"
	"retail-pos-system/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrTokenExpired       = errors.New("token has expired")
)

type AuthService struct {
	repo       repository.UserRepository
	dbPool     *pgxpool.Pool
	jwtSecret  string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

type Claims struct {
	ID          int      `json:"id"`
	Username    string   `json:"username"`
	RoleID      int      `json:"role_id"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
	StoreID     *int     `json:"store_id,omitempty"`
	jwt.RegisteredClaims
}

func NewAuthService(repo repository.UserRepository, dbPool *pgxpool.Pool) *AuthService {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "your-secret-key-change-in-production"
	}
	return &AuthService{
		repo:       repo,
		dbPool:     dbPool,
		jwtSecret:  secret,
		accessTTL:  15 * time.Minute,
		refreshTTL: 7 * 24 * time.Hour,
	}
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*domain.LoginResponse, error) {
	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if !user.IsActive {
		return nil, errors.New("user account is inactive")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	var perms []string
	if rp, ok := s.repo.(interface{ GetRolePermissions(context.Context, int) ([]domain.Permission, error) }); ok {
		permissions, _ := rp.GetRolePermissions(ctx, user.RoleID)
		perms = make([]string, len(permissions))
		for i, p := range permissions {
			perms[i] = p.Code
		}
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

	return &domain.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         *user,
	}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, oldRefreshToken string) (string, error) {
	claims, err := s.parseToken(oldRefreshToken, s.refreshTTL)
	if err != nil {
		return "", err
	}
	if exists, _ := s.refreshTokenExists(ctx, claims.ID, oldRefreshToken); !exists {
		return "", errors.New("invalid refresh token")
	}
	user, err := s.repo.GetByID(claims.ID)
	if err != nil {
		return "", ErrUserNotFound
	}
	var perms []string
	if rp, ok := s.repo.(interface{ GetRolePermissions(context.Context, int) ([]domain.Permission, error) }); ok {
		permissions, _ := rp.GetRolePermissions(ctx, user.RoleID)
		perms = make([]string, len(permissions))
		for i, p := range permissions {
			perms[i] = p.Code
		}
	}
	newAccessToken, err := s.generateToken(user, perms, s.accessTTL)
	if err != nil {
		return "", fmt.Errorf("failed to generate new access token: %w", err)
	}
	return newAccessToken, nil
}

func (s *AuthService) Logout(ctx context.Context, userID int, refreshToken string) error {
	return s.deleteRefreshToken(ctx, userID, refreshToken)
}

func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	return s.parseToken(tokenString, s.accessTTL)
}

func (s *AuthService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}

func (s *AuthService) generateToken(user *domain.User, permissions []string, ttl time.Duration) (string, error) {
	claims := Claims{
		ID:          user.ID,
		Username:    user.Username,
		RoleID:      user.RoleID,
		Role:        user.Role.Name,
		Permissions: permissions,
		StoreID:     user.StoreID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "retail-pos-system",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *AuthService) generateRefreshToken(user *domain.User) (string, error) {
	claims := Claims{
		ID:       user.ID,
		Username: user.Username,
		RoleID:   user.RoleID,
		Role:     user.Role.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.refreshTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "retail-pos-system-refresh",
			ID:        uuid.New().String(), // unique JTI to prevent token collisions
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret + "-refresh"))
}

func (s *AuthService) parseToken(tokenString string, ttl time.Duration) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})
	if err != nil {
		var ve interface{ Errors() uint32 }
		if errors.As(err, &ve) {
			if ve.Errors()&1 != 0 {
				return nil, ErrTokenExpired
			}
		}
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}

func (s *AuthService) storeRefreshToken(ctx context.Context, userID int, token string) error {
	_, err := s.dbPool.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, NOW() + INTERVAL '7 days')
	`, userID, token)
	return err
}

func (s *AuthService) refreshTokenExists(ctx context.Context, userID int, token string) (bool, error) {
	var exists bool
	err := s.dbPool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM refresh_tokens WHERE user_id = $1 AND token_hash = $2 AND expires_at > NOW())
	`, userID, token).Scan(&exists)
	return exists, err
}

func (s *AuthService) deleteRefreshToken(ctx context.Context, userID int, token string) error {
	_, err := s.dbPool.Exec(ctx, "DELETE FROM refresh_tokens WHERE user_id = $1 AND token_hash = $2", userID, token)
	return err
}
