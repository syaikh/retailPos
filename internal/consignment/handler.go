package consignment

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/permissions"
	"retail-pos-system/internal/shared"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc      *Service
	auditSvc audit.Creator
}

func NewHandler(svc *Service, auditSvc audit.Creator) *Handler {
	return &Handler{svc: svc, auditSvc: auditSvc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(permissions.Code) gin.HandlerFunc) {
	r.GET("/consignment/suppliers", auth, perm(permissions.ConsignmentView), h.ListSuppliers)
	r.GET("/consignment/arrangements", auth, perm(permissions.ConsignmentView), h.ListArrangements)
	r.POST("/consignment/arrangements", auth, perm(permissions.ConsignmentCreate), h.CreateArrangement)
	r.GET("/consignment/arrangements/:id", auth, perm(permissions.ConsignmentView), h.GetArrangement)
	r.PUT("/consignment/arrangements/:id/terms", auth, perm(permissions.ConsignmentUpdate), h.SetTerms)
	r.GET("/consignment/receipts", auth, perm(permissions.ConsignmentView), h.ListReceipts)
	r.POST("/consignment/receipts", auth, perm(permissions.ConsignmentCreate), h.CreateReceipt)
	r.GET("/consignment/receipts/:id", auth, perm(permissions.ConsignmentView), h.GetReceipt)
	r.GET("/consignment/stock", auth, perm(permissions.ConsignmentView), h.ListStock)
	r.GET("/consignment/pending-returns", auth, perm(permissions.ConsignmentView), h.ListPendingReturns)
	r.POST("/consignment/pending-returns", auth, perm(permissions.ConsignmentUpdate), h.CreatePendingReturn)
	r.GET("/consignment/returns", auth, perm(permissions.ConsignmentView), h.ListReturns)
	r.POST("/consignment/returns", auth, perm(permissions.ConsignmentCreate), h.CreateReturn)
	r.GET("/consignment/returns/:id", auth, perm(permissions.ConsignmentView), h.GetReturn)
	r.GET("/consignment/settlements/preview", auth, perm(permissions.ConsignmentSettle), h.GetSettlementPreview)
	r.GET("/consignment/settlements", auth, perm(permissions.ConsignmentView), h.ListSettlements)
	r.POST("/consignment/settlements", auth, perm(permissions.ConsignmentSettle), h.CreateSettlement)
	r.GET("/consignment/settlements/:id", auth, perm(permissions.ConsignmentView), h.GetSettlement)
	r.GET("/consignment/payment-methods", auth, perm(permissions.ConsignmentSettle), h.ListPaymentMethods)
	r.POST("/consignment/settlements/:id/payouts", auth, perm(permissions.ConsignmentPay), h.CreatePayout)
}

func (h *Handler) ListSuppliers(c *gin.Context) {
	ctx := c.Request.Context()
	suppliers, err := h.svc.ListSuppliers(ctx)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": suppliers})
}

func (h *Handler) CreateArrangement(c *gin.Context) {
	var req CreateArrangementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "invalid request")
		return
	}
	uid := userID(c)
	if uid == 0 {
		c.JSON(http.StatusUnauthorized, shared.NewError(shared.ErrUnauthorized, "invalid user"))
		return
	}
	a, err := h.svc.CreateArrangement(c.Request.Context(), &req, uid, shared.GetStoreID(c))
	if err != nil {
		writeError(c, err)
		return
	}
	h.writeAudit(c, "create_arrangement", a.ID, fmt.Sprintf("Created consignment arrangement with supplier %d", a.SupplierID), nil)
	c.JSON(http.StatusCreated, gin.H{"data": a})
}

func (h *Handler) ListArrangements(c *gin.Context) {
	limit, offset := shared.ParsePaginationParams(c.Query("limit"), c.Query("offset"))
	arrangements, total, err := h.svc.ListArrangements(c.Request.Context(), shared.GetStoreID(c), limit, offset, c.Query("search"), c.Query("status"))
	if err != nil {
		writeError(c, err)
		return
	}
	if arrangements == nil {
		arrangements = []Arrangement{}
	}
	shared.JSONPaginated(c, arrangements, total, limit, offset)
}

func (h *Handler) GetArrangement(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	a, err := h.svc.GetArrangement(c.Request.Context(), id, shared.GetStoreID(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": a})
}

func (h *Handler) SetTerms(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req []SetTermsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "invalid request")
		return
	}
	uid := userID(c)
	if uid == 0 {
		c.JSON(http.StatusUnauthorized, shared.NewError(shared.ErrUnauthorized, "invalid user"))
		return
	}
	terms, err := h.svc.SetTerms(c.Request.Context(), id, req, uid, shared.GetStoreID(c))
	if err != nil {
		writeError(c, err)
		return
	}
	h.writeAudit(c, "set_terms", id, fmt.Sprintf("Updated %d consignment terms", len(terms)), nil)
	c.JSON(http.StatusOK, gin.H{"data": terms})
}

