package appsettings

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/permissions"
	"retail-pos-system/internal/shared"

	"github.com/gin-gonic/gin"
)

// cacheEntry holds an in-memory snapshot of the public branding settings.
type cacheEntry struct {
	data      BrandingSettings
	expiresAt time.Time
}

// StoreProvider abstracts fetching and updating branch address/phone from the stores module.
type StoreProvider interface {
	GetStoreAddress(ctx context.Context, storeID int) (address, phone string, err error)
	UpdateStoreAddress(ctx context.Context, storeID int, address, phone string) error
}

// Handler handles HTTP requests for application settings.
type Handler struct {
	svc      ServiceIface
	auditSvc audit.Creator
	store    StoreProvider

	mu         sync.RWMutex
	cache      *cacheEntry
	cacheTTL   time.Duration
	logoDirAbs string
}

// ServiceIface is the subset of Service methods used by Handler.
type ServiceIface interface {
	GetAll(ctx context.Context) (map[string]string, error)
	GetMultiple(ctx context.Context, keys []string) (map[string]string, error)
	Upsert(ctx context.Context, settings map[string]string) error
	SaveLogoPath(ctx context.Context, logoPath string) error
}

// NewHandler returns a new Handler. It creates the logo upload directory if it
// does not already exist.
func NewHandler(svc ServiceIface, auditSvc audit.Creator, store StoreProvider) *Handler {
	logoDir := filepath.Join("uploads", "logos")
	if err := os.MkdirAll(logoDir, 0755); err != nil {
		slog.Warn("appsettings: could not create logo directory", "path", logoDir, "error", err)
	}

	absLogoDir, err := filepath.Abs(logoDir)
	if err != nil {
		absLogoDir = logoDir
	}

	return &Handler{
		svc:        svc,
		auditSvc:   auditSvc,
		store:      store,
		cacheTTL:   60 * time.Second,
		logoDirAbs: absLogoDir,
	}
}

// RegisterPublicRoutes registers routes that require no authentication.
func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.GET("/settings/public", h.GetPublicBranding)
	rg.GET("/settings/logo", h.ServeLogo)
}

// RegisterRoutes registers the authenticated settings routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(permissions.Code) gin.HandlerFunc) {
	sg := r.Group("/settings")
	// GET /api/settings is readable by any authenticated user (not gated by
	// app_settings.view). The payload is non-sensitive global config (store
	// branding, receipt text, per-branch address/phone) that is required for
	// receipt rendering across all roles, including cashiers. The Settings
	// *management UI* remains gated by app_settings.view via the frontend
	// routePermissions map, so this only relaxes reading the data, not editing it.
	sg.GET("", auth, h.GetAll)
	sg.PUT("", auth, perm(permissions.AppSettingsUpdate), h.UpdateAll)
	sg.POST("/logo", auth, perm(permissions.AppSettingsUpdate), h.UploadLogo)
	sg.DELETE("/logo", auth, perm(permissions.AppSettingsUpdate), h.RemoveLogo)
}

// ──────────────────────────────────────────────────────────────────────
// Public branding (no auth)
// ──────────────────────────────────────────────────────────────────────

// GetPublicBranding returns store_name, store_jargon, and logo_path with
// a 60-second in-memory cache.
func (h *Handler) GetPublicBranding(c *gin.Context) {
	h.mu.RLock()
	if h.cache != nil && time.Now().Before(h.cache.expiresAt) {
		data := h.cache.data
		h.mu.RUnlock()
		c.JSON(http.StatusOK, data)
		return
	}
	h.mu.RUnlock()

	settings, err := h.svc.GetMultiple(c.Request.Context(), []string{"store_name", "store_jargon", "logo_path"})
	if err != nil {
		shared.InternalError(c, err)
		return
	}

	branding := BrandingSettings{
		StoreName:   settings["store_name"],
		StoreJargon: settings["store_jargon"],
		LogoPath:    settings["logo_path"],
	}

	h.mu.Lock()
	h.cache = &cacheEntry{data: branding, expiresAt: time.Now().Add(h.cacheTTL)}
	h.mu.Unlock()

	c.JSON(http.StatusOK, branding)
}

