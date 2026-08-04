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

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/config"
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/permissions"
	"retail-pos-system/internal/shared"
)

type SaleService interface {
	CreateSale(ctx context.Context, sale *Sale, items []SaleItem, payments []CreatePaymentRequest) error
	CreateSaleWithParkedSale(ctx context.Context, sale *Sale, items []SaleItem, parkedSaleID *int, payments []CreatePaymentRequest) error
	GetSaleByID(ctx context.Context, id int, storeID *int) (*Sale, error)
	ListSales(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate, paymentMethods string, storeID *int, minTotal, maxTotal *int, cashierID *int) ([]Sale, int, error)
	GetSalesForExport(ctx context.Context, search, startDate, endDate, paymentMethods string, minTotal, maxTotal *int, storeID *int) ([]SaleExportRow, error)
	StreamSalesExportCSV(ctx context.Context, w io.Writer, search, startDate, endDate, paymentMethods string, minTotal, maxTotal *int, storeID *int) error
	GetNextInvoiceNumber(ctx context.Context) (string, error)
	GetAllPaymentMethods(ctx context.Context) ([]PaymentMethod, error)
	GetPaymentMethodByCode(ctx context.Context, code string) (*PaymentMethod, error)
	ParkSale(ctx context.Context, sale *Sale, items []SaleItem, recalledSaleID *int) error
	RecallSale(ctx context.Context, saleID int) (*Sale, error)
	CancelParkedSale(ctx context.Context, saleID int) error
	ListParkedSales(ctx context.Context, cashierID int) ([]Sale, error)
	GetParkedSaleByID(ctx context.Context, saleID int, cashierID int) (*Sale, error)

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
	CheckoutCart(ctx context.Context, cartID int, payments []CreatePaymentRequest, cashierID int) (*Sale, error)
	CheckoutCartWithPaymentMethod(ctx context.Context, cartID int, paymentMethod string, cashierID int) (*Sale, error)
}

type Handler struct {
	svc      SaleService
	auditSvc audit.AuditCreator
}

