package handler

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"net/mail"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"retail-pos-system/internal/auth"
	"retail-pos-system/internal/config"
	"retail-pos-system/internal/domain"
	"retail-pos-system/internal/repository"
	"retail-pos-system/pkg/websocket"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

func getCtx(c *gin.Context) context.Context {
	return c.Request.Context()
}

type Handler struct {
	authRepo       repository.UserRepository
	roleRepo       repository.RoleRepository
	productRepo    repository.ProductRepository
	paymentRepo    repository.PaymentMethodRepository
	saleRepo       repository.SaleRepository
	customerRepo   repository.CustomerRepository
	authService    *auth.AuthService
	hub            *websocket.Hub
	auditRepo      repository.AuditLogRepository
	categoryRepo   repository.CategoryRepository
}

func NewHandler(
	authRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
	productRepo repository.ProductRepository,
	paymentRepo repository.PaymentMethodRepository,
	saleRepo repository.SaleRepository,
	customerRepo repository.CustomerRepository,
	authService *auth.AuthService,
	hub *websocket.Hub,
	auditRepo repository.AuditLogRepository,
	categoryRepo repository.CategoryRepository,
) *Handler {
	return &Handler{
		authRepo:     authRepo,
		roleRepo:     roleRepo,
		productRepo:  productRepo,
		paymentRepo:  paymentRepo,
		saleRepo:     saleRepo,
		customerRepo: customerRepo,
		authService:  authService,
		hub:          hub,
		auditRepo:    auditRepo,
		categoryRepo: categoryRepo,
	}
}

func (h *Handler) userRole(c *gin.Context) string {
	role, exists := c.Get("role")
	if !exists {
		return ""
	}
	if roleStr, ok := role.(string); ok {
		return strings.ToLower(strings.TrimSpace(roleStr))
	}
	return ""
}

func (h *Handler) hasPermission(c *gin.Context, permission string) bool {
	perms, exists := c.Get("permissions")
	if !exists {
		return false
	}
	permissions, ok := perms.([]string)
	if !ok {
		return false
	}
	for _, perm := range permissions {
		if perm == permission {
			return true
		}
	}
	return false
}

func (h *Handler) canManageProduct(c *gin.Context, permission string) bool {
	role := h.userRole(c)
	if role == "superadmin" || role == "admin" || role == "staff" {
		return true
	}
	return h.hasPermission(c, permission)
}

func (h *Handler) canManageCategory(c *gin.Context, permission string) bool {
	role := h.userRole(c)
	if role == "superadmin" || role == "admin" {
		return true
	}
	return h.hasPermission(c, permission)
}

func (h *Handler) normalizeProductBarcode(product *domain.Product) {
	if product.Barcode != nil {
		trimmed := strings.TrimSpace(*product.Barcode)
		if trimmed == "" {
			product.Barcode = nil
		} else {
			product.Barcode = &trimmed
		}
	}
}

func (h *Handler) resolveProductCategory(c *gin.Context, product *domain.Product) error {
	if product.CategoryName == nil || strings.TrimSpace(*product.CategoryName) == "" {
		return fmt.Errorf("category is required")
	}

	categoryName := strings.TrimSpace(*product.CategoryName)
	categoryID, err := h.productRepo.GetCategoryIDByName(getCtx(c), categoryName)
	if err != nil {
		return err
	}
	product.CategoryID = &categoryID
	return nil
}

func (h *Handler) validateProductPayload(product *domain.Product) error {
	if strings.TrimSpace(product.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(product.SKU) == "" {
		return fmt.Errorf("sku is required")
	}
	if product.CategoryName == nil && product.CategoryID == nil {
		return fmt.Errorf("category is required")
	}
	if product.Price <= 0 {
		return fmt.Errorf("price must be greater than 0")
	}
	if product.Stock < 0 {
		return fmt.Errorf("stock must not be negative")
	}
	return nil
}

// Auth Handlers
func (h *Handler) Login(c *gin.Context) {
	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	resp, err := h.authService.Login(getCtx(c), req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Set refresh token as HttpOnly cookie (Secure: false in dev, true in prod)
	isProd := os.Getenv("ENV") == "production"
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    resp.RefreshToken,
		MaxAge:   7 * 24 * 3600,
		Path:     "/",
		HttpOnly: true,
		Secure:   isProd,
		SameSite: http.SameSiteLaxMode,
	})
	c.Set("userID", resp.User.ID)
	c.Set("username", resp.User.Username)
	c.Set("role", resp.User.Role.Name)
	h.logAudit(c, "login", "auth", resp.User.ID, nil, nil)

	c.JSON(http.StatusOK, gin.H{"access_token": resp.AccessToken, "refresh_token": resp.RefreshToken, "user": resp.User})
}

func (h *Handler) Logout(c *gin.Context) {
	userID := getUserID(c)
	token, _ := c.Cookie("refresh_token")
	if token != "" && userID > 0 {
		h.authService.Logout(getCtx(c), userID, token)
	}
	// Clear cookie
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
		Secure:   os.Getenv("ENV") == "production",
		SameSite: http.SameSiteLaxMode,
	})
	h.logAudit(c, "logout", "auth", userID, nil, nil)
	c.JSON(http.StatusOK, gin.H{"status": "logged out"})
}

func (h *Handler) RefreshToken(c *gin.Context) {
	refreshToken := c.GetHeader("X-Refresh-Token")
	if refreshToken == "" {
		refreshToken, _ = c.Cookie("refresh_token")
	}
	if refreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh token required"})
		return
	}
	newAccessToken, err := h.authService.RefreshToken(getCtx(c), refreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"access_token": newAccessToken})
}

func (h *Handler) ValidateSession(c *gin.Context) {
	userID := getUserID(c)
	role, _ := c.Get("role")
	permissions, _ := c.Get("permissions")
	storeID, _ := c.Get("storeID")

	user, err := h.authRepo.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}
	user.Password = ""

	var perms []string
	if p, ok := permissions.([]string); ok {
		perms = p
	}

	c.JSON(http.StatusOK, gin.H{"valid": true, "user": user, "role": role, "permissions": perms, "store_id": storeID})
}

