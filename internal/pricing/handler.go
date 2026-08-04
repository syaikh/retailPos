package pricing

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

type PricingService interface {
	GetByID(ctx context.Context, id int) (*PricingRule, error)
	GetByProductID(ctx context.Context, productID int) ([]PricingRule, error)
	GetAll(ctx context.Context, limit, offset int, search string, productID *int, pricingType, pricingMethod string, categoryID, brandID, customerGroupID, storeID *int, isActive *bool, status string) ([]PricingRule, int, error)
	Create(ctx context.Context, rule *PricingRule) error
	Update(ctx context.Context, rule *PricingRule) error
	Delete(ctx context.Context, id int) error
	FindConflictsForRule(ctx context.Context, rule *PricingRule, excludeID int) ([]PricingRule, error)
	SubmitForApproval(ctx context.Context, id int) error
	Approve(ctx context.Context, id int) error
	Reject(ctx context.Context, id int) error
}

// ProductSearcher is an optional interface for product autocomplete search.
type ProductSearcher interface {
	SearchProducts(ctx context.Context, query string, limit int) ([]ProductSearchResult, error)
}

type Handler struct {
	svc      PricingService
	resolver PriceResolver
	searcher ProductSearcher
	auditSvc audit.AuditCreator
}

func NewHandler(svc PricingService, resolver PriceResolver, auditSvc audit.AuditCreator) *Handler {
	return &Handler{svc: svc, resolver: resolver, auditSvc: auditSvc}
}

// SetProductSearcher sets the optional product search provider.
func (h *Handler) SetProductSearcher(s ProductSearcher) {
	h.searcher = s
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(permissions.Code) gin.HandlerFunc) {
	r.GET("/pricing-rules", auth, perm(permissions.PricingView), h.ListRules)
	r.GET("/pricing-rules/:id", auth, perm(permissions.PricingView), h.GetRule)
	r.POST("/pricing-rules", auth, perm(permissions.PricingCreate), h.CreateRule)
	r.PUT("/pricing-rules/:id", auth, perm(permissions.PricingUpdate), h.UpdateRule)
	r.DELETE("/pricing-rules/:id", auth, perm(permissions.PricingDelete), h.DeleteRule)
	r.POST("/pricing-rules/check-conflicts", auth, perm(permissions.PricingView), h.CheckConflicts)
	r.POST("/pricing-rules/:id/submit", auth, perm(permissions.PricingUpdate), h.SubmitForApproval)
	r.POST("/pricing-rules/:id/approve", auth, perm(permissions.PricingUpdate), h.ApproveRule)
	r.POST("/pricing-rules/:id/reject", auth, perm(permissions.PricingUpdate), h.RejectRule)
	r.POST("/pricing/resolve", auth, perm(permissions.PricingView), h.ResolvePrices)
	r.GET("/products/search", auth, perm(permissions.PricingView), h.SearchProducts)
}

// ListRules godoc
// @Summary      List pricing rules
// @Description  Get a paginated list of pricing rules with optional filters
// @Tags         Pricing
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        limit             query   int     false  "Page size"          default(20)
// @Param        offset            query   int     false  "Offset"             default(0)
// @Param        search            query   string  false  "Search rule name"
// @Param        product_id        query   int     false  "Filter by product ID"
// @Param        pricing_type      query   string  false  "Filter by type (special_price|promotion)"
// @Param        pricing_method    query   string  false  "Filter by method (fixed_price|discount_percent|discount_amount|markup_percent)"
// @Param        category_id       query   int     false  "Filter by category ID"
// @Param        brand_id          query   int     false  "Filter by brand ID"
// @Param        customer_group_id query   int     false  "Filter by customer group ID"
// @Param        store_id          query   int     false  "Filter by store ID"
// @Param        is_active         query   bool    false  "Filter by active status"
// @Param        status            query   string  false  "Filter by approval status (draft|pending|approved|rejected)"
// @Success      200  {object}  map[string]interface{}
// @Router       /pricing-rules [get]
func (h *Handler) ListRules(c *gin.Context) {
	limit, offset := shared.ParsePaginationParams(c.Query("limit"), c.Query("offset"))
	search := c.Query("search")

	var productID *int
	if v := c.Query("product_id"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			productID = &n
		}
	}

	pricingType := c.Query("pricing_type")
	pricingMethod := c.Query("pricing_method")

	var categoryID, brandID, customerGroupID, storeID *int
	if v := c.Query("category_id"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			categoryID = &n
		}
	}
	if v := c.Query("brand_id"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			brandID = &n
		}
	}
	if v := c.Query("customer_group_id"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			customerGroupID = &n
		}
	}
	if v := c.Query("store_id"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			storeID = &n
		}
	}

	var isActive *bool
	if v := c.Query("is_active"); v != "" {
		b := strings.EqualFold(v, "true") || v == "1"
		isActive = &b
	}

	status := c.Query("status")

	rules, total, err := h.svc.GetAll(c.Request.Context(), limit, offset, search, productID, pricingType, pricingMethod, categoryID, brandID, customerGroupID, storeID, isActive, status)
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	if rules == nil {
		rules = []PricingRule{}
	}
	shared.JSONPaginated(c, rules, total, limit, offset)
}

