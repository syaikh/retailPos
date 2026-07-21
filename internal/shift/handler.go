package shift

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/shared"
)

type ShiftService interface {
	OpenShift(ctx context.Context, userID int, storeID *int, openingBalance int) (*Shift, error)
	CloseShift(ctx context.Context, shiftID, userID int, closingBalance int, notes *string) (*Shift, error)
	GetActiveShift(ctx context.Context, userID int) (*Shift, error)
	ListShifts(ctx context.Context, userID *int, status string, limit, offset int, sortBy, sortDir string) ([]Shift, int, error)
	GetShiftByID(ctx context.Context, shiftID int) (*Shift, error)
	ReviewShift(ctx context.Context, shiftID, reviewerID int) (*Shift, error)
}

type Handler struct {
	svc      ShiftService
	auditSvc audit.AuditCreator
}

func NewHandler(svc ShiftService, auditSvc audit.AuditCreator) *Handler {
	return &Handler{svc: svc, auditSvc: auditSvc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(string) gin.HandlerFunc) {
	r.POST("/shifts/open", auth, perm("shift:create"), h.OpenShift)
	r.POST("/shifts/:id/close", auth, perm("shift:create"), h.CloseShift)
	r.POST("/shifts/:id/review", auth, perm("shift:review"), h.ReviewShift)
	r.GET("/shifts/active", auth, h.GetActiveShift)
	r.GET("/shifts", auth, perm("shift:read"), h.ListShifts)
	r.GET("/shifts/:id", auth, perm("shift:read"), h.GetShiftByID)
}

func (h *Handler) OpenShift(c *gin.Context) {
	var req struct {
		StoreID        *int `json:"store_id"`
		OpeningBalance int   `json:"opening_balance"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	userID, _ := c.Get("userID")
	uid, ok := userID.(int)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	shift, err := h.svc.OpenShift(c.Request.Context(), uid, req.StoreID, req.OpeningBalance)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.auditSvc != nil {
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      middleware.UserIDFromContext(c.Request.Context()),
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "create",
			EntityType:  "shift",
			EntityID:    &shift.ID,
			NewValues:   shared.ToJSONMap(map[string]interface{}{"opening_balance": req.OpeningBalance, "store_id": req.StoreID}),
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: "Opened shift",
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": shift})
}

func (h *Handler) CloseShift(c *gin.Context) {
	shiftID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid shift id"})
		return
	}

	var req struct {
		ClosingBalance int     `json:"closing_balance"`
		Notes          *string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	userID, _ := c.Get("userID")
	uid, ok := userID.(int)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	shift, err := h.svc.CloseShift(c.Request.Context(), shiftID, uid, req.ClosingBalance, req.Notes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.auditSvc != nil {
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      middleware.UserIDFromContext(c.Request.Context()),
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "update",
			EntityType:  "shift",
			EntityID:    &shift.ID,
			NewValues:   shared.ToJSONMap(map[string]interface{}{"closing_balance": req.ClosingBalance, "notes": req.Notes}),
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: "Closed shift",
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": shift})
}

func (h *Handler) GetActiveShift(c *gin.Context) {
	userID, _ := c.Get("userID")
	uid, ok := userID.(int)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	shift, err := h.svc.GetActiveShift(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active shift found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": shift})
}

func (h *Handler) ListShifts(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")
	sortBy := c.DefaultQuery("sort_by", "opened_at")
	sortDir := c.DefaultQuery("sort_dir", "DESC")
	status := c.Query("status")

	limit, offset := shared.ParsePaginationParams(limitStr, offsetStr)

	var userID *int
	if uidStr := c.Query("user_id"); uidStr != "" {
		if uid, err := strconv.Atoi(uidStr); err == nil {
			userID = &uid
		}
	}

	shifts, total, err := h.svc.ListShifts(c.Request.Context(), userID, status, limit, offset, sortBy, sortDir)
	if err != nil {
		shared.InternalError(c, err)
		return
	}

	shared.JSONPaginated(c, shifts, total, limit, offset)
}

func (h *Handler) GetShiftByID(c *gin.Context) {
	shiftID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid shift id"})
		return
	}

	shift, err := h.svc.GetShiftByID(c.Request.Context(), shiftID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "shift not found"})
		return
	}

	shared.JSONSuccess(c, shift)
}

func (h *Handler) ReviewShift(c *gin.Context) {
	shiftID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid shift id"})
		return
	}

	userID, _ := c.Get("userID")
	reviewerID, ok := userID.(int)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	shift, err := h.svc.ReviewShift(c.Request.Context(), shiftID, reviewerID)
	if err != nil {
		shared.InternalError(c, err)
		return
	}

	if h.auditSvc != nil {
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      middleware.UserIDFromContext(c.Request.Context()),
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "review",
			EntityType:  "shift",
			EntityID:    &shift.ID,
			NewValues:   shared.ToJSONMap(map[string]interface{}{"needs_review": false, "reviewed_by": reviewerID}),
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: "Reviewed shift discrepancy",
		})
	}

	shared.JSONSuccess(c, shift)
}