// Product Handlers
func (h *Handler) GetProducts(c *gin.Context) {
	limit := 50
	if l := c.Query("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 200 {
			limit = val
		}
	}
	offset := 0
	if o := c.Query("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}

	// Handle category parameter (supports multiple categories as comma-separated)
	var categoryIDs []int
	if cat := c.Query("category"); cat != "" && cat != "all" {
		catNames := strings.Split(cat, ",")
		for _, name := range catNames {
			name = strings.TrimSpace(name)
			if id, err := h.productRepo.GetCategoryIDByName(getCtx(c), name); err == nil {
				categoryIDs = append(categoryIDs, id)
			}
		}
	}

	// Handle maxStock parameter for low stock filtering
	var maxStock *int
	if ms := c.Query("maxStock"); ms != "" {
		if val, err := strconv.Atoi(ms); err == nil && val > 0 {
			maxStock = &val
		}
	}

	products, total, err := h.productRepo.GetAllProducts(getCtx(c), limit, offset, c.Query("search"), categoryIDs, "created_at", "DESC", maxStock, nil, c.Query("status"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch products"})
		return
	}
	if products == nil {
		products = []domain.Product{}
	}
	c.JSON(http.StatusOK, gin.H{"data": products, "total": total})
}

func (h *Handler) GetProductByID(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	product, err := h.productRepo.GetProductByID(getCtx(c), id, nil)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": product})
}

func (h *Handler) GetProductStockByID(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	stock, err := h.productRepo.GetStockByProductID(getCtx(c), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "stock not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": stock})
}

func (h *Handler) CreateProduct(c *gin.Context) {
	var product domain.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if !h.canManageProduct(c, "product:create") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permission"})
		return
	}

	h.normalizeProductBarcode(&product)
	if err := h.validateProductPayload(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.resolveProductCategory(c, &product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category"})
		return
	}

if product.Barcode != nil {
		deletedProduct, err := h.productRepo.GetDeletedProductByBarcode(getCtx(c), *product.Barcode, nil)
		if err == nil && deletedProduct != nil {
			old := *deletedProduct
			product.ID = deletedProduct.ID
			product.Status = "active"
			if err := h.productRepo.RestoreProduct(getCtx(c), &product); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to restore product"})
				return
			}
			h.logAudit(c, "restore", "product", product.ID, old, product)
			if h.hub != nil {
				websocket.BroadcastProductUpdate(h.hub, &product)
			}
			c.JSON(http.StatusOK, gin.H{"data": product})
			return
		}
		if err != nil && !strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check deleted product"})
			return
		}
	}

	if err := h.productRepo.CreateProduct(getCtx(c), &product); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create product: " + err.Error()})
		return
	}
	h.logAudit(c, "create", "product", product.ID, nil, product)
	if h.hub != nil {
		websocket.BroadcastProductUpdate(h.hub, &product)
	}
	c.JSON(http.StatusCreated, gin.H{"data": product})
}

func (h *Handler) UpdateProduct(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	old, _ := h.productRepo.GetProductByID(getCtx(c), id, nil)
	if old == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}

	if !h.canManageProduct(c, "product:update") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permission"})
		return
	}

	var product domain.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	product.ID = id
	h.normalizeProductBarcode(&product)
	if err := h.validateProductPayload(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.resolveProductCategory(c, &product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category"})
		return
	}
	if err := h.productRepo.UpdateProduct(getCtx(c), &product, nil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update product"})
		return
	}
	h.logAudit(c, "update", "product", product.ID, old, product)
	if h.hub != nil {
		websocket.BroadcastProductUpdate(h.hub, &product)
	}
	c.JSON(http.StatusOK, gin.H{"data": product})
}

func (h *Handler) DeleteProduct(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	old, err := h.productRepo.GetProductByID(getCtx(c), id, nil)
	if err != nil || old == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}

	role := h.userRole(c)
	if role != "superadmin" && role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permission"})
		return
	}

	if old.Stock > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product must have zero stock before deletion"})
		return
	}

	if err := h.productRepo.DeleteProduct(getCtx(c), id, nil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete product"})
		return
	}
	h.logAudit(c, "delete", "product", id, old, nil)
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// Sale Handlers
func (h *Handler) CreateSale(c *gin.Context) {
	var req domain.SaleCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	userID := getUserID(c)
	if userID == 0 {
		userID = req.CashierID
	}

	// Generate invoice number if not provided or invalid format
	invoiceNumber := req.InvoiceNumber
	if invoiceNumber == "" {
		var err error
		invoiceNumber, err = h.saleRepo.GetNextInvoiceNumber(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate invoice number"})
			return
		}
	}

	// Get storeID from context (set by AuthMiddleware)
	var storeID *int
	if sid, exists := c.Get("storeID"); exists {
		if v, ok := sid.(*int); ok {
			storeID = v
		}
	}

	// If storeID is still nil, fallback to request
	if storeID == nil {
		storeID = req.StoreID
	}

	sale := &domain.Sale{
		InvoiceNumber: invoiceNumber,
		CashierID:     userID,
		CustomerID:    req.CustomerID,
		StoreID:       storeID,
		Subtotal:      req.Subtotal,
		Discount:      req.Discount,
		Tax:           req.Tax,
		TotalAmount:   req.TotalAmount,
		PaymentMethod: req.PaymentMethod,
		Status:        "completed",
	}
	tx, err := h.saleRepo.BeginTx(getCtx(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to begin transaction"})
		return
	}
	defer tx.Rollback(getCtx(c))

	if err := h.saleRepo.CreateSale(getCtx(c), tx, sale, req.Items); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create sale: " + err.Error()})
		return
	}

	if err := tx.Commit(getCtx(c)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction: " + err.Error()})
		return
	}
	newSale, _ := h.saleRepo.GetSaleByID(getCtx(c), sale.ID)
	h.logAudit(c, "create", "sale", sale.ID, nil, newSale)
	// Broadcast sale created event
	if h.hub != nil {
		websocket.BroadcastSaleCreated(h.hub, newSale)
		// Also broadcast stock updates for sold items
		for _, item := range newSale.Items {
			if product, err := h.productRepo.GetProductByID(getCtx(c), item.ProductID, newSale.StoreID); err == nil {
				cfg := config.Load()
				websocket.BroadcastStockUpdate(h.hub, product, product.Stock <= cfg.StockCriticalThreshold)
				if product.Stock <= cfg.StockCriticalThreshold {
					websocket.BroadcastLowStockAlert(h.hub, product)
				}
			}
		}
	}

	c.JSON(http.StatusCreated, gin.H{"data": newSale})
}

