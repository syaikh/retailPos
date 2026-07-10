package inventory

import (
	"net/http"
	"strings"

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
	r.POST("/inventory/adjust", auth, perm("inventory:adjust"), h.AdjustStock)
}

func (h *Handler) AdjustStock(c *gin.Context) {
	var req struct {
		ProductID      int    `json:"product_id"`
		QuantityChange int    `json:"quantity_change"`
		Notes          string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.QuantityChange == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "quantity change must not be zero"})
		return
	}
	if strings.TrimSpace(req.Notes) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "notes are required"})
		return
	}
	userID, _ := c.Get("userID")
	uid, ok := userID.(int)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	if err := h.svc.AdjustStock(c.Request.Context(), req.ProductID, req.QuantityChange, uid, req.Notes); err != nil {
		shared.InternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
