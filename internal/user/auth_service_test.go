package user

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAuthService(t *testing.T) *AuthService {
	t.Helper()
	os.Setenv("JWT_SECRET", "test-secret-key-for-unit-tests-32bytes!!")
	t.Cleanup(func() { os.Unsetenv("JWT_SECRET") })
	return &AuthService{
		jwtSecret:     "test-secret-key-for-unit-tests-32bytes!!",
		refreshSecret: "test-refresh-secret-key-for-unit-tests-32b",
		accessTTL:     15 * time.Minute,
		refreshTTL:    7 * 24 * time.Hour,
	}
}

func TestHashToken(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"simple", "hello"},
		{"uuid-like", "550e8400-e29b-41d4-a716-446655440000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hashToken(tt.input)
			assert.Equal(t, 64, len(got), "SHA-256 hex should be 64 chars")
			assert.Equal(t, hashToken(tt.input), got, "same input should produce same hash")
		})
	}

	assert.NotEqual(t, hashToken("a"), hashToken("b"), "different inputs should produce different hashes")
}

func TestAuthService_HashPassword(t *testing.T) {
	svc := newTestAuthService(t)
	hashed, err := svc.HashPassword("mypassword123")
	require.NoError(t, err)
	assert.NotEmpty(t, hashed)
	assert.NotEqual(t, "mypassword123", hashed)

	// bcrypt hashes start with $2
	assert.True(t, strings.HasPrefix(hashed, "$2"))
}

func TestAuthService_GenerateAndParseToken(t *testing.T) {
	svc := newTestAuthService(t)
	storeID := 42
	user := &User{ID: 1, Username: "testuser", RoleID: 2, Role: Role{Name: "admin"}, StoreID: &storeID}
	perms := []string{"product.view", "sale.create"}

	token, err := svc.generateToken(user, perms, 15*time.Minute)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := svc.parseToken(token)
	require.NoError(t, err)
	assert.Equal(t, 1, claims.ID)
	assert.Equal(t, "testuser", claims.Username)
	assert.Equal(t, 2, claims.RoleID)
	assert.Equal(t, "admin", claims.Role)
	assert.Equal(t, perms, claims.Permissions)
	assert.NotNil(t, claims.StoreID)
	assert.Equal(t, 42, *claims.StoreID)
	assert.Equal(t, "retail-pos-system", claims.Issuer)
}

func TestAuthService_GenerateAndParseRefreshToken(t *testing.T) {
	svc := newTestAuthService(t)
	user := &User{ID: 5, Username: "refreshuser", RoleID: 1, Role: Role{Name: "cashier"}}

	token, err := svc.generateRefreshToken(user)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := svc.parseRefreshToken(token)
	require.NoError(t, err)
	assert.Equal(t, 5, claims.ID)
	assert.Equal(t, "refreshuser", claims.Username)
	assert.Equal(t, "retail-pos-system-refresh", claims.Issuer)
	assert.NotEmpty(t, claims.ID)
}