func (h *Handler) GetSalesHistory(c *gin.Context) {
	limit := 50
	if l := c.Query("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 200 {
			limit = val
		}
	}
	offset := 0
	if o := c.Query("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}

	search := c.Query("search")
	paymentMethods := c.Query("paymentMethods")
	minTotal := parseIntPtr(c.Query("minTotal"))
	maxTotal := parseIntPtr(c.Query("maxTotal"))

	const maxAmountFilter = 50000000
	if minTotal != nil && (*minTotal < 0 || *minTotal > maxAmountFilter) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "minTotal must be between 0 and 50,000,000"})
		return
	}
	if maxTotal != nil && (*maxTotal < 0 || *maxTotal > maxAmountFilter) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "maxTotal must be between 0 and 50,000,000"})
		return
	}
	if minTotal != nil && maxTotal != nil && *minTotal > *maxTotal {
		c.JSON(http.StatusBadRequest, gin.H{"error": "minTotal cannot exceed maxTotal"})
		return
	}

	sortBy := c.Query("sortBy")
	sortDir := c.Query("sortDir")
	allowedSortBy := map[string]bool{"created_at": true, "total_amount": true, "invoice_number": true, "payment_method": true, "status": true}
	allowedSortDir := map[string]bool{"ASC": true, "DESC": true}
	if !allowedSortBy[sortBy] {
		sortBy = "created_at"
	}
	if !allowedSortDir[sortDir] {
		sortDir = "DESC"
	}

	cfg := config.Load()
	now := time.Now().In(cfg.Timezone)
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")
	if startDate == "" {
		startDate = now.AddDate(0, 0, -30).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = now.Format("2006-01-02")
	}

	sales, total, err := h.saleRepo.GetAllSales(getCtx(c), limit, offset, search, sortBy, sortDir, startDate, endDate, nil, paymentMethods, minTotal, maxTotal)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch sales"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": sales, "total": total})
}

func (h *Handler) ExportSales(c *gin.Context) {
	format := c.Query("format")
	search := c.Query("search")
	paymentMethods := c.Query("paymentMethods")
	minTotal := parseIntPtr(c.Query("minTotal"))
	maxTotal := parseIntPtr(c.Query("maxTotal"))

	const maxAmountFilter = 50000000
	if minTotal != nil && (*minTotal < 0 || *minTotal > maxAmountFilter) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "minTotal must be between 0 and 50,000,000"})
		return
	}
	if maxTotal != nil && (*maxTotal < 0 || *maxTotal > maxAmountFilter) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "maxTotal must be between 0 and 50,000,000"})
		return
	}
	if minTotal != nil && maxTotal != nil && *minTotal > *maxTotal {
		c.JSON(http.StatusBadRequest, gin.H{"error": "minTotal cannot exceed maxTotal"})
		return
	}

	cfg := config.Load()
	now := time.Now().In(cfg.Timezone)
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")
	if startDate == "" {
		startDate = now.AddDate(0, 0, -30).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = now.Format("2006-01-02")
	}

	rows, err := h.saleRepo.GetSalesForExport(getCtx(c), search, startDate, endDate, paymentMethods, minTotal, maxTotal)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch sales data"})
		return
	}

	filename := "transactions-" + now.Format("2006-01-02")

	switch format {
	case "xlsx":
		wb := excelize.NewFile()
		sheet := "Transactions"
		wb.SetSheetName("Sheet1", sheet)

		headers := []string{"INVOICE", "DATE", "CUSTOMER", "ITEMS", "PAYMENT", "TOTAL (RP)"}
		for i, h := range headers {
			col, _ := excelize.ColumnNumberToName(i + 1)
			wb.SetCellValue(sheet, col+"1", h)
		}
		headerStyle, _ := wb.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
			Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"7C3AED"}},
		})
		wb.SetCellStyle(sheet, "A1", "F1", headerStyle)

		for i, row := range rows {
			r := i + 2
			wb.SetCellValue(sheet, fmt.Sprintf("A%d", r), row.InvoiceNumber)
			wb.SetCellValue(sheet, fmt.Sprintf("B%d", r), row.CreatedAt)
			wb.SetCellValue(sheet, fmt.Sprintf("C%d", r), row.CustomerName)
			wb.SetCellValue(sheet, fmt.Sprintf("D%d", r), row.ItemCount)
			wb.SetCellValue(sheet, fmt.Sprintf("E%d", r), row.PaymentMethod)
			wb.SetCellValue(sheet, fmt.Sprintf("F%d", r), row.TotalAmount)
		}

		colWidths := []float64{20, 22, 25, 8, 15, 18}
		for i, w := range colWidths {
			col, _ := excelize.ColumnNumberToName(i + 1)
			wb.SetColWidth(sheet, col, col, w)
		}

		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.xlsx"`, filename))

		if err := wb.Write(c.Writer); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write xlsx"})
			return
		}

	default: // csv
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, filename))
		// BOM for Excel compatibility
		c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

		writer := csv.NewWriter(c.Writer)
		writer.Write([]string{"INVOICE", "DATE", "CUSTOMER", "ITEMS", "PAYMENT", "TOTAL (RP)"})
		for _, row := range rows {
			writer.Write([]string{
				row.InvoiceNumber,
				row.CreatedAt,
				row.CustomerName,
				strconv.Itoa(row.ItemCount),
				row.PaymentMethod,
				strconv.Itoa(row.TotalAmount),
			})
		}
		writer.Flush()
	}
}

func parseIntPtr(s string) *int {
	if s == "" {
		return nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &v
}

func (h *Handler) GetSaleByID(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	sale, err := h.saleRepo.GetSaleByID(getCtx(c), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "sale not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": sale})
}

func (h *Handler) GetDashboardStats(c *gin.Context) {
	ctx := getCtx(c)
	cfg := config.Load()

	// Use configured timezone (default Asia/Jakarta) for consistent date calculations
	now := time.Now().In(cfg.Timezone)
	today := now.Format("2006-01-02")
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")

	// Get total revenue & sales (simplified)
	sales, _, err := h.saleRepo.GetAllSales(ctx, 10000, 0, "", "", "", today, today, nil, "", nil, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch sales data"})
		return
	}
	var todaysRev int
	for _, s := range sales {
		todaysRev += s.TotalAmount
	}

	// Get yesterday's revenue for trend display
	ydaySales, _, err2 := h.saleRepo.GetAllSales(ctx, 10000, 0, "", "", "", yesterday, yesterday, nil, "", nil, nil)
	var ydayRev int
	if err2 == nil {
		for _, s := range ydaySales {
			ydayRev += s.TotalAmount
		}
	}

	_, totalProducts, err := h.productRepo.GetAllProducts(ctx, 1, 0, "", []int{}, "", "", nil, nil, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch products data"})
		return
	}

	lowStock := cfg.StockCriticalThreshold
	_, lowStockCount, err := h.productRepo.GetAllProducts(ctx, 1, 0, "", []int{}, "", "", &lowStock, nil, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch low stock data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": map[string]interface{}{
		"todays_revenue":     todaysRev,
		"yesterday_revenue":  ydayRev,
		"todays_sales":       len(sales),
		"total_products":     totalProducts,
		"low_stock_count":    lowStockCount,
	}})
}

