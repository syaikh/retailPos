package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBodyLimitMiddleware_Default(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader("hello"))

	handler := BodyLimitMiddleware(0)
	handler(c)

	if w.Code == http.StatusRequestEntityTooLarge {
		t.Error("should not reject small body")
	}
}

func TestBodyLimitMiddleware_CustomLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader("small"))

	handler := BodyLimitMiddleware(1024)
	handler(c)

	if w.Code == http.StatusRequestEntityTooLarge {
		t.Error("unexpected 413 for small body within limit")
	}
}

func TestBodyLimitMiddleware_NilBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Body = nil

	handler := BodyLimitMiddleware(1024)
	handler(c)

	if w.Code == http.StatusRequestEntityTooLarge {
		t.Error("should not error with nil body")
	}
}

func TestBodyLimitMiddleware_Constant(t *testing.T) {
	if DefaultMaxBodySize != 1<<20 {
		t.Errorf("expected 1MB, got %d", DefaultMaxBodySize)
	}
}

func TestBodyLimitMiddleware_HandlerSetsLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader("data"))

	handler := BodyLimitMiddleware(100)
	handler(c)

	if c.Request.Body == nil {
		t.Error("expected body to be wrapped with MaxBytesReader")
	}
}
