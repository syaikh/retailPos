package supplier

import (
	"context"
	"errors"
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
	GetAll(ctx context.Context, limit, offset int, search string, isActive *bool, isConsignment *bool) ([]Supplier, int, error)
	Create(ctx context.Context, supplier *Supplier) error
	Update(ctx context.Context, supplier *Supplier) error
	Delete(ctx context.Context, id int) error
	LinkProduct(ctx context.Context, ps *ProductSupplier) error
	UnlinkProduct(ctx context.Context, productID, supplierID int, storeID *int) error
	GetProductSupplier(ctx context.Context, productID, supplierID int, storeID *int) (*ProductSupplier, error)
	GetPreferredSupplier(ctx context.Context, productID int, storeID *int) (*ProductSupplier, error)
	SetPreferredSupplier(ctx context.Context, productID, supplierID int, storeID *int) error
	UpdateProductSupplier(ctx context.Context, ps *ProductSupplier, storeID *int) error
	GetSuppliersByProductID(ctx context.Context, productID int, storeID *int) ([]ProductSupplier, error)
	GetProductsBySupplierID(ctx context.Context, supplierID int, storeID *int) ([]ProductSupplier, error)
	BulkUpdate(ctx context.Context, ids []int, isActive bool, updatedBy *int) (int, error)
	BulkDelete(ctx context.Context, ids []int) (int, error)
	GetUsage(ctx context.Context, id int) (SupplierUsage, error)
}

type Handler struct {
	svc      Service
	auditSvc audit.Creator
}

func NewHandler(svc Service, auditSvc audit.Creator) *Handler {
	return &Handler{svc: svc, auditSvc: auditSvc}
}

// isAdmin reports whether the caller bypasses store scoping on supplier terms.
// The supplier itself is global under audit D3 Option C, so this governs only
// product_suppliers -- but the unit cost and preferred-supplier choice living
// there are the commercially sensitive part, so they get the same treatment
// pricing rules already get.
func isAdmin(c *gin.Context) bool {
	return shared.GetRole(c) == permissions.RoleSuperadmin
}

// linkScope resolves the store scope for a product-supplier link operation: nil
// is unrestricted (superadmin), non-nil is the caller's own store.
//
// It fails closed on a missing claim. Every query below reads a nil scope as
// "every store", so a nil claim falling through would hand a store-scoped caller
// every store's negotiated cost -- the same trap pricing's ListRules documents.
func linkScope(c *gin.Context) (*int, bool) {
	if isAdmin(c) {
		return nil, true
	}
	claim := shared.GetStoreID(c)
	if claim == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "store scope is required for supplier links"})
		return nil, false
	}
	return claim, true
}

// authorizeLink refuses a store-scoped caller changing a global terms row.
//
// The write SQL already refuses it (writes match the store column exactly), but
// an inherited global link is *visible* to this caller, so a bare not-found
// would read as "this supplier is not linked to this product" while the link
// sits on screen. This is the accurate answer, and it matches pricing's
// authorizeStoreRule, which refuses a store-scoped caller editing a global rule.
func authorizeLink(c *gin.Context, link *ProductSupplier) bool {
	if isAdmin(c) || link.StoreID != nil {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{
		"error": "these supplier terms are the global default and apply to every store; " +
			"link the supplier for your store first to set your own terms",
	})
	return false
}

// linkNotFound answers a link the caller's store neither owns nor inherits.
func linkNotFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": "product-supplier link not found"})
}

