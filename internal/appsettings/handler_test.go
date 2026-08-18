package appsettings

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────

func futureTime() time.Time { return time.Now().Add(1 * time.Hour) }

func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	logoDir := t.TempDir()
	return &Handler{cacheTTL: 60 * time.Second, logoDirAbs: logoDir}
}

// ──────────────────────────────────────────────────────────────────────
// Public branding — cache path (no DB)
// ──────────────────────────────────────────────────────────────────────

func TestHandler_GetPublicBranding_CacheHit(t *testing.T) {
	h := newTestHandler(t)
	h.cache = &cacheEntry{
		data:      BrandingSettings{StoreName: "Cached Store", StoreJargon: "Jargon", LogoPath: "logo.png"},
		expiresAt: futureTime(),
	}

	r := gin.New()
	r.GET("/api/settings/public", h.GetPublicBranding)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/settings/public", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	var resp BrandingSettings
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Cached Store", resp.StoreName)
	assert.Equal(t, "Jargon", resp.StoreJargon)
	assert.Equal(t, "logo.png", resp.LogoPath)
}

// ──────────────────────────────────────────────────────────────────────
// ServeLogo tests
// ──────────────────────────────────────────────────────────────────────

func TestHandler_ServeLogo_FromCache(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "logo.png"), []byte("PNG-DATA"), 0644))

	h := &Handler{logoDirAbs: tmpDir}
	h.cache = &cacheEntry{
		data:      BrandingSettings{LogoPath: "logo.png"},
		expiresAt: futureTime(),
	}

	r := gin.New()
	r.GET("/api/settings/logo", h.ServeLogo)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/settings/logo", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "PNG-DATA", w.Body.String())
	assert.Equal(t, "image/png", w.Header().Get("Content-Type"))
}

func TestHandler_ServeLogo_FromCache_JPEG(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "photo.jpg"), []byte("JPEG-DATA"), 0644))

	h := &Handler{logoDirAbs: tmpDir}
	h.cache = &cacheEntry{
		data:      BrandingSettings{LogoPath: "photo.jpg"},
		expiresAt: futureTime(),
	}

	r := gin.New()
	r.GET("/api/settings/logo", h.ServeLogo)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/settings/logo", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "JPEG-DATA", w.Body.String())
	assert.Equal(t, "image/jpeg", w.Header().Get("Content-Type"))
}

func TestHandler_ServeLogo_PathTraversal(t *testing.T) {
	h := &Handler{logoDirAbs: t.TempDir()}
	h.cache = &cacheEntry{
		data:      BrandingSettings{LogoPath: "../../etc/passwd"},
		expiresAt: futureTime(),
	}

	r := gin.New()
	r.GET("/api/settings/logo", h.ServeLogo)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/settings/logo", nil))

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_ServeLogo_FileNotFound(t *testing.T) {
	h := &Handler{logoDirAbs: t.TempDir()}
	h.cache = &cacheEntry{
		data:      BrandingSettings{LogoPath: "nonexistent.png"},
		expiresAt: futureTime(),
	}

	r := gin.New()
	r.GET("/api/settings/logo", h.ServeLogo)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/settings/logo", nil))

	// File doesn't exist — gin serves 404.
	assert.True(t, w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError)
}

// ──────────────────────────────────────────────────────────────────────
// UpdateAll — request parsing contract (no DB)
// ──────────────────────────────────────────────────────────────────────

func TestHandler_UpdateAll_InvalidJSON(t *testing.T) {
	h := newTestHandler(t)

	r := gin.New()
	r.PUT("/api/settings", h.UpdateAll)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/settings", strings.NewReader("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateAll_MissingSettingsKey(t *testing.T) {
	h := newTestHandler(t)

	r := gin.New()
	r.PUT("/api/settings", h.UpdateAll)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/settings", strings.NewReader(`{"other":"value"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ──────────────────────────────────────────────────────────────────────
// UploadLogo — request parsing contract (no DB)
// ──────────────────────────────────────────────────────────────────────

func TestHandler_UploadLogo_InvalidExtension(t *testing.T) {
	h := newTestHandler(t)

	r := gin.New()
	r.POST("/api/settings/logo", h.UploadLogo)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "malware.exe")
	_, _ = part.Write([]byte("MZ"))
	writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/settings/logo", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "unsupported image type")
}

func TestHandler_UploadLogo_MissingFileField(t *testing.T) {
	h := newTestHandler(t)

	r := gin.New()
	r.POST("/api/settings/logo", h.UploadLogo)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("not_file", "value")
	writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/settings/logo", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "file is required or too large")
}

func TestHandler_UploadLogo_InvalidMIME(t *testing.T) {
	h := newTestHandler(t)

	r := gin.New()
	r.POST("/api/settings/logo", h.UploadLogo)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "fake.png")
	_, _ = part.Write([]byte("not-a-real-png-image"))
	writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/settings/logo", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ──────────────────────────────────────────────────────────────────────
// Cache behavior
// ──────────────────────────────────────────────────────────────────────

func TestHandler_InvalidateCache(t *testing.T) {
	h := newTestHandler(t)
	h.cache = &cacheEntry{data: BrandingSettings{StoreName: "test"}, expiresAt: futureTime()}

	h.invalidateCache()

	h.mu.RLock()
	defer h.mu.RUnlock()
	assert.Nil(t, h.cache)
}

func TestHandler_CacheConcurrency(t *testing.T) {
	h := newTestHandler(t)
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			h.mu.Lock()
			h.cache = &cacheEntry{data: BrandingSettings{StoreName: "c"}, expiresAt: futureTime()}
			h.mu.Unlock()
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}
