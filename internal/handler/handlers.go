package handler

import (
	"fmt"
	"net/http"
	"retailPos/internal/auth"
	model "retailPos/internal/model"
	"retailPos/internal/repo"
	"retailPos/internal/service"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	authService      auth.AuthService
	userRepo         *repo.UserRepo
	productRepo      *repo.ProductRepo
	productGroupRepo *repo.ProductGroupRepo
	statsRepo        *repo.StatsRepo
	salesRepo        *repo.SalesRepo
	salesService     *service.SalesService
}

func NewHandler(authService auth.AuthService, userRepo *repo.UserRepo, productRepo *repo.ProductRepo, productGroupRepo *repo.ProductGroupRepo, statsRepo *repo.StatsRepo, salesRepo *repo.SalesRepo, salesService *service.SalesService) *Handler {
	return &Handler{
		authService:      authService,
		userRepo:         userRepo,
		productRepo:      productRepo,
		productGroupRepo: productGroupRepo,
		statsRepo:        statsRepo,
		salesRepo:        salesRepo,
		salesService:     salesService,
	}
}

// Old AuthMiddleware has been removed and replaced by auth.AuthMiddleware

func (h *Handler) parseCommonParams(c *gin.Context) (limit, offset int, search, sortBy, sortDir string) {
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")
	limit, _ = strconv.Atoi(limitStr)
	offset, _ = strconv.Atoi(offsetStr)
	search = c.Query("search")
	sortBy = c.Query("sortBy")
	sortDir = c.DefaultQuery("sortDir", "asc")
	return
}

// Auth Handlers
func (h *Handler) Login(c *gin.Context) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ip := c.ClientIP()
	user, tokenPair, err := h.authService.Login(c.Request.Context(), input.Username, input.Password, ip)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Set HTTP-only cookie
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "session_token",
		Value:    tokenPair.AccessToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	// Token tidak lagi di-response body
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *Handler) Logout(c *gin.Context) {
	// Hapus cookie
	c.SetCookie("session_token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}

func (h *Handler) RefreshToken(c *gin.Context) {
	var input struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		refreshToken := c.GetHeader("X-Refresh-Token")
		if refreshToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Refresh token required"})
			return
		}
		input.RefreshToken = refreshToken
	}

	tokenPair, err := h.authService.RefreshToken(c.Request.Context(), input.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "session_token",
		Value:    tokenPair.AccessToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	c.JSON(http.StatusOK, gin.H{
		"message":       "Token refreshed",
		"refresh_token": tokenPair.RefreshToken,
	})
}

func (h *Handler) ValidateSession(c *gin.Context) {
	// Middleware sudah validasi, langsung return user info
	userID := c.GetInt("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No valid session"})
		return
	}

	user, err := h.userRepo.GetByID(userID)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Jangan return password hash
	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
		},
	})
}

// Product Handlers
func (h *Handler) GetProducts(c *gin.Context) {
	barcode := c.Query("barcode")
	if barcode != "" {
		// First try by barcode
		p, err := h.productRepo.GetByBarcode(barcode)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// If not found by barcode, try by SKU (legacy support)
		if p == nil {
			p, err = h.productRepo.GetBySKU(barcode)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

		if p == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		c.JSON(http.StatusOK, p)
		return
	}

	limit, offset, search, sortBy, sortDir := h.parseCommonParams(c)

	var groupID *int
	if g := c.Query("group_id"); g != "" {
		id, err := strconv.Atoi(g)
		if err == nil {
			groupID = &id
		}
	}

	var maxStock *int
	if ms := c.Query("maxStock"); ms != "" {
		val, err := strconv.Atoi(ms)
		if err == nil {
			maxStock = &val
		}
	}

	products, total, err := h.productRepo.GetAll(limit, offset, search, groupID, sortBy, sortDir, maxStock)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":   products,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *Handler) CreateProduct(c *gin.Context) {
	if c.GetString("role") != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin only"})
		return
	}

	var p model.Product
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Normalize empty barcode to nil for database NULL
	if p.Barcode != nil && *p.Barcode == "" {
		p.Barcode = nil
	}

	// 1. Check SKU Uniqueness (including deleted)
	existingBySKU, err := h.productRepo.GetBySKUWithDeleted(p.SKU)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check SKU existence"})
		return
	}

	if existingBySKU != nil {
		if existingBySKU.DeletedAt == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Product with this SKU already exists"})
			return
		}
		// Restore the deleted product
		p.ID = existingBySKU.ID
		if err := h.productRepo.Restore(&p); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restore product"})
			return
		}
		c.JSON(http.StatusCreated, p)
		return
	}

	// 2. Check Barcode Uniqueness if provided
	if p.Barcode != nil && *p.Barcode != "" {
		existingByBarcode, err := h.productRepo.GetByBarcodeWithDeleted(*p.Barcode)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check Barcode existence"})
			return
		}
		if existingByBarcode != nil {
			status := "exists"
			if existingByBarcode.DeletedAt != nil {
				status = "exists in deleted records"
			}
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Product with this Barcode already %s (SKU: %s)", status, existingByBarcode.SKU)})
			return
		}
	}

	// 3. Create new product
	if err := h.productRepo.Create(&p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, p)
}

