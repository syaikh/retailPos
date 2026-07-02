package uom

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
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
	c.JSON(http.StatusCreated, gin.H{"data": uom})
}

func (h *Handler) UpdateUnitOfMeasure(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid unit of measure id"})
		return
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
	c.JSON(http.StatusOK, gin.H{"data": uom})
}

func (h *Handler) DeleteUnitOfMeasure(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid unit of measure id"})
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete unit of measure"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}


