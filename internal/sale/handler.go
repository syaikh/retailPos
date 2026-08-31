package sale

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/config"
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/ownership"
	"retail-pos-system/internal/permissions"
	"retail-pos-system/internal/shared"
)

type Service interface {
	CreateSale(ctx context.Context, sale *Sale, items []Item, payments []CreatePaymentRequest) error
	CreateSaleTx(ctx context.Context, tx pgx.Tx, sale *Sale, items []Item, payments []CreatePaymentRequest) error
	CreateSaleWithParkedSale(ctx context.Context, sale *Sale, items []Item, parkedSaleID *int, payments []CreatePaymentRequest, caller Caller) error
	CreateSaleWithParkedSaleTx(ctx context.Context, tx pgx.Tx, sale *Sale, items []Item, parkedSaleID *int, payments []CreatePaymentRequest, caller Caller) error
	NotifySaleCreated(ctx context.Context, sale *Sale)
	InTx(ctx context.Context, fn func(tx pgx.Tx) error) error
	GetSaleByID(ctx context.Context, id int, storeID *int) (*Sale, error)
	ListSales(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate, paymentMethods string, storeID *int, minTotal, maxTotal *int, cashierID *int, status *string) ([]Sale, int, error)
	GetSalesForExport(ctx context.Context, search, startDate, endDate, paymentMethods string, minTotal, maxTotal *int, storeID *int) ([]ExportRow, error)
	StreamSalesExportCSV(ctx context.Context, w io.Writer, search, startDate, endDate, paymentMethods string, minTotal, maxTotal *int, storeID *int) error
	GetNextInvoiceNumber(ctx context.Context) (string, error)
	GetAllPaymentMethods(ctx context.Context) ([]PaymentMethod, error)
	GetPaymentMethodByCode(ctx context.Context, code string) (*PaymentMethod, error)
	ResolveCheckoutPrices(ctx context.Context, items []ResolveItem) ([]PriceSnapshot, error)
	ParkSale(ctx context.Context, sale *Sale, items []Item, recalledSaleID *int, caller Caller) error
	RecallSale(ctx context.Context, saleID int, caller Caller) (*Sale, error)
	RecallSaleTx(ctx context.Context, tx pgx.Tx, saleID int, caller Caller) (*Sale, error)
	CancelParkedSale(ctx context.Context, saleID int, caller Caller) error
	CancelParkedSaleTx(ctx context.Context, tx pgx.Tx, saleID int, caller Caller) error
	ListParkedSales(ctx context.Context, caller Caller) ([]Sale, error)
	GetParkedSaleByID(ctx context.Context, saleID int, caller Caller) (*Sale, error)

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
	CheckoutCartTx(ctx context.Context, tx pgx.Tx, cartID int, payments []CreatePaymentRequest, legacyPaymentMethod string, cashierID int) (*Sale, error)

	SetCartConfig(cfg CartConfig)
	SetPriceStore(ps ProductPriceGetter)
	SetPriceResolver(r PriceResolver)
	SetStockDeducer(sd StockDeducer)
	SetConsignmentCheckout(cc ConsignmentCheckout)
	SetShiftTotalUpdater(st ShiftTotalUpdater)
}

type Handler struct {
	svc      Service
	auditSvc audit.TxCreator
}

