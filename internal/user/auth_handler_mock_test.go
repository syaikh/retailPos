package user

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAuthLoginService struct {
	loginFn          func(ctx context.Context, username, password string) (*LoginResponse, error)
	refreshTokenFn   func(ctx context.Context, oldRefreshToken string) (string, string, error)
	validateTokenFn  func(tokenString string) (*AuthClaims, error)
	changePasswordFn func(ctx context.Context, userID int, currentPassword, newPassword string) error
	logoutFn         func(ctx context.Context, userID int, refreshToken string) error
	hashPasswordFn   func(password string) (string, error)
}

func (m *mockAuthLoginService) Login(ctx context.Context, username, password string) (*LoginResponse, error) {
	return m.loginFn(ctx, username, password)
}
func (m *mockAuthLoginService) RefreshToken(ctx context.Context, oldRefreshToken string) (string, string, error) {
	return m.refreshTokenFn(ctx, oldRefreshToken)
}
func (m *mockAuthLoginService) ValidateToken(tokenString string) (*AuthClaims, error) {
	return m.validateTokenFn(tokenString)
}
func (m *mockAuthLoginService) ChangePassword(ctx context.Context, userID int, currentPassword, newPassword string) error {
	return m.changePasswordFn(ctx, userID, currentPassword, newPassword)
}
func (m *mockAuthLoginService) Logout(ctx context.Context, userID int, refreshToken string) error {
	return m.logoutFn(ctx, userID, refreshToken)
}
func (m *mockAuthLoginService) HashPassword(password string) (string, error) {
	if m.hashPasswordFn != nil {
		return m.hashPasswordFn(password)
	}
	return "$2a$10$hashedpassword", nil
}

var _ AuthLoginService = (*mockAuthLoginService)(nil)

func setupMockAuthRouter(svc AuthLoginService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewAuthHandler(svc, nil)
	h.RegisterLoginRoute(r.Group("/auth"), func(c *gin.Context) { c.Next() })
	h.RegisterRefreshRoute(r.Group("/auth"))
	h.RegisterChangePasswordRoute(r.Group("/auth"))
	r.POST("/auth/validate", func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "testuser")
		c.Set("role", "admin")
		c.Set("permissions", []string{"user.view"})
		c.Set("storeID", (*int)(nil))
		h.ValidateSession(c)
	})
	r.POST("/auth/logout", func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "testuser")
		c.Set("role", "admin")
		h.Logout(c)
	})
	return r
}

