package sale

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/config"
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/shared"
)

type SaleService interface {
	CreateSale(ctx context.Context, sale *Sale, items []SaleItem) error
	GetSaleByID(ctx context.Context, id int, storeID *int) (*Sale, error)
	ListSales(ctx context.Context, limit, offset int, search, sortBy, sortDir, startDate, endDate, paymentMethods string, storeID *int, minTotal, maxTotal *int, cashierID *int) ([]Sale, int, error)
	GetSalesForExport(ctx context.Context, search, startDate, endDate, paymentMethods string, minTotal, maxTotal *int, storeID *int) ([]SaleExportRow, error)
	GetNextInvoiceNumber(ctx context.Context) (string, error)
	GetAllPaymentMethods(ctx context.Context) ([]PaymentMethod, error)
	GetPaymentMethodByCode(ctx context.Context, code string) (*PaymentMethod, error)
}

type Handler struct {
	svc      SaleService
	auditSvc audit.AuditCreator
}

func NewHandler(svc SaleService, auditSvc audit.AuditCreator) *Handler {
	return &Handler{svc: svc, auditSvc: auditSvc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(string) gin.HandlerFunc) {
	r.POST("/sales", auth, perm("sale:create"), h.CreateSale)
	r.GET("/sales", auth, perm("sale:read"), h.GetSalesHistory)
	r.GET("/sales/export", auth, perm("report:read"), h.ExportSales)
	r.GET("/sales/:id", auth, perm("sale:read"), h.GetSaleByID)
	r.GET("/payment-methods/:code", auth, h.GetPaymentMethodByCode)
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
	type createSaleReq struct {
		InvoiceNumber string           `json:"invoice_number"`
		CustomerID    *int             `json:"customer_id"`
		ShiftID       *int             `json:"shift_id"`
		StoreID       *int             `json:"store_id"`
		Items         []createSaleItem `json:"items" binding:"required"`
		PaymentMethod string           `json:"payment_method"`
		Discount      int              `json:"discount"`
	}

	var req createSaleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.JSONError(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx := c.Request.Context()
	userID, exists := c.Get("userID")
	if !exists {
		shared.JSONError(c, http.StatusUnauthorized, "user not authenticated")
		return
	}
	cashierID, ok := userID.(int)
	if !ok {
		shared.JSONError(c, http.StatusUnauthorized, "invalid user ID in context")
		return
	}
	storeIDPtr := shared.GetStoreID(c)

	invoiceNumber := req.InvoiceNumber
	if invoiceNumber == "" {
		var err error
		invoiceNumber, err = h.svc.GetNextInvoiceNumber(ctx)
		if err != nil {
			shared.JSONError(c, http.StatusInternalServerError, "failed to generate invoice number")
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "discount must not be negative"})
		return
	}
	if req.Discount > subtotal {
		c.JSON(http.StatusBadRequest, gin.H{"error": "discount must not exceed subtotal"})
		return
	}

	if req.PaymentMethod != "" {
		pm, err := h.svc.GetPaymentMethodByCode(ctx, req.PaymentMethod)
		if err != nil || pm == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payment method"})
			return
		}
	}

	sale := &Sale{
		InvoiceNumber: invoiceNumber,
		CashierID:     cashierID,
		ShiftID:       req.ShiftID,
		StoreID:       storeIDPtr,
		CustomerID:    req.CustomerID,
		Subtotal:      subtotal,
		Discount:      req.Discount,
		Tax:           0,
		TotalAmount:   subtotal - req.Discount,
		PaymentMethod: req.PaymentMethod,
		Status:        "completed",
	}

	if err := h.svc.CreateSale(ctx, sale, items); err != nil {
		if errors.Is(err, ErrInsufficientStock) {
			shared.JSONError(c, http.StatusConflict, "insufficient stock")
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
			shared.JSONError(c, http.StatusBadRequest, "min_total must be between 0 and 50,000,000")
			return
		}
	}
	if v := c.Query("max_total"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n >= 0 && n <= 50000000 {
			maxTotal = &n
		} else {
			shared.JSONError(c, http.StatusBadRequest, "max_total must be between 0 and 50,000,000")
			return
		}
	}
	if minTotal != nil && maxTotal != nil && *minTotal > *maxTotal {
		shared.JSONError(c, http.StatusBadRequest, "min_total cannot exceed max_total")
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
		shared.JSONError(c, http.StatusBadRequest, "invalid sale id")
		return
	}

	storeIDPtr := shared.GetStoreID(c)

	sale, err := h.svc.GetSaleByID(ctx, id, storeIDPtr)
	if err != nil {
		if errors.Is(err, ErrSaleNotFound) {
			shared.JSONError(c, http.StatusNotFound, err.Error())
			return
		}
		shared.InternalError(c, err)
		return
	}

	shared.JSONSuccess(c, sale)
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

	rows, err := h.svc.GetSalesForExport(ctx, search, startDate, endDate, paymentMethods, minTotal, maxTotal, storeIDPtr)
	if err != nil {
		shared.InternalError(c, err)
		return
	}

	switch format {
	case "xlsx":
		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Header("Content-Disposition", "attachment; filename=sales_export.xlsx")
		if err := WriteXLSX(rows, c.Writer); err != nil {
			log.Printf("failed to write xlsx: %v", err)
		}
	default:
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename=sales_export.csv")
		if err := WriteCSV(rows, c.Writer); err != nil {
			log.Printf("failed to write csv: %v", err)
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
		c.JSON(http.StatusNotFound, gin.H{"error": "payment method not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": method})
}