// authorizeLinkWrite loads the link a write names and checks the caller may
// change it. It reports the outcome itself, so a caller cannot forget the check.
func (h *Handler) authorizeLinkWrite(c *gin.Context, productID, supplierID int, scope *int) bool {
	link, err := h.svc.GetProductSupplier(c.Request.Context(), productID, supplierID, scope)
	if err != nil {
		if errors.Is(err, ErrProductSupplierNotFound) {
			linkNotFound(c)
			return false
		}
		shared.InternalError(c, err)
		return false
	}
	if link == nil {
		// A nil link with no error is a broken store, but treating it as "allowed"
		// would turn it into an authorization bypass, and dereferencing it would
		// take the request down. Answer 404 and make the write happen anyway is
		// never acceptable, so this returns without writing.
		linkNotFound(c)
		return false
	}
	return authorizeLink(c, link)
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(permissions.Code) gin.HandlerFunc) {
	r.GET("/suppliers", auth, perm(permissions.SupplierView), h.ListSuppliers)
	r.GET("/suppliers/:id", auth, perm(permissions.SupplierView), h.GetSupplier)
	r.GET("/suppliers/:id/usage", auth, perm(permissions.SupplierView), h.GetSupplierUsage)
	r.POST("/suppliers", auth, perm(permissions.SupplierCreate), h.CreateSupplier)
	r.PUT("/suppliers/:id", auth, perm(permissions.SupplierUpdate), h.UpdateSupplier)
	r.DELETE("/suppliers/:id", auth, perm(permissions.SupplierDelete), h.DeleteSupplier)

	r.PUT("/suppliers/bulk", auth, perm(permissions.SupplierUpdate), h.BulkUpdate)
	r.DELETE("/suppliers/bulk", auth, perm(permissions.SupplierDelete), h.BulkDelete)

	r.GET("/suppliers/:id/products", auth, perm(permissions.SupplierView), h.GetProductsBySupplier)
	r.POST("/suppliers/:id/products", auth, perm(permissions.SupplierUpdate), h.LinkProduct)
	r.DELETE("/suppliers/:id/products/:productId", auth, perm(permissions.SupplierUpdate), h.UnlinkProduct)
	r.PUT("/suppliers/:id/products/:productId", auth, perm(permissions.SupplierUpdate), h.UpdateProductSupplier)
	r.POST("/suppliers/:id/products/:productId/preferred", auth, perm(permissions.SupplierUpdate), h.SetPreferredSupplier)

	r.GET("/products/:id/suppliers", auth, perm(permissions.SupplierView), h.GetSuppliersByProduct)
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
// @Param is_consignment query string false "Filter by consignment status (true/false)"
// @Success 200 {object} map[string]interface{}
// @Router /suppliers [get]
func (h *Handler) ListSuppliers(c *gin.Context) {
	limit, offset, err := shared.ParsePaginationParams(c.Query("limit"), c.Query("offset"))
	if err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, err.Error())
		return
	}
	search := c.Query("search")

	var isActive *bool
	if v := c.Query("is_active"); v != "" {
		b := strings.EqualFold(v, "true") || v == "1"
		isActive = &b
	}

	var isConsignment *bool
	if v := c.Query("is_consignment"); v != "" {
		b := strings.EqualFold(v, "true") || v == "1"
		isConsignment = &b
	}

	suppliers, total, err := h.svc.GetAll(c.Request.Context(), limit, offset, search, isActive, isConsignment)
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

// GetSupplierUsage godoc
// @Summary Get a supplier's reference usage
// @Description Get the cross-module reference breakdown for a supplier (product links, open purchase orders, active consignments) so the UI can warn before deleting or deactivating.
// @Tags suppliers
// @Produce json
// @Param id path int true "Supplier ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /suppliers/{id}/usage [get]
func (h *Handler) GetSupplierUsage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid supplier id"})
		return
	}

	usage, err := h.svc.GetUsage(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrSupplierNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "supplier not found"})
			return
		}
		shared.InternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": usage})
}

// writeUsageConflict answers a delete/deactivate blocked by live references. The
// 409 body carries the breakdown and, for a bulk call, the specific ids that
// must be cleared, so the UI can explain the blast radius instead of a bare
// conflict.
func writeUsageConflict(c *gin.Context, inUse *SupplierInUseError) {
	body := gin.H{
		"error": inUse.Error(),
		"code":  "supplier_in_use",
		"usage": inUse.Usage,
	}
	if len(inUse.BlockedIDs) > 0 {
		body["blocked_ids"] = inUse.BlockedIDs
	}
	c.JSON(http.StatusConflict, body)
}

// respondSupplierWriteError maps a supplier write error to its HTTP status.
// Shared by the single and bulk handlers so a blocked delete and a blocked
// deactivate can never answer differently. fallback is the status for errors
// that are neither in-use nor not-found (400 for update input, 500 for delete).
func respondSupplierWriteError(c *gin.Context, err error, fallback int) {
	var inUse *SupplierInUseError
	switch {
	case errors.As(err, &inUse):
		writeUsageConflict(c, inUse)
	case errors.Is(err, ErrSupplierVersionConflict):
		c.JSON(http.StatusConflict, gin.H{
			"error": err.Error(),
			"code":  "supplier_version_conflict",
		})
	case errors.Is(err, ErrSupplierNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "supplier not found"})
	default:
		if fallback == http.StatusInternalServerError {
			shared.InternalError(c, err)
			return
		}
		c.JSON(fallback, gin.H{"error": err.Error()})
	}
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

	// Provenance is server-set; a client-supplied value is ignored.
	actor := middleware.UserIDFromContext(c.Request.Context())
	supplier.CreatedBy = actor
	supplier.UpdatedBy = actor

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
	// Provenance is server-set; a client-supplied value is ignored. Version is
	// taken from the request body (the value the client read); 0 means the
	// client did not opt into the concurrency check.
	supplier.UpdatedBy = middleware.UserIDFromContext(c.Request.Context())

	if oldSupplier != nil {
		if supplier.Code == "" {
			supplier.Code = oldSupplier.Code
		}
		if supplier.Name == "" {
			supplier.Name = oldSupplier.Name
		}
	}

	if err := h.svc.Update(c.Request.Context(), &supplier); err != nil {
		respondSupplierWriteError(c, err, http.StatusBadRequest)
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
		respondSupplierWriteError(c, err, http.StatusInternalServerError)
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

	scope, ok := linkScope(c)
	if !ok {
		return
	}

	products, err := h.svc.GetProductsBySupplierID(c.Request.Context(), id, scope)
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	if products == nil {
		products = []ProductSupplier{}
	}
	c.JSON(http.StatusOK, gin.H{"data": presentProductSuppliers(products, canViewUnitCost(c))})
}