func NewHandler(svc Service, auditSvc audit.TxCreator) *Handler {
	return &Handler{svc: svc, auditSvc: auditSvc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(permissions.Code) gin.HandlerFunc) {
	r.POST("/sales", auth, perm(permissions.SaleCreate), h.CreateSale)
	r.GET("/sales", auth, perm(permissions.SaleView), h.GetSalesHistory)
	r.GET("/sales/lookup", auth, perm(permissions.SaleLookup), h.GetSalesLookup)
	r.GET("/sales/lookup/:id", auth, perm(permissions.SaleDetail), h.GetSaleLookupDetail)
	r.GET("/sales/export", auth, perm(permissions.ReportView), h.ExportSales)
	r.GET("/sales/:id", auth, perm(permissions.SaleView), h.GetSaleByID)
	r.GET("/payment-methods/:code", auth, h.GetPaymentMethodByCode)

	r.POST("/sales/parked", auth, perm(permissions.SalePark), h.ParkSale)
	r.GET("/sales/parked", auth, perm(permissions.SalePark), h.ListParkedSales)
	r.GET("/sales/parked/:id", auth, perm(permissions.SalePark), h.GetParkedSaleByID)
	r.POST("/sales/parked/:id/recall", auth, perm(permissions.SalePark), h.RecallParkedSale)
	r.POST("/sales/parked/:id/complete", auth, perm(permissions.SalePark), h.CompleteParkedSale)
	r.DELETE("/sales/parked/:id", auth, perm(permissions.SalePark), h.CancelParkedSale)
}

// callerFromContext builds the P2-6 Caller from the authenticated request. The
// gin context values are set by the auth middleware; the request context values
// are the audit-path fallback for tests that only set one side.
func callerFromContext(c *gin.Context) Caller {
	var userID int
	if v, exists := c.Get("userID"); exists {
		if id, ok := v.(int); ok {
			userID = id
		}
	}
	role := middleware.RoleFromContext(c.Request.Context())
	if role == "" {
		if v, exists := c.Get("role"); exists {
			role, _ = v.(string)
		}
	}
	return Caller{UserID: userID, Role: role, StoreID: shared.GetStoreID(c)}
}

// buildItemsFromSnapshots converts server-resolved price snapshots into sale
// items plus their line totals. Shared by CreateSale, ParkSale and
// CompleteParkedSale so every path stays server-authoritative.
func buildItemsFromSnapshots(snapshots []PriceSnapshot, resolveItems []ResolveItem) ([]Item, int, int, error) {
	items := make([]Item, 0, len(snapshots))
	var subtotal, taxTotal int
	for i, snap := range snapshots {
		if snap.ProductID != resolveItems[i].ProductID {
			return nil, 0, 0, fmt.Errorf("failed to resolve price for product %d", resolveItems[i].ProductID)
		}
		quantity := resolveItems[i].Quantity
		lineSubtotal, dpp, lineTax := computeLineTotals(quantity, snap.UnitPrice, snap.TaxRate)
		item := Item{
			ProductID:         snap.ProductID,
			ProductName:       snap.ProductName,
			Quantity:          quantity,
			UnitPrice:         snap.UnitPrice,
			Subtotal:          lineSubtotal,
			DPPAmount:         dpp,
			TaxAmount:         lineTax,
			Type:              stringPtr(string(snap.Type)),
			Cost:              snap.Cost,
			TaxClassID:        snap.TaxClassID,
			TaxRate:           snap.TaxRate,
			SnapshotCreatedAt: snap.SnapshotAt.In(shared.JakartaLocation()).Format(time.RFC3339),
		}
		if snap.Rule != nil {
			ruleID := snap.Rule.ID
			ruleName := snap.Rule.Name
			ruleType := string(snap.Rule.Type)
			item.PricingRuleID = &ruleID
			item.PricingRuleName = &ruleName
			item.PricingRuleType = &ruleType
		}
		if snap.OriginalPrice > 0 {
			originalPrice := snap.OriginalPrice
			item.OriginalPrice = &originalPrice
		}
		items = append(items, item)
		subtotal += lineSubtotal
		taxTotal += lineTax
	}
	return items, subtotal, taxTotal, nil
}

func (h *Handler) RegisterPaymentMethodsPublicRoutes(r *gin.RouterGroup) {
	r.GET("/payment-methods", h.ListPaymentMethods)
}

// CreateSale godoc
// @Summary Create a new sale
// @Description Create a new sale with items. Prices, discounts, tax, store, and
// invoice number are all server-authoritative: unit prices are re-resolved from
// the pricing engine. Payloads carrying any pricing field (discount, tax,
// subtotal, total_amount, per-item unit_price/subtotal) or store_id/invoice
// number are rejected with 400 rather than silently corrected.
// @Tags sales
// @Accept json
// @Produce json
// @Param request body map[string]interface{} true "Sale payload"
// @Security BearerAuth
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Router /sales [post]
func (h *Handler) CreateSale(c *gin.Context) {
	type createSaleItem struct {
		ProductID int `json:"product_id"`
		Quantity  int `json:"quantity"`
		UnitPrice int `json:"unit_price"`
		Subtotal  int `json:"subtotal"`
	}
	type createPaymentReq struct {
		PaymentMethodCode string `json:"payment_method_code" binding:"required"`
		Amount            int    `json:"amount" binding:"required"`
		ReferenceNumber   string `json:"reference_number"`
	}
	type createSaleReq struct {
		CustomerID      *int               `json:"customer_id"`
		CustomerGroupID *int               `json:"customer_group_id"`
		ShiftID         *int               `json:"shift_id"`
		StoreID         *int               `json:"store_id"`
		Items           []createSaleItem   `json:"items"`
		Payments        []createPaymentReq `json:"payments"`
		PaymentMethod   string             `json:"payment_method"`
		InvoiceNumber   string             `json:"invoice_number"`
		Discount        int                `json:"discount"`
		Tax             int                `json:"tax"`
		Subtotal        int                `json:"subtotal"`
		TotalAmount     int                `json:"total_amount"`
		ParkedSaleID    *int               `json:"parked_sale_id"`
	}

	var req createSaleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, err.Error())
		return
	}

	ctx := c.Request.Context()
	userID, exists := c.Get("userID")
	if !exists {
		shared.JSONError(c, http.StatusUnauthorized, shared.ErrUnauthorized, "user not authenticated")
		return
	}
	cashierID, ok := userID.(int)
	if !ok {
		shared.JSONError(c, http.StatusUnauthorized, shared.ErrUnauthorized, "invalid user ID in context")
		return
	}
	storeIDPtr := shared.GetStoreID(c)

	if len(req.Items) == 0 {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "items is required")
		return
	}

	// Invoice numbers are generated server-side; client-supplied values are
	// rejected so a caller can never forge or collide invoice numbers.
	if req.InvoiceNumber != "" {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "invoice number is generated by the server")
		return
	}

	// Prices are server-authoritative; previously-accepted pricing fields are
	// rejected rather than silently ignored so a stale client can never submit
	// a price or discount that is then dropped at full price.
	if req.Discount != 0 {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "discount is not accepted")
		return
	}
	if req.Tax != 0 {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "tax is not accepted")
		return
	}
	if req.StoreID != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "store_id is not accepted")
		return
	}
	if req.Subtotal != 0 {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "subtotal is not accepted")
		return
	}
	if req.TotalAmount != 0 {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "total_amount is not accepted")
		return
	}
	for _, item := range req.Items {
		if item.UnitPrice != 0 {
			shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest,
				fmt.Sprintf("unit_price is not accepted for product %d", item.ProductID))
			return
		}
		if item.Subtotal != 0 {
			shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest,
				fmt.Sprintf("subtotal is not accepted for product %d", item.ProductID))
			return
		}
	}

	// Server-authoritative pricing: every unit price is re-resolved from the
	// pricing engine, which is the single source of truth (mirroring the cart
	// path). Legacy pricing fields were already rejected above.
	resolveItems := make([]ResolveItem, 0, len(req.Items))
	for _, item := range req.Items {
		if item.Quantity <= 0 {
			shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest,
				fmt.Sprintf("invalid quantity %d for product %d", item.Quantity, item.ProductID))
			return
		}
		resolveItems = append(resolveItems, ResolveItem{
			ProductID:       item.ProductID,
			Quantity:        item.Quantity,
			CustomerGroupID: req.CustomerGroupID,
			StoreID:         storeIDPtr,
		})
	}

	snapshots, err := h.svc.ResolveCheckoutPrices(ctx, resolveItems)
	if err != nil {
		if errors.Is(err, ErrCheckoutProductNotFound) {
			shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, err.Error())
			return
		}
		shared.JSONError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to resolve product prices")
		return
	}

	if len(snapshots) != len(resolveItems) {
		shared.InternalError(c, errors.New("price resolver returned unexpected number of prices"))
		return
	}

	// Invoice numbers are generated server-side only after the request has
	// passed validation and price resolution, so a failed request does not burn
	// a sequence value.
	invoiceNumber, err := h.svc.GetNextInvoiceNumber(ctx)
	if err != nil {
		shared.JSONError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to generate invoice number")
		return
	}

	items, subtotal, taxTotal, err := buildItemsFromSnapshots(snapshots, resolveItems)
	if err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, err.Error())
		return
	}

	// Build payments array: prefer new `payments` field, fall back to `payment_method` for backward compat
	var payments []CreatePaymentRequest
	if len(req.Payments) > 0 {
		payments = make([]CreatePaymentRequest, len(req.Payments))
		for i, p := range req.Payments {
			payments[i] = CreatePaymentRequest(p)
		}
	} else if req.PaymentMethod != "" {
		payments = []CreatePaymentRequest{
			{PaymentMethodCode: req.PaymentMethod, Amount: subtotal},
		}
	} else {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "payments or payment_method is required")
		return
	}

	sale := &Sale{
		InvoiceNumber: invoiceNumber,
		CashierID:     cashierID,
		ShiftID:       req.ShiftID,
		StoreID:       storeIDPtr,
		CustomerID:    req.CustomerID,
		Subtotal:      subtotal,
		Discount:      0,
		Tax:           taxTotal,
		TotalAmount:   subtotal,
		Status:        "completed",
	}

	if h.auditSvc != nil {
		if err := h.svc.InTx(ctx, func(tx pgx.Tx) error {
			if err := h.svc.CreateSaleWithParkedSaleTx(ctx, tx, sale, items, req.ParkedSaleID, payments, callerFromContext(c)); err != nil {
				return err
			}
			return h.auditCreateSaleTx(ctx, tx, sale)
		}); err != nil {
			h.respondSaleCreateError(c, err)
			return
		}
		h.svc.NotifySaleCreated(ctx, sale)
	} else {
		if err := h.svc.CreateSaleWithParkedSale(ctx, sale, items, req.ParkedSaleID, payments, callerFromContext(c)); err != nil {
			h.respondSaleCreateError(c, err)
			return
		}
	}

	if detail, err := h.svc.GetSaleByID(ctx, sale.ID, storeIDPtr); err == nil {
		c.JSON(http.StatusCreated, gin.H{"data": detail})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": sale})
}

