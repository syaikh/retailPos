package customergroup

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
	cg := r.Group("/customer-groups")
	cg.GET("", auth, perm("customer_group.view"), h.List)
	cg.GET("/:id", auth, perm("customer_group.view"), h.GetByID)
	cg.POST("", auth, perm("customer_group.create"), h.Create)
	cg.PUT("/:id", auth, perm("customer_group.update"), h.Update)
	cg.DELETE("/:id", auth, perm("customer_group.delete"), h.Delete)
	cg.PUT("/bulk", auth, perm("customer_group.update"), h.BulkUpdate)
	cg.DELETE("/bulk", auth, perm("customer_group.delete"), h.BulkDelete)
}

// List godoc
// @Summary      List customer groups
// @Description  Get a paginated list of customer groups
// @Tags         Customer Groups
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        limit    query   int     false  "Page size"  default(20)
// @Param        offset   query   int     false  "Offset"     default(0)
// @Param        search   query   string  false  "Search name or description"
// @Param        is_active query  bool    false  "Filter by active status"
// @Success      200  {object}  map[string]interface{}
// @Router       /customer-groups [get]
func (h *Handler) List(c *gin.Context) {
	limit, offset := shared.ParsePaginationParams(c.Query("limit"), c.Query("offset"))
	search := c.Query("search")

	var isActive *bool
	if v := c.Query("is_active"); v != "" {
		b := v == "true" || v == "1"
		isActive = &b
	}

	var hasCustomers *bool
	if v := c.Query("has_customers"); v != "" {
		b := v == "true" || v == "1"
		hasCustomers = &b
	}

	groups, total, err := h.svc.GetAll(c.Request.Context(), limit, offset, search, isActive, hasCustomers)
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	shared.JSONPaginated(c, groups, total, limit, offset)
}

// GetByID godoc
// @Summary      Get customer group by ID
// @Description  Get a single customer group by its ID
// @Tags         Customer Groups
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Customer Group ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /customer-groups/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	group, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "customer group not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": group})
}

// Create godoc
// @Summary      Create a customer group
// @Description  Create a new customer group
// @Tags         Customer Groups
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      CustomerGroupCreateRequest  true  "Customer Group data"
// @Success      201   {object}  map[string]interface{}
// @Router       /customer-groups [post]
func (h *Handler) Create(c *gin.Context) {
	var req CustomerGroupCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group, err := h.svc.Create(c.Request.Context(), req)
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
			EntityType:  "customer_group",
			EntityID:    &group.ID,
			NewValues:   shared.ToJSONMap(group),
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Created customer group %s", group.Name),
		})
	}

	c.JSON(http.StatusCreated, gin.H{"data": group})
}

// Update godoc
// @Summary      Update a customer group
// @Description  Update an existing customer group
// @Tags         Customer Groups
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int                          true  "Customer Group ID"
// @Param        body  body      CustomerGroupUpdateRequest   true  "Update data"
// @Success      200   {object}  map[string]interface{}
// @Router       /customer-groups/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req CustomerGroupUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group, err := h.svc.Update(c.Request.Context(), id, req)
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
			EntityType:  "customer_group",
			EntityID:    &group.ID,
			NewValues:   shared.ToJSONMap(group),
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Updated customer group %s", group.Name),
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": group})
}

// Delete godoc
// @Summary      Delete a customer group
// @Description  Delete a customer group by ID
// @Tags         Customer Groups
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Customer Group ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /customer-groups/{id} [delete]
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
			EntityType:  "customer_group",
			EntityID:    &id,
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Deleted customer group #%d", id),
		})
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// BulkUpdate godoc
// @Summary      Bulk update customer groups
// @Description  Activate or deactivate multiple customer groups
// @Tags         Customer Groups
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      object  true  "Bulk update payload"
// @Success      200   {object}  map[string]interface{}
// @Router       /customer-groups/bulk [put]
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
			EntityType:  "customer_group",
			NewValues:   map[string]interface{}{"ids": req.IDs, "is_active": req.IsActive},
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Bulk updated %d customer groups", updated),
		})
	}

	c.JSON(http.StatusOK, gin.H{"updated": updated})
}

// BulkDelete godoc
// @Summary      Bulk delete customer groups
// @Description  Delete multiple customer groups
// @Tags         Customer Groups
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      object  true  "Bulk delete payload"
// @Success      200   {object}  map[string]interface{}
// @Router       /customer-groups/bulk [delete]
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
			EntityType:  "customer_group",
			NewValues:   map[string]interface{}{"ids": req.IDs},
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Bulk deleted %d customer groups", deleted),
		})
	}

	c.JSON(http.StatusOK, gin.H{"deleted": deleted})
}
