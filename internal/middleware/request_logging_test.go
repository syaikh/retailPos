package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"retail-pos-system/internal/shared"
)

func setupRequestLoggingRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestLoggingMiddleware())
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": "pong"})
	})
	r.GET("/boom", func(c *gin.Context) {
		c.String(http.StatusInternalServerError, "boom")
	})
	return r
}

func TestRequestLogging_GeneratesRequestID(t *testing.T) {
	r := setupRequestLoggingRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	reqID := w.Header().Get("X-Request-ID")
	assert.NotEmpty(t, reqID, "X-Request-ID header should be set")
}

func TestRequestLogging_HonorsIncomingRequestID(t *testing.T) {
	r := setupRequestLoggingRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("X-Request-ID", "incoming-id-123")

	r.ServeHTTP(w, req)

	assert.Equal(t, "incoming-id-123", w.Header().Get("X-Request-ID"))
}

func TestRequestLogging_SetsRequestIDInContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	var captured string
	r.Use(RequestLoggingMiddleware())
	r.GET("/check", func(c *gin.Context) {
		captured = shared.GetRequestID(c.Request.Context())
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/check", nil)
	r.ServeHTTP(w, req)

	assert.NotEmpty(t, captured, "request ID should be available on request context")
	assert.Equal(t, captured, w.Header().Get("X-Request-ID"))
}

func TestRequestLogging_DoesNotMaskErrorStatus(t *testing.T) {
	r := setupRequestLoggingRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/boom", nil)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
}

func TestRequestIDFromContext_Missing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	assert.Equal(t, "", RequestIDFromContext(c))
}

func TestRequestIDFromContext_Present(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set("requestID", "abc-123")

	assert.Equal(t, "abc-123", RequestIDFromContext(c))
}