func (h *Handler) GetLiveDashboardStats(c *gin.Context) {
	ctx := getCtx(c)
	var storeID *int
	if sid, exists := c.Get("storeID"); exists {
		if v, ok := sid.(int); ok {
			storeID = &v
		}
	}

	todaysRevenue, todaysSales, totalProducts, lowStockCount, err :=
		h.saleRepo.GetLiveDashboardStats(ctx, storeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch live dashboard stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": map[string]interface{}{
		"todays_revenue":  todaysRevenue,
		"todays_sales":    todaysSales,
		"total_products":  totalProducts,
		"low_stock_count": lowStockCount,
	}})
}

func (h *Handler) GetSalesChartData(c *gin.Context) {
	ctx := getCtx(c)
	cfg := config.Load()

	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	if startDate == "" || endDate == "" {
		// Use configured timezone (default Asia/Jakarta) for consistent date calculations
		now := time.Now().In(cfg.Timezone)
		endDate = now.Format("2006-01-02")
		startDate = now.AddDate(0, 0, -6).Format("2006-01-02")
	}

	// Determine if hourly aggregation is needed (single day = realtime/yesterday)
	isHourly := startDate == endDate

	sales, _, err := h.saleRepo.GetAllSales(ctx, 10000, 0, "", "created_at", "ASC", startDate, endDate, nil, "", nil, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch sales"})
		return
	}

	if isHourly {
		// Hourly aggregation for realtime/yesterday
		hourlyTotals := make(map[int]int)
		for _, s := range sales {
			// Bug fix: use time.RFC3339 (the actual format produced by the repo) and
			// convert to Jakarta local time before extracting the hour.
			createdTime, err := time.Parse(time.RFC3339, s.CreatedAt)
			if err != nil {
				continue
			}
			hour := createdTime.In(cfg.Timezone).Hour()
			hourlyTotals[hour] += s.TotalAmount
		}

		type HourlyDataPoint struct {
			Hour  int `json:"hour"`
			Total int `json:"total"`
		}

		// Generate all 24 hours (0-23)
		var data []HourlyDataPoint
		for hour := 0; hour < 24; hour++ {
			data = append(data, HourlyDataPoint{
				Hour:  hour,
				Total: hourlyTotals[hour],
			})
		}

		c.JSON(http.StatusOK, gin.H{"data": data})
		return
	}

	// Daily aggregation
	dailyTotals := make(map[string]int)
	for _, s := range sales {
		// Bug fix: parse with RFC3339 and convert to Jakarta local time before
		// slicing the date, so midnight-straddling sales land in the correct day.
		createdTime, err := time.Parse(time.RFC3339, s.CreatedAt)
		var dateStr string
		if err == nil {
			dateStr = createdTime.In(cfg.Timezone).Format("2006-01-02")
		} else if len(s.CreatedAt) >= 10 {
			dateStr = s.CreatedAt[:10] // fallback
		}
		dailyTotals[dateStr] += s.TotalAmount
	}

	type ChartDataPoint struct {
		Date  string `json:"date"`
		Total int    `json:"total"`
	}

	var data []ChartDataPoint
	start, _ := time.ParseInLocation("2006-01-02", startDate, cfg.Timezone)
	end, _ := time.ParseInLocation("2006-01-02", endDate, cfg.Timezone)

	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		data = append(data, ChartDataPoint{
			Date:  dateStr,
			Total: dailyTotals[dateStr],
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) GetSalesWeeklyReport(c *gin.Context) {
	ctx := getCtx(c)
	cfg := config.Load()
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	if startDate == "" || endDate == "" {
		// Use configured timezone (default Asia/Jakarta) for consistent date calculations
		now := time.Now().In(cfg.Timezone)
		endDate = now.Format("2006-01-02")
		startDate = now.AddDate(0, 0, -84).Format("2006-01-02") // 12 weeks back
	}

	sales, _, err := 	h.saleRepo.GetAllSales(ctx, 50000, 0, "", "created_at", "ASC", startDate, endDate, nil, "", nil, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch sales"})
		return
	}

	weeklyTotals := make(map[string]map[string]interface{})
	for _, s := range sales {
		// Bug fix: use time.RFC3339 and convert to Jakarta local time
		createdTime, _ := time.Parse(time.RFC3339, s.CreatedAt)
		createdTime = createdTime.In(cfg.Timezone)
		// Get week start (Monday), handling Sunday case
		weekStart := createdTime.AddDate(0, 0, -int(createdTime.Weekday()-time.Monday))
		if createdTime.Weekday() == time.Sunday {
			weekStart = createdTime.AddDate(0, 0, -6)
		}
		weekEnd := weekStart.AddDate(0, 0, 6)
		weekStartStr := weekStart.Format("2006-01-02")
		weekEndStr := weekEnd.Format("2006-01-02")

		weekKey := weekStartStr + "|" + weekEndStr
		if weeklyTotals[weekKey] == nil {
			weeklyTotals[weekKey] = map[string]interface{}{
				"week_start":  weekStartStr,
				"week_end":    weekEndStr,
				"total":       0,
				"order_count": 0,
			}
		}
		weeklyTotals[weekKey]["total"] = weeklyTotals[weekKey]["total"].(int) + s.TotalAmount
		weeklyTotals[weekKey]["order_count"] = weeklyTotals[weekKey]["order_count"].(int) + 1
	}

	type WeeklyDataPoint struct {
		WeekStart  string `json:"week_start"`
		WeekEnd    string `json:"week_end"`
		Total      int    `json:"total"`
		OrderCount int    `json:"order_count"`
	}

	var data []WeeklyDataPoint
	for _, v := range weeklyTotals {
		data = append(data, WeeklyDataPoint{
			WeekStart:  v["week_start"].(string),
			WeekEnd:    v["week_end"].(string),
			Total:      v["total"].(int),
			OrderCount: v["order_count"].(int),
		})
	}

	// Sort by week start date
	sort.Slice(data, func(i, j int) bool {
		return data[i].WeekStart < data[j].WeekStart
	})

	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) GetSalesMonthlyReport(c *gin.Context) {
	ctx := getCtx(c)
	cfg := config.Load()
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	if startDate == "" || endDate == "" {
		// Use configured timezone (default Asia/Jakarta) for consistent date calculations
		now := time.Now().In(cfg.Timezone)
		endDate = now.Format("2006-01-02")
		startDate = now.AddDate(-1, 1, 0).Format("2006-01-02") // 12 months back
	}

	sales, _, err := 	h.saleRepo.GetAllSales(ctx, 50000, 0, "", "created_at", "ASC", startDate, endDate, nil, "", nil, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch sales"})
		return
	}

	monthlyTotals := make(map[string]map[string]interface{})
	for _, s := range sales {
		// Bug fix: use time.RFC3339 and convert to Jakarta local time
		createdTime, _ := time.Parse(time.RFC3339, s.CreatedAt)
		createdTime = createdTime.In(cfg.Timezone)
		monthKey := createdTime.Format("2006-01")
		monthStart := createdTime.Format("2006-01-01")

		if monthlyTotals[monthKey] == nil {
			monthlyTotals[monthKey] = map[string]interface{}{
				"month":       monthKey,
				"month_start": monthStart,
				"total":       0,
				"order_count": 0,
			}
		}
		monthlyTotals[monthKey]["total"] = monthlyTotals[monthKey]["total"].(int) + s.TotalAmount
		monthlyTotals[monthKey]["order_count"] = monthlyTotals[monthKey]["order_count"].(int) + 1
	}

	type MonthlyDataPoint struct {
		Month      string `json:"month"`
		MonthStart string `json:"month_start"`
		Total      int    `json:"total"`
		OrderCount int    `json:"order_count"`
	}

	var data []MonthlyDataPoint
	for _, v := range monthlyTotals {
		data = append(data, MonthlyDataPoint{
			Month:      v["month"].(string),
			MonthStart: v["month_start"].(string),
			Total:      v["total"].(int),
			OrderCount: v["order_count"].(int),
		})
	}

	// Sort by month
	sort.Slice(data, func(i, j int) bool {
		return data[i].Month < data[j].Month
	})

	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) GetPeriodComparison(c *gin.Context) {
	ctx := getCtx(c)
	cfg := config.Load()

	periodType := PeriodType(c.Query("period")) // daily, weekly, monthly, yearly
	if periodType == "" {
		periodType = PeriodDaily
	}

	// Use configured timezone (default Asia/Jakarta) for consistent date calculations
	refDate := time.Now().In(cfg.Timezone)
	if dateStr := c.Query("date"); dateStr != "" {
		if parsed, err := time.ParseInLocation("2006-01-02", dateStr, cfg.Timezone); err == nil {
			if c.Query("mode") == "realtime" {
				// For realtime: use the requested date but preserve the current hour/minute/second
				// so getRealtimeRanges correctly includes the partial current hour
				refDate = time.Date(
					parsed.Year(), parsed.Month(), parsed.Day(),
					refDate.Hour(), refDate.Minute(), refDate.Second(), refDate.Nanosecond(),
					cfg.Timezone,
				)
			} else {
				refDate = parsed
			}
		}
	}

	// Calculate ranges based on mode
	var ranges PeriodRange
	switch c.Query("mode") {
	case "realtime":
		ranges = getRealtimeRanges(refDate)
	case "completed":
		ranges = GetComparisonRanges(periodType, refDate, true)
	case "30days":
		ranges = get30DaysRanges(refDate)
	default:
		ranges = GetComparisonRanges(periodType, refDate, false)
	}

  comparison, err := h.saleRepo.GetPeriodComparison(ctx,
    ranges.CurrentStart, ranges.CurrentEnd,
    ranges.PreviousStart, ranges.PreviousEnd,
  )

  if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch comparison"})
    return
  }

  c.JSON(http.StatusOK, gin.H{
    "data": comparison,
    "meta": map[string]interface{}{
      "current_period": map[string]string{
        "start": ranges.CurrentStart.Format("2006-01-02"),
        "end":   ranges.CurrentEnd.AddDate(0, 0, -1).Format("2006-01-02"),
      },
      "previous_period": map[string]string{
        "start": ranges.PreviousStart.Format("2006-01-02"),
        "end":   ranges.PreviousEnd.AddDate(0, 0, -1).Format("2006-01-02"),
      },
      "is_partial":     ranges.IsPartial,
      "days_in_period": ranges.DaysInPeriod,
    },
  })
}

// Admin Handlers
func (h *Handler) ListUsers(c *gin.Context) {
	limit := 20
	if l := c.Query("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}
	offset := 0
	if o := c.Query("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}
	sortBy := c.Query("sort")
	sortDir := c.Query("dir")
	roleID := 0
	if r := c.Query("role"); r != "" {
		if val, err := strconv.Atoi(r); err == nil && val > 0 {
			roleID = val
		}
	}
	var isActive *bool
	if s := c.Query("status"); s != "" {
		switch s {
		case "true":
			v := true
			isActive = &v
		case "false":
			v := false
			isActive = &v
		}
	}
	users, total, err := h.authRepo.GetAllUsers(getCtx(c), limit, offset, c.Query("search"), sortBy, sortDir, roleID, isActive)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch users"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users, "total": total})
}

func (h *Handler) ListRoles(c *gin.Context) {
	roles, err := h.roleRepo.GetAllRoles(getCtx(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch roles"})
		return
	}

	// Fetch permissions for each role
	type roleWithPerms struct {
		domain.Role
		Permissions []string `json:"permissions"`
	}
	var resp []roleWithPerms
	for _, r := range roles {
		perms, _ := h.roleRepo.GetRolePermissions(getCtx(c), r.ID)
		pCodes := []string{}
		for _, p := range perms {
			pCodes = append(pCodes, p.Code)
		}
		resp = append(resp, roleWithPerms{Role: r, Permissions: pCodes})
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func (h *Handler) CreateRole(c *gin.Context) {
	var role domain.Role
	if err := c.ShouldBindJSON(&role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := h.roleRepo.CreateRole(getCtx(c), &role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create role: " + err.Error()})
		return
	}
	h.logAudit(c, "create", "role", role.ID, nil, role)
	c.JSON(http.StatusCreated, gin.H{"data": role})
}

func (h *Handler) UpdateRole(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	if h.userRole(c) != "superadmin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "only superadmin can modify roles"})
		return
	}

	oldRole, err := h.roleRepo.GetRoleByID(getCtx(c), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	updated := *oldRole
	updated.Name = req.Name
	updated.Description = req.Description

	if err := h.roleRepo.UpdateRole(getCtx(c), &updated); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update role"})
		return
	}

	h.logAudit(c, "update", "role", id, oldRole, updated)
	c.JSON(http.StatusOK, gin.H{"data": updated})
}


func (h *Handler) UpdateRolePermissions(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	// Only superadmin can modify roles
	if h.userRole(c) != "superadmin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "only superadmin can modify roles"})
		return
	}

	var req struct {
		PermissionIDs []int `json:"permission_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	oldRole, err := h.roleRepo.GetRoleByID(getCtx(c), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}

  if err := h.roleRepo.UpdateRolePermissions(getCtx(c), id, req.PermissionIDs); err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update role permissions"})
    return
  }
  h.logAudit(c, "update", "role", id, oldRole, map[string]interface{}{"permission_ids": req.PermissionIDs})
  c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

func (h *Handler) DeleteRole(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	// Only superadmin can delete roles
	if h.userRole(c) != "superadmin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "only superadmin can delete roles"})
		return
	}

	// Prevent deleting system roles
	role, err := h.roleRepo.GetRoleByID(getCtx(c), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}
	if role.IsSystem {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete system role"})
		return
	}

	if err := h.roleRepo.DeleteRole(getCtx(c), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete role"})
		return
	}
	h.logAudit(c, "delete", "role", id, role, nil)
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) ListPermissions(c *gin.Context) {
	perms, err := h.roleRepo.GetAllPermissions(getCtx(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch permissions"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": perms})
}

// Audit Logs
func (h *Handler) ListAuditLogs(c *gin.Context) {
	limit := 50
	if l := c.Query("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 200 {
			limit = val
		}
	}
	offset := 0
	if o := c.Query("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}
	var userID *int
	if uid := c.Query("user_id"); uid != "" {
		if val, err := strconv.Atoi(uid); err == nil {
			userID = &val
		}
	}

	search := c.Query("search")
	action := c.Query("action")
	entityType := c.Query("entity_type")

	var startDate, endDate *time.Time
	if sd := c.Query("start_date"); sd != "" {
		if t, err := time.Parse(time.RFC3339, sd); err == nil {
			startDate = &t
		}
	}
	if ed := c.Query("end_date"); ed != "" {
		if t, err := time.Parse(time.RFC3339, ed); err == nil {
			endDate = &t
		}
	}

	logs, total, err := h.auditRepo.GetAll(getCtx(c), limit, offset, userID, search, action, entityType, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch audit logs"})
		return
	}

	if logs == nil {
		logs = []domain.AuditLog{}
	}

	for i := range logs {
		logs[i].Description = h.generateAuditDescription(&logs[i])
	}

	c.JSON(http.StatusOK, gin.H{"data": logs, "total": total})
}

// User CRUD (Admin)
func (h *Handler) CreateUser(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
		RoleID   int    `json:"role_id" binding:"required"`
		StoreID  *int   `json:"store_id"`
		IsActive bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	normalized := normalizeUsername(input.Username)
	if normalized == "" || normalized != strings.ToLower(input.Username) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username must be 3-50 characters, alphanumeric (a-z, 0-9), and lowercase only."})
		return
	}
	if existing, _ := h.authRepo.GetByUsername(getCtx(c), normalized); existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		return
	}
	hashedPassword, err := h.authService.HashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	user := &domain.User{
		Username: normalized,
		Email:    input.Email,
		Password: hashedPassword,
		RoleID:   input.RoleID,
		StoreID:  input.StoreID,
		IsActive: input.IsActive,
	}
	if err := h.authRepo.CreateUser(getCtx(c), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user: " + err.Error()})
		return
	}
	// Fetch created user with role
	created, err := h.authRepo.GetByID(user.ID)
	if err == nil {
		created.Password = ""
		h.logAudit(c, "create", "user", created.ID, nil, created)
		c.JSON(http.StatusCreated, gin.H{"data": created})
		return
	}
	user.Password = ""
	c.JSON(http.StatusCreated, gin.H{"data": user})
}

func (h *Handler) UpdateUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var input struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"` // optional
		RoleID   int    `json:"role_id"`
		StoreID  *int   `json:"store_id"`
		IsActive bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	existing, err := h.authRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if input.RoleID == 1 && h.userRole(c) != "superadmin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "only superadmin can assign superadmin role"})
		return
	}

	old := *existing
	old.Password = existing.Password

	if input.Username != "" {
		normalized := normalizeUsername(input.Username)
		if normalized == "" || normalized != strings.ToLower(input.Username) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username must be 3-50 characters, alphanumeric (a-z, 0-9), and lowercase only."})
			return
		}

		userByUsername, err := h.authRepo.GetByUsername(getCtx(c), normalized)
		if err == nil && userByUsername != nil && userByUsername.ID != id {
			c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
			return
		}

		existing.Username = normalized
	}
	existing.Email = input.Email
	existing.RoleID = input.RoleID
	existing.StoreID = input.StoreID
	existing.IsActive = input.IsActive
	if input.Password != "" {
		hashed, err := h.authService.HashPassword(input.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
			return
		}
		existing.Password = hashed
	}
	if err := h.authRepo.UpdateUser(getCtx(c), existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}
	existing.Password = ""
	h.logAudit(c, "update", "user", existing.ID, &old, existing)
	c.JSON(http.StatusOK, gin.H{"data": existing})
}

func (h *Handler) DeleteUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	// Check existence before delete
	existing, err := h.authRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Only superadmin can delete superadmin users
	if existing.RoleID == 1 && h.userRole(c) != "superadmin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "only superadmin can delete superadmin users"})
		return
	}

	if err := h.authRepo.DeleteUser(getCtx(c), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete user"})
		return
	}
	h.logAudit(c, "delete", "user", id, existing, nil)
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) ExportInventory(c *gin.Context) {
	products, _, _ := h.productRepo.GetAllProducts(getCtx(c), 10000, 0, "", []int{}, "name", "ASC", nil, nil, "")
	c.JSON(http.StatusOK, gin.H{"data": products})
}

func (h *Handler) AdjustStock(c *gin.Context) {
	if !h.hasPermission(c, "inventory:adjust") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permission"})
		return
	}

	var req struct {
		ProductID     int    `json:"product_id" binding:"required"`
		QuantityChange int   `json:"quantity_change" binding:"required"`
		Notes         string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.QuantityChange == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "quantity_change must not be zero"})
		return
	}
	if strings.TrimSpace(req.Notes) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "notes are required"})
		return
	}

	userIDVal, _ := c.Get("userID")
	var userID int
	switch v := userIDVal.(type) {
	case int:
		userID = v
	case float64:
		userID = int(v)
	}
	userIDPtr := &userID
	if userID == 0 {
		userIDPtr = nil
	}

  if err := h.productRepo.AdjustStock(getCtx(c), req.ProductID, req.QuantityChange, userIDPtr, strings.TrimSpace(req.Notes)); err != nil {
    if strings.Contains(err.Error(), "product not found") {
      c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
      return
    }
    if strings.Contains(err.Error(), "insufficient stock") {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }
    c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to adjust stock"})
    return
  }

   c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) logAudit(c *gin.Context, action, entityType string, entityID int, oldValues, newValues interface{}) {
	userIDVal, _ := c.Get("userID")
	usernameVal, _ := c.Get("username")
	roleVal, _ := c.Get("role")
	ip := c.ClientIP()
	ua := c.Request.UserAgent()

	var userID int
	var username string
	var role string

	switch v := userIDVal.(type) {
	case int:
		userID = v
	case float64: // sometimes numbers come as float64
		userID = int(v)
	}
	if s, ok := usernameVal.(string); ok {
		username = s
	}
	if s, ok := roleVal.(string); ok {
		role = s
	}

	log := &domain.AuditLog{
		UserID:     &userID,
		Username:   username,
		Role:       role,
		Action:     action,
		EntityType: entityType,
		EntityID:   &entityID,
		OldValues:  oldValues,
		NewValues:  newValues,
		IPAddress:  ip,
		UserAgent:  ua,
		CreatedAt:  time.Now().In(config.Load().Timezone).Format(time.RFC3339),
	}
	log.Description = h.generateAuditDescription(log)

	// Fire and forget
	go h.auditRepo.Create(context.Background(), log)
}