func (h *Handler) CreateReceipt(c *gin.Context) {
	var req ReceiptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "invalid request")
		return
	}
	uid := userID(c)
	if uid == 0 {
		c.JSON(http.StatusUnauthorized, shared.NewError(shared.ErrUnauthorized, "invalid user"))
		return
	}
	rec, err := h.svc.CreateReceipt(c.Request.Context(), &req, uid, shared.GetStoreID(c))
	if err != nil {
		writeError(c, err)
		return
	}
	h.writeAudit(c, "create_receipt", rec.ID, fmt.Sprintf("Created consignment receipt %s", rec.ReceiptNumber), nil)
	c.JSON(http.StatusCreated, gin.H{"data": rec})
}

func (h *Handler) GetReceipt(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	rec, err := h.svc.GetReceipt(c.Request.Context(), id, shared.GetStoreID(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rec})
}

func (h *Handler) ListReceipts(c *gin.Context) {
	supplierID := queryInt(c, "supplier_id")
	if supplierID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "supplier_id required"})
		return
	}
	receipts, err := h.svc.ListReceipts(c.Request.Context(), supplierID, shared.GetStoreID(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": receipts})
}

func (h *Handler) ListStock(c *gin.Context) {
	supplierID := queryInt(c, "supplier_id")
	stock, err := h.svc.ListStock(c.Request.Context(), supplierID, shared.GetStoreID(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": stock})
}

func (h *Handler) CreatePendingReturn(c *gin.Context) {
	var req CreatePendingReturnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "invalid request")
		return
	}
	uid := userID(c)
	if uid == 0 {
		c.JSON(http.StatusUnauthorized, shared.NewError(shared.ErrUnauthorized, "invalid user"))
		return
	}
	pr, err := h.svc.CreatePendingReturn(c.Request.Context(), &req, uid, shared.GetStoreID(c))
	if err != nil {
		writeError(c, err)
		return
	}
	h.writeAudit(c, "create_pending_return", pr.ID, fmt.Sprintf("Created pending return for product %d", pr.ProductID), nil)
	c.JSON(http.StatusCreated, gin.H{"data": pr})
}

func (h *Handler) ListPendingReturns(c *gin.Context) {
	supplierID := queryInt(c, "supplier_id")
	if supplierID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "supplier_id required"})
		return
	}
	prs, err := h.svc.ListPendingReturns(c.Request.Context(), supplierID, shared.GetStoreID(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": prs})
}

func (h *Handler) CreateReturn(c *gin.Context) {
	var req ReturnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "invalid request")
		return
	}
	uid := userID(c)
	if uid == 0 {
		c.JSON(http.StatusUnauthorized, shared.NewError(shared.ErrUnauthorized, "invalid user"))
		return
	}
	ret, err := h.svc.CreateReturn(c.Request.Context(), &req, uid, shared.GetStoreID(c))
	if err != nil {
		writeError(c, err)
		return
	}
	h.writeAudit(c, "create_return", ret.ID, fmt.Sprintf("Created consignment return %s", ret.ReturnNumber), nil)
	c.JSON(http.StatusCreated, gin.H{"data": ret})
}

func (h *Handler) GetReturn(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ret, err := h.svc.GetReturn(c.Request.Context(), id, shared.GetStoreID(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": ret})
}

func (h *Handler) ListReturns(c *gin.Context) {
	supplierID := queryInt(c, "supplier_id")
	if supplierID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "supplier_id required"})
		return
	}
	returns, err := h.svc.ListReturns(c.Request.Context(), supplierID, shared.GetStoreID(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": returns})
}

func (h *Handler) GetSettlementPreview(c *gin.Context) {
	supplierID := queryInt(c, "supplier_id")
	if supplierID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "supplier_id required"})
		return
	}
	st, err := h.svc.GetSettlementPreview(c.Request.Context(), supplierID, shared.GetStoreID(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": st})
}

func (h *Handler) CreateSettlement(c *gin.Context) {
	var req CreateSettlementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "invalid request")
		return
	}
	uid := userID(c)
	if uid == 0 {
		c.JSON(http.StatusUnauthorized, shared.NewError(shared.ErrUnauthorized, "invalid user"))
		return
	}
	st, err := h.svc.CreateSettlement(c.Request.Context(), &req, uid, shared.GetStoreID(c))
	if err != nil {
		writeError(c, err)
		return
	}
	h.writeAudit(c, "create_settlement", st.ID, fmt.Sprintf("Created consignment settlement %s", st.SettlementNumber), nil)
	c.JSON(http.StatusCreated, gin.H{"data": st})
}

