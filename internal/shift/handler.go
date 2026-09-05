package shift

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/xuri/excelize/v2"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/config"
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/ownership"
	"retail-pos-system/internal/permissions"
	"retail-pos-system/internal/shared"
)

type Service interface {
	OpenShift(ctx context.Context, userID int, storeID *int, openingBalance int) (*Shift, error)
	OpenShiftTx(ctx context.Context, tx pgx.Tx, userID int, storeID *int, openingBalance int) (*Shift, error)
	CloseShift(ctx context.Context, shiftID, userID int, closingBalance int, notes *string) (*Shift, error)
	CloseShiftTx(ctx context.Context, tx pgx.Tx, shiftID, userID int, closingBalance int, notes *string) (*Shift, error)
	GetActiveShift(ctx context.Context, userID int) (*Shift, error)
	ListShifts(ctx context.Context, scope ownership.Scope, status string, needsReview *bool, discrepancyFilter string, limit, offset int, sortBy, sortDir string) ([]Shift, int, error)
	GetShiftByID(ctx context.Context, scope ownership.Scope, shiftID int) (*Shift, error)
	ReviewShift(ctx context.Context, shiftID, reviewerID int) (*Shift, error)
	FlagForReview(ctx context.Context, shiftID int) error
	GetDiscrepancyThreshold(ctx context.Context) int
	SetSettingsProvider(p SettingsProvider)
	AuditShift(ctx context.Context, shiftID int) (*Shift, int, error)
	ExportShifts(ctx context.Context, scope ownership.Scope, status string, needsReview *bool, discrepancyFilter string) ([]Shift, error)
	CreateCashMovement(ctx context.Context, shiftID, userID int, movementType string, amount int, description *string) (*CashMovement, error)
	CreateCashMovementTx(ctx context.Context, tx pgx.Tx, shiftID, userID int, movementType string, amount int, description *string) (*CashMovement, error)
	ListCashMovements(ctx context.Context, shiftID int) ([]CashMovement, error)
	ShiftCashMovementSummary(ctx context.Context, tx pgx.Tx, shiftID int) (CashMovementSummary, error)
	GetShiftReportData(ctx context.Context, shiftID int) (*ShiftReportData, error)
	InTx(ctx context.Context, fn func(tx pgx.Tx) error) error
}

type Handler struct {
	svc      Service
	auditSvc audit.TxCreator
}

func NewHandler(svc Service, auditSvc audit.TxCreator) *Handler {
	return &Handler{svc: svc, auditSvc: auditSvc}
}

// shiftScope resolves the row-level visibility for shift reads.
//
// Callers holding shift.review (superadmin/admin/manager) may see all shifts;
// everyone else is clamped to their own shifts. requestedUserID is the
// optional user_id filter from the request — honored only for all-access
// callers, never used to widen a restricted caller's view.
func (h *Handler) shiftScope(c *gin.Context, requestedUserID *int) ownership.Scope {
	return ownership.Resolve(
		middleware.GetUserID(c),
		ownership.CanAccessAll(middleware.GetPermissions(c), permissions.ShiftReview),
		requestedUserID,
	)
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(permissions.Code) gin.HandlerFunc) {
	r.POST("/shifts/open", auth, perm(permissions.ShiftCreate), h.OpenShift)
	r.POST("/shifts/:id/close", auth, perm(permissions.ShiftCreate), h.CloseShift)
	r.POST("/shifts/:id/review", auth, perm(permissions.ShiftReview), h.ReviewShift)
	r.POST("/shifts/:id/audit", auth, perm(permissions.ShiftAudit), h.AuditShift)
	r.POST("/shifts/:id/cash-movements", auth, perm(permissions.ShiftCashMovement), h.CreateCashMovement)
	r.GET("/shifts/active", auth, h.GetActiveShift)
	r.GET("/shifts", auth, perm(permissions.ShiftView), h.ListShifts)
	r.GET("/shifts/export", auth, perm(permissions.ShiftView), h.ExportShifts)
	r.GET("/shifts/:id", auth, perm(permissions.ShiftView), h.GetShiftByID)
	r.GET("/shifts/:id/cash-movements", auth, perm(permissions.ShiftView), h.ListCashMovements)
	r.GET("/shifts/:id/report", auth, perm(permissions.ShiftView), h.GetShiftReport)
}