// respondSaleCreateError maps a sale-creation domain error to the correct HTTP
// response. It is shared by the atomic and non-atomic create paths.
func (h *Handler) respondSaleCreateError(c *gin.Context, err error) {
	if errors.Is(err, ErrPermissionDenied) {
		shared.JSONError(c, http.StatusForbidden, shared.ErrForbidden, "manager can only complete a recalled parked sale")
		return
	}
	if errors.Is(err, ErrInsufficientStock) {
		shared.JSONError(c, http.StatusConflict, shared.ErrConflict, "insufficient stock")
		return
	}
	if errors.Is(err, ErrParkedSaleNotRecalled) {
		shared.JSONError(c, http.StatusConflict, shared.ErrConflict, "parked sale already checked out or cancelled")
		return
	}
	if errors.Is(err, shared.ErrShiftNotOpen) {
		shared.JSONError(c, http.StatusConflict, shared.ErrConflict, "shift is closed or no longer exists")
		return
	}
	if errors.Is(err, ErrPaymentOverTenderNonCash) || errors.Is(err, ErrPaymentTotalMismatch) || errors.Is(err, ErrDuplicatePaymentMethod) ||
		errors.Is(err, ErrPaymentMethodInactive) || errors.Is(err, ErrPaymentReferenceRequired) ||
		errors.Is(err, ErrZeroPaymentAmount) || errors.Is(err, ErrInvalidPaymentMethod) ||
		errors.Is(err, ErrMaxPaymentsExceeded) || errors.Is(err, ErrMultipleCashPayments) {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, err.Error())
		return
	}
	shared.InternalError(c, err)
}

// respondCompleteParkedSaleError maps a parked-sale completion domain error to
// the correct HTTP response. It is shared by the atomic (audited) and
// non-atomic completion paths. Unlike the create path, it also surfaces
// ErrSaleNotFound (the route's parked_sale_id may reference a missing sale).
func (h *Handler) respondCompleteParkedSaleError(c *gin.Context, err error) {
	if errors.Is(err, ErrPermissionDenied) {
		shared.JSONError(c, http.StatusForbidden, shared.ErrForbidden, "manager can only complete a recalled parked sale")
		return
	}
	if errors.Is(err, ErrSaleNotFound) {
		shared.JSONError(c, http.StatusNotFound, shared.ErrNotFound, "parked sale not found")
		return
	}
	if errors.Is(err, ErrParkedSaleNotRecalled) {
		shared.JSONError(c, http.StatusConflict, shared.ErrConflict, "parked sale already checked out or cancelled")
		return
	}
	if errors.Is(err, ErrInsufficientStock) {
		shared.JSONError(c, http.StatusConflict, shared.ErrConflict, "insufficient stock")
		return
	}
	if errors.Is(err, shared.ErrShiftNotOpen) {
		shared.JSONError(c, http.StatusConflict, shared.ErrConflict, "shift is closed or no longer exists")
		return
	}
	if errors.Is(err, ErrPaymentOverTenderNonCash) || errors.Is(err, ErrPaymentTotalMismatch) || errors.Is(err, ErrDuplicatePaymentMethod) ||
		errors.Is(err, ErrPaymentMethodInactive) || errors.Is(err, ErrPaymentReferenceRequired) ||
		errors.Is(err, ErrZeroPaymentAmount) || errors.Is(err, ErrInvalidPaymentMethod) ||
		errors.Is(err, ErrMaxPaymentsExceeded) || errors.Is(err, ErrMultipleCashPayments) {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, err.Error())
		return
	}
	shared.InternalError(c, err)
}
// new_values: it serializes the sale and strips customer-identifying fields
// (customer_name) so the audit trail does not persist identifiable customer
// data.
func scrubSaleAuditPayload(sale interface{}) interface{} {
	m := shared.ToJSONMap(sale)
	if m != nil {
		shared.ScrubPII(m)
	}
	return m
}