func TestAuthHandler_Login_Success(t *testing.T) {
	svc := &mockAuthLoginService{
		loginFn: func(ctx context.Context, username, password string) (*LoginResponse, error) {
			assert.Equal(t, "admin", username)
			assert.Equal(t, "password123", password)
			return &LoginResponse{
				AccessToken:  "access-token-abc",
				RefreshToken: "refresh-token-xyz",
				User: User{
					ID:       1,
					Username: "admin",
				},
			}, nil
		},
	}
	r := setupMockAuthRouter(svc)
	body := `{"username":"admin","password":"password123"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "access-token-abc", resp["access_token"])
}

func TestAuthHandler_Login_InvalidJSON(t *testing.T) {
	r := setupMockAuthRouter(&mockAuthLoginService{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader("{bad"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthHandler_Login_ServiceError(t *testing.T) {
	svc := &mockAuthLoginService{
		loginFn: func(ctx context.Context, username, password string) (*LoginResponse, error) {
			return nil, errors.New("invalid credentials")
		},
	}
	r := setupMockAuthRouter(svc)
	body := `{"username":"admin","password":"wrong"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid credentials")
}

func TestAuthHandler_Login_PasswordEmpty(t *testing.T) {
	svc := &mockAuthLoginService{
		loginFn: func(ctx context.Context, username, password string) (*LoginResponse, error) {
			return nil, errors.New("invalid credentials")
		},
	}
	r := setupMockAuthRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(`{"username":"admin","password":""}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandler_RefreshToken_Success(t *testing.T) {
	svc := &mockAuthLoginService{
		refreshTokenFn: func(ctx context.Context, oldRefreshToken string) (string, string, error) {
			assert.Equal(t, "old-refresh-token", oldRefreshToken)
			return "new-access-token", "new-refresh-token", nil
		},
	}
	r := setupMockAuthRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/auth/refresh", nil)
	req.Header.Set("X-Refresh-Token", "old-refresh-token")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "new-access-token", resp["access_token"])
}

func TestAuthHandler_RefreshToken_FromCookie(t *testing.T) {
	svc := &mockAuthLoginService{
		refreshTokenFn: func(ctx context.Context, oldRefreshToken string) (string, string, error) {
			assert.Equal(t, "cookie-token", oldRefreshToken)
			return "new-access", "new-refresh", nil
		},
	}
	r := setupMockAuthRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "cookie-token"})
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_RefreshToken_Missing(t *testing.T) {
	r := setupMockAuthRouter(&mockAuthLoginService{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/auth/refresh", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "refresh token is required")
}

func TestAuthHandler_RefreshToken_ServiceError(t *testing.T) {
	svc := &mockAuthLoginService{
		refreshTokenFn: func(ctx context.Context, oldRefreshToken string) (string, string, error) {
			return "", "", errors.New("token expired")
		},
	}
	r := setupMockAuthRouter(svc)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/auth/refresh", nil)
	req.Header.Set("X-Refresh-Token", "expired-token")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "token expired")
}

func TestAuthHandler_ValidateSession_Success(t *testing.T) {
	r := setupMockAuthRouter(&mockAuthLoginService{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/auth/validate", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	user, ok := resp["user"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(1), user["id"])
	assert.Equal(t, "testuser", user["username"])
	assert.Equal(t, "admin", user["role"])
}

func TestAuthHandler_ValidateSession_Permissions(t *testing.T) {
	r := setupMockAuthRouter(&mockAuthLoginService{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/auth/validate", nil)
	r.ServeHTTP(w, req)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	perms, ok := resp["permissions"].([]interface{})
	require.True(t, ok)
	assert.Contains(t, perms, "user.view")
}

func TestAuthHandler_ChangePassword_Success(t *testing.T) {
	svc := &mockAuthLoginService{
		changePasswordFn: func(ctx context.Context, userID int, currentPassword, newPassword string) error {
			assert.Equal(t, 1, userID)
			assert.Equal(t, "oldpass123", currentPassword)
			assert.Equal(t, "newpass456", newPassword)
			return nil
		},
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewAuthHandler(svc, nil)
	r.POST("/change-password", func(c *gin.Context) {
		c.Set("userID", 1)
		h.ChangePassword(c)
	})

	body := `{"current_password":"oldpass123","new_password":"newpass456"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/change-password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "password changed")
}

func TestAuthHandler_ChangePassword_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	svc := &mockAuthLoginService{}
	h := NewAuthHandler(svc, nil)
	r.POST("/change-password", func(c *gin.Context) {
		c.Set("userID", 1)
		h.ChangePassword(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/change-password", strings.NewReader("{bad"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthHandler_ChangePassword_MissingNewPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	svc := &mockAuthLoginService{}
	h := NewAuthHandler(svc, nil)
	r.POST("/change-password", func(c *gin.Context) {
		c.Set("userID", 1)
		h.ChangePassword(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/change-password", strings.NewReader(`{"current_password":"old"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthHandler_ChangePassword_InvalidUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	svc := &mockAuthLoginService{}
	h := NewAuthHandler(svc, nil)
	r.POST("/change-password", func(c *gin.Context) {
		c.Set("userID", "not-an-int")
		h.ChangePassword(c)
	})

	body := `{"current_password":"oldpass123","new_password":"newpass456"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/change-password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid user session")
}

func TestAuthHandler_ChangePassword_WrongPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	svc := &mockAuthLoginService{
		changePasswordFn: func(ctx context.Context, userID int, currentPassword, newPassword string) error {
			return ErrInvalidPassword
		},
	}
	h := NewAuthHandler(svc, nil)
	r.POST("/change-password", func(c *gin.Context) {
		c.Set("userID", 1)
		h.ChangePassword(c)
	})

	body := `{"current_password":"wrongpass","new_password":"newpass456"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/change-password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandler_ChangePassword_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	svc := &mockAuthLoginService{
		changePasswordFn: func(ctx context.Context, userID int, currentPassword, newPassword string) error {
			return errors.New("db error")
		},
	}
	h := NewAuthHandler(svc, nil)
	r.POST("/change-password", func(c *gin.Context) {
		c.Set("userID", 1)
		h.ChangePassword(c)
	})

	body := `{"current_password":"oldpass123","new_password":"newpass456"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/change-password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAuthHandler_Logout_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	svc := &mockAuthLoginService{
		logoutFn: func(ctx context.Context, userID int, refreshToken string) error {
			assert.Equal(t, 1, userID)
			assert.Equal(t, "some-token", refreshToken)
			return nil
		},
	}
	h := NewAuthHandler(svc, nil)
	r.POST("/logout", func(c *gin.Context) {
		c.Set("userID", 1)
		h.Logout(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/logout", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "some-token"})
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "logged out")
}

func TestAuthHandler_Logout_InvalidUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	svc := &mockAuthLoginService{}
	h := NewAuthHandler(svc, nil)
	r.POST("/logout", func(c *gin.Context) {
		h.Logout(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/logout", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandler_Logout_NoRefreshToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	svc := &mockAuthLoginService{
		logoutFn: func(ctx context.Context, userID int, refreshToken string) error {
			t.Fatal("logout should not be called without refresh token")
			return nil
		},
	}
	h := NewAuthHandler(svc, nil)
	r.POST("/logout", func(c *gin.Context) {
		c.Set("userID", 1)
		h.Logout(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/logout", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_Logout_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	svc := &mockAuthLoginService{
		logoutFn: func(ctx context.Context, userID int, refreshToken string) error {
			return errors.New("db error")
		},
	}
	h := NewAuthHandler(svc, nil)
	r.POST("/logout", func(c *gin.Context) {
		c.Set("userID", 1)
		h.Logout(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/logout", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "some-token"})
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
