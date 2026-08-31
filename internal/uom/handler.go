package uom

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/permissions"
	"retail-pos-system/internal/shared"

	"github.com/gin-gonic/gin"
)

type Service interface {
	GetByID(ctx context.Context, id int) (*UnitOfMeasure, error)
	GetAll(ctx context.Context) ([]UnitOfMeasure, error)
	GetAllPaginated(ctx context.Context, limit, offset int, search string) ([]UnitOfMeasure, int, error)
	GetIDByCode(ctx context.Context, code string) (int, error)
	Create(ctx context.Context, req *CreateRequest) (*UnitOfMeasure, error)
	Update(ctx context.Context, id int, req *UpdateRequest) (*UnitOfMeasure, error)
	Delete(ctx context.Context, id int) error
}

type Handler struct {
	svc      Service
	auditSvc audit.Creator
}

func NewHandler(svc Service, auditSvc audit.Creator) *Handler {
	return &Handler{svc: svc, auditSvc: auditSvc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(permissions.Code) gin.HandlerFunc) {
	r.POST("/units-of-measure", auth, perm(permissions.ProductCreate), h.CreateUnitOfMeasure)
	r.PUT("/units-of-measure/:id", auth, perm(permissions.ProductUpdate), h.UpdateUnitOfMeasure)
	r.DELETE("/units-of-measure/:id", auth, perm(permissions.ProductDelete), h.DeleteUnitOfMeasure)
}

func (h *Handler) RegisterPublicRoutes(r *gin.RouterGroup) {
	r.GET("/units-of-measure", h.ListUnitsOfMeasure)
}

func (h *Handler) ListUnitsOfMeasure(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	search := c.Query("search")

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	units, total, err := h.svc.GetAllPaginated(c.Request.Context(), limit, offset, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch units of measure"})
		return
	}
	if units == nil {
		units = []UnitOfMeasure{}
	}
	c.JSON(http.StatusOK, gin.H{"data": units, "total": total})
}

func (h *Handler) CreateUnitOfMeasure(c *gin.Context) {
	var req CreateRequest
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
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.Log{
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
			StoreID:     middleware.StoreIDFromContext(c.Request.Context()),
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

	var req UpdateRequest
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
		focusedOld, focusedNew := shared.DiffChanges(shared.ToJSONMap(oldUOM), shared.ToJSONMap(uom))
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.Log{
			UserID:      userID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "update",
			EntityType:  "uom",
			EntityID:    &uom.ID,
			OldValues:   focusedOld,
			NewValues:   focusedNew,
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Updated unit of measure %s", uom.Name),
			StoreID:     middleware.StoreIDFromContext(c.Request.Context()),
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
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.Log{
			UserID:      userID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "delete",
			EntityType:  "uom",
			EntityID:    &id,
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: description,
			StoreID:     middleware.StoreIDFromContext(c.Request.Context()),
		})
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