// GetRule godoc
// @Summary      Get pricing rule by ID
// @Description  Get a single pricing rule by its ID
// @Tags         Pricing
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Rule ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /pricing-rules/{id} [get]
func (h *Handler) GetRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule id"})
		return
	}

	rule, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pricing rule not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rule})
}

// CreateRule godoc
// @Summary      Create a pricing rule
// @Description  Create a new pricing rule. Requires at least one target (product_id, category_id, or brand_id).
// @Tags         Pricing
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      PricingRule  true  "Pricing rule data"
// @Success      201   {object}  map[string]interface{}
// @Router       /pricing-rules [post]
func (h *Handler) CreateRule(c *gin.Context) {
	var rule PricingRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.Create(c.Request.Context(), &rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.auditSvc != nil {
		userID := middleware.UserIDFromContext(c.Request.Context())
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      userID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "create",
			EntityType:  "pricing_rule",
			EntityID:    &rule.ID,
			NewValues:   shared.ToJSONMap(rule),
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Created pricing rule %s", rule.Name),
		})
	}
	c.JSON(http.StatusCreated, gin.H{"data": rule})
}

// UpdateRule godoc
// @Summary      Update a pricing rule
// @Description  Update an existing pricing rule
// @Tags         Pricing
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int           true  "Rule ID"
// @Param        body  body      PricingRule   true  "Update data"
// @Success      200   {object}  map[string]interface{}
// @Router       /pricing-rules/{id} [put]
func (h *Handler) UpdateRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule id"})
		return
	}

	var oldRule *PricingRule
	if h.auditSvc != nil {
		oldRule, _ = h.svc.GetByID(c.Request.Context(), id)
	}

	var rule PricingRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rule.ID = id

	if err := h.svc.Update(c.Request.Context(), &rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.auditSvc != nil {
		userID := middleware.UserIDFromContext(c.Request.Context())
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      userID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "update",
			EntityType:  "pricing_rule",
			EntityID:    &rule.ID,
			OldValues:   shared.ToJSONMap(oldRule),
			NewValues:   shared.ToJSONMap(rule),
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Updated pricing rule %s", rule.Name),
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": rule})
}

// DeleteRule godoc
// @Summary      Delete a pricing rule
// @Description  Delete a pricing rule by ID
// @Tags         Pricing
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Rule ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /pricing-rules/{id} [delete]
func (h *Handler) DeleteRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule id"})
		return
	}

	var oldRuleName string
	if h.auditSvc != nil {
		if r, err := h.svc.GetByID(c.Request.Context(), id); err == nil {
			oldRuleName = r.Name
		}
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		shared.InternalError(c, err)
		return
	}

	if h.auditSvc != nil {
		userID := middleware.UserIDFromContext(c.Request.Context())
		var description string
		if oldRuleName != "" {
			description = fmt.Sprintf("Deleted pricing rule %s", oldRuleName)
		} else {
			description = fmt.Sprintf("Deleted pricing rule #%d", id)
		}
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      userID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "delete",
			EntityType:  "pricing_rule",
			EntityID:    &id,
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: description,
		})
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// SubmitForApproval godoc
// @Summary      Submit rule for approval
// @Description  Transition a draft rule to pending status
// @Tags         Pricing
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Rule ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /pricing-rules/{id}/submit [post]
func (h *Handler) SubmitForApproval(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule id"})
		return
	}

	if err := h.svc.SubmitForApproval(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "pending"})
}

