package brand

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
	auditSvc *audit.Service
}

func NewHandler(svc *Service, auditSvc *audit.Service) *Handler {
	return &Handler{svc: svc, auditSvc: auditSvc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(string) gin.HandlerFunc) {
	r.POST("/brands", auth, perm("product:create"), h.CreateBrand)
	r.PUT("/brands/:id", auth, perm("product:update"), h.UpdateBrand)
	r.DELETE("/brands/:id", auth, perm("product:delete"), h.DeleteBrand)
}

func (h *Handler) RegisterPublicRoutes(r *gin.RouterGroup) {
	r.GET("/brands", h.ListBrands)
}

func (h *Handler) ListBrands(c *gin.Context) {
	brands, err := h.svc.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch brands"})
		return
	}
	if brands == nil {
		brands = []Brand{}
	}
	c.JSON(http.StatusOK, gin.H{"data": brands})
}

func (h *Handler) CreateBrand(c *gin.Context) {
	var req BrandCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	brand, err := h.svc.Create(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create brand"})
		return
	}

	if h.auditSvc != nil {
		userID := middleware.UserIDFromContext(c.Request.Context())
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      userID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "create",
			EntityType:  "brand",
			EntityID:    &brand.ID,
			NewValues:   shared.ToJSONMap(brand),
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Created brand %s", brand.Name),
		})
	}
	c.JSON(http.StatusCreated, gin.H{"data": brand})
}

func (h *Handler) UpdateBrand(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid brand id"})
		return
	}

	var oldBrand *Brand
	if h.auditSvc != nil {
		oldBrand, _ = h.svc.GetByID(c.Request.Context(), id)
	}

	var req BrandUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	brand, err := h.svc.Update(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update brand"})
		return
	}

	if h.auditSvc != nil {
		userID := middleware.UserIDFromContext(c.Request.Context())
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      userID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "update",
			EntityType:  "brand",
			EntityID:    &brand.ID,
			OldValues:   shared.ToJSONMap(oldBrand),
			NewValues:   shared.ToJSONMap(brand),
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Updated brand %s", brand.Name),
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": brand})
}

func (h *Handler) DeleteBrand(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid brand id"})
		return
	}

	var oldBrandName string
	if h.auditSvc != nil {
		if b, err := h.svc.GetByID(c.Request.Context(), id); err == nil {
			oldBrandName = b.Name
		}
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete brand"})
		return
	}

	if h.auditSvc != nil {
		userID := middleware.UserIDFromContext(c.Request.Context())
		var description string
		if oldBrandName != "" {
			description = fmt.Sprintf("Deleted brand %s", oldBrandName)
		} else {
			description = fmt.Sprintf("Deleted brand #%d", id)
		}
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      userID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "delete",
			EntityType:  "brand",
			EntityID:    &id,
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: description,
		})
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}