func (h *Handler) GetSettlement(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	st, err := h.svc.GetSettlement(c.Request.Context(), id, shared.GetStoreID(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": st})
}

func (h *Handler) ListSettlements(c *gin.Context) {
	supplierID := queryInt(c, "supplier_id")
	if supplierID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "supplier_id required"})
		return
	}
	settlements, err := h.svc.ListSettlements(c.Request.Context(), supplierID, shared.GetStoreID(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": settlements})
}

func (h *Handler) ListPaymentMethods(c *gin.Context) {
	methods, err := h.svc.ListPaymentMethods(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": methods})
}

func (h *Handler) CreatePayout(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req CreatePayoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "invalid request")
		return
	}
	uid := userID(c)
	if uid == 0 {
		c.JSON(http.StatusUnauthorized, shared.NewError(shared.ErrUnauthorized, "invalid user"))
		return
	}
	payout, err := h.svc.CreatePayout(c.Request.Context(), id, &req, uid, shared.GetStoreID(c))
	if err != nil {
		writeError(c, err)
		return
	}
	h.writeAudit(c, "create_payout", id, fmt.Sprintf("Created consignment payout %s", payout.PayoutNumber), nil)
	c.JSON(http.StatusCreated, gin.H{"data": payout})
}

// --- helpers ---

func (h *Handler) writeAudit(c *gin.Context, action string, entityID int, description string, extra map[string]interface{}) {
	if h.auditSvc == nil {
		return
	}
	ctx := c.Request.Context()
	_ = h.auditSvc.CreateAuditLog(ctx, &audit.Log{
		UserID:      middleware.UserIDFromContext(ctx),
		Username:    middleware.UsernameFromContext(ctx),
		Role:        middleware.RoleFromContext(ctx),
		Action:      action,
		EntityType:  "consignment",
		EntityID:    &entityID,
		NewValues:   shared.ToJSONMap(extra),
		IPAddress:   middleware.IPAddressFromContext(ctx),
		UserAgent:   middleware.UserAgentFromContext(ctx),
		Description: description,
		StoreID:     middleware.StoreIDFromContext(ctx),
	})
}

func userID(c *gin.Context) int {
	if id := middleware.UserIDFromContext(c.Request.Context()); id != nil {
		return *id
	}
	return 0
}

func idParam(c *gin.Context, name string) (int, bool) {
	id, err := strconv.Atoi(c.Param(name))
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func queryInt(c *gin.Context, name string) int {
	v, err := strconv.Atoi(c.Query(name))
	if err != nil {
		return 0
	}
	return v
}

func writeError(c *gin.Context, err error) {
	if err == nil {
		shared.InternalError(c, errors.New("unexpected error"))
		return
	}
	var status, code = http.StatusInternalServerError, "CNS-500"
	switch {
	case errors.Is(err, ErrConsignmentNotFound),
		errors.Is(err, ErrReturnNotFound),
		errors.Is(err, ErrPendingReturnNotFound),
		errors.Is(err, ErrSettlementNotFound):
		status, code = http.StatusNotFound, "CNS-103"
	case errors.Is(err, ErrNotConsignmentSupplier),
		errors.Is(err, ErrInvalidShareType),
		errors.Is(err, ErrInvalidShareValue),
		errors.Is(err, ErrInvalidShareValueForType),
		errors.Is(err, ErrInvalidPrice),
		errors.Is(err, ErrFixedShareExceedsPrice),
		errors.Is(err, ErrDuplicateProduct),
		errors.Is(err, ErrInvalidQty),
		errors.Is(err, ErrInvalidReason),
		errors.Is(err, ErrPaymentMethodNotFound),
		errors.Is(err, ErrEmptySettlement):
		status, code = http.StatusBadRequest, "CNS-102"
	case errors.Is(err, ErrActiveArrangementExists),
		errors.Is(err, ErrConflictStoreStock),
		errors.Is(err, ErrConflictOtherSupplier),
		errors.Is(err, ErrPendingReturnBlocksTransfer),
		errors.Is(err, ErrSettlementAlreadyPaid):
		status, code = http.StatusConflict, "CNS-201"
	case errors.Is(err, ErrInsufficientConsignmentStock),
		errors.Is(err, ErrInvalidPayoutAmount):
		status, code = http.StatusUnprocessableEntity, "CNS-402"
	case errors.Is(err, ErrArrangementEnded):
		status, code = http.StatusBadRequest, "CNS-102"
	case errors.Is(err, ErrStoreForbidden):
		status, code = http.StatusForbidden, "CNS-303"
	default:
		shared.LogError(context.Background(), "consignment error", err)
	}
	shared.JSONError(c, status, code, err.Error())
}
