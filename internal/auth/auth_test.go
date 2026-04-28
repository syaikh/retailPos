package auth

import (
	"context"
	"os"
	"testing"
	"time"

	"retail-pos-system/internal/domain"
	"retail-pos-system/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthService_NewAuthService(t *testing.T) {
	testDB := repository.NewTestDB(t)
	defer testDB.Close(t)

	repo := repository.NewPostgresRepository(testDB.Pool())
	authService := NewAuthService(repo, testDB.Pool())

	assert.NotNil(t, authService)
	assert.Equal(t, "your-secret-key-change-in-production", authService.jwtSecret)
	assert.Equal(t, 15*time.Minute, authService.accessTTL)
	assert.Equal(t, 7*24*time.Hour, authService.refreshTTL)
}

func TestAuthService_NewAuthService_WithEnv(t *testing.T) {
	// Set environment variable
	originalSecret := os.Getenv("JWT_SECRET")
	defer func() {
		if originalSecret != "" {
			os.Setenv("JWT_SECRET", originalSecret)
		} else {
			os.Unsetenv("JWT_SECRET")
		}
	}()

	os.Setenv("JWT_SECRET", "test-secret-key")

	testDB := repository.NewTestDB(t)
	defer testDB.Close(t)

	repo := repository.NewPostgresRepository(testDB.Pool())
	authService := NewAuthService(repo, testDB.Pool())

	assert.Equal(t, "test-secret-key", authService.jwtSecret)
}

func TestAuthService_Login_Success(t *testing.T) {
	testDB := repository.NewTestDB(t)
	defer testDB.Close(t)

	repo := repository.NewPostgresRepository(testDB.Pool())
	authService := NewAuthService(repo, testDB.Pool())

	// Test login with superadmin user from seed data
	response, err := authService.Login(context.Background(), "superadmin", "admin123")

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotEmpty(t, response.AccessToken)
	assert.NotEmpty(t, response.RefreshToken)
	assert.Equal(t, 1, response.User.ID)
	assert.Equal(t, "superadmin", response.User.Username)
	assert.Empty(t, response.User.Password) // Password should be cleared

	// Verify tokens can be parsed
	claims, err := authService.parseToken(response.AccessToken, authService.accessTTL)
	require.NoError(t, err)
	assert.Equal(t, 1, claims.ID)
	assert.Equal(t, "superadmin", claims.Username)

	// Superadmin should have all permissions
	assert.Greater(t, len(claims.Permissions), 20) // Should have many permissions
	assert.Contains(t, claims.Permissions, "product:create")
	assert.Contains(t, claims.Permissions, "user:manage")
	assert.Contains(t, claims.Permissions, "sale:create")
}

func TestAuthService_Login_InvalidCredentials(t *testing.T) {
	testDB := repository.NewTestDB(t)
	defer testDB.Close(t)

	repo := repository.NewPostgresRepository(testDB.Pool())
	authService := NewAuthService(repo, testDB.Pool())

	// Test with wrong password
	response, err := authService.Login(context.Background(), "superadmin", "wrongpass")

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidCredentials, err)
	assert.Nil(t, response)
}

func TestAuthService_Login_InactiveUser(t *testing.T) {
	testDB := repository.NewTestDB(t)
	defer testDB.Close(t)

	repo := repository.NewPostgresRepository(testDB.Pool())

	// Update an existing user to be inactive for testing
	_, err := testDB.Pool().Exec(context.Background(),
		"UPDATE users SET is_active = false WHERE username = 'cashier'")
	require.NoError(t, err)

	authService := NewAuthService(repo, testDB.Pool())

	// Test login with inactive user
	response, err := authService.Login(context.Background(), "cashier", "admin123")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user account is inactive")
	assert.Nil(t, response)
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	testDB := repository.NewTestDB(t)
	defer testDB.Close(t)

	repo := repository.NewPostgresRepository(testDB.Pool())
	authService := NewAuthService(repo, testDB.Pool())

	// Test login with non-existent user
	response, err := authService.Login(context.Background(), "nonexistent", "anypassword")

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidCredentials, err)
	assert.Nil(t, response)
}

func TestAuthService_GenerateToken(t *testing.T) {
	testDB := repository.NewTestDB(t)
	defer testDB.Close(t)

	repo := repository.NewPostgresRepository(testDB.Pool())
	authService := NewAuthService(repo, testDB.Pool())

	user := &domain.User{
		ID:       1,
		Username: "testuser",
		RoleID:   2,
	}
	permissions := []string{"product:read", "product:create"}
	ttl := 15 * time.Minute

	token, err := authService.generateToken(user, permissions, ttl)

	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Parse and verify token
	claims, err := authService.parseToken(token, ttl)
	require.NoError(t, err)

	assert.Equal(t, user.ID, claims.ID)
	assert.Equal(t, user.Username, claims.Username)
	assert.Equal(t, user.RoleID, claims.RoleID)
	assert.Equal(t, permissions, claims.Permissions)
	assert.True(t, claims.ExpiresAt.After(time.Now()))
	assert.True(t, claims.IssuedAt.Before(time.Now().Add(time.Second)))
}

func TestAuthService_ParseToken_Expired(t *testing.T) {
	testDB := repository.NewTestDB(t)
	defer testDB.Close(t)

	repo := repository.NewPostgresRepository(testDB.Pool())
	authService := NewAuthService(repo, testDB.Pool())

	// Create expired token (TTL of 0)
	user := &domain.User{ID: 1, Username: "testuser"}
	token, err := authService.generateToken(user, []string{}, 0)
	require.NoError(t, err)

	// Parse with short TTL (should fail due to expiration)
	_, err = authService.parseToken(token, time.Nanosecond)
	assert.Error(t, err)
}

func TestAuthService_ValidateToken(t *testing.T) {
	testDB := repository.NewTestDB(t)
	defer testDB.Close(t)

	repo := repository.NewPostgresRepository(testDB.Pool())
	authService := NewAuthService(repo, testDB.Pool())

	user := &domain.User{
		ID:       1,
		Username: "testuser",
		RoleID:   1,
	}
	permissions := []string{"product:read"}

	token, err := authService.generateToken(user, permissions, 15*time.Minute)
	require.NoError(t, err)

	// Validate token
	claims, err := authService.ValidateToken(token)
	require.NoError(t, err)

	assert.Equal(t, user.ID, claims.ID)
	assert.Equal(t, user.Username, claims.Username)
	assert.Equal(t, permissions, claims.Permissions)
}

func TestAuthService_ValidateToken_Invalid(t *testing.T) {
	testDB := repository.NewTestDB(t)
	defer testDB.Close(t)

	repo := repository.NewPostgresRepository(testDB.Pool())
	authService := NewAuthService(repo, testDB.Pool())

	// Test with invalid token
	_, err := authService.ValidateToken("invalid.jwt.token")
	assert.Error(t, err)
}