func NewHandler(svc SaleService, auditSvc audit.AuditCreator) *Handler {
	return &Handler{svc: svc, auditSvc: auditSvc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(permissions.Code) gin.HandlerFunc) {
	r.POST("/sales", auth, perm(permissions.SaleCreate), h.CreateSale)
	r.GET("/sales", auth, perm(permissions.SaleView), h.GetSalesHistory)
	r.GET("/sales/export", auth, perm(permissions.ReportView), h.ExportSales)
	r.GET("/sales/:id", auth, perm(permissions.SaleView), h.GetSaleByID)
	r.GET("/payment-methods/:code", auth, h.GetPaymentMethodByCode)

	r.POST("/sales/parked", auth, perm(permissions.SalePark), h.ParkSale)
	r.GET("/sales/parked", auth, perm(permissions.SalePark), h.ListParkedSales)
	r.GET("/sales/parked/:id", auth, perm(permissions.SalePark), h.GetParkedSaleByID)
	r.POST("/sales/parked/:id/recall", auth, perm(permissions.SalePark), h.RecallParkedSale)
	r.DELETE("/sales/parked/:id", auth, perm(permissions.SalePark), h.CancelParkedSale)
}

func (h *Handler) RegisterPaymentMethodsPublicRoutes(r *gin.RouterGroup) {
	r.GET("/payment-methods", h.ListPaymentMethods)
}

// CreateSale godoc
// @Summary Create a new sale
// @Description Create a new sale with items. Invoice number is auto-generated if omitted.
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
		Subtotal  int `json:"subtotal"`
	}
	type createPaymentReq struct {
		PaymentMethodCode string `json:"payment_method_code" binding:"required"`
		Amount            int    `json:"amount" binding:"required"`
		ReferenceNumber   string `json:"reference_number"`
	}
	type createSaleReq struct {
		InvoiceNumber string             `json:"invoice_number"`
		CustomerID    *int               `json:"customer_id"`
		ShiftID       *int               `json:"shift_id"`
		StoreID       *int               `json:"store_id"`
		Items         []createSaleItem   `json:"items"`
		Payments      []createPaymentReq `json:"payments"`
		PaymentMethod string             `json:"payment_method"`
		Discount      int                `json:"discount"`
		Tax           int                `json:"tax"`
		ParkedSaleID  *int               `json:"parked_sale_id"`
		CartSessionID *int               `json:"cart_session_id"`
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

	// RT-16 case 3: when cart_session_id is provided, POST /sales behaves like
	// CheckoutCart — the sale is built from the immutable cart snapshots.
	if req.CartSessionID != nil {
		// Backward compat: if only the legacy `payment_method` field is given,
		// CheckoutCartWithPaymentMethod derives the amount from the recomputed
		// sale total inside the checkout transaction (no lock-free pre-read).
		if len(req.Payments) == 0 && req.PaymentMethod != "" {
			h.createSaleFromCartWithPaymentMethod(c, ctx, *req.CartSessionID, req.PaymentMethod, cashierID, storeIDPtr)
			return
		}
		payments := make([]CreatePaymentRequest, len(req.Payments))
		for i, p := range req.Payments {
			payments[i] = CreatePaymentRequest{
				PaymentMethodCode: p.PaymentMethodCode,
				Amount:            p.Amount,
				ReferenceNumber:   p.ReferenceNumber,
			}
		}
		h.createSaleFromCart(c, ctx, *req.CartSessionID, payments, cashierID, storeIDPtr)
		return
	}

	if len(req.Items) == 0 {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "items is required")
		return
	}

	invoiceNumber := req.InvoiceNumber
	if invoiceNumber == "" {
		var err error
		invoiceNumber, err = h.svc.GetNextInvoiceNumber(ctx)
		if err != nil {
			shared.JSONError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to generate invoice number")
			return
		}
	}

	var subtotal int
	items := make([]SaleItem, 0, len(req.Items))
	for _, item := range req.Items {
		unitPrice := 0
		if item.Quantity > 0 {
			unitPrice = item.Subtotal / item.Quantity
		}
		items = append(items, SaleItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: unitPrice,
			Subtotal:  item.Subtotal,
			DPPAmount: item.Subtotal,
			TaxAmount: 0,
		})
		subtotal += item.Subtotal
	}

	if req.Discount < 0 {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "discount must not be negative")
		return
	}
	if req.Discount > subtotal {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "discount must not exceed subtotal")
		return
	}

	// Build payments array: prefer new `payments` field, fall back to `payment_method` for backward compat
	var payments []CreatePaymentRequest
	if len(req.Payments) > 0 {
		payments = make([]CreatePaymentRequest, len(req.Payments))
		for i, p := range req.Payments {
			payments[i] = CreatePaymentRequest{
				PaymentMethodCode: p.PaymentMethodCode,
				Amount:            p.Amount,
				ReferenceNumber:   p.ReferenceNumber,
			}
		}
	} else if req.PaymentMethod != "" {
		totalAmount := subtotal - req.Discount
		payments = []CreatePaymentRequest{
			{PaymentMethodCode: req.PaymentMethod, Amount: totalAmount},
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
		Discount:      req.Discount,
		Tax:           req.Tax,
		TotalAmount:   subtotal - req.Discount,
		Status:        "completed",
	}

	if err := h.svc.CreateSaleWithParkedSale(ctx, sale, items, req.ParkedSaleID, payments); err != nil {
		if errors.Is(err, ErrInsufficientStock) {
			shared.JSONError(c, http.StatusConflict, shared.ErrConflict, "insufficient stock")
			return
		}
		if errors.Is(err, ErrParkedSaleNotRecalled) {
			shared.JSONError(c, http.StatusConflict, shared.ErrConflict, "parked sale already checked out or cancelled")
			return
		}
		if errors.Is(err, ErrPaymentTotalMismatch) || errors.Is(err, ErrDuplicatePaymentMethod) ||
			errors.Is(err, ErrPaymentMethodInactive) || errors.Is(err, ErrPaymentReferenceRequired) ||
			errors.Is(err, ErrZeroPaymentAmount) || errors.Is(err, ErrInvalidPaymentMethod) ||
			errors.Is(err, ErrMaxPaymentsExceeded) || errors.Is(err, ErrMultipleCashPayments) {
			shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, err.Error())
			return
		}
		shared.InternalError(c, err)
		return
	}

	if h.auditSvc != nil {
		actorID := middleware.UserIDFromContext(c.Request.Context())
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      actorID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "create",
			EntityType:  "sale",
			EntityID:    &sale.ID,
			NewValues:   shared.ToJSONMap(sale),
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Created sale %s with total %d", sale.InvoiceNumber, sale.TotalAmount),
		})
	}

	if detail, err := h.svc.GetSaleByID(ctx, sale.ID, storeIDPtr); err == nil {
		c.JSON(http.StatusCreated, gin.H{"data": detail})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": sale})
}

