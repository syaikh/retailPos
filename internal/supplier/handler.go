package supplier

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/permissions"
	"retail-pos-system/internal/shared"

	"github.com/gin-gonic/gin"
)

type Service interface {
	GetByID(ctx context.Context, id int) (*Supplier, error)
	GetByCode(ctx context.Context, code string) (*Supplier, error)
	GetAll(ctx context.Context, limit, offset int, search string, isActive *bool) ([]Supplier, int, error)
	Create(ctx context.Context, supplier *Supplier) error
	Update(ctx context.Context, supplier *Supplier) error
	Delete(ctx context.Context, id int) error
	LinkProduct(ctx context.Context, ps *ProductSupplier) error
	UnlinkProduct(ctx context.Context, productID, supplierID int) error
	GetProductSupplier(ctx context.Context, productID, supplierID int) (*ProductSupplier, error)
	GetPreferredSupplier(ctx context.Context, productID int) (*ProductSupplier, error)
	SetPreferredSupplier(ctx context.Context, productID, supplierID int) error
	UpdateProductSupplier(ctx context.Context, ps *ProductSupplier) error
	GetSuppliersByProductID(ctx context.Context, productID int) ([]ProductSupplier, error)
	GetProductsBySupplierID(ctx context.Context, supplierID int) ([]ProductSupplier, error)
	BulkUpdate(ctx context.Context, ids []int, isActive bool) (int, error)
	BulkDelete(ctx context.Context, ids []int) (int, error)
}

type Handler struct {
	svc      Service
	auditSvc audit.Creator
}

func NewHandler(svc Service, auditSvc audit.Creator) *Handler {
	return &Handler{svc: svc, auditSvc: auditSvc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(permissions.Code) gin.HandlerFunc) {
	r.GET("/suppliers", auth, perm(permissions.PricingView), h.ListSuppliers)
	r.GET("/suppliers/:id", auth, perm(permissions.PricingView), h.GetSupplier)
	r.POST("/suppliers", auth, perm(permissions.PricingCreate), h.CreateSupplier)
	r.PUT("/suppliers/:id", auth, perm(permissions.PricingUpdate), h.UpdateSupplier)
	r.DELETE("/suppliers/:id", auth, perm(permissions.PricingDelete), h.DeleteSupplier)

	r.PUT("/suppliers/bulk", auth, perm(permissions.PricingUpdate), h.BulkUpdate)
	r.DELETE("/suppliers/bulk", auth, perm(permissions.PricingDelete), h.BulkDelete)

	r.GET("/suppliers/:id/products", auth, perm(permissions.PricingView), h.GetProductsBySupplier)
	r.POST("/suppliers/:id/products", auth, perm(permissions.PricingUpdate), h.LinkProduct)
	r.DELETE("/suppliers/:id/products/:productId", auth, perm(permissions.PricingUpdate), h.UnlinkProduct)
	r.PUT("/suppliers/:id/products/:productId", auth, perm(permissions.PricingUpdate), h.UpdateProductSupplier)
	r.POST("/suppliers/:id/products/:productId/preferred", auth, perm(permissions.PricingUpdate), h.SetPreferredSupplier)

	r.GET("/products/:id/suppliers", auth, perm(permissions.PricingView), h.GetSuppliersByProduct)
}

// ListSuppliers godoc
// @Summary List suppliers
// @Description Get paginated list of suppliers with optional filters
// @Tags suppliers
// @Accept json
// @Produce json
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Param search query string false "Search by name, code, or contact name"
// @Param is_active query string false "Filter by active status (true/false)"
// @Success 200 {object} map[string]interface{}
// @Router /suppliers [get]
func (h *Handler) ListSuppliers(c *gin.Context) {
	limit, offset := shared.ParsePaginationParams(c.Query("limit"), c.Query("offset"))
	search := c.Query("search")

	var isActive *bool
	if v := c.Query("is_active"); v != "" {
		b := strings.EqualFold(v, "true") || v == "1"
		isActive = &b
	}

	suppliers, total, err := h.svc.GetAll(c.Request.Context(), limit, offset, search, isActive)
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	if suppliers == nil {
		suppliers = []Supplier{}
	}
	shared.JSONPaginated(c, suppliers, total, limit, offset)
}

// GetSupplier godoc
// @Summary Get a supplier by ID
// @Description Get a single supplier by its ID
// @Tags suppliers
// @Accept json
// @Produce json
// @Param id path int true "Supplier ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /suppliers/{id} [get]
func (h *Handler) GetSupplier(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid supplier id"})
		return
	}

	supplier, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "supplier not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": supplier})
}

