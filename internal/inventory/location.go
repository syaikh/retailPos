package inventory

import (
	"fmt"
	"net/http"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/shared"

	"github.com/gin-gonic/gin"
)

// ListLocationStock godoc
// @Summary      List rack-level stock
// @Description  Rack stock rows (product per storage location). Filter by product_id and/or location_id.
// @Tags         Inventory
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        product_id  query  int  false  "Product ID"
// @Param        location_id query  int  false  "Storage location ID"
// @Success      200  {array}   LocationStockItem
// @Router       /inventory/locations [get]
func (h *Handler) ListLocationStock(c *gin.Context) {
	productID, _ := shared.ParseIntParam(c.Query("product_id"))
	locationID, _ := shared.ParseIntParam(c.Query("location_id"))
	items, err := h.svc.ListLocationStock(c.Request.Context(), productID, locationID)
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

// SetLocationStock godoc
// @Summary      Record rack stock for a product
// @Description  Sets how much of a product sits in a storage location (upsert). Does not change global stock.
// @Tags         Inventory
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      object  true  "{product_id, location_id, quantity}"
// @Success      200  {object}  map[string]interface{}
// @Router       /inventory/locations [post]
func (h *Handler) SetLocationStock(c *gin.Context) {
	var req struct {
		ProductID  int `json:"product_id"`
		LocationID int `json:"location_id"`
		Quantity   int `json:"quantity"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.ProductID <= 0 || req.LocationID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product_id and location_id are required"})
		return
	}
	uid, ok := contextUserID(c)
	if !ok {
		return
	}
	if err := h.svc.SetLocationStock(c.Request.Context(), req.ProductID, req.LocationID, req.Quantity, uid); err != nil {
		badRequestOrInternal(c, err)
		return
	}
	h.auditLocation(c, "Set rack stock", req.ProductID, map[string]interface{}{
		"location_id": req.LocationID, "quantity": req.Quantity,
	})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// TransferLocationStock godoc
// @Summary      Transfer stock between racks
// @Description  Moves quantity of a product from one storage location to another. Global stock is unchanged.
// @Tags         Inventory
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      object  true  "{product_id, from_location_id, to_location_id, quantity}"
// @Success      200  {object}  map[string]interface{}
// @Router       /inventory/locations/transfer [post]
func (h *Handler) TransferLocationStock(c *gin.Context) {
	var req struct {
		ProductID      int `json:"product_id"`
		FromLocationID int `json:"from_location_id"`
		ToLocationID   int `json:"to_location_id"`
		Quantity       int `json:"quantity"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.ProductID <= 0 || req.FromLocationID <= 0 || req.ToLocationID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product_id, from_location_id and to_location_id are required"})
		return
	}
	uid, ok := contextUserID(c)
	if !ok {
		return
	}
	if err := h.svc.TransferLocationStock(c.Request.Context(), req.ProductID, req.FromLocationID, req.ToLocationID, req.Quantity, uid); err != nil {
		badRequestOrInternal(c, err)
		return
	}
	h.auditLocation(c, "Transfer rack stock", req.ProductID, map[string]interface{}{
		"from_location_id": req.FromLocationID, "to_location_id": req.ToLocationID, "quantity": req.Quantity,
	})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func contextUserID(c *gin.Context) (int, bool) {
	userID, _ := c.Get("userID")
	uid, ok := userID.(int)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return 0, false
	}
	return uid, true
}

func badRequestOrInternal(c *gin.Context, err error) {
	switch err {
	case ErrInsufficientLocationStock, ErrLocationInactive, ErrLocationNotFound, ErrSameLocation,
		ErrNegativeQuantity, ErrNonPositiveQuantity:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		shared.InternalError(c, err)
	}
}

func (h *Handler) auditLocation(c *gin.Context, description string, productID int, newValues map[string]interface{}) {
	if h.auditSvc == nil {
		return
	}
	_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
		UserID:      middleware.UserIDFromContext(c.Request.Context()),
		Username:    middleware.UsernameFromContext(c.Request.Context()),
		Role:        middleware.RoleFromContext(c.Request.Context()),
		Action:      "update",
		EntityType:  "inventory_location",
		EntityID:    &productID,
		NewValues:   shared.ToJSONMap(newValues),
		IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
		UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
		Description: fmt.Sprintf("%s for product #%d", description, productID),
	})
}
