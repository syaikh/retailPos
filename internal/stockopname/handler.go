package stockopname

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/shared"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc      *Service
	auditSvc audit.AuditCreator
}

func NewHandler(svc *Service, auditSvc audit.AuditCreator) *Handler {
	return &Handler{svc: svc, auditSvc: auditSvc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(string) gin.HandlerFunc) {
	r.POST("/stock-opnames", auth, perm("stock_opname.create"), h.CreateSession)
	r.GET("/stock-opnames", auth, perm("stock_opname.view"), h.ListSessions)
	r.GET("/stock-opnames/assignable-users", auth, perm("stock_opname.assign"), h.ListAssignableUsers)
	r.GET("/stock-opnames/:id", auth, perm("stock_opname.view"), h.GetSession)
	r.POST("/stock-opnames/:id/cancel", auth, perm("stock_opname.cancel"), h.CancelSession)
	r.POST("/stock-opnames/:id/assignments", auth, perm("stock_opname.assign"), h.AssignCounter)
	r.GET("/stock-opnames/:id/assignments", auth, perm("stock_opname.view"), h.GetAssignments)
	r.PUT("/stock-opnames/:id/assignments/:assignmentId", auth, perm("stock_opname.assign"), h.ReassignCounter)
	r.PUT("/stock-opnames/items/:itemId/count", auth, perm("stock_opname.count"), h.SaveCount)
	r.GET("/stock-opnames/items/:itemId/counts", auth, perm("stock_opname.view"), h.GetCountHistory)
	r.POST("/stock-opnames/:id/start", auth, perm("stock_opname.count"), h.StartCounting)
	r.POST("/stock-opnames/:id/submit", auth, perm("stock_opname.submit"), h.SubmitSession)
	r.POST("/stock-opnames/:id/approve", auth, perm("stock_opname.approve"), h.ApproveSession)
	r.POST("/stock-opnames/:id/reject", auth, perm("stock_opname.reject"), h.RejectSession)
	r.POST("/stock-opnames/:id/recount", auth, perm("stock_opname.recount"), h.RequestRecount)
	r.POST("/stock-opnames/:id/resume", auth, perm("stock_opname.count"), h.ResumeCounting)
	r.GET("/stock-opnames/:id/summary", auth, perm("stock_opname.view"), h.Summary)
	r.GET("/stock-opnames/:id/difference", auth, perm("stock_opname.view"), h.DifferenceReport)
	r.GET("/stock-opnames/:id/export", auth, perm("stock_opname.export"), h.ExportReport)
}

func (h *Handler) CreateSession(c *gin.Context) {
	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "invalid request")
		return
	}
	uid := userID(c)
	if uid == 0 {
		c.JSON(http.StatusUnauthorized, shared.NewError(shared.ErrUnauthorized, "invalid user"))
		return
	}
	session, err := h.svc.CreateSession(c.Request.Context(), &req, uid)
	if err != nil {
		writeError(c, err)
		return
	}
	h.writeAudit(c, "create", session.ID, fmt.Sprintf("Created stock opname session %s", session.SessionNumber),
		map[string]interface{}{"scope_type": session.ScopeType, "scope_id": session.ScopeID, "blind_count": session.BlindCount})
	c.JSON(http.StatusCreated, gin.H{"data": session})
}

func (h *Handler) ListSessions(c *gin.Context) {
	limit, offset := shared.ParsePaginationParams(c.Query("limit"), c.Query("offset"))
	status := c.Query("status")
	search := c.Query("search")
	sessions, total, err := h.svc.ListSessions(c.Request.Context(), limit, offset, status, search)
	if err != nil {
		writeError(c, err)
		return
	}
	shared.JSONPaginated(c, sessions, total, limit, offset)
}

func (h *Handler) ListAssignableUsers(c *gin.Context) {
	users, err := h.svc.ListAssignableUsers(c.Request.Context(), c.Query("search"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users})
}

func (h *Handler) GetSession(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, shared.NewError(shared.ErrBadRequest, "invalid id"))
		return
	}
	session, err := h.svc.GetSessionForUser(c.Request.Context(), id, userID(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": session})
}

func (h *Handler) CancelSession(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, shared.NewError(shared.ErrBadRequest, "invalid id"))
		return
	}
	uid := userID(c)
	if err := h.svc.CancelSession(c.Request.Context(), id, uid); err != nil {
		writeError(c, err)
		return
	}
	h.writeAudit(c, "cancel", id, fmt.Sprintf("Cancelled stock opname session #%d", id), nil)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) AssignCounter(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, shared.NewError(shared.ErrBadRequest, "invalid id"))
		return
	}
	var req AssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "invalid request")
		return
	}
	if err := h.svc.AssignCounter(c.Request.Context(), id, req.UserID, req.Role); err != nil {
		writeError(c, err)
		return
	}
	h.writeAudit(c, "assign", id, fmt.Sprintf("Assigned user #%d as %s on session #%d", req.UserID, req.Role, id),
		map[string]interface{}{"user_id": req.UserID, "role": req.Role})
	c.JSON(http.StatusCreated, gin.H{"status": "ok"})
}