// createSaleFromCart handles POST /sales with cart_session_id (RT-16 case 3).
// It delegates to CheckoutCart so the sale is built from immutable snapshots,
// deducts stock, validates payments, and marks the cart as checked out.
func (h *Handler) createSaleFromCart(c *gin.Context, ctx context.Context, cartID int, payments []CreatePaymentRequest, cashierID int, storeIDPtr *int) {
	sale, err := h.svc.CheckoutCart(ctx, cartID, payments, cashierID)
	h.respondSaleFromCart(c, ctx, sale, err, cartID, storeIDPtr)
}

func (h *Handler) createSaleFromCartWithPaymentMethod(c *gin.Context, ctx context.Context, cartID int, paymentMethod string, cashierID int, storeIDPtr *int) {
	sale, err := h.svc.CheckoutCartWithPaymentMethod(ctx, cartID, paymentMethod, cashierID)
	h.respondSaleFromCart(c, ctx, sale, err, cartID, storeIDPtr)
}

func (h *Handler) respondSaleFromCart(c *gin.Context, ctx context.Context, sale *Sale, err error, cartID int, storeIDPtr *int) {
	if err != nil {
		h.cartError(c, err)
		return
	}

	if h.auditSvc != nil {
		actorID := middleware.UserIDFromContext(ctx)
		_ = h.auditSvc.CreateAuditLog(ctx, &audit.AuditLog{
			UserID:      actorID,
			Username:    middleware.UsernameFromContext(ctx),
			Role:        middleware.RoleFromContext(ctx),
			Action:      "create",
			EntityType:  "sale",
			EntityID:    &sale.ID,
			NewValues:   shared.ToJSONMap(sale),
			IPAddress:   middleware.IPAddressFromContext(ctx),
			UserAgent:   middleware.UserAgentFromContext(ctx),
			Description: fmt.Sprintf("Checked out cart %d as sale %s (total %d)", cartID, sale.InvoiceNumber, sale.TotalAmount),
		})
	}

	if detail, err := h.svc.GetSaleByID(ctx, sale.ID, storeIDPtr); err == nil {
		c.JSON(http.StatusCreated, gin.H{"data": detail})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": sale})
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

	sales, total, err := h.svc.ListSales(ctx, limit, offset, search, sortBy, sortDir, startDate, endDate, paymentMethods, storeIDPtr, minTotal, maxTotal, cashierID)
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

	shared.JSONSuccess(c, presentSale(sale, canViewCost(c)))
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
	type createSaleItem struct {
		ProductID int `json:"product_id"`
		Quantity  int `json:"quantity"`
		Subtotal  int `json:"subtotal"`
	}
	type parkSaleReq struct {
		InvoiceNumber  string           `json:"invoice_number"`
		Items          []createSaleItem `json:"items" binding:"required"`
		PaymentMethod  string           `json:"payment_method"`
		RecalledSaleID *int             `json:"recalled_sale_id"`
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

	invoiceNumber := req.InvoiceNumber
	if invoiceNumber == "" {
		var err error
		invoiceNumber, err = h.svc.GetNextInvoiceNumber(ctx)
		if err != nil {
			shared.JSONError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to generate invoice number")
			return
		}
	}

	var subtotal int
	items := make([]SaleItem, 0, len(req.Items))
	for _, item := range req.Items {
		unitPrice := 0
		if item.Quantity > 0 {
			unitPrice = item.Subtotal / item.Quantity
		}
		items = append(items, SaleItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: unitPrice,
			Subtotal:  item.Subtotal,
		})
		subtotal += item.Subtotal
	}

	sale := &Sale{
		InvoiceNumber: invoiceNumber,
		CashierID:     cashierID,
		StoreID:       shared.GetStoreID(c),
		Subtotal:      subtotal,
		TotalAmount:   subtotal,
		PaymentMethod: req.PaymentMethod,
		Status:        "parked",
	}

	if err := h.svc.ParkSale(ctx, sale, items, req.RecalledSaleID); err != nil {
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
	cashierID, ok := userID.(int)
	if !ok {
		shared.JSONError(c, http.StatusUnauthorized, shared.ErrUnauthorized, "invalid user ID in context")
		return
	}

	sales, err := h.svc.ListParkedSales(ctx, cashierID)
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
	cashierID, ok := userID.(int)
	if !ok {
		shared.JSONError(c, http.StatusUnauthorized, shared.ErrUnauthorized, "invalid user ID in context")
		return
	}

	sale, err := h.svc.GetParkedSaleByID(ctx, id, cashierID)
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
	sale, err := h.svc.RecallSale(ctx, id)
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
	if err := h.svc.CancelParkedSale(ctx, id); err != nil {
		if errors.Is(err, ErrSaleNotFound) {
			shared.JSONError(c, http.StatusNotFound, shared.ErrNotFound, "sale not found")
			return
		}
		shared.InternalError(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