// CreateSupplier godoc
// @Summary Create a new supplier
// @Description Create a new supplier
// @Tags suppliers
// @Accept json
// @Produce json
// @Param supplier body Supplier true "Supplier"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /suppliers [post]
func (h *Handler) CreateSupplier(c *gin.Context) {
	var supplier Supplier
	if err := c.ShouldBindJSON(&supplier); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.Create(c.Request.Context(), &supplier); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.auditSvc != nil {
		userID := middleware.UserIDFromContext(c.Request.Context())
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.Log{
			UserID:      userID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "create",
			EntityType:  "supplier",
			EntityID:    &supplier.ID,
			NewValues:   shared.ToJSONMap(supplier),
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Created supplier %s", supplier.Name),
			StoreID:     middleware.StoreIDFromContext(c.Request.Context()),
		})
	}
	c.JSON(http.StatusCreated, gin.H{"data": supplier})
}

// UpdateSupplier godoc
// @Summary Update a supplier
// @Description Update an existing supplier by ID
// @Tags suppliers
// @Accept json
// @Produce json
// @Param id path int true "Supplier ID"
// @Param supplier body Supplier true "Supplier"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /suppliers/{id} [put]
func (h *Handler) UpdateSupplier(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid supplier id"})
		return
	}

	oldSupplier, _ := h.svc.GetByID(c.Request.Context(), id)

	var supplier Supplier
	if err := c.ShouldBindJSON(&supplier); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	supplier.ID = id

	if oldSupplier != nil {
		if supplier.Code == "" {
			supplier.Code = oldSupplier.Code
		}
		if supplier.Name == "" {
			supplier.Name = oldSupplier.Name
		}
	}

	if err := h.svc.Update(c.Request.Context(), &supplier); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.auditSvc != nil {
		userID := middleware.UserIDFromContext(c.Request.Context())
		focusedOld, focusedNew := shared.DiffChanges(shared.ToJSONMap(oldSupplier), shared.ToJSONMap(supplier))
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.Log{
			UserID:      userID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "update",
			EntityType:  "supplier",
			EntityID:    &supplier.ID,
			OldValues:   focusedOld,
			NewValues:   focusedNew,
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Updated supplier %s", supplier.Name),
			StoreID:     middleware.StoreIDFromContext(c.Request.Context()),
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": supplier})
}

// DeleteSupplier godoc
// @Summary Delete a supplier
// @Description Soft delete a supplier by ID
// @Tags suppliers
// @Accept json
// @Produce json
// @Param id path int true "Supplier ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /suppliers/{id} [delete]
func (h *Handler) DeleteSupplier(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid supplier id"})
		return
	}

	var oldSupplierName string
	if h.auditSvc != nil {
		if s, err := h.svc.GetByID(c.Request.Context(), id); err == nil {
			oldSupplierName = s.Name
		}
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		shared.InternalError(c, err)
		return
	}

	if h.auditSvc != nil {
		userID := middleware.UserIDFromContext(c.Request.Context())
		var description string
		if oldSupplierName != "" {
			description = fmt.Sprintf("Deleted supplier %s", oldSupplierName)
		} else {
			description = fmt.Sprintf("Deleted supplier #%d", id)
		}
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.Log{
			UserID:      userID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "delete",
			EntityType:  "supplier",
			EntityID:    &id,
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: description,
			StoreID:     middleware.StoreIDFromContext(c.Request.Context()),
		})
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// GetProductsBySupplier godoc
// @Summary Get products linked to a supplier
// @Description Get all products linked to a specific supplier
// @Tags suppliers
// @Accept json
// @Produce json
// @Param id path int true "Supplier ID"
// @Success 200 {object} map[string]interface{}
// @Router /suppliers/{id}/products [get]
func (h *Handler) GetProductsBySupplier(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid supplier id"})
		return
	}

	products, err := h.svc.GetProductsBySupplierID(c.Request.Context(), id)
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	if products == nil {
		products = []ProductSupplier{}
	}
	c.JSON(http.StatusOK, gin.H{"data": products})
}

func (h *Handler) GetSuppliersByProduct(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	suppliers, err := h.svc.GetSuppliersByProductID(c.Request.Context(), id)
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	if suppliers == nil {
		suppliers = []ProductSupplier{}
	}
	c.JSON(http.StatusOK, gin.H{"data": suppliers})
}

// LinkProduct godoc
// @Summary Link a product to a supplier
// @Description Create a product-supplier relationship
// @Tags suppliers
// @Accept json
// @Produce json
// @Param id path int true "Supplier ID"
// @Param product body ProductSupplier true "Product Supplier link"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /suppliers/{id}/products [post]
func (h *Handler) LinkProduct(c *gin.Context) {
	supplierID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid supplier id"})
		return
	}

	var ps ProductSupplier
	if err := c.ShouldBindJSON(&ps); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ps.SupplierID = supplierID

	if err := h.svc.LinkProduct(c.Request.Context(), &ps); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.auditSvc != nil {
		userID := middleware.UserIDFromContext(c.Request.Context())
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.Log{
			UserID:      userID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "create",
			EntityType:  "product_supplier",
			NewValues:   shared.ToJSONMap(ps),
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Linked product #%d to supplier #%d", ps.ProductID, ps.SupplierID),
			StoreID:     middleware.StoreIDFromContext(c.Request.Context()),
		})
	}
	c.JSON(http.StatusCreated, gin.H{"data": ps})
}

