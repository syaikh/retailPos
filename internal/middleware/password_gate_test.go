package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"retail-pos-system/internal/shared"
	"retail-pos-system/internal/user"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// signedToken builds a valid access token the given auth middleware accepts,
// optionally carrying the must_change_password claim.
func signedToken(t *testing.T, mustChange bool) string {
	t.Helper()
	claims := jwt.MapClaims{
		"id":          1,
		"username":    "rotation_user",
		"role_id":     1,
		"role":        "cashier",
		"permissions": []string{"sale.create"},
		"exp":         time.Now().Add(15 * time.Minute).Unix(),
		"iat":         time.Now().Unix(),
		"sub":         "1",
	}
	if mustChange {
		claims["must_change_password"] = true
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("test-secret"))
	require.NoError(t, err)
	return token
}

func gateRequest(t *testing.T, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	mw := NewModularAuthMiddleware(user.NewAuthServiceForTest("test-secret"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, path, nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)
	c.Request.RemoteAddr = "127.0.0.1:8080"

	mw(c)
	return w
}

func TestPasswordChangeGate_AbortsWith428(t *testing.T) {
	w := gateRequest(t, "/", signedToken(t, true))

	assert.Equal(t, http.StatusPreconditionRequired, w.Code)
	assert.Contains(t, w.Body.String(), shared.ErrPasswordChangeRequired)
	assert.Contains(t, w.Body.String(), "password must be changed")
}

func TestPasswordChangeGate_AllowlistedPathsPass(t *testing.T) {
	token := signedToken(t, true)
	for _, path := range []string{"/api/change-password", "/api/logout", "/api/validate"} {
		w := gateRequest(t, path, token)
		assert.Equal(t, http.StatusOK, w.Code, "allowlisted path %s must not be gated", path)
	}
}

func TestPasswordChangeGate_ClearTokenPassesEverything(t *testing.T) {
	token := signedToken(t, false)
	for _, path := range []string{"/", "/api/stores", "/api/reports/sales"} {
		w := gateRequest(t, path, token)
		assert.Equal(t, http.StatusOK, w.Code, "a rotated session must not be gated on %s", path)
	}
}