// auditCreateSaleTx writes the "create" (sale) and per-payment "payment.created"
// audit rows inside an existing transaction, so the sale and its audit trail are
// atomic.
func (h *Handler) auditCreateSaleTx(ctx context.Context, tx pgx.Tx, sale *Sale) error {
	actorID := middleware.UserIDFromContext(ctx)
	if err := h.auditSvc.CreateAuditLogTx(ctx, tx, &audit.Log{
		UserID:      actorID,
		Username:    middleware.UsernameFromContext(ctx),
		Role:        middleware.RoleFromContext(ctx),
		Action:      "create",
		EntityType:  "sale",
		EntityID:    &sale.ID,
		NewValues:   scrubSaleAuditPayload(sale),
		IPAddress:   middleware.IPAddressFromContext(ctx),
		UserAgent:   middleware.UserAgentFromContext(ctx),
		Description: fmt.Sprintf("Created sale %s with total %d", sale.InvoiceNumber, sale.TotalAmount),
		StoreID:     middleware.StoreIDFromContext(ctx),
	}); err != nil {
		return err
	}
	return h.auditSalePaymentsTx(ctx, tx, actorID, sale)
}

// auditSalePaymentsTx writes one "payment.created" audit row per payment inside
// an existing transaction.
func (h *Handler) auditSalePaymentsTx(ctx context.Context, tx pgx.Tx, actorID *int, sale *Sale) error {
	for i := range sale.Payments {
		p := sale.Payments[i]
		if err := h.auditSvc.CreateAuditLogTx(ctx, tx, &audit.Log{
			UserID:      actorID,
			Username:    middleware.UsernameFromContext(ctx),
			Role:        middleware.RoleFromContext(ctx),
			Action:      "payment.created",
			EntityType:  "payment",
			EntityID:    &p.ID,
			NewValues:   shared.ToJSONMap(map[string]interface{}{"sale_id": sale.ID, "payment_method": p.PaymentMethodCode, "amount": p.Amount, "reference_number": p.ReferenceNumber}),
			IPAddress:   middleware.IPAddressFromContext(ctx),
			UserAgent:   middleware.UserAgentFromContext(ctx),
			Description: fmt.Sprintf("Recorded %s payment of %d for sale %s", p.PaymentMethodCode, p.Amount, sale.InvoiceNumber),
			StoreID:     middleware.StoreIDFromContext(ctx),
		}); err != nil {
			return err
		}
	}
	return nil
}

// auditSaleActionTx writes a single sale audit row (e.g. recall_sale,
// complete_parked_sale) inside an existing transaction.
func (h *Handler) auditSaleActionTx(ctx context.Context, tx pgx.Tx, caller Caller, sale *Sale, action, description string) error {
	return h.auditSvc.CreateAuditLogTx(ctx, tx, &audit.Log{
		UserID:      middleware.UserIDFromContext(ctx),
		Username:    middleware.UsernameFromContext(ctx),
		Role:        caller.Role,
		Action:      action,
		EntityType:  "sale",
		EntityID:    &sale.ID,
		NewValues:   scrubSaleAuditPayload(sale),
		IPAddress:   middleware.IPAddressFromContext(ctx),
		UserAgent:   middleware.UserAgentFromContext(ctx),
		Description: description,
		StoreID:     middleware.StoreIDFromContext(ctx),
	})
}

// auditCancelSaleTx writes the "cancel" audit row for a parked sale inside an
// existing transaction.
func (h *Handler) auditCancelSaleTx(ctx context.Context, tx pgx.Tx, id int) error {
	return h.auditSvc.CreateAuditLogTx(ctx, tx, &audit.Log{
		UserID:      middleware.UserIDFromContext(ctx),
		Username:    middleware.UsernameFromContext(ctx),
		Role:        middleware.RoleFromContext(ctx),
		Action:      "cancel",
		EntityType:  "sale",
		EntityID:    &id,
		IPAddress:   middleware.IPAddressFromContext(ctx),
		UserAgent:   middleware.UserAgentFromContext(ctx),
		Description: fmt.Sprintf("Cancelled parked sale %d", id),
		StoreID:     middleware.StoreIDFromContext(ctx),
	})
}

// GetSalesHistory godoc
// @Summary Get sales history
// @Description Get paginated list of sales with optional filters
// @Tags sales
// @Accept json
// @Produce json
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Param search query string false "Search by invoice number or customer name"
// @Param payment_methods query string false "Filter by payment methods (comma-separated codes)"
// @Param min_total query int false "Minimum total amount"
// @Param max_total query int false "Maximum total amount"
// @Param sort_by query string false "Sort field (created_at, total_amount, invoice_number, payment_method, status)" default(created_at)
// @Param sort_dir query string false "Sort direction (ASC or DESC)" default(DESC)
// @Param start_date query string false "Start date (YYYY-MM-DD, Jakarta time)"
// @Param end_date query string false "End date (YYYY-MM-DD, Jakarta time)"
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /sales [get]
func (h *Handler) GetSalesHistory(c *gin.Context) {
	ctx := c.Request.Context()

	limit, offset := shared.ParsePaginationParams(c.Query("limit"), c.Query("offset"))

	search := c.Query("search")
	paymentMethods := c.Query("payment_methods")

	var minTotal, maxTotal *int
	if v := c.Query("min_total"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n >= 0 && n <= 50000000 {
			minTotal = &n
		} else {
			shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "min_total must be between 0 and 50,000,000")
			return
		}
	}
	if v := c.Query("max_total"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n >= 0 && n <= 50000000 {
			maxTotal = &n
		} else {
			shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "max_total must be between 0 and 50,000,000")
			return
		}
	}
	if minTotal != nil && maxTotal != nil && *minTotal > *maxTotal {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "min_total cannot exceed max_total")
		return
	}

	sortBy := c.DefaultQuery("sort_by", "created_at")
	allowedSortBy := map[string]bool{"created_at": true, "total_amount": true, "invoice_number": true, "payment_method": true, "status": true}
	if !allowedSortBy[sortBy] {
		sortBy = "created_at"
	}
	sortDir := c.DefaultQuery("sort_dir", "DESC")
	allowedSortDir := map[string]bool{"ASC": true, "DESC": true}
	if !allowedSortDir[sortDir] {
		sortDir = "DESC"
	}

	tz := config.Load().Timezone
	now := time.Now().In(tz)
	endDate := now.Format("2006-01-02")
	startDate := now.AddDate(0, 0, -30).Format("2006-01-02")
	if sd := c.Query("start_date"); sd != "" {
		startDate = sd
	}
	if ed := c.Query("end_date"); ed != "" {
		endDate = ed
	}

	storeIDPtr := shared.GetStoreID(c)

	var cashierID *int
	if v := c.Query("cashier_id"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cashierID = &n
		}
	}

	// Row-level scope: callers without report.view are clamped to their own
	// sales regardless of any requested cashier_id filter.
	scope := ownership.Resolve(
		middleware.GetUserID(c),
		ownership.CanAccessAll(middleware.GetPermissions(c), permissions.ReportView),
		cashierID,
	)
	if ownID, restricted := scope.OwnID(); restricted {
		cashierID = &ownID
	}

	sales, total, err := h.svc.ListSales(ctx, limit, offset, search, sortBy, sortDir, startDate, endDate, paymentMethods, storeIDPtr, minTotal, maxTotal, cashierID, nil)
	if err != nil {
		shared.InternalError(c, err)
		return
	}

	shared.JSONPaginated(c, sales, total, limit, offset)
}

