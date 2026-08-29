package appsettings

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"retail-pos-system/internal/middleware"

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
	_ = writer.Close()

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
	_ = writer.Close()

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
	_ = writer.Close()

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

// ──────────────────────────────────────────────────────────────────────
// Mocks
// ──────────────────────────────────────────────────────────────────────

type mockStoreProvider struct {
	getAddrFn    func(ctx context.Context, storeID int) (string, string, error)
	updateAddrFn func(ctx context.Context, storeID int, address, phone string) error
}

func (m *mockStoreProvider) GetStoreAddress(ctx context.Context, storeID int) (string, string, error) {
	if m.getAddrFn != nil {
		return m.getAddrFn(ctx, storeID)
	}
	return "", "", nil
}

func (m *mockStoreProvider) UpdateStoreAddress(ctx context.Context, storeID int, address, phone string) error {
	if m.updateAddrFn != nil {
		return m.updateAddrFn(ctx, storeID, address, phone)
	}
	return nil
}

type mockService struct {
	getAllFn     func(ctx context.Context) (map[string]string, error)
	getMultipleFn func(ctx context.Context, keys []string) (map[string]string, error)
	upsertFn     func(ctx context.Context, settings map[string]string) error
	saveLogoFn   func(ctx context.Context, path string) error
}

func (m *mockService) GetAll(ctx context.Context) (map[string]string, error) {
	if m.getAllFn != nil {
		return m.getAllFn(ctx)
	}
	return map[string]string{
		"store_name":    "Test Store",
		"store_jargon":  "Jargon",
		"logo_path":     "logo.png",
		"receipt_header": "Header",
		"receipt_footer": "Footer",
	}, nil
}

func (m *mockService) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	if m.getMultipleFn != nil {
		return m.getMultipleFn(ctx, keys)
	}
	return map[string]string{}, nil
}

func (m *mockService) Upsert(ctx context.Context, settings map[string]string) error {
	if m.upsertFn != nil {
		return m.upsertFn(ctx, settings)
	}
	return nil
}

func (m *mockService) SaveLogoPath(ctx context.Context, path string) error {
	if m.saveLogoFn != nil {
		return m.saveLogoFn(ctx, path)
	}
	return nil
}

// ──────────────────────────────────────────────────────────────────────
// GetAll — branch data merge
// ──────────────────────────────────────────────────────────────────────

func TestHandler_GetAll_IncludesBranchData(t *testing.T) {
	storeID := 42
	svc := &mockService{}
	provider := &mockStoreProvider{
		getAddrFn: func(_ context.Context, id int) (string, string, error) {
			assert.Equal(t, storeID, id)
			return "Jl. Sudirman 123", "021-1234567", nil
		},
	}
	h := NewHandler(svc, nil, provider)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		ctx := middleware.ContextWithStoreID(c.Request.Context(), &storeID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	r.GET("/api/settings", h.GetAll)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/settings", nil))

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]AllSettings
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	settings := resp["settings"]
	assert.Equal(t, "Jl. Sudirman 123", settings.StoreAddress)
	assert.Equal(t, "021-1234567", settings.StorePhone)
	assert.Equal(t, "Test Store", settings.StoreName)
}

func TestHandler_GetAll_NoStoreID(t *testing.T) {
	svc := &mockService{}
	provider := &mockStoreProvider{}
	h := NewHandler(svc, nil, provider)

	r := gin.New()
	r.GET("/api/settings", h.GetAll)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/settings", nil))

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]AllSettings
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Empty(t, resp["settings"].StoreAddress)
	assert.Empty(t, resp["settings"].StorePhone)
}

// ──────────────────────────────────────────────────────────────────────
// UpdateAll — store_* routing
// ──────────────────────────────────────────────────────────────────────

func TestHandler_UpdateAll_RoutesStoreFields(t *testing.T) {
	storeID := 7
	var capturedAddr, capturedPhone string
	var capturedSettings map[string]string

	svc := &mockService{
		upsertFn: func(_ context.Context, settings map[string]string) error {
			capturedSettings = settings
			return nil
		},
	}
	provider := &mockStoreProvider{
		updateAddrFn: func(_ context.Context, id int, addr, phone string) error {
			assert.Equal(t, storeID, id)
			capturedAddr = addr
			capturedPhone = phone
			return nil
		},
	}
	h := NewHandler(svc, nil, provider)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		ctx := middleware.ContextWithStoreID(c.Request.Context(), &storeID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	r.PUT("/api/settings", h.UpdateAll)

	body := `{"settings":{"store_name":"New","store_address":"Jl. Thamrin","store_phone":"021-999"}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	// store_address/store_phone should NOT be in app_settings upsert
	assert.NotContains(t, capturedSettings, "store_address")
	assert.NotContains(t, capturedSettings, "store_phone")
	assert.Equal(t, "New", capturedSettings["store_name"])
	// store_address/store_phone should be routed to store provider
	assert.Equal(t, "Jl. Thamrin", capturedAddr)
	assert.Equal(t, "021-999", capturedPhone)
}

func TestHandler_UpdateAll_OnlyStoreFields_NoAppSettingsUpsert(t *testing.T) {
	storeID := 7
	var upsertCalled bool

	svc := &mockService{
		upsertFn: func(_ context.Context, _ map[string]string) error {
			upsertCalled = true
			return nil
		},
	}
	provider := &mockStoreProvider{
		updateAddrFn: func(_ context.Context, _ int, _, _ string) error { return nil },
	}
	h := NewHandler(svc, nil, provider)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		ctx := middleware.ContextWithStoreID(c.Request.Context(), &storeID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	r.PUT("/api/settings", h.UpdateAll)

	body := `{"settings":{"store_address":"Addr","store_phone":"Phone"}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.False(t, upsertCalled, "Upsert should not be called when only store fields are sent")
}

func TestHandler_UpdateAll_ClearsStoreFields(t *testing.T) {
	storeID := 7
	var capturedAddr, capturedPhone string
	var providerCalled bool

	svc := &mockService{}
	provider := &mockStoreProvider{
		updateAddrFn: func(_ context.Context, id int, addr, phone string) error {
			providerCalled = true
			capturedAddr = addr
			capturedPhone = phone
			return nil
		},
	}
	h := NewHandler(svc, nil, provider)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		ctx := middleware.ContextWithStoreID(c.Request.Context(), &storeID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	r.PUT("/api/settings", h.UpdateAll)

	body := `{"settings":{"store_address":"","store_phone":""}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, providerCalled, "UpdateStoreAddress should be called even with empty values")
	assert.Equal(t, "", capturedAddr)
	assert.Equal(t, "", capturedPhone)
}
