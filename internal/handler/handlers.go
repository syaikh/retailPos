package handler

import (
	"fmt"
	"net/http"
	"os"
	"retailPos/internal/auth"
	model "retailPos/internal/model"
	"retailPos/internal/repo"
	"retailPos/internal/service"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Handler struct {
	authService      *auth.AuthService
	productRepo      *repo.ProductRepo
	productGroupRepo *repo.ProductGroupRepo
	statsRepo        *repo.StatsRepo
	salesRepo        *repo.SalesRepo
	salesService     *service.SalesService
}

func NewHandler(authService *auth.AuthService, productRepo *repo.ProductRepo, productGroupRepo *repo.ProductGroupRepo, statsRepo *repo.StatsRepo, salesRepo *repo.SalesRepo, salesService *service.SalesService) *Handler {
	return &Handler{
		authService:      authService,
		productRepo:      productRepo,
		productGroupRepo: productGroupRepo,
		statsRepo:        statsRepo,
		salesRepo:        salesRepo,
		salesService:     salesService,
	}
}

// Middleware
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(os.Getenv("JWT_SECRET")), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		claims, _ := token.Claims.(jwt.MapClaims)
		c.Set("user_id", int(claims["user_id"].(float64)))
		c.Set("role", claims["role"].(string))
		c.Next()
	}
}

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

	token, user, err := h.authService.Login(input.Username, input.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}

// Product Handlers
func (h *Handler) GetProducts(c *gin.Context) {
	barcode := c.Query("barcode")
	if barcode != "" {
		p, err := h.productRepo.GetBySKU(barcode)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
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

	products, total, err := h.productRepo.GetAll(limit, offset, search, groupID, sortBy, sortDir)
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

	// 3-Layer Validation Flow for Create/Restore
	existingP, err := h.productRepo.GetBySKUWithDeleted(p.SKU)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check SKU existence"})
		return
	}

	if existingP != nil {
		if existingP.DeletedAt == nil {
			// Case 2: Ada & Aktif
			c.JSON(http.StatusConflict, gin.H{"error": "Product with this SKU already exists"})
			return
		} else {
			// Case 3: Ada & Soft Deleted -> Restore
			p.ID = existingP.ID
			if err := h.productRepo.Restore(&p); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restore product"})
				return
			}
			c.JSON(http.StatusCreated, p)
			return
		}
	}

	// Case 1: Tidak ada -> Insert produk baru
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