// ApproveRule godoc
// @Summary      Approve a pricing rule
// @Description  Transition a pending rule to approved status
// @Tags         Pricing
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Rule ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /pricing-rules/{id}/approve [post]
func (h *Handler) ApproveRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule id"})
		return
	}

	if err := h.svc.Approve(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "approved"})
}

// RejectRule godoc
// @Summary      Reject a pricing rule
// @Description  Transition a pending rule to rejected status
// @Tags         Pricing
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Rule ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /pricing-rules/{id}/reject [post]
func (h *Handler) RejectRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule id"})
		return
	}

	if err := h.svc.Reject(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "rejected"})
}

type checkConflictsRequest struct {
	ProductID       *int          `json:"product_id"`
	CategoryID      *int          `json:"category_id"`
	BrandID         *int          `json:"brand_id"`
	PricingType     PricingType   `json:"pricing_type" binding:"required"`
	PricingMethod   PricingMethod `json:"pricing_method" binding:"required"`
	PricingValue    float64       `json:"pricing_value"`
	MinimumQuantity int           `json:"minimum_quantity"`
	MaximumQuantity *int          `json:"maximum_quantity"`
	Priority        int           `json:"priority"`
	ExcludeID       int           `json:"exclude_id"`
}

// CheckConflicts godoc
// @Summary      Check for conflicting pricing rules
// @Description  Check if a proposed rule would conflict with existing active rules
// @Tags         Pricing
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      checkConflictsRequest  true  "Rule to check"
// @Success      200   {object}  map[string]interface{}
// @Router       /pricing-rules/check-conflicts [post]
func (h *Handler) CheckConflicts(c *gin.Context) {
	var req checkConflictsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.MinimumQuantity < 1 {
		req.MinimumQuantity = 1
	}

	rule := &PricingRule{
		ProductID:       req.ProductID,
		CategoryID:      req.CategoryID,
		BrandID:         req.BrandID,
		PricingType:     req.PricingType,
		PricingMethod:   req.PricingMethod,
		PricingValue:    req.PricingValue,
		MinimumQuantity: req.MinimumQuantity,
		MaximumQuantity: req.MaximumQuantity,
		Priority:        req.Priority,
	}

	conflicts, err := h.svc.FindConflictsForRule(c.Request.Context(), rule, req.ExcludeID)
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	if conflicts == nil {
		conflicts = []PricingRule{}
	}
	c.JSON(http.StatusOK, gin.H{"data": conflicts, "has_conflicts": len(conflicts) > 0})
}

type resolveRequest struct {
	Items []ResolveItem `json:"items" binding:"required,min=1"`
}

// ResolvePrices godoc
// @Summary      Resolve prices for cart items
// @Description  Resolve the effective selling price for one or more products given context (quantity, customer group, store)
// @Tags         Pricing
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      resolveRequest  true  "Items to resolve"
// @Success      200   {object}  map[string]interface{}
// @Router       /pricing/resolve [post]
func (h *Handler) ResolvePrices(c *gin.Context) {
	var req resolveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results, err := h.resolver.ResolveBatch(c.Request.Context(), req.Items)
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": results})
}

// SearchProducts godoc
// @Summary      Search products for pricing autocomplete
// @Description  Search products by name or SKU for the pricing rule form autocomplete
// @Tags         Pricing
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        q     query   string  true   "Search query"
// @Param        limit query   int     false  "Max results (1-50)"  default(10)
// @Success      200   {object}  map[string]interface{}
// @Router       /products/search [get]
func (h *Handler) SearchProducts(c *gin.Context) {
	query := c.Query("q")
	limit := 10
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}

	if h.searcher != nil {
		results, err := h.searcher.SearchProducts(c.Request.Context(), query, limit)
		if err != nil {
			shared.InternalError(c, err)
			return
		}
		if results == nil {
			results = []ProductSearchResult{}
		}
		c.JSON(http.StatusOK, gin.H{"data": results})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": []ProductSearchResult{}})
}