func (h *Handler) generateAuditDescription(log *domain.AuditLog) string {
	action := strings.ToLower(log.Action)
	entity := strings.ToLower(log.EntityType)

	getIdentifier := func(val interface{}) string {
		if val == nil {
			return ""
		}
		// Try to cast to map if it was unmarshaled from JSON (for ListAuditLogs)
		if m, ok := val.(map[string]interface{}); ok {
			if name, ok := m["name"].(string); ok {
				return name
			}
			if name, ok := m["username"].(string); ok {
				return name
			}
			if inv, ok := m["invoice_number"].(string); ok {
				return inv
			}
		}
		// Fallback for direct struct usage (for logAudit)
		switch v := val.(type) {
		case *domain.Product:
			return v.Name
		case domain.Product:
			return v.Name
		case *domain.User:
			return v.Username
		case domain.User:
			return v.Username
		case *domain.Sale:
			return v.InvoiceNumber
		case domain.Sale:
			return v.InvoiceNumber
		case *domain.Category:
			return v.Name
		case domain.Category:
			return v.Name
		case *domain.Role:
			return v.Name
		case domain.Role:
			return v.Name
		}
		return ""
	}

 	identifier := getIdentifier(log.NewValues)
	if identifier == "" {
		identifier = getIdentifier(log.OldValues)
	}

	// Format action for display
	displayAction := action
	switch action {
	case "create":
		displayAction = "Created"
	case "update":
		displayAction = "Updated"
	case "delete":
		displayAction = "Deleted"
	case "login":
		displayAction = "Logged in"
	case "logout":
		displayAction = "Logged out"
	default:
		displayAction = strings.Title(action)
	}

	if identifier != "" {
		return fmt.Sprintf("%s %s: %s", displayAction, entity, identifier)
	}

	if log.EntityID != nil && *log.EntityID > 0 {
		if entity == "auth" && (action == "login" || action == "logout") {
			if log.Username != "" {
				return fmt.Sprintf("%s %s", displayAction, log.Username)
			}
			return displayAction
		}
		return fmt.Sprintf("%s %s #%d", displayAction, entity, *log.EntityID)
	}

	if entity == "auth" && (action == "login" || action == "logout") {
		if log.Username != "" {
			return fmt.Sprintf("%s %s", displayAction, log.Username)
		}
		return displayAction
	}

	return fmt.Sprintf("%s %s", displayAction, entity)
}