func TestAuthService_ParseToken_Expired(t *testing.T) {
	svc := newTestAuthService(t)
	user := &User{ID: 1, Username: "u", RoleID: 1, Role: Role{Name: "r"}}

	expiredToken, err := svc.generateToken(user, nil, -1*time.Hour)
	require.NoError(t, err)

	_, err = svc.parseToken(expiredToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestAuthService_ParseRefreshToken_Expired(t *testing.T) {
	svc := newTestAuthService(t)
	user := &User{ID: 1, Username: "u", RoleID: 1, Role: Role{Name: "r"}}

	// Manually create an expired token with the same secret
	svc2 := &AuthService{jwtSecret: svc.jwtSecret, refreshSecret: svc.refreshSecret, refreshTTL: -1 * time.Hour}
	expiredToken, err := svc2.generateRefreshToken(user)
	require.NoError(t, err)

	_, err = svc.parseRefreshToken(expiredToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestAuthService_ParseToken_Invalid(t *testing.T) {
	svc := newTestAuthService(t)

	_, err := svc.parseToken("not-a-jwt-token")
	assert.Error(t, err)

	_, err = svc.parseToken("")
	assert.Error(t, err)
}

func TestAuthService_ParseRefreshToken_Invalid(t *testing.T) {
	svc := newTestAuthService(t)

	_, err := svc.parseRefreshToken("not-a-jwt-token")
	assert.Error(t, err)
}

func TestAuthService_ParseToken_WrongSecret(t *testing.T) {
	svc1 := newTestAuthService(t)
	svc2 := &AuthService{jwtSecret: "completely-different-secret-32bytes!!!", accessTTL: 15 * time.Minute}

	user := &User{ID: 1, Username: "u", RoleID: 1, Role: Role{Name: "r"}}
	token, err := svc1.generateToken(user, nil, 15*time.Minute)
	require.NoError(t, err)

	_, err = svc2.parseToken(token)
	assert.Error(t, err)
}

func TestAuthService_ParseRefreshToken_WrongSecret(t *testing.T) {
	svc1 := newTestAuthService(t)
	svc2 := &AuthService{jwtSecret: "completely-different-secret-32bytes!!!", refreshSecret: "also-different-refresh-secret-32bytes!!", refreshTTL: 7 * 24 * time.Hour}

	user := &User{ID: 1, Username: "u", RoleID: 1, Role: Role{Name: "r"}}
	token, err := svc1.generateRefreshToken(user)
	require.NoError(t, err)

	_, err = svc2.parseRefreshToken(token)
	assert.Error(t, err)
}

func TestValidateSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/validate", nil)

	storeID := 7
	c.Set("userID", 42)
	c.Set("username", "admin")
	c.Set("role", "admin")
	c.Set("permissions", []string{"sale.create", "product.view"})
	c.Set("storeID", &storeID)

	handler := &AuthHandler{svc: nil, auditSvc: nil}
	handler.ValidateSession(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)

	user := body["user"].(map[string]interface{})
	assert.Equal(t, float64(42), user["id"])
	assert.Equal(t, "admin", user["username"])
	assert.Equal(t, "admin", user["role"])

	sid := user["store_id"].(float64)
	assert.Equal(t, float64(7), sid)

	perms := body["permissions"].([]interface{})
	assert.Equal(t, 2, len(perms))
}

func TestValidateSession_EmptyContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/validate", nil)

	handler := &AuthHandler{svc: nil, auditSvc: nil}
	handler.ValidateSession(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)

	user := body["user"].(map[string]interface{})
	assert.Equal(t, float64(0), user["id"])
	assert.Equal(t, "", user["username"])
	assert.Equal(t, "", user["role"])

	perms := body["permissions"].([]interface{})
	assert.Equal(t, 0, len(perms))
}

func TestAuthService_ParseToken_NonHMAC(t *testing.T) {
	svc := newTestAuthService(t)
	// jwt with alg: RS256 (non-HMAC) - valid base64url header
	token := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6MX0."
	_, err := svc.parseToken(token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected signing method")
}

func TestAuthService_ParseRefreshToken_NonHMAC(t *testing.T) {
	svc := newTestAuthService(t)
	token := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6MX0."
	_, err := svc.parseRefreshToken(token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected signing method")
}

func TestAuthService_ParseToken_Expired_IsSentinel(t *testing.T) {
	svc := newTestAuthService(t)
	user := &User{ID: 1, Username: "u", RoleID: 1, Role: Role{Name: "r"}}

	expiredToken, err := svc.generateToken(user, nil, -1*time.Hour)
	require.NoError(t, err)

	_, err = svc.parseToken(expiredToken)
	assert.ErrorIs(t, err, ErrTokenExpired)
}

func TestAuthService_ParseRefreshToken_Expired_IsSentinel(t *testing.T) {
	svc := newTestAuthService(t)
	user := &User{ID: 1, Username: "u", RoleID: 1, Role: Role{Name: "r"}}

	svc2 := &AuthService{jwtSecret: svc.jwtSecret, refreshSecret: svc.refreshSecret, refreshTTL: -1 * time.Hour}
	expiredToken, err := svc2.generateRefreshToken(user)
	require.NoError(t, err)

	_, err = svc.parseRefreshToken(expiredToken)
	assert.ErrorIs(t, err, ErrTokenExpired)
}

func TestAuthService_ValidateToken_Success(t *testing.T) {
	svc := newTestAuthService(t)
	user := &User{ID: 1, Username: "u", RoleID: 1, Role: Role{Name: "r"}}
	token, err := svc.generateToken(user, []string{"test:perm"}, 15*time.Minute)
	require.NoError(t, err)

	claims, err := svc.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, 1, claims.ID)
	assert.Equal(t, "u", claims.Username)
}

func TestAuthService_ValidateToken_Expired(t *testing.T) {
	svc := newTestAuthService(t)
	user := &User{ID: 1, Username: "u", RoleID: 1, Role: Role{Name: "r"}}
	token, err := svc.generateToken(user, nil, -1*time.Hour)
	require.NoError(t, err)

	_, err = svc.ValidateToken(token)
	assert.Error(t, err)
}

func TestAuthService_ValidateToken_Invalid(t *testing.T) {
	svc := newTestAuthService(t)

	_, err := svc.ValidateToken("garbage")
	assert.Error(t, err)
}

func newAuthServiceWithDB(t *testing.T) *AuthService {
	t.Helper()
	os.Setenv("JWT_SECRET", "test-secret-for-testing-only")
	t.Cleanup(func() { os.Unsetenv("JWT_SECRET") })
	repo := NewRepository(dbPool)
	return NewAuthService(repo)
}

func TestAuthService_Login_Success(t *testing.T) {
	svc := newAuthServiceWithDB(t)
	ctx := context.Background()
	hash := testPasswordHash()

	user := &User{
		Username: "login_success_test",
		Email:    "login_success@test.com",
		Password: hash,
		RoleID:   1,
		IsActive: true,
	}
	err := NewRepository(dbPool).CreateUser(ctx, user)
	require.NoError(t, err)

	resp, err := svc.Login(ctx, "login_success_test", "password")
	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, "", resp.User.Password)
	assert.Equal(t, "login_success_test", resp.User.Username)
	assert.Equal(t, 1, resp.User.RoleID)
}

func TestAuthService_Login_InvalidCredentials(t *testing.T) {
	svc := newAuthServiceWithDB(t)
	ctx := context.Background()
	hash := testPasswordHash()

	user := &User{
		Username: "login_invalid_test",
		Email:    "login_invalid@test.com",
		Password: hash,
		RoleID:   1,
		IsActive: true,
	}
	err := NewRepository(dbPool).CreateUser(ctx, user)
	require.NoError(t, err)

	_, err = svc.Login(ctx, "login_invalid_test", "wrongpassword")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestAuthService_Login_InactiveUser(t *testing.T) {
	svc := newAuthServiceWithDB(t)
	ctx := context.Background()
	hash := testPasswordHash()

	user := &User{
		Username: "inactive_auth_test",
		Email:    "inactive_auth@test.com",
		Password: hash,
		RoleID:   1,
		IsActive: false,
	}
	err := NewRepository(dbPool).CreateUser(ctx, user)
	require.NoError(t, err)

	_, err = svc.Login(ctx, "inactive_auth_test", "password")
	assert.Error(t, err)
	assert.Equal(t, "invalid username or password", err.Error())
}

func TestAuthService_RefreshToken_Success(t *testing.T) {
	svc := newAuthServiceWithDB(t)
	ctx := context.Background()
	hash := testPasswordHash()

	user := &User{
		Username: "refresh_success_test",
		Email:    "refresh_success@test.com",
		Password: hash,
		RoleID:   1,
		IsActive: true,
	}
	err := NewRepository(dbPool).CreateUser(ctx, user)
	require.NoError(t, err)

	resp, err := svc.Login(ctx, "refresh_success_test", "password")
	require.NoError(t, err)

	newAccess, newRefresh, err := svc.RefreshToken(ctx, resp.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, newAccess)
	assert.NotEmpty(t, newRefresh)

	_, _, err = svc.RefreshToken(ctx, resp.RefreshToken)
	assert.Error(t, err)
}

func TestAuthService_RefreshToken_InvalidToken(t *testing.T) {
	svc := newAuthServiceWithDB(t)
	ctx := context.Background()

	_, _, err := svc.RefreshToken(ctx, "not-a-jwt-token")
	assert.Error(t, err)
}

func TestAuthService_RefreshToken_TokenNotFound(t *testing.T) {
	svc := newAuthServiceWithDB(t)
	ctx := context.Background()

	user := &User{ID: 999, Username: "ghost", RoleID: 1, Role: Role{Name: "admin"}}
	token, err := svc.generateRefreshToken(user)
	require.NoError(t, err)

	_, _, err = svc.RefreshToken(ctx, token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid refresh token")
}

func TestAuthService_Logout_Success(t *testing.T) {
	svc := newAuthServiceWithDB(t)
	ctx := context.Background()
	hash := testPasswordHash()

	user := &User{
		Username: "logout_success_test",
		Email:    "logout_success@test.com",
		Password: hash,
		RoleID:   1,
		IsActive: true,
	}
	err := NewRepository(dbPool).CreateUser(ctx, user)
	require.NoError(t, err)

	resp, err := svc.Login(ctx, "logout_success_test", "password")
	require.NoError(t, err)

	err = svc.Logout(ctx, resp.User.ID, resp.RefreshToken)
	require.NoError(t, err)

	_, _, err = svc.RefreshToken(ctx, resp.RefreshToken)
	assert.Error(t, err)
}

func TestAuthService_ChangePassword_Success(t *testing.T) {
	svc := newAuthServiceWithDB(t)
	ctx := context.Background()
	hash := testPasswordHash()

	user := &User{
		Username: "changepw_success",
		Email:    "changepw_success@test.com",
		Password: hash,
		RoleID:   1,
		IsActive: true,
	}
	err := NewRepository(dbPool).CreateUser(ctx, user)
	require.NoError(t, err)

	resp, err := svc.Login(ctx, "changepw_success", "password")
	require.NoError(t, err)

	err = svc.ChangePassword(ctx, user.ID, "password", "newpassword456")
	require.NoError(t, err)

	_, _, err = svc.RefreshToken(ctx, resp.RefreshToken)
	assert.Error(t, err)

	resp2, err := svc.Login(ctx, "changepw_success", "newpassword456")
	require.NoError(t, err)
	assert.Equal(t, "changepw_success", resp2.User.Username)
}

func TestAuthService_ChangePassword_WrongCurrent(t *testing.T) {
	svc := newAuthServiceWithDB(t)
	ctx := context.Background()
	hash := testPasswordHash()

	user := &User{
		Username: "changepw_wrong",
		Email:    "changepw_wrong@test.com",
		Password: hash,
		RoleID:   1,
		IsActive: true,
	}
	err := NewRepository(dbPool).CreateUser(ctx, user)
	require.NoError(t, err)

	err = svc.ChangePassword(ctx, user.ID, "wrongpassword", "newpassword")
	assert.ErrorIs(t, err, ErrInvalidPassword)
}

func TestAuthService_ChangePassword_UserNotFound(t *testing.T) {
	svc := newAuthServiceWithDB(t)
	ctx := context.Background()

	err := svc.ChangePassword(ctx, 99999, "any", "newpassword")
	assert.ErrorIs(t, err, ErrUserNotFound)
}