func (h *Handler) UnlinkProduct(c *gin.Context) {
	supplierID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid supplier id"})
		return
	}
	productID, err := strconv.Atoi(c.Param("productId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	if err := h.svc.UnlinkProduct(c.Request.Context(), productID, supplierID); err != nil {
		shared.InternalError(c, err)
		return
	}

	if h.auditSvc != nil {
		userID := middleware.UserIDFromContext(c.Request.Context())
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.Log{
			UserID:      userID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "delete",
			EntityType:  "product_supplier",
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Unlinked product #%d from supplier #%d", productID, supplierID),
			StoreID:     middleware.StoreIDFromContext(c.Request.Context()),
		})
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) UpdateProductSupplier(c *gin.Context) {
	supplierID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid supplier id"})
		return
	}
	productID, err := strconv.Atoi(c.Param("productId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	var oldPS *ProductSupplier
	if h.auditSvc != nil {
		oldPS, _ = h.svc.GetProductSupplier(c.Request.Context(), productID, supplierID)
	}

	var ps ProductSupplier
	if err := c.ShouldBindJSON(&ps); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ps.ProductID = productID
	ps.SupplierID = supplierID

	if err := h.svc.UpdateProductSupplier(c.Request.Context(), &ps); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.auditSvc != nil {
		userID := middleware.UserIDFromContext(c.Request.Context())
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.Log{
			UserID:      userID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "update",
			EntityType:  "product_supplier",
			OldValues:   shared.ToJSONMap(oldPS),
			NewValues:   shared.ToJSONMap(ps),
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Updated product-supplier link for product #%d supplier #%d", productID, supplierID),
			StoreID:     middleware.StoreIDFromContext(c.Request.Context()),
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": ps})
}

func (h *Handler) SetPreferredSupplier(c *gin.Context) {
	supplierID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid supplier id"})
		return
	}
	productID, err := strconv.Atoi(c.Param("productId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	if err := h.svc.SetPreferredSupplier(c.Request.Context(), productID, supplierID); err != nil {
		shared.InternalError(c, err)
		return
	}

	if h.auditSvc != nil {
		userID := middleware.UserIDFromContext(c.Request.Context())
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.Log{
			UserID:      userID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "update",
			EntityType:  "product_supplier",
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Set supplier #%d as preferred for product #%d", supplierID, productID),
			StoreID:     middleware.StoreIDFromContext(c.Request.Context()),
		})
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

// BulkUpdate godoc
// @Summary Bulk update suppliers
// @Description Bulk update supplier active status
// @Tags suppliers
// @Accept json
// @Produce json
// @Param body body object true "Bulk update payload"
// @Success 200 {object} map[string]interface{}
// @Router /suppliers/bulk [put]
func (h *Handler) BulkUpdate(c *gin.Context) {
	var req struct {
		IDs      []int `json:"ids" binding:"required"`
		IsActive bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updated, err := h.svc.BulkUpdate(c.Request.Context(), req.IDs, req.IsActive)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.auditSvc != nil {
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.Log{
			UserID:      middleware.UserIDFromContext(c.Request.Context()),
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "bulk_update",
			EntityType:  "supplier",
			NewValues:   map[string]interface{}{"ids": req.IDs, "is_active": req.IsActive},
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Bulk updated %d suppliers", updated),
			StoreID:     middleware.StoreIDFromContext(c.Request.Context()),
		})
	}

	c.JSON(http.StatusOK, gin.H{"updated": updated})
}

// BulkDelete godoc
// @Summary Bulk delete suppliers
// @Description Soft delete multiple suppliers
// @Tags suppliers
// @Accept json
// @Produce json
// @Param body body object true "Bulk delete payload"
// @Success 200 {object} map[string]interface{}
// @Router /suppliers/bulk [delete]
func (h *Handler) BulkDelete(c *gin.Context) {
	var req struct {
		IDs []int `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deleted, err := h.svc.BulkDelete(c.Request.Context(), req.IDs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.auditSvc != nil {
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.Log{
			UserID:      middleware.UserIDFromContext(c.Request.Context()),
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "bulk_delete",
			EntityType:  "supplier",
			NewValues:   map[string]interface{}{"ids": req.IDs},
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Bulk deleted %d suppliers", deleted),
			StoreID:     middleware.StoreIDFromContext(c.Request.Context()),
		})
	}

	c.JSON(http.StatusOK, gin.H{"deleted": deleted})
}
