package sale

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/permissions"
	"retail-pos-system/internal/shared"
)

type CartService interface {
	CreateOrGetOpenCart(ctx context.Context, cashierID int, storeID, shiftID, customerID *int) (*CartSession, error)
	GetOpenCart(ctx context.Context, cashierID int) (*CartSession, error)
	GetCartByID(ctx context.Context, cartID int, cashierID int) (*CartSession, error)
	ListHeldCarts(ctx context.Context, cashierID int) ([]CartSession, error)
	UpdateCartCustomer(ctx context.Context, cartID int, customerID *int, cashierID int) (*CartSession, error)
	AddCartItem(ctx context.Context, cartID int, productID, quantity int, customerGroupID *int, cashierID int) (*CartSession, error)
	UpdateCartItemQuantity(ctx context.Context, cartID, itemID, quantity int, cashierID int) (*CartSession, error)
	RemoveCartItem(ctx context.Context, cartID, itemID int, cashierID int) (*CartSession, error)
	HoldCart(ctx context.Context, cartID int, cashierID int) (*CartSession, error)
	ResumeCart(ctx context.Context, cartID int, cashierID int) (*CartSession, error)
	CancelCart(ctx context.Context, cartID int, cashierID int) (*CartSession, error)
	CheckoutCart(ctx context.Context, cartID int, payments []CreatePaymentRequest, cashierID int) (*Sale, error)
}

// RegisterCartRoutes registers the POS cart API under the given router group.
func (h *Handler) RegisterCartRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(permissions.Code) gin.HandlerFunc) {
	r.POST("/pos/cart", auth, perm(permissions.SaleCreate), h.CreateOrGetOpenCart)
	r.GET("/pos/cart", auth, perm(permissions.SaleCreate), h.GetOpenCart)
	r.GET("/pos/cart/held", auth, perm(permissions.SaleCreate), h.ListHeldCarts)
	r.GET("/pos/cart/:id", auth, perm(permissions.SaleCreate), h.GetCart)
	r.POST("/pos/cart/items", auth, perm(permissions.SaleCreate), h.AddCartItem)
	r.PATCH("/pos/cart/items/:itemId", auth, perm(permissions.SaleCreate), h.UpdateCartItemQuantity)
	r.DELETE("/pos/cart/items/:itemId", auth, perm(permissions.SaleCreate), h.RemoveCartItem)
	r.PATCH("/pos/cart/:id/customer", auth, perm(permissions.SaleCreate), h.UpdateCartCustomer)
	r.POST("/pos/cart/:id/hold", auth, perm(permissions.SaleCreate), h.HoldCart)
	r.POST("/pos/cart/:id/resume", auth, perm(permissions.SaleCreate), h.ResumeCart)
	r.POST("/pos/cart/:id/cancel", auth, perm(permissions.SaleCreate), h.CancelCart)
	r.POST("/pos/cart/:id/checkout", auth, perm(permissions.SaleCreate), h.CheckoutCart)
}

func (h *Handler) cartCashierID(c *gin.Context) (int, bool) {
	userID, exists := c.Get("userID")
	if !exists {
		shared.JSONError(c, http.StatusUnauthorized, shared.ErrUnauthorized, "user not authenticated")
		return 0, false
	}
	cashierID, ok := userID.(int)
	if !ok {
		shared.JSONError(c, http.StatusUnauthorized, shared.ErrUnauthorized, "invalid user ID in context")
		return 0, false
	}
	return cashierID, true
}

func (h *Handler) cartParamID(c *gin.Context, name string) (int, bool) {
	id, err := strconv.Atoi(c.Param(name))
	if err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "invalid "+name)
		return 0, false
	}
	return id, true
}