func (h *Handler) GetAssignments(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, shared.NewError(shared.ErrBadRequest, "invalid id"))
		return
	}
	assignments, err := h.svc.GetAssignments(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": assignments})
}

func (h *Handler) ReassignCounter(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, shared.NewError(shared.ErrBadRequest, "invalid id"))
		return
	}
	assignmentID, ok := idParam(c, "assignmentId")
	if !ok {
		c.JSON(http.StatusBadRequest, shared.NewError(shared.ErrBadRequest, "invalid assignment id"))
		return
	}
	var req ReassignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "invalid request")
		return
	}
	if err := h.svc.ReassignCounter(c.Request.Context(), id, assignmentID, req.Role); err != nil {
		writeError(c, err)
		return
	}
	h.writeAudit(c, "assign", id, fmt.Sprintf("Reassigned assignment #%d on session #%d to %s", assignmentID, id, req.Role),
		map[string]interface{}{"assignment_id": assignmentID, "role": req.Role})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) SaveCount(c *gin.Context) {
	itemID, ok := idParam(c, "itemId")
	if !ok {
		c.JSON(http.StatusBadRequest, shared.NewError(shared.ErrBadRequest, "invalid item id"))
		return
	}
	var req SaveCountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "invalid request")
		return
	}
	uid := userID(c)
	if err := h.svc.SaveCount(c.Request.Context(), itemID, uid, req.PhysicalQty, req.Remarks); err != nil {
		writeError(c, err)
		return
	}
	h.writeAudit(c, "count", itemID, fmt.Sprintf("Saved count for item #%d: %.2f", itemID, req.PhysicalQty),
		map[string]interface{}{"item_id": itemID, "physical_qty": req.PhysicalQty, "remarks": req.Remarks})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) GetCountHistory(c *gin.Context) {
	itemID, ok := idParam(c, "itemId")
	if !ok {
		c.JSON(http.StatusBadRequest, shared.NewError(shared.ErrBadRequest, "invalid item id"))
		return
	}
	history, err := h.svc.GetCountHistory(c.Request.Context(), itemID)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": history})
}