func (h *Handler) OpenShift(c *gin.Context) {
	var req struct {
		StoreID        *int `json:"store_id"`
		OpeningBalance int  `json:"opening_balance"`
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

	var shift *Shift
	var err error
	if h.auditSvc != nil {
		err = h.svc.InTx(c.Request.Context(), func(tx pgx.Tx) error {
			var e error
			shift, e = h.svc.OpenShiftTx(c.Request.Context(), tx, uid, req.StoreID, req.OpeningBalance)
			if e != nil {
				return e
			}
			return h.auditSvc.CreateAuditLogTx(c.Request.Context(), tx, &audit.Log{
				UserID:      middleware.UserIDFromContext(c.Request.Context()),
				Username:    middleware.UsernameFromContext(c.Request.Context()),
				Role:        middleware.RoleFromContext(c.Request.Context()),
				Action:      "shift_opened",
				EntityType:  "shift",
				EntityID:    &shift.ID,
				NewValues:   shared.ToJSONMap(map[string]interface{}{"opening_balance": req.OpeningBalance, "store_id": req.StoreID}),
				IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
				UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
				Description: "Opened shift",
				StoreID:     middleware.StoreIDFromContext(c.Request.Context()),
			})
		})
	} else {
		shift, err = h.svc.OpenShift(c.Request.Context(), uid, req.StoreID, req.OpeningBalance)
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
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

	var shift *Shift
	if h.auditSvc != nil {
		err = h.svc.InTx(c.Request.Context(), func(tx pgx.Tx) error {
			var e error
			shift, e = h.svc.CloseShiftTx(c.Request.Context(), tx, shiftID, uid, req.ClosingBalance, req.Notes)
			if e != nil {
				return e
			}
			return h.auditSvc.CreateAuditLogTx(c.Request.Context(), tx, &audit.Log{
				UserID:      middleware.UserIDFromContext(c.Request.Context()),
				Username:    middleware.UsernameFromContext(c.Request.Context()),
				Role:        middleware.RoleFromContext(c.Request.Context()),
				Action:      "shift_closed",
				EntityType:  "shift",
				EntityID:    &shift.ID,
				NewValues:   shared.ToJSONMap(map[string]interface{}{"closing_balance": req.ClosingBalance, "notes": req.Notes}),
				IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
				UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
				Description: "Closed shift",
				StoreID:     middleware.StoreIDFromContext(c.Request.Context()),
			})
		})
	} else {
		shift, err = h.svc.CloseShift(c.Request.Context(), shiftID, uid, req.ClosingBalance, req.Notes)
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
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
	discFilter := c.Query("discrepancy")

	limit, offset := shared.ParsePaginationParams(limitStr, offsetStr)

	var userID *int
	if uidStr := c.Query("user_id"); uidStr != "" {
		if uid, err := strconv.Atoi(uidStr); err == nil {
			userID = &uid
		}
	}

	var needsReview *bool
	if nr := c.Query("needs_review"); nr != "" {
		val := nr == "true"
		needsReview = &val
	}

	shifts, total, err := h.svc.ListShifts(c.Request.Context(), h.shiftScope(c, userID), status, needsReview, discFilter, limit, offset, sortBy, sortDir)
	if err != nil {
		shared.InternalError(c, err)
		return
	}

	shared.JSONPaginated(c, shifts, total, limit, offset)
}

func (h *Handler) ExportShifts(c *gin.Context) {
	format := c.Query("format")
	status := c.Query("status")
	discFilter := c.Query("discrepancy")

	var userID *int
	if uidStr := c.Query("user_id"); uidStr != "" {
		if uid, err := strconv.Atoi(uidStr); err == nil {
			userID = &uid
		}
	}

	var needsReview *bool
	if nr := c.Query("needs_review"); nr != "" {
		val := nr == "true"
		needsReview = &val
	}

	scope := h.shiftScope(c, userID)
	shifts, err := h.svc.ExportShifts(c.Request.Context(), scope, status, needsReview, discFilter)
	if err != nil {
		shared.InternalError(c, err)
		return
	}

	if shifts == nil {
		shifts = []Shift{}
	}

	rowCount := len(shifts)

	cfg := config.Load()
	now := time.Now().In(cfg.Timezone)
	filename := "shifts-" + now.Format("2006-01-02")

	switch format {
	case "xlsx":
		wb := excelize.NewFile()
		sheet := "Shifts"
		_ = wb.SetSheetName("Sheet1", sheet)

		headers := []string{"Cashier", "Store", "Status", "Opening Balance", "Cash Sales", "Non-Cash Sales", "Total Sales", "Transactions", "Discrepancy", "Needs Review", "Opened At", "Closed At"}
		for i, hdr := range headers {
			col, _ := excelize.ColumnNumberToName(i + 1)
			_ = wb.SetCellValue(sheet, col+"1", hdr)
		}
		headerStyle, _ := wb.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
			Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"7C3AED"}},
		})
		_ = wb.SetCellStyle(sheet, "A1", fmt.Sprintf("%s1", mustColumnName(len(headers))), headerStyle)

		for i, s := range shifts {
			r := i + 2
			_ = wb.SetCellValue(sheet, fmt.Sprintf("A%d", r), s.Username)
			storeName := ""
			if s.StoreName != "" {
				storeName = s.StoreName
			}
			_ = wb.SetCellValue(sheet, fmt.Sprintf("B%d", r), storeName)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("C%d", r), s.Status)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("D%d", r), s.OpeningBalance)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("E%d", r), s.CashSales)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("F%d", r), s.NonCashSales)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("G%d", r), s.TotalSales)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("H%d", r), s.TransactionCount)
			discVal := 0
			if s.Discrepancy != nil {
				discVal = *s.Discrepancy
			}
			_ = wb.SetCellValue(sheet, fmt.Sprintf("I%d", r), discVal)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("J%d", r), map[bool]string{true: "Yes", false: "No"}[s.NeedsReview])
			_ = wb.SetCellValue(sheet, fmt.Sprintf("K%d", r), s.OpenedAt)
			closed := ""
			if s.ClosedAt != "" {
				closed = s.ClosedAt
			}
			_ = wb.SetCellValue(sheet, fmt.Sprintf("L%d", r), closed)
		}

		colWidths := []float64{20, 15, 10, 15, 15, 15, 15, 12, 15, 12, 22, 22}
		for i, w := range colWidths {
			col, _ := excelize.ColumnNumberToName(i + 1)
			_ = wb.SetColWidth(sheet, col, col, w)
		}

		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.xlsx"`, filename))

		if err := wb.Write(c.Writer); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write xlsx"})
			return
		}

	default:
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, filename))
		_, _ = c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

		writer := csv.NewWriter(c.Writer)
		_ = shared.WriteCSVRow(writer, []string{"Cashier", "Store", "Status", "Opening Balance", "Cash Sales", "Non-Cash Sales", "Total Sales", "Transactions", "Discrepancy", "Needs Review", "Opened At", "Closed At"})
		for _, s := range shifts {
			storeName := ""
			if s.StoreName != "" {
				storeName = s.StoreName
			}
			discVal := "0"
			if s.Discrepancy != nil {
				discVal = strconv.Itoa(*s.Discrepancy)
			}
			closed := ""
			if s.ClosedAt != "" {
				closed = s.ClosedAt
			}
			_ = shared.WriteCSVRow(writer, []string{
				s.Username,
				storeName,
				s.Status,
				strconv.Itoa(s.OpeningBalance),
				strconv.Itoa(s.CashSales),
				strconv.Itoa(s.NonCashSales),
				strconv.Itoa(s.TotalSales),
				strconv.Itoa(s.TransactionCount),
				discVal,
				map[bool]string{true: "Yes", false: "No"}[s.NeedsReview],
				s.OpenedAt,
				closed,
			})
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to flush csv"})
			return
		}
	}

	if h.auditSvc != nil {
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.Log{
			UserID:      middleware.UserIDFromContext(c.Request.Context()),
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "export",
			EntityType:  "shift",
			NewValues:   shared.ToJSONMap(map[string]interface{}{"format": format, "status": status, "needs_review": needsReview, "discrepancy": discFilter, "row_count": rowCount}),
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Exported %d shifts as %s", rowCount, format),
			StoreID:     middleware.StoreIDFromContext(c.Request.Context()),
		})
	}
}

