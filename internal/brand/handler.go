package brand

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
	GetByID(ctx context.Context, id int) (*Brand, error)
	GetAll(ctx context.Context) ([]Brand, error)
	GetAllPaginated(ctx context.Context, limit, offset int, search string) ([]Brand, int, error)
	GetIDByName(ctx context.Context, name string) (int, error)
	Create(ctx context.Context, req *CreateRequest) (*Brand, error)
	Update(ctx context.Context, id int, req *UpdateRequest) (*Brand, error)
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
	r.POST("/brands", auth, perm(permissions.ProductCreate), h.CreateBrand)
	r.PUT("/brands/:id", auth, perm(permissions.ProductUpdate), h.UpdateBrand)
	r.DELETE("/brands/:id", auth, perm(permissions.ProductDelete), h.DeleteBrand)
}

func (h *Handler) RegisterPublicRoutes(r *gin.RouterGroup) {
	r.GET("/brands", h.ListBrands)
}

func (h *Handler) ListBrands(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	search := c.Query("search")

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	brands, total, err := h.svc.GetAllPaginated(c.Request.Context(), limit, offset, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch brands"})
		return
	}
	if brands == nil {
		brands = []Brand{}
	}
	c.JSON(http.StatusOK, gin.H{"data": brands, "total": total})
}

func (h *Handler) CreateBrand(c *gin.Context) {
	var req CreateRequest
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
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.Log{
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
			StoreID:     middleware.StoreIDFromContext(c.Request.Context()),
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

	var req UpdateRequest
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
		focusedOld, focusedNew := shared.DiffChanges(shared.ToJSONMap(oldBrand), shared.ToJSONMap(brand))
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.Log{
			UserID:      userID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "update",
			EntityType:  "brand",
			EntityID:    &brand.ID,
			OldValues:   focusedOld,
			NewValues:   focusedNew,
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Updated brand %s", brand.Name),
			StoreID:     middleware.StoreIDFromContext(c.Request.Context()),
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
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.Log{
			UserID:      userID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "delete",
			EntityType:  "brand",
			EntityID:    &id,
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: description,
			StoreID:     middleware.StoreIDFromContext(c.Request.Context()),
		})
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
