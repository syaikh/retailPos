package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"retail-pos-system/internal/user"
)

func TestGetUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name   string
		userID interface{}
		setKey bool
		want   int
	}{
		{
			name:   "userID set",
			userID: 42,
			setKey: true,
			want:   42,
		},
		{
			name:   "userID not set",
			setKey: false,
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			if tt.setKey {
				c.Set("userID", tt.userID)
			}

			got := GetUserID(c)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetUserRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name   string
		role   interface{}
		setKey bool
		want   string
	}{
		{
			name:   "role set",
			role:   "admin",
			setKey: true,
			want:   "admin",
		},
		{
			name:   "role not set",
			setKey: false,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			if tt.setKey {
				c.Set("role", tt.role)
			}

			got := GetUserRole(c)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name   string
		perms  interface{}
		setKey bool
		want   []string
	}{
		{
			name:   "permissions set",
			perms:  []string{"a", "b"},
			setKey: true,
			want:   []string{"a", "b"},
		},
		{
			name:   "permissions not set",
			setKey: false,
			want:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			if tt.setKey {
				c.Set("permissions", tt.perms)
			}

			got := GetPermissions(c)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetStoreID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("storeID set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		val := 5
		c.Set("storeID", &val)

		got := GetStoreID(c)
		assert.NotNil(t, got)
		assert.Equal(t, 5, *got)
	})

	t.Run("storeID not set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		got := GetStoreID(c)
		assert.Nil(t, got)
	})
}

func TestHasPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name      string
		userPerms []string
		required  string
		want      bool
	}{
		{
			name:      "permission exists",
			userPerms: []string{"a", "b", "c"},
			required:  "b",
			want:      true,
		},
		{
			name:      "permission not found",
			userPerms: []string{"a", "b"},
			required:  "c",
			want:      false,
		},
		{
			name:      "empty permissions",
			userPerms: []string{},
			required:  "a",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasPermission(tt.userPerms, tt.required)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		authHeader  string
		cookieValue string
		setCookie   bool
		want        string
	}{
		{
			name:       "Bearer token",
			authHeader: "Bearer abc123",
			want:       "abc123",
		},
		{
			name:       "Token scheme",
			authHeader: "Token xyz789",
			want:       "xyz789",
		},
		{
			name:       "wrong scheme",
			authHeader: "Basic abc",
			want:       "",
		},
		{
			name:       "no header no cookie",
			authHeader: "",
			setCookie:  false,
			want:       "",
		},
		{
			name:        "cookie fallback",
			authHeader:  "",
			setCookie:   true,
			cookieValue: "tok123",
			want:        "tok123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authHeader != "" {
				c.Request.Header.Set("Authorization", tt.authHeader)
			}
			if tt.setCookie {
				c.Request.AddCookie(&http.Cookie{Name: "access_token", Value: tt.cookieValue})
			}

			got := extractToken(c)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRoleMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		role         interface{}
		setRole      bool
		requiredRole string
		wantCode     int
	}{
		{
			name:         "matching role passes",
			role:         "admin",
			setRole:      true,
			requiredRole: "admin",
			wantCode:     http.StatusOK,
		},
		{
			name:         "non-matching role returns 403",
			role:         "cashier",
			setRole:      true,
			requiredRole: "admin",
			wantCode:     http.StatusForbidden,
		},
		{
			name:         "no role returns 401",
			setRole:      false,
			requiredRole: "admin",
			wantCode:     http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.setRole {
				c.Set("role", tt.role)
			}

			middleware := RoleMiddleware(tt.requiredRole)
			middleware(c)

			assert.Equal(t, tt.wantCode, w.Code)
		})
	}
}

func TestRequirePermission(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		perms    interface{}
		setPerms bool
		required string
		wantCode int
	}{
		{
			name:     "permission granted",
			perms:    []string{"sale.create", "sale.view"},
			setPerms: true,
			required: "sale.create",
			wantCode: http.StatusOK,
		},
		{
			name:     "permission denied",
			perms:    []string{"sale.view"},
			setPerms: true,
			required: "sale.create",
			wantCode: http.StatusForbidden,
		},
		{
			name:     "no permissions in context",
			setPerms: false,
			required: "sale.create",
			wantCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.setPerms {
				c.Set("permissions", tt.perms)
			}

			middleware := RequirePermission(tt.required)
			middleware(c)

			assert.Equal(t, tt.wantCode, w.Code)
		})
	}
}

func TestRequireAnyPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		perms       interface{}
		setPerms    bool
		requiredAny []string
		wantCode    int
	}{
		{
			name:        "first permission matches",
			perms:       []string{"sale.create"},
			setPerms:    true,
			requiredAny: []string{"sale.create", "sale.view"},
			wantCode:    http.StatusOK,
		},
		{
			name:        "no permission matches",
			perms:       []string{"report.view"},
			setPerms:    true,
			requiredAny: []string{"sale.create", "sale.view"},
			wantCode:    http.StatusForbidden,
		},
		{
			name:        "no permissions in context",
			setPerms:    false,
			requiredAny: []string{"sale.create", "sale.view"},
			wantCode:    http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.setPerms {
				c.Set("permissions", tt.perms)
			}

			middleware := RequireAnyPermission(tt.requiredAny...)
			middleware(c)

			assert.Equal(t, tt.wantCode, w.Code)
		})
	}
}

func TestAdminOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		role     interface{}
		setRole  bool
		wantCode int
	}{
		{
			name:     "superadmin passes",
			role:     "superadmin",
			setRole:  true,
			wantCode: http.StatusOK,
		},
		{
			name:     "admin passes",
			role:     "admin",
			setRole:  true,
			wantCode: http.StatusOK,
		},
		{
			name:     "cashier returns 403",
			role:     "cashier",
			setRole:  true,
			wantCode: http.StatusForbidden,
		},
		{
			name:     "no role returns 401",
			setRole:  false,
			wantCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.setRole {
				c.Set("role", tt.role)
			}

			middleware := AdminOnly()
			middleware(c)

			assert.Equal(t, tt.wantCode, w.Code)
		})
	}
}

func generateTestToken(secret string, claims jwt.RegisteredClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(secret))
	return signed
}

func TestNewModularAuthMiddleware_NoToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	authService := user.NewAuthServiceForTest("test-secret")
	middleware := NewModularAuthMiddleware(authService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.RemoteAddr = "127.0.0.1:8080"

	middleware(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestNewModularAuthMiddleware_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	authService := user.NewAuthServiceForTest("test-secret")
	middleware := NewModularAuthMiddleware(authService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set("Authorization", "Bearer invalid-token")
	c.Request.RemoteAddr = "127.0.0.1:8080"

	middleware(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestNewModularAuthMiddleware_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	authService := user.NewAuthServiceForTest("test-secret")
	middleware := NewModularAuthMiddleware(authService)

	authClaims := user.AuthClaims{
		ID:         1,
		Username:   "testuser",
		RoleID:     1,
		Role:       "superadmin",
		Permissions: []string{"sale.create", "report.view"},
		StoreID:    intPtr(2),
	}
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Subject:   "1",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":          authClaims.ID,
		"username":    authClaims.Username,
		"role_id":     authClaims.RoleID,
		"role":        authClaims.Role,
		"permissions": authClaims.Permissions,
		"store_id":    authClaims.StoreID,
		"exp":         claims.ExpiresAt.Unix(),
		"iat":         claims.IssuedAt.Unix(),
		"sub":         claims.Subject,
	})
	signedToken, _ := token.SignedString([]byte("test-secret"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set("Authorization", "Bearer "+signedToken)
	c.Request.Header.Set("User-Agent", "test-agent")
	c.Request.RemoteAddr = "127.0.0.1:8080"

	middleware(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, c.GetInt("userID"))
	assert.Equal(t, "testuser", c.GetString("username"))
	assert.Equal(t, "superadmin", c.GetString("role"))
	assert.Equal(t, 1, c.GetInt("roleID"))
}

func TestNewModularAuthMiddleware_CookieFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	authService := user.NewAuthServiceForTest("test-secret")
	middleware := NewModularAuthMiddleware(authService)

	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Subject:   "1",
	}
	token := generateTestToken("test-secret", claims)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.AddCookie(&http.Cookie{Name: "access_token", Value: token})
	c.Request.Header.Set("User-Agent", "test-agent")
	c.Request.RemoteAddr = "127.0.0.1:8080"

	middleware(c)

	assert.Equal(t, http.StatusOK, w.Code)
}