// GetSaleByID godoc
// @Summary Get a sale by ID
// @Description Get sale details including items by sale ID
// @Tags sales
// @Accept json
// @Produce json
// @Param id path int true "Sale ID"
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /sales/{id} [get]
func (h *Handler) GetSaleByID(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "invalid sale id")
		return
	}

	storeIDPtr := shared.GetStoreID(c)

	sale, err := h.svc.GetSaleByID(ctx, id, storeIDPtr)
	if err != nil {
		if errors.Is(err, ErrSaleNotFound) {
			shared.JSONError(c, http.StatusNotFound, shared.ErrNotFound, err.Error())
			return
		}
		shared.InternalError(c, err)
		return
	}

	// Ownership gate: without report.view a caller may only read its own
	// sales. A foreign sale is reported as 404 so its existence is not leaked.
	scope := ownership.Resolve(
		middleware.GetUserID(c),
		ownership.CanAccessAll(middleware.GetPermissions(c), permissions.ReportView),
		nil,
	)
	if !scope.CanAccess(sale.CashierID) {
		shared.JSONError(c, http.StatusNotFound, shared.ErrNotFound, "sale not found")
		return
	}

	shared.JSONSuccess(c, presentSale(sale, canViewCost(c)))
}

// GetSalesLookup godoc
// @Summary Cross-cashier transaction lookup (redacted summary)
// @Description Search transactions across all cashiers. Unlike /sales (which
// is scoped to the caller's own sales via sale.view), this endpoint is
// available to holders of sale.lookup and returns a redacted summary only
// (invoice, cashier, time, total, status) — no items, cost, customer, or
// payment details. Enables a cashier to find a co-worker's transaction for
// returns/receipt reprints without exposing sensitive data.
// @Tags sales
// @Accept json
// @Produce json
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Param search query string false "Search by invoice number, receipt number, or customer name"
// @Param payment_methods query string false "Filter by payment methods (comma-separated codes)"
// @Param min_total query int false "Minimum total amount"
// @Param max_total query int false "Maximum total amount"
// @Param sort_by query string false "Sort field (created_at, total_amount, invoice_number, payment_method, status)" default(created_at)
// @Param sort_dir query string false "Sort direction (ASC or DESC)" default(DESC)
// @Param start_date query string false "Start date (YYYY-MM-DD, Jakarta time)"
// @Param end_date query string false "End date (YYYY-MM-DD, Jakarta time)"
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /sales/lookup [get]
func (h *Handler) GetSalesLookup(c *gin.Context) {
	ctx := c.Request.Context()

	limit, offset := shared.ParsePaginationParams(c.Query("limit"), c.Query("offset"))

	search := c.Query("search")
	paymentMethods := c.Query("payment_methods")

	var minTotal, maxTotal *int
	if v := c.Query("min_total"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 && n <= 50000000 {
			minTotal = &n
		}
	}
	if v := c.Query("max_total"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 && n <= 50000000 {
			maxTotal = &n
		}
	}
	if minTotal != nil && maxTotal != nil && *minTotal > *maxTotal {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "min_total cannot exceed max_total")
		return
	}

	sortBy := c.DefaultQuery("sort_by", "created_at")
	allowedSortBy := map[string]bool{"created_at": true, "total_amount": true, "invoice_number": true, "payment_method": true, "status": true}
	if !allowedSortBy[sortBy] {
		sortBy = "created_at"
	}
	sortDir := c.DefaultQuery("sort_dir", "DESC")
	allowedSortDir := map[string]bool{"ASC": true, "DESC": true}
	if !allowedSortDir[sortDir] {
		sortDir = "DESC"
	}

	tz := config.Load().Timezone
	now := time.Now().In(tz)
	endDate := now.Format("2006-01-02")
	startDate := now.AddDate(0, 0, -30).Format("2006-01-02")
	if sd := c.Query("start_date"); sd != "" {
		startDate = sd
	}
	if ed := c.Query("end_date"); ed != "" {
		endDate = ed
	}

	storeIDPtr := shared.GetStoreID(c)

	// Cross-cashier lookup: intentionally NOT clamped to the caller's own
	// sales. Access is gated by the sale.lookup permission at the route level,
	// and the response is a redacted summary (see presentSaleLookup), so the
	// callers never receive sensitive fields for other cashiers' transactions.
	// Only finalized (completed) sales are surfaced: held/discarded carts and
	// other non-completed states have no customer-service value here.
	lookupStatus := "completed"
	sales, total, err := h.svc.ListSales(ctx, limit, offset, search, sortBy, sortDir, startDate, endDate, paymentMethods, storeIDPtr, minTotal, maxTotal, nil, &lookupStatus)
	if err != nil {
		shared.InternalError(c, err)
		return
	}

	summaries := make([]any, 0, len(sales))
	for _, s := range sales {
		summaries = append(summaries, presentSaleLookup(s))
	}
	shared.JSONPaginated(c, summaries, total, limit, offset)
}

