package store

import (
	"errors"
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
	auditSvc audit.Creator
}

func NewHandler(svc *Service, auditSvc audit.Creator) *Handler {
	return &Handler{svc: svc, auditSvc: auditSvc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(permissions.Code) gin.HandlerFunc) {
	// Warehouses were previously public; scoping requires the caller's
	// identity, so the route now lives behind auth (store-boundary filter:
	// store-scoped callers only see their own store's warehouses).
	r.GET("/warehouses", auth, h.ListWarehouses)
	sg := r.Group("/stores")
	sg.GET("", auth, perm(permissions.StoreView), h.List)
	sg.GET("/active", auth, perm(permissions.StoreView), h.ListActive)
	sg.GET("/:id", auth, perm(permissions.StoreView), h.GetByID)
	sg.POST("", auth, perm(permissions.StoreCreate), h.Create)
	sg.PUT("/:id", auth, perm(permissions.StoreUpdate), h.Update)
	sg.DELETE("/:id", auth, perm(permissions.StoreDelete), h.Delete)
}

// requireOwnStore blocks a store-scoped caller from reading or mutating a
// different store, mirroring the list scoping (superadmin's nil store bypasses).
// It writes the 403 response itself and returns false when the request must stop.
func requireOwnStore(c *gin.Context, id int) bool {
	if sid := shared.GetStoreID(c); sid != nil && *sid != id {
		c.JSON(http.StatusForbidden, gin.H{"error": "store is outside your scope"})
		return false
	}
	return true
}

// writeError maps store service errors to HTTP responses: 404 for
// ErrNotFound, 500 for repository failures (ErrInternal), 400 for
// validation errors.
func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "store not found"})
	case errors.Is(err, ErrInternal):
		shared.InternalError(c, err)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

// ListWarehouses godoc
// @Summary List warehouses
// @Description Get active warehouses, scoped to the caller's store (superadmin sees all, including central warehouses)
// @Tags Stores
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /warehouses [get]
func (h *Handler) ListWarehouses(c *gin.Context) {
	warehouses, err := h.svc.GetAllWarehouses(c.Request.Context(), shared.GetStoreID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch warehouses"})
		return
	}
	if warehouses == nil {
		warehouses = []Warehouse{}
	}
	c.JSON(http.StatusOK, gin.H{"data": warehouses})
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

	stores, total, err := h.svc.GetAll(c.Request.Context(), limit, offset, search, isActive, shared.GetStoreID(c))
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
	stores, err := h.svc.GetAllActive(c.Request.Context(), shared.GetStoreID(c))
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
	if !requireOwnStore(c, id) {
		return
	}

	store, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
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
// @Param        body  body      CreateRequest  true  "Store data"
// @Success      201   {object}  map[string]interface{}
// @Router       /stores [post]
func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	st, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		writeError(c, err)
		return
	}

	if h.auditSvc != nil {
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.Log{
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
			StoreID:     middleware.StoreIDFromContext(c.Request.Context()),
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
// @Param        body  body      UpdateRequest    true  "Update data"
// @Success      200   {object}  map[string]interface{}
// @Router       /stores/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if !requireOwnStore(c, id) {
		return
	}

	var oldStore *Store
	if h.auditSvc != nil {
		oldStore, _ = h.svc.GetByID(c.Request.Context(), id)
	}

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	st, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		writeError(c, err)
		return
	}

	if h.auditSvc != nil {
		userID := middleware.UserIDFromContext(c.Request.Context())
		focusedOld, focusedNew := shared.DiffChanges(shared.ToJSONMap(oldStore), shared.ToJSONMap(st))
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.Log{
			UserID:      userID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "update",
			EntityType:  "store",
			EntityID:    &st.ID,
			OldValues:   focusedOld,
			NewValues:   focusedNew,
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Updated store %s", st.Name),
			StoreID:     middleware.StoreIDFromContext(c.Request.Context()),
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
	if !requireOwnStore(c, id) {
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		writeError(c, err)
		return
	}

	if h.auditSvc != nil {
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.Log{
			UserID:      middleware.UserIDFromContext(c.Request.Context()),
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "delete",
			EntityType:  "store",
			EntityID:    &id,
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Deleted store #%d", id),
			StoreID:     middleware.StoreIDFromContext(c.Request.Context()),
		})
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
