package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	model "retailPos/internal/model"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrUnauthorized        = errors.New("unauthorized access")
	ErrInternalServerError = errors.New("internal server error")
)

// UserRepository dependency to fetch user from database
type UserRepository interface {
	GetByUsername(username string) (*model.User, error)
	GetByID(id int) (*model.User, error)
	GetUserRole(userID int) (*model.Role, error)
	ListUserPermissions(userID int) ([]string, error)
}

type AuthService interface {
	Login(ctx context.Context, username, password, ip string) (*model.User, *TokenPair, error)
	RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error)
	Logout(ctx context.Context, refreshToken string) error
}

type authService struct {
	userRepo     UserRepository
	authRepo     AuthRepo
	tokenService TokenService
}

func NewAuthService(userRepo UserRepository, authRepo AuthRepo, tokenService TokenService) AuthService {
	return &authService{
		userRepo:     userRepo,
		authRepo:     authRepo,
		tokenService: tokenService,
	}
}

func (s *authService) Login(ctx context.Context, username, password, ip string) (*model.User, *TokenPair, error) {
	user, err := s.userRepo.GetByUsername(username)
	if err != nil || user == nil {
		s.authRepo.LogLoginAttempt(ctx, username, false, ip)
		return nil, nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		s.authRepo.LogLoginAttempt(ctx, username, false, ip)
		return nil, nil, ErrInvalidCredentials
	}

	// Load authoritative role and permissions
	role, err := s.userRepo.GetUserRole(user.ID)
	if err != nil {
		return nil, nil, ErrInternalServerError
	}
	user.Role = role.Name
	user.RoleID = role.ID

	permissions, err := s.userRepo.ListUserPermissions(user.ID)
	if err != nil {
		return nil, nil, ErrInternalServerError
	}
	user.Permissions = permissions

	// Password matches
	s.authRepo.LogLoginAttempt(ctx, username, true, ip)

	tokens, err := s.tokenService.GenerateTokenPair(user.ID, user.RoleID, user.Role)
	if err != nil {
		return nil, nil, ErrInternalServerError
	}

	// Store hashed refresh token
	tokenHash := hashToken(tokens.RefreshToken)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	if err := s.authRepo.StoreRefreshToken(ctx, tokenHash, user.ID, expiresAt); err != nil {
		return nil, nil, ErrInternalServerError
	}

	return user, tokens, nil
}

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	userID, err := s.tokenService.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	tokenHash := hashToken(refreshToken)
	dbUserID, err := s.authRepo.ValidateRefreshTokenHash(ctx, tokenHash)
	if err != nil || dbUserID != userID {
		return nil, ErrInvalidToken
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil || user == nil {
		return nil, ErrInvalidCredentials
	}

	// Remove old token (simple rotation by deleting previous token)
	s.authRepo.DeleteRefreshToken(ctx, tokenHash)

	// Generate new pair
	tokens, err := s.tokenService.GenerateTokenPair(user.ID, user.RoleID, user.Role)
	if err != nil {
		return nil, ErrInternalServerError
	}

	// Store new hashed refresh token
	newTokenHash := hashToken(tokens.RefreshToken)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	if err := s.authRepo.StoreRefreshToken(ctx, newTokenHash, user.ID, expiresAt); err != nil {
		return nil, ErrInternalServerError
	}

	return tokens, nil
}

func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := hashToken(refreshToken)
	return s.authRepo.DeleteRefreshToken(ctx, tokenHash)
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