func (h *Handler) cartError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrCartNotFound):
		shared.JSONError(c, http.StatusNotFound, shared.ErrNotFound, err.Error())
	case errors.Is(err, ErrCartNotOwned):
		shared.JSONError(c, http.StatusForbidden, shared.ErrForbidden, err.Error())
	case errors.Is(err, ErrCartNotOpen), errors.Is(err, ErrCartItemNotFound), errors.Is(err, ErrCartEmpty):
		shared.JSONError(c, http.StatusConflict, shared.ErrConflict, err.Error())
	case errors.Is(err, ErrCartExpired):
		shared.JSONError(c, http.StatusConflict, shared.ErrConflict, err.Error())
	case errors.Is(err, ErrCartAlreadyCheckedOut):
		shared.JSONError(c, http.StatusConflict, shared.ErrConflict, err.Error())
	case errors.Is(err, ErrInsufficientStock):
		shared.JSONError(c, http.StatusConflict, shared.ErrConflict, "insufficient stock")
	case errors.Is(err, ErrPaymentOverTenderNonCash) || errors.Is(err, ErrPaymentTotalMismatch) || errors.Is(err, ErrDuplicatePaymentMethod) ||
		errors.Is(err, ErrPaymentMethodInactive) || errors.Is(err, ErrPaymentReferenceRequired) ||
		errors.Is(err, ErrZeroPaymentAmount) || errors.Is(err, ErrInvalidPaymentMethod) ||
		errors.Is(err, ErrMaxPaymentsExceeded) || errors.Is(err, ErrMultipleCashPayments):
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, err.Error())
	case errors.Is(err, ErrPriceMismatch):
		shared.JSONError(c, http.StatusConflict, shared.ErrConflict, err.Error())
	case errors.Is(err, ErrCheckoutProductNotFound):
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, err.Error())
	case errors.Is(err, shared.ErrShiftNotOpen):
		shared.JSONError(c, http.StatusConflict, shared.ErrConflict, "shift is closed or no longer exists")
	default:
		shared.InternalError(c, err)
	}
}

// CreateOrGetOpenCart godoc
// @Summary Get or create the open cart for the current cashier
// @Tags cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /pos/cart [post]
func (h *Handler) CreateOrGetOpenCart(c *gin.Context) {
	var req CreateCartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = CreateCartRequest{}
	}

	cashierID, ok := h.cartCashierID(c)
	if !ok {
		return
	}
	storeID := shared.GetStoreID(c)

	ctx := c.Request.Context()
	cart, err := h.svc.CreateOrGetOpenCart(ctx, cashierID, storeID, req.ShiftID, req.CustomerID)
	if err != nil {
		h.cartError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": presentCart(cart, canViewCost(c))})
}

// GetOpenCart godoc
// @Summary Get the open cart for the current cashier
// @Tags cart
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /pos/cart [get]
func (h *Handler) GetOpenCart(c *gin.Context) {
	cashierID, ok := h.cartCashierID(c)
	if !ok {
		return
	}

	cart, err := h.svc.GetOpenCart(c.Request.Context(), cashierID)
	if err != nil {
		if errors.Is(err, ErrCartNotFound) {
			c.JSON(http.StatusOK, gin.H{"data": nil})
			return
		}
		h.cartError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": presentCart(cart, canViewCost(c))})
}

// ListHeldCarts godoc
// @Summary List held carts for the current cashier
// @Tags cart
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /pos/cart/held [get]
func (h *Handler) ListHeldCarts(c *gin.Context) {
	cashierID, ok := h.cartCashierID(c)
	if !ok {
		return
	}

	carts, err := h.svc.ListHeldCarts(c.Request.Context(), cashierID)
	if err != nil {
		h.cartError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": presentCarts(carts, canViewCost(c))})
}

// GetCart godoc
// @Summary Get a cart by ID
// @Tags cart
// @Produce json
// @Security BearerAuth
// @Param id path int true "Cart ID"
// @Success 200 {object} map[string]interface{}
// @Router /pos/cart/{id} [get]
func (h *Handler) GetCart(c *gin.Context) {
	cartID, ok := h.cartParamID(c, "id")
	if !ok {
		return
	}

	cashierID, ok := h.cartCashierID(c)
	if !ok {
		return
	}

	cart, err := h.svc.GetCartByID(c.Request.Context(), cartID, cashierID)
	if err != nil {
		h.cartError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": presentCart(cart, canViewCost(c))})
}

// AddCartItem godoc
// @Summary Add an item to the open cart
// @Tags cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body AddCartItemRequest true "Item payload"
// @Success 200 {object} map[string]interface{}
// @Router /pos/cart/items [post]
func (h *Handler) AddCartItem(c *gin.Context) {
	var req AddCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, err.Error())
		return
	}

	cashierID, ok := h.cartCashierID(c)
	if !ok {
		return
	}

	ctx := c.Request.Context()
	storeID := req.StoreID
	if storeID == nil {
		storeID = shared.GetStoreID(c)
	}
	cart, err := h.svc.CreateOrGetOpenCart(ctx, cashierID, storeID, req.ShiftID, req.CustomerID)
	if err != nil {
		h.cartError(c, err)
		return
	}

	cart, err = h.svc.AddCartItem(ctx, cart.ID, req.ProductID, req.Quantity, req.CustomerGroupID, cashierID)
	if err != nil {
		h.cartError(c, err)
		return
	}

	h.auditCart(c, "add", "cart_item", "Added product #%d x%d to cart", req.ProductID, req.Quantity)
	c.JSON(http.StatusOK, gin.H{"data": presentCart(cart, canViewCost(c))})
}

// UpdateCartItemQuantity godoc
// @Summary Update the quantity of a cart item
// @Tags cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param itemId path int true "Cart Item ID"
// @Param request body UpdateCartItemQuantityRequest true "Quantity payload"
// @Success 200 {object} map[string]interface{}
// @Router /pos/cart/items/{itemId} [patch]
func (h *Handler) UpdateCartItemQuantity(c *gin.Context) {
	itemID, ok := h.cartParamID(c, "itemId")
	if !ok {
		return
	}
	var req UpdateCartItemQuantityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, err.Error())
		return
	}

	cashierID, ok := h.cartCashierID(c)
	if !ok {
		return
	}

	ctx := c.Request.Context()
	cart, err := h.svc.GetOpenCart(ctx, cashierID)
	if err != nil {
		h.cartError(c, err)
		return
	}

	cart, err = h.svc.UpdateCartItemQuantity(ctx, cart.ID, itemID, req.Quantity, cashierID)
	if err != nil {
		h.cartError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": presentCart(cart, canViewCost(c))})
}

