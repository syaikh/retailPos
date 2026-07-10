package sale

import (
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"

	"retail-pos-system/internal/config"
	"retail-pos-system/internal/shared"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
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

func (h *Handler) CreateSale(c *gin.Context) {
	type createSaleItem struct {
		ProductID int `json:"product_id"`
		Quantity  int `json:"quantity"`
		Subtotal  int `json:"subtotal"`
	}
	type createSaleReq struct {
		InvoiceNumber string          `json:"invoice_number"`
		CustomerID    *int            `json:"customer_id"`
		StoreID       *int            `json:"store_id"`
		Items         []createSaleItem `json:"items" binding:"required"`
		PaymentMethod string          `json:"payment_method"`
		Discount      int             `json:"discount"`
	}

	var req createSaleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.JSONError(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx := c.Request.Context()
	userID, _ := c.Get("userID")
	cashierID, _ := userID.(int)
	storeID, _ := c.Get("storeID")
	storeIDPtr, _ := storeID.(*int)

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

	sale := &Sale{
		InvoiceNumber: invoiceNumber,
		CashierID:     cashierID,
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

	c.JSON(http.StatusCreated, gin.H{"data": sale})
}

func (h *Handler) GetSalesHistory(c *gin.Context) {
	ctx := c.Request.Context()

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if offset < 0 {
		offset = 0
	}

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

	storeID, _ := c.Get("storeID")
	storeIDPtr, _ := storeID.(*int)

	sales, total, err := h.svc.ListSales(ctx, limit, offset, search, sortBy, sortDir, startDate, endDate, paymentMethods, storeIDPtr, minTotal, maxTotal)
	if err != nil {
		shared.InternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": sales, "total": total})
}

func (h *Handler) GetSaleByID(c *gin.Context) {
	ctx := c.Request.Context()

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		shared.JSONError(c, http.StatusBadRequest, "invalid sale id")
		return
	}

	storeID, _ := c.Get("storeID")
	storeIDPtr, _ := storeID.(*int)

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
	rows, err := h.svc.GetSalesForExport(ctx, search, startDate, endDate, paymentMethods, minTotal, maxTotal)
	if err != nil {
		shared.InternalError(c, err)
		return
	}

	switch format {
	case "xlsx":
		h.exportXLSX(c, rows)
	default:
		h.exportCSV(c, rows)
	}
}

func (h *Handler) exportCSV(c *gin.Context, rows []SaleExportRow) {
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=sales_export.csv")

	w := csv.NewWriter(c.Writer)
	shared.WriteCSVRow(w, []string{"Invoice Number", "Date", "Customer", "Items", "Payment Method", "Total Amount"})
	for _, row := range rows {
		shared.WriteCSVRow(w, []string{
			row.InvoiceNumber,
			row.CreatedAt,
			row.CustomerName,
			strconv.Itoa(row.ItemCount),
			row.PaymentMethod,
			fmt.Sprintf("%d", row.TotalAmount),
		})
	}
	w.Flush()
}

func (h *Handler) exportXLSX(c *gin.Context, rows []SaleExportRow) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := f.GetSheetName(0)
	f.SetCellValue(sheet, "A1", "Invoice Number")
	f.SetCellValue(sheet, "B1", "Date")
	f.SetCellValue(sheet, "C1", "Customer")
	f.SetCellValue(sheet, "D1", "Items")
	f.SetCellValue(sheet, "E1", "Payment Method")
	f.SetCellValue(sheet, "F1", "Total Amount")

	for i, row := range rows {
		r := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", r), row.InvoiceNumber)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", r), row.CreatedAt)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", r), row.CustomerName)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", r), row.ItemCount)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", r), row.PaymentMethod)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", r), row.TotalAmount)
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=sales_export.xlsx")

	if err := f.Write(c.Writer); err != nil {
		log.Printf("failed to write xlsx: %v", err)
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
