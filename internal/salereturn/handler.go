package salereturn

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/permissions"
	"retail-pos-system/internal/shared"
)

// Handler exposes the Sales Return endpoints (PRD §9). Route shape and
// permissions:
//
//	GET  /sales/:id/returnable          sale.return        pre-flight snapshot for the return modal
//	POST /sales/:id/return              sale.return        create/process a return
//	GET  /sales/:id/returns             sale.view          returns of one sale
//	GET  /sales/returns/pending         sale.return.approve pending approval queue
//	GET  /sales/returns/:returnId       sale.view          return detail
//	POST /sales/returns/:returnId/approve  sale.return.approve
//	POST /sales/returns/:returnId/reject   sale.return.approve
//
// "/sales/returns/..." is registered as a static sibling of the sale module's
// "/sales/:id" — gin's tree supports static + wildcard siblings (same
// precedent as /sales/lookup, /sales/parked and /shifts/active).
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(permissions.Code) gin.HandlerFunc) {
	r.GET("/sales/:id/returnable", auth, perm(permissions.SaleReturn), h.GetReturnable)
	r.POST("/sales/:id/return", auth, perm(permissions.SaleReturn), h.ProcessReturn)
	r.GET("/sales/:id/returns", auth, perm(permissions.SaleView), h.ListSaleReturns)
	r.GET("/sales/returns/pending", auth, perm(permissions.SaleReturnApprove), h.ListPendingReturns)
	r.GET("/sales/returns/:returnId", auth, perm(permissions.SaleView), h.GetReturn)
	r.POST("/sales/returns/:returnId/approve", auth, perm(permissions.SaleReturnApprove), h.ApproveReturn)
	r.POST("/sales/returns/:returnId/reject", auth, perm(permissions.SaleReturnApprove), h.RejectReturn)
}

// actorFrom assembles the audit identity (PRD §13) from the auth context.
func actorFrom(c *gin.Context) Actor {
	ctx := c.Request.Context()
	return Actor{
		ID:        middleware.GetUserID(c),
		Username:  middleware.UsernameFromContext(ctx),
		Role:      middleware.RoleFromContext(ctx),
		StoreID:   middleware.StoreIDFromContext(ctx),
		IPAddress: middleware.IPAddressFromContext(ctx),
		UserAgent: middleware.UserAgentFromContext(ctx),
	}
}

type itemBody struct {
	SaleItemID int     `json:"sale_item_id"`
	Quantity   int     `json:"quantity"`
	Notes      *string `json:"notes"`
}

type processBody struct {
	Reason    string     `json:"reason"`
	Condition string     `json:"condition"`
	Notes     *string    `json:"notes"`
	Items     []itemBody `json:"items"`
}

type approveBody struct {
	Note *string `json:"note"`
}

type rejectBody struct {
	Reason string `json:"reason"`
}

// GetReturnable returns the pre-flight snapshot for the return modal
// (GET /sales/:id/returnable): sale identity, server-authoritative
// eligibility, policy values, and per-line returnable quantity/stock.
func (h *Handler) GetReturnable(c *gin.Context) {
	saleID, ok := pathID(c, "id")
	if !ok {
		respondError(c, ErrInvalidReturn)
		return
	}
	out, err := h.svc.GetReturnable(c.Request.Context(), actorFrom(c), shared.GetStoreID(c), saleID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

// ProcessReturn creates a return for a sale (POST /sales/:id/return).
func (h *Handler) ProcessReturn(c *gin.Context) {
	saleID, ok := pathID(c, "id")
	if !ok {
		respondError(c, ErrInvalidReturn)
		return
	}
	var body processBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	req := ProcessRequest{Reason: body.Reason, Condition: body.Condition, Notes: body.Notes}
	for _, it := range body.Items {
		req.Items = append(req.Items, ItemRequest(it))
	}

	ret, err := h.svc.ProcessReturn(c.Request.Context(), actorFrom(c), shared.GetStoreID(c), saleID, req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": presentReturn(ret)})
}

// ApproveReturn completes a pending return (POST /sales/returns/:returnId/approve).
func (h *Handler) ApproveReturn(c *gin.Context) {
	returnID, ok := pathID(c, "returnId")
	if !ok {
		respondError(c, ErrReturnNotFound)
		return
	}
	var body approveBody
	if err := c.ShouldBindJSON(&body); err != nil {
		body = approveBody{}
	}
	ret, err := h.svc.ApproveReturn(c.Request.Context(), actorFrom(c), shared.GetStoreID(c), returnID, body.Note)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": presentReturn(ret)})
}

// RejectReturn cancels a pending return (POST /sales/returns/:returnId/reject).
func (h *Handler) RejectReturn(c *gin.Context) {
	returnID, ok := pathID(c, "returnId")
	if !ok {
		respondError(c, ErrReturnNotFound)
		return
	}
	var body rejectBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	ret, err := h.svc.RejectReturn(c.Request.Context(), actorFrom(c), shared.GetStoreID(c), returnID, body.Reason)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": presentReturn(ret)})
}

// GetReturn returns one return's detail (GET /sales/returns/:returnId).
func (h *Handler) GetReturn(c *gin.Context) {
	returnID, ok := pathID(c, "returnId")
	if !ok {
		respondError(c, ErrReturnNotFound)
		return
	}
	ret, err := h.svc.GetReturn(c.Request.Context(), shared.GetStoreID(c), returnID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": presentReturn(ret)})
}

// ListSaleReturns lists the returns of one sale (GET /sales/:id/returns).
func (h *Handler) ListSaleReturns(c *gin.Context) {
	saleID, ok := pathID(c, "id")
	if !ok {
		respondError(c, ErrInvalidReturn)
		return
	}
	limit, offset, err := shared.ParsePaginationParams(c.Query("limit"), c.Query("offset"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	status := optionalStatus(c.Query("status"))

	returns, total, err := h.svc.ListReturns(c.Request.Context(), shared.GetStoreID(c), &saleID, status, limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, shared.NewPaginatedResponse(presentReturns(returns), total, limit, offset))
}

// ListPendingReturns lists actionable pending approvals, oldest first
// (GET /sales/returns/pending).
func (h *Handler) ListPendingReturns(c *gin.Context) {
	limit, offset, err := shared.ParsePaginationParams(c.Query("limit"), c.Query("offset"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	returns, total, err := h.svc.ListPending(c.Request.Context(), shared.GetStoreID(c), limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, shared.NewPaginatedResponse(presentReturns(returns), total, limit, offset))
}

// ==================== HELPERS ====================

func pathID(c *gin.Context, param string) (int, bool) {
	id, err := strconv.Atoi(c.Param(param))
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func optionalStatus(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// respondError maps domain errors to HTTP status codes: missing/scope-hidden
// → 404, state conflicts → 409, everything else → 400 with the domain message.
func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrSaleNotFound), errors.Is(err, ErrReturnNotFound), errors.Is(err, ErrStoreMismatch):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, ErrReturnNotPending), errors.Is(err, ErrApprovalExpired), errors.Is(err, ErrShiftNotOpen), errors.Is(err, ErrStockChanged):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}