// ServeLogo serves the logo file from uploads/logos/.
func (h *Handler) ServeLogo(c *gin.Context) {
	// Read the logo_path from settings (or cache) to know which file to serve.
	logoPath := ""
	h.mu.RLock()
	if h.cache != nil && time.Now().Before(h.cache.expiresAt) {
		logoPath = h.cache.data.LogoPath
	}
	h.mu.RUnlock()

	if logoPath == "" {
		// Cache miss or empty; fetch from DB.
		settings, err := h.svc.GetMultiple(c.Request.Context(), []string{"logo_path"})
		if err != nil {
			shared.InternalError(c, err)
			return
		}
		logoPath = settings["logo_path"]
	}

	if logoPath == "" {
		c.Status(http.StatusNoContent)
		return
	}

	// Resolve absolute path and ensure it stays within the logo directory.
	absPath := filepath.Join(h.logoDirAbs, filepath.Base(logoPath))
	cleanPath := filepath.Clean(absPath)
	if !strings.HasPrefix(cleanPath, h.logoDirAbs) {
		c.Status(http.StatusNotFound)
		return
	}

	// Detect content type from extension.
	ext := strings.ToLower(filepath.Ext(cleanPath))
	contentType := "application/octet-stream"
	switch ext {
	case ".png":
		contentType = "image/png"
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	}
	c.Header("Content-Type", contentType)
	c.File(cleanPath)
}

// ──────────────────────────────────────────────────────────────────────
// Protected settings (auth + permission)
// ──────────────────────────────────────────────────────────────────────

// GetAll returns all application settings merged with the caller's branch data.
func (h *Handler) GetAll(c *gin.Context) {
	rawSettings, err := h.svc.GetAll(c.Request.Context())
	if err != nil {
		shared.InternalError(c, err)
		return
	}

	settings := AllSettings{
		BrandingSettings: BrandingSettings{
			StoreName:   rawSettings["store_name"],
			StoreJargon: rawSettings["store_jargon"],
			LogoPath:    rawSettings["logo_path"],
		},
		ReceiptHeader: rawSettings["receipt_header"],
		ReceiptFooter: rawSettings["receipt_footer"],
	}

	// Merge per-branch address/phone from the stores table using the caller's store_id.
	storeID := middleware.StoreIDFromContext(c.Request.Context())
	if storeID != nil && h.store != nil {
		if addr, phone, err := h.store.GetStoreAddress(c.Request.Context(), *storeID); err == nil {
			settings.StoreAddress = addr
			settings.StorePhone = phone
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"settings": settings,
	})
}