// GetSaleLookupDetail godoc
// @Summary Cross-cashier transaction detail (redacted itemized, for reprint)
// @Description Fetch a single transaction's itemized detail for receipt reprint.
// Unlike /sales/:id (owner-scoped via sale.view), this endpoint is cross-cashier
// and available to holders of sale.detail. It returns a redacted payload: item
// lines (no cost/margin), payments without reference, and no customer PII — so a
// cashier can reprint a co-worker's receipt without exposing sensitive data.
// @Tags sales
// @Param id path int true "Sale ID"
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /sales/lookup/{id} [get]
func (h *Handler) GetSaleLookupDetail(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "invalid sale id")
		return
	}

	storeIDPtr := shared.GetStoreID(c)

	sale, err := h.svc.GetSaleByID(ctx, id, storeIDPtr)
	if err != nil {
		if errors.Is(err, ErrSaleNotFound) {
			shared.JSONError(c, http.StatusNotFound, shared.ErrNotFound, err.Error())
			return
		}
		shared.InternalError(c, err)
		return
	}

	// Cross-cashier by design: the sale.detail permission at the route level is
	// the only gate, and the response is redacted (see presentSaleLookupDetail),
	// so no ownership check is applied here (a foreign sale is a 404 only when it
	// does not exist, never leaked by existence).
	//
	// Find Transaction is scoped to completed sales (the list pins
	// status = 'completed'); the detail endpoint enforces the same boundary so a
	// holder of sale.detail cannot drill into a held/refunded/voided sale's lines
	// via a direct id guess. The payload is redacted regardless, so this is purely
	// a consistency guard with the completed-only scope.
	if sale.Status != "completed" {
		shared.JSONError(c, http.StatusNotFound, shared.ErrNotFound, "sale not found")
		return
	}
	shared.JSONSuccess(c, presentSaleLookupDetail(*sale))
}

// ExportSales godoc
// @Summary Export sales data
// @Description Export sales to CSV or XLSX file
// @Tags sales
// @Accept json
// @Produce text/csv
// @Param format query string false "Export format (csv or xlsx)" default(csv)
// @Param search query string false "Search by invoice number or customer name"
// @Param payment_methods query string false "Filter by payment methods (comma-separated codes)"
// @Param min_total query int false "Minimum total amount"
// @Param max_total query int false "Maximum total amount"
// @Param start_date query string false "Start date (YYYY-MM-DD, Jakarta time)"
// @Param end_date query string false "End date (YYYY-MM-DD, Jakarta time)"
// @Security BearerAuth
// @Success 200 {file} binary "Exported file"
// @Router /sales/export [get]
func (h *Handler) ExportSales(c *gin.Context) {
	format := c.DefaultQuery("format", "csv")
	search := c.Query("search")
	paymentMethods := c.Query("payment_methods")

	var minTotal, maxTotal *int
	if v := c.Query("min_total"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			minTotal = &n
		}
	}
	if v := c.Query("max_total"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			maxTotal = &n
		}
	}

	tz := config.Load().Timezone
	now := time.Now().In(tz)
	startDate := now.Format("2006-01-02")
	endDate := now.Format("2006-01-02")
	if sd := c.Query("start_date"); sd != "" {
		startDate = sd
	}
	if ed := c.Query("end_date"); ed != "" {
		endDate = ed
	}

	ctx := c.Request.Context()
	storeIDPtr := shared.GetStoreID(c)

	switch format {
	case "xlsx":
		rows, err := h.svc.GetSalesForExport(ctx, search, startDate, endDate, paymentMethods, minTotal, maxTotal, storeIDPtr)
		if err != nil {
			shared.InternalError(c, err)
			return
		}
		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Header("Content-Disposition", "attachment; filename=sales_export.xlsx")
		if err := WriteXLSX(rows, c.Writer); err != nil {
			slog.Warn("failed to write xlsx", "error", err)
		}
	default:
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename=sales_export.csv")
		if err := h.svc.StreamSalesExportCSV(ctx, c.Writer, search, startDate, endDate, paymentMethods, minTotal, maxTotal, storeIDPtr); err != nil {
			slog.Warn("failed to stream csv", "error", err)
		}
	}
}

func (h *Handler) ListPaymentMethods(c *gin.Context) {
	ctx := c.Request.Context()
	methods, err := h.svc.GetAllPaymentMethods(ctx)
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": methods})
}

func (h *Handler) GetPaymentMethodByCode(c *gin.Context) {
	ctx := c.Request.Context()
	code := c.Param("code")
	method, err := h.svc.GetPaymentMethodByCode(ctx, code)
	if err != nil {
		shared.JSONError(c, http.StatusNotFound, shared.ErrNotFound, "payment method not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": method})
}

