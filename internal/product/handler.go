package product

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/shared"

	"github.com/gin-gonic/gin"
)

func parseIDs(raw string) []int {
	if raw == "" {
		return nil
	}
	seen := make(map[int]bool)
	var ids []int
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		v, err := strconv.Atoi(s)
		if err != nil || v <= 0 {
			continue
		}
		if !seen[v] {
			seen[v] = true
			ids = append(ids, v)
		}
	}
	return ids
}

type ProductService interface {
	GetAllProducts(ctx context.Context, limit, offset int, search, sortBy, sortDir, category string, storeID *int, isActive *bool, maxStock *int, status string, supplierID *int) ([]Product, int, error)
	GetProductsByIDs(ctx context.Context, ids []int) ([]Product, error)
	GetProductByID(ctx context.Context, id, storeID int) (*Product, error)
	CreateProduct(ctx context.Context, product *Product) error
	UpdateProduct(ctx context.Context, product *Product) error
	DeleteProduct(ctx context.Context, id int, storeID *int) error
	BulkUpdateProductStatus(ctx context.Context, ids []int, isActive bool, storeID *int) error
	GetNextSKU(ctx context.Context) (string, error)
	GetAllTaxClasses(ctx context.Context) ([]TaxClass, error)
}

type Handler struct {
	svc      ProductService
	auditSvc audit.AuditCreator
}

func NewHandler(svc ProductService, auditSvc audit.AuditCreator) *Handler {
	return &Handler{svc: svc, auditSvc: auditSvc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(string) gin.HandlerFunc) {
	r.GET("/products", h.GetProducts)
	r.GET("/products/next-sku", h.GetNextSKU)

	r.GET("/products/:id", auth, h.GetProductByID)
	r.POST("/products", auth, perm("product.create"), h.CreateProduct)
	r.PUT("/products/:id", auth, perm("product.update"), h.UpdateProduct)
	r.DELETE("/products/:id", auth, perm("product.delete"), h.DeleteProduct)
	r.POST("/products/bulk/status", auth, perm("product.update"), h.BulkUpdateProductStatus)
}

func (h *Handler) RegisterPublicRoutes(r *gin.RouterGroup) {
	r.GET("/tax-classes", h.ListTaxClasses)
	r.GET("/stock-thresholds", h.GetStockThresholds)
}

func validateProduct(p *Product) error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if p.Price < 0 {
		return fmt.Errorf("price must not be negative")
	}
	if p.Cost < 0 {
		return fmt.Errorf("cost must not be negative")
	}
	if p.Stock < 0 {
		return fmt.Errorf("stock must not be negative")
	}
	return nil
}

// GetProducts godoc
// @Summary List products
// @Description Get paginated list of products with optional filters
// @Tags products
// @Accept json
// @Produce json
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Param search query string false "Search by name, SKU, or barcode"
// @Param sortBy query string false "Sort field"
// @Param sortDir query string false "Sort direction (asc or desc)"
// @Param category query string false "Filter by category name"
// @Param status query string false "Filter by status (active, inactive)"
// @Param isActive query string false "Filter by active status (true/false)"
// @Param maxStock query int false "Maximum stock threshold"
// @Success 200 {object} map[string]interface{}
// @Router /products [get]
func (h *Handler) GetProducts(c *gin.Context) {
	if idsParam := c.Query("ids"); idsParam != "" {
		ids := parseIDs(idsParam)
		if len(ids) == 0 {
			c.JSON(http.StatusOK, gin.H{"data": []Product{}, "total": 0})
			return
		}
		products, err := h.svc.GetProductsByIDs(c.Request.Context(), ids)
		if err != nil {
			shared.InternalError(c, err)
			return
		}
		if products == nil {
			products = []Product{}
		}
		c.JSON(http.StatusOK, gin.H{"data": products, "total": len(products)})
		return
	}

	limit, offset := shared.ParsePaginationParams(c.Query("limit"), c.Query("offset"))

	search := c.Query("search")
	sortBy := c.Query("sortBy")
	sortDir := c.Query("sortDir")
	category := c.Query("category")

	if c.Query("minPrice") != "" || c.Query("maxPrice") != "" || c.Query("brand") != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "minPrice, maxPrice, and brand filters are not yet implemented. Please use search or category filters instead."})
		return
	}

	status := c.Query("status")
	var isActive *bool
	if status == "" {
		if s := c.Query("isActive"); s != "" {
			v := strings.EqualFold(s, "true") || s == "1"
			isActive = &v
		}
	}

	var maxStock *int
	if ms := c.Query("maxStock"); ms != "" {
		if val, err := strconv.Atoi(ms); err == nil && val >= 0 {
			maxStock = &val
		}
	}

	storeID := shared.GetStoreID(c)

	var supplierID *int
	if sid := c.Query("supplier_id"); sid != "" {
		if val, err := strconv.Atoi(sid); err == nil && val > 0 {
			supplierID = &val
		}
	}

	products, total, err := h.svc.GetAllProducts(c.Request.Context(), limit, offset, search, sortBy, sortDir, category, storeID, isActive, maxStock, status, supplierID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch products"})
		return
	}
	if products == nil {
		products = []Product{}
	}
	shared.JSONPaginated(c, products, total, limit, offset)
}

