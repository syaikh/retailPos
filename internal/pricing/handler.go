package pricing

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

type PricingService interface {
	GetByID(ctx context.Context, id int) (*PricingRule, error)
	GetByProductID(ctx context.Context, productID int) ([]PricingRule, error)
	GetAll(ctx context.Context, limit, offset int, search string, productID *int, pricingType string, isActive *bool) ([]PricingRule, int, error)
	Create(ctx context.Context, rule *PricingRule) error
	Update(ctx context.Context, rule *PricingRule) error
	Delete(ctx context.Context, id int) error
}

type Handler struct {
	svc      PricingService
	resolver PriceResolver
	auditSvc audit.AuditCreator
}

func NewHandler(svc PricingService, resolver PriceResolver, auditSvc audit.AuditCreator) *Handler {
	return &Handler{svc: svc, resolver: resolver, auditSvc: auditSvc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(string) gin.HandlerFunc) {
	r.GET("/pricing-rules", auth, perm("pricing:read"), h.ListRules)
	r.GET("/pricing-rules/:id", auth, perm("pricing:read"), h.GetRule)
	r.POST("/pricing-rules", auth, perm("pricing:create"), h.CreateRule)
	r.PUT("/pricing-rules/:id", auth, perm("pricing:update"), h.UpdateRule)
	r.DELETE("/pricing-rules/:id", auth, perm("pricing:delete"), h.DeleteRule)
	r.POST("/pricing/resolve", auth, perm("pricing:read"), h.ResolvePrices)
}

// ListRules godoc
// @Summary List pricing rules
// @Description Get paginated list of pricing rules with optional filters
// @Tags pricing
// @Accept json
// @Produce json
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Param search query string false "Search by name or pricing type"
// @Param product_id query int false "Filter by product ID"
// @Param pricing_type query string false "Filter by pricing type (discount, wholesale, member, promotion)"
// @Param is_active query string false "Filter by active status (true/false)"
// @Success 200 {object} map[string]interface{}
// @Router /pricing-rules [get]
func (h *Handler) ListRules(c *gin.Context) {
	limit, offset := shared.ParsePaginationParams(c.Query("limit"), c.Query("offset"))
	search := c.Query("search")

	var productID *int
	if v := c.Query("product_id"); v != "" {
		if id, err := strconv.Atoi(v); err == nil {
			productID = &id
		}
	}

	var pricingType string
	if v := c.Query("pricing_type"); v != "" {
		pricingType = strings.ToLower(v)
	}

	var isActive *bool
	if v := c.Query("is_active"); v != "" {
		b := strings.EqualFold(v, "true") || v == "1"
		isActive = &b
	}

	rules, total, err := h.svc.GetAll(c.Request.Context(), limit, offset, search, productID, pricingType, isActive)
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
// @Summary Get a pricing rule by ID
// @Description Get a single pricing rule by its ID
// @Tags pricing
// @Accept json
// @Produce json
// @Param id path int true "Pricing Rule ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /pricing-rules/{id} [get]
func (h *Handler) GetRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pricing rule id"})
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
// @Summary Create a new pricing rule
// @Description Create a new pricing rule for a product
// @Tags pricing
// @Accept json
// @Produce json
// @Param rule body PricingRule true "Pricing Rule"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /pricing-rules [post]
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
// @Summary Update a pricing rule
// @Description Update an existing pricing rule by ID
// @Tags pricing
// @Accept json
// @Produce json
// @Param id path int true "Pricing Rule ID"
// @Param rule body PricingRule true "Pricing Rule"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /pricing-rules/{id} [put]
func (h *Handler) UpdateRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pricing rule id"})
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
// @Summary Delete a pricing rule
// @Description Delete a pricing rule by ID
// @Tags pricing
// @Accept json
// @Produce json
// @Param id path int true "Pricing Rule ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /pricing-rules/{id} [delete]
func (h *Handler) DeleteRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pricing rule id"})
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

// ResolvePrices godoc
// @Summary Resolve prices for cart items
// @Description Batch resolve effective selling prices for products using pricing rules
// @Tags pricing
// @Accept json
// @Produce json
// @Param items body object true "Items to resolve"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /pricing/resolve [post]
func (h *Handler) ResolvePrices(c *gin.Context) {
	var request struct {
		Items []ResolveItem `json:"items" binding:"required,dive"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.resolver == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "pricing resolver not available"})
		return
	}

	results, err := h.resolver.ResolveBatch(c.Request.Context(), request.Items)
	if err != nil {
		shared.InternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": results})
}