// RemoveCartItem godoc
// @Summary Remove an item from the open cart
// @Tags cart
// @Security BearerAuth
// @Param itemId path int true "Cart Item ID"
// @Success 200 {object} map[string]interface{}
// @Router /pos/cart/items/{itemId} [delete]
func (h *Handler) RemoveCartItem(c *gin.Context) {
	itemID, ok := h.cartParamID(c, "itemId")
	if !ok {
		return
	}

	cashierID, ok := h.cartCashierID(c)
	if !ok {
		return
	}

	ctx := c.Request.Context()
	cart, err := h.svc.GetOpenCart(ctx, cashierID)
	if err != nil {
		h.cartError(c, err)
		return
	}

	cart, err = h.svc.RemoveCartItem(ctx, cart.ID, itemID, cashierID)
	if err != nil {
		h.cartError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": presentCart(cart, canViewCost(c))})
}

// UpdateCartCustomer godoc
// @Summary Set the customer on a cart
// @Tags cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Cart ID"
// @Param request body UpdateCartCustomerRequest true "Customer payload"
// @Success 200 {object} map[string]interface{}
// @Router /pos/cart/{id}/customer [patch]
func (h *Handler) UpdateCartCustomer(c *gin.Context) {
	cartID, ok := h.cartParamID(c, "id")
	if !ok {
		return
	}
	var req UpdateCartCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, err.Error())
		return
	}

	cashierID, ok := h.cartCashierID(c)
	if !ok {
		return
	}

	cart, err := h.svc.UpdateCartCustomer(c.Request.Context(), cartID, req.CustomerID, cashierID)
	if err != nil {
		h.cartError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": presentCart(cart, canViewCost(c))})
}

