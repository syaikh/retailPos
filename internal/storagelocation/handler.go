package storagelocation

import (
	"fmt"
	"net/http"
	"strconv"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/permissions"
	"retail-pos-system/internal/shared"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc      *Service
	auditSvc audit.AuditCreator
}

func NewHandler(svc *Service, auditSvc audit.AuditCreator) *Handler {
	return &Handler{svc: svc, auditSvc: auditSvc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(permissions.Code) gin.HandlerFunc) {
	sl := r.Group("/storage-locations")
	sl.GET("", auth, perm(permissions.StorageLocationView), h.List)
	sl.GET("/:id", auth, perm(permissions.StorageLocationView), h.GetByID)
	sl.POST("", auth, perm(permissions.StorageLocationCreate), h.Create)
	sl.PUT("/:id", auth, perm(permissions.StorageLocationUpdate), h.Update)
	sl.DELETE("/:id", auth, perm(permissions.StorageLocationDelete), h.Delete)
	sl.PUT("/bulk", auth, perm(permissions.StorageLocationUpdate), h.BulkUpdate)
	sl.DELETE("/bulk", auth, perm(permissions.StorageLocationDelete), h.BulkDelete)
}

// List godoc
// @Summary      List storage locations
// @Description  Get a paginated list of storage locations
// @Tags         Storage Locations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        limit     query   int     false  "Page size"  default(20)
// @Param        offset    query   int     false  "Offset"     default(0)
// @Param        search    query   string  false  "Search name or code"
// @Param        is_active query   bool    false  "Filter by active status"
// @Success      200  {object}  map[string]interface{}
// @Router       /storage-locations [get]
func (h *Handler) List(c *gin.Context) {
	limit, offset := shared.ParsePaginationParams(c.Query("limit"), c.Query("offset"))
	search := c.Query("search")

	var isActive *bool
	if v := c.Query("is_active"); v != "" {
		b := v == "true" || v == "1"
		isActive = &b
	}

	locations, total, err := h.svc.GetAll(c.Request.Context(), limit, offset, search, isActive)
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	shared.JSONPaginated(c, locations, total, limit, offset)
}

// GetByID godoc
// @Summary      Get storage location by ID
// @Description  Get a single storage location by its ID
// @Tags         Storage Locations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Storage Location ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /storage-locations/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	location, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "storage location not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": location})
}

// Create godoc
// @Summary      Create a storage location
// @Description  Create a new storage location
// @Tags         Storage Locations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      StorageLocationCreateRequest  true  "Storage Location data"
// @Success      201   {object}  map[string]interface{}
// @Router       /storage-locations [post]
func (h *Handler) Create(c *gin.Context) {
	var req StorageLocationCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	location, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.auditSvc != nil {
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      middleware.UserIDFromContext(c.Request.Context()),
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "create",
			EntityType:  "storage_location",
			EntityID:    &location.ID,
			NewValues:   shared.ToJSONMap(location),
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Created storage location %s", location.Name),
		})
	}

	c.JSON(http.StatusCreated, gin.H{"data": location})
}

// Update godoc
// @Summary      Update a storage location
// @Description  Update an existing storage location
// @Tags         Storage Locations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int                          true  "Storage Location ID"
// @Param        body  body      StorageLocationUpdateRequest  true  "Update data"
// @Success      200   {object}  map[string]interface{}
// @Router       /storage-locations/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req StorageLocationUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	location, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.auditSvc != nil {
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      middleware.UserIDFromContext(c.Request.Context()),
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "update",
			EntityType:  "storage_location",
			EntityID:    &location.ID,
			NewValues:   shared.ToJSONMap(location),
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Updated storage location %s", location.Name),
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": location})
}

// Delete godoc
// @Summary      Delete a storage location
// @Description  Delete a storage location by ID
// @Tags         Storage Locations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Storage Location ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /storage-locations/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.auditSvc != nil {
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      middleware.UserIDFromContext(c.Request.Context()),
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "delete",
			EntityType:  "storage_location",
			EntityID:    &id,
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Deleted storage location #%d", id),
		})
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// BulkUpdate godoc
// @Summary      Bulk update storage locations
// @Description  Activate or deactivate multiple storage locations
// @Tags         Storage Locations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      object  true  "Bulk update payload"
// @Success      200   {object}  map[string]interface{}
// @Router       /storage-locations/bulk [put]
func (h *Handler) BulkUpdate(c *gin.Context) {
	var req struct {
		IDs      []int `json:"ids" binding:"required"`
		IsActive bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updated, err := h.svc.BulkUpdate(c.Request.Context(), req.IDs, req.IsActive)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.auditSvc != nil {
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      middleware.UserIDFromContext(c.Request.Context()),
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "bulk_update",
			EntityType:  "storage_location",
			NewValues:   map[string]interface{}{"ids": req.IDs, "is_active": req.IsActive},
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Bulk updated %d storage locations", updated),
		})
	}

	c.JSON(http.StatusOK, gin.H{"updated": updated})
}

// BulkDelete godoc
// @Summary      Bulk delete storage locations
// @Description  Delete multiple storage locations
// @Tags         Storage Locations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      object  true  "Bulk delete payload"
// @Success      200   {object}  map[string]interface{}
// @Router       /storage-locations/bulk [delete]
func (h *Handler) BulkDelete(c *gin.Context) {
	var req struct {
		IDs []int `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deleted, err := h.svc.BulkDelete(c.Request.Context(), req.IDs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.auditSvc != nil {
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      middleware.UserIDFromContext(c.Request.Context()),
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "bulk_delete",
			EntityType:  "storage_location",
			NewValues:   map[string]interface{}{"ids": req.IDs},
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Bulk deleted %d storage locations", deleted),
		})
	}

	c.JSON(http.StatusOK, gin.H{"deleted": deleted})
}