func getUserID(c *gin.Context) int {
	if uid, exists := c.Get("userID"); exists {
		return uid.(int)
	}
	return 0
}

func (h *Handler) ListCategories(c *gin.Context) {
	categories, err := h.productRepo.ListCategories(getCtx(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list categories"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": categories})
}

// ==================== CATEGORY MANAGEMENT ====================

func (h *Handler) ListCategoriesManagement(c *gin.Context) {
	role := h.userRole(c)
	if role == "cashier" {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}
	if !h.hasPermission(c, "category:read") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permission"})
		return
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}
	offset := 0
	if o := c.Query("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}
	search := c.Query("search")

	categories, total, err := h.categoryRepo.GetAllCategories(getCtx(c), limit, offset, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch categories"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": categories, "total": total})
}

func (h *Handler) CreateCategoryHandler(c *gin.Context) {
	if !h.canManageCategory(c, "category.create") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permission"})
		return
	}

	var req domain.CategoryCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	category := &domain.Category{
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		IsActive:    true,
	}

	if err := h.categoryRepo.CreateCategory(getCtx(c), category); err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "violate") {
			c.JSON(http.StatusConflict, gin.H{"error": "category name or slug already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create category"})
		return
	}
	h.logAudit(c, "create", "category", category.ID, nil, category)
	c.JSON(http.StatusCreated, gin.H{"data": category})
}

