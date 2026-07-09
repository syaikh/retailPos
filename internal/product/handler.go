package product

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(string) gin.HandlerFunc) {
	r.GET("/products", h.GetProducts)
	r.GET("/products/next-sku", h.GetNextSKU)

	r.GET("/products/:id", auth, h.GetProductByID)
	r.POST("/products", auth, perm("product:create"), h.CreateProduct)
	r.PUT("/products/:id", auth, perm("product:update"), h.UpdateProduct)
	r.DELETE("/products/:id", auth, perm("product:delete"), h.DeleteProduct)
	r.POST("/products/bulk/status", auth, perm("product:update"), h.BulkUpdateProductStatus)
}

func (h *Handler) RegisterPublicRoutes(r *gin.RouterGroup) {
	r.GET("/tax-classes", h.ListTaxClasses)
	r.GET("/stock-thresholds", h.GetStockThresholds)
}

func (h *Handler) getStoreID(c *gin.Context) *int {
	if sid, exists := c.Get("storeID"); exists {
		if v, ok := sid.(*int); ok {
			return v
		}
	}
	return nil
}

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

	search := c.Query("search")
	sortBy := c.Query("sortBy")
	sortDir := c.Query("sortDir")
	category := c.Query("category")
	brand := c.Query("brand")

	status := c.Query("status")
	var isActive *bool
	if status == "" {
		if s := c.Query("isActive"); s != "" {
			v := strings.EqualFold(s, "true") || s == "1"
			isActive = &v
		}
	}

	var minPrice, maxPrice *float64
	if mp := c.Query("minPrice"); mp != "" {
		if val, err := strconv.ParseFloat(mp, 64); err == nil {
			minPrice = &val
		}
	}
	if mxp := c.Query("maxPrice"); mxp != "" {
		if val, err := strconv.ParseFloat(mxp, 64); err == nil {
			maxPrice = &val
		}
	}

	var maxStock *int
	if ms := c.Query("maxStock"); ms != "" {
		if val, err := strconv.Atoi(ms); err == nil && val >= 0 {
			maxStock = &val
		}
	}

	storeID := h.getStoreID(c)

	products, total, err := h.svc.GetAllProducts(c.Request.Context(), limit, offset, search, sortBy, sortDir, category, brand, storeID, isActive, minPrice, maxPrice, maxStock, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch products"})
		return
	}
	if products == nil {
		products = []Product{}
	}
	c.JSON(http.StatusOK, gin.H{"data": products, "total": total})
}

func (h *Handler) GetProductByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	storeID := h.getStoreID(c)
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

func (h *Handler) CreateProduct(c *gin.Context) {
	var product Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := h.svc.CreateProduct(c.Request.Context(), &product); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create product"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": product})
}

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

	if err := h.svc.UpdateProduct(c.Request.Context(), &product); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update product"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": product})
}

func (h *Handler) DeleteProduct(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	if err := h.svc.DeleteProduct(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete product"})
		return
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

	if err := h.svc.BulkUpdateProductStatus(c.Request.Context(), req.IDs, req.IsActive); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update product statuses"})
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