func (h *Handler) GetSuppliersByProduct(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	scope, ok := linkScope(c)
	if !ok {
		return
	}

	suppliers, err := h.svc.GetSuppliersByProductID(c.Request.Context(), id, scope)
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	if suppliers == nil {
		suppliers = []ProductSupplier{}
	}
	c.JSON(http.StatusOK, gin.H{"data": presentProductSuppliers(suppliers, canViewUnitCost(c))})
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

	scope, ok := linkScope(c)
	if !ok {
		return
	}
	// A store-scoped caller writes its own store's terms. A body-supplied store
	// is rejected rather than honoured, so the claim always wins and a store
	// cannot write terms into somebody else's price list.
	if scope != nil {
		if ps.StoreID != nil && *ps.StoreID != *scope {
			c.JSON(http.StatusForbidden, gin.H{"error": "store is not in your store"})
			return
		}
		ps.StoreID = scope
	}

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
	c.JSON(http.StatusCreated, gin.H{"data": presentProductSupplier(ps, canViewUnitCost(c))})
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

	scope, ok := linkScope(c)
	if !ok {
		return
	}
	if !h.authorizeLinkWrite(c, productID, supplierID, scope) {
		return
	}

	if err := h.svc.UnlinkProduct(c.Request.Context(), productID, supplierID, scope); err != nil {
		if errors.Is(err, ErrProductSupplierNotFound) {
			linkNotFound(c)
			return
		}
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

	scope, ok := linkScope(c)
	if !ok {
		return
	}

	// The body is parsed before the link lookup on purpose: a malformed payload
	// is a bad request whatever the state of the link, and answering 400 without
	// a query keeps the error about what the caller actually got wrong.
	var ps ProductSupplier
	if err := c.ShouldBindJSON(&ps); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ps.ProductID = productID
	ps.SupplierID = supplierID

	// Validated here as well as in the service, because the payload is checked
	// before the link lookup: unit_cost of -5 is a bad request whether or not
	// this caller may edit the row it names. The service still validates, so
	// non-HTTP callers keep the same guarantee.
	if err := validateProductSupplier(&ps); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !h.authorizeLinkWrite(c, productID, supplierID, scope) {
		return
	}

	// The pre-edit row is read after authorization so the audit trail records
	// what was actually replaced, and only for the store that may change it.
	var oldPS *ProductSupplier
	if h.auditSvc != nil {
		oldPS, _ = h.svc.GetProductSupplier(c.Request.Context(), productID, supplierID, scope)
	}

	if err := h.svc.UpdateProductSupplier(c.Request.Context(), &ps, scope); err != nil {
		if errors.Is(err, ErrProductSupplierNotFound) {
			linkNotFound(c)
			return
		}
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
	c.JSON(http.StatusOK, gin.H{"data": presentProductSupplier(ps, canViewUnitCost(c))})
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

	scope, ok := linkScope(c)
	if !ok {
		return
	}
	if !h.authorizeLinkWrite(c, productID, supplierID, scope) {
		return
	}

	if err := h.svc.SetPreferredSupplier(c.Request.Context(), productID, supplierID, scope); err != nil {
		if errors.Is(err, ErrProductSupplierNotFound) {
			linkNotFound(c)
			return
		}
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

	updated, err := h.svc.BulkUpdate(c.Request.Context(), req.IDs, req.IsActive, middleware.UserIDFromContext(c.Request.Context()))
	if err != nil {
		respondSupplierWriteError(c, err, http.StatusBadRequest)
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
		respondSupplierWriteError(c, err, http.StatusBadRequest)
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