// UpdateAll bulk-updates settings from a JSON body.
func (h *Handler) UpdateAll(c *gin.Context) {
	var req struct {
		Settings map[string]string `json:"settings" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, shared.NewError(shared.ErrBadRequest, "invalid request body"))
		return
	}

	// Extract per-branch fields that live in the stores table, not app_settings.
	storeAddr := req.Settings["store_address"]
	storePhone := req.Settings["store_phone"]
	_, hasStoreAddr := req.Settings["store_address"]
	_, hasStorePhone := req.Settings["store_phone"]
	delete(req.Settings, "store_address")
	delete(req.Settings, "store_phone")

	if len(req.Settings) > 0 {
		if err := h.svc.Upsert(c.Request.Context(), req.Settings); err != nil {
			c.JSON(http.StatusBadRequest, shared.NewError(shared.ErrValidation, err.Error()))
			return
		}
	}

	// Persist branch address/phone if the caller has a store assignment.
	if (hasStoreAddr || hasStorePhone) && h.store != nil {
		storeID := middleware.StoreIDFromContext(c.Request.Context())
		if storeID != nil {
			if err := h.store.UpdateStoreAddress(c.Request.Context(), *storeID, storeAddr, storePhone); err != nil {
				shared.InternalError(c, err)
				return
			}
		}
	}

	h.invalidateCache()

	if h.auditSvc != nil {
		auditValues := make(map[string]string, len(req.Settings)+2)
		for k, v := range req.Settings {
			auditValues[k] = v
		}
		if hasStoreAddr {
			auditValues["store_address"] = storeAddr
		}
		if hasStorePhone {
			auditValues["store_phone"] = storePhone
		}
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.Log{
			UserID:      middleware.UserIDFromContext(c.Request.Context()),
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "update",
			EntityType:  "app_settings",
			NewValues:   auditValues,
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: "Updated application settings",
			StoreID:     middleware.StoreIDFromContext(c.Request.Context()),
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": "settings updated"})
}

// UploadLogo handles multipart logo image uploads.
func (h *Handler) UploadLogo(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2<<20)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, shared.NewError(shared.ErrBadRequest, "file is required or too large (max 2MB)"))
		return
	}
	defer func() { _ = file.Close() }()

	// Validate extension.
	ext := filepath.Ext(header.Filename)
	cleanExt, err := ValidateLogoExtension(ext)
	if err != nil {
		c.JSON(http.StatusBadRequest, shared.NewError(shared.ErrValidation, err.Error()))
		return
	}

	// Read a small buffer to sniff MIME type.
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	if n > 0 {
		mimeType := http.DetectContentType(buf[:n])
		validMime := map[string]bool{
			"image/png": true, "image/jpeg": true,
		}
		if !validMime[mimeType] {
			c.JSON(http.StatusBadRequest, shared.NewError(shared.ErrValidation, "file content does not match image type"))
			return
		}
	}
	if seeker, ok := file.(io.Seeker); ok {
		_, _ = seeker.Seek(0, io.SeekStart)
	}

	// Save file.
	destPath := LogoPathForFile(cleanExt)
	absDest := filepath.Join(h.logoDirAbs, filepath.Base(destPath))

	// Remove old logo file if it exists and differs from the new one.
	oldSettings, _ := h.svc.GetMultiple(c.Request.Context(), []string{"logo_path"})
	if oldPath := oldSettings["logo_path"]; oldPath != "" && oldPath != filepath.Base(destPath) {
		oldAbs := filepath.Join(h.logoDirAbs, filepath.Base(oldPath))
		if err := os.Remove(oldAbs); err != nil && !os.IsNotExist(err) {
			slog.Warn("appsettings: could not delete old logo on replacement", "path", oldAbs, "error", err)
		}
	}

	out, err := os.Create(absDest)
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, file); err != nil {
		shared.InternalError(c, err)
		return
	}

	// Persist the path in settings (relative path for portability).
	if err := h.svc.SaveLogoPath(c.Request.Context(), filepath.Base(destPath)); err != nil {
		shared.InternalError(c, err)
		return
	}

	h.invalidateCache()

	if h.auditSvc != nil {
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.Log{
			UserID:      middleware.UserIDFromContext(c.Request.Context()),
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "upload",
			EntityType:  "app_settings",
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: "Uploaded application logo",
			StoreID:     middleware.StoreIDFromContext(c.Request.Context()),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"logo_path": filepath.Base(destPath),
		"message":   "logo uploaded successfully",
	})
}

// RemoveLogo deletes the current logo file and clears the setting.
func (h *Handler) RemoveLogo(c *gin.Context) {
	settings, err := h.svc.GetMultiple(c.Request.Context(), []string{"logo_path"})
	if err != nil {
		shared.InternalError(c, err)
		return
	}

	currentPath := settings["logo_path"]
	if currentPath != "" {
		absPath := filepath.Join(h.logoDirAbs, filepath.Base(currentPath))
		if err := os.Remove(absPath); err != nil && !os.IsNotExist(err) {
			slog.Warn("appsettings: could not delete old logo", "path", absPath, "error", err)
		}
	}

	if err := h.svc.SaveLogoPath(c.Request.Context(), ""); err != nil {
		shared.InternalError(c, err)
		return
	}

	h.invalidateCache()

	if h.auditSvc != nil {
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.Log{
			UserID:      middleware.UserIDFromContext(c.Request.Context()),
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "remove",
			EntityType:  "app_settings",
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: "Removed application logo",
			StoreID:     middleware.StoreIDFromContext(c.Request.Context()),
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": "logo removed"})
}

// ──────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────

func (h *Handler) invalidateCache() {
	h.mu.Lock()
	h.cache = nil
	h.mu.Unlock()
}
