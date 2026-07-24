package shared

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetUserID_Present(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("userID", 42)
	if got := GetUserID(c); got != 42 {
		t.Errorf("expected 42, got %d", got)
	}
}

func TestGetUserID_Missing(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	if got := GetUserID(c); got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

func TestGetUserID_WrongType(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("userID", "not-int")
	if got := GetUserID(c); got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

func TestGetUsername_Present(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("username", "admin")
	if got := GetUsername(c); got != "admin" {
		t.Errorf("expected admin, got %s", got)
	}
}

func TestGetUsername_Missing(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	if got := GetUsername(c); got != "" {
		t.Errorf("expected empty, got %s", got)
	}
}

func TestGetUsername_WrongType(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("username", 12345)
	if got := GetUsername(c); got != "" {
		t.Errorf("expected empty for wrong type, got %s", got)
	}
}

func TestGetRole_Present(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("role", "superadmin")
	if got := GetRole(c); got != "superadmin" {
		t.Errorf("expected superadmin, got %s", got)
	}
}

func TestGetRole_Missing(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	if got := GetRole(c); got != "" {
		t.Errorf("expected empty, got %s", got)
	}
}

func TestGetRole_WrongType(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("role", 42)
	if got := GetRole(c); got != "" {
		t.Errorf("expected empty for wrong type, got %s", got)
	}
}

func TestGetStoreID_Present(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	sid := 7
	c.Set("storeID", &sid)
	got := GetStoreID(c)
	if got == nil || *got != 7 {
		t.Errorf("expected 7, got %v", got)
	}
}

func TestGetStoreID_Missing(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	if got := GetStoreID(c); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestGetStoreID_WrongType(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("storeID", 123)
	if got := GetStoreID(c); got != nil {
		t.Errorf("expected nil for wrong type, got %v", got)
	}
}

func TestGetIPAddress_FromRemoteAddr(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.RemoteAddr = "192.168.1.1:12345"

	got := GetIPAddress(c)
	if got != "192.168.1.1" {
		t.Errorf("expected 192.168.1.1, got %s", got)
	}
}

func TestGetIPAddress_NoPort(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.RemoteAddr = "10.0.0.1"

	got := GetIPAddress(c)
	if got != "10.0.0.1" {
		t.Errorf("expected 10.0.0.1, got %s", got)
	}
}

func TestGetUserAgent(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set("User-Agent", "TestBot/1.0")

	if got := GetUserAgent(c); got != "TestBot/1.0" {
		t.Errorf("expected TestBot/1.0, got %s", got)
	}
}

func TestGetUserAgent_Missing(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	if got := GetUserAgent(c); got != "" {
		t.Errorf("expected empty, got %s", got)
	}
}

func TestGetIPAddress_IPv6(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.RemoteAddr = "[::1]:8080"

	got := GetIPAddress(c)
	if got != "::1" {
		t.Errorf("expected ::1, got %s", got)
	}
}

func TestGetIPAddress_FromRealIP(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request = req
	addr := &net.TCPAddr{IP: net.ParseIP("172.16.0.1"), Port: 54321}
	c.Request.RemoteAddr = addr.String()
	got := GetIPAddress(c)
	if got != "172.16.0.1" {
		t.Errorf("expected 172.16.0.1, got %s", got)
	}
}

func TestGetIPAddress_XForwardedFor(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.RemoteAddr = "192.168.1.1:12345"
	c.Request.Header.Set("X-Forwarded-For", "10.0.0.1")

	got := GetIPAddress(c)
	if got != "192.168.1.1" {
		t.Errorf("expected 192.168.1.1 (RemoteAddr used despite XFF), got %s", got)
	}
}

func TestAbortUnauthorized(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	AbortUnauthorized(c, ErrUnauthorized, "not logged in")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAbortForbidden(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	AbortForbidden(c, ErrForbidden, "no permission")

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestAbortInternalError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	AbortInternalError(c, ErrInternal, "db down")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