func (h *Handler) ParkSale(c *gin.Context) {
	type parkSaleItem struct {
		ProductID int `json:"product_id"`
		Quantity  int `json:"quantity"`
		UnitPrice int `json:"unit_price"`
		Subtotal  int `json:"subtotal"`
	}
	type parkSaleReq struct {
		InvoiceNumber   string         `json:"invoice_number"`
		Items           []parkSaleItem `json:"items" binding:"required"`
		PaymentMethod   string         `json:"payment_method"`
		CustomerGroupID *int           `json:"customer_group_id"`
		CustomerID      *int           `json:"customer_id"`
		HoldNote        string         `json:"hold_note"`
		RecalledSaleID  *int           `json:"recalled_sale_id"`
	}

	var req parkSaleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, err.Error())
		return
	}
	if len(req.Items) == 0 {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "items is required")
		return
	}

	ctx := c.Request.Context()
	userID, exists := c.Get("userID")
	if !exists {
		shared.JSONError(c, http.StatusUnauthorized, shared.ErrUnauthorized, "user not authenticated")
		return
	}
	cashierID, ok := userID.(int)
	if !ok {
		shared.JSONError(c, http.StatusUnauthorized, shared.ErrUnauthorized, "invalid user ID in context")
		return
	}

	// Legacy client-priced fields are rejected so the server remains the
	// single source of truth for parked totals too (mirrors CreateSale).
	for _, item := range req.Items {
		if item.UnitPrice != 0 {
			shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest,
				fmt.Sprintf("unit_price is not accepted for product %d", item.ProductID))
			return
		}
		if item.Subtotal != 0 {
			shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest,
				fmt.Sprintf("subtotal is not accepted for product %d", item.ProductID))
			return
		}
	}

	storeIDPtr := shared.GetStoreID(c)
	resolveItems := make([]ResolveItem, 0, len(req.Items))
	for _, item := range req.Items {
		if item.Quantity <= 0 {
			shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest,
				fmt.Sprintf("invalid quantity %d for product %d", item.Quantity, item.ProductID))
			return
		}
		resolveItems = append(resolveItems, ResolveItem{
			ProductID:       item.ProductID,
			Quantity:        item.Quantity,
			CustomerGroupID: req.CustomerGroupID,
			StoreID:         storeIDPtr,
		})
	}

	snapshots, err := h.svc.ResolveCheckoutPrices(ctx, resolveItems)
	if err != nil {
		if errors.Is(err, ErrCheckoutProductNotFound) {
			shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, err.Error())
			return
		}
		shared.JSONError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to resolve product prices")
		return
	}

	if len(snapshots) != len(resolveItems) {
		shared.InternalError(c, errors.New("price resolver returned unexpected number of prices"))
		return
	}

	// Invoice numbers are generated only after resolution passes, so a failed
	// park request does not burn a sequence value.
	invoiceNumber := req.InvoiceNumber
	if invoiceNumber == "" {
		invoiceNumber, err = h.svc.GetNextInvoiceNumber(ctx)
		if err != nil {
			shared.JSONError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to generate invoice number")
			return
		}
	}

	items, subtotal, _, err := buildItemsFromSnapshots(snapshots, resolveItems)
	if err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, err.Error())
		return
	}

	sale := &Sale{
		InvoiceNumber: invoiceNumber,
		CashierID:     cashierID,
		StoreID:       storeIDPtr,
		CustomerID:    req.CustomerID,
		HoldNote:      req.HoldNote,
		Subtotal:      subtotal,
		TotalAmount:   subtotal,
		PaymentMethod: req.PaymentMethod,
		Status:        "parked",
	}

	if err := h.svc.ParkSale(ctx, sale, items, req.RecalledSaleID, callerFromContext(c)); err != nil {
		shared.InternalError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": sale})
}

func (h *Handler) ListParkedSales(c *gin.Context) {
	ctx := c.Request.Context()
	userID, exists := c.Get("userID")
	if !exists {
		shared.JSONError(c, http.StatusUnauthorized, shared.ErrUnauthorized, "user not authenticated")
		return
	}
	if _, ok := userID.(int); !ok {
		shared.JSONError(c, http.StatusUnauthorized, shared.ErrUnauthorized, "invalid user ID in context")
		return
	}

	caller := callerFromContext(c)
	sales, err := h.svc.ListParkedSales(ctx, caller)
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": sales})
}

func (h *Handler) GetParkedSaleByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "invalid sale ID")
		return
	}

	ctx := c.Request.Context()
	userID, exists := c.Get("userID")
	if !exists {
		shared.JSONError(c, http.StatusUnauthorized, shared.ErrUnauthorized, "user not authenticated")
		return
	}
	if _, ok := userID.(int); !ok {
		shared.JSONError(c, http.StatusUnauthorized, shared.ErrUnauthorized, "invalid user ID in context")
		return
	}

	sale, err := h.svc.GetParkedSaleByID(ctx, id, callerFromContext(c))
	if err != nil {
		if errors.Is(err, ErrSaleNotFound) {
			shared.JSONError(c, http.StatusNotFound, shared.ErrNotFound, "sale not found")
			return
		}
		shared.InternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": sale})
}

func (h *Handler) RecallParkedSale(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "invalid sale ID")
		return
	}

	ctx := c.Request.Context()
	caller := callerFromContext(c)
	var sale *Sale
	if h.auditSvc != nil {
		err = h.svc.InTx(ctx, func(tx pgx.Tx) error {
			s, e := h.svc.RecallSaleTx(ctx, tx, id, caller)
			if e != nil {
				return e
			}
			sale = s
			return h.auditSaleActionTx(ctx, tx, caller, sale, "recall_sale",
				fmt.Sprintf("Recalled parked sale %s (cashier %d)", sale.InvoiceNumber, sale.CashierID))
		})
	} else {
		sale, err = h.svc.RecallSale(ctx, id, caller)
	}
	if err != nil {
		if errors.Is(err, ErrSaleNotFound) {
			shared.JSONError(c, http.StatusNotFound, shared.ErrNotFound, "sale not found")
			return
		}
		shared.InternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": sale})
}

