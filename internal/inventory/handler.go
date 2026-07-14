package inventory

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/shared"

	"github.com/gin-gonic/gin"
)

type InventoryService interface {
	AdjustStock(ctx context.Context, productID int, quantityChange int, userID int, notes string) error
}

type AuditCreator interface {
	CreateAuditLog(ctx context.Context, log *audit.AuditLog) error
}

type Handler struct {
	svc      InventoryService
	auditSvc AuditCreator
}

func NewHandler(svc InventoryService, auditSvc AuditCreator) *Handler {
	return &Handler{svc: svc, auditSvc: auditSvc}
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

	if h.auditSvc != nil {
		actorID := middleware.UserIDFromContext(c.Request.Context())
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      actorID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "update",
			EntityType:  "inventory",
			EntityID:    &req.ProductID,
			NewValues:   shared.ToJSONMap(map[string]interface{}{"product_id": req.ProductID, "quantity_change": req.QuantityChange, "notes": req.Notes}),
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Adjusted stock for product #%d by %d", req.ProductID, req.QuantityChange),
		})
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