func (h *Handler) GetProductByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	storeID := shared.GetStoreID(c)
	sid := 0
	if storeID != nil {
		sid = *storeID
	}

	product, err := h.svc.GetProductByID(c.Request.Context(), id, sid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": product})
}

// CreateProduct godoc
// @Summary Create a new product
// @Description Create a new product with the provided details
// @Tags products
// @Accept json
// @Produce json
// @Param request body Product true "Product payload"
// @Security BearerAuth
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /products [post]
func (h *Handler) CreateProduct(c *gin.Context) {
	var product Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := validateProduct(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.CreateProduct(c.Request.Context(), &product); err != nil {
		shared.InternalError(c, err)
		return
	}
	if h.auditSvc != nil {
		userID := middleware.UserIDFromContext(c.Request.Context())
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      userID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "create",
			EntityType:  "product",
			EntityID:    &product.ID,
			NewValues:   shared.ToJSONMap(product),
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Created product %s", product.Name),
		})
	}
	c.JSON(http.StatusCreated, gin.H{"data": product})
}

// UpdateProduct godoc
// @Summary Update a product
// @Description Update an existing product by ID
// @Tags products
// @Accept json
// @Produce json
// @Param id path int true "Product ID"
// @Param request body Product true "Product payload"
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /products/{id} [put]
func (h *Handler) UpdateProduct(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	var product Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	product.ID = id
	product.StoreID = shared.GetStoreID(c)

	if err := validateProduct(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var oldProduct *Product
	if h.auditSvc != nil {
		sid := 0
		if product.StoreID != nil {
			sid = *product.StoreID
		}
		oldProduct, _ = h.svc.GetProductByID(c.Request.Context(), id, sid)
	}

	if err := h.svc.UpdateProduct(c.Request.Context(), &product); err != nil {
		shared.InternalError(c, err)
		return
	}

	if h.auditSvc != nil {
		userID := middleware.UserIDFromContext(c.Request.Context())
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      userID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "update",
			EntityType:  "product",
			EntityID:    &product.ID,
			OldValues:   shared.ToJSONMap(oldProduct),
			NewValues:   shared.ToJSONMap(product),
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Updated product %s", product.Name),
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": product})
}

// DeleteProduct godoc
// @Summary Delete a product
// @Description Delete a product by ID
// @Tags products
// @Accept json
// @Produce json
// @Param id path int true "Product ID"
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /products/{id} [delete]
func (h *Handler) DeleteProduct(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	var oldProduct *Product
	if h.auditSvc != nil {
		sid := 0
		if storeID := shared.GetStoreID(c); storeID != nil {
			sid = *storeID
		}
		oldProduct, _ = h.svc.GetProductByID(c.Request.Context(), id, sid)
	}

	storeID := shared.GetStoreID(c)
	if err := h.svc.DeleteProduct(c.Request.Context(), id, storeID); err != nil {
		shared.InternalError(c, err)
		return
	}

	if h.auditSvc != nil {
		userID := middleware.UserIDFromContext(c.Request.Context())
		var description string
		if oldProduct != nil {
			description = fmt.Sprintf("Deleted product %s", oldProduct.Name)
		} else {
			description = fmt.Sprintf("Deleted product #%d", id)
		}
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      userID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "delete",
			EntityType:  "product",
			EntityID:    &id,
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: description,
		})
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) BulkUpdateProductStatus(c *gin.Context) {
	var req struct {
		IDs      []int `json:"ids"`
		IsActive bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no product IDs provided"})
		return
	}
	if len(req.IDs) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "too many product IDs (max 200)"})
		return
	}

	storeID := shared.GetStoreID(c)
	if err := h.svc.BulkUpdateProductStatus(c.Request.Context(), req.IDs, req.IsActive, storeID); err != nil {
		shared.InternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated", "updated_count": len(req.IDs)})
}

func (h *Handler) GetNextSKU(c *gin.Context) {
	sku, err := h.svc.GetNextSKU(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate SKU"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": sku})
}

func (h *Handler) ListTaxClasses(c *gin.Context) {
	taxClasses, err := h.svc.GetAllTaxClasses(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tax classes"})
		return
	}
	if taxClasses == nil {
		taxClasses = []TaxClass{}
	}
	c.JSON(http.StatusOK, gin.H{"data": taxClasses})
}

func (h *Handler) GetStockThresholds(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"warning": 10, "critical": 5})
}
