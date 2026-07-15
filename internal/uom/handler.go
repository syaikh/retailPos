package uom

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/shared"

	"github.com/gin-gonic/gin"
)

type UOMService interface {
	GetByID(ctx context.Context, id int) (*UnitOfMeasure, error)
	GetAll(ctx context.Context) ([]UnitOfMeasure, error)
	Create(ctx context.Context, req *UOMCreateRequest) (*UnitOfMeasure, error)
	Update(ctx context.Context, id int, req *UOMUpdateRequest) (*UnitOfMeasure, error)
	Delete(ctx context.Context, id int) error
}

type Handler struct {
	svc      UOMService
	auditSvc audit.AuditCreator
}

func NewHandler(svc UOMService, auditSvc audit.AuditCreator) *Handler {
	return &Handler{svc: svc, auditSvc: auditSvc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(string) gin.HandlerFunc) {
	r.POST("/units-of-measure", auth, perm("product:create"), h.CreateUnitOfMeasure)
	r.PUT("/units-of-measure/:id", auth, perm("product:update"), h.UpdateUnitOfMeasure)
	r.DELETE("/units-of-measure/:id", auth, perm("product:delete"), h.DeleteUnitOfMeasure)
}

func (h *Handler) RegisterPublicRoutes(r *gin.RouterGroup) {
	r.GET("/units-of-measure", h.ListUnitsOfMeasure)
}

func (h *Handler) ListUnitsOfMeasure(c *gin.Context) {
	units, err := h.svc.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch units of measure"})
		return
	}
	if units == nil {
		units = []UnitOfMeasure{}
	}
	c.JSON(http.StatusOK, gin.H{"data": units})
}

func (h *Handler) CreateUnitOfMeasure(c *gin.Context) {
	var req UOMCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uom, err := h.svc.Create(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create unit of measure"})
		return
	}

	if h.auditSvc != nil {
		userID := middleware.UserIDFromContext(c.Request.Context())
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      userID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "create",
			EntityType:  "uom",
			EntityID:    &uom.ID,
			NewValues:   shared.ToJSONMap(uom),
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Created unit of measure %s", uom.Name),
		})
	}
	c.JSON(http.StatusCreated, gin.H{"data": uom})
}

func (h *Handler) UpdateUnitOfMeasure(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid unit of measure id"})
		return
	}

	var oldUOM *UnitOfMeasure
	if h.auditSvc != nil {
		oldUOM, _ = h.svc.GetByID(c.Request.Context(), id)
	}

	var req UOMUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uom, err := h.svc.Update(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update unit of measure"})
		return
	}

	if h.auditSvc != nil {
		userID := middleware.UserIDFromContext(c.Request.Context())
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      userID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "update",
			EntityType:  "uom",
			EntityID:    &uom.ID,
			OldValues:   shared.ToJSONMap(oldUOM),
			NewValues:   shared.ToJSONMap(uom),
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Updated unit of measure %s", uom.Name),
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": uom})
}

func (h *Handler) DeleteUnitOfMeasure(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid unit of measure id"})
		return
	}

	var oldUOMName string
	if h.auditSvc != nil {
		if u, err := h.svc.GetByID(c.Request.Context(), id); err == nil {
			oldUOMName = u.Name
		}
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete unit of measure"})
		return
	}

	if h.auditSvc != nil {
		userID := middleware.UserIDFromContext(c.Request.Context())
		var description string
		if oldUOMName != "" {
			description = fmt.Sprintf("Deleted unit of measure %s", oldUOMName)
		} else {
			description = fmt.Sprintf("Deleted unit of measure #%d", id)
		}
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      userID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "delete",
			EntityType:  "uom",
			EntityID:    &id,
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: description,
		})
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}


