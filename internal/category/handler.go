package category

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

type CategoryService interface {
	ListCategories(ctx context.Context) ([]Category, error)
	GetCategoryByID(ctx context.Context, id int) (*Category, error)
	GetAllCategories(ctx context.Context, limit, offset int, search string) ([]Category, int, error)
	CreateCategory(ctx context.Context, req *CategoryCreateRequest) (*Category, error)
	UpdateCategory(ctx context.Context, id int, req *CategoryUpdateRequest) (*Category, error)
	DeleteCategory(ctx context.Context, id int) error
}

type Handler struct {
	svc      CategoryService
	auditSvc audit.AuditCreator
}

func NewHandler(svc CategoryService, auditSvc audit.AuditCreator) *Handler {
	return &Handler{svc: svc, auditSvc: auditSvc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(string) gin.HandlerFunc) {
	r.GET("/categories", h.ListCategories)
	r.GET("/categories/manage", auth, perm("category:read"), h.ListCategoriesManagement)
	r.POST("/categories", auth, perm("category:create"), h.CreateCategoryHandler)
	r.PUT("/categories/:id", auth, perm("category:update"), h.UpdateCategoryHandler)
	r.DELETE("/categories/:id", auth, perm("category:delete"), h.DeleteCategoryHandler)
}

func (h *Handler) ListCategories(c *gin.Context) {
	categories, err := h.svc.ListCategories(c.Request.Context())
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	if categories == nil {
		categories = []Category{}
	}
	c.JSON(http.StatusOK, gin.H{"data": categories})
}

func (h *Handler) ListCategoriesManagement(c *gin.Context) {
	limit, offset := shared.ParsePaginationParams(c.Query("limit"), c.Query("offset"))
	search := c.Query("search")

	categories, total, err := h.svc.GetAllCategories(c.Request.Context(), limit, offset, search)
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	shared.JSONPaginated(c, categories, total, limit, offset)
}

func (h *Handler) CreateCategoryHandler(c *gin.Context) {
	var req CategoryCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	category, err := h.svc.CreateCategory(c.Request.Context(), &req)
	if err != nil {
		shared.InternalError(c, err)
		return
	}

	if h.auditSvc != nil {
		userID := middleware.UserIDFromContext(c.Request.Context())
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      userID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "create",
			EntityType:  "category",
			EntityID:    &category.ID,
			NewValues:   shared.ToJSONMap(category),
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Created category %s", category.Name),
		})
	}
	c.JSON(http.StatusCreated, gin.H{"data": category})
}

func (h *Handler) UpdateCategoryHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var oldCategory *Category
	if h.auditSvc != nil {
		oldCategory, _ = h.svc.GetCategoryByID(c.Request.Context(), id)
	}

	var req CategoryUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	category, err := h.svc.UpdateCategory(c.Request.Context(), id, &req)
	if err != nil {
		shared.InternalError(c, err)
		return
	}

	if h.auditSvc != nil {
		userID := middleware.UserIDFromContext(c.Request.Context())
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      userID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "update",
			EntityType:  "category",
			EntityID:    &category.ID,
			OldValues:   shared.ToJSONMap(oldCategory),
			NewValues:   shared.ToJSONMap(category),
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Updated category %s", category.Name),
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": category})
}

func (h *Handler) DeleteCategoryHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var oldCategoryName string
	if h.auditSvc != nil {
		if cat, err := h.svc.GetCategoryByID(c.Request.Context(), id); err == nil {
			oldCategoryName = cat.Name
		}
	}

	if err := h.svc.DeleteCategory(c.Request.Context(), id); err != nil {
		shared.InternalError(c, err)
		return
	}

	if h.auditSvc != nil {
		userID := middleware.UserIDFromContext(c.Request.Context())
		var description string
		if oldCategoryName != "" {
			description = fmt.Sprintf("Deleted category %s", oldCategoryName)
		} else {
			description = fmt.Sprintf("Deleted category #%d", id)
		}
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      userID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "delete",
			EntityType:  "category",
			EntityID:    &id,
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: description,
		})
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