func mustColumnName(n int) string {
	name, err := excelize.ColumnNumberToName(n)
	if err != nil {
		return "Z"
	}
	return name
}

func (h *Handler) GetShiftByID(c *gin.Context) {
	shiftID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid shift id"})
		return
	}

	shift, err := h.svc.GetShiftByID(c.Request.Context(), h.shiftScope(c, nil), shiftID)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.auditSvc != nil {
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.Log{
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
			StoreID:     middleware.StoreIDFromContext(c.Request.Context()),
		})
	}

	shared.JSONSuccess(c, shift)
}

func (h *Handler) AuditShift(c *gin.Context) {
	shiftID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid shift id"})
		return
	}

	var req struct {
		ActualBalance int `json:"actual_balance"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	shift, cashSales, err := h.svc.AuditShift(c.Request.Context(), shiftID)
	if err != nil {
		shared.InternalError(c, err)
		return
	}

	expected := shift.OpeningBalance + cashSales
	offBy := req.ActualBalance - expected

	if h.auditSvc != nil {
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.Log{
			UserID:      middleware.UserIDFromContext(c.Request.Context()),
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "audit",
			EntityType:  "shift",
			EntityID:    &shift.ID,
			NewValues:   shared.ToJSONMap(map[string]interface{}{"actual_balance": req.ActualBalance, "expected": expected, "off_by": offBy}),
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: "Audited shift cash balance",
			StoreID:     middleware.StoreIDFromContext(c.Request.Context()),
		})
	}

	absOffBy := offBy
	if absOffBy < 0 {
		absOffBy = -absOffBy
	}
	threshold := h.svc.GetDiscrepancyThreshold(c.Request.Context())
	flaggedForReview := absOffBy > threshold
	if flaggedForReview {
		_ = h.svc.FlagForReview(c.Request.Context(), shiftID)
	}

	shared.JSONSuccess(c, gin.H{
		"shift":             shift,
		"expected_cash":     expected,
		"actual_balance":    req.ActualBalance,
		"off_by":            offBy,
		"flagged_for_review": flaggedForReview,
	})
}

func (h *Handler) CreateCashMovement(c *gin.Context) {
	shiftID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid shift id"})
		return
	}

	userID := middleware.GetUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		Type        string  `json:"type" binding:"required"`
		Amount      int     `json:"amount" binding:"required"`
		Description *string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	mov, err := h.svc.CreateCashMovement(c.Request.Context(), shiftID, userID, req.Type, req.Amount, req.Description)
	if err != nil {
		if errors.Is(err, ErrShiftClosed) || errors.Is(err, ErrInvalidMovementType) || errors.Is(err, ErrNotShiftOwner) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		shared.InternalError(c, err)
		return
	}

	if h.auditSvc != nil {
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.Log{
			UserID:      middleware.UserIDFromContext(c.Request.Context()),
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "create",
			EntityType:  "cash_movement",
			EntityID:    &mov.ID,
			NewValues:   shared.ToJSONMap(map[string]interface{}{"shift_id": shiftID, "type": req.Type, "amount": req.Amount}),
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Recorded %s of %d", req.Type, req.Amount),
			StoreID:     middleware.StoreIDFromContext(c.Request.Context()),
		})
	}

	shared.JSONSuccess(c, mov)
}

func (h *Handler) ListCashMovements(c *gin.Context) {
	shiftID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid shift id"})
		return
	}

	movements, err := h.svc.ListCashMovements(c.Request.Context(), shiftID)
	if err != nil {
		shared.InternalError(c, err)
		return
	}

	shared.JSONSuccess(c, movements)
}

func (h *Handler) GetShiftReport(c *gin.Context) {
	shiftID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid shift id"})
		return
	}

	report, err := h.svc.GetShiftReportData(c.Request.Context(), shiftID)
	if err != nil {
		shared.InternalError(c, err)
		return
	}

	shared.JSONSuccess(c, report)
}