func (h *Handler) CancelParkedSale(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "invalid sale ID")
		return
	}

	ctx := c.Request.Context()
	caller := callerFromContext(c)
	if h.auditSvc != nil {
		err = h.svc.InTx(ctx, func(tx pgx.Tx) error {
			if e := h.svc.CancelParkedSaleTx(ctx, tx, id, caller); e != nil {
				return e
			}
			return h.auditCancelSaleTx(ctx, tx, id)
		})
	} else {
		err = h.svc.CancelParkedSale(ctx, id, caller)
	}
	if err != nil {
		if errors.Is(err, ErrSaleNotFound) {
			shared.JSONError(c, http.StatusNotFound, shared.ErrNotFound, "sale not found")
			return
		}
		if errors.Is(err, ErrPermissionDenied) {
			shared.JSONError(c, http.StatusForbidden, shared.ErrForbidden, "manager cannot cancel a parked sale")
			return
		}
		shared.InternalError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// CompleteParkedSale godoc
// @Summary Complete a recalled parked sale
// @Description Manager scoped-completion path (P2-6 D4): completes the recalled
// parked sale identified by the route and creates a completed sale from
// server-resolved prices. The parked_sale_id is bound to the URL so a manager
// can never complete a sale without one (no blanket sale.create for Manager).
// Pricing fields in the payload are rejected exactly like POST /sales.
// @Tags sales
// @Accept json
// @Produce json
// @Param id path int true "Parked sale ID"
// @Param request body map[string]interface{} true "Sale payload (no pricing fields)"
// @Security BearerAuth
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Router /sales/parked/:id/complete [post]
//
//gocritic:ignore unnamedResult
func (h *Handler) CompleteParkedSale(c *gin.Context) {
	type completeItem struct {
		ProductID int `json:"product_id"`
		Quantity  int `json:"quantity"`
		UnitPrice int `json:"unit_price"`
		Subtotal  int `json:"subtotal"`
	}
	type completePaymentReq struct {
		PaymentMethodCode string `json:"payment_method_code" binding:"required"`
		Amount            int    `json:"amount" binding:"required"`
		ReferenceNumber   string `json:"reference_number"`
	}
	type completeReq struct {
		CustomerID      *int                 `json:"customer_id"`
		CustomerGroupID *int                 `json:"customer_group_id"`
		ShiftID         *int                 `json:"shift_id"`
		Items           []completeItem       `json:"items" binding:"required"`
		Payments        []completePaymentReq `json:"payments"`
		PaymentMethod   string               `json:"payment_method"`
		InvoiceNumber   string               `json:"invoice_number"`
		Discount        int                  `json:"discount"`
		Tax             int                  `json:"tax"`
		Subtotal        int                  `json:"subtotal"`
		TotalAmount     int                  `json:"total_amount"`
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "invalid sale ID")
		return
	}

	var req completeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, err.Error())
		return
	}

	ctx := c.Request.Context()
	userID, exists := c.Get("userID")
	if !exists {
		shared.JSONError(c, http.StatusUnauthorized, shared.ErrUnauthorized, "user not authenticated")
		return
	}
	cashierID, ok := userID.(int)
	if !ok {
		shared.JSONError(c, http.StatusUnauthorized, shared.ErrUnauthorized, "invalid user ID in context")
		return
	}
	storeIDPtr := shared.GetStoreID(c)

	if len(req.Items) == 0 {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "items is required")
		return
	}

	// Server-authoritative pricing mirrors POST /sales: client pricing and
	// server-generated identity fields are rejected, never silently corrected.
	if req.InvoiceNumber != "" {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "invoice number is generated by the server")
		return
	}
	if req.Discount != 0 || req.Tax != 0 || req.Subtotal != 0 || req.TotalAmount != 0 {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "discount, tax, subtotal and total_amount are not accepted")
		return
	}
	for _, item := range req.Items {
		if item.UnitPrice != 0 {
			shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest,
				fmt.Sprintf("unit_price is not accepted for product %d", item.ProductID))
			return
		}
		if item.Subtotal != 0 {
			shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest,
				fmt.Sprintf("subtotal is not accepted for product %d", item.ProductID))
			return
		}
	}

	resolveItems := make([]ResolveItem, 0, len(req.Items))
	for _, item := range req.Items {
		if item.Quantity <= 0 {
			shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest,
				fmt.Sprintf("invalid quantity %d for product %d", item.Quantity, item.ProductID))
			return
		}
		resolveItems = append(resolveItems, ResolveItem{
			ProductID:       item.ProductID,
			Quantity:        item.Quantity,
			CustomerGroupID: req.CustomerGroupID,
			StoreID:         storeIDPtr,
		})
	}

	snapshots, err := h.svc.ResolveCheckoutPrices(ctx, resolveItems)
	if err != nil {
		if errors.Is(err, ErrCheckoutProductNotFound) {
			shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, err.Error())
			return
		}
		shared.JSONError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to resolve product prices")
		return
	}
	if len(snapshots) != len(resolveItems) {
		shared.InternalError(c, errors.New("price resolver returned unexpected number of prices"))
		return
	}

	items, subtotal, taxTotal, err := buildItemsFromSnapshots(snapshots, resolveItems)
	if err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, err.Error())
		return
	}

	invoiceNumber, err := h.svc.GetNextInvoiceNumber(ctx)
	if err != nil {
		shared.JSONError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to generate invoice number")
		return
	}

	payments := make([]CreatePaymentRequest, len(req.Payments))
	for i, p := range req.Payments {
		payments[i] = CreatePaymentRequest(p)
	}

	sale := &Sale{
		InvoiceNumber: invoiceNumber,
		CashierID:     cashierID,
		ShiftID:       req.ShiftID,
		StoreID:       storeIDPtr,
		CustomerID:    req.CustomerID,
		Subtotal:      subtotal,
		Discount:      0,
		Tax:           taxTotal,
		TotalAmount:   subtotal,
		Status:        "completed",
	}

	caller := callerFromContext(c)
	if h.auditSvc != nil && caller.IsManager() {
		if err := h.svc.InTx(ctx, func(tx pgx.Tx) error {
			if err := h.svc.CreateSaleWithParkedSaleTx(ctx, tx, sale, items, &id, payments, caller); err != nil {
				return err
			}
			return h.auditSaleActionTx(ctx, tx, caller, sale, "complete_parked_sale",
				fmt.Sprintf("Manager completed recalled parked sale %d as sale %s (total %d)", id, sale.InvoiceNumber, sale.TotalAmount))
		}); err != nil {
			h.respondCompleteParkedSaleError(c, err)
			return
		}
		h.svc.NotifySaleCreated(ctx, sale)
	} else {
		if err := h.svc.CreateSaleWithParkedSale(ctx, sale, items, &id, payments, caller); err != nil {
			h.respondCompleteParkedSaleError(c, err)
			return
		}
	}

	if detail, err := h.svc.GetSaleByID(ctx, sale.ID, storeIDPtr); err == nil {
		c.JSON(http.StatusCreated, gin.H{"data": detail})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": sale})
}
