package store

import (
	"fmt"
	"net/http"
	"strconv"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/middleware"
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

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(string) gin.HandlerFunc) {
	sg := r.Group("/stores")
	sg.GET("", auth, perm("store:read"), h.List)
	sg.GET("/active", auth, perm("store:read"), h.ListActive)
	sg.GET("/:id", auth, perm("store:read"), h.GetByID)
	sg.POST("", auth, perm("store:create"), h.Create)
	sg.PUT("/:id", auth, perm("store:update"), h.Update)
	sg.DELETE("/:id", auth, perm("store:delete"), h.Delete)
}

// List godoc
// @Summary      List stores
// @Description  Get a paginated list of stores
// @Tags         Stores
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        limit     query   int     false  "Page size"  default(20)
// @Param        offset    query   int     false  "Offset"     default(0)
// @Param        search    query   string  false  "Search name"
// @Param        is_active query   bool    false  "Filter by active status"
// @Success      200  {object}  map[string]interface{}
// @Router       /stores [get]
func (h *Handler) List(c *gin.Context) {
	limit, offset := shared.ParsePaginationParams(c.Query("limit"), c.Query("offset"))
	search := c.Query("search")

	var isActive *bool
	if v := c.Query("is_active"); v != "" {
		b := v == "true" || v == "1"
		isActive = &b
	}

	stores, total, err := h.svc.GetAll(c.Request.Context(), limit, offset, search, isActive)
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	shared.JSONPaginated(c, stores, total, limit, offset)
}

// ListActive godoc
// @Summary      List active stores
// @Description  Get all active stores (no pagination)
// @Tags         Stores
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Router       /stores/active [get]
func (h *Handler) ListActive(c *gin.Context) {
	stores, err := h.svc.GetAllActive(c.Request.Context())
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": stores})
}

// GetByID godoc
// @Summary      Get store by ID
// @Description  Get a single store by its ID
// @Tags         Stores
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Store ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /stores/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	store, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "store not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": store})
}

// Create godoc
// @Summary      Create a store
// @Description  Create a new store
// @Tags         Stores
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      StoreCreateRequest  true  "Store data"
// @Success      201   {object}  map[string]interface{}
// @Router       /stores [post]
func (h *Handler) Create(c *gin.Context) {
	var req StoreCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	st, err := h.svc.Create(c.Request.Context(), req)
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
			EntityType:  "store",
			EntityID:    &st.ID,
			NewValues:   shared.ToJSONMap(st),
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Created store %s", st.Name),
		})
	}

	c.JSON(http.StatusCreated, gin.H{"data": st})
}

// Update godoc
// @Summary      Update a store
// @Description  Update an existing store
// @Tags         Stores
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int                   true  "Store ID"
// @Param        body  body      StoreUpdateRequest    true  "Update data"
// @Success      200   {object}  map[string]interface{}
// @Router       /stores/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req StoreUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	st, err := h.svc.Update(c.Request.Context(), id, req)
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
			EntityType:  "store",
			EntityID:    &st.ID,
			NewValues:   shared.ToJSONMap(st),
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Updated store %s", st.Name),
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": st})
}

// Delete godoc
// @Summary      Delete a store
// @Description  Delete a store by ID
// @Tags         Stores
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Store ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /stores/{id} [delete]
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
			EntityType:  "store",
			EntityID:    &id,
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Deleted store #%d", id),
		})
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