func (h *Handler) UpdateCategoryHandler(c *gin.Context) {
	if !h.canManageCategory(c, "category.update") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permission"})
		return
	}

	id, _ := strconv.Atoi(c.Param("id"))

	old, err := h.categoryRepo.GetCategoryByID(getCtx(c), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "category not found"})
		return
	}

	var req domain.CategoryUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	updated := *old
	updated.Name = strings.TrimSpace(req.Name)
	updated.Description = strings.TrimSpace(req.Description)
	if req.IsActive != nil {
		updated.IsActive = *req.IsActive
	}

	if err := h.categoryRepo.UpdateCategory(getCtx(c), &updated); err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "violate") {
			c.JSON(http.StatusConflict, gin.H{"error": "category name or slug already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update category"})
		return
	}
	h.logAudit(c, "update", "category", updated.ID, old, updated)
	c.JSON(http.StatusOK, gin.H{"data": updated})
}

func (h *Handler) DeleteCategoryHandler(c *gin.Context) {
	if !h.canManageCategory(c, "category.delete") {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permission"})
		return
	}

	id, _ := strconv.Atoi(c.Param("id"))
	old, err := h.categoryRepo.GetCategoryByID(getCtx(c), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "category not found"})
		return
	}

	hasProducts, err := h.categoryRepo.HasActiveProducts(getCtx(c), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check product association"})
		return
	}
	if hasProducts {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Gagal menghapus! Kategori masih digunakan oleh produk aktif.",
		})
		return
	}

	if err := h.categoryRepo.DeleteCategory(getCtx(c), id); err != nil {
		if strings.Contains(err.Error(), "restrict") || strings.Contains(err.Error(), "violates foreign key") {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Gagal menghapus! Kategori masih digunakan oleh produk aktif.",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete category"})
		return
	}
	h.logAudit(c, "delete", "category", id, old, nil)
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// Phase 1 Extension Handlers - Brands, Tax, UOM, Warehouses