func (h *Handler) StartCounting(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, shared.NewError(shared.ErrBadRequest, "invalid id"))
		return
	}
	uid := userID(c)
	if err := h.svc.StartCounting(c.Request.Context(), id, uid); err != nil {
		writeError(c, err)
		return
	}
	h.writeAudit(c, "count", id, fmt.Sprintf("Started counting for stock opname session #%d", id), nil)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) SubmitSession(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, shared.NewError(shared.ErrBadRequest, "invalid id"))
		return
	}
	uid := userID(c)
	if err := h.svc.SubmitSession(c.Request.Context(), id, uid); err != nil {
		writeError(c, err)
		return
	}
	h.writeAudit(c, "submit", id, fmt.Sprintf("Submitted stock opname session #%d", id), nil)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) ApproveSession(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, shared.NewError(shared.ErrBadRequest, "invalid id"))
		return
	}
	var req ApproveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "invalid request")
		return
	}
	uid := userID(c)
	if err := h.svc.ApproveSession(c.Request.Context(), id, uid, req.Comment); err != nil {
		writeError(c, err)
		return
	}
	h.writeAudit(c, "approve", id, fmt.Sprintf("Approved stock opname session #%d", id), map[string]interface{}{"comment": req.Comment})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) RejectSession(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, shared.NewError(shared.ErrBadRequest, "invalid id"))
		return
	}
	var req RejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "invalid request")
		return
	}
	uid := userID(c)
	if err := h.svc.RejectSession(c.Request.Context(), id, uid, req.Comment); err != nil {
		writeError(c, err)
		return
	}
	h.writeAudit(c, "reject", id, fmt.Sprintf("Rejected stock opname session #%d", id), map[string]interface{}{"comment": req.Comment})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) RequestRecount(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, shared.NewError(shared.ErrBadRequest, "invalid id"))
		return
	}
	var req RecountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "invalid request")
		return
	}
	uid := userID(c)
	if err := h.svc.RequestRecount(c.Request.Context(), id, uid, req.Comment); err != nil {
		writeError(c, err)
		return
	}
	h.writeAudit(c, "recount", id, fmt.Sprintf("Requested recount for stock opname session #%d", id), map[string]interface{}{"comment": req.Comment})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) ResumeCounting(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, shared.NewError(shared.ErrBadRequest, "invalid id"))
		return
	}
	uid := userID(c)
	if err := h.svc.ResumeCounting(c.Request.Context(), id, uid); err != nil {
		writeError(c, err)
		return
	}
	h.writeAudit(c, "recount", id, fmt.Sprintf("Resumed counting for stock opname session #%d", id), nil)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) Summary(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, shared.NewError(shared.ErrBadRequest, "invalid id"))
		return
	}
	summary, err := h.svc.Summary(c.Request.Context(), id, userID(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": summary})
}

func (h *Handler) DifferenceReport(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, shared.NewError(shared.ErrBadRequest, "invalid id"))
		return
	}
	session, err := h.svc.DifferenceReport(c.Request.Context(), id, userID(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": session})
}

func (h *Handler) ExportReport(c *gin.Context) {
	id, ok := idParam(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, shared.NewError(shared.ErrBadRequest, "invalid id"))
		return
	}
	session, err := h.svc.DifferenceReport(c.Request.Context(), id, userID(c))
	if err != nil {
		writeError(c, err)
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="stock-opname-%s.csv"`, session.SessionNumber))
	w := csv.NewWriter(c.Writer)
	defer w.Flush()
	_ = shared.WriteCSVRow(w, []string{"session_number", "product_id", "sku", "barcode", "product_name", "opening_qty", "expected_qty", "physical_qty", "difference_qty", "adjustment_qty", "status"})
	for _, it := range session.Items {
		_ = shared.WriteCSVRow(w, []string{
			session.SessionNumber,
			strconv.Itoa(it.ProductID),
			it.SKU,
			it.Barcode,
			it.ProductName,
			formatQty(it.OpeningQty),
			formatQty(it.ExpectedQty),
			formatQty(it.PhysicalQty),
			formatQty(it.DifferenceQty),
			formatQty(it.AdjustmentQty),
			it.Status,
		})
	}
}

// --- helpers ---

func (h *Handler) writeAudit(c *gin.Context, action string, entityID int, description string, extra map[string]interface{}) {
	if h.auditSvc == nil {
		return
	}
	ctx := c.Request.Context()
	_ = h.auditSvc.CreateAuditLog(ctx, &audit.AuditLog{
		UserID:      middleware.UserIDFromContext(ctx),
		Username:    middleware.UsernameFromContext(ctx),
		Role:        middleware.RoleFromContext(ctx),
		Action:      action,
		EntityType:  "stock_opname",
		EntityID:    &entityID,
		NewValues:   shared.ToJSONMap(extra),
		IPAddress:   middleware.IPAddressFromContext(ctx),
		UserAgent:   middleware.UserAgentFromContext(ctx),
		Description: description,
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

func formatQty(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func writeError(c *gin.Context, err error) {
	if err == nil {
		shared.InternalError(c, errors.New("unexpected error"))
		return
	}
	status := http.StatusInternalServerError
	code := "SO-501"
	switch {
	case errors.Is(err, ErrNotFound):
		status, code = http.StatusNotFound, "SO-002"
	case errors.Is(err, ErrActiveSessionExists):
		status, code = http.StatusConflict, "SO-001"
	case errors.Is(err, ErrInvalidState):
		status, code = http.StatusConflict, "SO-003"
	case errors.Is(err, ErrSessionLocked):
		status, code = http.StatusForbidden, "SO-004"
	case errors.Is(err, ErrAlreadySubmitted), errors.Is(err, ErrAlreadyApproved):
		status, code = http.StatusConflict, "SO-203"
	case errors.Is(err, ErrNotAllItemsCounted):
		status, code = http.StatusConflict, "SO-201"
	case errors.Is(err, ErrNotAssigned):
		status, code = http.StatusForbidden, "SO-105"
	case errors.Is(err, ErrSeparationOfDuties):
		status, code = http.StatusForbidden, "SO-303"
	case errors.Is(err, ErrItemNotFound), errors.Is(err, ErrAssignmentNotFound):
		status, code = http.StatusNotFound, "SO-103"
	case errors.Is(err, ErrInvalidQuantity):
		status, code = http.StatusBadRequest, "SO-102"
	case errors.Is(err, ErrInvalidAssigneeRole), errors.Is(err, ErrAssigneeNotFound):
		status, code = http.StatusBadRequest, "SO-104"
	case errors.Is(err, ErrApprovalCommentReq):
		status, code = http.StatusUnprocessableEntity, "SO-402"
	case errors.Is(err, ErrUnsupportedScope):
		status, code = http.StatusBadRequest, "SO-401"
	case errors.Is(err, ErrAdjustmentFailed):
		status, code = http.StatusInternalServerError, "SO-205"
	default:
		shared.LogError(context.Background(), "stock opname error", err)
	}
	shared.JSONError(c, status, code, err.Error())
}
