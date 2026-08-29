package inventory

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/permissions"
	"retail-pos-system/internal/shared"

	"github.com/gin-gonic/gin"
)

type Service interface {
	AdjustStock(ctx context.Context, productID int, quantityChange int, storeID *int, userID int, notes string) error
	AdjustStockBatch(ctx context.Context, adjustments []StockAdjustment, userID int, notes string) error
	GetStockByProductID(ctx context.Context, productID int) (*ProductStock, error)
	ListLocationStock(ctx context.Context, productID, locationID int, storeID *int) ([]LocationStockItem, error)
	SetLocationStock(ctx context.Context, productID, locationID, quantity, userID int, storeID *int) error
	TransferLocationStock(ctx context.Context, productID, fromLocationID, toLocationID, quantity, userID int, storeID *int) error
}

type Handler struct {
	svc      Service
	auditSvc audit.Creator
}

func NewHandler(svc Service, auditSvc audit.Creator) *Handler {
	return &Handler{svc: svc, auditSvc: auditSvc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(permissions.Code) gin.HandlerFunc) {
	r.POST("/inventory/adjust", auth, perm(permissions.InventoryAdjust), h.AdjustStock)
	r.GET("/inventory/locations", auth, perm(permissions.ProductView), h.ListLocationStock)
	r.POST("/inventory/locations", auth, perm(permissions.InventoryAdjust), h.SetLocationStock)
	r.POST("/inventory/locations/transfer", auth, perm(permissions.InventoryAdjust), h.TransferLocationStock)
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
	storeID := shared.GetStoreID(c)
	if err := h.svc.AdjustStock(c.Request.Context(), req.ProductID, req.QuantityChange, storeID, uid, req.Notes); err != nil {
		if errors.Is(err, ErrStoreForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		shared.InternalError(c, err)
		return
	}

	if h.auditSvc != nil {
		actorID := middleware.UserIDFromContext(c.Request.Context())
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.Log{
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
			StoreID:     middleware.StoreIDFromContext(c.Request.Context()),
		})
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