func (h *Handler) UpdateProduct(c *gin.Context) {
	if c.GetString("role") != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin only"})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required"})
		return
	}

	idInt, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var p model.Product
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p.ID = idInt

	// Normalize empty barcode to nil for database NULL
	if p.Barcode != nil && *p.Barcode == "" {
		p.Barcode = nil
	}

	if err := h.productRepo.Update(&p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, p)
}

func (h *Handler) DeleteProduct(c *gin.Context) {
	if c.GetString("role") != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin only"})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required"})
		return
	}

	idInt, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	existingP, err := h.productRepo.GetByID(idInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check product"})
		return
	}

	if existingP == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	if existingP.Stock > 0 {
		c.JSON(http.StatusFailedDependency, gin.H{"error": "Product can only be deleted if stock is 0"})
		return
	}

	if err := h.productRepo.Delete(idInt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product deleted successfully"})
}

// Sales Handlers
func (h *Handler) CreateSale(c *gin.Context) {
	var sale model.Sale
	if err := c.ShouldBindJSON(&sale); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sale.CashierID = c.GetInt("user_id")
	if err := h.salesService.CreateSale(c.Request.Context(), &sale); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, sale)
}

// Product Group Handlers
func (h *Handler) GetProductGroups(c *gin.Context) {
	limit, offset, search, sortBy, sortDir := h.parseCommonParams(c)
	groups, total, err := h.productGroupRepo.GetAll(limit, offset, search, sortBy, sortDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":   groups,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *Handler) CreateProductGroup(c *gin.Context) {
	if c.GetString("role") != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin only"})
		return
	}

	var g model.ProductGroup
	if err := c.ShouldBindJSON(&g); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.productGroupRepo.Create(&g); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, g)
}

func (h *Handler) UpdateProductGroup(c *gin.Context) {
	if c.GetString("role") != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin only"})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required"})
		return
	}

	idInt, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var g model.ProductGroup
	if err := c.ShouldBindJSON(&g); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	g.ID = idInt

	if err := h.productGroupRepo.Update(&g); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, g)
}

func (h *Handler) DeleteProductGroup(c *gin.Context) {
	if c.GetString("role") != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin only"})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required"})
		return
	}

	idInt, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := h.productGroupRepo.Delete(idInt); err != nil {
		// Provide a friendly error if it's a foreign key constraint violation
		if strings.Contains(err.Error(), "violates foreign key constraint") {
			c.JSON(http.StatusConflict, gin.H{"error": "Kategori tidak dapat dihapus karena masih digunakan oleh satu atau lebih produk."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product Group deleted successfully"})
}

func (h *Handler) GetDashboardStats(c *gin.Context) {
	stats, err := h.statsRepo.GetDashboardStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *Handler) GetSalesHistory(c *gin.Context) {
	limit, offset, search, sortBy, sortDir := h.parseCommonParams(c)
	sales, total, err := h.salesRepo.GetAll(limit, offset, search, sortBy, sortDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":   sales,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *Handler) GetSalesChartData(c *gin.Context) {
	startDate := c.DefaultQuery("start_date", time.Now().AddDate(0, -1, 0).Format("2006-01-02"))
	endDate := c.DefaultQuery("end_date", time.Now().Format("2006-01-02"))
	groupBy := c.DefaultQuery("group_by", "day")

	fmt.Printf("CHART HANDLER: start=%s, end=%s, group=%s\n", startDate, endDate, groupBy)

	data, err := h.statsRepo.GetSalesChartData(startDate, endDate, groupBy)
	if err != nil {
		fmt.Printf("CHART ERROR: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	fmt.Printf("CHART DATA: %+v\n", data)
	c.JSON(http.StatusOK, data)
}