// HoldCart godoc
// @Summary Hold the cart, parking it for later recall
// @Tags cart
// @Produce json
// @Security BearerAuth
// @Param id path int true "Cart ID"
// @Success 200 {object} map[string]interface{}
// @Router /pos/cart/{id}/hold [post]
func (h *Handler) HoldCart(c *gin.Context) {
	cartID, ok := h.cartParamID(c, "id")
	if !ok {
		return
	}

	cashierID, ok := h.cartCashierID(c)
	if !ok {
		return
	}

	cart, err := h.svc.HoldCart(c.Request.Context(), cartID, cashierID)
	if err != nil {
		h.cartError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": presentCart(cart, canViewCost(c))})
}

// ResumeCart godoc
// @Summary Resume a held cart
// @Tags cart
// @Produce json
// @Security BearerAuth
// @Param id path int true "Cart ID"
// @Success 200 {object} map[string]interface{}
// @Router /pos/cart/{id}/resume [post]
func (h *Handler) ResumeCart(c *gin.Context) {
	cartID, ok := h.cartParamID(c, "id")
	if !ok {
		return
	}

	cashierID, ok := h.cartCashierID(c)
	if !ok {
		return
	}

	cart, err := h.svc.ResumeCart(c.Request.Context(), cartID, cashierID)
	if err != nil {
		h.cartError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": presentCart(cart, canViewCost(c))})
}

// CancelCart godoc
// @Summary Cancel (discard) a held cart session
// @Tags cart
// @Produce json
// @Security BearerAuth
// @Param id path int true "Cart ID"
// @Success 200 {object} map[string]interface{}
// @Router /pos/cart/{id}/cancel [post]
func (h *Handler) CancelCart(c *gin.Context) {
	cartID, ok := h.cartParamID(c, "id")
	if !ok {
		return
	}

	cashierID, ok := h.cartCashierID(c)
	if !ok {
		return
	}

	cart, err := h.svc.CancelCart(c.Request.Context(), cartID, cashierID)
	if err != nil {
		h.cartError(c, err)
		return
	}

	if h.auditSvc != nil {
		ctx := c.Request.Context()
		actorID := middleware.UserIDFromContext(ctx)
		_ = h.auditSvc.CreateAuditLog(ctx, &audit.Log{
			UserID:      actorID,
			Username:    middleware.UsernameFromContext(ctx),
			Role:        middleware.RoleFromContext(ctx),
			Action:      "cancel_cart",
			EntityType:  "cart",
			EntityID:    &cartID,
			IPAddress:   middleware.IPAddressFromContext(ctx),
			UserAgent:   middleware.UserAgentFromContext(ctx),
			Description: fmt.Sprintf("Cancelled held cart %d", cartID),
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": presentCart(cart, canViewCost(c))})
}

// CheckoutCart godoc
// @Summary Checkout a cart, converting it into a completed sale
// @Tags cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Cart ID"
// @Param request body map[string]interface{} true "Payments payload"
// @Success 201 {object} map[string]interface{}
// @Router /pos/cart/{id}/checkout [post]
func (h *Handler) CheckoutCart(c *gin.Context) {
	cartID, ok := h.cartParamID(c, "id")
	if !ok {
		return
	}

	type checkoutReq struct {
		Payments []CreatePaymentRequest `json:"payments"`
	}
	var req checkoutReq
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, err.Error())
		return
	}

	ctx := c.Request.Context()
	cashierID, ok := h.cartCashierID(c)
	if !ok {
		return
	}
	sale, err := h.svc.CheckoutCart(ctx, cartID, req.Payments, cashierID)
	if err != nil {
		h.cartError(c, err)
		return
	}

	if h.auditSvc != nil {
		actorID := middleware.UserIDFromContext(ctx)
		_ = h.auditSvc.CreateAuditLog(ctx, &audit.Log{
			UserID:      actorID,
			Username:    middleware.UsernameFromContext(ctx),
			Role:        middleware.RoleFromContext(ctx),
			Action:      "checkout",
			EntityType:  "cart",
			EntityID:    &cartID,
			NewValues:   shared.ToJSONMap(sale),
			IPAddress:   middleware.IPAddressFromContext(ctx),
			UserAgent:   middleware.UserAgentFromContext(ctx),
			Description: fmt.Sprintf("Checked out cart %d as sale %s (total %d)", cartID, sale.InvoiceNumber, sale.TotalAmount),
		})
	}

	canViewCost := canViewCost(c)
	if detail, err := h.svc.GetSaleByID(ctx, sale.ID, shared.GetStoreID(c)); err == nil {
		c.JSON(http.StatusCreated, gin.H{"data": presentSale(detail, canViewCost)})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": presentSale(sale, canViewCost)})
}

func (h *Handler) auditCart(c *gin.Context, action, entityType, format string, args ...interface{}) {
	if h.auditSvc == nil {
		return
	}
	ctx := c.Request.Context()
	actorID := middleware.UserIDFromContext(ctx)
	_ = h.auditSvc.CreateAuditLog(ctx, &audit.Log{
		UserID:      actorID,
		Username:    middleware.UsernameFromContext(ctx),
		Role:        middleware.RoleFromContext(ctx),
		Action:      action,
		EntityType:  entityType,
		IPAddress:   middleware.IPAddressFromContext(ctx),
		UserAgent:   middleware.UserAgentFromContext(ctx),
		Description: fmt.Sprintf(format, args...),
	})
}
