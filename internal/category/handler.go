package category

import (
	"net/http"
	"strconv"

	"retail-pos-system/internal/shared"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
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
	limit := 50
	if l, err := strconv.Atoi(c.DefaultQuery("limit", "50")); err == nil && l > 0 {
		limit = l
	}
	if limit > 200 {
		limit = 200
	}
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if offset < 0 {
		offset = 0
	}
	search := c.Query("search")

	categories, total, err := h.svc.GetAllCategories(c.Request.Context(), limit, offset, search)
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": categories, "total": total})
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
	c.JSON(http.StatusCreated, gin.H{"data": category})
}

func (h *Handler) UpdateCategoryHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
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
	c.JSON(http.StatusOK, gin.H{"data": category})
}

func (h *Handler) DeleteCategoryHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.svc.DeleteCategory(c.Request.Context(), id); err != nil {
		shared.InternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}