func (h *Handler) GetNextSKU(c *gin.Context) {
	sku, err := h.productRepo.GetNextSKU(getCtx(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate SKU"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"sku": sku})
}

func (h *Handler) GetBrands(c *gin.Context) {
	brands, err := h.productRepo.GetAllBrands(getCtx(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch brands"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": brands})
}

func (h *Handler) CreateBrand(c *gin.Context) {
	var brand domain.Brand
	if err := c.ShouldBindJSON(&brand); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := h.productRepo.CreateBrand(getCtx(c), &brand); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create brand"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": brand})
}

func (h *Handler) UpdateBrand(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var brand domain.Brand
	if err := c.ShouldBindJSON(&brand); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	brand.ID = id
	if err := h.productRepo.UpdateBrand(getCtx(c), &brand); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update brand"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": brand})
}

func (h *Handler) DeleteBrand(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.productRepo.DeleteBrand(getCtx(c), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete brand"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) GetTaxClasses(c *gin.Context) {
	taxClasses, err := h.productRepo.GetAllTaxClasses(getCtx(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tax classes"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": taxClasses})
}

func (h *Handler) GetUnitsOfMeasure(c *gin.Context) {
	uoms, err := h.productRepo.GetAllUnitsOfMeasure(getCtx(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch units of measure"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": uoms})
}

func (h *Handler) GetStockThresholds(c *gin.Context) {
	cfg := config.Load()
	c.JSON(http.StatusOK, gin.H{
		"warning":  cfg.StockWarningThreshold,
		"critical": cfg.StockCriticalThreshold,
	})
}

// GetAvailableYears returns distinct years that have sales data
// ==================== PAYMENT METHODS ====================

func (h *Handler) ListPaymentMethods(c *gin.Context) {
	methods, err := h.paymentRepo.GetAllActive(getCtx(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch payment methods"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": methods})
}

func (h *Handler) GetPaymentMethodByCode(c *gin.Context) {
	code := c.Param("code")
	method, err := h.paymentRepo.GetPaymentMethodByCode(getCtx(c), code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment method not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": method})
}

func (h *Handler) GetAvailableYears(c *gin.Context) {
	var storeID *int
	if sid, exists := c.Get("storeID"); exists {
		if v, ok := sid.(*int); ok && v != nil {
			storeID = v
		}
	}

	years, err := h.saleRepo.GetAvailableYears(getCtx(c), storeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch available years"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": years})
}

func (h *Handler) GetWarehouses(c *gin.Context) {
	var storeID *int
	if sid, exists := c.Get("storeID"); exists {
		if v, ok := sid.(*int); ok && v != nil {
			storeID = v
		}
	}
	warehouses, err := h.productRepo.GetAllWarehouses(getCtx(c), storeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch warehouses"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": warehouses})
}

func (h *Handler) GetCustomers(c *gin.Context) {
	limit := 50
	if l := c.Query("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 200 {
			limit = val
		}
	}
	offset := 0
	if o := c.Query("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}
	search := c.Query("search")
	var isActive *bool
	if activeStr := c.Query("isActive"); activeStr != "" {
		b := activeStr == "true"
		isActive = &b
	}
	customers, total, err := h.customerRepo.GetAllCustomers(getCtx(c), limit, offset, search, isActive)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch customers"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": customers, "total": total})
}

func (h *Handler) GetCustomerByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer id"})
		return
	}
	customer, err := h.customerRepo.GetCustomerByID(getCtx(c), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "customer not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": customer})
}

var phoneRegexp = regexp.MustCompile(`^[0-9+\-() ]{7,20}$`)

func validateCustomerRequest(name string, email, phone *string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("name is required")
	}
	if len(name) > 200 {
		return fmt.Errorf("name must be at most 200 characters")
	}
	if phone == nil || strings.TrimSpace(*phone) == "" {
		return fmt.Errorf("phone is required")
	}
	if !phoneRegexp.MatchString(strings.TrimSpace(*phone)) {
		return fmt.Errorf("invalid phone format")
	}
	if email == nil || strings.TrimSpace(*email) == "" {
		return fmt.Errorf("email is required")
	}
	if _, err := mail.ParseAddress(*email); err != nil {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

func (h *Handler) CreateCustomer(c *gin.Context) {
	var req domain.CustomerCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := validateCustomerRequest(req.Name, &req.Email, &req.Phone); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	customer := &domain.Customer{
		Name:     strings.TrimSpace(req.Name),
		Phone:    &req.Phone,
		Email:    &req.Email,
		Address:  req.Address,
		TaxID:    req.TaxID,
		Note:     req.Note,
		IsActive: true,
		IsWalkIn: false,
	}
	if err := h.customerRepo.CreateCustomer(getCtx(c), customer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create customer"})
		return
	}
	h.logAudit(c, "create", "customer", customer.ID, nil, customer)
	c.JSON(http.StatusCreated, gin.H{"data": customer})
}

func (h *Handler) UpdateCustomer(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer id"})
		return
	}
	var req domain.CustomerUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	existing, err := h.customerRepo.GetCustomerByID(getCtx(c), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "customer not found"})
		return
	}
	if existing.IsWalkIn {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot modify walk-in customer"})
		return
	}
	old := *existing
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Phone != nil {
		existing.Phone = req.Phone
	}
	if req.Email != nil {
		existing.Email = req.Email
	}
	if req.Address != nil {
		existing.Address = req.Address
	}
	if req.TaxID != nil {
		existing.TaxID = req.TaxID
	}
	if req.Note != nil {
		existing.Note = req.Note
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}
	if err := validateCustomerRequest(existing.Name, existing.Email, existing.Phone); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.customerRepo.UpdateCustomer(getCtx(c), existing, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update customer"})
		return
	}
	h.logAudit(c, "update", "customer", existing.ID, &old, existing)
	c.JSON(http.StatusOK, gin.H{"data": existing})
}

func (h *Handler) DeleteCustomer(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer id"})
	return
	}
	existing, err := h.customerRepo.GetCustomerByID(getCtx(c), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "customer not found"})
		return
	}
	if existing.IsWalkIn {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete walk-in customer"})
		return
	}
	if err := h.customerRepo.DeleteCustomer(getCtx(c), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete customer"})
		return
	}
	h.logAudit(c, "delete", "customer", id, existing, nil)
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func getRole(c *gin.Context) string {
	if role, exists := c.Get("role"); exists {
		return role.(string)
	}
	return ""
}

func (h *Handler) ServeWS(c *gin.Context) {
	if h.hub != nil {
		websocket.ServeWebSocket(h.hub, c)
	}
}

func normalizeUsername(username string) string {
	s := strings.TrimSpace(username)
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}
